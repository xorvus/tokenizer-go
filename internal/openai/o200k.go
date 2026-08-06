package openai

const PatternO200K = `[^\r\n\p{L}\p{N}]?[\p{Lu}\p{Lt}\p{Lm}\p{Lo}\p{M}]*[\p{Ll}\p{Lm}\p{Lo}\p{M}]+(?i:'s|'t|'re|'ve|'m|'ll|'d)?|[^\r\n\p{L}\p{N}]?[\p{Lu}\p{Lt}\p{Lm}\p{Lo}\p{M}]+[\p{Ll}\p{Lm}\p{Lo}\p{M}]*(?i:'s|'t|'re|'ve|'m|'ll|'d)?|\p{N}{1,3}| ?[^\s\p{L}\p{N}]+[\r\n]*|\s*[\r\n]+|\s+(?!\S)|\s+`

func SpecialTokensO200K() map[string]int {
	return map[string]int{
		"<|endoftext|>":   199999,
		"<|endofprompt|>": 200018,
	}
}

func SpecialTokensO200KHarmony() map[string]int {
	tokens := SpecialTokensO200K()
	tokens["<|startoftext|>"] = 200019
	tokens["<|im_start|>"] = 200020
	tokens["<|im_end|>"] = 200021
	tokens["<|call|>"] = 200022
	tokens["<|return|>"] = 200023
	return tokens
}
