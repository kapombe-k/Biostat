package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/cache"
	"github.com/kapombe-k/Biostat/internal/clients"
	"github.com/kapombe-k/Biostat/internal/config"
	"github.com/kapombe-k/Biostat/internal/seed"
)

// WorkspaceBundle aggregates everything required to render the interactive molecular lab for a protein/structure pair.
type WorkspaceBundle struct {
	Protein             *bio.ProteinRecord     `json:"protein"`
	AvailableStructures []bio.StructureSummary `json:"available_structures"`
	ActiveStructure     *bio.StructureDetail   `json:"active_structure"`
	ActiveChain         *bio.ChainModel        `json:"active_chain"`
	DefaultColorScheme  string                 `json:"default_color_scheme"`
	Warnings            []string               `json:"warnings,omitempty"`
}

// RawCoordinatePayload stores raw structure file bytes alongside provenance metadata.
type RawCoordinatePayload struct {
	ID       string
	Kind     bio.StructureKind
	Format   string
	Data     []byte
	Endpoint string
	Mode     bio.DeliveryMode
}

// MVSResidueAnnotation supplies MolViewSpec / 3Dmol per-residue color and rich scientific tooltip metadata.
type MVSResidueAnnotation struct {
	AuthAsymID string `json:"auth_asym_id"`
	AuthSeqID  int    `json:"auth_seq_id"`
	InsCode    string `json:"pdbx_PDB_ins_code,omitempty"`
	Color      string `json:"color"`
	Tooltip    string `json:"tooltip"`
}

// ExplorerService orchestrates discovery, coordinate parsing, residue mapping, caching, and reproducible exports.
type ExplorerService struct {
	cfg       *config.Config
	uniprot   clients.UniProtClient
	rcsb      clients.RCSBClient
	alphafold clients.AlphaFoldClient
	pdbe      clients.PDBeClient
	blast     *BlastManager
	logger    *slog.Logger

	proteinCache *cache.Cache[string, *bio.ProteinRecord]
	coordCache   *cache.Cache[string, *RawCoordinatePayload]
	detailCache  *cache.Cache[string, *bio.StructureDetail]
	searchCache  *cache.Cache[string, []bio.SearchHit]

	proteinFlight *cache.FlightGroup[*bio.ProteinRecord]
	coordFlight   *cache.FlightGroup[*RawCoordinatePayload]
}

// NewExplorerService wires upstream clients, bounded TTL caches, and singleflight deduplication.
func NewExplorerService(
	cfg *config.Config,
	uniprot clients.UniProtClient,
	rcsb clients.RCSBClient,
	alphafold clients.AlphaFoldClient,
	pdbe clients.PDBeClient,
	blast *BlastManager,
	logger *slog.Logger,
) *ExplorerService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ExplorerService{
		cfg:           cfg,
		uniprot:       uniprot,
		rcsb:          rcsb,
		alphafold:     alphafold,
		pdbe:          pdbe,
		blast:         blast,
		logger:        logger,
		proteinCache:  cache.New[string, *bio.ProteinRecord](cfg.CacheMaxItems, cfg.CacheTTL),
		coordCache:    cache.New[string, *RawCoordinatePayload](cfg.CacheMaxItems/2+16, cfg.CacheTTL),
		detailCache:   cache.New[string, *bio.StructureDetail](cfg.CacheMaxItems, cfg.CacheTTL),
		searchCache:   cache.New[string, []bio.SearchHit](cfg.CacheMaxItems, cfg.CacheTTL/2),
		proteinFlight: cache.NewFlightGroup[*bio.ProteinRecord](),
		coordFlight:   cache.NewFlightGroup[*RawCoordinatePayload](),
	}
}

// CacheStats returns combined occupancy telemetry across bounded caches.
func (s *ExplorerService) CacheStats() map[string]cache.Stats {
	return map[string]cache.Stats{
		"proteins":    s.proteinCache.Stats(),
		"coordinates": s.coordCache.Stats(),
		"structures":  s.detailCache.Stats(),
		"searches":    s.searchCache.Stats(),
	}
}

// FeaturedProteins returns the curated showcase entries for the landing page.
func (s *ExplorerService) FeaturedProteins() []bio.SearchHit {
	return seed.SearchProteins("")
}

