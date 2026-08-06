package tokenizer_test

import (
	"testing"

	"github.com/xorvus/tokenizer-go"
)

type dummyEngine struct{}

func (d *dummyEngine) Encode(text string) ([]int, error) { return []int{1, 2}, nil }
func (d *dummyEngine) Decode(tokens []int) (string, error) { return "hello", nil }
func (d *dummyEngine) Count(text string) (int, error) { return 2, nil }

func TestEngineContract(t *testing.T) {
	var eng tokenizer.Engine = &dummyEngine{}
	tokens, err := eng.Encode("hello")
	if err != nil || len(tokens) != 2 {
		t.Fatalf("unexpected encode result: %v, %v", tokens, err)
	}
}
