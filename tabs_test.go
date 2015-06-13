package edit

import "testing"

func TestExpand(t *testing.T) {
	want := "12 123      1234  1"
	got := string(expand([]rune("12\t123\t\t1234\t1"), 3))
	if want != got {
		t.Errorf("expand() == %#v; want %#v", got, want)
	}
	// test mutex unlock
	want, got = "", string(expand([]rune(""), 8))
	if want != got {
		t.Errorf("expand() == %#v; want %#v", got, want)
	}
}

func TestColumns(t *testing.T) {
	if want, got := 19, columns([]rune("12\t123\t\t1234\t1"), 3); want != got {
		t.Errorf("columns() == %#v; want %#v", got, want)
	}
}
