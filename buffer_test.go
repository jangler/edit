package edit

import (
	"math/rand"
	"strings"
	"testing"
)

func TestBuffer(t *testing.T) {
	// Init tests
	b := NewBuffer()
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

func randBuffer(numLines int) *Buffer {
	buf := NewBuffer()
	lines := make([]string, numLines)
	line := make([]byte, 80)
	for i := 0; i < numLines; i++ {
		lineLen := rand.Int() % 80
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
		begin := b.clip(Index{1 + rand.Int()%b.lines.Len(), rand.Int() % 80})
		end := b.clip(Index{1 + rand.Int()%b.lines.Len(), rand.Int() % 80})
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

// Average time to delete text of up to 25 lines at a random index in a
// 2000-line buffer.
//
// Current benchmark: 16000 ns/op
func BenchmarkBufferDelete(b *testing.B) {
	buf := randBuffer(2000)
	indexes := randIndexes(buf, b.N, 25)

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

// Average time to get text of up to 25 lines at a random index in a 2000-line
// buffer.
//
// Current benchmark: 12000 ns/op
func BenchmarkBufferGet(b *testing.B) {
	buf := randBuffer(2000)
	indexes := randIndexes(buf, b.N, 25)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Get(indexes[i*2], indexes[i*2+1])
	}
}

// Average time to insert text of up to 25 lines at a random index in a
// 2000-line buffer.
//
// Current benchmark: 27000 ns/op
func BenchmarkBufferInsert(b *testing.B) {
	buf := randBuffer(2000)
	indexes := randIndexes(buf, b.N, 25)

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

// Average time to check modified state in a 2000-line buffer.
// Current benchmark: 1200000 ns/op
func BenchmarkBufferModified(b *testing.B) {
	buf := randBuffer(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Modified()
	}
}

// Average time to reset modified state in a 2000-line buffer.
// Current benchmark: 1200000 ns/op
func BenchmarkBufferResetModified(b *testing.B) {
	buf := randBuffer(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.ResetModified()
	}
}
