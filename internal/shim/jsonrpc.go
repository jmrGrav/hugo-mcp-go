package shim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func decodeRequest(raw []byte) (*RPCRequest, error) {
	var req RPCRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.JSONRPC) != "2.0" {
		return nil, fmt.Errorf("invalid jsonrpc version")
	}
	if strings.TrimSpace(req.Method) == "" {
		return nil, fmt.Errorf("missing method")
	}
	return &req, nil
}

func encodeResponse(v any) ([]byte, error) {
	return json.Marshal(v)
}

func errorResponse(id json.RawMessage, code int, message string) map[string]any {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if len(id) > 0 {
		resp["id"] = json.RawMessage(append([]byte(nil), id...))
	}
	return resp
}
