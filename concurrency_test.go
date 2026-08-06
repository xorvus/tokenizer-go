package tokenizer_test

import (
	"sync"
	"testing"

	"github.com/pkoukk/tokenizer-go"
)

func TestConcurrentEncoding(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.CL100KBase)
	if err != nil {
		t.Fatalf("GetEncoding error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tokens, err := tok.EncodeOrdinary("hello world")
			if err != nil {
				t.Errorf("goroutine %d error: %v", id, err)
				return
			}
			if len(tokens) != 2 {
				t.Errorf("goroutine %d len = %d, want 2", id, len(tokens))
			}
		}(i)
	}
	wg.Wait()
}
