package clients

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/config"
)

// ErrPollRateLimited indicates a status poll attempted before NCBI's minimum per-RID poll window elapsed.
var ErrPollRateLimited = errors.New("NCBI BLAST rate policy limits status polling to at most once per minute per RID")

var (
	reBlastRID    = regexp.MustCompile(`(?m)^\s*RID\s*=\s*([A-Z0-9-]+)\s*$`)
	reBlastRTOE   = regexp.MustCompile(`(?m)^\s*RTOE\s*=\s*(\d+)\s*$`)
	reBlastStatus = regexp.MustCompile(`(?m)^\s*Status\s*=\s*([A-Z]+)\s*$`)
	reValidRID    = regexp.MustCompile(`^[A-Z0-9-]{4,32}$`)
	reSpeciesBrk  = regexp.MustCompile(`\[([^\]]+)\]\s*$`)
)

var allowedBlastDatabases = map[string]bool{
	"swissprot":      true,
	"pdb":            true,
	"refseq_protein": true,
	"nr":             true,
}

// HTTPBlastClient implements BlastClient against https://blast.ncbi.nlm.nih.gov/Blast.cgi
// while strictly enforcing NCBI global request pacing and per-RID 60-second poll intervals.
type HTTPBlastClient struct {
	http            *SafeHTTPClient
	baseURL         string
	contactEmail    string
	minPollInterval time.Duration
	minSubmitGap    time.Duration

	mu           sync.Mutex
	lastSubmitAt time.Time
	lastPollByID map[string]time.Time
	nowFn        func() time.Time
}

// NewBlastClient constructs a rate-policy-enforcing NCBI QBlast client.
func NewBlastClient(httpClient *SafeHTTPClient, cfg *config.Config) *HTTPBlastClient {
	return &HTTPBlastClient{
		http:            httpClient,
		baseURL:         strings.TrimSpace(cfg.BlastBaseURL),
		contactEmail:    cfg.ContactEmail,
		minPollInterval: cfg.BlastPollMinInterval,
		minSubmitGap:    cfg.BlastSubmitMinInterval,
		lastPollByID:    make(map[string]time.Time),
		nowFn:           time.Now,
	}
}

// SubmitSearch submits an asynchronous blastp search (`CMD=Put`) with validated parameters.
func (c *HTTPBlastClient) SubmitSearch(ctx context.Context, database, sequence string) (string, int, string, error) {
	db := strings.ToLower(strings.TrimSpace(database))
	if db == "" {
		db = "swissprot"
	}
	if !allowedBlastDatabases[db] {
		return "", 0, "", fmt.Errorf("unsupported BLAST database %q; allowed: swissprot, pdb, refseq_protein, nr", database)
	}

	// Enforce minimum spacing between submissions to protect NCBI shared infrastructure.
	c.mu.Lock()
	elapsed := c.nowFn().Sub(c.lastSubmitAt)
	if !c.lastSubmitAt.IsZero() && elapsed < c.minSubmitGap {
		wait := c.minSubmitGap - elapsed
		c.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return "", 0, "", ctx.Err()
		}
		c.mu.Lock()
	}
	c.lastSubmitAt = c.nowFn()
	c.mu.Unlock()

	form := url.Values{}
	form.Set("CMD", "Put")
	form.Set("PROGRAM", "blastp")
	form.Set("DATABASE", db)
	form.Set("QUERY", sequence)
	form.Set("HITLIST_SIZE", "15")
	form.Set("FORMAT_TYPE", "XML")
	form.Set("TOOL", "BioScope")
	form.Set("EMAIL", c.contactEmail)

	raw, _, err := c.http.Do(ctx, http.MethodPost, c.baseURL, "application/x-www-form-urlencoded", []byte(form.Encode()))
	if err != nil {
		return "", 0, c.baseURL, err
	}

	text := string(raw)
	ridMatch := reBlastRID.FindStringSubmatch(text)
	if len(ridMatch) < 2 {
		return "", 0, c.baseURL, errors.New("NCBI BLAST response did not contain a valid Request ID (RID)")
	}
	rid := strings.TrimSpace(ridMatch[1])
	rtoe := 20
	if rtMatch := reBlastRTOE.FindStringSubmatch(text); len(rtMatch) >= 2 {
		if n, err := strconv.Atoi(rtMatch[1]); err == nil && n > 0 {
			rtoe = n
		}
	}

	c.mu.Lock()
	c.lastPollByID[rid] = c.nowFn()
	c.mu.Unlock()

	return rid, rtoe, c.baseURL, nil
}

