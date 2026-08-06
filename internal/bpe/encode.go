package bpe

import (
	"runtime"
	"sync"

	"github.com/dlclark/regexp2/v2"
)

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func (bp *CoreBPE) EncodePieceTo(piece string, out []int) []int {
	if token, ok := bp.RankIndex.Lookup(piece); ok {
		return append(out, token)
	}
	return BytePairEncodeToWithIndex(piece, &bp.RankIndex, out)
}

func (bp *CoreBPE) FindNextAllowedSpecial(text string, startRune int, allowed map[string]any) (SpecialMatch, bool, error) {
	if bp.SpecialRegex == nil || len(allowed) == 0 {
		return SpecialMatch{}, false, nil
	}
	ascii := isASCII(text)
	for {
		match, err := bp.SpecialRegex.FindStringMatchStartingAt(text, startRune)
		if err != nil || match == nil {
			return SpecialMatch{}, false, err
		}
		var byteStart, byteLength int
		if ascii {
			byteStart = match.RuneIndex
			byteLength = match.RuneLength
		} else {
			byteStart, byteLength = match.ByteRange()
		}
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

type MatchIterator struct {
	re     *regexp2.Regexp
	cursor ByteCursor
	match  *regexp2.Match
	err    error
}

func NewMatchIterator(text string, re *regexp2.Regexp) MatchIterator {
	m, err := re.FindStringMatch(text)
	return MatchIterator{
		re:     re,
		cursor: NewByteCursor(text),
		match:  m,
		err:    err,
	}
}

func (it *MatchIterator) Next(text string) (int, int, bool, error) {
	if it.err != nil || it.match == nil {
		return 0, 0, false, it.err
	}
	var start, end int
	if it.cursor.ASCII {
		start = it.match.RuneIndex
		end = start + it.match.RuneLength
	} else {
		byteStart, err := it.cursor.AdvanceTo(text, it.match.RuneIndex)
		if err != nil {
			return 0, 0, false, err
		}
		byteEnd, err := it.cursor.AdvanceTo(text, it.match.RuneIndex+it.match.RuneLength)
		if err != nil {
			return 0, 0, false, err
		}
		start, end = byteStart, byteEnd
	}
	nextMatch, err := it.re.FindNextMatch(it.match)
	it.match = nextMatch
	it.err = err
	return start, end, true, nil
}

const maxParallelWorkers = 4
const maxBatchWorkers = 4

func (bp *CoreBPE) EncodeOrdinarySequential(text string) ([]int, error) {
	indices, err := bp.Regex.FindAllStringIndex(text, -1)
	if err != nil {
		return nil, err
	}
	ret := make([]int, 0, len(indices))
	for _, pair := range indices {
		ret = bp.EncodePieceTo(text[pair[0]:pair[1]], ret)
	}
	return ret, nil
}

func (bp *CoreBPE) CountPiece(piece string) int {
	if _, ok := bp.RankIndex.Lookup(piece); ok {
		return 1
	}
	return CountPieceToWithIndex(piece, &bp.RankIndex)
}

func (bp *CoreBPE) CountOrdinarySequential(text string) (int, error) {
	indices, err := bp.Regex.FindAllStringIndex(text, -1)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, pair := range indices {
		count += bp.CountPiece(text[pair[0]:pair[1]])
	}
	return count, nil
}

func partitionByBytes(texts []string, maxWorkers int) []int {
	n := len(texts)
	if maxWorkers <= 1 || n <= 1 {
		return []int{0, n}
	}
	totalBytes := 0
	for _, t := range texts {
		totalBytes += len(t)
	}
	if totalBytes < 16384 {
		return []int{0, n}
	}
	targetBytesPerWorker := (totalBytes + maxWorkers - 1) / maxWorkers
	bounds := make([]int, 0, maxWorkers+1)
	bounds = append(bounds, 0)
	currentBytes := 0
	for i, t := range texts {
		currentBytes += len(t)
		if currentBytes >= targetBytesPerWorker && len(bounds) < maxWorkers {
			bounds = append(bounds, i+1)
			currentBytes = 0
		}
	}
	if bounds[len(bounds)-1] != n {
		bounds = append(bounds, n)
	}
	return bounds
}

func (bp *CoreBPE) EncodeOrdinaryBatchNative(texts []string) ([][]int, error) {
	n := len(texts)
	if n == 0 {
		return [][]int{}, nil
	}

	results := make([][]int, n)
	if n == 1 {
		tokens, err := bp.EncodeOrdinarySequential(texts[0])
		if err != nil {
			return nil, err
		}
		results[0] = tokens
		return results, nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > maxBatchWorkers {
		numWorkers = maxBatchWorkers
	}
	bounds := partitionByBytes(texts, numWorkers)
	numPartitions := len(bounds) - 1

	if numPartitions <= 1 {
		for i := 0; i < n; i++ {
			tokens, err := bp.EncodeOrdinarySequential(texts[i])
			if err != nil {
				return nil, err
			}
			results[i] = tokens
		}
		return results, nil
	}

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for w := 0; w < numPartitions; w++ {
		startIdx := bounds[w]
		endIdx := bounds[w+1]

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				tokens, err := bp.EncodeOrdinarySequential(texts[i])
				if err != nil {
					errOnce.Do(func() {
						firstErr = err
					})
					return
				}
				results[i] = tokens
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (bp *CoreBPE) CountOrdinaryBatchNative(texts []string) ([]int, error) {
	n := len(texts)
	if n == 0 {
		return []int{}, nil
	}

	results := make([]int, n)
	if n == 1 {
		count, err := bp.CountOrdinarySequential(texts[0])
		if err != nil {
			return nil, err
		}
		results[0] = count
		return results, nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > maxBatchWorkers {
		numWorkers = maxBatchWorkers
	}
	bounds := partitionByBytes(texts, numWorkers)
	numPartitions := len(bounds) - 1

	if numPartitions <= 1 {
		for i := 0; i < n; i++ {
			count, err := bp.CountOrdinarySequential(texts[i])
			if err != nil {
				return nil, err
			}
			results[i] = count
		}
		return results, nil
	}

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for w := 0; w < numPartitions; w++ {
		startIdx := bounds[w]
		endIdx := bounds[w+1]

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				count, err := bp.CountOrdinarySequential(texts[i])
				if err != nil {
					errOnce.Do(func() {
						firstErr = err
					})
					return
				}
				results[i] = count
			}
		}(startIdx, endIdx)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (bp *CoreBPE) EncodeSubTextMatches(subText string, ret []int) ([]int, int, error) {
	indices, err := bp.Regex.FindAllStringIndex(subText, -1)
	if err != nil {
		return nil, 0, err
	}
	numPieces := len(indices)
	if numPieces >= 10000 && len(subText) >= 1000000 {
		numWorkers := runtime.GOMAXPROCS(0)
		if numWorkers > maxParallelWorkers {
			numWorkers = maxParallelWorkers
		}
		if numWorkers > 1 {
			chunkSize := (numPieces + numWorkers - 1) / numWorkers
			chunks := make([][]int, numWorkers)
			var wg sync.WaitGroup
			for i := 0; i < numWorkers; i++ {
				startIdx := i * chunkSize
				if startIdx >= numPieces {
					break
				}
				endIdx := startIdx + chunkSize
				if endIdx > numPieces {
					endIdx = numPieces
				}
				wg.Add(1)
				go func(workerID, start, end int) {
					defer wg.Done()
					pieceIndices := indices[start:end]
					buf := make([]int, 0, (end-start)*2)
					for _, pair := range pieceIndices {
						buf = bp.EncodePieceTo(subText[pair[0]:pair[1]], buf)
					}
					chunks[workerID] = buf
				}(i, startIdx, endIdx)
			}
			wg.Wait()
			lastLen := 0
			for _, chunk := range chunks {
				ret = append(ret, chunk...)
				if len(chunk) > 0 {
					lastLen = len(chunk)
				}
			}
			return ret, lastLen, nil
		}
	}

	lastLen := 0
	for _, pair := range indices {
		before := len(ret)
		ret = bp.EncodePieceTo(subText[pair[0]:pair[1]], ret)
		lastLen = len(ret) - before
	}
	return ret, lastLen, nil
}

func (bp *CoreBPE) EncodeNative(text string, allowed map[string]any) ([]int, int, error) {
	ret := make([]int, 0, (len(text)+2)/3)
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
	ret := make([]int, 0, (len(text)+2)/3)
	ret, _, err := bp.EncodeSubTextMatches(text, ret)
	return ret, err
}
