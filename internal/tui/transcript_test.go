package tui

import (
	"testing"
)

func TestTrimTrailingEmpty(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "a\nb\n",
			expected: []string{"a", "b"},
		},
		{
			input:    "a\n\n",
			expected: []string{"a", ""},
		},
		{
			input:    "\n",
			expected: []string{""},
		},
		{
			input:    "a\r\nb\r\n",
			expected: []string{"a", "b"},
		},
		{
			input:    "",
			expected: []string{""},
		},
		{
			input:    "a",
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := trimTrailingEmpty(SanitizeLines(tt.input), tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("length = %d, want %d; got %v", len(got), len(tt.expected), got)
			}
			for i, exp := range tt.expected {
				if i >= len(got) {
					t.Errorf("missing element at index %d", i)
					break
				}
				if got[i] != exp {
					t.Errorf("element %d = %q, want %q", i, got[i], exp)
				}
			}
		})
	}
}

func TestSanitizeLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single newline",
			input: "\n",
			want:  []string{"", ""},
		},
		{
			name:  "content with trailing empty",
			input: "a\n\n",
			want:  []string{"a", "", ""},
		},
		{
			name:  "CRLF line endings normalized to LF",
			input: "a\r\nb\r\n",
			want:  []string{"a", "b", ""},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("length = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if i >= len(got) {
					t.Errorf("missing element at index %d", i)
					break
				}
				if got[i] != tt.want[i] {
					t.Errorf("element %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestToAppOutcome(t *testing.T) {
	tests := []struct {
		name  string
		input ApprovalOutcome
		want  int
	}{
		{
			name:  "Approved maps to 0 (ApprovalOutcomeOnce)",
			input: Approved,
			want:  0,
		},
		{
			name:  "ApprovedSession maps to 1 (ApprovalOutcomeSession)",
			input: ApprovedSession,
			want:  1,
		},
		{
			name:  "Rejected maps to 2 (ApprovalOutcomeDeny)",
			input: Rejected,
			want:  2,
		},
		{
			name:  "Cancelled maps to 3 (ApprovalOutcomeCancelled)",
			input: Cancelled,
			want:  3,
		},
		{
			name:  "Unresolved maps to 3 (ApprovalOutcomeCancelled)",
			input: Unresolved,
			want:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToAppOutcome(tt.input)
			if got != tt.want {
				t.Errorf("ToAppOutcome(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
