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
	text string
	disp *list.Element
}

// Buffer is a thread-safe text-editing buffer.
type Buffer struct {
	lines      *list.List // List of lineInfos
	dLines     *list.List // Display lines; list of fragLists
	unlock     chan int   // Used as mutex
	strings    []string   // For misc. use *only* when locked
	checksum   [md5.Size]byte
	syntax     syntax
	cols, rows int // Display size
	tabWidth   int
	scroll     int
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
	}
	dLine := fragList{list.New(), false}
	dLine.PushBack(Fragment{})
	b.lines.PushBack(lineInfo{"", b.dLines.PushBack(dLine)})
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
		for frag := range b.syntax.split(elem.Value.(lineInfo).text) {
			text := expand(frag.Text, b.tabWidth)
			for {
				if len(text)+col <= b.cols {
					dLine.PushBack(Fragment{text, frag.Tag})
					col += len(text)
					text = ""
				} else {
					if col < b.cols {
						dLine.PushBack(Fragment{text[:b.cols-col], frag.Tag})
					}
					prev = b.dLines.InsertAfter(dLine, prev)
					dLine = fragList{list.New(), true}
					text = text[b.cols-col:]
					col = 0
				}
				if text == "" {
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

// Delete removes the text in the buffer between begin and end.
func (b *Buffer) Delete(begin, end Index) {
	<-b.unlock
	if end.Less(begin) || end == begin {
		b.unlock <- 1
		return
	}
	begin, end = b.clip(begin), b.clip(end)
	elem := getElem(b.lines, begin.Line)
	if n := end.Line - begin.Line; n == 0 {
		text := elem.Value.(lineInfo).text
		elem.Value = lineInfo{text[:begin.Char] + text[end.Char:],
			elem.Value.(lineInfo).disp}
	} else {
		firstLine := elem.Value.(lineInfo).text
		for i := 0; i < n; i++ {
			elem = elem.Next()
			e := b.lines.Remove(elem.Prev()).(lineInfo).disp
			for e != nil {
				e = e.Next()
				if e != nil && e.Value.(fragList).cont {
					b.dLines.Remove(e)
				} else {
					break
				}
			}
		}
		elem.Value = lineInfo{firstLine[:begin.Char] +
			elem.Value.(lineInfo).text[end.Char:], elem.Value.(lineInfo).disp}
	}
	b.redisplay(begin.Line, begin.Line)
	b.unlock <- 1
}

// DisplayLines returns a slice of Lists of Fragments, one list for each line
// on the buffer's current display.
func (b *Buffer) DisplayLines() []*list.List {
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
		lines[i] = elem.Value.(lineInfo).text
		elem = elem.Next()
	}
	if n > 1 {
		lines[0] = lines[0][begin.Char:]
		lines[n-1] = lines[n-1][:end.Char]
	} else {
		lines[0] = lines[0][begin.Char:end.Char]
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

// Insert inserts text into the buffer at index.
func (b *Buffer) Insert(index Index, text string) {
	<-b.unlock
	index = b.clip(index)
	elem := getElem(b.lines, index.Line)
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		li := elem.Value.(lineInfo)
		elem.Value = lineInfo{li.text[:index.Char] + lines[0] +
			li.text[index.Char:], li.disp}
	} else {
		firstText := elem.Value.(lineInfo).text
		disp := elem.Value.(lineInfo).disp
		for disp.Next() != nil && disp.Next().Value.(fragList).cont {
			disp = disp.Next()
		}
		for i, line := range lines {
			if i == 0 {
				li := elem.Value.(lineInfo)
				elem.Value = lineInfo{firstText[:index.Char] + line, li.disp}
			} else if i == len(lines)-1 {
				disp = b.dLines.InsertAfter(fragList{list.New(), false}, disp)
				elem = b.lines.InsertAfter(
					lineInfo{line + firstText[index.Char:], disp}, elem)
			} else {
				disp = b.dLines.InsertAfter(fragList{list.New(), false}, disp)
				elem = b.lines.InsertAfter(lineInfo{line, disp}, elem)
			}
		}
	}
	b.redisplay(index.Line, index.Line+len(lines)-1)
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

// ResetModified sets the comparison point for future calls to Modified to the
// current contents of the buffer. This operation is expensive, since it must
// hash the entire buffer contents.
func (b *Buffer) ResetModified() {
	<-b.unlock
	b.checksum = md5.Sum([]byte(b.get(Index{1, 0}, b.end())))
	b.unlock <- 1
}

// SetSize sets the display size of the buffer.
func (b *Buffer) SetSize(cols, rows int) {
	<-b.unlock
	b.cols, b.rows = cols, rows
	if b.cols < 1 {
		b.cols = 1
	}
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

// ShiftIndex returns index shifted right by chars. If chars is negative, index
// is shifted left.
func (b *Buffer) ShiftIndex(index Index, chars int) Index {
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

// Index denotes a position in a Buffer.
type Index struct {
	Line, Char int
}

// Less returns true if the index is less than other.
func (i *Index) Less(other Index) bool {
	return i.Line < other.Line || (i.Line == other.Line && i.Char < other.Char)
}
