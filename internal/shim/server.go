package shim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
)

type Server struct {
	cfg    Config
	child  childBridge
	log    *slog.Logger
	queue  chan struct{}
	server *http.Server
}

type childBridge interface {
	Generation() uint64
	Bootstrap(context.Context) error
	Send(context.Context, *RPCRequest) ([]byte, error)
	Close() error
}

func NewServer(cfg Config, logger *slog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = observability.New()
	}
	return &Server{
		cfg:   cfg,
		child: newRawChild(cfg),
		log:   logger,
		queue: make(chan struct{}, 8),
	}, nil
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.BindAddr, s.cfg.BindPort)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Close(ctx context.Context) error {
	_ = s.child.Close()
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	status := http.StatusOK
	idType := "missing"
	respBytes := 0
	defer func() {
		s.log.Info("shim request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"id_type", idType,
			"child_generation", s.child.Generation(),
			"bytes_out", respBytes,
		)
	}()

	if r.URL.Path != "/mcp" {
		status = http.StatusNotFound
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		status = http.StatusMethodNotAllowed
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct == "" || !strings.Contains(ct, "application/json") {
		status = http.StatusUnsupportedMediaType
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	if !s.authorize(r.Header.Get("Authorization")) {
		status = http.StatusUnauthorized
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !s.acquire() {
		status = http.StatusTooManyRequests
		http.Error(w, "overloaded", http.StatusTooManyRequests)
		return
	}
	defer s.release()

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.cfg.MaxRequestBytes))
	if err != nil {
		status = http.StatusRequestEntityTooLarge
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}

	req, err := decodeRequest(body)
	if err != nil {
		idType = "malformed"
		status = http.StatusBadRequest
		http.Error(w, "malformed json-rpc", http.StatusBadRequest)
		return
	}
	idType = detectIDType(req.ID)

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.RequestTimeoutMS)*time.Millisecond)
	defer cancel()

	switch req.Method {
	case "initialize":
		status = http.StatusOK
		respBytes = writeJSON(w, fixedInitializeResponse(req.ID))
		return
	case "resources/list", "prompts/list":
		status = http.StatusOK
		respBytes = writeJSON(w, errorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method)))
		return
	case "notifications/initialized":
		if req.ID == nil || len(req.ID) == 0 {
			status = http.StatusAccepted
			w.WriteHeader(http.StatusAccepted)
			return
		}
		status = http.StatusOK
		respBytes = writeJSON(w, errorResponse(req.ID, -32600, fmt.Sprintf("invalid request: unexpected id for %q", req.Method)))
		return
	}

	if err := s.child.Bootstrap(ctx); err != nil {
		status = translateChildStatus(err)
		http.Error(w, err.Error(), status)
		return
	}

	if strings.TrimSpace(req.Method) == "tools/list" {
		req.Params = json.RawMessage(`{}`)
	}

	resp, err := s.child.Send(ctx, req)
	if err != nil {
		status = translateChildStatus(err)
		http.Error(w, err.Error(), status)
		return
	}

	if !json.Valid(resp) {
		status = http.StatusBadGateway
		http.Error(w, "invalid child response", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	respBytes, _ = w.Write(resp)
}

func (s *Server) authorize(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	const prefix = "Bearer "
	return strings.HasPrefix(value, prefix) && strings.TrimSpace(strings.TrimPrefix(value, prefix)) == s.cfg.BackendToken
}

func (s *Server) acquire() bool {
	select {
	case s.queue <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Server) release() {
	select {
	case <-s.queue:
	default:
	}
}

func detectIDType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "notification"
	}
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "null":
		return "null"
	case len(trimmed) > 0 && trimmed[0] == '"':
		return "string"
	case len(trimmed) > 0 && (trimmed[0] == '-' || (trimmed[0] >= '0' && trimmed[0] <= '9')):
		return "number"
	default:
		return "other"
	}
}

func translateChildStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := err.Error()
	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(msg, "context deadline exceeded"):
		return http.StatusGatewayTimeout
	case strings.Contains(msg, "backend unavailable"), strings.Contains(msg, "child stopped"):
		return http.StatusServiceUnavailable
	case strings.Contains(msg, "overloaded"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "write child"):
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}

func fixedInitializeResponse(id json.RawMessage) map[string]any {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"result": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "hugo-mcp",
				"version": "1.0.0",
			},
		},
	}
	if len(id) > 0 {
		resp["id"] = json.RawMessage(append([]byte(nil), id...))
	}
	return resp
}

func writeJSON(w http.ResponseWriter, v any) int {
	raw, err := encodeResponse(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return 0
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	n, _ := w.Write(raw)
	return n
}
