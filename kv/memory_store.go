package kv

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/spaolacci/murmur3"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Put(ctx context.Context, key []byte, value []byte) error
	Delete(ctx context.Context, key []byte) error
	Close() error
	Name() string

	hash([]byte) (uint64, error)
}

type memoryStore struct {
	mtx sync.RWMutex
	m   map[uint64][]byte
	log *slog.Logger
}

func NewMemoryStore() Store {
	return &memoryStore{
		mtx: sync.RWMutex{},
		m:   map[uint64][]byte{},
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})),
	}
}

var _ Store = &memoryStore{}

func (s *memoryStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	h, err := s.hash(key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	slog.InfoContext(ctx, "Get",
		slog.String("key", string(key)),
		slog.Uint64("hash", h),
	)

	v, ok := s.m[h]
	if !ok {
		return nil, ErrNotFound
	}

	return v, nil
}

func (s *memoryStore) Put(ctx context.Context, key []byte, value []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	h, err := s.hash(key)
	if err != nil {
		return errors.WithStack(err)
	}

	s.m[h] = value
	s.log.InfoContext(ctx, "Put",
		slog.String("key", string(key)),
		slog.Uint64("hash", h),
		slog.String("value", string(value)),
	)

	return nil
}

func (s *memoryStore) Delete(ctx context.Context, key []byte) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	h, err := s.hash(key)
	if err != nil {
		return errors.WithStack(err)
	}

	delete(s.m, h)
	s.log.InfoContext(ctx, "Delete",
		slog.String("key", string(key)),
		slog.Uint64("hash", h),
	)

	return nil
}

func (s *memoryStore) hash(key []byte) (uint64, error) {
	h := murmur3.New64()
	if _, err := h.Write(key); err != nil {
		return 0, errors.WithStack(err)
	}
	return h.Sum64(), nil
}

func (s *memoryStore) Close() error {
	return nil
}

func (s *memoryStore) Name() string {
	return "memory"
}
