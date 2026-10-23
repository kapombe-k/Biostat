package services

import (
	"context"
	"testing"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/clients"
	"github.com/kapombe-k/Biostat/internal/config"
)

func newTestExplorer(t *testing.T) *ExplorerService {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.OfflineFallback = true
	// Point upstream URLs at an unreachable closed port on allowed host so seed fallback triggers deterministically in <5ms
	cfg.UniProtBaseURL = "http://127.0.0.1:1"
	cfg.RCSBSearchURL = "http://127.0.0.1:1"
	cfg.RCSBDataURL = "http://127.0.0.1:1"
	cfg.RCSBFilesURL = "http://127.0.0.1:1"
	cfg.AlphaFoldBaseURL = "http://127.0.0.1:1"
	cfg.PDBeBaseURL = "http://127.0.0.1:1"
	cfg.BlastBaseURL = "http://127.0.0.1:1/Blast.cgi"
	cfg.MaxRetries = 0
	_ = cfg.RefreshAllowedHosts()

	safeHTTP := clients.NewSafeHTTPClient(cfg, nil)
	blastMgr := NewBlastManager(cfg, clients.NewBlastClient(safeHTTP, cfg), nil)
	return NewExplorerService(
		cfg,
		clients.NewUniProtClient(safeHTTP, cfg.UniProtBaseURL),
		clients.NewRCSBClient(safeHTTP, cfg.RCSBSearchURL, cfg.RCSBDataURL, cfg.RCSBFilesURL),
		clients.NewAlphaFoldClient(safeHTTP, cfg.AlphaFoldBaseURL),
		clients.NewPDBeClient(safeHTTP, cfg.PDBeBaseURL),
		blastMgr,
		nil,
	)
}

func TestMapperZeroFalseEquivalenciesAcrossComplexStructures(t *testing.T) {
	exp := newTestExplorer(t)
	ctx := context.Background()

	// 1. Heterodimer 1YCR (MDM2 Chain A + p53 Chain B) against p53 (P04637)
	b1, err := exp.ResolveWorkspace(ctx, "P04637", "1YCR", "")
	if err != nil {
		t.Fatalf("ResolveWorkspace(P04637, 1YCR): %v", err)
	}
	if b1.ActiveChain == nil || b1.ActiveChain.AuthAsymID != "B" {
		t.Fatalf("expected active chain to auto-select p53 Chain B, got %+v", b1.ActiveChain)
	}
	if b1.ActiveChain.UniProtStart != 17 || b1.ActiveChain.UniProtEnd != 29 {
		t.Fatalf("expected p53 Chain B mapped to UniProt 17..29, got %d..%d", b1.ActiveChain.UniProtStart, b1.ActiveChain.UniProtEnd)
	}
	// Verify partner Chain A (MDM2) has ZERO false UniProt equivalencies to p53!
	chainA := b1.ActiveStructure.Chains[0]
	if chainA.MappingMethod != "heteromeric-partner-chain" {
		t.Fatalf("expected MDM2 Chain A to be marked heteromeric-partner-chain, got %s", chainA.MappingMethod)
	}
	for _, r := range chainA.Residues {
		if r.UniProtPos != nil {
			t.Fatalf("expected nil UniProtPos on partner Chain A residue %s, got %d", r.AuthLabel, *r.UniProtPos)
		}
	}

	// 2. Mature cleaved two-chain Insulin (4INS) against Preproinsulin (P01308)
	bIns, err := exp.ResolveWorkspace(ctx, "P01308", "4INS", "A")
	if err != nil {
		t.Fatalf("ResolveWorkspace(P01308, 4INS): %v", err)
	}
	if bIns.ActiveChain.UniProtStart != 90 || bIns.ActiveChain.UniProtEnd != 110 {
		t.Fatalf("expected Insulin Chain A mapped to UniProt 90..110, got %d..%d", bIns.ActiveChain.UniProtStart, bIns.ActiveChain.UniProtEnd)
	}

	// 3. Internal missing density loop inside ABL1 kinase (2HYY)
	bAbl, err := exp.ResolveWorkspace(ctx, "P00519", "2HYY", "A")
	if err != nil {
		t.Fatalf("ResolveWorkspace(P00519, 2HYY): %v", err)
	}
	if bAbl.ActiveChain.MissingCount != 1 || bAbl.ActiveChain.PercentIdentity < 99.0 {
		t.Fatalf("expected 1 missing residue with continuous alignment across 2HYY, got missing=%d id=%.1f",
			bAbl.ActiveChain.MissingCount, bAbl.ActiveChain.PercentIdentity)
	}

	// 4. Export record & MVS annotations
	anns := BuildMVSAnnotations(bAbl.ActiveStructure, "secondary")
	if len(anns) != bAbl.ActiveChain.ObservedCount {
		t.Fatalf("expected %d MVS annotations, got %d", bAbl.ActiveChain.ObservedCount, len(anns))
	}
	export := exp.BuildExport(bAbl, bio.ExplorationState{
		Accession: "P00519", StructureID: "2HYY", ChainID: "A", Representation: "cartoon", ColorScheme: "secondary",
	}, "https://bioscope.hf.space")
	if len(export.Provenance) < 2 || export.ActiveModel.CoordinateSHA == "" {
		t.Fatalf("expected complete export provenance and coordinate SHA256")
	}

	// 5. Search modes (keyword, PDB ID, raw FASTA sequence) + all MVS color schemes + cache metrics
	for _, q := range []string{"hemoglobin", "1GBT", ">query\nVLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH"} {
		kind, hits, _, err := exp.Search(ctx, q)
		if err != nil || len(hits) == 0 || kind == "" {
			t.Fatalf("Search(%q) failed: hits=%d kind=%s err=%v", q, len(hits), kind, err)
		}
	}
	for _, scheme := range []string{"plddt", "bfactor", "hydrophobicity", "chain"} {
		if got := BuildMVSAnnotations(bAbl.ActiveStructure, scheme); len(got) == 0 {
			t.Fatalf("expected annotations for scheme %s", scheme)
		}
	}
	if stats := exp.CacheStats(); len(stats) != 4 {
		t.Fatalf("expected 4 cache buckets, got %d", len(stats))
	}

	// 6. BlastManager reference + fallback execution & status lookup
	job, _, err := exp.blast.Submit(ctx, ">hba\nVLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH", "swissprot", "reference")
	if err != nil || job.Status != bio.BlastReady || len(job.Hits) == 0 {
		t.Fatalf("BlastManager.Submit reference failed: job=%+v err=%v", job, err)
	}
	fetched, _, gerr := exp.blast.GetAndPoll(ctx, job.JobID)
	if gerr != nil || fetched.JobID != job.JobID {
		t.Fatalf("BlastManager.GetAndPoll failed for %s: %v", job.JobID, gerr)
	}
}