// Search resolves keywords, accessions, structure IDs, or pasted FASTA sequences into ranked hits.
func (s *ExplorerService) Search(ctx context.Context, rawQuery string) (bio.QueryKind, []bio.SearchHit, *bio.ParsedSequence, error) {
	q := strings.TrimSpace(rawQuery)
	if q == "" {
		return bio.QueryKeyword, s.FeaturedProteins(), nil, nil
	}

	kind, norm := bio.ClassifyQuery(q)
	if kind == bio.QueryFASTA {
		parsed, err := bio.ParseAndValidateSequence(q, s.cfg.MinSequenceLength, s.cfg.MaxSequenceLength)
		if err != nil {
			return bio.QueryFASTA, nil, nil, err
		}
		// Immediately provide instant sequence similarity preview against reference proteins + offer full NCBI BLAST!
		simHits := seed.LocalSimilaritySearch(parsed.Sequence)
		var out []bio.SearchHit
		for _, bh := range simHits {
			if p, ok := seed.GetProtein(bh.UniProtGuess); ok {
				var sids []string
				for _, xr := range p.PDBCrossRefs {
					sids = append(sids, xr.ID)
				}
				gene := ""
				if len(p.GeneNames) > 0 {
					gene = p.GeneNames[0]
				}
				out = append(out, bio.SearchHit{
					Accession:        p.Accession,
					EntryName:        p.EntryName,
					Title:            p.RecommendedName,
					GeneName:         gene,
					Organism:         p.OrganismName,
					Reviewed:         p.Reviewed,
					Length:           p.Length,
					StructureIDs:     sids,
					AlphaFoldID:      p.AlphaFoldID,
					MatchedBy:        fmt.Sprintf("Sequence Homology (%.1f%% id, E=%.1e)", bh.IdentityPct, bh.EValue),
					HighlightSnippet: p.FunctionSummary,
				})
			}
		}
		return bio.QueryFASTA, out, parsed, nil
	}

	cacheKey := strings.ToLower(string(kind) + ":" + norm)
	if cached, ok := s.searchCache.Get(cacheKey); ok {
		return kind, cached, nil, nil
	}

	var hits []bio.SearchHit
	switch kind {
	case bio.QueryUniProtAccession:
		if rec, err := s.GetProtein(ctx, norm); err == nil && rec != nil {
			hits = append(hits, proteinToHit(rec, "Exact UniProt Accession"))
		}
	case bio.QueryAlphaFoldID:
		acc := bio.ExtractUniProtFromAFID(norm)
		if acc != "" {
			if rec, err := s.GetProtein(ctx, acc); err == nil && rec != nil {
				h := proteinToHit(rec, "AlphaFold Structure ID ("+norm+")")
				h.AlphaFoldID = norm
				hits = append(hits, h)
			}
		}
	case bio.QueryPDBID:
		// Check seed index and live UniProt cross-reference search (`(xref:pdb-XXXX)`)
		seedMatches := seed.SearchProteins(norm)
		hits = append(hits, seedMatches...)
		if liveHits, err := s.uniprot.SearchProteins(ctx, fmt.Sprintf("(xref:pdb-%s)", norm), 6); err == nil {
			hits = mergeUniqueHits(hits, liveHits)
		}
		if len(hits) == 0 {
			// Direct structure-only entry fallback if PDB exists without UniProt linkage
			if sum, _, err := s.rcsb.GetEntrySummary(ctx, norm); err == nil && sum != nil {
				hits = append(hits, bio.SearchHit{
					Accession:        norm,
					Title:            sum.Title,
					Organism:         "PDB Structure Entry",
					Reviewed:         true,
					StructureIDs:     []string{norm},
					MatchedBy:        "Direct RCSB PDB Entry",
					HighlightSnippet: fmt.Sprintf("%s · %s", sum.SourceDB, sum.Method),
				})
			}
		}
	default:
		liveHits, err := s.uniprot.SearchProteins(ctx, norm, 12)
		if err == nil && len(liveHits) > 0 {
			hits = liveHits
		} else if s.cfg.OfflineFallback {
			hits = seed.SearchProteins(norm)
		}
	}

	if len(hits) == 0 && s.cfg.OfflineFallback {
		hits = seed.SearchProteins(norm)
	}
	if len(hits) > 0 {
		s.searchCache.Set(cacheKey, hits)
	}
	return kind, hits, nil, nil
}

