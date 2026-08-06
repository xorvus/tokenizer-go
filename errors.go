package tokenizer

import "errors"

var (
	ErrUnknownModel       = errors.New("unknown model")
	ErrUnknownEncoding    = errors.New("unknown encoding")
	ErrDisallowedSpecial  = errors.New("disallowed special token")
	ErrUnknownTokenID     = errors.New("unknown token ID")
	ErrInvalidSingleToken = errors.New("text does not map to exactly one token")
)
