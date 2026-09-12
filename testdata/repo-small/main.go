package main

import "fmt"

func main() {
	fmt.Println(Greet("world"))
}

// Greet returns a greeting for name.
func Greet(name string) string {
	return "hello, " + name
}
