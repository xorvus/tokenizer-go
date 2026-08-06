package openai

const PatternO200K = `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`

func SpecialTokensO200K() map[string]int {
	return map[string]int{
		"<|endoftext|>":   199999,
		"<|endofprompt|>": 200018,
	}
}
