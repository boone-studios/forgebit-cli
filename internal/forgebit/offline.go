package forgebit

import (
	"context"
	"os"
	"path/filepath"
)

type OfflineStore struct {
	Dir string
}

func NewOfflineStore() (*OfflineStore, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "forgebit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &OfflineStore{Dir: dir}, nil
}

func (s *OfflineStore) Ping(ctx context.Context) (Status, error) {
	return Status{Source: "offline", Details: s.Dir}, nil
}
