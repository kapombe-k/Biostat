package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/config"
)

func testClientConfig(t *testing.T, serverURL string) (*config.Config, *SafeHTTPClient) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.UniProtBaseURL = serverURL
	cfg.RCSBSearchURL = serverURL
	cfg.RCSBDataURL = serverURL
	cfg.RCSBFilesURL = serverURL
	cfg.AlphaFoldBaseURL = serverURL
	cfg.PDBeBaseURL = serverURL
	cfg.BlastBaseURL = serverURL + "/Blast.cgi"
	cfg.RetryBaseDelay = 10 * time.Millisecond
	cfg.BlastSubmitMinInterval = 5 * time.Millisecond
	cfg.BlastPollMinInterval = 60 * time.Second
	if err := cfg.RefreshAllowedHosts(); err != nil {
		t.Fatalf("RefreshAllowedHosts: %v", err)
	}
	return cfg, NewSafeHTTPClient(cfg, nil)
}

func TestUpstreamClientsRetryAndParsing(t *testing.T) {
	var uniprotCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/uniprotkb/P04637.json", func(w http.ResponseWriter, r *http.Request) {
		if uniprotCalls.Add(1) == 1 {
			// Simulate transient 503 on first attempt to verify automatic backoff recovery
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"primaryAccession": "P04637",
			"uniProtkbId": "P53_HUMAN",
			"entryType": "UniProtKB reviewed (Swiss-Prot)",
			"organism": {"scientificName": "Homo sapiens", "commonName": "Human", "taxonId": 9606, "lineage": ["Eukaryota", "Chordata"]},
			"proteinDescription": {"recommendedName": {"fullName": {"value": "Cellular tumor antigen p53"}}},
			"genes": [{"geneName": {"value": "TP53"}}],
			"comments": [{"commentType": "FUNCTION", "texts": [{"value": "Acts as a tumor suppressor in many tumor types."}]}],
			"features": [{"type": "Domain", "description": "p53 DNA-binding", "location": {"start": {"value": 94}, "end": {"value": 292}}}],
			"uniProtKBCrossReferences": [{"database": "PDB", "id": "1YCR", "properties": [{"key": "Method", "value": "X-ray"}, {"key": "Resolution", "value": "2.60 A"}, {"key": "Chains", "value": "B=17-29"}]}],
			"sequence": {"value": "MEEPQSDPSVEPPLSQETFSDLWKLLPENNV", "length": 31, "molWeight": 3600}
		}`))
	})

	mux.HandleFunc("/rest/v1/core/entry/1YCR", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"rcsb_id": "1YCR",
			"struct": {"title": "MDM2 BOUND TO THE TRANSACTIVATION DOMAIN OF P53"},
			"exptl": [{"method": "X-RAY DIFFRACTION"}],
			"rcsb_entry_info": {"resolution_combined": [2.6]},
			"refine": [{"ls_r_factor_r_work": 0.21, "ls_r_factor_r_free": 0.27}],
			"rcsb_accession_info": {"deposit_date": "1996-08-26T00:00:00Z", "initial_release_date": "1996-11-08T00:00:00Z"},
			"rcsb_primary_citation": {"title": "Structure of the MDM2 oncoprotein bound to p53", "rcsb_journal_abbrev": "Science", "year": 1996, "pdbx_database_id_doi": "10.1126/science.274.5289.948", "rcsb_authors": ["Kussie, P.H."]}
		}`))
	})

	mux.HandleFunc("/api/prediction/P04637", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"entryId": "AF-P04637-F1",
			"uniprotAccession": "P04637",
			"latestVersion": 6,
			"globalMetricValue": 69.8,
			"fractionPlddtVeryHigh": 0.39,
			"fractionPlddtConfident": 0.14,
			"fractionPlddtLow": 0.08,
			"fractionPlddtVeryLow": 0.39
		}]`))
	})

	mux.HandleFunc("/api/mappings/uniprot/1ycr", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"1ycr": {
				"UniProt": {
					"P04637": {
						"mappings": [{
							"chain_id": "B",
							"struct_asym_id": "B",
							"unp_start": 17,
							"unp_end": 29,
							"start": {"residue_number": 1, "author_residue_number": 17},
							"end": {"residue_number": 13, "author_residue_number": 29}
						}]
					}
				}
			}
		}`))
	})

	mux.HandleFunc("/evil-redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg, safeHTTP := testClientConfig(t, srv.URL)
	ctx := context.Background()

	// 1. UniProt retry + parsing
	uniprot := NewUniProtClient(safeHTTP, cfg.UniProtBaseURL)
	rec, err := uniprot.GetProtein(ctx, "P04637")
	if err != nil {
		t.Fatalf("GetProtein failed: %v", err)
	}
	if !rec.Reviewed || rec.EntryName != "P53_HUMAN" || uniprotCalls.Load() != 2 {
		t.Fatalf("unexpected UniProt result: reviewed=%v entry=%s calls=%d", rec.Reviewed, rec.EntryName, uniprotCalls.Load())
	}

	// 2. RCSB entry summary
	rcsb := NewRCSBClient(safeHTTP, cfg.RCSBSearchURL, cfg.RCSBDataURL, cfg.RCSBFilesURL)
	sum, enrich, err := rcsb.GetEntrySummary(ctx, "1YCR")
	if err != nil {
		t.Fatalf("GetEntrySummary failed: %v", err)
	}
	if sum.ResolutionAngstrom == nil || *sum.ResolutionAngstrom != 2.6 || enrich.DOI != "10.1126/science.274.5289.948" {
		t.Fatalf("unexpected RCSB summary: %+v", sum)
	}

	// 3. AlphaFold DB prediction
	af := NewAlphaFoldClient(safeHTTP, cfg.AlphaFoldBaseURL)
	pred, err := af.GetPrediction(ctx, "P04637")
	if err != nil || pred.EntryID != "AF-P04637-F1" || pred.ModelVersion != 6 {
		t.Fatalf("unexpected AlphaFold prediction: %+v, err=%v", pred, err)
	}

	// 4. PDBe SIFTS mapping
	pdbe := NewPDBeClient(safeHTTP, cfg.PDBeBaseURL)
	sifts, err := pdbe.GetSIFTSMappings(ctx, "1YCR")
	if err != nil || len(sifts) != 1 || sifts[0].UniProtStart != 17 {
		t.Fatalf("unexpected SIFTS mapping: %+v, err=%v", sifts, err)
	}

	// 5. SSRF protection across HTTP redirects
	if _, _, err := safeHTTP.Do(ctx, http.MethodGet, srv.URL+"/evil-redirect", "", nil); err == nil {
		t.Fatalf("expected SSRF check to block cross-host redirect to metadata IP")
	}
}

