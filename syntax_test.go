package edit

import "testing"

func TestNewRule(t *testing.T) {
	if _, err := NewRule("b", "", 0); err != nil {
		t.Error("NewRule returned error for valid expressions")
	}
	if _, err := NewRule("b", "e", 0); err != nil {
		t.Error("NewRule returned error for valid expressions")
	}
	if _, err := NewRule("b\\", "", 0); err == nil {
		t.Error("NewRule did not return error for invalid begin expr")
	}
	if _, err := NewRule("b", "e\\", 0); err == nil {
		t.Error("NewRule did not return error for invalid end expr")
	}
}

func TestSyntaxSplit(t *testing.T) {
	// Test empty string
	var rules syntax = []Rule{}
	c := rules.split("")
	if _, ok := <-c; ok {
		t.Error("channel should be closed")
	}

	// Test no rules
	c = rules.split(" ")
	if want, got := (Fragment{" ", noneTag}), <-c; want != got {
		t.Errorf("<-c == %#v; want %#v", got, want)
	}
	if _, ok := <-c; ok {
		t.Error("channel should be closed")
	}

	// Test begin-only rule
	keywordRule, _ := NewRule("(var|const)", "", 0)
	rules = []Rule{keywordRule}
	fragments := []Fragment{{"var", 0}, {" def ", noneTag}, {"const", 0}}
	c = rules.split("var def const")
	for _, frag := range fragments {
		if want, got := frag, <-c; want != got {
			t.Errorf("<-c == %#v; want %#v", got, want)
		}
	}
	if _, ok := <-c; ok {
		t.Error("channel should be closed")
	}

	// Test begin and end rules
	commentRule, _ := NewRule(`/\*`, `\*/`, 1)
	rules = []Rule{keywordRule, commentRule}
	fragments = []Fragment{{"var", 0}, {"/*var*/", 1}, {"const", 0},
		{"/*const", noneTag}}
	c = rules.split("var/*var*/const/*const")
	for _, frag := range fragments {
		if want, got := frag, <-c; want != got {
			t.Errorf("<-c == %#v; want %#v", got, want)
		}
	}
	if _, ok := <-c; ok {
		t.Error("channel should be closed")
	}
}

const testSource = `package main

import "fmt"

func main() {
	v := 42 // change me!
	fmt.Printf("v is of type %T\n", v)
}
`

var (
	// Simplified set of rules for highlighting go source code
	keywordRule, _ = NewRule(`\b(break|case|chan|const|continue|default|`+
		`defer|else|fallthrough|for|func|go|goto|if|import|interface|map|`+
		`package|range|return|select|struct|switch|type|var)\b`, "", 0)
	numRule, _     = NewRule(`\b\d+(\.\d+)?\b`, "", 1)
	stringRule, _  = NewRule(`"`, `"`, 1)
	commentRule, _ = NewRule(`//`, "($|\n)", 2)

	goRules syntax = []Rule{keywordRule, numRule, stringRule, commentRule}
)

// Current benchmark: 340000 ns/op
func BenchmarkSyntaxSplit(b *testing.B) {
	// Make sure the rules work
	fragments := []Fragment{{"package", 0}, {" main\n\n", noneTag},
		{"import", 0}, {" ", noneTag}, {`"fmt"`, 1}, {"\n\n", noneTag},
		{"func", 0}, {" main() {\n\tv := ", noneTag}, {"42", 1},
		{" ", noneTag}, {"// change me!\n", 2}, {"\tfmt.Printf(", noneTag},
		{`"v is of type %T\n"`, 1}, {", v)\n}\n", noneTag}}
	c := goRules.split(testSource)
	for _, frag := range fragments {
		if want, got := frag, <-c; want != got {
			b.Fatalf("<-c == %#v; want %#v", got, want)
		}
	}
	if _, ok := <-c; ok {
		b.Fatal("channel should be closed")
	}

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c = goRules.split(testSource)
		for _ = range c {
			// Do nothing
		}
	}
}
