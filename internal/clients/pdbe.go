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

// HTTPPDBeClient implements PDBeClient against EMBL-EBI PDBe and SIFTS REST endpoints.
type HTTPPDBeClient struct {
	http    *SafeHTTPClient
	baseURL string
}

// NewPDBeClient creates a production EMBL-EBI PDBe API client.
func NewPDBeClient(httpClient *SafeHTTPClient, baseURL string) *HTTPPDBeClient {
	return &HTTPPDBeClient{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

type pdbePercentilesItem struct {
	Clashscore        *struct{ Absolute *float64 `json:"percentiles"` } `json:"clashscore"`
	RamaOutliers      *struct{ RawValue *float64 `json:"rawvalue"` }    `json:"percent-rama-outliers"`
	SidechainOutliers *struct{ RawValue *float64 `json:"rawvalue"` }    `json:"percent-sidechain-outliers"`
}

// GetValidationMetrics fetches experimental geometry validation percentiles from PDBe.
func (c *HTTPPDBeClient) GetValidationMetrics(ctx context.Context, pdbID string) (*PDBeValidation, error) {
	lowerID := strings.ToLower(strings.TrimSpace(pdbID))
	endpoint := fmt.Sprintf("%s/api/validation/global-percentiles/entry/%s", c.baseURL, url.PathEscape(lowerID))
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	var root map[string]pdbePercentilesItem
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse PDBe validation %s: %w", lowerID, err)
	}
	item, ok := root[lowerID]
	if !ok {
		for _, v := range root {
			item = v
			ok = true
			break
		}
	}
	if !ok {
		return nil, ErrNotFound
	}
	out := &PDBeValidation{SourceURL: endpoint}
	if item.Clashscore != nil {
		out.ClashscorePercentile = item.Clashscore.Absolute
	}
	if item.RamaOutliers != nil {
		out.RamachandranOutliers = item.RamaOutliers.RawValue
	}
	if item.SidechainOutliers != nil {
		out.SidechainOutliers = item.SidechainOutliers.RawValue
	}
	return out, nil
}

type pdbeSIFTSEntry struct {
	UniProt map[string]struct {
		Mappings []struct {
			ChainID      string `json:"chain_id"`
			StructAsymID string `json:"struct_asym_id"`
			UnpStart     int    `json:"unp_start"`
			UnpEnd       int    `json:"unp_end"`
			Start        struct {
				ResidueNumber       int  `json:"residue_number"`
				AuthorResidueNumber *int `json:"author_residue_number"`
			} `json:"start"`
			End struct {
				ResidueNumber       int  `json:"residue_number"`
				AuthorResidueNumber *int `json:"author_residue_number"`
			} `json:"end"`
		} `json:"mappings"`
	} `json:"UniProt"`
}

// GetSIFTSMappings fetches residue-level PDB <-> UniProt coordinate intervals from EMBL-EBI SIFTS.
func (c *HTTPPDBeClient) GetSIFTSMappings(ctx context.Context, pdbID string) ([]bio.SIFTSRange, error) {
	lowerID := strings.ToLower(strings.TrimSpace(pdbID))
	endpoint := fmt.Sprintf("%s/api/mappings/uniprot/%s", c.baseURL, url.PathEscape(lowerID))
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	var root map[string]pdbeSIFTSEntry
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse SIFTS mapping %s: %w", lowerID, err)
	}
	entry, ok := root[lowerID]
	if !ok {
		for _, v := range root {
			entry = v
			ok = true
			break
		}
	}
	if !ok {
		return nil, nil
	}

	var out []bio.SIFTSRange
	for unpAcc, block := range entry.UniProt {
		for _, m := range block.Mappings {
			out = append(out, bio.SIFTSRange{
				ChainID:       m.ChainID,
				StructAsymID:  m.StructAsymID,
				UniProtAcc:    strings.ToUpper(unpAcc),
				UniProtStart:  m.UnpStart,
				UniProtEnd:    m.UnpEnd,
				LabelSeqStart: m.Start.ResidueNumber,
				LabelSeqEnd:   m.End.ResidueNumber,
				AuthSeqStart:  m.Start.AuthorResidueNumber,
				AuthSeqEnd:    m.End.AuthorResidueNumber,
			})
		}
	}
	return out, nil
}
