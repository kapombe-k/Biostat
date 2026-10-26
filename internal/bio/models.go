package bio

import "time"

// DeliveryMode distinguishes live upstream API responses from embedded classroom fallbacks.
type DeliveryMode string

const (
	DeliveryLive         DeliveryMode = "live"
	DeliverySeedFallback DeliveryMode = "curated-seed-fallback"
)

// StructureKind strictly separates experimental determinations from computational predictions.
type StructureKind string

const (
	StructureExperimental StructureKind = "experimental"
	StructurePredicted    StructureKind = "predicted"
)

// QueryKind classifies user input across accessions, PDB/AFDB identifiers, raw FASTA, or keywords.
type QueryKind string

const (
	QueryKeyword          QueryKind = "keyword"
	QueryUniProtAccession QueryKind = "uniprot_accession"
	QueryPDBID            QueryKind = "pdb_id"
	QueryAlphaFoldID      QueryKind = "alphafold_id"
	QueryFASTA            QueryKind = "fasta_sequence"
)

// ProvenanceRecord captures transparent scientific provenance for reproducibility.
type ProvenanceRecord struct {
	SourceName   string       `json:"source_name"`
	RecordID     string       `json:"record_id"`
	RecordStatus string       `json:"record_status"`
	Chain        string       `json:"chain,omitempty"`
	EndpointURL  string       `json:"endpoint_url"`
	RetrievedAt  time.Time    `json:"retrieved_at"`
	DeliveryMode DeliveryMode `json:"delivery_mode"`
	Citation     string       `json:"citation,omitempty"`
	License      string       `json:"license,omitempty"`
	SHA256       string       `json:"sha256,omitempty"`
}

// FeatureSpan represents a biological sequence region or active residue in UniProt coordinates.
type FeatureSpan struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Ligand      string `json:"ligand,omitempty"`
}

// DiseaseAnnotation summarizes clinical/phenotypic associations from UniProt.
type DiseaseAnnotation struct {
	Name        string `json:"name"`
	Acronym     string `json:"acronym,omitempty"`
	Description string `json:"description"`
	MIM         string `json:"mim,omitempty"`
}

