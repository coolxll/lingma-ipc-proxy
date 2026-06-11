package lingmaipc

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveSharedClientInfoFromJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".info.json")
	content := `{"websocketPort":36510,"pid":14060,"ipcServerPath":"\\\\.\\pipe\\lingma-bf0f32","isDev":false}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write shared info json: %v", err)
	}

	info, err := resolveSharedClientInfoFromPaths([]string{path})
	if err != nil {
		t.Fatalf("resolve shared info json: %v", err)
	}
	if info.WebSocketPort != 36510 {
		t.Fatalf("unexpected websocket port: %d", info.WebSocketPort)
	}
	if info.PID != 14060 {
		t.Fatalf("unexpected pid: %d", info.PID)
	}
	if info.IPCServerPath != `\\.\pipe\lingma-bf0f32` {
		t.Fatalf("unexpected pipe path: %q", info.IPCServerPath)
	}
}

func TestResolveSharedClientInfoFromLegacyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".info")
	content := "36510\n14060\n\\\\.\\pipe\\lingma-bf0f32\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write shared info legacy: %v", err)
	}

	info, err := resolveSharedClientInfoFromPaths([]string{path})
	if err != nil {
		t.Fatalf("resolve shared info legacy: %v", err)
	}
	if info.WebSocketPort != 36510 {
		t.Fatalf("unexpected websocket port: %d", info.WebSocketPort)
	}
	if info.PID != 14060 {
		t.Fatalf("unexpected pid: %d", info.PID)
	}
	if info.IPCServerPath != `\\.\pipe\lingma-bf0f32` {
		t.Fatalf("unexpected pipe path: %q", info.IPCServerPath)
	}
}

func TestNormalizeWebSocketURLAddsRootPath(t *testing.T) {
	got, err := normalizeWebSocketURL("ws://127.0.0.1:36510")
	if err != nil {
		t.Fatalf("normalize websocket url: %v", err)
	}
	if got != "ws://127.0.0.1:36510/" {
		t.Fatalf("unexpected normalized websocket url: %q", got)
	}
}

func TestSharedClientInfoSearchBasesDarwinIncludesLingmaVSCodePath(t *testing.T) {
	homeDir := "/Users/tester"
	got := sharedClientInfoSearchBases("darwin", homeDir, "", "/Users/tester/Library/Application Support")
	want := filepath.Join(homeDir, ".lingma", "vscode", "sharedClientCache")
	if len(got) == 0 || got[0] != want {
		t.Fatalf("expected first darwin search base %q, got %v", want, got)
	}
}

func TestResolvePipePathNonWindowsReturnsExplicitError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows behavior only")
	}
	_, err := ResolvePipePath("")
	if err == nil {
		t.Fatal("expected pipe resolution to fail on non-windows")
	}
	if !strings.Contains(err.Error(), "requires Windows") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDialOptionsAutoUsesWebSocketOnNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows behavior only")
	}
	t.Setenv("LINGMA_PROXY_WS_URL", "ws://127.0.0.1:36510")
	opts, err := ResolveDialOptions(TransportAuto, "", "")
	if err != nil {
		t.Fatalf("resolve auto transport: %v", err)
	}
	if opts.Transport != TransportWebSocket {
		t.Fatalf("unexpected transport: %s", opts.Transport)
	}
	if opts.WebSocketURL != "ws://127.0.0.1:36510/" {
		t.Fatalf("unexpected websocket url: %q", opts.WebSocketURL)
	}
}
