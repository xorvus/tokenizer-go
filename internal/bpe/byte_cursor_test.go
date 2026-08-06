package bpe

import (
	"testing"
)

func TestByteCursorAdvanceASCII(t *testing.T) {
	text := "hello world"
	c := NewByteCursor(text)
	pos, err := c.AdvanceTo(text, 5)
	if err != nil || pos != 5 {
		t.Fatalf("expected 5, got %d, err %v", pos, err)
	}
}

func TestByteCursorAdvanceUnicode(t *testing.T) {
	text := "你好世界"
	c := NewByteCursor(text)
	pos, err := c.AdvanceTo(text, 2)
	if err != nil || pos != 6 {
		t.Fatalf("expected 6 byte pos for 2 runes, got %d, err %v", pos, err)
	}
}

func TestByteCursorMixedScripts(t *testing.T) {
	text := "hello 世界 test"
	c := NewByteCursor(text)
	pos, err := c.AdvanceTo(text, 6)
	if err != nil || pos != 6 {
		t.Fatalf("expected 6, got %d, err %v", pos, err)
	}
	pos, err = c.AdvanceTo(text, 8)
	if err != nil || pos != 12 {
		t.Fatalf("expected 12, got %d, err %v", pos, err)
	}
}

func TestByteCursorNonMonotonic(t *testing.T) {
	text := "hello world"
	c := NewByteCursor(text)
	_, _ = c.AdvanceTo(text, 5)
	_, err := c.AdvanceTo(text, 2)
	if err != ErrNonMonotonicTarget {
		t.Fatalf("expected ErrNonMonotonicTarget, got %v", err)
	}
}
