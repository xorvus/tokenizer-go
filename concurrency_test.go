package tokenizer_test

import (
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/xorvus/tokenizer-go"
)

func TestConcurrentCallersPercentiles(t *testing.T) {
	data, err := os.ReadFile("/tmp/udhr.txt")
	if err != nil {
		t.Skip("/tmp/udhr.txt not found")
	}
	tok, err := tokenizer.GetEncoding("cl100k_base")
	if err != nil {
		t.Fatalf("failed to load encoding: %v", err)
	}

	callersList := []int{1, 2, 4, 8}
	for _, numCallers := range callersList {
		t.Run("Callers", func(t *testing.T) {
			durations := make([]time.Duration, numCallers)
			var wg sync.WaitGroup
			batchStart := time.Now()
			for i := 0; i < numCallers; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					reqStart := time.Now()
					_, _ = tok.EncodeOrdinary(string(data))
					durations[idx] = time.Since(reqStart)
				}(i)
			}
			wg.Wait()
			batchElapsed := time.Since(batchStart)

			sort.Slice(durations, func(i, j int) bool {
				return durations[i] < durations[j]
			})

			p50 := durations[len(durations)/2]
			p95 := durations[int(float64(len(durations))*0.95)]
			p99 := durations[int(float64(len(durations))*0.99)]

			t.Logf("Callers=%d | BatchWallTime=%v | req/s=%.2f | p50=%v | p95=%v | p99=%v",
				numCallers, batchElapsed, float64(numCallers)/batchElapsed.Seconds(), p50, p95, p99)
		})
	}
}
