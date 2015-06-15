// Package edit implements a thread-safe text-editing buffer.
package edit

import (
	"container/list"
	"crypto/md5"
	"strings"
)

func getElem(l *list.List, n int) *list.Element {
	var elem *list.Element
	if n > l.Len()/2 {
		elem = l.Back()
		for i := l.Len(); i > n; i-- {
			elem = elem.Prev()
		}
	} else {
		elem = l.Front()
		for n = n; n > 1; n-- {
			elem = elem.Next()
		}
	}
	return elem
}

type fragList struct {
	*list.List
	cont bool
}

type lineInfo struct {
	text []rune
	disp *list.Element
}

type bufferOp struct {
	insert     bool // if not insert, then delete
	start, end Index
	text       []rune
}

type separator struct{} // for use in undo and redo stacks

// Buffer is a thread-safe text-editing buffer.
type Buffer struct {
	lines      *list.List // list of lineInfos
	dLines     *list.List // display lines; list of fragLists
	unlock     chan int   // used as mutex
	strings    []string   // for misc. use *only* when locked
	checksum   [md5.Size]byte
	syntax     syntax
	cols, rows int // display size
	tabWidth   int
	scroll     int
	marks      map[int]Index
	undo, redo *list.List // undo and redo stacks
}

// NewBuffer initializes and returns a new empty Buffer.
func NewBuffer() *Buffer {
	b := Buffer{
		lines:    list.New(),
		dLines:   list.New(),
		unlock:   make(chan int, 1),
		strings:  make([]string, 0),
		checksum: md5.Sum([]byte{}),
		syntax:   []Rule{},
		cols:     80,
		rows:     25,
		tabWidth: 8,
		scroll:   0,
		marks:    make(map[int]Index),
		undo:     list.New(),
		redo:     list.New(),
	}
	dLine := fragList{list.New(), false}
	dLine.PushBack(Fragment{})
	b.lines.PushBack(lineInfo{[]rune(""), b.dLines.PushBack(dLine)})
	b.unlock <- 1
	return &b
}

func (b *Buffer) clip(index Index) Index {
	if index.Line < 1 {
		index.Line = 1
	} else if index.Line > b.lines.Len() {
		index.Line = b.lines.Len()
	}
	if index.Char < 0 {
		index.Char = 0
	} else {
		lineLen := len(getElem(b.lines, index.Line).Value.(lineInfo).text)
		if index.Char > lineLen {
			index.Char = lineLen
		}
	}
	return index
}

func (b *Buffer) redisplay(begin, end int) {
	beginElem, endElem := getElem(b.lines, begin), getElem(b.lines, end)
	for elem := beginElem; elem != nil; elem = elem.Next() {
		// Remove existing display lines, but keep first as an anchor
		first := elem.Value.(lineInfo).disp
		disp := first.Next()
		for disp != nil && disp.Value.(fragList).cont {
			next := disp.Next()
			b.dLines.Remove(disp)
			disp = next
		}
		// Insert new display lines
		prev := first
		dLine, col := fragList{list.New(), false}, 0
		fragments := b.syntax.split(string(expand(elem.Value.(lineInfo).text,
			b.tabWidth)))
		for frag := range fragments {
			text := []rune(frag.Text)
			for {
				if len(text)+col <= b.cols {
					dLine.PushBack(Fragment{string(text), frag.Tag})
					col += len(text)
					text = text[:0]
				} else {
					if col < b.cols {
						dLine.PushBack(Fragment{string(text[:b.cols-col]),
							frag.Tag})
					}
					prev = b.dLines.InsertAfter(dLine, prev)
					dLine = fragList{list.New(), true}
					text = text[b.cols-col:]
					col = 0
				}
				if len(text) == 0 {
					break
				}
			}
		}
		b.dLines.InsertAfter(dLine, prev)
		elem.Value = lineInfo{elem.Value.(lineInfo).text, first.Next()}
		// Remove leftover display line
		b.dLines.Remove(first)
		// Break if end line has been processed
		if elem == endElem {
			break
		}
	}
}

