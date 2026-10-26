package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/kapombe-k/Biostat/internal/bio"
)

// HTTPRCSBClient implements RCSBClient against RCSB PDB Search, Data, and Coordinate File APIs.
type HTTPRCSBClient struct {
	http      *SafeHTTPClient
	searchURL string
	dataURL   string
	filesURL  string
}

// NewRCSBClient creates a production RCSB PDB client.
func NewRCSBClient(httpClient *SafeHTTPClient, searchURL, dataURL, filesURL string) *HTTPRCSBClient {
	return &HTTPRCSBClient{
		http:      httpClient,
		searchURL: strings.TrimRight(searchURL, "/"),
		dataURL:   strings.TrimRight(dataURL, "/"),
		filesURL:  strings.TrimRight(filesURL, "/"),
	}
}

type rcsbSearchResponse struct {
	ResultSet []struct {
		Identifier string `json:"identifier"`
	} `json:"result_set"`
}

// SearchByUniProt finds experimental PDB entry IDs matching a UniProt accession.
func (c *HTTPRCSBClient) SearchByUniProt(ctx context.Context, accession string, limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	payload := map[string]any{
		"query": map[string]any{
			"type": "terminal",
			"service": "text",
			"parameters": map[string]any{
				"attribute": "rcsb_polymer_entity_container_identifiers.reference_sequence_identifiers.database_accession",
				"operator":  "in",
				"value":     []string{strings.ToUpper(strings.TrimSpace(accession))},
			},
		},
		"return_type": "entry",
		"request_options": map[string]any{
			"paginate": map[string]int{"start": 0, "rows": limit},
		},
	}
	return c.executeSearch(ctx, payload)
}

// SearchByKeyword finds experimental PDB entry IDs matching free text.
func (c *HTTPRCSBClient) SearchByKeyword(ctx context.Context, keyword string, limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	payload := map[string]any{
		"query": map[string]any{
			"type":    "terminal",
			"service": "full_text",
			"parameters": map[string]any{
				"value": strings.TrimSpace(keyword),
			},
		},
		"return_type": "entry",
		"request_options": map[string]any{
			"paginate": map[string]int{"start": 0, "rows": limit},
		},
	}
	return c.executeSearch(ctx, payload)
}

func (c *HTTPRCSBClient) executeSearch(ctx context.Context, payload map[string]any) ([]string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/rcsbsearch/v2/query", c.searchURL)
	raw, status, err := c.http.Do(ctx, http.MethodPost, endpoint, "application/json", body)
	if status == http.StatusNoContent {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var resp rcsbSearchResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse RCSB search response: %w", err)
	}
	ids := make([]string, 0, len(resp.ResultSet))
	for _, r := range resp.ResultSet {
		if r.Identifier != "" {
			ids = append(ids, strings.ToUpper(r.Identifier))
		}
	}
	return ids, nil
}

type rcsbEntryJSON struct {
	RcsbID string `json:"rcsb_id"`
	Struct struct {
		Title string `json:"title"`
	} `json:"struct"`
	Exptl []struct {
		Method string `json:"method"`
	} `json:"exptl"`
	RcsbEntryInfo struct {
		ResolutionCombined   []float64 `json:"resolution_combined"`
		PolymerEntityCount   int       `json:"polymer_entity_count"`
		NonpolymerCompIDs    []string  `json:"nonpolymer_bound_components"`
	} `json:"rcsb_entry_info"`
	Refine []struct {
		RWork float64 `json:"ls_r_factor_r_work"`
		RFree float64 `json:"ls_r_factor_r_free"`
	} `json:"refine"`
	RcsbAccessionInfo struct {
		DepositDate        string `json:"deposit_date"`
		InitialReleaseDate string `json:"initial_release_date"`
	} `json:"rcsb_accession_info"`
	RcsbPrimaryCitation struct {
		Title   string   `json:"title"`
		Journal string   `json:"rcsb_journal_abbrev"`
		Year    int      `json:"year"`
		DOI     string   `json:"pdbx_database_id_doi"`
		Authors []string `json:"rcsb_authors"`
	} `json:"rcsb_primary_citation"`
}

