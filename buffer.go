// Package edit implements a thread-safe text-editing buffer.
package edit

import (
	"container/list"
	"strings"
)

// Buffer is a thread-safe text-editing buffer.
type Buffer struct {
	lines  *list.List
	unlock chan int
}

// NewBuffer initializes and returns a new empty Buffer.
func NewBuffer() *Buffer {
	b := Buffer{
		lines:  list.New(),
		unlock: make(chan int, 1),
	}
	b.lines.PushBack("")
	b.unlock <- 1
	return &b
}

func (b *Buffer) getLine(n int) *list.Element {
	elem := b.lines.Front()
	for n = n; n > 1; n-- {
		elem = elem.Next()
	}
	return elem
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
		lineLen := len(b.getLine(index.Line).Value.(string))
		if index.Char > lineLen {
			index.Char = lineLen
		}
	}
	return index
}

// End returns an Index after the last character in the Buffer.
func (b *Buffer) End() Index {
	<-b.unlock
	index := Index{1, 0}
	if b.lines.Len() > 0 {
		index = Index{b.lines.Len(), len(b.lines.Back().Value.(string))}
	}
	b.unlock <- 1
	return index
}

// Get returns the text in the Buffer between begin and end.
func (b *Buffer) Get(begin, end Index) string {
	<-b.unlock
	if end.Less(begin) || end == begin {
		b.unlock <- 1
		return ""
	}
	begin, end = b.clip(begin), b.clip(end)
	n := 1 + end.Line - begin.Line
	lines := make([]string, n)
	elem := b.getLine(begin.Line)
	for i := 0; i < n; i++ {
		lines[i] = elem.Value.(string)
		elem = elem.Next()
	}
	b.unlock <- 1
	if n > 1 {
		lines[0] = lines[0][begin.Char:]
		lines[n-1] = lines[n-1][:end.Char]
	} else if n == 1 {
		lines[0] = lines[0][begin.Char:end.Char]
	}
	return strings.Join(lines, "\n")
}

// Insert inserts text into the buffer at index.
func (b *Buffer) Insert(index Index, text string) {
	<-b.unlock
	index = b.clip(index)
	elem := b.getLine(index.Line)
	first := elem
	for _, line := range strings.Split(text, "\n") {
		elem = b.lines.InsertAfter(line, elem)
	}
	elem.Value = elem.Value.(string) + first.Value.(string)[index.Char:]
	first.Value = first.Value.(string)[:index.Char] +
		b.lines.Remove(first.Next()).(string)
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
