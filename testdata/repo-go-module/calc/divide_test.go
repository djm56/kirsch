package calc

import "testing"

func TestDivide(t *testing.T) {
	got, err := Divide(6, 3)
	if err != nil {
		t.Fatalf("Divide() error = %v", err)
	}
	if got != 2 {
		t.Errorf("Divide(6, 3) = %v, want 2", got)
	}
}
