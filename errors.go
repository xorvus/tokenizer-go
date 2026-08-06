package tokenizer

import "errors"

var (
	ErrUnknownEncoding   = errors.New("unknown encoding")
	ErrDisallowedSpecial = errors.New("disallowed special token")
	ErrUnknownTokenID    = errors.New("unknown token ID")
)