// GetProtein retrieves normalized UniProt biological annotations with caching, deduplication, and seed fallback.
func (s *ExplorerService) GetProtein(ctx context.Context, accession string) (*bio.ProteinRecord, error) {
	acc := strings.ToUpper(strings.TrimSpace(accession))
	if cached, ok := s.proteinCache.Get(acc); ok {
		return cached, nil
	}

	rec, err := s.proteinFlight.Do(ctx, acc, func() (*bio.ProteinRecord, error) {
		if v, ok := s.proteinCache.Get(acc); ok {
			return v, nil
		}
		liveRec, liveErr := s.uniprot.GetProtein(ctx, acc)
		if liveErr == nil && liveRec != nil {
			s.proteinCache.Set(acc, liveRec)
			return liveRec, nil
		}
		if s.cfg.OfflineFallback {
			if seedRec, ok := seed.GetProtein(acc); ok {
				s.proteinCache.Set(acc, seedRec)
				return seedRec, nil
			}
		}
		return nil, fmt.Errorf("protein %q not found: %w", acc, liveErr)
	})
	return rec, err
}

// ResolveWorkspace resolves any protein accession, structure ID, or keyword into a complete interactive workspace bundle.
func (s *ExplorerService) ResolveWorkspace(ctx context.Context, rawTarget, requestedStructID, requestedChainID string) (*WorkspaceBundle, error) {
	target := strings.TrimSpace(rawTarget)
	if target == "" {
		target = "P69905"
	}

	kind, norm := bio.ClassifyQuery(target)
	var protein *bio.ProteinRecord
	var forcedStruct string

	switch kind {
	case bio.QueryAlphaFoldID:
		forcedStruct = norm
		acc := bio.ExtractUniProtFromAFID(norm)
		if acc != "" {
			protein, _ = s.GetProtein(ctx, acc)
		}
	case bio.QueryPDBID:
		forcedStruct = norm
		if m, ok := seed.GetStructureMeta(norm); ok && m.PrimaryUniProt != "" {
			protein, _ = s.GetProtein(ctx, m.PrimaryUniProt)
		}
		if protein == nil {
			if hits, err := s.uniprot.SearchProteins(ctx, fmt.Sprintf("(xref:pdb-%s)", norm), 1); err == nil && len(hits) > 0 {
				protein, _ = s.GetProtein(ctx, hits[0].Accession)
			}
		}
	case bio.QueryUniProtAccession:
		var err error
		protein, err = s.GetProtein(ctx, norm)
		if err != nil {
			return nil, err
		}
	case bio.QueryFASTA:
		_, hits, parsed, err := s.Search(ctx, target)
		if err != nil {
			return nil, err
		}
		if len(hits) > 0 {
			protein, _ = s.GetProtein(ctx, hits[0].Accession)
		} else if parsed != nil {
			protein = &bio.ProteinRecord{
				Accession:       "USER_SEQ",
				EntryName:       parsed.Header,
				RecommendedName: parsed.Header,
				OrganismName:    "User-submitted sequence",
				Sequence:        parsed.Sequence,
				Length:          parsed.Length,
				FunctionSummary: "Custom protein sequence supplied directly via FASTA input. Run a BLAST similarity search on the Sequence Similarity tab to identify homologous experimental structures and functional annotations.",
				Provenance: bio.ProvenanceRecord{
					SourceName:   "User Direct FASTA Input",
					RecordID:     parsed.Header,
					RecordStatus: "Unannotated Query Sequence",
					EndpointURL:  "bioscope://user-input/fasta",
					RetrievedAt:  time.Now().UTC(),
					DeliveryMode: bio.DeliveryLive,
				},
			}
		}
	default:
		_, hits, _, _ := s.Search(ctx, norm)
		if len(hits) > 0 {
			protein, _ = s.GetProtein(ctx, hits[0].Accession)
		}
	}

	if requestedStructID != "" {
		forcedStruct = normalizeStructureID(requestedStructID)
	}

	available := s.discoverStructures(ctx, protein, forcedStruct)

	activeStructID := forcedStruct
	if activeStructID == "" && len(available) > 0 {
		activeStructID = available[0].ID
	}

	var activeDetail *bio.StructureDetail
	var warnings []string
	if activeStructID != "" {
		det, err := s.GetStructureDetail(ctx, activeStructID, protein)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Could not load 3D structure %s: %v", activeStructID, err))
			// Attempt fallback to next available structure if first failed
			for _, cand := range available {
				if cand.ID != activeStructID {
					if det2, err2 := s.GetStructureDetail(ctx, cand.ID, protein); err2 == nil {
						activeDetail = det2
						break
					}
				}
			}
		} else {
			activeDetail = det
		}
	}

	// Synthesize minimal placeholder protein record if user opened an unlinked PDB ID directly
	if protein == nil && activeDetail != nil {
		protein = &bio.ProteinRecord{
			Accession:       activeDetail.ID,
			EntryName:       activeDetail.ID,
			RecommendedName: firstStrLocal(activeDetail.Title, "Structure Entry "+activeDetail.ID),
			OrganismName:    firstStrLocal(activeDetail.Organism, "Experimental Structure"),
			FunctionSummary: "Direct structure entry loaded from coordinates.",
			Provenance:      activeDetail.Provenance[0],
		}
	}
	if protein == nil {
		return nil, fmt.Errorf("no matching protein or 3D structure found for %q", rawTarget)
	}

	var activeChain *bio.ChainModel
	if activeDetail != nil && len(activeDetail.Chains) > 0 {
		activeChain = selectBestChain(activeDetail, protein, requestedChainID)
	}

	defaultColor := "secondary"
	if activeDetail != nil {
		if activeDetail.Kind == bio.StructurePredicted {
			defaultColor = "plddt"
		} else if len(activeDetail.Chains) > 1 {
			defaultColor = "chain"
		}
	}

	return &WorkspaceBundle{
		Protein:             protein,
		AvailableStructures: available,
		ActiveStructure:     activeDetail,
		ActiveChain:         activeChain,
		DefaultColorScheme:  defaultColor,
		Warnings:            warnings,
	}, nil
}

