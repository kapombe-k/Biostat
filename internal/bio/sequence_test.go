package bio

import (
	"strings"
	"testing"
)

func TestParseAndValidateSequence(t *testing.T) {
	fasta := ">sp|P01308|INS_HUMAN Insulin OS=Homo sapiens\nMALWMRLLPLLALLALWGPDPAAAFVNQHLCGSHLVEALYLVCGERGFFYTPKTRREAED*"
	parsed, err := ParseAndValidateSequence(fasta, 10, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Length != 60 {
		t.Fatalf("expected length 60, got %d", parsed.Length)
	}
	if !strings.HasPrefix(parsed.Header, "sp|P01308") {
		t.Fatalf("unexpected header: %s", parsed.Header)
	}

	// Reject multi-FASTA
	multi := ">seq1\nACDEFGHIKLMNPQRSTVWY\n>seq2\nACDEFGHIKLMNPQRSTVWY"
	if _, err := ParseAndValidateSequence(multi, 10, 5000); err == nil {
		t.Fatalf("expected multi-FASTA error")
	}

	// Reject invalid non-IUPAC symbols
	if _, err := ParseAndValidateSequence("ACDEFGHIKLMN$@#", 5, 5000); err == nil {
		t.Fatalf("expected invalid symbol error")
	}

	// Warn on nucleotide-looking input
	nt, err := ParseAndValidateSequence("ATGGCCCTGTGGATGCGCCTCCTGCCCCTGCTGGCGCTGCTGGCCCTCTGGGGACCTGAC", 10, 5000)
	if err != nil {
		t.Fatalf("unexpected error on nucleotide string: %v", err)
	}
	if len(nt.Warnings) == 0 {
		t.Fatalf("expected nucleotide warning")
	}
}

func TestClassifyQuery(t *testing.T) {
	cases := []struct {
		in   string
		kind QueryKind
		norm string
	}{
		{"P04637", QueryUniProtAccession, "P04637"},
		{"p01308-2", QueryUniProtAccession, "P01308"},
		{"AF-P04637-F1-model_v6", QueryAlphaFoldID, "AF-P04637-F1"},
		{"1gzx", QueryPDBID, "1GZX"},
		{">my_prot\nACDEFGHIKLMNPQRSTVWY", QueryFASTA, ">my_prot\nACDEFGHIKLMNPQRSTVWY"},
		{"MALWMRLLPLLALLALWGPDPAAAFVNQHLCGSHLV", QueryFASTA, "MALWMRLLPLLALLALWGPDPAAAFVNQHLCGSHLV"},
		{"human hemoglobin", QueryKeyword, "human hemoglobin"},
	}

	for _, tc := range cases {
		gotKind, gotNorm := ClassifyQuery(tc.in)
		if gotKind != tc.kind || gotNorm != tc.norm {
			t.Errorf("ClassifyQuery(%q) = (%s, %q); want (%s, %q)", tc.in, gotKind, gotNorm, tc.kind, gotNorm)
		}
	}
}