// CoordsFromIndex returns the display coordinates of index. Coordinates may be
// out of bounds of the buffer's current display.
func (b *Buffer) CoordsFromIndex(index Index) (col, row int) {
	<-b.unlock
	row -= b.scroll
	index = b.clip(index)
	line := getElem(b.lines, index.Line).Value.(lineInfo)
	for e := b.dLines.Front(); e != line.disp; e = e.Next() {
		row++
	}
	col = columns(line.text[:index.Char], b.tabWidth)
	row += col / b.cols
	col %= b.cols
	b.unlock <- 1
	return
}

// delete_ performs a deletion without modifying the undo stack.
func (b *Buffer) delete(begin, end Index) {
	// perform deletion
	elem := getElem(b.lines, begin.Line)
	if n := end.Line - begin.Line; n == 0 {
		text := elem.Value.(lineInfo).text
		elem.Value = lineInfo{append(text[:begin.Char], text[end.Char:]...),
			elem.Value.(lineInfo).disp}
	} else {
		firstLine := elem.Value.(lineInfo).text
		for i := 0; i < n; i++ {
			elem = elem.Next()
			e := b.lines.Remove(elem.Prev()).(lineInfo).disp
			for {
				next := e.Next()
				b.dLines.Remove(e)
				e = next
				if e == nil || !e.Value.(fragList).cont {
					break
				}
			}
		}
		elem.Value = lineInfo{append(firstLine[:begin.Char],
			elem.Value.(lineInfo).text[end.Char:]...),
			elem.Value.(lineInfo).disp}
	}
	b.redisplay(begin.Line, begin.Line)

	// update marks
	for k, v := range b.marks {
		if v.Line > begin.Line ||
			(v.Line == begin.Line && v.Char >= begin.Char) {
			if v.Line <= end.Line {
				if v.Line < end.Line || v.Char <= end.Char {
					v = begin
				} else {
					v.Line = begin.Line
					v.Char += begin.Char - end.Char
				}
			} else {
				v.Line -= end.Line - begin.Line
			}
		}
		b.marks[k] = v
	}
}

// Delete removes the text in the buffer between begin and end.
func (b *Buffer) Delete(begin, end Index) {
	<-b.unlock
	if end.Less(begin) || end == begin {
		b.unlock <- 1
		return
	}
	begin, end = b.clip(begin), b.clip(end)
	runes := []rune(b.get(begin, end))

	// insert undo operation (merge with previous deletion if possible)
	merged := false
	if b.undo.Len() != 0 {
		if op, ok := b.undo.Back().Value.(bufferOp); ok && !op.insert {
			if op.start == end {
				b.undo.Remove(b.undo.Back())
				op.start = begin
				op.text = append(runes, op.text...)
				b.undo.PushBack(op)
				merged = true
			} else if op.end == begin {
				b.undo.Remove(b.undo.Back())
				op.end = end
				op.text = append(op.text, runes...)
				b.undo.PushBack(op)
				merged = true
			}
		}
	}
	if !merged {
		b.undo.PushBack(bufferOp{false, begin, end, runes})
	}
	b.redo.Init()

	b.delete(begin, end)
	b.unlock <- 1
}

// DisplayLines returns a slice of Lists of Fragments, one list for each line
// on the buffer's current display.
func (b *Buffer) DisplayLines() []*list.List {
	<-b.unlock
	lines := make([]*list.List, b.rows)
	dLine := getElem(b.dLines, b.scroll+1)
	for i := range lines {
		fragments := list.New()
		if l := dLine; l != nil {
			for e := l.Value.(fragList).Front(); e != nil; e = e.Next() {
				fragments.PushBack(e.Value.(Fragment))
			}
			dLine = dLine.Next()
		}
		lines[i] = fragments
	}
	b.unlock <- 1
	return lines
}

func (b *Buffer) end() Index {
	index := Index{1, 0}
	if b.lines.Len() > 0 {
		index = Index{b.lines.Len(), len(b.lines.Back().Value.(lineInfo).text)}
	}
	return index
}

// End returns an Index after the last character in the Buffer.
func (b *Buffer) End() Index {
	<-b.unlock
	index := b.end()
	b.unlock <- 1
	return index
}