func selectBestChain(detail *bio.StructureDetail, protein *bio.ProteinRecord, requested string) *bio.ChainModel {
	if requested != "" {
		for i := range detail.Chains {
			if strings.EqualFold(detail.Chains[i].AuthAsymID, requested) ||
				strings.EqualFold(detail.Chains[i].LabelAsymID, requested) {
				return &detail.Chains[i]
			}
		}
	}
	if protein != nil {
		for i := range detail.Chains {
			if strings.EqualFold(detail.Chains[i].UniProtAccession, protein.Accession) {
				return &detail.Chains[i]
			}
		}
	}
	return &detail.Chains[0]
}

// GetStructureCoordinates returns raw PDB or mmCIF coordinate bytes for same-origin viewer rendering.
func (s *ExplorerService) GetStructureCoordinates(ctx context.Context, structureID string) (*RawCoordinatePayload, error) {
	sid := normalizeStructureID(structureID)
	if cached, ok := s.coordCache.Get(sid); ok {
		return cached, nil
	}

	return s.coordFlight.Do(ctx, sid, func() (*RawCoordinatePayload, error) {
		if v, ok := s.coordCache.Get(sid); ok {
			return v, nil
		}
		isAF := strings.HasPrefix(sid, "AF-")
		if isAF {
			acc := bio.ExtractUniProtFromAFID(sid)
			if acc != "" {
				if pred, err := s.alphafold.GetPrediction(ctx, acc); err == nil && pred != nil {
					if raw, format, ep, err2 := s.alphafold.FetchCoordinates(ctx, pred); err2 == nil && len(raw) > 0 {
						p := &RawCoordinatePayload{
							ID: sid, Kind: bio.StructurePredicted, Format: format, Data: raw, Endpoint: ep, Mode: bio.DeliveryLive,
						}
						s.coordCache.Set(sid, p)
						return p, nil
					}
				}
			}
		} else {
			if raw, format, ep, err := s.rcsb.FetchCoordinates(ctx, sid); err == nil && len(raw) > 0 {
				p := &RawCoordinatePayload{
					ID: sid, Kind: bio.StructureExperimental, Format: format, Data: raw, Endpoint: ep, Mode: bio.DeliveryLive,
				}
				s.coordCache.Set(sid, p)
				return p, nil
			}
		}

		if s.cfg.OfflineFallback {
			if raw, format, meta, err := seed.ReadStructureCoordinates(sid); err == nil {
				ep := fmt.Sprintf("https://files.rcsb.org/download/%s.%s", sid, map[string]string{"mmcif": "cif", "pdb": "pdb"}[format])
				if meta.Kind == bio.StructurePredicted {
					ep = fmt.Sprintf("https://alphafold.ebi.ac.uk/files/%s", sid)
				}
				p := &RawCoordinatePayload{
					ID: sid, Kind: meta.Kind, Format: format, Data: raw, Endpoint: ep, Mode: bio.DeliverySeedFallback,
				}
				s.coordCache.Set(sid, p)
				return p, nil
			}
		}

		return nil, fmt.Errorf("coordinates unavailable for structure %s", sid)
	})
}

