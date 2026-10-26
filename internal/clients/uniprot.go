package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
)

// HTTPUniProtClient implements UniProtClient against https://rest.uniprot.org.
type HTTPUniProtClient struct {
	http    *SafeHTTPClient
	baseURL string
}

// NewUniProtClient creates a UniProtKB REST API client.
func NewUniProtClient(httpClient *SafeHTTPClient, baseURL string) *HTTPUniProtClient {
	return &HTTPUniProtClient{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

type uniprotSearchResponse struct {
	Results []uniprotEntryJSON `json:"results"`
}

type uniprotEntryJSON struct {
	PrimaryAccession string `json:"primaryAccession"`
	UniProtkbID      string `json:"uniProtkbId"`
	EntryType        string `json:"entryType"`
	Organism         struct {
		ScientificName string   `json:"scientificName"`
		CommonName     string   `json:"commonName"`
		TaxonID        int      `json:"taxonId"`
		Lineage        []string `json:"lineage"`
	} `json:"organism"`
	ProteinDescription struct {
		RecommendedName struct {
			FullName struct {
				Value string `json:"value"`
			} `json:"fullName"`
		} `json:"recommendedName"`
		AlternativeNames []struct {
			FullName struct {
				Value string `json:"value"`
			} `json:"fullName"`
		} `json:"alternativeNames"`
		SubmissionNames []struct {
			FullName struct {
				Value string `json:"value"`
			} `json:"fullName"`
		} `json:"submissionNames"`
	} `json:"proteinDescription"`
	Genes []struct {
		GeneName struct {
			Value string `json:"value"`
		} `json:"geneName"`
	} `json:"genes"`
	Comments []struct {
		CommentType string `json:"commentType"`
		Texts       []struct {
			Value string `json:"value"`
		} `json:"texts"`
		Reaction *struct {
			Name string `json:"name"`
		} `json:"reaction"`
		SubcellularLocations []struct {
			Location struct {
				Value string `json:"value"`
			} `json:"location"`
		} `json:"subcellularLocations"`
		Disease *struct {
			DiseaseID   string `json:"diseaseId"`
			Acronym     string `json:"acronym"`
			Description string `json:"description"`
			DiseaseCrossReference *struct {
				ID string `json:"id"`
			} `json:"diseaseCrossReference"`
		} `json:"disease"`
		Note *struct {
			Texts []struct {
				Value string `json:"value"`
			} `json:"texts"`
		} `json:"note"`
	} `json:"comments"`
	Features []struct {
		Type        string `json:"type"`
		Description string `json:"description"`
		Ligand      *struct {
			Name string `json:"name"`
		} `json:"ligand"`
		Location struct {
			Start struct {
				Value int `json:"value"`
			} `json:"start"`
			End struct {
				Value int `json:"value"`
			} `json:"end"`
		} `json:"location"`
	} `json:"features"`
	CrossReferences []struct {
		Database   string `json:"database"`
		ID         string `json:"id"`
		Properties []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"properties"`
	} `json:"uniProtKBCrossReferences"`
	Sequence struct {
		Value     string `json:"value"`
		Length    int    `json:"length"`
		MolWeight int    `json:"molWeight"`
	} `json:"sequence"`
}

// GetProtein fetches and parses a full UniProtKB record by accession.
func (c *HTTPUniProtClient) GetProtein(ctx context.Context, accession string) (*bio.ProteinRecord, error) {
	acc := strings.ToUpper(strings.TrimSpace(accession))
	endpoint := fmt.Sprintf("%s/uniprotkb/%s.json", c.baseURL, url.PathEscape(acc))
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	var entry uniprotEntryJSON
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, fmt.Errorf("parse UniProt entry %s: %w", acc, err)
	}
	rec := mapUniProtEntry(entry, endpoint)
	return &rec, nil
}

// SearchProteins executes a UniProtKB query prioritizing reviewed Swiss-Prot entries.
func (c *HTTPUniProtClient) SearchProteins(ctx context.Context, query string, limit int) ([]bio.SearchHit, error) {
	if limit <= 0 || limit > 25 {
		limit = 12
	}
	endpoint := fmt.Sprintf("%s/uniprotkb/search?query=%s&format=json&size=%d",
		c.baseURL, url.QueryEscape(strings.TrimSpace(query)), limit)
	raw, _, err := c.http.Do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	var resp uniprotSearchResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse UniProt search: %w", err)
	}

	hits := make([]bio.SearchHit, 0, len(resp.Results))
	for _, item := range resp.Results {
		rec := mapUniProtEntry(item, endpoint)
		var structs []string
		for _, xr := range rec.PDBCrossRefs {
			structs = append(structs, xr.ID)
			if len(structs) >= 4 {
				break
			}
		}
		gene := ""
		if len(rec.GeneNames) > 0 {
			gene = rec.GeneNames[0]
		}
		hits = append(hits, bio.SearchHit{
			Accession:        rec.Accession,
			EntryName:        rec.EntryName,
			Title:            rec.RecommendedName,
			GeneName:         gene,
			Organism:         rec.OrganismName,
			Reviewed:         rec.Reviewed,
			Length:           rec.Length,
			StructureIDs:     structs,
			AlphaFoldID:      rec.AlphaFoldID,
			MatchedBy:        "UniProtKB Search",
			HighlightSnippet: truncateText(rec.FunctionSummary, 180),
		})
	}
	return hits, nil
}

func mapUniProtEntry(e uniprotEntryJSON, endpoint string) bio.ProteinRecord {
	reviewed := strings.Contains(strings.ToLower(e.EntryType), "reviewed") &&
		!strings.Contains(strings.ToLower(e.EntryType), "unreviewed")

	name := e.ProteinDescription.RecommendedName.FullName.Value
	if name == "" && len(e.ProteinDescription.SubmissionNames) > 0 {
		name = e.ProteinDescription.SubmissionNames[0].FullName.Value
	}
	if name == "" {
		name = e.UniProtkbID
	}

	rec := bio.ProteinRecord{
		Accession:       e.PrimaryAccession,
		EntryName:       e.UniProtkbID,
		RecommendedName: name,
		Reviewed:        reviewed,
		OrganismName:    e.Organism.ScientificName,
		OrganismCommon:  e.Organism.CommonName,
		TaxonID:         e.Organism.TaxonID,
		Lineage:         e.Organism.Lineage,
		Sequence:        e.Sequence.Value,
		Length:          e.Sequence.Length,
		MassDa:          e.Sequence.MolWeight,
		AlphaFoldID:     fmt.Sprintf("AF-%s-F1", e.PrimaryAccession),
		Provenance: bio.ProvenanceRecord{
			SourceName: "UniProtKB",
			RecordID:   e.PrimaryAccession,
			RecordStatus: func() string {
				if reviewed {
					return "Reviewed (Swiss-Prot)"
				}
				return "Unreviewed (TrEMBL)"
			}(),
			EndpointURL:  endpoint,
			RetrievedAt:  time.Now().UTC(),
			DeliveryMode: bio.DeliveryLive,
			Citation:     "The UniProt Consortium (2025). UniProt: the Universal Protein Knowledgebase. Nucleic Acids Research.",
			License:      "CC BY 4.0",
		},
	}

	for _, alt := range e.ProteinDescription.AlternativeNames {
		if alt.FullName.Value != "" {
			rec.AlternativeNames = append(rec.AlternativeNames, alt.FullName.Value)
		}
	}
	for _, g := range e.Genes {
		if g.GeneName.Value != "" {
			rec.GeneNames = append(rec.GeneNames, g.GeneName.Value)
		}
	}

	for _, c := range e.Comments {
		switch c.CommentType {
		case "FUNCTION":
			if len(c.Texts) > 0 && rec.FunctionSummary == "" {
				rec.FunctionSummary = c.Texts[0].Value
			}
		case "CATALYTIC ACTIVITY":
			if c.Reaction != nil && c.Reaction.Name != "" {
				rec.CatalyticActivities = append(rec.CatalyticActivities, c.Reaction.Name)
			}
		case "SUBUNIT":
			if len(c.Texts) > 0 && rec.SubunitNote == "" {
				rec.SubunitNote = c.Texts[0].Value
			}
		case "SUBCELLULAR LOCATION":
			for _, sl := range c.SubcellularLocations {
				if sl.Location.Value != "" {
					rec.SubcellularLocations = append(rec.SubcellularLocations, sl.Location.Value)
				}
			}
		case "DISEASE":
			if c.Disease != nil && c.Disease.DiseaseID != "" {
				mim := ""
				if c.Disease.DiseaseCrossReference != nil {
					mim = c.Disease.DiseaseCrossReference.ID
				}
				rec.Diseases = append(rec.Diseases, bio.DiseaseAnnotation{
					Name:        c.Disease.DiseaseID,
					Acronym:     c.Disease.Acronym,
					Description: c.Disease.Description,
					MIM:         mim,
				})
			}
		}
	}

	for _, f := range e.Features {
		ligName := ""
		if f.Ligand != nil {
			ligName = f.Ligand.Name
		}
		span := bio.FeatureSpan{
			Type:        f.Type,
			Description: f.Description,
			Start:       f.Location.Start.Value,
			End:         f.Location.End.Value,
			Ligand:      ligName,
		}
		if f.Type == "Domain" || f.Type == "Region" || f.Type == "Chain" || f.Type == "Signal" || f.Type == "Propeptide" {
			rec.Domains = append(rec.Domains, span)
		}
		if f.Type == "Active site" || f.Type == "Binding site" || f.Type == "Metal binding" || f.Type == "Disulfide bond" || f.Type == "Site" || f.Type == "Mutagenesis" {
			rec.Features = append(rec.Features, span)
		}
	}

	for _, xr := range e.CrossReferences {
		props := make(map[string]string, len(xr.Properties))
		for _, p := range xr.Properties {
			props[p.Key] = p.Value
		}
		switch xr.Database {
		case "PDB":
			var resPtr *float64
			if rStr := strings.TrimSuffix(strings.TrimSpace(props["Resolution"]), " A"); rStr != "" && rStr != "-" {
				if v, err := strconv.ParseFloat(rStr, 64); err == nil {
					resPtr = &v
				}
			}
			rec.PDBCrossRefs = append(rec.PDBCrossRefs, bio.PDBCrossRef{
				ID:         strings.ToUpper(xr.ID),
				Method:     props["Method"],
				Resolution: resPtr,
				Chains:     props["Chains"],
			})
		case "AlphaFoldDB":
			rec.AlphaFoldID = fmt.Sprintf("AF-%s-F1", xr.ID)
		case "GO", "Pfam", "InterPro":
			label := firstStr(props["GoTerm"], props["EntryName"], xr.ID)
			if len(rec.Terms) < 16 {
				rec.Terms = append(rec.Terms, bio.TermRef{
					Database: xr.Database,
					ID:       xr.ID,
					Name:     label,
				})
			}
		}
	}

	return rec
}

func firstStr(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max]) + "…"
}