func (b *Buffer) get(begin, end Index) string {
	if end.Less(begin) || end == begin {
		return ""
	}
	begin, end = b.clip(begin), b.clip(end)
	n := 1 + end.Line - begin.Line
	if len(b.strings) < n {
		b.strings = make([]string, n*2)
	}
	lines := b.strings[:n]
	elem := getElem(b.lines, begin.Line)
	for i := 0; i < n; i++ {
		lines[i] = string(elem.Value.(lineInfo).text)
		elem = elem.Next()
	}
	if n > 1 {
		lines[0] = string([]rune(lines[0])[begin.Char:])
		lines[n-1] = string([]rune(lines[n-1])[:end.Char])
	} else {
		lines[0] = string([]rune(lines[0])[begin.Char:end.Char])
	}
	return strings.Join(lines, "\n")
}

// Get returns the text in the Buffer between begin and end.
func (b *Buffer) Get(begin, end Index) string {
	<-b.unlock
	s := b.get(begin, end)
	b.unlock <- 1
	return s
}

// IndexFromCoords returns the closest index to the given display coordinates.
func (b *Buffer) IndexFromCoords(col, row int) Index {
	<-b.unlock

	// clip values
	if col < 0 {
		col = 0
	}
	row += b.scroll + 1
	if row > b.dLines.Len() {
		row = b.dLines.Len()
	} else if row < 1 {
		row = 1
	}

	// get line
	disp := getElem(b.dLines, row)
	for disp.Value.(fragList).cont {
		disp = disp.Prev()
		col += b.cols
	}
	index := Index{1, 0}
	var e *list.Element
	for e = b.lines.Front(); e.Value.(lineInfo).disp != disp; e = e.Next() {
		index.Line++
	}

	// get char
	c := 0
	for _, ch := range e.Value.(lineInfo).text {
		if ch == '\t' {
			c += b.tabWidth - c%b.tabWidth
		} else {
			c++
		}
		if c > col {
			break
		}
		index.Char++
	}

	b.unlock <- 1
	return index
}

// IndexFromMark returns the current index of the mark with ID id, or a
// zero-value index if no mark with ID id is set.
func (b *Buffer) IndexFromMark(id int) Index {
	<-b.unlock
	index := b.marks[id]
	b.unlock <- 1
	return index
}

// insert performs and inseration without undo stack modification.
func (b *Buffer) insert(index Index, text string) {
	elem := getElem(b.lines, index.Line)
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		li := elem.Value.(lineInfo)
		elem.Value = lineInfo{[]rune(string(li.text[:index.Char]) + lines[0] +
			string(li.text[index.Char:])), li.disp}
	} else {
		firstText := elem.Value.(lineInfo).text
		disp := elem.Value.(lineInfo).disp
		for disp.Next() != nil && disp.Next().Value.(fragList).cont {
			disp = disp.Next()
		}
		for i, line := range lines {
			if i == 0 {
				li := elem.Value.(lineInfo)
				elem.Value = lineInfo{append(firstText[:index.Char],
					[]rune(line)...), li.disp}
			} else if i == len(lines)-1 {
				disp = b.dLines.InsertAfter(fragList{list.New(), false}, disp)
				elem = b.lines.InsertAfter(lineInfo{append([]rune(line),
					firstText[index.Char:]...), disp}, elem)
			} else {
				disp = b.dLines.InsertAfter(fragList{list.New(), false}, disp)
				elem = b.lines.InsertAfter(lineInfo{[]rune(line), disp}, elem)
			}
		}
	}
	b.redisplay(index.Line, index.Line+len(lines)-1)

	// update marks
	for k, v := range b.marks {
		if v.Line == index.Line && v.Char >= index.Char {
			if len(lines) == 1 {
				v.Char += len(lines[0])
			} else {
				v.Char = len(lines[len(lines)-1])
			}
			v.Line += len(lines) - 1
		} else if v.Line > index.Line {
			v.Line += len(lines) - 1
		}
		b.marks[k] = v
	}
}

