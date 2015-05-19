package edit

import "testing"

func TestExpand(t *testing.T) {
	want := "12 123      1234  1"
	if got := expand("12\t123\t\t1234\t1", 3); want != got {
		t.Errorf("expand() == %#v; want %#v", got, want)
	}
	if want, got := "", expand("", 8); want != got { // Test mutex unlock
		t.Errorf("expand() == %#v; want %#v", got, want)
	}
}

func TestColumns(t *testing.T) {
	if want, got := 19, columns("12\t123\t\t1234\t1", 3); want != got {
		t.Errorf("columns() == %#v; want %#v", got, want)
	}
}
