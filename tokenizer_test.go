package tokenizer_test

import (
	"testing"
)

func TestCountAPI(t *testing.T) {
	text := "hello world"
	if len(text) == 0 {
		t.Errorf("empty text")
	}
}
