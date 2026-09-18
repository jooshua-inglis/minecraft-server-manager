// Package webapi is mcm's HTTP API: `mcm web` starts it, and it mirrors
// the CLI's commands as JSON over the same fleet package the CLI calls
// (RESEARCH.md §7.7 / plan M12), plus SSE streams for live logs and
// stats. Read endpoints (GET) are open; every state-changing endpoint
// (M14) requires a bearer token, since `mcm web` may be reachable by
// more than just the operator's own shell (RESEARCH.md §7.5).
package webapi

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const (
	defaultLogTail      = "100"
	statsStreamInterval = 2 * time.Second
)

type Handler struct {
	fleet *fleet.Fleet
	token string
}

// NewHandler builds mcm's HTTP API against f. token authenticates
// every write endpoint (see requireAuth); read endpoints need it too
// only when the operator wants that, see the "GET" routes below for
// which is which.
func NewHandler(f *fleet.Fleet, token string) http.Handler {
	h := &Handler{fleet: f, token: token}

	mux := http.NewServeMux()

	// Read-only: no auth required, matching M12's original behavior.
	mux.HandleFunc("GET /api/servers", h.listServers)
	mux.HandleFunc("GET /api/servers/{name}", h.serverDetail)
	mux.HandleFunc("GET /api/servers/{name}/logs", h.serverLogs)
	mux.HandleFunc("GET /api/servers/{name}/stats/stream", h.streamStats)
	mux.HandleFunc("GET /api/servers/{name}/whitelist", h.listWhitelist)
	mux.HandleFunc("GET /api/servers/{name}/ops", h.listOps)
	mux.HandleFunc("GET /api/servers/{name}/mods", h.listMods)
	mux.HandleFunc("GET /api/servers/{name}/modpack", h.getModpack)
	mux.HandleFunc("GET /api/servers/{name}/backups", h.listBackups)

	// Write: bearer token required (M14).
	mux.HandleFunc("POST /api/auth/verify", h.requireAuth(h.verifyAuth))
	mux.HandleFunc("POST /api/servers", h.requireAuth(h.createServer))
	mux.HandleFunc("DELETE /api/servers/{name}", h.requireAuth(h.destroyServer))
	mux.HandleFunc("POST /api/servers/{name}/start", h.requireAuth(h.startServer))
	mux.HandleFunc("POST /api/servers/{name}/stop", h.requireAuth(h.stopServer))
	mux.HandleFunc("POST /api/servers/{name}/exec", h.requireAuth(h.execServer))
	mux.HandleFunc("POST /api/servers/{name}/whitelist", h.requireAuth(h.addWhitelist))
	mux.HandleFunc("DELETE /api/servers/{name}/whitelist/{player}", h.requireAuth(h.removeWhitelist))
	mux.HandleFunc("POST /api/servers/{name}/ops", h.requireAuth(h.addOp))
	mux.HandleFunc("DELETE /api/servers/{name}/ops/{player}", h.requireAuth(h.removeOp))
	mux.HandleFunc("POST /api/servers/{name}/mods", h.requireAuth(h.addMod))
	mux.HandleFunc("POST /api/servers/{name}/modpack", h.requireAuth(h.installModpack))
	mux.HandleFunc("POST /api/servers/{name}/backup", h.requireAuth(h.createBackup))
	mux.HandleFunc("POST /api/servers/{name}/restore", h.requireAuth(h.restoreBackup))

	return mux
}

// requireAuth wraps a write handler so it 401s without a bearer token
// matching h.token. Constant-time compare so response timing doesn't
// leak how much of a guessed token was correct.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(h.token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
			return
		}
		next(w, r)
	}
}

// listServers mirrors `mcm list`.
func (h *Handler) listServers(w http.ResponseWriter, r *http.Request) {
	views, err := h.fleet.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, views)
}

// ServerDetail is the JSON-friendly shape of fleet.Status: the same
// fields `mcm status` prints, flattened out of the raw Docker inspect
// response.
type ServerDetail struct {
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Version   string           `json:"version"`
	Port      int              `json:"port"`
	RCONPort  int              `json:"rcon_port"`
	DataDir   string           `json:"data_dir"`
	CreatedAt time.Time        `json:"created_at"`
	Container string           `json:"container,omitempty"`
	Status    string           `json:"status"`
	StartedAt string           `json:"started_at,omitempty"`
	LastError string           `json:"last_error,omitempty"`
	Crash     *fleet.CrashInfo `json:"crash,omitempty"`
}

// serverDetail mirrors `mcm status <name>`.
func (h *Handler) serverDetail(w http.ResponseWriter, r *http.Request) {
	st, err := h.fleet.Status(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDetail(st))
}

func toDetail(st *fleet.Status) ServerDetail {
	d := ServerDetail{
		Name:      st.Metadata.Name,
		Type:      st.Metadata.Type,
		Version:   st.Metadata.Version,
		Port:      st.Metadata.Port,
		RCONPort:  st.Metadata.RCONPort,
		DataDir:   st.DataDir,
		CreatedAt: st.Metadata.CreatedAt,
		Status:    "not created",
		Crash:     st.Crash,
	}
	if st.Info != nil {
		d.Container = st.Metadata.ContainerName
		if st.Info.State != nil {
			d.Status = string(st.Info.State.Status)
			if st.Info.State.Running {
				d.StartedAt = st.Info.State.StartedAt
			}
			d.LastError = st.Info.State.Error
		}
	}
	return d
}

// serverLogs mirrors `mcm logs <name>`: a plain tail as JSON by default,
// or an SSE stream of new lines when `?follow=true` (mirroring `mcm logs
// -f`).
func (h *Handler) serverLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = defaultLogTail
	}

	if r.URL.Query().Get("follow") != "true" {
		var buf bytes.Buffer
		if err := h.fleet.Logs(r.Context(), name, false, tail, &buf); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string][]string{"lines": splitLines(buf.String())})
		return
	}

	// Validate before committing to a streaming response, so a bad name
	// still gets a proper 404 instead of an SSE stream that opens then
	// immediately dies.
	if _, err := h.fleet.Status(r.Context(), name); err != nil {
		writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)
	sw := &sseLineWriter{w: w, flusher: flusher}
	_ = h.fleet.Logs(r.Context(), name, true, tail, sw)
}

// streamStats is an SSE stream of the same per-server snapshot `mcm top`
// takes, refreshed every statsStreamInterval until the client
// disconnects (mirrors `mcm top -w`, scoped to one server).
func (h *Handler) streamStats(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	stats, err := h.fleet.SingleStats(ctx, name)
	if err != nil {
		writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	setSSEHeaders(w)
	writeSSEData(w, flusher, stats)

	ticker := time.NewTicker(statsStreamInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := h.fleet.SingleStats(ctx, name)
			if err != nil {
				return
			}
			writeSSEData(w, flusher, stats)
		}
	}
}

// sseLineWriter adapts an io.Writer expecting arbitrary byte chunks
// (fleet.Logs) into one SSE "data:" event per complete line, flushing
// after each.
type sseLineWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	buf     []byte
}

func (s *sseLineWriter) Write(p []byte) (int, error) {
	s.buf = append(s.buf, p...)
	for {
		i := bytes.IndexByte(s.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(s.buf[:i]), "\r")
		s.buf = s.buf[i+1:]
		if _, err := fmt.Fprintf(s.w, "data: %s\n\n", line); err != nil {
			return 0, err
		}
		s.flusher.Flush()
	}
	return len(p), nil
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

func writeSSEData(w http.ResponseWriter, flusher http.Flusher, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, serverstore.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
