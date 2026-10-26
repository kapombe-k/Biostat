package config

import (
	"testing"
)

func TestLoadDefaultsAndSSRFGuard(t *testing.T) {
	t.Setenv("PORT", "7860")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Addr() != "0.0.0.0:7860" {
		t.Fatalf("expected 0.0.0.0:7860, got %s", cfg.Addr())
	}

	allowed := []string{
		"https://rest.uniprot.org/uniprotkb/P04637.json",
		"https://data.rcsb.org/rest/v1/core/entry/1GZX",
		"https://files.rcsb.org/download/1GZX.pdb",
		"https://alphafold.ebi.ac.uk/api/prediction/P04637",
		"https://www.ebi.ac.uk/pdbe/api/pdb/entry/summary/1gzx",
		"https://blast.ncbi.nlm.nih.gov/Blast.cgi?CMD=Get",
	}
	for _, u := range allowed {
		if err := cfg.IsAllowedURL(u); err != nil {
			t.Errorf("expected %s to be allowed, got: %v", u, err)
		}
	}

	blocked := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost:8080/internal",
		"https://evil.example.com/uniprotkb/P04637.json",
		"file:///etc/passwd",
		"https://user:pass@rest.uniprot.org/uniprotkb/P04637.json",
	}
	for _, u := range blocked {
		if err := cfg.IsAllowedURL(u); err == nil {
			t.Errorf("expected SSRF policy to block %s", u)
		}
	}
}
