package edit

import (
	"container/list"
	"math/rand"
	"strings"
	"testing"
)

func TestBuffer(t *testing.T) {
	// Init tests
	b := NewBuffer()
	b.SetSize(10, 25)
	want, got := "", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	if b.Modified() {
		t.Errorf("Modified returned true for new buffer")
	}

	// Insert tests
	b.Insert(Index{1, -1}, "world") // Index char too low
	want, got = "world", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	b.Insert(Index{2, 0}, "hello, ") // Index line too high
	want, got = "hello, world", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	b.Insert(Index{1, 20}, "!\nhello again!") // Index char too high
	want, got = "hello, world!\nhello again!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	b.Insert(Index{0, 0}, "\n\n") // Index line too low
	want, got = "\n\nhello, world!\nhello again!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}

	if !b.Modified() {
		t.Errorf("Modified returned false for modified buffer")
	}

	// Additional Get tests for getLine
	want, got = "\nhello, world!\nhello again!", b.Get(Index{2, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	want, got = "hello, world!\nhello again!", b.Get(Index{3, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}

	b.ResetModified()
	if b.Modified() {
		t.Errorf("Modified returned true directly after ResetModified")
	}

	// Delete tests
	b.Delete(Index{1, 0}, Index{1, 0}) // Delete nothing
	want, got = "\n\nhello, world!\nhello again!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	b.Delete(Index{2, 0}, Index{4, 6}) // Delete multiple lines
	want, got = "\nagain!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
	b.Delete(Index{2, 1}, Index{2, 4}) // Delete single line
	want, got = "\nan!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}

	if !b.Modified() {
		t.Errorf("Modified returned false for modified buffer")
	}
}

func TestBufferDisplay(t *testing.T) {
	b := NewBuffer()
	b.DisplayLines() // Shouldn't panic if there's no size or text
	iRule, _ := NewRule(`\b\w*i\w*\b`, "", 0)
	b.SetSyntax([]Rule{iRule})
	b.SetSize(8, 10)
	b.SetTabWidth(4)
	b.Insert(Index{1, 0},
		"package main\nfunc main() {\n\tprintln(\"hi\")\n}\n")
	dLines := make([]*list.List, 10)
	for i := range dLines {
		dLines[i] = list.New()
	}
	dLines[0].PushBack(Fragment{"package ", noneTag})
	dLines[1].PushBack(Fragment{"main", 0})
	dLines[2].PushBack(Fragment{"func ", noneTag})
	dLines[2].PushBack(Fragment{"mai", 0})
	dLines[3].PushBack(Fragment{"n", 0})
	dLines[3].PushBack(Fragment{"() {", noneTag})
	dLines[4].PushBack(Fragment{"    ", noneTag})
	dLines[4].PushBack(Fragment{"prin", 0})
	dLines[5].PushBack(Fragment{"tln", 0})
	dLines[5].PushBack(Fragment{`("`, noneTag})
	dLines[5].PushBack(Fragment{"hi", 0})
	dLines[5].PushBack(Fragment{`"`, noneTag})
	dLines[6].PushBack(Fragment{")", noneTag})
	dLines[7].PushBack(Fragment{"}", noneTag})
	for i, dLine := range b.DisplayLines() {
		e1, e2 := dLines[i].Front(), dLine.Front()
		for e1 != nil && e2 != nil {
			if e1.Value.(Fragment) != e2.Value.(Fragment) {
				t.Errorf("DisplayLines: got %#v, want %#v",
					e2.Value.(Fragment), e1.Value.(Fragment))
			}
			e1, e2 = e1.Next(), e2.Next()
		}
		if e1 != e2 { // Should both be nil
			t.Errorf("DisplayLines: got %#v, want %#v", e2, e1)
		}
	}
}

func randBuffer(numLines int) *Buffer {
	buf := NewBuffer()
	lines := make([]string, numLines)
	line := make([]byte, benchMaxLine)
	for i := 0; i < numLines; i++ {
		lineLen := rand.Int() % benchMaxLine
		for j := 0; j < lineLen; j++ {
			line[j] = byte(0x20 + rand.Int()%0x60)
		}
		lines[i] = string(line[:lineLen])
	}
	buf.Insert(buf.End(), strings.Join(lines, "\n"))
	return buf
}

func randIndexes(b *Buffer, n, maxLines int) []Index {
	indexes := make([]Index, n*2)
	for i := 0; i < n*2; i += 2 {
		begin := b.clip(Index{1 + rand.Int()%b.lines.Len(),
			rand.Int() % benchMaxLine})
		end := b.clip(Index{1 + rand.Int()%b.lines.Len(),
			rand.Int() % benchMaxLine})
		if end.Less(begin) {
			begin, end = end, begin
		}
		if end.Line-begin.Line <= maxLines {
			indexes[i], indexes[i+1] = begin, end
		} else {
			i -= 2
		}
	}
	return indexes
}

const (
	benchBufLines = 2000 // Lines in a benchmarking buffer
	benchOpLines  = 25   // Maximum lines in a benchmarking operation
	benchMaxLine  = 80   // Maximum characters in a benchmarking line
)

// Current benchmark: 70000 ns/op (was 16000 before redisplay)
func BenchmarkBufferDelete(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N, benchOpLines)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		text := buf.Get(indexes[i*2], indexes[i*2+1])
		b.StartTimer()
		buf.Delete(indexes[i*2], indexes[i*2+1])
		b.StopTimer()
		buf.Insert(indexes[i*2], text)
		b.StartTimer()
	}
}

// Current benchmark: 13000 ns/op (was 10000 before redisplay)
func BenchmarkBufferDisplayLines(b *testing.B) {
	buf := NewBuffer()
	for i := 0; i < benchBufLines/8; i++ {
		buf.Insert(buf.End(), testSource)
	}
	for i := 0; i < b.N; i++ {
		buf.DisplayLines()
	}
}

// Current benchmark: 21000 ns/op (was 12000 before redisplay)
func BenchmarkBufferGet(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N, benchOpLines)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Get(indexes[i*2], indexes[i*2+1])
	}
}

