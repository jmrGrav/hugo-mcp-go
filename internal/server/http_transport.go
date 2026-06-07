package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jmrGrav/hugo-mcp-go/internal/observability"
	"github.com/jmrGrav/hugo-mcp-go/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func newHTTPHandler(svc *Service, cfg transport.Config, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = observability.New()
	}
	if svc == nil || svc.server == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "server not initialized", http.StatusInternalServerError)
		})
	}

	inner := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return svc.server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                  true,
		JSONResponse:               true,
		DisableLocalhostProtection: true,
	})
	var streamingInner http.Handler
	if cfg.StreamingEnabled {
		streamingInner = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return svc.server
		}, &mcp.StreamableHTTPOptions{
			Stateless:                  true,
			DisableLocalhostProtection: true,
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		status := http.StatusOK
		defer func() {
			logger.Info("mcp http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"latency_ms", time.Since(start).Milliseconds(),
			)
		}()

		switch r.URL.Path {
		case "/mcp":
			if r.Method != http.MethodPost {
				status = http.StatusMethodNotAllowed
				w.Header().Set("Allow", http.MethodPost)
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if ct := r.Header.Get("Content-Type"); ct == "" || !strings.Contains(strings.ToLower(ct), "application/json") {
				status = http.StatusUnsupportedMediaType
				http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
			if !authorizeBearer(r.Header.Get("Authorization"), cfg.HTTPToken) {
				status = http.StatusUnauthorized
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, cfg.MaxRequestBytes+1))
			_ = r.Body.Close()
			if err != nil {
				status = http.StatusBadRequest
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			if int64(len(body)) > cfg.MaxRequestBytes {
				status = http.StatusRequestEntityTooLarge
				http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
				return
			}

			if method, id, ok := decodeTransportMethod(body); ok && method == "initialize" {
				status = http.StatusOK
				writeJSONResponse(w, fixedInitializeResponse(id))
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			r.Header.Set("Accept", "application/json, text/event-stream")

			rec := &statusRecorder{ResponseWriter: w}
			inner.ServeHTTP(rec, r)
			if rec.status != 0 {
				status = rec.status
			}
		case "/mcp/events":
			if !cfg.StreamingEnabled || streamingInner == nil {
				status = http.StatusNotFound
				http.NotFound(w, r)
				return
			}
			if !authorizeBearer(r.Header.Get("Authorization"), cfg.HTTPToken) {
				status = http.StatusUnauthorized
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if r.Method == http.MethodPost {
				body, err := io.ReadAll(io.LimitReader(r.Body, cfg.MaxRequestBytes+1))
				_ = r.Body.Close()
				if err != nil {
					status = http.StatusBadRequest
					http.Error(w, "failed to read request body", http.StatusBadRequest)
					return
				}
				if int64(len(body)) > cfg.MaxRequestBytes {
					status = http.StatusRequestEntityTooLarge
					http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))
			}
			rec := &statusRecorder{ResponseWriter: w}
			streamingInner.ServeHTTP(rec, r)
			if rec.status != 0 {
				status = rec.status
			}
		default:
			status = http.StatusNotFound
			http.NotFound(w, r)
		}
	})
}

func decodeTransportMethod(body []byte) (string, json.RawMessage, bool) {
	var payload struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, false
	}
	return strings.TrimSpace(payload.Method), payload.ID, true
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

func writeJSONResponse(w http.ResponseWriter, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func authorizeBearer(value, want string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix)) == want
}

func (s *Service) RunHTTP(ctx context.Context, cfg transport.Config, logger *slog.Logger) error {
	if s == nil || s.server == nil {
		return fmt.Errorf("server not initialized")
	}
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTPBindAddr, cfg.HTTPBindPort),
		Handler:           newHTTPHandler(s, cfg, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	done := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done <- srv.Shutdown(shutdownCtx)
	}()
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return <-done
	}
	return err
}
