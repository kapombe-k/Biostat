package clients

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kapombe-k/Biostat/internal/config"
)

// ErrNotFound signifies an HTTP 404 from an upstream scientific database.
var ErrNotFound = errors.New("upstream record not found")

// ErrCircuitOpen indicates an upstream host is temporarily short-circuited after consecutive failures.
var ErrCircuitOpen = errors.New("upstream host circuit breaker temporarily open")

type hostCircuit struct {
	failures  int
	openUntil time.Time
}

// SafeHTTPClient enforces SSRF host allowlists (including across redirects), request timeouts,
// bounded response sizes, polite User-Agent attribution, exponential backoff retries, and per-host circuit breaking.
type SafeHTTPClient struct {
	cfg             *config.Config
	client          *http.Client
	logger          *slog.Logger
	mu              sync.Mutex
	circuits        map[string]*hostCircuit
	circuitCooldown time.Duration
	nowFn           func() time.Time
}

// NewSafeHTTPClient creates a resilient outbound HTTP wrapper governed by cfg.
func NewSafeHTTPClient(cfg *config.Config, logger *slog.Logger) *SafeHTTPClient {
	if logger == nil {
		logger = slog.Default()
	}
	hc := &http.Client{
		Timeout: cfg.HTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many upstream redirects")
			}
			return cfg.IsAllowedURL(req.URL.String())
		},
	}
	return &SafeHTTPClient{
		cfg:             cfg,
		client:          hc,
		logger:          logger,
		circuits:        make(map[string]*hostCircuit),
		circuitCooldown: 45 * time.Second,
		nowFn:           time.Now,
	}
}

// Do executes a GET or POST request with SSRF validation, circuit breaking, bounded body reading, and retry/backoff.
func (s *SafeHTTPClient) Do(ctx context.Context, method, targetURL, contentType string, body []byte) ([]byte, int, error) {
	if err := s.cfg.IsAllowedURL(targetURL); err != nil {
		return nil, 0, err
	}

	u, _ := url.Parse(targetURL)
	hostKey := strings.ToLower(u.Host)
	if s.isCircuitOpen(hostKey) {
		return nil, 0, fmt.Errorf("%s: %w", hostKey, ErrCircuitOpen)
	}

	var lastErr error
	maxAttempts := s.cfg.MaxRetries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := s.cfg.RetryBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			}
		}

		reqCtx, cancel := context.WithTimeout(ctx, s.cfg.HTTPTimeout)
		var reader io.Reader
		if len(body) > 0 {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, targetURL, reader)
		if err != nil {
			cancel()
			return nil, 0, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("User-Agent", fmt.Sprintf("BioScope/1.0 (Interactive Molecular Learning Lab; contact=%s)", s.cfg.ContactEmail))
		req.Header.Set("Accept", "application/json, text/plain, */*")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			s.logger.Debug("upstream HTTP transport error", "url", targetURL, "attempt", attempt+1, "err", err)
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, 0, ctx.Err()
			}
			continue
		}

		limited := io.LimitReader(resp.Body, s.cfg.MaxResponseBytes+1)
		data, readErr := io.ReadAll(limited)
		resp.Body.Close()
		cancel()

		if readErr != nil {
			lastErr = readErr
			continue
		}
		if int64(len(data)) > s.cfg.MaxResponseBytes {
			return nil, resp.StatusCode, fmt.Errorf("upstream response exceeded safety cap (%d bytes)", s.cfg.MaxResponseBytes)
		}

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			s.recordSuccess(hostKey)
			return nil, resp.StatusCode, ErrNotFound
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("upstream status %d", resp.StatusCode)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if sec, err := strconv.Atoi(ra); err == nil && sec > 0 && sec <= 5 {
					select {
					case <-time.After(time.Duration(sec) * time.Second):
					case <-ctx.Done():
						return nil, resp.StatusCode, ctx.Err()
					}
				}
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			s.recordFailure(hostKey)
			return nil, resp.StatusCode, fmt.Errorf("upstream returned status %d", resp.StatusCode)
		}
		s.recordSuccess(hostKey)
		return data, resp.StatusCode, nil
	}

	s.recordFailure(hostKey)
	return nil, 0, fmt.Errorf("request to %s failed after %d attempts: %w", targetURL, maxAttempts, lastErr)
}

func (s *SafeHTTPClient) isCircuitOpen(host string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	hc, ok := s.circuits[host]
	if !ok {
		return false
	}
	return s.nowFn().Before(hc.openUntil)
}

func (s *SafeHTTPClient) recordFailure(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hc, ok := s.circuits[host]
	if !ok {
		hc = &hostCircuit{}
		s.circuits[host] = hc
	}
	hc.failures++
	if hc.failures >= 2 {
		hc.openUntil = s.nowFn().Add(s.circuitCooldown)
	}
}

func (s *SafeHTTPClient) recordSuccess(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hc, ok := s.circuits[host]; ok {
		hc.failures = 0
		hc.openUntil = time.Time{}
	}
}