// Current benchmark: 190000 ns/op (was 31000 before redisplay)
func BenchmarkBufferInsert(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N, benchOpLines)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		text := buf.Get(indexes[i*2], indexes[i*2+1])
		b.StartTimer()
		buf.Insert(indexes[i*2], text)
		b.StopTimer()
		buf.Delete(indexes[i*2], indexes[i*2+1])
		b.StartTimer()
	}
}

// Current benchmark: 1700000 ns/op (was 1200000 before redisplay)
func BenchmarkBufferModified(b *testing.B) {
	buf := randBuffer(benchBufLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Modified()
	}
}

// Current benchmark: 1700000 ns/op (was 1200000 before redisplay)
func BenchmarkBufferResetModified(b *testing.B) {
	buf := randBuffer(benchBufLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.ResetModified()
	}
}

// Current benchmark: 110 ms/op
func BenchmarkBufferSetSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := NewBuffer()
		for i := 0; i < benchBufLines/8; i++ {
			buf.Insert(buf.End(), testSource)
		}
		buf.SetSyntax(goRules)
		b.StartTimer()
		buf.SetSize(1+rand.Int()%160, 25)
	}
}

// Current benchmark: 110 ms/op
func BenchmarkBufferSetSyntax(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := NewBuffer()
		for i := 0; i < benchBufLines/8; i++ {
			buf.Insert(buf.End(), testSource)
		}
		b.StartTimer()
		buf.SetSyntax(goRules)
	}
}

// Current benchmark: 110 ms/op
func BenchmarkBufferSetTabWidth(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := NewBuffer()
		for i := 0; i < benchBufLines/8; i++ {
			buf.Insert(buf.End(), testSource)
		}
		buf.SetSyntax(goRules)
		b.StartTimer()
		buf.SetTabWidth(1 + rand.Int()%8)
	}
}