// GetStructureDetail parses coordinates, enriches experimental/AlphaFold quality metrics, and maps chains to protein.
func (s *ExplorerService) GetStructureDetail(ctx context.Context, structureID string, protein *bio.ProteinRecord) (*bio.StructureDetail, error) {
	sid := normalizeStructureID(structureID)
	accKey := "NONE"
	if protein != nil {
		accKey = protein.Accession
	}
	cacheKey := sid + "::" + accKey
	if cached, ok := s.detailCache.Get(cacheKey); ok {
		return cached, nil
	}

	payload, err := s.GetStructureCoordinates(ctx, sid)
	if err != nil {
		return nil, err
	}

	detail, err := bio.ParseCoordinates(sid, payload.Kind, payload.Data)
	if err != nil {
		return nil, fmt.Errorf("parse coordinates %s: %w", sid, err)
	}
	detail.CoordinateURL = fmt.Sprintf("/api/structures/%s/coordinates", url.PathEscape(sid))

	now := time.Now().UTC()
	var sifts []bio.SIFTSRange
	var partnerNotes map[string]string

	if seedMeta, hasSeed := seed.GetStructureMeta(sid); hasSeed {
		partnerNotes = seedMeta.SecondaryUniProt
		if detail.Title == "" {
			detail.Title = seedMeta.Title
		}
		if seedMeta.Method != "" {
			detail.Method = seedMeta.Method
		}
		if detail.Quality.ResolutionAngstrom == nil && seedMeta.Resolution != nil {
			detail.Quality.ResolutionAngstrom = seedMeta.Resolution
		}
		if detail.Quality.RWork == nil && seedMeta.RWork != nil {
			detail.Quality.RWork = seedMeta.RWork
		}
		if detail.Quality.RFree == nil && seedMeta.RFree != nil {
			detail.Quality.RFree = seedMeta.RFree
		}
		detail.Quality.ClashscorePercentile = seedMeta.ClashscorePercentile
		detail.Quality.RamachandranOutliers = seedMeta.RamachandranOutliers
		detail.Quality.SidechainOutliers = seedMeta.SidechainOutliers
		detail.ReleaseDate = seedMeta.ReleaseDate
		detail.Organism = seedMeta.Organism
		detail.Citation = seedMeta.Citation
		detail.DOI = seedMeta.DOI
	}

	// Parallel enrichment against live RCSB + PDBe APIs when experimental
	if payload.Kind == bio.StructureExperimental && payload.Mode == bio.DeliveryLive {
		var wg sync.WaitGroup
		var enrich *clients.EntryEnrichment
		var val *clients.PDBeValidation

		wg.Add(3)
		go func() {
			defer wg.Done()
			_, enrich, _ = s.rcsb.GetEntrySummary(ctx, sid)
		}()
		go func() {
			defer wg.Done()
			val, _ = s.pdbe.GetValidationMetrics(ctx, sid)
		}()
		go func() {
			defer wg.Done()
			sifts, _ = s.pdbe.GetSIFTSMappings(ctx, sid)
		}()
		wg.Wait()

		if enrich != nil {
			if enrich.Title != "" {
				detail.Title = enrich.Title
			}
			if enrich.Method != "" {
				detail.Method = enrich.Method
			}
			if enrich.Resolution != nil {
				detail.Quality.ResolutionAngstrom = enrich.Resolution
			}
			if enrich.RWork != nil {
				detail.Quality.RWork = enrich.RWork
			}
			if enrich.RFree != nil {
				detail.Quality.RFree = enrich.RFree
			}
			detail.DepositionDate = enrich.DepositionDate
			detail.ReleaseDate = enrich.ReleaseDate
			detail.Citation = enrich.Citation
			detail.DOI = enrich.DOI
		}
		if val != nil {
			detail.Quality.ClashscorePercentile = val.ClashscorePercentile
			detail.Quality.RamachandranOutliers = val.RamachandranOutliers
			detail.Quality.SidechainOutliers = val.SidechainOutliers
			detail.Provenance = append(detail.Provenance, bio.ProvenanceRecord{
				SourceName:   "EMBL-EBI PDBe Validation & SIFTS",
				RecordID:     sid,
				RecordStatus: "Experimental Structure Validation",
				EndpointURL:  val.SourceURL,
				RetrievedAt:  now,
				DeliveryMode: bio.DeliveryLive,
				Citation:     "Mir, S. et al. (2018) PDBe: towards reusable data delivery infrastructure at protein data bank in Europe. Nucleic Acids Research.",
				License:      "CC0 1.0 Public Domain",
			})
		}
	}

	primaryProv := bio.ProvenanceRecord{
		SourceName: detail.SourceDB,
		RecordID:   sid,
		RecordStatus: func() string {
			if detail.Kind == bio.StructurePredicted {
				return "Computational Structure Prediction (AlphaFold)"
			}
			return "Experimental Structure Determination (" + detail.Method + ")"
		}(),
		EndpointURL:  payload.Endpoint,
		RetrievedAt:  now,
		DeliveryMode: payload.Mode,
		Citation:     detail.Citation,
		License: func() string {
			if detail.Kind == bio.StructurePredicted {
				return "CC BY 4.0 (EMBL-EBI AlphaFold DB)"
			}
			return "CC0 1.0 Universal (wwPDB Archive)"
		}(),
		SHA256: detail.CoordinateSHA,
	}
	detail.Provenance = append([]bio.ProvenanceRecord{primaryProv}, detail.Provenance...)

	if protein != nil {
		MapStructureToProtein(detail, protein, sifts, partnerNotes)
	}

	s.detailCache.Set(cacheKey, detail)
	return detail, nil
}

