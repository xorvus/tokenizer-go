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
	if len(text) == 0 {
		return []int{}, nil
	}
	if len(text) < 2048 {
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

	it := NewMatchIterator(text, bp.Regex)
	ret := make([]int, 0, len(text)/4)
	for {
		start, end, ok, err := it.Next(text)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		ret = bp.EncodePieceTo(text[start:end], ret)
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
	if len(text) == 0 {
		return 0, nil
	}
	if len(text) < 2048 {
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

	it := NewMatchIterator(text, bp.Regex)
	count := 0
	for {
		start, end, ok, err := it.Next(text)
		if err != nil {
			return 0, err
		}
		if !ok {
			break
		}
		count += bp.CountPiece(text[start:end])
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

// runParallelBatch runs fn over texts, parallel when partitioning pays off.
// Shared by the encode and count batch paths so they stay identical.
func runParallelBatch[T any](texts []string, fn func(string) (T, error)) ([]T, error) {
	n := len(texts)
	results := make([]T, n)
	if n == 0 {
		return results, nil
	}
	if n == 1 {
		v, err := fn(texts[0])
		if err != nil {
			return nil, err
		}
		results[0] = v
		return results, nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > maxBatchWorkers {
		numWorkers = maxBatchWorkers
	}
	bounds := partitionByBytes(texts, numWorkers)
	if len(bounds) <= 2 {
		return runSequential(texts, results, fn)
	}
	return runParallelChunks(texts, results, bounds, fn)
}

func runSequential[T any](texts []string, results []T, fn func(string) (T, error)) ([]T, error) {
	for i := 0; i < len(texts); i++ {
		v, err := fn(texts[i])
		if err != nil {
			return nil, err
		}
		results[i] = v
	}
	return results, nil
}

// runParallelChunks fans out bounds-defined spans of texts across
// goroutines; errOnce captures the first error race-free.
func runParallelChunks[T any](texts []string, results []T, bounds []int, fn func(string) (T, error)) ([]T, error) {
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for w := 0; w < len(bounds)-1; w++ {
		startIdx := bounds[w]
		endIdx := bounds[w+1]

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				v, err := fn(texts[i])
				if err != nil {
					errOnce.Do(func() {
						firstErr = err
					})
					return
				}
				results[i] = v
			}
		}(startIdx, endIdx)
	}
	wg.Wait()
	return results, firstErr
}

func (bp *CoreBPE) EncodeOrdinaryBatchNative(texts []string) ([][]int, error) {
	return runParallelBatch(texts, bp.EncodeOrdinarySequential)
}

func (bp *CoreBPE) CountOrdinaryBatchNative(texts []string) ([]int, error) {
	return runParallelBatch(texts, bp.CountOrdinarySequential)
}

func (bp *CoreBPE) EncodeSubTextMatches(subText string, ret []int) ([]int, int, error) {
	indices, err := bp.Regex.FindAllStringIndex(subText, -1)
	if err != nil {
		return nil, 0, err
	}
	if len(indices) >= 10000 && len(subText) >= 1000000 {
		if res, lastLen, ok := bp.encodeSubTextParallel(subText, indices, ret); ok {
			return res, lastLen, nil
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

// encodeSubTextParallel encodes indices across workers; ok=false when only
// one worker is available, so the caller falls back to sequential.
func (bp *CoreBPE) encodeSubTextParallel(subText string, indices [][]int, ret []int) ([]int, int, bool) {
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > maxParallelWorkers {
		numWorkers = maxParallelWorkers
	}
	if numWorkers <= 1 {
		return nil, 0, false
	}

	chunkSize := (len(indices) + numWorkers - 1) / numWorkers
	chunks := make([][]int, numWorkers)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		startIdx := i * chunkSize
		if startIdx >= len(indices) {
			break
		}
		endIdx := startIdx + chunkSize
		if endIdx > len(indices) {
			endIdx = len(indices)
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
	return ret, lastLen, true
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
