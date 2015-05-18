package edit

import (
	"math/rand"
	"strings"
	"testing"
)

func TestBuffer(t *testing.T) {
	b := NewBuffer()

	want, got := "", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}

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

	b.Insert(Index{0, 0}, "\n") // Index line too low
	want, got = "\nhello, world!\nhello again!", b.Get(Index{1, 0}, b.End())
	if want != got {
		t.Errorf("Get returned %#v; want %#v", got, want)
	}
}

type insertion struct {
	index Index
	text  string
}

var ascii = "\n !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"

func randString() string {
	r := make([]byte, rand.Int()%80)
	for i := 0; i < len(r); i++ {
		r[i] = ascii[rand.Int()%80]
	}
	return string(r)
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

// Average time to insert a 0-80 character string into a 2000-line buffer at a
// random position.
//
// 2014/05/18 15:48 - 61000 ns/op
// 2014/05/18 17:17 - 19000 ns/op
func BenchmarkBufferInsert(b *testing.B) {
	insertions := make([]insertion, b.N)
	for i := 0; i < b.N; i++ {
		insertions[i] = insertion{
			index: Index{1 + rand.Int()%(1+i%4800/2), rand.Int() % 80},
			text:  randString(),
		}
	}

	b.ResetTimer()

	var buf *Buffer
	for i := 0; i < b.N; i++ {
		if i%4800 == 0 {
			b.StopTimer()
			buf = NewBuffer()
			b.StartTimer()
		}
		buf.Insert(insertions[i].index, insertions[i].text)
	}
}

// Average time to get text in the range of two random indices in a 2000-line
// buffer.
//
// 2014/05/18 15:48 - 230000 ns/op
// 2014/05/18 17:17 - 110000 ns/op
func BenchmarkBufferGet(b *testing.B) {
	buf := randBuffer(2000)
	indexes := make([]Index, b.N*2)
	for i := 0; i < b.N*2; i++ {
		indexes[i] = Index{1 + rand.Int()%2000, rand.Int() % 80}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Get(indexes[i*2], indexes[i*2+1])
	}
}
