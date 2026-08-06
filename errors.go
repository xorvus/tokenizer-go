package tokenizer

import "errors"

var (
	ErrUnknownModel       = errors.New("unknown model")
	ErrUnknownEncoding    = errors.New("unknown encoding")
	ErrDisallowedSpecial  = errors.New("disallowed special token")
	ErrUnknownTokenID     = errors.New("unknown token ID")
	ErrInvalidSingleToken = errors.New("text does not map to exactly one token")
	ErrEmptyPattern       = errors.New("pattern cannot be empty")
	ErrEmptyVocabulary    = errors.New("vocabulary cannot be empty")
	ErrMissingByteToken   = errors.New("missing single-byte token (0-255 coverage required)")
	ErrNegativeRank       = errors.New("rank cannot be negative")
	ErrDuplicateRank      = errors.New("duplicate rank ID detected")
	ErrTokenConflict             = errors.New("special token conflicts with regular token")
	ErrExactTokenizerUnavailable = errors.New("exact tokenizer unavailable for model")
)
