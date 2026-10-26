package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxSupabaseResponse = 2 << 20

// SupabaseStore uses the Supabase PostgREST API. The service-role secret only
// exists in server configuration and is never returned to browser templates.
type SupabaseStore struct {
	baseURL string
	secret  string
	client  *http.Client
}

func NewSupabaseStore(rawURL, serviceRoleKey string) (*SupabaseStore, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(rawURL), "/"))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid Supabase URL")
	}
	if u.Scheme != "https" {
		host := strings.ToLower(u.Hostname())
		ip := net.ParseIP(host)
		if u.Scheme != "http" || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return nil, fmt.Errorf("Supabase URL must use HTTPS (HTTP is allowed only for localhost development)")
		}
	}
	if strings.TrimSpace(serviceRoleKey) == "" {
		return nil, fmt.Errorf("Supabase service-role key is required")
	}
	return &SupabaseStore{
		baseURL: strings.TrimRight(u.String(), "/"),
		secret:  serviceRoleKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (s *SupabaseStore) List(ctx context.Context, owner, kind string) ([]Record, error) {
	q := url.Values{
		"select":   {"record_key,kind,payload,updated_at"},
		"owner_id": {"eq." + owner},
		"order":    {"updated_at.desc"},
		"limit":    {"100"},
	}
	if kind != "" {
		q.Set("kind", "eq."+kind)
	}
	body, err := s.request(ctx, http.MethodGet, "/rest/v1/bioscope_records?"+q.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Key     string          `json:"record_key"`
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
		Updated time.Time       `json:"updated_at"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode Supabase library records: %w", err)
	}
	out := make([]Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, Record{OwnerID: owner, Key: row.Key, Kind: row.Kind, Payload: row.Payload, Updated: row.Updated})
	}
	return out, nil
}

func (s *SupabaseStore) Put(ctx context.Context, record Record) error {
	if record.Updated.IsZero() {
		record.Updated = time.Now().UTC()
	}
	row := struct {
		Owner   string          `json:"owner_id"`
		Key     string          `json:"record_key"`
		Kind    string          `json:"kind"`
		Payload json.RawMessage `json:"payload"`
		Updated time.Time       `json:"updated_at"`
	}{record.OwnerID, record.Key, record.Kind, record.Payload, record.Updated}
	payload, err := json.Marshal(row)
	if err != nil {
		return err
	}
	_, err = s.request(ctx, http.MethodPost, "/rest/v1/bioscope_records?on_conflict=owner_id,kind,record_key", payload, "resolution=merge-duplicates,return=minimal")
	return err
}

func (s *SupabaseStore) Delete(ctx context.Context, owner, key string) error {
	q := url.Values{"owner_id": {"eq." + owner}, "record_key": {"eq." + key}}
	_, err := s.request(ctx, http.MethodDelete, "/rest/v1/bioscope_records?"+q.Encode(), nil, "return=minimal")
	return err
}

func (s *SupabaseStore) request(ctx context.Context, method, path string, payload []byte, prefer string) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", s.secret)
	req.Header.Set("Authorization", "Bearer "+s.secret)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Supabase request failed: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxSupabaseResponse+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSupabaseResponse {
		return nil, fmt.Errorf("Supabase response exceeded size limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Supabase returned HTTP %d", resp.StatusCode)
	}
	return b, nil
}
