// Package web exposes the local safeops console and JSON API.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/14122mhl/safeops-go/internal/agent/rag"
	agentService "github.com/14122mhl/safeops-go/internal/agent/service"
	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/trace"
)

//go:embed assets/index.html
var assets embed.FS

// Server adapts the Agent Kernel to a local HTTP API.
type Server struct {
	Config     config.Config
	Agent      agentService.Service
	TraceStore trace.Store
	Documents  rag.LocalSearcher
}

// GoalPayload is the intentionally small public request contract.
type GoalPayload struct {
	Goal           string   `json:"goal"`
	Playbook       string   `json:"playbook"`
	Inventory      string   `json:"inventory"`
	Environment    string   `json:"env"`
	Limit          string   `json:"limit"`
	ExtraVars      []string `json:"extra_vars"`
	Apply          bool     `json:"apply"`
	Approve        bool     `json:"approve"`
	Confirm        string   `json:"confirm"`
	PlanOnly       *bool    `json:"plan_only"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type captureSink struct {
	Lines      []string `json:"lines"`
	StdoutText string   `json:"stdout"`
	StderrText string   `json:"stderr"`
}

func (sink *captureSink) Line(value string)   { sink.Lines = append(sink.Lines, value) }
func (sink *captureSink) Stdout(value string) { sink.StdoutText += value }
func (sink *captureSink) Stderr(value string) { sink.StderrText += value }

// Handler returns the complete local web application.
func (server Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", server.index)
	mux.HandleFunc("GET /api/status", server.status)
	mux.HandleFunc("POST /api/goal", server.goal)
	mux.HandleFunc("GET /api/rag", server.ragSummary)
	mux.HandleFunc("GET /api/runs", server.runs)
	return noStore(recoverPanics(mux))
}

// Serve runs until the context is canceled, then performs graceful shutdown.
func (server Server) Serve(ctx context.Context, address string) error {
	httpServer := &http.Server{Addr: address, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	errorChannel := make(chan error, 1)
	go func() { errorChannel <- httpServer.ListenAndServe() }()
	select {
	case err := <-errorChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownContext)
	}
}

func (server Server) index(writer http.ResponseWriter, _ *http.Request) {
	data, err := assets.ReadFile("assets/index.html")
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(data)
}

func (server Server) status(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"app": "safeops-go", "api_provider": server.Config.API.Provider,
		"api_enabled": server.Config.API.Provider != "" && server.Config.API.DeepSeek.Enabled,
		"rag_enabled": server.Config.RAG.Enabled, "config": server.Config.Masked(),
	})
}

func (server Server) goal(writer http.ResponseWriter, request *http.Request) {
	var payload GoalPayload
	request.Body = http.MaxBytesReader(writer, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(writer, http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(writer, http.StatusBadRequest, errors.New("request body must contain one JSON object"))
		return
	}
	if payload.Goal == "" {
		writeError(writer, http.StatusBadRequest, errors.New("goal is required"))
		return
	}
	planOnly := true
	if payload.PlanOnly != nil {
		planOnly = *payload.PlanOnly
	}
	timeout := 10 * time.Minute
	if payload.TimeoutSeconds > 0 {
		timeout = time.Duration(payload.TimeoutSeconds) * time.Second
	}
	sink := &captureSink{Lines: []string{}}
	response := server.Agent.Run(request.Context(), agentService.Request{
		Goal: payload.Goal, Playbook: payload.Playbook, Inventory: payload.Inventory,
		Environment: payload.Environment, Limit: payload.Limit, ExtraVars: payload.ExtraVars,
		ExplicitApply: payload.Apply, Approved: payload.Approve, ProductionConfirm: payload.Confirm,
		PlanOnly: planOnly, Timeout: timeout,
	}, sink)
	writeJSON(writer, http.StatusOK, map[string]any{"response": response, "output": sink})
}

func (server Server) ragSummary(writer http.ResponseWriter, request *http.Request) {
	documents, err := server.Documents.List(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"enabled": server.Config.RAG.Enabled, "paths": server.Config.RAG.Paths, "document_count": len(documents), "documents": documents})
}

func (server Server) runs(writer http.ResponseWriter, _ *http.Request) {
	runs, err := server.TraceStore.Recent(12)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"runs": runs})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(writer, request)
	})
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				writeError(writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
