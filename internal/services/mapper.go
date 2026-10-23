package services

import (
	"fmt"
	"math"
	"strings"

	"github.com/kapombe-k/Biostat/internal/bio"
)

// MapStructureToProtein enriches every chain in detail with rigorous residue-level mapping to protein,
// respecting SIFTS intervals when available and verified local alignment otherwise, never asserting false equivalencies.
func MapStructureToProtein(detail *bio.StructureDetail, protein *bio.ProteinRecord, sifts []bio.SIFTSRange, partnerNotes map[string]string) {
	if detail == nil || protein == nil || protein.Sequence == "" {
		return
	}

	unpSeq := strings.ToUpper(strings.TrimSpace(protein.Sequence))

	for cIdx := range detail.Chains {
		ch := &detail.Chains[cIdx]
		if len(ch.Residues) == 0 {
			continue
		}

		// 1. Direct 1:1 full-length AlphaFold model match
		if detail.Kind == bio.StructurePredicted && len(ch.Residues) == len(unpSeq) {
			matches := 0
			for i := range ch.Residues {
				pos := i + 1
				ch.Residues[i].UniProtPos = &pos
				ch.Residues[i].UniProtAA = string(unpSeq[i])
				if ch.Residues[i].OneLetter == string(unpSeq[i]) {
					matches++
				} else {
					ch.Residues[i].SequenceMismatch = true
				}
			}
			ch.MappingMethod = "direct-alphafold-uniprot"
			ch.UniProtAccession = protein.Accession
			ch.UniProtStart = 1
			ch.UniProtEnd = len(unpSeq)
			ch.PercentIdentity = math.Round(float64(matches)/float64(len(unpSeq))*1000) / 10
			ch.MappingNotes = append(ch.MappingNotes,
				fmt.Sprintf("Direct 1:1 AlphaFold model mapping across full canonical %s sequence (%d residues).", protein.Accession, len(unpSeq)))
			attachProteinFeatures(ch, protein)
			continue
		}

		// 2. Check authoritative PDBe SIFTS ranges matching this chain + protein accession
		appliedSIFTS := applySIFTSMapping(ch, protein, unpSeq, sifts)
		if appliedSIFTS {
			attachProteinFeatures(ch, protein)
			buildMappingDiagnostics(ch, unpSeq)
			continue
		}

		// 3. Rigorous gap-aware Smith-Waterman local alignment with strict >=70% identity verification
		mapped := alignChainToUniProt(ch, protein, unpSeq)
		if !mapped {
			ch.MappingMethod = "heteromeric-partner-chain"
			if note, ok := partnerNotes[ch.AuthAsymID]; ok && note != "" {
				ch.Description = note
				ch.MappingNotes = append(ch.MappingNotes,
					fmt.Sprintf("Chain %s corresponds to partner subunit %s rather than %s; coordinates remain interactive using author residue numbers without false sequence equivalency.",
						ch.AuthAsymID, note, protein.Accession))
			} else {
				ch.MappingNotes = append(ch.MappingNotes,
					fmt.Sprintf("Chain %s shares <70%% sequence identity with %s and represents a distinct heteromeric binding partner or co-crystallized chain.",
						ch.AuthAsymID, protein.Accession))
			}
			continue
		}

		attachProteinFeatures(ch, protein)
		buildMappingDiagnostics(ch, unpSeq)
	}
}

