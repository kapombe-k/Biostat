package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/clients"
	"github.com/kapombe-k/Biostat/internal/config"
	"github.com/kapombe-k/Biostat/internal/services"
	"github.com/kapombe-k/Biostat/internal/templates"
)

func setupIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("SUPABASE_URL", "")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	cfg.OfflineFallback = true
	cfg.MaxRetries = 0
	cfg.UniProtBaseURL = "http://127.0.0.1:1"
	cfg.RCSBSearchURL = "http://127.0.0.1:1"
	cfg.RCSBDataURL = "http://127.0.0.1:1"
	cfg.RCSBFilesURL = "http://127.0.0.1:1"
	cfg.AlphaFoldBaseURL = "http://127.0.0.1:1"
	cfg.PDBeBaseURL = "http://127.0.0.1:1"
	cfg.BlastBaseURL = "http://127.0.0.1:1/Blast.cgi"
	_ = cfg.RefreshAllowedHosts()

	renderer, err := templates.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	safeHTTP := clients.NewSafeHTTPClient(cfg, nil)
	blastMgr := services.NewBlastManager(cfg, clients.NewBlastClient(safeHTTP, cfg), nil)
	explorer := services.NewExplorerService(
		cfg,
		clients.NewUniProtClient(safeHTTP, cfg.UniProtBaseURL),
		clients.NewRCSBClient(safeHTTP, cfg.RCSBSearchURL, cfg.RCSBDataURL, cfg.RCSBFilesURL),
		clients.NewAlphaFoldClient(safeHTTP, cfg.AlphaFoldBaseURL),
		clients.NewPDBeClient(safeHTTP, cfg.PDBeBaseURL),
		blastMgr,
		nil,
	)

	app := NewServer(cfg, explorer, blastMgr, renderer, filepath.Join("..", "..", "static"), nil)
	return httptest.NewServer(app.Handler())
}

