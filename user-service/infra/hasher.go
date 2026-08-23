package infra

import (
	"context"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
	sem chan struct{}
}

func NewHasher(maxConcurrent int) *Hasher {
	if maxConcurrent <= 0 {
		maxConcurrent = runtime.NumCPU()
	}
	return &Hasher{sem: make(chan struct{}, maxConcurrent)}
}

func (h *Hasher) Generate(ctx context.Context, password string, cost int) ([]byte, error) {
	select {
	case h.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-h.sem }()

	return bcrypt.GenerateFromPassword([]byte(password), cost)
}

func (h *Hasher) Compare(ctx context.Context, hash, password []byte) error {
	select {
	case h.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-h.sem }()

	return bcrypt.CompareHashAndPassword(hash, password)
}
