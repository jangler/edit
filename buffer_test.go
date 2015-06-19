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
	iRule, _ := NewRule(`\b\w*i\w*\b`, 0)
	b.SetSyntax([]Rule{iRule})
	b.SetSize(0, 1)  // Make sure 0 cols is ok--should correct to 1
	b.SetSize(1, -1) // Make sure -1 rows is ok--should correct to 0
	b.SetSize(8, 9)
	b.SetTabWidth(4)
	b.Insert(Index{1, 0},
		"package main\nfunc main() {\n\tprintln(\"hi\")\n}\n")
	b.SetSize(3, 3)
	b.SetSize(8, 9)
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
	dLines[8].PushBack(Fragment{"", noneTag})
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

func TestBufferShiftIndex(t *testing.T) {
	b := NewBuffer()
	b.Insert(b.End(), testSource)
	// No-op
	if want, got := (Index{3, 3}), b.ShiftIndex(Index{3, 3}, 0); want != got {
		t.Errorf("ShiftIndex() == %#v; want %#v", got, want)
	}
	// Don't overshoot
	if wnt, got := (Index{8, 1}), b.ShiftIndex(Index{1, 0}, 102); wnt != got {
		t.Errorf("ShiftIndex() == %#v; want %#v", got, wnt)
	}
	if want, got := (Index{1, 1}), b.ShiftIndex(b.End(), -102); want != got {
		t.Errorf("ShiftIndex() == %#v; want %#v", got, want)
	}
	// Overshoot
	if want, got := b.End(), b.ShiftIndex(Index{1, 0}, 104); want != got {
		t.Errorf("ShiftIndex() == %#v; want %#v", got, want)
	}
	if want, got := (Index{1, 0}), b.ShiftIndex(b.End(), -104); want != got {
		t.Errorf("ShiftIndex() == %#v; want %#v", got, want)
	}
}

func TestBufferMark(t *testing.T) {
	// invalid ID
	b := NewBuffer()
	if want, got := (Index{0, 0}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}

	// valid ID
	b.Mark(b.End(), 0)
	if want, got := (Index{1, 0}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}

	// insertion
	b.Insert(b.End(), "hello")
	if want, got := (Index{1, 5}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}
	b.Insert(b.End(), "\nhi")
	if want, got := (Index{2, 2}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}
	b.Insert(Index{1, 0}, "\n")
	if want, got := (Index{3, 2}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}

	// deletion
	b.Delete(Index{1, 0}, Index{2, 0})
	if want, got := (Index{2, 2}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}
	b.Delete(Index{1, 0}, Index{2, 1})
	if want, got := (Index{1, 1}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}
	b.Delete(Index{1, 0}, b.End())
	if want, got := (Index{1, 0}), b.IndexFromMark(0); want != got {
		t.Errorf("IndexFromMark() == %v; want %v", got, want)
	}
}

func TestBufferIndexCoords(t *testing.T) {
	// IndexFromCoords
	b := NewBuffer()
	if want, got := (Index{1, 0}), b.IndexFromCoords(-1, -1); want != got {
		t.Errorf("IndexFromCoords() == %v; want %v", got, want)
	}
	if want, got := (Index{1, 0}), b.IndexFromCoords(1, 1); want != got {
		t.Errorf("IndexFromCoords() == %v; want %v", got, want)
	}
	b.SetSize(4, 4)
	b.Insert(b.End(), "\n\thello")
	if want, got := (Index{2, 3}), b.IndexFromCoords(2, 3); want != got {
		t.Errorf("IndexFromCoords() == %v; want %v", got, want)
	}

	// CoordsFromIndex
	wX, wY := 2, 3
	if gX, gY := b.CoordsFromIndex(Index{2, 3}); wX != gX || wY != gY {
		t.Errorf("CoordsFromIndex() == %v, %v; want %v, %v", gX, gY, wX, wY)
	}
}

func TestBufferScroll(t *testing.T) {
	b := NewBuffer()
	b.SetSize(80, 1)
	if want, got := -1.0, b.ScrollFraction(); want != got {
		t.Errorf("ScrollFraction() == %v; want %v", got, want)
	}
	b.Insert(b.End(), "hello\nthere\nworld")
	if want, got := 0.0, b.ScrollFraction(); want != got {
		t.Errorf("ScrollFraction() == %v; want %v", got, want)
	}
	b.Scroll(1)
	if want, got := 0.5, b.ScrollFraction(); want != got {
		t.Errorf("ScrollFraction() == %v; want %v", got, want)
	}
	b.Scroll(2)
	if want, got := 1.0, b.ScrollFraction(); want != got {
		t.Errorf("ScrollFraction() == %v; want %v", got, want)
	}
	b.Scroll(-3)
	if want, got := 0.0, b.ScrollFraction(); want != got {
		t.Errorf("ScrollFraction() == %v; want %v", got, want)
	}
}

