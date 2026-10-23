package bio

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	reUniProtAcc  = regexp.MustCompile(`^(?:[OPQ][0-9][A-Z0-9]{3}[0-9]|[A-NR-Z][0-9](?:[A-Z][A-Z0-9]{2}[0-9]){1,2})(?:-\d+)?$`)
	reAlphaFoldID = regexp.MustCompile(`(?i)^AF-([A-Z0-9]+)-F\d+(?:-model_v\d+)?$`)
	rePDBID       = regexp.MustCompile(`^[1-9][A-Za-z0-9]{3}$`)
)

// Valid IUPAC amino acid letters accepted in protein sequence validation.
const validProteinLetters = "ACDEFGHIKLMNPQRSTVWYBZXUO"

var threeToOneMap = map[string]string{
	"ALA": "A", "ARG": "R", "ASN": "N", "ASP": "D", "CYS": "C",
	"GLN": "Q", "GLU": "E", "GLY": "G", "HIS": "H", "ILE": "I",
	"LEU": "L", "LYS": "K", "MET": "M", "PHE": "F", "PRO": "P",
	"SER": "S", "THR": "T", "TRP": "W", "TYR": "Y", "VAL": "V",
	// Common modified / protonation state residues encountered in PDB/mmCIF entries
	"MSE": "M", "SEP": "S", "TPO": "T", "PTR": "Y", "HYP": "P",
	"CSO": "C", "CSD": "C", "CME": "C", "CSS": "C", "CYX": "C",
	"HSD": "H", "HSE": "H", "HSP": "H", "HID": "H", "HIE": "H", "HIP": "H",
	"KCX": "K", "MLY": "K", "PCA": "E", "SEC": "U", "PYL": "O",
	"ASX": "B", "GLX": "Z", "UNK": "X",
}

// ThreeToOne converts a 3-letter PDB monomer code into a 1-letter amino acid symbol.
func ThreeToOne(compID string) string {
	if one, ok := threeToOneMap[strings.ToUpper(strings.TrimSpace(compID))]; ok {
		return one
	}
	return "X"
}

// IsStandardPolymerComp returns true if compID is a known amino acid residue rather than solvent/ligand.
func IsStandardPolymerComp(compID string) bool {
	_, ok := threeToOneMap[strings.ToUpper(strings.TrimSpace(compID))]
	return ok
}

// FormatAuthResidue formats an author sequence number and optional PDB insertion code (e.g. 184 + "A" -> "184A").
func FormatAuthResidue(authSeqID int, insCode string) string {
	ic := strings.TrimSpace(insCode)
	if ic == "." || ic == "?" {
		ic = ""
	}
	return fmt.Sprintf("%d%s", authSeqID, ic)
}

// PLDDTBand returns the standard AlphaFold confidence category name for a per-residue pLDDT score.
func PLDDTBand(score float64) string {
	switch {
	case score >= 90:
		return "very-high"
	case score >= 70:
		return "confident"
	case score >= 50:
		return "low"
	default:
		return "very-low"
	}
}

// ParsedSequence holds cleaned sequence data and optional warnings from FASTA/raw sequence parsing.
type ParsedSequence struct {
	Header   string   `json:"header"`
	Sequence string   `json:"sequence"`
	Length   int      `json:"length"`
	Warnings []string `json:"warnings,omitempty"`
}