// TermRef represents an external ontology or family cross-reference (GO, Pfam, InterPro).
type TermRef struct {
	Database string `json:"database"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

// PDBCrossRef represents a structure cross-reference linked from a protein record.
type PDBCrossRef struct {
	ID         string   `json:"id"`
	Method     string   `json:"method"`
	Resolution *float64 `json:"resolution,omitempty"`
	Chains     string   `json:"chains,omitempty"`
}

// ProteinRecord represents normalized biological annotations from UniProtKB.
type ProteinRecord struct {
	Accession            string              `json:"accession"`
	EntryName            string              `json:"entry_name"`
	RecommendedName      string              `json:"recommended_name"`
	AlternativeNames     []string            `json:"alternative_names,omitempty"`
	GeneNames            []string            `json:"gene_names,omitempty"`
	Reviewed             bool                `json:"reviewed"`
	OrganismName         string              `json:"organism_name"`
	OrganismCommon       string              `json:"organism_common,omitempty"`
	TaxonID              int                 `json:"taxon_id,omitempty"`
	Lineage              []string            `json:"lineage,omitempty"`
	Sequence             string              `json:"sequence"`
	Length               int                 `json:"length"`
	MassDa               int                 `json:"mass_da,omitempty"`
	FunctionSummary      string              `json:"function_summary"`
	CatalyticActivities  []string            `json:"catalytic_activities,omitempty"`
	SubunitNote          string              `json:"subunit_note,omitempty"`
	SubcellularLocations []string            `json:"subcellular_locations,omitempty"`
	Diseases             []DiseaseAnnotation `json:"diseases,omitempty"`
	Domains              []FeatureSpan       `json:"domains,omitempty"`
	Features             []FeatureSpan       `json:"features,omitempty"`
	Terms                []TermRef           `json:"terms,omitempty"`
	PDBCrossRefs         []PDBCrossRef       `json:"pdb_cross_refs,omitempty"`
	AlphaFoldID          string              `json:"alphafold_id,omitempty"`
	Provenance           ProvenanceRecord    `json:"provenance"`
}

// SearchHit represents a unified protein or structure search hit.
type SearchHit struct {
	Accession       string   `json:"accession"`
	EntryName       string   `json:"entry_name,omitempty"`
	Title           string   `json:"title"`
	GeneName        string   `json:"gene_name,omitempty"`
	Organism        string   `json:"organism"`
	Reviewed        bool     `json:"reviewed"`
	Length          int      `json:"length,omitempty"`
	StructureIDs    []string `json:"structure_ids,omitempty"`
	AlphaFoldID     string   `json:"alphafold_id,omitempty"`
	MatchedBy       string   `json:"matched_by"`
	HighlightSnippet string  `json:"highlight_snippet,omitempty"`
}

// StructureSummary summarizes an experimental or predicted 3D model available for inspection.
type StructureSummary struct {
	ID                 string        `json:"id"`
	Kind               StructureKind `json:"kind"`
	SourceDB           string        `json:"source_db"`
	Title              string        `json:"title"`
	Method             string        `json:"method"`
	ResolutionAngstrom *float64      `json:"resolution_angstrom,omitempty"`
	MeanPLDDT          *float64      `json:"mean_plddt,omitempty"`
	ChainsSummary      string        `json:"chains_summary,omitempty"`
	CoverageRange      string        `json:"coverage_range,omitempty"`
	LigandIDs          []string      `json:"ligand_ids,omitempty"`
	ReleaseDate        string        `json:"release_date,omitempty"`
}

// ResidueRef addresses a specific structural residue including author number and insertion code.
type ResidueRef struct {
	ChainID     string  `json:"chain_id"`
	AuthSeqID   int     `json:"auth_seq_id"`
	InsCode     string  `json:"ins_code,omitempty"`
	CompID      string  `json:"comp_id"`
	DistanceAng float64 `json:"distance_ang,omitempty"`
}

// AuthLabel formats a residue's author sequence number concatenated with optional insertion code (e.g. "184A").
func (r ResidueRef) AuthLabel() string {
	return FormatAuthResidue(r.AuthSeqID, r.InsCode)
}

// LigandInfo describes a bound non-polymer cofactor, substrate analog, drug, or metal ion.
type LigandInfo struct {
	CompID          string       `json:"comp_id"`
	Name            string       `json:"name"`
	Role            string       `json:"role,omitempty"`
	ChainID         string       `json:"chain_id"`
	AuthSeqID       int          `json:"auth_seq_id"`
	AtomCount       int          `json:"atom_count"`
	ContactResidues []ResidueRef `json:"contact_residues,omitempty"`
}

// QualityMetrics captures either experimental crystallographic/cryo-EM quality or AlphaFold pLDDT confidence.
type QualityMetrics struct {
	// Experimental quality indicators
	ResolutionAngstrom     *float64 `json:"resolution_angstrom,omitempty"`
	RWork                  *float64 `json:"r_work,omitempty"`
	RFree                  *float64 `json:"r_free,omitempty"`
	ClashscorePercentile   *float64 `json:"clashscore_percentile,omitempty"`
	RamachandranOutliers   *float64 `json:"ramachandran_outliers_pct,omitempty"`
	SidechainOutliers      *float64 `json:"sidechain_outliers_pct,omitempty"`
	ObservedResiduePct     float64  `json:"observed_residue_pct"`
	MeanBFactor            float64  `json:"mean_b_factor,omitempty"`

	// AlphaFold confidence indicators (strictly per-residue pLDDT, never crystallographic resolution)
	MeanPLDDT        float64 `json:"mean_plddt,omitempty"`
	PctVeryHigh      float64 `json:"pct_very_high,omitempty"`      // pLDDT >= 90
	PctConfident     float64 `json:"pct_confident,omitempty"`      // 70 <= pLDDT < 90
	PctLow           float64 `json:"pct_low,omitempty"`            // 50 <= pLDDT < 70
	PctVeryLow       float64 `json:"pct_very_low,omitempty"`       // pLDDT < 50
	InterpretationNote string `json:"interpretation_note"`
}

// ResidueNode links one polymer position between 3D coordinates and biological sequence without false equivalencies.
type ResidueNode struct {
	Index             int      `json:"index"`
	ChainID           string   `json:"chain_id"`
	LabelAsymID       string   `json:"label_asym_id"`
	LabelSeqID        int      `json:"label_seq_id"`
	AuthSeqID         int      `json:"auth_seq_id"`
	InsCode           string   `json:"ins_code,omitempty"`
	AuthLabel         string   `json:"auth_label"`
	CompID            string   `json:"comp_id"`
	OneLetter         string   `json:"one_letter"`
	Observed          bool     `json:"observed"`
	UniProtPos        *int     `json:"uniprot_pos,omitempty"`
	UniProtAA         string   `json:"uniprot_aa,omitempty"`
	SequenceMismatch  bool     `json:"sequence_mismatch,omitempty"`
	Secondary         string   `json:"secondary"` // "helix", "sheet", "coil"
	MetricValue       float64  `json:"metric_value"`
	ConfidenceBand    string   `json:"confidence_band"`
	FeatureLabels     []string `json:"feature_labels,omitempty"`
	LigandContacts    []string `json:"ligand_contacts,omitempty"`
}

// UnmodeledSegment explicitly records regions of a UniProt sequence absent from a given 3D chain.
type UnmodeledSegment struct {
	UniProtStart int    `json:"uniprot_start"`
	UniProtEnd   int    `json:"uniprot_end"`
	Reason       string `json:"reason"`
}

// ChainModel represents one polymeric chain parsed from coordinates and mapped to biological sequence.
type ChainModel struct {
	AuthAsymID         string             `json:"auth_asym_id"`
	LabelAsymID        string             `json:"label_asym_id"`
	EntityID           string             `json:"entity_id"`
	Description        string             `json:"description"`
	PolymerType        string             `json:"polymer_type"` // "protein", "nucleic"
	TotalResidues      int                `json:"total_residues"`
	ObservedCount      int                `json:"observed_count"`
	MissingCount       int                `json:"missing_count"`
	InsertionCodeCount int                `json:"insertion_code_count"`
	MappingMethod      string             `json:"mapping_method"`
	UniProtAccession   string             `json:"uniprot_accession,omitempty"`
	UniProtStart       int                `json:"uniprot_start,omitempty"`
	UniProtEnd         int                `json:"uniprot_end,omitempty"`
	PercentIdentity    float64            `json:"percent_identity,omitempty"`
	MappingNotes       []string           `json:"mapping_notes,omitempty"`
	UnmodeledSegments  []UnmodeledSegment `json:"unmodeled_segments,omitempty"`
	Residues           []ResidueNode      `json:"residues"`
}

// StructureDetail holds the complete parsed 3D model, quality metrics, mappings, and provenance.
type StructureDetail struct {
	ID              string             `json:"id"`
	Kind            StructureKind      `json:"kind"`
	Format          string             `json:"format"` // "pdb" or "mmcif"
	SourceDB        string             `json:"source_db"`
	Title           string             `json:"title"`
	Method          string             `json:"method"`
	DepositionDate  string             `json:"deposition_date,omitempty"`
	ReleaseDate     string             `json:"release_date,omitempty"`
	Organism        string             `json:"organism,omitempty"`
	Citation        string             `json:"citation,omitempty"`
	DOI             string             `json:"doi,omitempty"`
	CoordinateURL   string             `json:"coordinate_url"`
	CoordinateSHA   string             `json:"coordinate_sha256"`
	AtomCount       int                `json:"atom_count"`
	Quality         QualityMetrics     `json:"quality"`
	Ligands         []LigandInfo       `json:"ligands,omitempty"`
	Chains          []ChainModel       `json:"chains"`
	Provenance      []ProvenanceRecord `json:"provenance"`
}

// SIFTSRange captures residue-level mapping intervals returned by PDBe SIFTS.
type SIFTSRange struct {
	ChainID        string `json:"chain_id"`
	StructAsymID   string `json:"struct_asym_id"`
	UniProtAcc     string `json:"uniprot_acc"`
	UniProtStart   int    `json:"uniprot_start"`
	UniProtEnd     int    `json:"uniprot_end"`
	LabelSeqStart  int    `json:"label_seq_start"`
	LabelSeqEnd    int    `json:"label_seq_end"`
	AuthSeqStart   *int   `json:"auth_seq_start,omitempty"`
	AuthSeqEnd     *int   `json:"auth_seq_end,omitempty"`
}

// BlastJobStatus represents lifecycle states of an asynchronous NCBI BLAST search.
type BlastJobStatus string

const (
	BlastQueued    BlastJobStatus = "QUEUED"
	BlastRunning   BlastJobStatus = "RUNNING"
	BlastReady     BlastJobStatus = "READY"
	BlastFailed    BlastJobStatus = "FAILED"
)

// BlastHit represents a sequence similarity hit parsed from NCBI BLAST output.
type BlastHit struct {
	Rank           int     `json:"rank"`
	Accession      string  `json:"accession"`
	UniProtGuess   string  `json:"uniprot_guess,omitempty"`
	PDBChainGuess  string  `json:"pdb_chain_guess,omitempty"`
	Description    string  `json:"description"`
	Organism       string  `json:"organism,omitempty"`
	Length         int     `json:"length"`
	BitScore       float64 `json:"bit_score"`
	EValue         float64 `json:"e_value"`
	IdentityPct    float64 `json:"identity_pct"`
	QueryCoverPct  float64 `json:"query_cover_pct"`
	QueryFrom      int     `json:"query_from"`
	QueryTo        int     `json:"query_to"`
	HitFrom        int     `json:"hit_from"`
	HitTo          int     `json:"hit_to"`
	QueryAlign     string  `json:"query_align,omitempty"`
	Midline        string  `json:"midline,omitempty"`
	HitAlign       string  `json:"hit_align,omitempty"`
}

// BlastJob tracks an asynchronous sequence similarity search with strict rate-limit telemetry.
type BlastJob struct {
	JobID             string           `json:"job_id"`
	RID               string           `json:"rid"`
	Program           string           `json:"program"`
	Database          string           `json:"database"`
	QueryHeader       string           `json:"query_header"`
	QuerySequence     string           `json:"query_sequence"`
	QueryLength       int              `json:"query_length"`
	Status            BlastJobStatus   `json:"status"`
	DeliveryMode      DeliveryMode     `json:"delivery_mode"`
	SubmittedAt       time.Time        `json:"submitted_at"`
	LastPolledAt      time.Time        `json:"last_polled_at"`
	NextAllowedPollAt time.Time        `json:"next_allowed_poll_at"`
	PollCount         int              `json:"poll_count"`
	EstimatedWaitSec  int              `json:"estimated_wait_sec,omitempty"`
	ErrorMessage      string           `json:"error_message,omitempty"`
	Hits              []BlastHit       `json:"hits,omitempty"`
	Provenance        ProvenanceRecord `json:"provenance"`
}

// ExplorationState captures reproducible UI/viewer state for shareable URLs and exports.
type ExplorationState struct {
	Query          string `json:"query"`
	Accession      string `json:"accession,omitempty"`
	StructureID    string `json:"structure_id,omitempty"`
	ChainID        string `json:"chain_id,omitempty"`
	Representation string `json:"representation,omitempty"`
	ColorScheme    string `json:"color_scheme,omitempty"`
	FocusResidues  string `json:"focus_residues,omitempty"`
	ViewerEngine   string `json:"viewer_engine,omitempty"`
}

// ExplorationExport bundles a complete, self-documenting scientific exploration record.
type ExplorationExport struct {
	Application    string             `json:"application"`
	SchemaVersion  string             `json:"schema_version"`
	ExportedAt     time.Time          `json:"exported_at"`
	ShareableURL   string             `json:"shareable_url"`
	State          ExplorationState   `json:"state"`
	Protein        *ProteinRecord     `json:"protein,omitempty"`
	ActiveModel    *StructureDetail   `json:"active_structure,omitempty"`
	AvailableModels []StructureSummary `json:"available_structures,omitempty"`
	Provenance     []ProvenanceRecord `json:"provenance"`
	ScientificNotes []string          `json:"scientific_notes"`
}
