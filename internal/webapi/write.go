package webapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// writeActionError is writeError's counterpart for mutating endpoints:
// almost everything a write action can fail with (bad name, already
// exists, EULA not accepted, wrong state, unknown source, ...) is a
// client-caused 400, not a server fault, except a missing server.
func writeActionError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, serverstore.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

// verifyAuth exists purely so the UI can check a pasted token is
// correct immediately, rather than finding out on the first real
// write action.
func (h *Handler) verifyAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type createServerRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Version    string `json:"version"`
	Memory     string `json:"memory"`
	Port       int    `json:"port"`
	AcceptEULA bool   `json:"accept_eula"`
}

// createServer mirrors `mcm create`.
func (h *Handler) createServer(w http.ResponseWriter, r *http.Request) {
	var req createServerRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(w, "invalid request body")
		return
	}
	meta, err := h.fleet.Create(r.Context(), req.Name, fleet.CreateOptions{
		Type:       req.Type,
		Version:    req.Version,
		Memory:     req.Memory,
		Port:       req.Port,
		AcceptEULA: req.AcceptEULA,
	})
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, meta)
}

// destroyServer mirrors `mcm destroy`. ?purge=true also deletes the
// server's data directory, matching the CLI's --purge flag.
func (h *Handler) destroyServer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	purge := r.URL.Query().Get("purge") == "true"
	if err := h.fleet.Destroy(r.Context(), name, fleet.DestroyOptions{Purge: purge}); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
}

// startServer mirrors `mcm start`.
func (h *Handler) startServer(w http.ResponseWriter, r *http.Request) {
	if err := h.fleet.Start(r.Context(), r.PathValue("name")); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// stopServer mirrors `mcm stop`.
func (h *Handler) stopServer(w http.ResponseWriter, r *http.Request) {
	if err := h.fleet.Stop(r.Context(), r.PathValue("name")); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

type execRequest struct {
	Command string `json:"command"`
}

// execServer mirrors `mcm exec`.
func (h *Handler) execServer(w http.ResponseWriter, r *http.Request) {
	var req execRequest
	if err := readJSON(r, &req); err != nil || req.Command == "" {
		badRequest(w, "missing command")
		return
	}
	output, err := h.fleet.Exec(r.Context(), r.PathValue("name"), req.Command)
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": output})
}

// listWhitelist mirrors `mcm whitelist list`.
func (h *Handler) listWhitelist(w http.ResponseWriter, r *http.Request) {
	entries, err := h.fleet.WhitelistList(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

type playerRequest struct {
	Player  string `json:"player"`
	Offline bool   `json:"offline"`
}

// addWhitelist mirrors `mcm whitelist add`.
func (h *Handler) addWhitelist(w http.ResponseWriter, r *http.Request) {
	var req playerRequest
	if err := readJSON(r, &req); err != nil || req.Player == "" {
		badRequest(w, "missing player")
		return
	}
	if err := h.fleet.WhitelistAdd(r.Context(), r.PathValue("name"), req.Player, req.Offline); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// removeWhitelist mirrors `mcm whitelist remove`.
func (h *Handler) removeWhitelist(w http.ResponseWriter, r *http.Request) {
	if err := h.fleet.WhitelistRemove(r.Context(), r.PathValue("name"), r.PathValue("player")); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// listOps mirrors `mcm op list`.
func (h *Handler) listOps(w http.ResponseWriter, r *http.Request) {
	entries, err := h.fleet.OpList(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// addOp mirrors `mcm op add`.
func (h *Handler) addOp(w http.ResponseWriter, r *http.Request) {
	var req playerRequest
	if err := readJSON(r, &req); err != nil || req.Player == "" {
		badRequest(w, "missing player")
		return
	}
	if err := h.fleet.OpAdd(r.Context(), r.PathValue("name"), req.Player, req.Offline); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// removeOp mirrors `mcm op remove`.
func (h *Handler) removeOp(w http.ResponseWriter, r *http.Request) {
	if err := h.fleet.OpRemove(r.Context(), r.PathValue("name"), r.PathValue("player")); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// listMods mirrors `mcm mods list`.
func (h *Handler) listMods(w http.ResponseWriter, r *http.Request) {
	mods, err := h.fleet.ModsList(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mods)
}

type sourceRefRequest struct {
	Source string `json:"source"`
	Ref    string `json:"ref"`
}

// addMod mirrors `mcm mods add`.
func (h *Handler) addMod(w http.ResponseWriter, r *http.Request) {
	var req sourceRefRequest
	if err := readJSON(r, &req); err != nil || req.Source == "" || req.Ref == "" {
		badRequest(w, "missing source/ref")
		return
	}
	if err := h.fleet.ModsAdd(r.Context(), r.PathValue("name"), req.Source, req.Ref); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// getModpack mirrors `mcm modpack status`.
func (h *Handler) getModpack(w http.ResponseWriter, r *http.Request) {
	status, err := h.fleet.ModpackStatus(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// installModpack mirrors `mcm modpack install`.
func (h *Handler) installModpack(w http.ResponseWriter, r *http.Request) {
	var req sourceRefRequest
	if err := readJSON(r, &req); err != nil || req.Source == "" || req.Ref == "" {
		badRequest(w, "missing source/ref")
		return
	}
	if err := h.fleet.ModpackInstall(r.Context(), r.PathValue("name"), req.Source, req.Ref); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "installed"})
}

// listBackups mirrors `mcm backup list`.
func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := h.fleet.ListBackups(r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, backups)
}

// createBackup mirrors `mcm backup`.
func (h *Handler) createBackup(w http.ResponseWriter, r *http.Request) {
	info, err := h.fleet.Backup(r.Context(), r.PathValue("name"))
	if err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

type restoreRequest struct {
	BackupID string `json:"backup_id"`
}

// restoreBackup mirrors `mcm restore`.
func (h *Handler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	var req restoreRequest
	if err := readJSON(r, &req); err != nil || req.BackupID == "" {
		badRequest(w, "missing backup_id")
		return
	}
	if err := h.fleet.Restore(r.Context(), r.PathValue("name"), req.BackupID); err != nil {
		writeActionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}