func (s *ExplorerService) discoverStructures(ctx context.Context, protein *bio.ProteinRecord, ensureStructID string) []bio.StructureSummary {
	seen := make(map[string]bool)
	var out []bio.StructureSummary

	addSummary := func(sum bio.StructureSummary) {
		key := strings.ToUpper(sum.ID)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, sum)
	}

	if protein != nil {
		// Seed summaries carry rich curated descriptions when matched
		for _, sm := range seed.ListStructuresForProtein(protein.Accession) {
			addSummary(sm)
		}

		// Live RCSB + UniProt PDB cross-references (cap enrichment calls to keep response snappy)
		limit := 6
		for i, xr := range protein.PDBCrossRefs {
			if i >= limit {
				break
			}
			if seen[xr.ID] {
				continue
			}
			if sum, _, err := s.rcsb.GetEntrySummary(ctx, xr.ID); err == nil && sum != nil {
				sum.ChainsSummary = xr.Chains
				addSummary(*sum)
			} else {
				addSummary(bio.StructureSummary{
					ID:                 xr.ID,
					Kind:               bio.StructureExperimental,
					SourceDB:           "RCSB PDB",
					Title:              fmt.Sprintf("%s Experimental Structure (%s)", protein.RecommendedName, xr.ID),
					Method:             firstStrLocal(xr.Method, "Experimental"),
					ResolutionAngstrom: xr.Resolution,
					ChainsSummary:      xr.Chains,
				})
			}
		}

		// Check AlphaFold DB prediction if not already listed
		if protein.AlphaFoldID != "" && !seen[strings.ToUpper(protein.AlphaFoldID)] {
			if pred, err := s.alphafold.GetPrediction(ctx, protein.Accession); err == nil && pred != nil {
				plddt := math.Round(pred.MeanPLDDT*10) / 10
				addSummary(bio.StructureSummary{
					ID:            pred.EntryID,
					Kind:          bio.StructurePredicted,
					SourceDB:      "AlphaFold DB",
					Title:         fmt.Sprintf("AlphaFold Monomer v%d Prediction for %s", pred.ModelVersion, protein.RecommendedName),
					Method:        "ALPHAFOLD MONOMER PREDICTION",
					MeanPLDDT:     &plddt,
					CoverageRange: fmt.Sprintf("UniProt 1–%d (Full length)", protein.Length),
				})
			}
		}
	}

	if ensureStructID != "" && !seen[ensureStructID] {
		if m, ok := seed.GetStructureMeta(ensureStructID); ok {
			addSummary(bio.StructureSummary{
				ID:                 m.ID,
				Kind:               m.Kind,
				SourceDB:           m.SourceDB,
				Title:              m.Title,
				Method:             m.Method,
				ResolutionAngstrom: m.Resolution,
				MeanPLDDT:          m.MeanPLDDT,
				ChainsSummary:      m.ChainsSummary,
				CoverageRange:      m.CoverageRange,
				LigandIDs:          m.LigandIDs,
				ReleaseDate:        m.ReleaseDate,
			})
		} else if sum, _, err := s.rcsb.GetEntrySummary(ctx, ensureStructID); err == nil && sum != nil {
			addSummary(*sum)
		}
	}

	return out
}