func TestEndToEndWorkflowIntegration(t *testing.T) {
	srv := setupIntegrationServer(t)
	defer srv.Close()

	// 1. Healthz check
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("healthz failed: status=%v err=%v", res, err)
	}
	res.Body.Close()

	// 2. Home page renders showcase molecules & guided lessons cleanly
	res, err = http.Get(srv.URL + "/")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("GET / failed: %v", err)
	}
	res.Body.Close()

	// 3. XSS HTML-injection protection on search partial
	xssQuery := `<script>alert("xss")</script>`
	res, err = http.Get(srv.URL + "/partials/search?q=" + url.QueryEscape(xssQuery))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("search partial failed: %v", err)
	}
	var bodyBuf strings.Builder
	buf := make([]byte, 4096)
	for {
		n, rerr := res.Body.Read(buf)
		if n > 0 {
			bodyBuf.Write(buf[:n])
		}
		if rerr != nil {
			break
		}
	}
	res.Body.Close()
	if strings.Contains(bodyBuf.String(), `<script>alert("xss")</script>`) {
		t.Fatalf("expected raw script tag to be HTML-escaped in search partial!")
	}

	// 4. Explore workspace for AlphaFold p53 & insertion-code Trypsin 1GBT
	for _, path := range []string{
		"/explore?q=P04637&struct=AF-P04637-F1&color=plddt",
		"/explore?q=P00760&struct=1GBT&focus=A:184A",
		"/partials/sequence?q=P69905&struct=1GZX&chain=B",
		"/api/structures/1GBT/coordinates",
		"/api/structures/AF-P04637-F1/annotations?scheme=plddt&q=P04637",
	} {
		r2, err2 := http.Get(srv.URL + path)
		if err2 != nil || r2.StatusCode != http.StatusOK {
			t.Fatalf("GET %s failed: code=%d err=%v", path, r2.StatusCode, err2)
		}
		r2.Body.Close()
	}

	// 5. Guided lessons + HTMX step scene trigger + Quiz evaluation
	stepRes, err := http.Get(srv.URL + "/lessons/enzyme-active-sites-and-inhibitors/step/2")
	if err != nil || stepRes.StatusCode != http.StatusOK {
		t.Fatalf("lesson step partial failed: %v", err)
	}
	hxTrigger := stepRes.Header.Get("HX-Trigger")
	stepRes.Body.Close()
	if !strings.Contains(hxTrigger, "bioscope:applyScene") || !strings.Contains(hxTrigger, "2HYY") {
		t.Fatalf("expected HX-Trigger header to carry 2HYY SceneSpec, got %q", hxTrigger)
	}

	quizRes, err := http.PostForm(srv.URL+"/lessons/enzyme-active-sites-and-inhibitors/quiz/q1-insertion",
		url.Values{"option": {"b"}})
	if err != nil || quizRes.StatusCode != http.StatusOK {
		t.Fatalf("quiz submission failed: %v", err)
	}
	quizRes.Body.Close()

	// 6. BLAST sequence validation failure (400) + valid reference search (200)
	badBlast, err := http.PostForm(srv.URL+"/partials/blast/submit", url.Values{"sequence": {"TOO_SHORT$$"}})
	if err != nil || badBlast.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400 on invalid BLAST sequence, got %v (err=%v)", badBlast.StatusCode, err)
	}
	badBlast.Body.Close()

	okBlast, err := http.PostForm(srv.URL+"/partials/blast/submit", url.Values{
		"sequence": {"VLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSHGSAQVKGHGKKVADALTNAVAHVDDMPNALS"},
		"database": {"swissprot"},
		"mode":     {"reference"},
	})
	if err != nil || okBlast.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 on valid BLAST submission, got %v", okBlast.StatusCode)
	}
	okBlast.Body.Close()

	// 7. Reproducible JSON & FASTA exports
	expRes, err := http.Get(srv.URL + "/api/export/json?q=P00519&struct=2HYY")
	if err != nil || expRes.StatusCode != http.StatusOK {
		t.Fatalf("export JSON failed: %v", err)
	}
	var exported bio.ExplorationExport
	if err := json.NewDecoder(expRes.Body).Decode(&exported); err != nil {
		t.Fatalf("decode exported JSON: %v", err)
	}
	expRes.Body.Close()
	if exported.ActiveModel == nil || exported.ActiveModel.ID != "2HYY" || len(exported.Provenance) < 2 {
		t.Fatalf("unexpected exported JSON payload: %+v", exported)
	}

	fastaRes, err := http.Get(srv.URL + "/api/export/fasta?q=P01308")
	if err != nil || fastaRes.StatusCode != http.StatusOK {
		t.Fatalf("export FASTA failed: %v", err)
	}
	fastaRes.Body.Close()

	// 8. User-scoped persistent library API (memory fallback in tests)
	ownerID := "11111111-1111-4111-8111-111111111111"
	recordID := "22222222-2222-4222-8222-222222222222"
	libraryPayload := `{"owner_id":"` + ownerID + `","record_key":"` + recordID + `","kind":"search","payload":{"label":"TP53","query":"P04637","href":"/explore?q=P04637"}}`
	putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/library", strings.NewReader(libraryPayload))
	putReq.Header.Set("Content-Type", "application/json")
	putRes, err := http.DefaultClient.Do(putReq)
	if err != nil || putRes.StatusCode != http.StatusNoContent {
		t.Fatalf("library save failed: %v status=%v", err, putRes)
	}
	putRes.Body.Close()
	listRes, err := http.Get(srv.URL + "/api/library?owner_id=" + ownerID)
	if err != nil || listRes.StatusCode != http.StatusOK {
		t.Fatalf("library list failed: %v status=%v", err, listRes)
	}
	var libraryData struct {
		Records []map[string]any `json:"records"`
		Durable bool             `json:"durable"`
	}
	if err := json.NewDecoder(listRes.Body).Decode(&libraryData); err != nil {
		t.Fatal(err)
	}
	listRes.Body.Close()
	if len(libraryData.Records) != 1 || libraryData.Durable {
		t.Fatalf("unexpected library payload: %#v", libraryData)
	}
	badReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/library", strings.NewReader(strings.Replace(libraryPayload, ownerID, "not-a-uuid", 1)))
	badReq.Header.Set("Content-Type", "application/json")
	badRes, _ := http.DefaultClient.Do(badReq)
	if badRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid owner to be rejected, got %d", badRes.StatusCode)
	}
	badRes.Body.Close()
	deleteReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/library/"+recordID+"?owner_id="+ownerID, nil)
	deleteRes, err := http.DefaultClient.Do(deleteReq)
	if err != nil || deleteRes.StatusCode != http.StatusNoContent {
		t.Fatalf("library delete failed: %v status=%v", err, deleteRes)
	}
	deleteRes.Body.Close()
}