// Insert inserts text into the buffer at index.
func (b *Buffer) Insert(index Index, text string) {
	<-b.unlock
	index = b.clip(index)
	b.insert(index, text)
	runes := []rune(text)

	// insert undo operation (merge with previous insertion if possible)
	merged := false
	if b.undo.Len() != 0 {
		if op, ok := b.undo.Back().Value.(bufferOp); ok && op.insert {
			if op.start == index {
				b.undo.Remove(b.undo.Back())
				op.end = b.shiftIndex(op.end, len(runes))
				op.text = append(runes, op.text...)
				b.undo.PushBack(op)
				merged = true
			} else if op.end == index {
				b.undo.Remove(b.undo.Back())
				op.end = b.shiftIndex(op.end, len(runes))
				op.text = append(op.text, runes...)
				b.undo.PushBack(op)
				merged = true
			}
		}
	}
	if !merged {
		b.undo.PushBack(bufferOp{true, index, b.shiftIndex(index, len(runes)),
			runes})
	}
	b.redo.Init()

	b.unlock <- 1
}

// Mark sets a mark with ID id at index. The mark's position is automatically
// updated when the buffer contents are modified. If a mark with ID id already
// exists, its position is updated. Multiple IDs can be specified to set
// multiple marks at the same time.
func (b *Buffer) Mark(index Index, id ...int) {
	<-b.unlock
	for _, id := range id {
		b.marks[id] = b.clip(index)
	}
	b.unlock <- 1
}

// Modified returns true if and only if the buffer's contents differ from the
// contents at the last time ResetModified was called. This operation is
// expsensive, since it must hash the entire buffer contents.
func (b *Buffer) Modified() bool {
	checksum := md5.Sum([]byte(b.Get(Index{1, 0}, b.End())))
	<-b.unlock
	val := checksum != b.checksum
	b.unlock <- 1
	return val
}

// Redo redoes the last undone sequence of insertions and deletions and returns
// true, or returns false if the redo stack is empty. The given marks are
// positioned at the index of the redone operation.
func (b *Buffer) Redo(mark ...int) bool {
	<-b.unlock
	if b.redo.Len() > 0 {
		opRedone := false
		loop := true
		for loop && b.redo.Len() > 0 {
			switch v := b.redo.Remove(b.redo.Back()); v := v.(type) {
			case bufferOp:
				if v.insert {
					b.insert(v.start, string(v.text))
					for _, id := range mark {
						b.marks[id] = v.end
					}
				} else {
					b.delete(v.start, v.end)
					for _, id := range mark {
						b.marks[id] = v.start
					}
				}
				opRedone = true
				b.undo.PushBack(v)
			case separator:
				if opRedone {
					loop = false
				}
				b.undo.PushBack(v)
			}
		}
		b.unlock <- 1
		return opRedone
	}
	b.unlock <- 1
	return false
}

// ResetModified sets the comparison point for future calls to Modified to the
// current contents of the buffer. This operation is expensive, since it must
// hash the entire buffer contents.
func (b *Buffer) ResetModified() {
	<-b.unlock
	b.checksum = md5.Sum([]byte(b.get(Index{1, 0}, b.end())))
	b.unlock <- 1
}

// ResetUndo clears the buffer's undo and redo stacks.
func (b *Buffer) ResetUndo() {
	<-b.unlock
	b.undo.Init()
	b.redo.Init()
	b.unlock <- 1
}

// Scroll scrolls the buffer's display down by delta lines.
func (b *Buffer) Scroll(delta int) {
	<-b.unlock
	b.scroll += delta
	if b.scroll < 0 || b.dLines.Len() < b.rows {
		b.scroll = 0
	} else if b.scroll+b.rows > b.dLines.Len() {
		b.scroll = b.dLines.Len() - b.rows
	}
	b.unlock <- 1
}

// ScrollFraction returns a number in the range [0, 1] describing the vertical
// scroll fraction of the buffer display. If the entire content is visible, -1
// is returned instead.
func (b *Buffer) ScrollFraction() float64 {
	<-b.unlock
	f := -1.0
	if b.rows < b.dLines.Len() {
		f = float64(b.scroll) / float64(b.dLines.Len()-b.rows)
	}
	b.unlock <- 1
	return f
}

// Separate inserts a separator onto the undo stack in order to delimit
// sequences of insertions and deletions.
func (b *Buffer) Separate() {
	<-b.unlock
	if b.undo.Len() != 0 {
		if _, ok := b.undo.Back().Value.(bufferOp); ok {
			b.undo.PushBack(separator{})
		}
	}
	b.unlock <- 1
}

