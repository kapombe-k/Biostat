package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HTTPAlphaFoldClient implements AlphaFoldClient against https://alphafold.ebi.ac.uk.
type HTTPAlphaFoldClient struct {
	http    *SafeHTTPClient
	baseURL string
}

// NewAlphaFoldClient creates a production AlphaFold DB API client.
func NewAlphaFoldClient(httpClient *SafeHTTPClient, baseURL string) *HTTPAlphaFoldClient {
	return &HTTPAlphaFoldClient{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

type afEntryJSON struct {
	EntryID               string  `json:"entryId"`
	UniProtAccession      string  `json:"uniprotAccession"`
	LatestVersion         int     `json:"latestVersion"`
	UniProtSequence       string  `json:"uniprotSequence"`
	GlobalMetricValue     float64 `json:"globalMetricValue"`
	FractionPlddtVeryHigh float64 `json:"fractionPlddtVeryHigh"`
	FractionPlddtConfident float64 `json:"fractionPlddtConfident"`
	FractionPlddtLow      float64 `json:"fractionPlddtLow"`
	FractionPlddtVeryLow  float64 `json:"fractionPlddtVeryLow"`
	CIFURL                string  `json:"cifUrl"`
	PDBURL                string  `json:"pdbUrl"`
	PAEImageURL           string  `json:"paeImageUrl"`
}

// GetPrediction queries AlphaFold DB for predicted models corresponding to a UniProt accession.
func (c *HTTPAlphaFoldClient) GetPrediction(ctx context.Context, uniprotAccession string) (*AlphaFoldPrediction, error) {
	acc := strings.ToUpper(strings.TrimSpace(uniprotAccession))
	endpoint := fmt.Sprintf("%s/api/prediction/%s", c.baseURL, url.PathEscape(acc))
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}

	var list []afEntryJSON
	if err := json.Unmarshal(raw, &list); err != nil {
		var single afEntryJSON
		if err2 := json.Unmarshal(raw, &single); err2 != nil {
			return nil, fmt.Errorf("parse AlphaFold prediction %s: %w", acc, err)
		}
		list = []afEntryJSON{single}
	}
	if len(list) == 0 || list[0].EntryID == "" {
		return nil, ErrNotFound
	}
	item := list[0]
	return &AlphaFoldPrediction{
		EntryID:        item.EntryID,
		UniProtAcc:     firstStr(item.UniProtAccession, acc),
		ModelVersion:   item.LatestVersion,
		Sequence:       item.UniProtSequence,
		MeanPLDDT:      item.GlobalMetricValue,
		FracVeryHigh:   item.FractionPlddtVeryHigh,
		FracConfident:  item.FractionPlddtConfident,
		FracLow:        item.FractionPlddtLow,
		FracVeryLow:    item.FractionPlddtVeryLow,
		CIFURL:         item.CIFURL,
		PDBURL:         item.PDBURL,
		PAEImageURL:    item.PAEImageURL,
		MetadataAPIURL: endpoint,
	}, nil
}

// FetchCoordinates downloads mmCIF (preferred) or PDB coordinates for a validated AlphaFold prediction.
func (c *HTTPAlphaFoldClient) FetchCoordinates(ctx context.Context, pred *AlphaFoldPrediction) ([]byte, string, string, error) {
	if pred == nil {
		return nil, "", "", ErrNotFound
	}
	if pred.CIFURL != "" {
		raw, _, err := c.http.Do(ctx, http.MethodGet, pred.CIFURL, "", nil)
		if err == nil && len(raw) > 0 {
			return raw, "mmcif", pred.CIFURL, nil
		}
	}
	if pred.PDBURL != "" {
		raw, _, err := c.http.Do(ctx, http.MethodGet, pred.PDBURL, "", nil)
		if err == nil && len(raw) > 0 {
			return raw, "pdb", pred.PDBURL, nil
		}
	}
	fallbackCIF := fmt.Sprintf("%s/files/%s-model_v4.cif", c.baseURL, url.PathEscape(pred.EntryID))
	raw, _, err := c.http.Do(ctx, http.MethodGet, fallbackCIF, "", nil)
	if err != nil {
		return nil, "", fallbackCIF, err
	}
	return raw, "mmcif", fallbackCIF, nil
}
