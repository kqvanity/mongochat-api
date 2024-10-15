package main

import (
	"fmt"
	"testing"
)

func TestEscapeJSON(t *testing.T) {
	t.Run("Case 1", func(t *testing.T) {
		s := escapeJSON(" (\\n")
		fmt.Println(s)
	})
}