// SetSize sets the display size of the buffer.
func (b *Buffer) SetSize(cols, rows int) {
	<-b.unlock
	b.cols, b.rows = cols, rows
	if b.cols < 1 {
		b.cols = 1
	}
	if b.rows < 0 {
		b.rows = 0
	}
	// TODO: Make sure scroll isn't out of bounds now.
	// TODO:
	// This also needs to recompute display lines, but it doesn't need to
	// re-split them into fragments, so it might be faster to use a different
	// algorithm than the one used for SetSyntax.
	b.redisplay(1, b.lines.Len())
	b.unlock <- 1
}

// SetSyntax sets the syntax highlighting rules for the buffer to rules.
func (b *Buffer) SetSyntax(rules []Rule) {
	<-b.unlock
	// Copy rules to negate risk of concurrent modification
	if len(b.syntax) < len(rules) {
		b.syntax = make([]Rule, len(rules))
	}
	for i, rule := range rules {
		b.syntax[i] = rule
	}
	b.syntax = b.syntax[:len(rules)]
	b.redisplay(1, b.lines.Len())
	b.unlock <- 1
}

// SetTabWidth sets the tab width of the buffer to cols.
func (b *Buffer) SetTabWidth(cols int) {
	<-b.unlock
	b.tabWidth = cols
	// TODO: Same as SetSize.
	b.redisplay(1, b.lines.Len())
	b.unlock <- 1
}

// shiftIndex shitfs and index without locking the buffer.
func (b *Buffer) shiftIndex(index Index, chars int) Index {
	index = b.clip(index)
	elem := getElem(b.lines, index.Line)
	for chars < 0 {
		if index.Char == 0 {
			if index.Line == 1 {
				chars = 0
			} else {
				index.Line--
				elem = elem.Prev()
				index.Char = len(elem.Value.(lineInfo).text)
				chars++
			}
		} else if -chars > index.Char {
			chars += index.Char
			index.Char = 0
		} else {
			index.Char += chars
			chars = 0
		}
	}
	for chars > 0 {
		lineLen := len(elem.Value.(lineInfo).text)
		if index.Char == lineLen {
			if index.Line == b.lines.Len() {
				chars = 0
			} else {
				index.Line++
				index.Char = 0
				elem = elem.Next()
				chars--
			}
		} else if chars > lineLen-index.Char {
			chars -= lineLen - index.Char
			index.Char = lineLen
		} else {
			index.Char += chars
			chars = 0
		}
	}
	return index
}

// ShiftIndex returns index shifted right by chars. If chars is negative, index
// is shifted left.
func (b *Buffer) ShiftIndex(index Index, chars int) Index {
	<-b.unlock
	index = b.shiftIndex(index, chars)
	b.unlock <- 1
	return index
}

// Undo undoes the last sequence of insertions and deletions and returns true,
// or returns false if the redo stack is empty. The given marks are positioned
// at the index of the undone operation.
func (b *Buffer) Undo(mark ...int) bool {
	<-b.unlock
	if b.undo.Len() > 0 {
		opUndone := false
		loop := true
		for loop && b.undo.Len() > 0 {
			switch v := b.undo.Remove(b.undo.Back()); v := v.(type) {
			case bufferOp:
				if v.insert {
					b.delete(v.start, v.end)
					for _, id := range mark {
						b.marks[id] = v.start
					}
				} else {
					b.insert(v.start, string(v.text))
					for _, id := range mark {
						b.marks[id] = v.end
					}
				}
				opUndone = true
				b.redo.PushBack(v)
			case separator:
				if opUndone {
					loop = false
				}
				b.redo.PushBack(v)
			}
		}
		b.unlock <- 1
		return opUndone
	}
	b.unlock <- 1
	return false
}

// Index denotes a position in a Buffer.
type Index struct {
	Line, Char int
}

// Less returns true if the index is less than other.
func (i *Index) Less(other Index) bool {
	return i.Line < other.Line || (i.Line == other.Line && i.Char < other.Char)
}
