package shim

import "testing"

func TestDecodeRequestAndJSONHelpers(t *testing.T) {
	if _, err := decodeRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)); err != nil {
		t.Fatalf("decodeRequest() error = %v", err)
	}
	for _, raw := range [][]byte{
		[]byte(`{"jsonrpc":"1.0","id":1,"method":"ping"}`),
		[]byte(`{"jsonrpc":"2.0","id":1}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"ping","extra":1}`),
	} {
		if _, err := decodeRequest(raw); err == nil {
			t.Fatalf("decodeRequest(%s) expected error", string(raw))
		}
	}

	if raw, err := encodeResponse(map[string]any{"ok": true}); err != nil || string(raw) == "" {
		t.Fatalf("encodeResponse() raw=%s err=%v", string(raw), err)
	}
	if got := errorResponse(nil, -32601, "x"); got["jsonrpc"] != "2.0" {
		t.Fatalf("errorResponse() = %#v", got)
	}
}