func TestBlastRateLimitingAndXMLParsing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/Blast.cgi", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte("<!--\nQBlastInfoBegin\n\tRID = TESTRID99\n\tRTOE = 18\nQBlastInfoEnd\n-->"))
			return
		}
		if r.URL.Query().Get("FORMAT_OBJECT") == "SearchInfo" {
			_, _ = w.Write([]byte("QBlastInfoBegin\n\tStatus=READY\nQBlastInfoEnd\n"))
			return
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<BlastOutput>
  <BlastOutput_iterations>
    <Iteration>
      <Iteration_hits>
        <Hit>
          <Hit_num>1</Hit_num>
          <Hit_id>sp|P69905.2|HBA_HUMAN</Hit_id>
          <Hit_def>RecName: Full=Hemoglobin subunit alpha [Homo sapiens]</Hit_def>
          <Hit_accession>P69905</Hit_accession>
          <Hit_len>142</Hit_len>
          <Hit_hsps>
            <Hsp>
              <Hsp_bit-score>285.4</Hsp_bit-score>
              <Hsp_evalue>1.4e-98</Hsp_evalue>
              <Hsp_query-from>1</Hsp_query-from>
              <Hsp_query-to>141</Hsp_query-to>
              <Hsp_hit-from>2</Hsp_hit-from>
              <Hsp_hit-to>142</Hsp_hit-to>
              <Hsp_identity>141</Hsp_identity>
              <Hsp_align-len>141</Hsp_align-len>
              <Hsp_qseq>VLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH</Hsp_qseq>
              <Hsp_hseq>VLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH</Hsp_hseq>
              <Hsp_midline>VLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH</Hsp_midline>
            </Hsp>
          </Hit_hsps>
        </Hit>
      </Iteration_hits>
    </Iteration>
  </BlastOutput_iterations>
</BlastOutput>`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg, safeHTTP := testClientConfig(t, srv.URL)
	blast := NewBlastClient(safeHTTP, cfg)
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	blast.nowFn = func() time.Time { return now }

	ctx := context.Background()
	rid, rtoe, _, err := blast.SubmitSearch(ctx, "swissprot", "VLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSH")
	if err != nil || rid != "TESTRID99" || rtoe != 18 {
		t.Fatalf("unexpected submit response: rid=%s rtoe=%d err=%v", rid, rtoe, err)
	}

	// Immediate poll after submit (< 60s) MUST be blocked by NCBI rate limit policy!
	now = now.Add(15 * time.Second)
	if _, _, err := blast.PollStatus(ctx, rid); !errors.Is(err, ErrPollRateLimited) {
		t.Fatalf("expected ErrPollRateLimited after 15s, got %v", err)
	}

	// Poll after 61 seconds succeeds!
	now = now.Add(46 * time.Second)
	status, _, err := blast.PollStatus(ctx, rid)
	if err != nil || status != bio.BlastReady {
		t.Fatalf("expected READY status after 61s, got status=%s err=%v", status, err)
	}

	hits, err := blast.FetchResults(ctx, rid, 141)
	if err != nil || len(hits) != 1 || hits[0].UniProtGuess != "P69905" || hits[0].IdentityPct != 100 {
		t.Fatalf("unexpected BLAST parsed hits: %+v, err=%v", hits, err)
	}

	// Verify per-host circuit breaker trips on consecutive upstream 403 proxy rejections and recovers after cooldown
	var tripHits atomic.Int32
	tripSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		c := tripHits.Add(1)
		if c <= 2 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer tripSrv.Close()

	_, tripClient := testClientConfig(t, tripSrv.URL)
	cbTime := time.Date(2026, 10, 23, 13, 0, 0, 0, time.UTC)
	tripClient.nowFn = func() time.Time { return cbTime }

	_, _, _ = tripClient.Do(ctx, http.MethodGet, tripSrv.URL+"/probe", "", nil)
	_, _, _ = tripClient.Do(ctx, http.MethodGet, tripSrv.URL+"/probe", "", nil)
	if _, _, err := tripClient.Do(ctx, http.MethodGet, tripSrv.URL+"/probe", "", nil); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen on 3rd call, got %v", err)
	}
	cbTime = cbTime.Add(60 * time.Second)
	if _, code, err := tripClient.Do(ctx, http.MethodGet, tripSrv.URL+"/probe", "", nil); err != nil || code != 200 {
		t.Fatalf("expected circuit recovery after cooldown, got code=%d err=%v", code, err)
	}
}

