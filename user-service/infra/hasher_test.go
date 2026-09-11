package infra

import (
	"context"
	"errors"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHasher_RoundTrip(t *testing.T) {
	h := NewHasher(1)
	ctx := context.Background()

	hash, err := h.Generate(ctx, "s3cret-pass", bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == "s3cret-pass" {
		t.Fatal("hash must not be plaintext")
	}

	if err := h.Compare(ctx, hash, []byte("s3cret-pass")); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := h.Compare(ctx, hash, []byte("wrong-pass")); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestHasher_InvalidCost(t *testing.T) {
	h := NewHasher(0)
	if _, err := h.Generate(context.Background(), "pw", 72); err == nil {
		t.Fatal("bcrypt must reject cost > 31")
	}
}

func TestHasher_CancelledContext(t *testing.T) {
	// Fill the semaphore so Generate/Compare block; a cancelled ctx then wins
	// the select and returns before the expensive bcrypt kernel runs.
	h := &Hasher{sem: make(chan struct{}, 1)}
	h.sem <- struct{}{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := h.Generate(ctx, "pw", bcrypt.MinCost); !errors.Is(err, context.Canceled) {
		t.Fatalf("generate err = %v, want canceled", err)
	}
	if err := h.Compare(ctx, []byte("x"), []byte("pw")); !errors.Is(err, context.Canceled) {
		t.Fatalf("compare err = %v, want canceled", err)
	}
}

func TestHasher_ConcurrencyLimit(t *testing.T) {
	// maxConcurrent=1 serializes bcrypt work without deadlocking.
	h := NewHasher(1)
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hash, err := h.Generate(ctx, "pw", bcrypt.MinCost)
			if err != nil {
				errs <- err
				return
			}
			errs <- h.Compare(ctx, hash, []byte("pw"))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent hashing: %v", err)
		}
	}
}
