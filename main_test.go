package main

import (
	"testing"
	"fmt"
)

func acc(m map[string]string) {
	m["a"] = "b"
}

func TestMap(t *testing.T) {
	m := make(map[string]string)
	m["a"] = "c"
	acc(m)
	fmt.Printf("%v\n", m)
}