// ParseAndValidateSequence validates raw or FASTA-formatted amino acid input within configured length bounds.
func ParseAndValidateSequence(raw string, minLen, maxLen int) (*ParsedSequence, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("sequence input is empty")
	}

	var header string
	var seqBuilder strings.Builder
	lines := strings.Split(trimmed, "\n")

	for idx, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, ">") {
			if idx == 0 {
				header = strings.TrimSpace(strings.TrimPrefix(l, ">"))
				continue
			}
			return nil, errors.New("multiple FASTA entries detected; please submit a single protein sequence at a time")
		}
		if strings.HasPrefix(l, ";") {
			continue
		}
		for _, r := range l {
			if unicode.IsSpace(r) || unicode.IsDigit(r) || r == '-' {
				continue
			}
			if r == '*' {
				// Terminal translation stop symbol is harmless at end of ORF.
				continue
			}
			upper := unicode.ToUpper(r)
			if !strings.ContainsRune(validProteinLetters, upper) {
				return nil, fmt.Errorf("invalid amino acid symbol %q in sequence; expected standard IUPAC protein letters", string(r))
			}
			seqBuilder.WriteRune(upper)
		}
	}

	seq := seqBuilder.String()
	if len(seq) < minLen {
		return nil, fmt.Errorf("sequence length (%d aa) is shorter than minimum required length (%d aa)", len(seq), minLen)
	}
	if maxLen > 0 && len(seq) > maxLen {
		return nil, fmt.Errorf("sequence length (%d aa) exceeds maximum allowed length (%d aa)", len(seq), maxLen)
	}

	if header == "" {
		header = fmt.Sprintf("User_Query_%daa", len(seq))
	}

	var warnings []string
	ntCount := 0
	for _, r := range seq {
		if r == 'A' || r == 'C' || r == 'G' || r == 'T' || r == 'U' || r == 'N' {
			ntCount++
		}
	}
	if len(seq) >= 30 && float64(ntCount)/float64(len(seq)) >= 0.94 {
		warnings = append(warnings, "Input contains >=94% A/C/G/T/U symbols and may be a nucleotide (DNA/RNA) sequence. BioScope runs protein BLAST (blastp); consider translating your coding sequence first.")
	}

	return &ParsedSequence{
		Header:   header,
		Sequence: seq,
		Length:   len(seq),
		Warnings: warnings,
	}, nil
}

// FormatFASTA renders a protein sequence into standard 60-character wrapped FASTA text.
func FormatFASTA(header, sequence string) string {
	var b strings.Builder
	h := strings.TrimSpace(header)
	if h == "" {
		h = "BioScope_Sequence"
	}
	b.WriteString(">")
	b.WriteString(h)
	b.WriteByte('\n')
	clean := strings.TrimSpace(sequence)
	for i := 0; i < len(clean); i += 60 {
		end := i + 60
		if end > len(clean) {
			end = len(clean)
		}
		b.WriteString(clean[i:end])
		b.WriteByte('\n')
	}
	return b.String()
}

// ClassifyQuery determines whether user input is FASTA, an AlphaFold model ID, a PDB ID, a UniProt accession, or a search keyword.
func ClassifyQuery(raw string) (QueryKind, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return QueryKeyword, ""
	}
	if strings.HasPrefix(trimmed, ">") || strings.Contains(trimmed, "\n") {
		return QueryFASTA, trimmed
	}

	upper := strings.ToUpper(trimmed)
	if reAlphaFoldID.MatchString(upper) {
		// Normalize e.g. AF-P04637-F1-MODEL_V6 -> AF-P04637-F1
		parts := strings.Split(upper, "-")
		if len(parts) >= 3 {
			return QueryAlphaFoldID, fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		}
		return QueryAlphaFoldID, upper
	}
	if reUniProtAcc.MatchString(upper) {
		base := strings.Split(upper, "-")[0]
		return QueryUniProtAccession, base
	}
	if rePDBID.MatchString(upper) {
		return QueryPDBID, upper
	}

	// Single-line raw amino acid string without spaces of length >= 20
	if len(trimmed) >= 20 && !strings.ContainsAny(trimmed, " \t_-:") {
		allAA := true
		for _, r := range upper {
			if !strings.ContainsRune(validProteinLetters, r) {
				allAA = false
				break
			}
		}
		if allAA {
			return QueryFASTA, upper
		}
	}

	return QueryKeyword, trimmed
}

// ExtractUniProtFromAFID extracts "P04637" from "AF-P04637-F1".
func ExtractUniProtFromAFID(afID string) string {
	m := reAlphaFoldID.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(afID)))
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}