// PollStatus queries `CMD=Get&FORMAT_OBJECT=SearchInfo` while strictly rejecting polls faster than minPollInterval.
func (c *HTTPBlastClient) PollStatus(ctx context.Context, rid string) (bio.BlastJobStatus, string, error) {
	cleanRID := strings.ToUpper(strings.TrimSpace(rid))
	if !reValidRID.MatchString(cleanRID) {
		return bio.BlastFailed, "", errors.New("invalid BLAST RID format")
	}

	c.mu.Lock()
	last, exists := c.lastPollByID[cleanRID]
	now := c.nowFn()
	if exists && now.Sub(last) < c.minPollInterval {
		c.mu.Unlock()
		return bio.BlastRunning, "", ErrPollRateLimited
	}
	c.lastPollByID[cleanRID] = now
	c.mu.Unlock()

	q := url.Values{}
	q.Set("CMD", "Get")
	q.Set("FORMAT_OBJECT", "SearchInfo")
	q.Set("RID", cleanRID)
	q.Set("TOOL", "BioScope")
	q.Set("EMAIL", c.contactEmail)

	endpoint := c.baseURL + "?" + q.Encode()
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return bio.BlastRunning, "", err
	}
	m := reBlastStatus.FindStringSubmatch(string(raw))
	if len(m) < 2 {
		return bio.BlastRunning, "Waiting for NCBI scheduler status", nil
	}
	switch m[1] {
	case "WAITING":
		return bio.BlastRunning, "Search queued / executing on NCBI BLAST cluster", nil
	case "READY":
		return bio.BlastReady, "Alignments ready", nil
	case "FAILED", "UNKNOWN":
		return bio.BlastFailed, fmt.Sprintf("NCBI reported status %s", m[1]), nil
	default:
		return bio.BlastRunning, m[1], nil
	}
}

type blastXMLOutput struct {
	Iterations []struct {
		Hits []struct {
			Num       int    `xml:"Hit_num"`
			ID        string `xml:"Hit_id"`
			Def       string `xml:"Hit_def"`
			Accession string `xml:"Hit_accession"`
			Len       int    `xml:"Hit_len"`
			Hsps      []struct {
				BitScore  float64 `xml:"Hsp_bit-score"`
				Evalue    float64 `xml:"Hsp_evalue"`
				QueryFrom int     `xml:"Hsp_query-from"`
				QueryTo   int     `xml:"Hsp_query-to"`
				HitFrom   int     `xml:"Hsp_hit-from"`
				HitTo     int     `xml:"Hsp_hit-to"`
				Identity  int     `xml:"Hsp_identity"`
				AlignLen  int     `xml:"Hsp_align-len"`
				Qseq      string  `xml:"Hsp_qseq"`
				Hseq      string  `xml:"Hsp_hseq"`
				Midline   string  `xml:"Hsp_midline"`
			} `xml:"Hit_hsps>Hsp"`
		} `xml:"Iteration_hits>Hit"`
	} `xml:"BlastOutput_iterations>Iteration"`
}

