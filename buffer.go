// Package edit implements a thread-safe text-editing buffer.
package edit

import (
	"strings"

	"github.com/petar/GoLLRB/llrb"
)

type lineItem struct {
	num int
	text string
}

func (l *lineItem) Less(than llrb.Item) bool {
	return l.num < than.(*lineItem).num
}

// Buffer is a thread-safe text-editing buffer.
type Buffer struct {
	lines *llrb.LLRB
	unlock chan int
}

// NewBuffer initializes and returns a new empty Buffer.
func NewBuffer() *Buffer {
	b := Buffer{
		lines: llrb.New(),
		unlock: make(chan int, 1),
	}
	b.unlock <- 1
	return &b
}

// Clip returns the closest valid Index to index.
func (b *Buffer) Clip(index Index) Index {
	<-b.unlock
	if index.Line > b.lines.Len() {
		index.Line = b.lines.Len()
	}
	if index.Line < 1 {
		index.Line = 1
	}
	if b.lines.Len() == 0 || index.Char < 0 {
		index.Char = 0
	} else {
		lineLen := len(b.lines.Get(&lineItem{index.Line, ""}).(*lineItem).text)
		if index.Char > lineLen {
			index.Char = lineLen
		}
	}
	b.unlock <- 1
	return index
}

// End returns an Index after the last character in the Buffer.
func (b *Buffer) End() Index {
	<-b.unlock
	index := Index{1, 0}
	if b.lines.Len() > 0 {
		index = Index{b.lines.Len(), len(b.lines.Max().(*lineItem).text)}
	}
	b.unlock <- 1
	return index
}

// Get returns the text in the Buffer between begin and end.
func (b *Buffer) Get(begin, end Index) string {
	if end.Less(begin) || end == begin {
		return ""
	}
	<-b.unlock
	beginItem, endItem := lineItem{begin.Line, ""}, lineItem{end.Line + 1, ""}
	lines := make([]string, 1+end.Line-begin.Line)
	i := 0
	b.lines.AscendRange(&beginItem, &endItem, func(item llrb.Item) bool {
		lines[i] = item.(*lineItem).text
		i++
		return true
	})
	if i > 1 {
		lines[0] = lines[0][begin.Char:]
		lines[i-1] = lines[i-1][:end.Char]
	} else if i == 1 {
		lines[0] = lines[0][begin.Char:end.Char]
	}
	b.unlock <- 1
	return strings.Join(lines[:i], "\n")
}

// Insert inserts text into the buffer at index.
func (b *Buffer) Insert(index Index, text string) {
	<-b.unlock
	lines := strings.Split(text, "\n")
	n := len(lines)
	firstLine := b.lines.Get(&lineItem{index.Line, ""})
	if n > 1 {
		pivot := &lineItem{index.Line+1, ""}
		b.lines.AscendGreaterOrEqual(pivot, func(i llrb.Item) bool {
			i.(*lineItem).num += n-1
			return true
		})
		for i, line := range lines {
			text := line
			if firstLine != nil {
				if i == 0 {
					text = firstLine.(*lineItem).text[:index.Char] + line
				} else if i == n - 1 {
					text = line + firstLine.(*lineItem).text[index.Char:]
				}
			}
			b.lines.ReplaceOrInsert(&lineItem{index.Line+i, text})
		}
	} else {
		text := lines[0]
		if firstLine != nil {
			text = firstLine.(*lineItem).text
			text = text[:index.Char] + lines[0] + text[index.Char:]
		}
		b.lines.ReplaceOrInsert(&lineItem{index.Line, text})
	}
	b.unlock <- 1
}

// Index denotes a position in a Buffer.
type Index struct {
	Line, Char int
}

// Less returns true if the index is less than other.
func (i *Index) Less(other Index) bool {
	return i.Line < other.Line || (i.Line == other.Line && i.Char < other.Char)
}
