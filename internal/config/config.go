package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds validated runtime configuration for BioScope.
type Config struct {
	Host        string
	Port        int
	LogLevel    string
	Environment string

	// Cache settings
	CacheMaxItems int
	CacheTTL      time.Duration

	// Outbound HTTP & safety bounds
	HTTPTimeout       time.Duration
	MaxRetries        int
	RetryBaseDelay    time.Duration
	MaxResponseBytes  int64
	MaxSequenceLength int
	MinSequenceLength int
	ContactEmail      string
	OfflineFallback   bool

	// Upstream scientific API endpoints
	UniProtBaseURL   string
	RCSBSearchURL    string
	RCSBDataURL      string
	RCSBFilesURL     string
	AlphaFoldBaseURL string
	PDBeBaseURL      string
	BlastBaseURL     string

	// NCBI BLAST job & rate limit enforcement
	BlastMaxJobs           int
	BlastPollMinInterval   time.Duration
	BlastSubmitMinInterval time.Duration

	allowedHosts map[string]struct{}
}

// Load reads environment variables and returns a validated Config with sensible defaults
// compatible with Hugging Face Spaces (0.0.0.0:7860) and local development.
func Load() (*Config, error) {
	cfg := &Config{
		Host:                   envString("BIOSCOPE_HOST", "0.0.0.0"),
		Port:                   envInt("PORT", envInt("BIOSCOPE_PORT", 7860)),
		LogLevel:               strings.ToLower(envString("BIOSCOPE_LOG_LEVEL", "info")),
		Environment:            envString("BIOSCOPE_ENV", "production"),
		CacheMaxItems:          envInt("BIOSCOPE_CACHE_MAX_ITEMS", 512),
		CacheTTL:               time.Duration(envInt("BIOSCOPE_CACHE_TTL_SECONDS", 1800)) * time.Second,
		HTTPTimeout:            time.Duration(envInt("BIOSCOPE_HTTP_TIMEOUT_SECONDS", 12)) * time.Second,
		MaxRetries:             envInt("BIOSCOPE_HTTP_MAX_RETRIES", 2),
		RetryBaseDelay:         time.Duration(envInt("BIOSCOPE_HTTP_RETRY_BASE_MS", 300)) * time.Millisecond,
		MaxResponseBytes:       int64(envInt("BIOSCOPE_MAX_RESPONSE_BYTES", 25*1024*1024)),
		MaxSequenceLength:      envInt("BIOSCOPE_MAX_SEQUENCE_LENGTH", 5000),
		MinSequenceLength:      envInt("BIOSCOPE_MIN_SEQUENCE_LENGTH", 10),
		ContactEmail:           envString("BIOSCOPE_CONTACT_EMAIL", "bioscope-learning-lab@users.noreply.github.com"),
		OfflineFallback:        envBool("BIOSCOPE_OFFLINE_FALLBACK", true),
		UniProtBaseURL:         strings.TrimRight(envString("BIOSCOPE_UNIPROT_BASE_URL", "https://rest.uniprot.org"), "/"),
		RCSBSearchURL:          strings.TrimRight(envString("BIOSCOPE_RCSB_SEARCH_URL", "https://search.rcsb.org"), "/"),
		RCSBDataURL:            strings.TrimRight(envString("BIOSCOPE_RCSB_DATA_URL", "https://data.rcsb.org"), "/"),
		RCSBFilesURL:           strings.TrimRight(envString("BIOSCOPE_RCSB_FILES_URL", "https://files.rcsb.org"), "/"),
		AlphaFoldBaseURL:       strings.TrimRight(envString("BIOSCOPE_ALPHAFOLD_BASE_URL", "https://alphafold.ebi.ac.uk"), "/"),
		PDBeBaseURL:            strings.TrimRight(envString("BIOSCOPE_PDBE_BASE_URL", "https://www.ebi.ac.uk/pdbe"), "/"),
		BlastBaseURL:           strings.TrimRight(envString("BIOSCOPE_BLAST_BASE_URL", "https://blast.ncbi.nlm.nih.gov/Blast.cgi"), "/"),
		BlastMaxJobs:           envInt("BIOSCOPE_BLAST_MAX_JOBS", 32),
		BlastPollMinInterval:   time.Duration(envInt("BIOSCOPE_BLAST_POLL_MIN_SECONDS", 60)) * time.Second,
		BlastSubmitMinInterval: time.Duration(envInt("BIOSCOPE_BLAST_SUBMIT_MIN_INTERVAL_MS", 3500)) * time.Millisecond,
	}

	if err := cfg.RefreshAllowedHosts(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// RefreshAllowedHosts builds the strict SSRF outbound allowlist strictly from configured API base URLs.
func (c *Config) RefreshAllowedHosts() error {
	c.allowedHosts = make(map[string]struct{})
	endpoints := []struct {
		name string
		raw  string
	}{
		{"UniProtBaseURL", c.UniProtBaseURL},
		{"RCSBSearchURL", c.RCSBSearchURL},
		{"RCSBDataURL", c.RCSBDataURL},
		{"RCSBFilesURL", c.RCSBFilesURL},
		{"AlphaFoldBaseURL", c.AlphaFoldBaseURL},
		{"PDBeBaseURL", c.PDBeBaseURL},
		{"BlastBaseURL", c.BlastBaseURL},
	}

	for _, ep := range endpoints {
		u, err := url.Parse(ep.raw)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return fmt.Errorf("invalid %s %q: must be a valid http(s) URL", ep.name, ep.raw)
		}
		c.allowedHosts[strings.ToLower(u.Host)] = struct{}{}
	}
	return nil
}

// IsAllowedURL verifies that an outbound request targets only an explicitly configured scientific host.
func (c *Config) IsAllowedURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("disallowed URL scheme %q", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("URL userinfo is not allowed")
	}
	host := strings.ToLower(u.Host)
	if _, ok := c.allowedHosts[host]; !ok {
		return fmt.Errorf("outbound host %q blocked by BioScope SSRF policy", host)
	}
	return nil
}

// Addr returns the host:port listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