// FetchResults downloads and parses XML BLAST alignments (`CMD=Get&FORMAT_TYPE=XML`) for a completed RID.
func (c *HTTPBlastClient) FetchResults(ctx context.Context, rid string, queryLen int) ([]bio.BlastHit, error) {
	cleanRID := strings.ToUpper(strings.TrimSpace(rid))
	if !reValidRID.MatchString(cleanRID) {
		return nil, errors.New("invalid BLAST RID format")
	}

	q := url.Values{}
	q.Set("CMD", "Get")
	q.Set("FORMAT_TYPE", "XML")
	q.Set("RID", cleanRID)
	q.Set("TOOL", "BioScope")
	q.Set("EMAIL", c.contactEmail)

	endpoint := c.baseURL + "?" + q.Encode()
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	return ParseBlastXML(raw, queryLen)
}

// ParseBlastXML parses standard NCBI BlastOutput XML into normalized BlastHit records.
func ParseBlastXML(raw []byte, queryLen int) ([]bio.BlastHit, error) {
	var parsed blastXMLOutput
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse BLAST XML output: %w", err)
	}

	var out []bio.BlastHit
	for _, it := range parsed.Iterations {
		for _, h := range it.Hits {
			if len(h.Hsps) == 0 {
				continue
			}
			best := h.Hsps[0]
			identPct := 0.0
			if best.AlignLen > 0 {
				identPct = math.Round((float64(best.Identity)/float64(best.AlignLen))*1000) / 10
			}
			coverPct := 0.0
			if queryLen > 0 {
				span := best.QueryTo - best.QueryFrom + 1
				if span > queryLen {
					span = queryLen
				}
				if span > 0 {
					coverPct = math.Round((float64(span)/float64(queryLen))*1000) / 10
				}
			}

			defLine := strings.TrimSpace(h.Def)
			if cut := strings.Index(defLine, " >"); cut > 0 {
				defLine = defLine[:cut]
			}
			org := ""
			if m := reSpeciesBrk.FindStringSubmatch(defLine); len(m) >= 2 {
				org = m[1]
			} else if idx := strings.Index(defLine, " OS="); idx > 0 {
				rest := defLine[idx+4:]
				if end := strings.Index(rest, " OX="); end > 0 {
					org = rest[:end]
				}
				defLine = strings.TrimSpace(defLine[:idx])
			}

			acc := strings.TrimSpace(h.Accession)
			unpGuess, pdbGuess := inferLinkedIDs(h.ID, acc)

			out = append(out, bio.BlastHit{
				Rank:          len(out) + 1,
				Accession:     acc,
				UniProtGuess:  unpGuess,
				PDBChainGuess: pdbGuess,
				Description:   defLine,
				Organism:      org,
				Length:        h.Len,
				BitScore:      math.Round(best.BitScore*10) / 10,
				EValue:        best.Evalue,
				IdentityPct:   identPct,
				QueryCoverPct: coverPct,
				QueryFrom:     best.QueryFrom,
				QueryTo:       best.QueryTo,
				HitFrom:       best.HitFrom,
				HitTo:         best.HitTo,
				QueryAlign:    best.Qseq,
				Midline:       best.Midline,
				HitAlign:      best.Hseq,
			})
		}
	}
	return out, nil
}

func inferLinkedIDs(hitID, acc string) (string, string) {
	// e.g. sp|P04637.4|P53_HUMAN or pdb|1GZX|A
	parts := strings.Split(hitID, "|")
	if len(parts) >= 2 {
		switch strings.ToLower(parts[0]) {
		case "sp", "tr":
			return strings.Split(parts[1], ".")[0], ""
		case "pdb":
			return "", strings.ToUpper(parts[1])
		}
	}
	baseAcc := strings.Split(acc, ".")[0]
	k, norm := bio.ClassifyQuery(baseAcc)
	switch k {
	case bio.QueryUniProtAccession:
		return norm, ""
	case bio.QueryPDBID:
		return "", norm
	}
	if len(baseAcc) == 6 && strings.Contains(baseAcc, "_") {
		p := strings.Split(baseAcc, "_")
		if len(p[0]) == 4 {
			return "", strings.ToUpper(p[0])
		}
	}
	return "", ""
}