var chainPalette = []string{
	"#38BDF8", // Sky Blue
	"#34D399", // Emerald
	"#F472B6", // Pink
	"#FBBF24", // Amber
	"#A78BFA", // Violet
	"#FB923C", // Orange
}

var hydroScale = map[string]bool{
	"I": true, "V": true, "L": true, "F": true, "C": true, "M": true, "A": true, "W": true, "Y": true,
}

// BuildMVSAnnotations computes exact per-residue hex colors and scientific hover tooltips for any coloring mode.
func BuildMVSAnnotations(detail *bio.StructureDetail, scheme string) []MVSResidueAnnotation {
	if detail == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(scheme))
	if mode == "" {
		if detail.Kind == bio.StructurePredicted {
			mode = "plddt"
		} else if len(detail.Chains) > 1 {
			mode = "chain"
		} else {
			mode = "secondary"
		}
	}

	minB, maxB := math.MaxFloat64, -math.MaxFloat64
	for _, ch := range detail.Chains {
		for _, r := range ch.Residues {
			if r.Observed {
				if r.MetricValue < minB {
					minB = r.MetricValue
				}
				if r.MetricValue > maxB {
					maxB = r.MetricValue
				}
			}
		}
	}
	if maxB <= minB {
		maxB = minB + 1.0
	}

	var out []MVSResidueAnnotation
	for cIdx, ch := range detail.Chains {
		chColor := chainPalette[cIdx%len(chainPalette)]
		for _, r := range ch.Residues {
			if !r.Observed {
				continue
			}
			var hexColor string
			switch mode {
			case "plddt":
				switch bio.PLDDTBand(r.MetricValue) {
				case "very-high":
					hexColor = "#0053D6"
				case "confident":
					hexColor = "#65CBF3"
				case "low":
					hexColor = "#FFDB13"
				default:
					hexColor = "#FF7D45"
				}
			case "chain":
				hexColor = chColor
			case "bfactor":
				t := (r.MetricValue - minB) / (maxB - minB)
				hexColor = interpolateBlueWhiteRed(t)
			case "hydrophobicity":
				if hydroScale[r.OneLetter] {
					hexColor = "#F59E0B"
				} else {
					hexColor = "#06B6D4"
				}
			default: // "secondary"
				switch r.Secondary {
				case "helix":
					hexColor = "#F43F5E"
				case "sheet":
					hexColor = "#F59E0B"
				default:
					hexColor = "#94A3B8"
				}
			}

			tipParts := []string{
				fmt.Sprintf("%s %s (Chain %s)", r.CompID, r.AuthLabel, r.ChainID),
			}
			if r.UniProtPos != nil && ch.UniProtAccession != "" {
				tipParts = append(tipParts, fmt.Sprintf("%s %s%d", ch.UniProtAccession, r.UniProtAA, *r.UniProtPos))
			}
			tipParts = append(tipParts, strings.ToUpper(r.Secondary[:1])+r.Secondary[1:])
			if detail.Kind == bio.StructurePredicted {
				tipParts = append(tipParts, fmt.Sprintf("pLDDT %.1f", r.MetricValue))
			} else {
				tipParts = append(tipParts, fmt.Sprintf("B-factor %.1f Å²", r.MetricValue))
			}
			if len(r.FeatureLabels) > 0 {
				tipParts = append(tipParts, r.FeatureLabels[0])
			}

			out = append(out, MVSResidueAnnotation{
				AuthAsymID: r.ChainID,
				AuthSeqID:  r.AuthSeqID,
				InsCode:    r.InsCode,
				Color:      hexColor,
				Tooltip:    strings.Join(tipParts, " · "),
			})
		}
	}
	return out
}

