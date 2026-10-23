package clients

import (
	"context"

	"github.com/kapombe-k/Biostat/internal/bio"
)

// UniProtClient defines operations against the UniProtKB REST API.
type UniProtClient interface {
	GetProtein(ctx context.Context, accession string) (*bio.ProteinRecord, error)
	SearchProteins(ctx context.Context, query string, limit int) ([]bio.SearchHit, error)
}

// RCSBClient defines structure discovery, metadata, and coordinate retrieval from RCSB PDB.
type RCSBClient interface {
	SearchByUniProt(ctx context.Context, accession string, limit int) ([]string, error)
	SearchByKeyword(ctx context.Context, keyword string, limit int) ([]string, error)
	GetEntrySummary(ctx context.Context, pdbID string) (*bio.StructureSummary, *EntryEnrichment, error)
	FetchCoordinates(ctx context.Context, pdbID string) (raw []byte, format string, endpoint string, err error)
}

// EntryEnrichment holds bibliographic and refinement metadata fetched from RCSB/PDBe APIs.
type EntryEnrichment struct {
	Title          string
	Method         string
	Resolution     *float64
	RWork          *float64
	RFree          *float64
	DepositionDate string
	ReleaseDate    string
	Citation       string
	DOI            string
	Organism       string
	LigandIDs      []string
}

// AlphaFoldClient defines model lookup and coordinate retrieval from the AlphaFold Protein Structure Database.
type AlphaFoldClient interface {
	GetPrediction(ctx context.Context, uniprotAccession string) (*AlphaFoldPrediction, error)
	FetchCoordinates(ctx context.Context, prediction *AlphaFoldPrediction) (raw []byte, format string, endpoint string, err error)
}

// AlphaFoldPrediction represents metadata returned by the AlphaFold DB REST API.
type AlphaFoldPrediction struct {
	EntryID         string
	UniProtAcc      string
	ModelVersion    int
	Sequence        string
	MeanPLDDT       float64
	FracVeryHigh    float64
	FracConfident   float64
	FracLow         float64
	FracVeryLow     float64
	CIFURL          string
	PDBURL          string
	PAEImageURL     string
	MetadataAPIURL  string
}

// PDBeClient defines enrichment calls to EMBL-EBI PDBe and PDBe-KB (including SIFTS residue mappings).
type PDBeClient interface {
	GetValidationMetrics(ctx context.Context, pdbID string) (*PDBeValidation, error)
	GetSIFTSMappings(ctx context.Context, pdbID string) ([]bio.SIFTSRange, error)
}

// PDBeValidation holds structure validation percentiles from EMBL-EBI PDBe.
type PDBeValidation struct {
	ClashscorePercentile *float64
	RamachandranOutliers *float64
	SidechainOutliers    *float64
	SourceURL            string
}

// BlastClient defines asynchronous sequence similarity operations against the NCBI BLAST URL API.
type BlastClient interface {
	SubmitSearch(ctx context.Context, database, sequence string) (rid string, rtoeSec int, endpoint string, err error)
	PollStatus(ctx context.Context, rid string) (status bio.BlastJobStatus, message string, err error)
	FetchResults(ctx context.Context, rid string, queryLen int) ([]bio.BlastHit, error)
}
