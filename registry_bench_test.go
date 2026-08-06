package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

func TestResolveModelZeroAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = tokenizer.ResolveModel("gpt-4o")
	})
	if allocs > 0 {
		t.Errorf("ResolveModel allocated %f heap objects, want 0", allocs)
	}
}

func BenchmarkResolveModel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = tokenizer.ResolveModel("gpt-4o")
	}
}