func TestBufferUndo(t *testing.T) {
	b := NewBuffer()

	// test operations on new buffer
	if want, got := false, b.Undo(); want != got {
		t.Errorf("b.Undo() == %v, want %v", got, want)
	}
	if want, got := false, b.Redo(); want != got {
		t.Errorf("b.Redo() == %v, want %v", got, want)
	}

	// test undo/redo single insertion
	b.Insert(b.End(), "hello")
	if want, got := true, b.Undo(); want != got {
		t.Errorf("b.Undo() == %v, want %v", got, want)
	}
	if want, got := "", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	if want, got := true, b.Redo(); want != got {
		t.Errorf("b.Redo() == %v, want %v", got, want)
	}
	if want, got := "hello", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}

	// test undo/redo single deletion
	b = NewBuffer()
	b.Insert(b.End(), "hello")
	b.Separate()
	b.Delete(Index{1, 1}, Index{1, 4})
	b.Undo()
	if want, got := "hello", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	b.Redo()
	if want, got := "ho", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}

	// test groups of operations and marks
	b = NewBuffer()
	b.Separate()
	b.Insert(b.End(), "hello")
	b.Insert(b.End(), ", world!")
	b.Separate()
	b.Separate()
	b.Delete(Index{1, 7}, Index{1, 12})
	b.Insert(Index{1, 7}, "there")
	b.Separate()
	b.Undo(1)
	if want, got := "hello, world!", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	if want, got := (Index{1, 12}), b.IndexFromMark(1); want != got {
		t.Errorf("b.IndexFromMark() == %v, want %v", got, want)
	}
	b.Undo()
	if want, got := "", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	if want, got := false, b.Undo(); want != got {
		t.Errorf("b.Undo() == %v, want %v", got, want)
	}
	b.Redo()
	if want, got := "hello, world!", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	b.Redo(1)
	if want, got := "hello, there!", b.Get(Index{1, 0}, b.End()); want != got {
		t.Errorf("b.Get() == %v, want %v", got, want)
	}
	if want, got := (Index{1, 12}), b.IndexFromMark(1); want != got {
		t.Errorf("b.IndexFromMark() == %v, want %v", got, want)
	}
	if want, got := false, b.Redo(); want != got {
		t.Errorf("b.Redo() == %v, want %v", got, want)
	}

	// test operation merging
	b = NewBuffer()
	b.Insert(b.End(), "b")
	b.Insert(Index{1, 0}, "a")
	b.Insert(b.End(), "c")
	if want, got := 1, b.undo.Len(); want != got {
		t.Errorf("b.undo.Len() == %v, want %v", got, want)
	}
	b.Delete(Index{1, 1}, Index{1, 2})
	b.Delete(Index{1, 2}, Index{1, 3})
	b.Delete(Index{1, 0}, Index{1, 1})
	if want, got := 2, b.undo.Len(); want != got {
		t.Errorf("b.undo.Len() == %v, want %v", got, want)
	}
	b.Undo()
	if want, got := 0, b.undo.Len(); want != got {
		t.Errorf("b.undo.Len() == %v, want %v", got, want)
	}

	// test reset
	b.ResetUndo()
	if want, got := false, b.Redo(); want != got {
		t.Errorf("b.Redo() == %v, want %v", got, want)
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

// Current benchmark: 30000 ns/op
func BenchmarkBufferCoordsFromIndex(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N/2+1, benchOpLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.CoordsFromIndex(indexes[i])
	}
}

// Current benchmark: 130000 ns/op (86000 ns/op before []rune)
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

// Current benchmark: 29000 ns/op
func BenchmarkBufferDisplayLines(b *testing.B) {
	buf := NewBuffer()
	for i := 0; i < benchBufLines/8; i++ {
		buf.Insert(buf.End(), testSource)
	}
	for i := 0; i < b.N; i++ {
		buf.DisplayLines()
	}
}

// Current benchmark: 48000 ns/op (21000 ns/op before []rune)
func BenchmarkBufferGet(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N, benchOpLines)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Get(indexes[i*2], indexes[i*2+1])
	}
}

// Current benchmark: 47000 ns/op
func BenchmarkBufferIndexFromCoords(b *testing.B) {
	buf := randBuffer(benchBufLines)
	coords := make([][]int, b.N)
	for i := 0; i < b.N; i++ {
		coords[i] = []int{rand.Int() % benchMaxLine,
			rand.Int() % benchBufLines}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.IndexFromCoords(coords[i][0], coords[i][1])
	}
}

// Current benchmark: 290000 ns/op (200000 ns/op before []rune)
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

// Current benchmark: 4400000 ns/op (1500000 ns/op before []rune)
func BenchmarkBufferModified(b *testing.B) {
	buf := randBuffer(benchBufLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Modified()
	}
}

// Current benchmark: 4400000 ns/op (1500000 ns/op before []rune)
func BenchmarkBufferResetModified(b *testing.B) {
	buf := randBuffer(benchBufLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.ResetModified()
	}
}

// Current benchmark: 8700000 ns/op
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

// Current benchmark: 120 ms/op
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

// Current benchmark: 7600000 ns/op
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

// Current benchmark: 12000 ns/op
func BenchmarkBufferShiftIndex(b *testing.B) {
	buf := randBuffer(benchBufLines)
	indexes := randIndexes(buf, b.N/2+1, benchOpLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.ShiftIndex(indexes[i], rand.Int()%51-25)
	}
}
