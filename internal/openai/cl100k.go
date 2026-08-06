package openai

const PatternCL100K = `(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\r\n\p{L}\p{N}]?\p{L}+|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`

func SpecialTokensCL100K() map[string]int {
	return map[string]int{
		"<|endoftext|>":   100257,
		"<|fim_prefix|>":  100258,
		"<|fim_middle|>":  100259,
		"<|fim_suffix|>":  100260,
		"<|endofprompt|>": 100276,
	}
}
