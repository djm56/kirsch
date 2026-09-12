package util

import "strings"

// Divide is here so search_code has a multi-file match target.
func Divide(a, b float64) float64 {
	return a / b
}

// Title upper-cases the first rune of s.
func Title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
