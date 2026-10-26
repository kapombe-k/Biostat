package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kapombe-k/Biostat/internal/persistence"
)

var libraryUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
var libraryKey = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type libraryPutRequest struct {
	OwnerID string          `json:"owner_id"`
	Key     string          `json:"record_key"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner_id")
	kind := r.URL.Query().Get("kind")
	if !libraryUUID.MatchString(owner) || (kind != "" && kind != "search" && kind != "workspace") {
		http.Error(w, "valid owner_id UUID and optional supported kind are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	records, err := s.library.List(ctx, owner, kind)
	if err != nil {
		s.logger.Warn("library list failed", "err", err)
		http.Error(w, "Library is temporarily unavailable; browser-local records are still available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		Records []persistence.Record `json:"records"`
		Durable bool                 `json:"durable"`
	}{records, s.libraryPersistent})
}

func (s *Server) handleLibraryPut(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 72*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in libraryPutRequest
	if err := dec.Decode(&in); err != nil {
		http.Error(w, "invalid library record", http.StatusBadRequest)
		return
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		http.Error(w, "request must contain exactly one JSON object", http.StatusBadRequest)
		return
	}
	if !libraryUUID.MatchString(in.OwnerID) || !libraryKey.MatchString(in.Key) || (in.Kind != "search" && in.Kind != "workspace") || len(in.Payload) == 0 || len(in.Payload) > 64*1024 {
		http.Error(w, "invalid owner, record key, kind, or payload size", http.StatusBadRequest)
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(in.Payload, &payload); err != nil || payload == nil {
		http.Error(w, "payload must be a JSON object", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	err := s.library.Put(ctx, persistence.Record{
		OwnerID: strings.ToLower(in.OwnerID), Key: strings.ToLower(in.Key), Kind: in.Kind,
		Payload: in.Payload, Updated: time.Now().UTC(),
	})
	if err != nil {
		s.logger.Warn("library save failed", "err", err)
		http.Error(w, "Library is temporarily unavailable; record remains in this browser", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLibraryDelete(w http.ResponseWriter, r *http.Request) {
	owner, key := r.URL.Query().Get("owner_id"), r.PathValue("key")
	if !libraryUUID.MatchString(owner) || !libraryKey.MatchString(key) {
		http.Error(w, "valid owner_id and record key UUIDs are required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	if err := s.library.Delete(ctx, strings.ToLower(owner), strings.ToLower(key)); err != nil {
		s.logger.Warn("library delete failed", "err", err)
		http.Error(w, "Library is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