// GetEntrySummary fetches core experimental metadata for a 4-character PDB entry.
func (c *HTTPRCSBClient) GetEntrySummary(ctx context.Context, pdbID string) (*bio.StructureSummary, *EntryEnrichment, error) {
	cleanID := strings.ToUpper(strings.TrimSpace(pdbID))
	endpoint := fmt.Sprintf("%s/rest/v1/core/entry/%s", c.dataURL, url.PathEscape(cleanID))
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, nil, err
	}
	var entry rcsbEntryJSON
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, nil, fmt.Errorf("parse RCSB entry %s: %w", cleanID, err)
	}

	method := "EXPERIMENTAL STRUCTURE"
	if len(entry.Exptl) > 0 && entry.Exptl[0].Method != "" {
		method = entry.Exptl[0].Method
	}
	var resPtr *float64
	if len(entry.RcsbEntryInfo.ResolutionCombined) > 0 && entry.RcsbEntryInfo.ResolutionCombined[0] > 0 {
		v := entry.RcsbEntryInfo.ResolutionCombined[0]
		resPtr = &v
	}
	var rwPtr, rfPtr *float64
	if len(entry.Refine) > 0 {
		if entry.Refine[0].RWork > 0 {
			v := entry.Refine[0].RWork
			rwPtr = &v
		}
		if entry.Refine[0].RFree > 0 {
			v := entry.Refine[0].RFree
			rfPtr = &v
		}
	}

	var citation string
	if entry.RcsbPrimaryCitation.Title != "" {
		lead := ""
		if len(entry.RcsbPrimaryCitation.Authors) > 0 {
			lead = entry.RcsbPrimaryCitation.Authors[0] + " et al. "
		}
		citation = fmt.Sprintf("%s(%d) %s. %s", lead, entry.RcsbPrimaryCitation.Year, entry.RcsbPrimaryCitation.Title, entry.RcsbPrimaryCitation.Journal)
	}
	release := cleanDatePrefix(entry.RcsbAccessionInfo.InitialReleaseDate)

	summary := &bio.StructureSummary{
		ID:                 cleanID,
		Kind:               bio.StructureExperimental,
		SourceDB:           "RCSB PDB",
		Title:              entry.Struct.Title,
		Method:             method,
		ResolutionAngstrom: resPtr,
		LigandIDs:          entry.RcsbEntryInfo.NonpolymerCompIDs,
		ReleaseDate:        release,
	}
	enrich := &EntryEnrichment{
		Title:          entry.Struct.Title,
		Method:         method,
		Resolution:     resPtr,
		RWork:          rwPtr,
		RFree:          rfPtr,
		DepositionDate: cleanDatePrefix(entry.RcsbAccessionInfo.DepositDate),
		ReleaseDate:    release,
		Citation:       citation,
		DOI:            entry.RcsbPrimaryCitation.DOI,
		LigandIDs:      entry.RcsbEntryInfo.NonpolymerCompIDs,
	}
	return summary, enrich, nil
}

// FetchCoordinates downloads mmCIF (preferred for large/modern entries) with automatic fallback to PDB format.
func (c *HTTPRCSBClient) FetchCoordinates(ctx context.Context, pdbID string) ([]byte, string, string, error) {
	cleanID := strings.ToUpper(strings.TrimSpace(pdbID))
	cifURL := fmt.Sprintf("%s/download/%s.cif", c.filesURL, url.PathEscape(cleanID))
	raw, _, err := c.http.Do(ctx, http.MethodGet, cifURL, "", nil)
	if err == nil && len(raw) > 0 {
		return raw, "mmcif", cifURL, nil
	}

	pdbURL := fmt.Sprintf("%s/download/%s.pdb", c.filesURL, url.PathEscape(cleanID))
	raw, _, err = c.http.Do(ctx, http.MethodGet, pdbURL, "", nil)
	if err != nil {
		return nil, "", cifURL, err
	}
	return raw, "pdb", pdbURL, nil
}

func cleanDatePrefix(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