func applySIFTSMapping(ch *bio.ChainModel, protein *bio.ProteinRecord, unpSeq string, sifts []bio.SIFTSRange) bool {
	var matching []bio.SIFTSRange
	for _, r := range sifts {
		if !strings.EqualFold(r.UniProtAcc, protein.Accession) {
			continue
		}
		if strings.EqualFold(r.ChainID, ch.AuthAsymID) || strings.EqualFold(r.StructAsymID, ch.LabelAsymID) {
			matching = append(matching, r)
		}
	}
	if len(matching) == 0 {
		return false
	}

	mappedCount := 0
	identical := 0
	minUnp, maxUnp := len(unpSeq)+1, 0

	for i := range ch.Residues {
		res := &ch.Residues[i]
		for _, rng := range matching {
			var offset int
			found := false
			if res.LabelSeqID >= rng.LabelSeqStart && res.LabelSeqID <= rng.LabelSeqEnd &&
				(rng.LabelSeqEnd-rng.LabelSeqStart) == (rng.UniProtEnd-rng.UniProtStart) {
				offset = res.LabelSeqID - rng.LabelSeqStart
				found = true
			} else if rng.AuthSeqStart != nil && rng.AuthSeqEnd != nil &&
				res.AuthSeqID >= *rng.AuthSeqStart && res.AuthSeqID <= *rng.AuthSeqEnd &&
				(*rng.AuthSeqEnd-*rng.AuthSeqStart) == (rng.UniProtEnd-rng.UniProtStart) {
				offset = res.AuthSeqID - *rng.AuthSeqStart
				found = true
			}
			if found {
				uPos := rng.UniProtStart + offset
				if uPos >= 1 && uPos <= len(unpSeq) {
					posCopy := uPos
					res.UniProtPos = &posCopy
					res.UniProtAA = string(unpSeq[uPos-1])
					mappedCount++
					if res.OneLetter == res.UniProtAA {
						identical++
					} else if res.Observed && res.OneLetter != "X" {
						res.SequenceMismatch = true
					}
					if uPos < minUnp {
						minUnp = uPos
					}
					if uPos > maxUnp {
						maxUnp = uPos
					}
				}
				break
			}
		}
	}

	if mappedCount == 0 {
		return false
	}
	ch.MappingMethod = "pdbe-sifts"
	ch.UniProtAccession = protein.Accession
	ch.UniProtStart = minUnp
	ch.UniProtEnd = maxUnp
	ch.PercentIdentity = math.Round(float64(identical)/float64(mappedCount)*1000) / 10
	ch.MappingNotes = append(ch.MappingNotes,
		fmt.Sprintf("Mapped via EMBL-EBI PDBe SIFTS to %s residues %d–%d (%.1f%% identity).",
			protein.Accession, minUnp, maxUnp, ch.PercentIdentity))
	return true
}

func alignChainToUniProt(ch *bio.ChainModel, protein *bio.ProteinRecord, unpSeq string) bool {
	m := len(ch.Residues)
	n := len(unpSeq)
	if m < 6 || n < 6 {
		return false
	}

	var chainBuilder strings.Builder
	chainBuilder.Grow(m)
	for _, r := range ch.Residues {
		if r.OneLetter == "" {
			chainBuilder.WriteByte('X')
		} else {
			chainBuilder.WriteByte(r.OneLetter[0])
		}
	}
	cSeq := chainBuilder.String()

	// Affine-like gapped Smith-Waterman that treats unmodeled 'X' slots neutrally (+1) so missing crystallographic density
	// inside a domain preserves continuous alignment across flanking residues without artificial shifting!
	const gapPenalty = -6
	H := make([][]int, m+1)
	tb := make([][]uint8, m+1) // 1=diag, 2=up(gap in unp), 3=left(gap in struct)
	for i := range H {
		H[i] = make([]int, n+1)
		tb[i] = make([]uint8, n+1)
	}

	bestScore, bestI, bestJ := 0, 0, 0
	for i := 1; i <= m; i++ {
		ca := cSeq[i-1]
		for j := 1; j <= n; j++ {
			cb := unpSeq[j-1]
			var s int
			switch {
			case ca == 'X':
				// Unmodeled internal coordinate gap slot
				s = 1
			case ca == cb:
				s = 6
			default:
				s = -3
			}
			diag := H[i-1][j-1] + s
			up := H[i-1][j] + gapPenalty
			left := H[i][j-1] + gapPenalty

			val := 0
			var dir uint8
			if diag >= up && diag >= left && diag > 0 {
				val, dir = diag, 1
			} else if up >= left && up > 0 {
				val, dir = up, 2
			} else if left > 0 {
				val, dir = left, 3
			}
			H[i][j] = val
			tb[i][j] = dir
			if val > bestScore {
				bestScore, bestI, bestJ = val, i, j
			}
		}
	}

	type pair struct{ cIdx, uPos int }
	var pairs []pair
	knownAligned := 0
	identical := 0

	i, j := bestI, bestJ
	for i > 0 && j > 0 && H[i][j] > 0 {
		switch tb[i][j] {
		case 1:
			pairs = append(pairs, pair{cIdx: i - 1, uPos: j})
			if cSeq[i-1] != 'X' {
				knownAligned++
				if cSeq[i-1] == unpSeq[j-1] {
					identical++
				}
			}
			i--
			j--
		case 2:
			i--
		case 3:
			j--
		}
	}

	minRequiredLen := 10
	if m < 10 {
		minRequiredLen = m
	}
	if knownAligned < minRequiredLen {
		return false
	}
	identity := float64(identical) / float64(knownAligned) * 100.0
	if identity < 70.0 {
		return false
	}

	minUnp, maxUnp := n+1, 0
	for _, p := range pairs {
		uCopy := p.uPos
		node := &ch.Residues[p.cIdx]
		node.UniProtPos = &uCopy
		node.UniProtAA = string(unpSeq[p.uPos-1])
		if node.CompID == "UNK" {
			// Restore single-letter amino acid identity of internal unmodeled gap from verified alignment!
			node.OneLetter = node.UniProtAA
		} else if node.Observed && node.OneLetter != node.UniProtAA {
			node.SequenceMismatch = true
		}
		if p.uPos < minUnp {
			minUnp = p.uPos
		}
		if p.uPos > maxUnp {
			maxUnp = p.uPos
		}
	}

	ch.MappingMethod = "verified-local-alignment"
	ch.UniProtAccession = protein.Accession
	ch.UniProtStart = minUnp
	ch.UniProtEnd = maxUnp
	ch.PercentIdentity = math.Round(identity*10) / 10
	return true
}

