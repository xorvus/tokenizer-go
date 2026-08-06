package bpe

import "fmt"

func (bp *CoreBPE) DecodeNative(tokens []int) ([]byte, error) {
	ret := make([]byte, 0, len(tokens)*4)
	for _, token := range tokens {
		if token >= 0 && token < len(bp.DecoderSlice) {
			if val := bp.DecoderSlice[token]; val != "" {
				ret = append(ret, val...)
				continue
			}
		}
		if val, ok := bp.SpecialTokensDecoder[token]; ok {
			ret = append(ret, val...)
			continue
		}
		return nil, fmt.Errorf("unknown token ID: %d", token)
	}
	return ret, nil
}