// BuildShareURL encodes an ExplorationState into a clean, reproducible query permalink.
func BuildShareURL(baseOrigin string, st bio.ExplorationState) string {
	v := url.Values{}
	if st.Accession != "" {
		v.Set("q", st.Accession)
	} else if st.Query != "" {
		v.Set("q", st.Query)
	}
	if st.StructureID != "" {
		v.Set("struct", st.StructureID)
	}
	if st.ChainID != "" {
		v.Set("chain", st.ChainID)
	}
	if st.Representation != "" {
		v.Set("rep", st.Representation)
	}
	if st.ColorScheme != "" {
		v.Set("color", st.ColorScheme)
	}
	if st.FocusResidues != "" {
		v.Set("focus", st.FocusResidues)
	}
	if st.ViewerEngine != "" {
		v.Set("engine", st.ViewerEngine)
	}
	path := "/explore?" + v.Encode()
	if baseOrigin == "" {
		return path
	}
	return strings.TrimRight(baseOrigin, "/") + path
}

// BuildExport constructs a complete reproducible JSON exploration archive with provenance & disclaimers.
func (s *ExplorerService) BuildExport(bundle *WorkspaceBundle, st bio.ExplorationState, baseOrigin string) *bio.ExplorationExport {
	var provs []bio.ProvenanceRecord
	var notes []string

	if bundle.Protein != nil {
		provs = append(provs, bundle.Protein.Provenance)
	}
	if bundle.ActiveStructure != nil {
		provs = append(provs, bundle.ActiveStructure.Provenance...)
		notes = append(notes, bundle.ActiveStructure.Quality.InterpretationNote)
	}
	if bundle.ActiveChain != nil {
		notes = append(notes, bundle.ActiveChain.MappingNotes...)
	}
	notes = append(notes,
		"BioScope Educational Disclaimer: Structural visualizations and annotations are provided for research exploration and molecular biology instruction, not clinical diagnosis or therapeutic decision-making.")

	return &bio.ExplorationExport{
		Application:     "BioScope: Interactive Molecular Learning Lab",
		SchemaVersion:   "1.0.0",
		ExportedAt:      time.Now().UTC(),
		ShareableURL:    BuildShareURL(baseOrigin, st),
		State:           st,
		Protein:         bundle.Protein,
		ActiveModel:     bundle.ActiveStructure,
		AvailableModels: bundle.AvailableStructures,
		Provenance:      provs,
		ScientificNotes: notes,
	}
}

func normalizeStructureID(id string) string {
	upper := strings.ToUpper(strings.TrimSpace(id))
	if kind, norm := bio.ClassifyQuery(upper); kind == bio.QueryAlphaFoldID || kind == bio.QueryPDBID {
		return norm
	}
	return upper
}

func proteinToHit(rec *bio.ProteinRecord, matchedBy string) bio.SearchHit {
	var sids []string
	for _, xr := range rec.PDBCrossRefs {
		sids = append(sids, xr.ID)
		if len(sids) >= 4 {
			break
		}
	}
	gene := ""
	if len(rec.GeneNames) > 0 {
		gene = rec.GeneNames[0]
	}
	return bio.SearchHit{
		Accession:        rec.Accession,
		EntryName:        rec.EntryName,
		Title:            rec.RecommendedName,
		GeneName:         gene,
		Organism:         rec.OrganismName,
		Reviewed:         rec.Reviewed,
		Length:           rec.Length,
		StructureIDs:     sids,
		AlphaFoldID:      rec.AlphaFoldID,
		MatchedBy:        matchedBy,
		HighlightSnippet: rec.FunctionSummary,
	}
}

func mergeUniqueHits(a, b []bio.SearchHit) []bio.SearchHit {
	seen := make(map[string]bool, len(a)+len(b))
	var out []bio.SearchHit
	for _, item := range append(a, b...) {
		if seen[item.Accession] {
			continue
		}
		seen[item.Accession] = true
		out = append(out, item)
	}
	return out
}

func interpolateBlueWhiteRed(t float64) string {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	var r, g, b int
	if t < 0.5 {
		u := t * 2.0
		r = int(37 + u*(241-37))
		g = int(99 + u*(245-99))
		b = int(235 + u*(249-235))
	} else {
		u := (t - 0.5) * 2.0
		r = int(241 + u*(239-241))
		g = int(245 + u*(68-245))
		b = int(249 + u*(68-249))
	}
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func firstStrLocal(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
