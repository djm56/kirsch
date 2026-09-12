package util

import "testing"

func TestTitle(t *testing.T) {
	if got := Title("abc"); got != "Abc" {
		t.Errorf("Title() = %q", got)
	}
}

func TestDivide(t *testing.T) {
	if got := Divide(6, 3); got != 2 {
		t.Errorf("Divide() = %v", got)
	}
}
