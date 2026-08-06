package bpe

import (
	"errors"
	"unicode/utf8"
)

var ErrNonMonotonicTarget = errors.New("target rune index must be >= current rune position")

type ByteCursor struct {
	RunePos int
	BytePos int
	ASCII   bool
}

func NewByteCursor(text string) ByteCursor {
	return ByteCursor{
		RunePos: 0,
		BytePos: 0,
		ASCII:   isASCII(text),
	}
}

func (c *ByteCursor) AdvanceTo(text string, targetRune int) (int, error) {
	if targetRune < c.RunePos {
		return 0, ErrNonMonotonicTarget
	}
	if c.ASCII {
		c.RunePos = targetRune
		c.BytePos = targetRune
		return targetRune, nil
	}
	for c.RunePos < targetRune && c.BytePos < len(text) {
		_, size := utf8.DecodeRuneInString(text[c.BytePos:])
		c.BytePos += size
		c.RunePos++
	}
	return c.BytePos, nil
}
