//go:build windows

package lingmaipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"

	winio "github.com/Microsoft/go-winio"
)

func ResolvePipePath(explicit string) (string, error) {
	if pipe := strings.TrimSpace(explicit); pipe != "" {
		return normalizePipePath(pipe), nil
	}
	if pipe := strings.TrimSpace(os.Getenv("LINGMA_IPC_PIPE")); pipe != "" {
		return normalizePipePath(pipe), nil
	}
	if info, err := resolveSharedClientInfo(); err == nil {
		if pipe := strings.TrimSpace(info.IPCServerPath); pipe != "" {
			return normalizePipePath(pipe), nil
		}
	}

	entries, err := os.ReadDir(PipeDir)
	if err != nil {
		return "", fmt.Errorf("enumerate Lingma named pipes: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, PipePrefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", errors.New("no active Lingma named pipe was found")
	}
	return PipeDir + names[len(names)-1], nil
}

type pipeTransport struct {
	path   string
	conn   net.Conn
	reader *framedReader
	write  sync.Mutex
}

func connectPipeTransport(ctx context.Context, pipePath string) (*pipeTransport, error) {
	conn, err := winio.DialPipeContext(ctx, pipePath)
	if err != nil {
		return nil, fmt.Errorf("connect Lingma IPC pipe %s: %w", pipePath, err)
	}
	return &pipeTransport{
		path:   pipePath,
		conn:   conn,
		reader: newFramedReader(conn),
	}, nil
}

func (t *pipeTransport) ReadFrame() ([]byte, error) {
	return t.reader.ReadFrame()
}

func (t *pipeTransport) WriteFrame(body []byte) error {
	t.write.Lock()
	defer t.write.Unlock()

	frame := []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)))
	if _, err := t.conn.Write(frame); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := t.conn.Write(body); err != nil {
		return fmt.Errorf("write frame body: %w", err)
	}
	return nil
}

func (t *pipeTransport) Close() error {
	return t.conn.Close()
}

func (t *pipeTransport) Address() string {
	return t.path
}
