package bpe

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func (bp *CoreBPE) EncodePieceTo(piece string, out []int) []int {
	if token, ok := bp.Encoder[piece]; ok {
		return append(out, token)
	}
	return BytePairEncodeTo(piece, bp.Encoder, out)
}

func (bp *CoreBPE) FindNextAllowedSpecial(text string, startRune int, allowed map[string]any) (SpecialMatch, bool, error) {
	if bp.SpecialRegex == nil || len(allowed) == 0 {
		return SpecialMatch{}, false, nil
	}
	for {
		match, err := bp.SpecialRegex.FindStringMatchStartingAt(text, startRune)
		if err != nil || match == nil {
			return SpecialMatch{}, false, err
		}
		byteStart, byteLength := match.ByteRange()
		token := text[byteStart : byteStart+byteLength]
		if _, ok := allowed[token]; ok {
			return SpecialMatch{
				Token:     token,
				RuneEnd:   match.RuneIndex + match.RuneLength,
				ByteStart: byteStart,
				ByteEnd:   byteStart + byteLength,
			}, true, nil
		}
		nextRune := match.RuneIndex + match.RuneLength
		if nextRune <= startRune {
			nextRune = startRune + 1
		}
		startRune = nextRune
	}
}

func (bp *CoreBPE) EncodeSubTextMatches(subText string, ret []int) ([]int, int, error) {
	lastLen := 0
	match, err := bp.Regex.FindStringMatch(subText)
	if err != nil {
		return nil, 0, err
	}
	ascii := isASCII(subText)
	for match != nil {
		var piece string
		if ascii {
			piece = subText[match.RuneIndex : match.RuneIndex+match.RuneLength]
		} else {
			start, length := match.ByteRange()
			piece = subText[start : start+length]
		}
		before := len(ret)
		ret = bp.EncodePieceTo(piece, ret)
		lastLen = len(ret) - before
		match, err = bp.Regex.FindNextMatch(match)
		if err != nil {
			return nil, 0, err
		}
	}
	return ret, lastLen, nil
}

func (bp *CoreBPE) EncodeNative(text string, allowed map[string]any) ([]int, int, error) {
	ret := make([]int, 0, len(text)/4)
	lastLen := 0
	startRune, startByte := 0, 0
	for {
		sm, found, err := bp.FindNextAllowedSpecial(text, startRune, allowed)
		if err != nil {
			return nil, 0, err
		}
		endByte := len(text)
		if found {
			endByte = sm.ByteStart
		}
		subText := text[startByte:endByte]
		var errEnc error
		ret, lastLen, errEnc = bp.EncodeSubTextMatches(subText, ret)
		if errEnc != nil {
			return nil, 0, errEnc
		}
		if !found {
			break
		}
		ret = append(ret, bp.SpecialTokensEncoder[sm.Token])
		startRune, startByte = sm.RuneEnd, sm.ByteEnd
		lastLen = 0
	}
	return ret, lastLen, nil
}

func (bp *CoreBPE) EncodeOrdinaryNative(text string) ([]int, error) {
	ret := make([]int, 0, len(text)/4)
	ret, _, err := bp.EncodeSubTextMatches(text, ret)
	return ret, err
}