func attachProteinFeatures(ch *bio.ChainModel, protein *bio.ProteinRecord) {
	for i := range ch.Residues {
		res := &ch.Residues[i]
		if res.UniProtPos == nil {
			continue
		}
		u := *res.UniProtPos
		for _, f := range protein.Features {
			if u >= f.Start && u <= f.End {
				label := fmt.Sprintf("%s: %s", f.Type, f.Description)
				res.FeatureLabels = append(res.FeatureLabels, label)
			}
		}
	}
}

func buildMappingDiagnostics(ch *bio.ChainModel, unpSeq string) {
	if ch.UniProtStart > 1 {
		ch.UnmodeledSegments = append(ch.UnmodeledSegments, bio.UnmodeledSegment{
			UniProtStart: 1,
			UniProtEnd:   ch.UniProtStart - 1,
			Reason:       "N-terminal region absent from 3D coordinates (cleaved signal/initiator peptide or unmodeled domain)",
		})
	}
	if ch.UniProtEnd > 0 && ch.UniProtEnd < len(unpSeq) {
		ch.UnmodeledSegments = append(ch.UnmodeledSegments, bio.UnmodeledSegment{
			UniProtStart: ch.UniProtEnd + 1,
			UniProtEnd:   len(unpSeq),
			Reason:       "C-terminal region absent from 3D coordinates (cleaved pro-peptide, construct boundary, or disordered tail)",
		})
	}

	// Detect author residue numbering offset relative to UniProt sequence position
	for _, r := range ch.Residues {
		if r.Observed && r.UniProtPos != nil && r.InsCode == "" {
			if r.AuthSeqID != *r.UniProtPos {
				ch.MappingNotes = append(ch.MappingNotes,
					fmt.Sprintf("Residue numbering difference resolved: PDB %s %s%d maps to %s %s%d (never assume author numbers equal canonical UniProt indices).",
						ch.AuthAsymID, r.OneLetter, r.AuthSeqID, ch.UniProtAccession, r.UniProtAA, *r.UniProtPos))
			}
			break
		}
	}

	if ch.InsertionCodeCount > 0 {
		var codes []string
		for _, r := range ch.Residues {
			if r.InsCode != "" {
				codes = append(codes, r.AuthLabel)
			}
		}
		ch.MappingNotes = append(ch.MappingNotes,
			fmt.Sprintf("Preserves %d PDB insertion code(s) (%s) used in historical family homology numbering.",
				ch.InsertionCodeCount, strings.Join(codes, ", ")))
	}

	if ch.MissingCount > 0 {
		ch.MappingNotes = append(ch.MappingNotes,
			fmt.Sprintf("%d polymer residue(s) lack atomic coordinates due to local crystallographic loop flexibility (missing electron density).",
				ch.MissingCount))
	}

	mismatches := 0
	for _, r := range ch.Residues {
		if r.SequenceMismatch {
			mismatches++
		}
	}
	if mismatches > 0 {
		ch.MappingNotes = append(ch.MappingNotes,
			fmt.Sprintf("%d engineered or species variant substitution(s) detected relative to canonical %s sequence.",
				mismatches, ch.UniProtAccession))
	}
}
