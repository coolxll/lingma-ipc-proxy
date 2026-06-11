package lingmaipc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLingmaWebSocketProbe(t *testing.T) {
	if os.Getenv("LINGMA_WS_PROBE") != "1" {
		t.Skip("set LINGMA_WS_PROBE=1 to run the real Lingma websocket probe")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	wsURL, err := ResolveWebSocketURL("")
	if err != nil {
		t.Fatalf("resolve websocket url: %v", err)
	}

	client, err := Connect(ctx, DialOptions{
		Transport:    TransportWebSocket,
		WebSocketURL: wsURL,
	})
	if err != nil {
		t.Fatalf("connect websocket: %v", err)
	}
	defer client.Close()

	if err := client.Request(ctx, "initialize", map[string]any{
		"protocolVersion":    1,
		"clientCapabilities": map[string]any{},
		"timestamp":          time.Now().UnixMilli(),
	}, nil); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	t.Logf("connected to %s", client.Address())

	var rawModels any
	if err := client.Request(ctx, "config/queryModels", map[string]any{}, &rawModels); err != nil {
		t.Fatalf("query models: %v", err)
	}
	modelBody, _ := json.Marshal(rawModels)
	t.Logf("queryModels response: %s", string(modelBody))

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	var created struct {
		SessionID string `json:"sessionId"`
		ID        string `json:"id"`
	}
	if err := client.Request(ctx, "session/new", map[string]any{
		"cwd":        cwd,
		"mcpServers": []any{},
		"_meta":      map[string]any{},
		"timestamp":  time.Now().UnixMilli(),
	}, &created); err != nil {
		t.Fatalf("session/new: %v", err)
	}

	sessionID := strings.TrimSpace(created.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(created.ID)
	}
	if sessionID == "" {
		t.Fatal("session/new returned an empty session id")
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := client.Request(cleanupCtx, "chat/deleteSessionById", map[string]any{"sessionId": sessionID}, nil); err != nil {
			_ = client.Request(cleanupCtx, "chat/deleteSessionById", map[string]any{"id": sessionID}, nil)
		}
	}()

	modelID := strings.TrimSpace(os.Getenv("LINGMA_WS_PROBE_MODEL"))
	if modelID == "" {
		modelID = "dashscope_qmodel"
	}

	requestID := CreateRequestID("ws-probe")
	meta := CreateMeta(MetaOptions{
		RequestID:  requestID,
		Mode:       "chat",
		Model:      modelID,
		ShellType:  DefaultShellType(),
		EnabledMCP: []any{},
	})

	if err := client.Request(ctx, "session/set_model", map[string]any{
		"sessionId": sessionID,
		"modelId":   modelID,
		"timestamp": time.Now().UnixMilli(),
		"_meta":     meta,
	}, nil); err != nil {
		t.Fatalf("session/set_model(%s): %v", modelID, err)
	}

	prompt := strings.TrimSpace(os.Getenv("LINGMA_WS_PROBE_PROMPT"))
	if prompt == "" {
		prompt = "Reply with exactly: WS_PROBE_OK"
	}

	notifications, cancelSub := client.Subscribe()
	defer cancelSub()

	if err := client.Send("session/prompt", map[string]any{
		"sessionId": sessionID,
		"prompt": []map[string]any{
			{"type": "text", "text": prompt},
		},
		"_meta": meta,
	}); err != nil {
		t.Fatalf("session/prompt: %v", err)
	}

	var chunks strings.Builder
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for websocket response; partial=%q", chunks.String())
		case notification, ok := <-notifications:
			if !ok {
				t.Fatalf("notification stream closed; partial=%q", chunks.String())
			}
			if notification.Method != "session/update" {
				continue
			}
			if nestedStringFromMap(notification.Params, "_meta", MetaRequestID) != requestID {
				continue
			}

			update := nestedMap(notification.Params, "update")
			switch nestedString(update, "sessionUpdate") {
			case "agent_message_chunk":
				chunk := nestedString(nestedMap(update, "content"), "text")
				if chunk != "" {
					chunks.WriteString(chunk)
					t.Logf("chunk: %q", chunk)
				}
			case "notification":
				updateType := nestedString(update, "type")
				data := nestedMap(update, "data")
				dataBody, _ := json.Marshal(data)
				t.Logf("notification type=%s data=%s", updateType, string(dataBody))
				if updateType == "chat_finish" {
					reply := strings.TrimSpace(chunks.String())
					if reply == "" {
						t.Fatal("chat finished without any assistant text")
					}
					t.Logf("final reply: %s", reply)
					return
				}
			}
		}
	}
}

func nestedMap(m map[string]any, key string) map[string]any {
	value, ok := m[key]
	if !ok {
		return map[string]any{}
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}

func nestedString(m map[string]any, key string) string {
	value, ok := m[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}

func nestedStringFromMap(m map[string]any, parent string, key string) string {
	return nestedString(nestedMap(m, parent), key)
}
