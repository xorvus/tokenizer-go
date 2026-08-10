package tokenizer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xorvus/tokenizer-go"
)

func TestContextCancellation(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}

	// 1. Pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = tok.EncodeContext(ctx, "hello world")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("EncodeContext expected context.Canceled, got %v", err)
	}

	_, err = tok.CountContext(ctx, "hello world")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("CountContext expected context.Canceled, got %v", err)
	}

	texts := []string{"text 1", "text 2", "text 3"}
	_, err = tok.EncodeOrdinaryBatchContext(ctx, texts)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("EncodeOrdinaryBatchContext expected context.Canceled, got %v", err)
	}

	_, err = tok.CountOrdinaryBatchContext(ctx, texts)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("CountOrdinaryBatchContext expected context.Canceled, got %v", err)
	}

	// 2. Timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer timeoutCancel()
	time.Sleep(2 * time.Millisecond)

	_, err = tok.EncodeContext(timeoutCtx, "hello world")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("EncodeContext expected context.DeadlineExceeded, got %v", err)
	}
}

func TestContextBatchSuccessAndNil(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}
	texts := []string{"text 1", "text 2", "text 3"}

	got, err := tok.CountOrdinaryBatchContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("CountOrdinaryBatchContext error: %v", err)
	}
	want, err := tok.CountOrdinaryBatch(texts)
	if err != nil {
		t.Fatalf("CountOrdinaryBatch error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("CountOrdinaryBatchContext = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("CountOrdinaryBatchContext[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	enc, err := tok.EncodeOrdinaryBatchContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("EncodeOrdinaryBatchContext error: %v", err)
	}
	encWant, err := tok.EncodeOrdinaryBatch(texts)
	if err != nil {
		t.Fatalf("EncodeOrdinaryBatch error: %v", err)
	}
	if len(enc) != len(encWant) {
		t.Fatalf("EncodeOrdinaryBatchContext len = %d, want %d", len(enc), len(encWant))
	}

	// nil context is allowed and proceeds
	if _, err := tok.CountContext(nil, "hello"); err != nil {
		t.Errorf("CountContext(nil) error: %v", err)
	}
	if _, err := tok.EncodeContext(nil, "hello"); err != nil {
		t.Errorf("EncodeContext(nil) error: %v", err)
	}
}

func TestEncodeSingleTokenErrors(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}
	// text that splits into multiple tokens
	if _, err := tok.EncodeSingleToken("supercalifragilisticexpialidocious"); err == nil {
		t.Error("expected ErrInvalidSingleToken for multi-token text")
	}
	// unknown token id
	if _, err := tok.DecodeSingleTokenBytes(9999999); err == nil {
		t.Error("expected error for unknown token id")
	}
}

func TestWithOptionsConfiguration(t *testing.T) {
	tok, err := tokenizer.GetEncoding(tokenizer.O200KBase)
	if err != nil {
		t.Fatalf("failed loading o200k_base: %v", err)
	}

	customTok := tok.WithOptions(tokenizer.Options{
		MaxWorkers:            2,
		ParallelByteThreshold: 8192,
		BatchByteThreshold:    8192,
	})

	tokens, err := customTok.EncodeOrdinary("Hello world from options test")
	if err != nil {
		t.Fatalf("customTok EncodeOrdinary error: %v", err)
	}
	if len(tokens) == 0 {
		t.Errorf("expected non-empty tokens")
	}
}
