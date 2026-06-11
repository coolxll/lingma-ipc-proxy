//go:build !windows

package lingmaipc

import (
	"context"
	"errors"
)

func ResolvePipePath(explicit string) (string, error) {
	return "", errors.New("Lingma pipe transport currently requires Windows")
}

func connectPipeTransport(ctx context.Context, pipePath string) (framedTransport, error) {
	return nil, errors.New("Lingma pipe transport currently requires Windows")
}
