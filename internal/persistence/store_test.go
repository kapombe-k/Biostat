package persistence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemoryStoreScopesKindsAndDeletes(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	records := []Record{
		{OwnerID: "owner-a", Key: "one", Kind: "search", Payload: json.RawMessage(`{"query":"TP53"}`)},
		{OwnerID: "owner-a", Key: "two", Kind: "workspace", Payload: json.RawMessage(`{"structure":"1GZX"}`)},
		{OwnerID: "owner-b", Key: "one", Kind: "search", Payload: json.RawMessage(`{"query":"insulin"}`)},
	}
	for _, record := range records {
		if err := store.Put(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.List(ctx, "owner-a", "search")
	if err != nil || len(got) != 1 || got[0].Key != "one" {
		t.Fatalf("scoped list = %#v, %v", got, err)
	}
	if err := store.Delete(ctx, "owner-a", "one"); err != nil {
		t.Fatal(err)
	}
	got, _ = store.List(ctx, "owner-a", "")
	if len(got) != 1 || got[0].Key != "two" {
		t.Fatalf("delete left %#v", got)
	}
	other, _ := store.List(ctx, "owner-b", "")
	if len(other) != 1 {
		t.Fatalf("owner isolation failed: %#v", other)
	}
}

func TestSupabaseStoreRESTHeadersAndOperations(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Header.Get("apikey") != "secret-test-key" || r.Header.Get("Authorization") != "Bearer secret-test-key" {
			t.Errorf("missing Supabase service auth headers")
		}
		switch r.Method {
		case http.MethodGet:
			if !strings.Contains(r.URL.RawQuery, "owner_id=eq.") {
				t.Errorf("list query missing owner scope: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"record_key":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","kind":"search","payload":{"query":"TP53"},"updated_at":"2026-10-26T00:00:00Z"}]`))
		case http.MethodPost:
			if !strings.Contains(r.URL.RawQuery, "on_conflict=owner_id") {
				t.Errorf("upsert conflict columns missing: %s", r.URL.RawQuery)
			}
			var row map[string]any
			if err := json.NewDecoder(r.Body).Decode(&row); err != nil || row["owner_id"] == nil {
				t.Errorf("invalid upsert body: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if !strings.Contains(r.URL.RawQuery, "record_key=eq.") {
				t.Errorf("delete key filter missing: %s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	store, err := NewSupabaseStore(server.URL, "secret-test-key")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const owner = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const key = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	got, err := store.List(ctx, owner, "search")
	if err != nil || len(got) != 1 || got[0].Kind != "search" {
		t.Fatalf("list: %#v %v", got, err)
	}
	if err := store.Put(ctx, Record{OwnerID: owner, Key: key, Kind: "search", Payload: json.RawMessage(`{"query":"TP53"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner, key); err != nil {
		t.Fatal(err)
	}
	if strings.Join(methods, ",") != "GET,POST,DELETE" {
		t.Fatalf("HTTP operations: %v", methods)
	}
}

func TestSupabaseStoreRejectsUnsafeURLsAndRedirects(t *testing.T) {
	for _, raw := range []string{"http://example.com", "file:///etc/passwd", "https://user:pass@example.com", "https://example.com/?x=y"} {
		if _, err := NewSupabaseStore(raw, "key"); err == nil {
			t.Errorf("expected unsafe URL rejected: %s", raw)
		}
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1", http.StatusFound)
	}))
	defer redirect.Close()
	store, err := NewSupabaseStore(redirect.URL, "key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(context.Background(), "owner", "search"); err == nil {
		t.Fatal("expected non-2xx redirect to fail without following it")
	}
}
