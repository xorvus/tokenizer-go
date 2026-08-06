package bpe

import (
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/dlclark/regexp2/v2"
	"github.com/xorvus/tokenizer-go/internal/openai"
)

func encodeParallel(bp *CoreBPE, subText string) ([]int, error) {
	indices, err := bp.Regex.FindAllStringIndex(subText, -1)
	if err != nil {
		return nil, err
	}
	numPieces := len(indices)
	if numPieces < 10000 {
		ret := make([]int, 0, (len(subText)+2)/3)
		for _, pair := range indices {
			ret = bp.EncodePieceTo(subText[pair[0]:pair[1]], ret)
		}
		return ret, nil
	}

	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > 8 {
		numWorkers = 8
	}

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

	totalTokens := 0
	for _, chunk := range chunks {
		totalTokens += len(chunk)
	}

	ret := make([]int, 0, totalTokens)
	for _, chunk := range chunks {
		ret = append(ret, chunk...)
	}
	return ret, nil
}

func TestParallelPieceBPEParity(t *testing.T) {
	ranks := map[string]int{"hello": 1, "world": 2, "test": 3}
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	bp := &CoreBPE{
		RankIndex: NewRankIndex(ranks),
		Regex:     re,
	}

	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		t.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)

	seq, _ := bp.EncodeOrdinaryNative(text)
	par, _ := encodeParallel(bp, text)

	if len(seq) != len(par) {
		t.Fatalf("length mismatch: sequential %d vs parallel %d", len(seq), len(par))
	}
	for i := range seq {
		if seq[i] != par[i] {
			t.Fatalf("token mismatch at %d: seq %d vs par %d", i, seq[i], par[i])
		}
	}
}

func BenchmarkPieceParallelUDHR(b *testing.B) {
	re := regexp2.MustCompile(openai.PatternCL100K, regexp2.None)
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		b.Skip("/tmp/udhr.txt not found")
	}
	text := string(data)
	bp := &CoreBPE{
		RankIndex: NewRankIndex(map[string]int{}),
		Regex:     re,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = encodeParallel(bp, text)
	}
}
