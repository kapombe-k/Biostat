package bio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type atomPoint struct {
	x, y, z float64
	name    string
	bfac    float64
	isHeavy bool
}

type rawResidue struct {
	chainID     string
	labelAsymID string
	entityID    string
	labelSeqID  int
	authSeqID   int
	insCode     string
	compID      string
	observed    bool
	secondary   string
	atoms       []atomPoint
	caPos       *[3]float64
}

type rawLigandKey struct {
	compID    string
	chainID   string
	authSeqID int
}

type rawLigand struct {
	key   rawLigandKey
	atoms []atomPoint
}

var commonLigandNames = map[string][2]string{
	"HEM": {"Protoporphyrin IX containing Fe (Heme)", "Prosthetic oxygen-carrying cofactor"},
	"HEB": {"Heme B", "Prosthetic cofactor"},
	"HEC": {"Heme C", "Prosthetic cofactor"},
	"OXY": {"Dioxygen (O₂)", "Bound respiratory ligand"},
	"STI": {"Imatinib (Gleevec / STI-571)", "Type-II ATP-competitive kinase inhibitor"},
	"GBS": {"4-Guanidinobenzoic Acid", "Covalent active-site acyl-enzyme inhibitor"},
	"ZN":  {"Zinc Ion (Zn²⁺)", "Structural / catalytic metal cofactor"},
	"MG":  {"Magnesium Ion (Mg²⁺)", "Catalytic divalent cation"},
	"CA":  {"Calcium Ion (Ca²⁺)", "Structural stabilizing metal ion"},
	"FE":  {"Iron Ion", "Redox / coordination metal center"},
	"ATP": {"Adenosine-5'-Triphosphate", "Nucleotide cosubstrate"},
	"ADP": {"Adenosine-5'-Diphosphate", "Nucleotide product"},
	"GTP": {"Guanosine-5'-Triphosphate", "Nucleotide signaling cofactor"},
	"NAD": {"Nicotinamide Adenine Dinucleotide", "Redox coenzyme"},
	"NAP": {"NADP⁺", "Redox coenzyme"},
	"FAD": {"Flavin Adenine Dinucleotide", "Redox prosthetic group"},
	"SO4": {"Sulfate Ion (SO₄²⁻)", "Crystallization buffer oxyanion"},
	"PO4": {"Phosphate Ion (PO₄³⁻)", "Oxyanion ligand"},
	"NAG": {"N-Acetyl-D-Glucosamine", "N-linked glycan monosaccharide"},
}

var ignoredSolvent = map[string]bool{
	"HOH": true, "WAT": true, "DOD": true, "H2O": true,
}

// ParseCoordinates parses either PDB or mmCIF text into a strongly-typed StructureDetail.
func ParseCoordinates(id string, kind StructureKind, raw []byte) (*StructureDetail, error) {
	sum := sha256.Sum256(raw)
	shaHex := hex.EncodeToString(sum[:])

	text := string(raw)
	isCIF := looksLikeCIF(text)

	var (
		detail    *StructureDetail
		chainList []*parsedChainRaw
		ligands   []*rawLigand
		err       error
	)

	if isCIF {
		detail, chainList, ligands, err = parseMMCIF(id, kind, text)
	} else {
		detail, chainList, ligands, err = parsePDB(id, kind, text)
	}
	if err != nil {
		return nil, err
	}
	detail.CoordinateSHA = shaHex

	// Finalize per-chain secondary structure (falling back to C-alpha geometry when headers omit HELIX/SHEET),
	// B-factor / pLDDT statistics, and 4.5 Angstrom atomic ligand contact discovery.
	var (
		totalSlots    int
		totalObserved int
		sumMetric     float64
		nVeryHigh     int
		nConfident    int
		nLow          int
		nVeryLow      int
	)

	// Index residue nodes by pointer for fast ligand contact annotation
	type resLookup struct {
		chainIdx int
		resIdx   int
		raw      *rawResidue
	}
	var allObserved []resLookup

	detail.Chains = make([]ChainModel, 0, len(chainList))
	for cIdx, pc := range chainList {
		hasExplicitSS := false
		for _, rr := range pc.residues {
			if rr.secondary == "helix" || rr.secondary == "sheet" {
				hasExplicitSS = true
				break
			}
		}
		if !hasExplicitSS {
			inferSecondaryFromCA(pc.residues)
		}

		cm := ChainModel{
			AuthAsymID:  pc.authAsymID,
			LabelAsymID: pc.labelAsymID,
			EntityID:    pc.entityID,
			Description: pc.description,
			PolymerType: pc.polymerType,
			Residues:    make([]ResidueNode, 0, len(pc.residues)),
		}
		if cm.PolymerType == "" {
			cm.PolymerType = "protein"
		}
		if cm.Description == "" {
			cm.Description = fmt.Sprintf("Polymer Chain %s", cm.AuthAsymID)
		}

		for rIdx, rr := range pc.residues {
			var meanVal float64
			if rr.observed && len(rr.atoms) > 0 {
				var s float64
				for _, a := range rr.atoms {
					s += a.bfac
				}
				meanVal = math.Round((s/float64(len(rr.atoms)))*100) / 100
			}
			ss := rr.secondary
			if ss == "" {
				ss = "coil"
			}
			band := "experimental"
			if kind == StructurePredicted && rr.observed {
				band = PLDDTBand(meanVal)
				switch band {
				case "very-high":
					nVeryHigh++
				case "confident":
					nConfident++
				case "low":
					nLow++
				default:
					nVeryLow++
				}
			}
			if rr.insCode != "" {
				cm.InsertionCodeCount++
			}
			if rr.observed {
				cm.ObservedCount++
				totalObserved++
				sumMetric += meanVal
			} else {
				cm.MissingCount++
			}
			totalSlots++

			node := ResidueNode{
				Index:          rIdx,
				ChainID:        rr.chainID,
				LabelAsymID:    rr.labelAsymID,
				LabelSeqID:     rr.labelSeqID,
				AuthSeqID:      rr.authSeqID,
				InsCode:        rr.insCode,
				AuthLabel:      FormatAuthResidue(rr.authSeqID, rr.insCode),
				CompID:         rr.compID,
				OneLetter:      ThreeToOne(rr.compID),
				Observed:       rr.observed,
				Secondary:      ss,
				MetricValue:    meanVal,
				ConfidenceBand: band,
			}
			cm.Residues = append(cm.Residues, node)
			if rr.observed {
				allObserved = append(allObserved, resLookup{
					chainIdx: cIdx,
					resIdx:   rIdx,
					raw:      rr,
				})
			}
		}
		cm.TotalResidues = len(cm.Residues)
		detail.Chains = append(detail.Chains, cm)
	}

	// Compute spatial ligand-binding pockets within 4.5 Angstroms of heavy atoms.
	const contactCutoffSq = 4.5 * 4.5
	for _, lig := range ligands {
		meta, ok := commonLigandNames[lig.key.compID]
		name, role := lig.key.compID, "Bound non-polymer heterogen"
		if ok {
			name, role = meta[0], meta[1]
		}
		li := LigandInfo{
			CompID:    lig.key.compID,
			Name:      name,
			Role:      role,
			ChainID:   lig.key.chainID,
			AuthSeqID: lig.key.authSeqID,
			AtomCount: len(lig.atoms),
		}

		for _, cand := range allObserved {
			minSq := math.MaxFloat64
			for _, la := range lig.atoms {
				if !la.isHeavy {
					continue
				}
				for _, pa := range cand.raw.atoms {
					if !pa.isHeavy {
						continue
					}
					dx := la.x - pa.x
					dy := la.y - pa.y
					dz := la.z - pa.z
					dsq := dx*dx + dy*dy + dz*dz
					if dsq < minSq {
						minSq = dsq
					}
				}
			}
			if minSq <= contactCutoffSq {
				dist := math.Round(math.Sqrt(minSq)*100) / 100
				li.ContactResidues = append(li.ContactResidues, ResidueRef{
					ChainID:     cand.raw.chainID,
					AuthSeqID:   cand.raw.authSeqID,
					InsCode:     cand.raw.insCode,
					CompID:      cand.raw.compID,
					DistanceAng: dist,
				})
				node := &detail.Chains[cand.chainIdx].Residues[cand.resIdx]
				tag := fmt.Sprintf("%s (%.1f Å)", lig.key.compID, dist)
				node.LigandContacts = append(node.LigandContacts, tag)
				node.FeatureLabels = append(node.FeatureLabels, fmt.Sprintf("Binding pocket contact: %s at %.2f Å", lig.key.compID, dist))
			}
		}
		sort.Slice(li.ContactResidues, func(i, j int) bool {
			return li.ContactResidues[i].DistanceAng < li.ContactResidues[j].DistanceAng
		})
		detail.Ligands = append(detail.Ligands, li)
	}

	if totalSlots > 0 {
		detail.Quality.ObservedResiduePct = math.Round((float64(totalObserved)/float64(totalSlots))*1000) / 10
	}
	if totalObserved > 0 {
		avg := math.Round((sumMetric/float64(totalObserved))*10) / 10
		if kind == StructurePredicted {
			detail.Quality.MeanPLDDT = avg
			detail.Quality.PctVeryHigh = math.Round((float64(nVeryHigh)/float64(totalObserved))*1000) / 10
			detail.Quality.PctConfident = math.Round((float64(nConfident)/float64(totalObserved))*1000) / 10
			detail.Quality.PctLow = math.Round((float64(nLow)/float64(totalObserved))*1000) / 10
			detail.Quality.PctVeryLow = math.Round((float64(nVeryLow)/float64(totalObserved))*1000) / 10
			detail.Quality.InterpretationNote = "Predicted AlphaFold model: B-factor column stores per-residue pLDDT confidence (0–100), NOT crystallographic temperature factors. Regions with pLDDT < 50 are frequently intrinsically disordered in solution and lack fixed relative orientation without binding partners or cofactors."
		} else {
			detail.Quality.MeanBFactor = avg
			detail.Quality.InterpretationNote = "Experimental structure: atomic coordinates derive from physical diffraction/microscopy/NMR measurements. Missing residues indicate local flexibility with weak electron density rather than sequence deletion."
		}
	}

	return detail, nil
}

type parsedChainRaw struct {
	authAsymID  string
	labelAsymID string
	entityID    string
	description string
	polymerType string
	residues    []*rawResidue
}

type resKey struct {
	chain   string
	authSeq int
	insCode string
}

func looksLikeCIF(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\r\n")
	return strings.HasPrefix(trimmed, "data_") || strings.Contains(text, "\n_atom_site.")
}

func parsePDB(id string, kind StructureKind, text string) (*StructureDetail, []*parsedChainRaw, []*rawLigand, error) {
	detail := &StructureDetail{
		ID:     strings.ToUpper(strings.TrimSpace(id)),
		Kind:   kind,
		Format: "pdb",
	}
	if kind == StructurePredicted {
		detail.SourceDB = "AlphaFold DB"
		detail.Method = "ALPHAFOLD MONOMER PREDICTION"
	} else {
		detail.SourceDB = "RCSB PDB"
		detail.Method = "X-RAY DIFFRACTION"
	}

	ssMap := make(map[resKey]string)
	missingByChain := make(map[string][]*rawResidue)
	chainMap := make(map[string]*parsedChainRaw)
	var chainOrder []string
	resIndex := make(map[resKey]*rawResidue)
	ligMap := make(map[rawLigandKey]*rawLigand)
	var ligOrder []rawLigandKey

	var titleParts []string
	inModel := 0

	for _, line := range strings.Split(text, "\n") {
		if len(line) < 6 {
			continue
		}
		rec := strings.TrimSpace(line[:6])
		switch rec {
		case "TITLE":
			if len(line) > 10 {
				titleParts = append(titleParts, strings.TrimSpace(line[10:]))
			}
		case "EXPDTA":
			if len(line) > 10 && kind == StructureExperimental {
				detail.Method = strings.TrimSpace(line[10:])
			}
		case "REMARK":
			if len(line) > 26 && strings.TrimSpace(line[7:10]) == "2" && strings.Contains(line, "RESOLUTION.") {
				fields := strings.Fields(line[11:])
				for _, f := range fields {
					if v, err := strconv.ParseFloat(f, 64); err == nil && v > 0 && v < 50 {
						detail.Quality.ResolutionAngstrom = &v
						break
					}
				}
			} else if len(line) >= 27 && strings.TrimSpace(line[7:10]) == "465" {
				// REMARK 465 M RES C SSSEQI
				sub := line[15:]
				if len(sub) >= 11 {
					resn := strings.TrimSpace(sub[0:4])
					ch := strings.TrimSpace(sub[4:6])
					seqStr := strings.TrimSpace(sub[6:11])
					ic := ""
					if len(sub) >= 12 {
						ic = strings.TrimSpace(sub[11:12])
					}
					if n, err := strconv.Atoi(seqStr); err == nil && IsStandardPolymerComp(resn) && ch != "" {
						missingByChain[ch] = append(missingByChain[ch], &rawResidue{
							chainID:     ch,
							labelAsymID: ch,
							authSeqID:   n,
							insCode:     ic,
							compID:      resn,
							observed:    false,
						})
					}
				}
			}
		case "HELIX":
			if len(line) >= 37 {
				ch := strings.TrimSpace(string(line[19]))
				s1, e1 := strconv.Atoi(strings.TrimSpace(line[21:25]))
				s2, e2 := strconv.Atoi(strings.TrimSpace(line[33:37]))
				if e1 == nil && e2 == nil {
					for r := s1; r <= s2; r++ {
						ssMap[resKey{chain: ch, authSeq: r}] = "helix"
					}
				}
			}
		case "SHEET":
			if len(line) >= 37 {
				ch := strings.TrimSpace(string(line[21]))
				s1, e1 := strconv.Atoi(strings.TrimSpace(line[22:26]))
				s2, e2 := strconv.Atoi(strings.TrimSpace(line[33:37]))
				if e1 == nil && e2 == nil {
					for r := s1; r <= s2; r++ {
						ssMap[resKey{chain: ch, authSeq: r}] = "sheet"
					}
				}
			}
		case "MODEL":
			inModel++
		case "ENDMDL":
			if inModel >= 1 {
				goto doneLines
			}
		case "ATOM", "HETATM":
			if inModel > 1 || len(line) < 54 {
				continue
			}
			altLoc := line[16]
			if altLoc != ' ' && altLoc != 'A' && altLoc != '1' {
				continue
			}
			atomName := strings.TrimSpace(line[12:16])
			compID := strings.TrimSpace(line[17:20])
			chainID := strings.TrimSpace(string(line[21]))
			if chainID == "" {
				chainID = "A"
			}
			authSeq, err := strconv.Atoi(strings.TrimSpace(line[22:26]))
			if err != nil {
				continue
			}
			insCode := ""
			if len(line) > 26 && line[26] != ' ' {
				insCode = string(line[26])
			}
			x, errX := strconv.ParseFloat(strings.TrimSpace(line[30:38]), 64)
			y, errY := strconv.ParseFloat(strings.TrimSpace(line[38:46]), 64)
			z, errZ := strconv.ParseFloat(strings.TrimSpace(line[46:54]), 64)
			if errX != nil || errY != nil || errZ != nil {
				continue
			}
			bfac := 0.0
			if len(line) >= 66 {
				bfac, _ = strconv.ParseFloat(strings.TrimSpace(line[60:66]), 64)
			}
			elem := ""
			if len(line) >= 78 {
				elem = strings.TrimSpace(line[76:78])
			}
			if elem == "" && len(atomName) > 0 {
				elem = string(atomName[0])
			}
			isHeavy := !strings.EqualFold(elem, "H") && !strings.EqualFold(elem, "D")
			pt := atomPoint{x: x, y: y, z: z, name: atomName, bfac: bfac, isHeavy: isHeavy}
			detail.AtomCount++

			if rec == "ATOM" || (rec == "HETATM" && IsStandardPolymerComp(compID)) {
				pc, ok := chainMap[chainID]
				if !ok {
					pc = &parsedChainRaw{
						authAsymID:  chainID,
						labelAsymID: chainID,
						entityID:    fmt.Sprintf("%d", len(chainOrder)+1),
						polymerType: "protein",
					}
					chainMap[chainID] = pc
					chainOrder = append(chainOrder, chainID)
				}
				rk := resKey{chain: chainID, authSeq: authSeq, insCode: insCode}
				rr, exists := resIndex[rk]
				if !exists {
					ss := ssMap[rk]
					if ss == "" {
						ss = ssMap[resKey{chain: chainID, authSeq: authSeq}]
					}
					rr = &rawResidue{
						chainID:     chainID,
						labelAsymID: chainID,
						entityID:    pc.entityID,
						labelSeqID:  len(pc.residues) + 1,
						authSeqID:   authSeq,
						insCode:     insCode,
						compID:      compID,
						observed:    true,
						secondary:   ss,
					}
					resIndex[rk] = rr
					pc.residues = append(pc.residues, rr)
				}
				rr.atoms = append(rr.atoms, pt)
				if atomName == "CA" {
					rr.caPos = &[3]float64{x, y, z}
				}
			} else if rec == "HETATM" && !ignoredSolvent[strings.ToUpper(compID)] {
				lk := rawLigandKey{compID: strings.ToUpper(compID), chainID: chainID, authSeqID: authSeq}
				lg, ok := ligMap[lk]
				if !ok {
					lg = &rawLigand{key: lk}
					ligMap[lk] = lg
					ligOrder = append(ligOrder, lk)
				}
				lg.atoms = append(lg.atoms, pt)
			}
		}
	}

doneLines:
	if len(titleParts) > 0 {
		detail.Title = strings.Join(titleParts, " ")
	}

	// Merge any REMARK 465 missing residues or internal small sequence gaps cleanly without false equivalencies.
	var chains []*parsedChainRaw
	for _, cid := range chainOrder {
		pc := chainMap[cid]
		merged := mergeMissingResiduesPDB(pc.residues, missingByChain[cid])
		for idx, rr := range merged {
			rr.labelSeqID = idx + 1
		}
		pc.residues = merged
		chains = append(chains, pc)
	}

	var ligands []*rawLigand
	for _, lk := range ligOrder {
		ligands = append(ligands, ligMap[lk])
	}
	return detail, chains, ligands, nil
}

func mergeMissingResiduesPDB(observed []*rawResidue, remarkMissing []*rawResidue) []*rawResidue {
	if len(observed) == 0 {
		return observed
	}
	if len(remarkMissing) > 0 {
		byKey := make(map[resKey]*rawResidue, len(observed)+len(remarkMissing))
		all := make([]*rawResidue, 0, len(observed)+len(remarkMissing))
		for _, r := range observed {
			k := resKey{chain: r.chainID, authSeq: r.authSeqID, insCode: r.insCode}
			byKey[k] = r
			all = append(all, r)
		}
		for _, m := range remarkMissing {
			k := resKey{chain: m.chainID, authSeq: m.authSeqID, insCode: m.insCode}
			if _, exists := byKey[k]; !exists {
				byKey[k] = m
				all = append(all, m)
			}
		}
		sort.SliceStable(all, func(i, j int) bool {
			if all[i].authSeqID != all[j].authSeqID {
				return all[i].authSeqID < all[j].authSeqID
			}
			return all[i].insCode < all[j].insCode
		})
		return all
	}

	// If REMARK 465 was stripped from a PDB extract, detect internal missing loop positions (small jumps <= 25 residues).
	out := make([]*rawResidue, 0, len(observed)+8)
	for i, cur := range observed {
		if i > 0 {
			prev := observed[i-1]
			gap := cur.authSeqID - prev.authSeqID - 1
			if prev.insCode == "" && cur.insCode == "" && gap > 0 && gap <= 25 {
				for missingNum := prev.authSeqID + 1; missingNum < cur.authSeqID; missingNum++ {
					out = append(out, &rawResidue{
						chainID:     cur.chainID,
						labelAsymID: cur.labelAsymID,
						entityID:    cur.entityID,
						authSeqID:   missingNum,
						compID:      "UNK",
						observed:    false,
					})
				}
			}
		}
		out = append(out, cur)
	}
	return out
}

func parseMMCIF(id string, kind StructureKind, text string) (*StructureDetail, []*parsedChainRaw, []*rawLigand, error) {
	detail := &StructureDetail{
		ID:     strings.ToUpper(strings.TrimSpace(id)),
		Kind:   kind,
		Format: "mmcif",
	}
	if kind == StructurePredicted {
		detail.SourceDB = "AlphaFold DB"
		detail.Method = "ALPHAFOLD MONOMER PREDICTION"
	} else {
		detail.SourceDB = "RCSB PDB"
	}

	blocks := parseCIFTables(text)

	if t := blocks.scalar("_struct.title"); t != "" {
		detail.Title = t
	}
	if m := blocks.scalar("_exptl.method"); m != "" && kind == StructureExperimental {
		detail.Method = m
	}
	if resStr := blocks.firstScalar("_refine.ls_d_res_high", "_em_3d_reconstruction.resolution"); resStr != "" {
		if v, err := strconv.ParseFloat(resStr, 64); err == nil && v > 0 {
			detail.Quality.ResolutionAngstrom = &v
		}
	}
	if rwStr := blocks.scalar("_refine.ls_R_factor_R_work"); rwStr != "" {
		if v, err := strconv.ParseFloat(rwStr, 64); err == nil && v > 0 {
			detail.Quality.RWork = &v
		}
	}
	if rfStr := blocks.scalar("_refine.ls_R_factor_R_free"); rfStr != "" {
		if v, err := strconv.ParseFloat(rfStr, 64); err == nil && v > 0 {
			detail.Quality.RFree = &v
		}
	}
	if dep := blocks.scalar("_pdbx_database_status.recvd_initial_deposition_date"); dep != "" {
		detail.DepositionDate = dep
	}

	entityDesc := make(map[string]string)
	for _, row := range blocks.rows("_entity") {
		eid := row["id"]
		desc := cleanCIFVal(row["pdbx_description"])
		if eid != "" && desc != "" {
			entityDesc[eid] = desc
		}
	}

	ssRanges := make([]struct {
		chain string
		start int
		end   int
		kind  string
	}, 0)

	for _, row := range blocks.rows("_struct_conf") {
		confType := row["conf_type_id"]
		if strings.HasPrefix(confType, "HELX") {
			ch := firstNonEmpty(cleanCIFVal(row["beg_auth_asym_id"]), cleanCIFVal(row["beg_label_asym_id"]))
			s, e1 := strconv.Atoi(row["beg_auth_seq_id"])
			e, e2 := strconv.Atoi(row["end_auth_seq_id"])
			if e1 == nil && e2 == nil {
				ssRanges = append(ssRanges, struct {
					chain string
					start int
					end   int
					kind  string
				}{ch, s, e, "helix"})
			}
		}
	}
	for _, row := range blocks.rows("_struct_sheet_range") {
		ch := firstNonEmpty(cleanCIFVal(row["beg_auth_asym_id"]), cleanCIFVal(row["beg_label_asym_id"]))
		s, e1 := strconv.Atoi(row["beg_auth_seq_id"])
		e, e2 := strconv.Atoi(row["end_auth_seq_id"])
		if e1 == nil && e2 == nil {
			ssRanges = append(ssRanges, struct {
				chain string
				start int
				end   int
				kind  string
			}{ch, s, e, "sheet"})
		}
	}

	lookupSS := func(ch string, authSeq int) string {
		for _, r := range ssRanges {
			if r.chain == ch && authSeq >= r.start && authSeq <= r.end {
				return r.kind
			}
		}
		return ""
	}

	// Read _pdbx_poly_seq_scheme if available to capture EVERY polymer residue including crystallographic unmodeled loops!
	type schemeEntry struct {
		asymID    string
		entityID  string
		monID     string
		seqID     int
		pdbSeqNum int
		authSeq   string
		insCode   string
		strandID  string
	}
	schemeByStrand := make(map[string][]schemeEntry)
	for _, row := range blocks.rows("_pdbx_poly_seq_scheme") {
		strand := firstNonEmpty(cleanCIFVal(row["pdb_strand_id"]), cleanCIFVal(row["asym_id"]))
		seqID, _ := strconv.Atoi(row["seq_id"])
		pdbNum, err := strconv.Atoi(cleanCIFVal(row["pdb_seq_num"]))
		if err != nil {
			pdbNum = seqID
		}
		ins := cleanCIFVal(row["pdb_ins_code"])
		schemeByStrand[strand] = append(schemeByStrand[strand], schemeEntry{
			asymID:    cleanCIFVal(row["asym_id"]),
			entityID:  cleanCIFVal(row["entity_id"]),
			monID:     cleanCIFVal(row["mon_id"]),
			seqID:     seqID,
			pdbSeqNum: pdbNum,
			authSeq:   cleanCIFVal(row["auth_seq_num"]),
			insCode:   ins,
			strandID:  strand,
		})
	}

	chainMap := make(map[string]*parsedChainRaw)
	var chainOrder []string
	resIndex := make(map[resKey]*rawResidue)
	byLabelSeq := make(map[string]map[int]*rawResidue)
	ligMap := make(map[rawLigandKey]*rawLigand)
	var ligOrder []rawLigandKey

	firstModel := ""
	for _, row := range blocks.rows("_atom_site") {
		modelNum := cleanCIFVal(row["pdbx_PDB_model_num"])
		if modelNum != "" {
			if firstModel == "" {
				firstModel = modelNum
			} else if modelNum != firstModel {
				continue
			}
		}
		altID := cleanCIFVal(row["label_alt_id"])
		if altID != "" && altID != "A" && altID != "1" {
			continue
		}

		group := row["group_PDB"]
		compID := firstNonEmpty(cleanCIFVal(row["auth_comp_id"]), cleanCIFVal(row["label_comp_id"]))
		authAsym := firstNonEmpty(cleanCIFVal(row["auth_asym_id"]), cleanCIFVal(row["label_asym_id"]), "A")
		labelAsym := firstNonEmpty(cleanCIFVal(row["label_asym_id"]), authAsym)
		entityID := firstNonEmpty(cleanCIFVal(row["label_entity_id"]), "1")
		labelSeq, _ := strconv.Atoi(cleanCIFVal(row["label_seq_id"]))
		authSeq, err := strconv.Atoi(cleanCIFVal(row["auth_seq_id"]))
		if err != nil {
			authSeq = labelSeq
		}
		insCode := cleanCIFVal(row["pdbx_PDB_ins_code"])
		atomName := firstNonEmpty(cleanCIFVal(row["label_atom_id"]), cleanCIFVal(row["auth_atom_id"]))
		elem := cleanCIFVal(row["type_symbol"])

		x, errX := strconv.ParseFloat(row["Cartn_x"], 64)
		y, errY := strconv.ParseFloat(row["Cartn_y"], 64)
		z, errZ := strconv.ParseFloat(row["Cartn_z"], 64)
		if errX != nil || errY != nil || errZ != nil {
			continue
		}
		bfac, _ := strconv.ParseFloat(row["B_iso_or_equiv"], 64)
		isHeavy := !strings.EqualFold(elem, "H") && !strings.EqualFold(elem, "D")
		pt := atomPoint{x: x, y: y, z: z, name: atomName, bfac: bfac, isHeavy: isHeavy}
		detail.AtomCount++

		if group == "ATOM" || (group == "HETATM" && labelSeq > 0 && IsStandardPolymerComp(compID)) {
			pc, ok := chainMap[authAsym]
			if !ok {
				pc = &parsedChainRaw{
					authAsymID:  authAsym,
					labelAsymID: labelAsym,
					entityID:    entityID,
					description: entityDesc[entityID],
					polymerType: "protein",
				}
				chainMap[authAsym] = pc
				chainOrder = append(chainOrder, authAsym)
				byLabelSeq[authAsym] = make(map[int]*rawResidue)
			}
			rk := resKey{chain: authAsym, authSeq: authSeq, insCode: insCode}
			rr, exists := resIndex[rk]
			if !exists {
				rr = &rawResidue{
					chainID:     authAsym,
					labelAsymID: labelAsym,
					entityID:    entityID,
					labelSeqID:  labelSeq,
					authSeqID:   authSeq,
					insCode:     insCode,
					compID:      compID,
					observed:    true,
					secondary:   lookupSS(authAsym, authSeq),
				}
				resIndex[rk] = rr
				if labelSeq > 0 {
					byLabelSeq[authAsym][labelSeq] = rr
				}
				pc.residues = append(pc.residues, rr)
			}
			rr.atoms = append(rr.atoms, pt)
			if atomName == "CA" {
				rr.caPos = &[3]float64{x, y, z}
			}
		} else if group == "HETATM" && !ignoredSolvent[strings.ToUpper(compID)] {
			lk := rawLigandKey{compID: strings.ToUpper(compID), chainID: authAsym, authSeqID: authSeq}
			lg, ok := ligMap[lk]
			if !ok {
				lg = &rawLigand{key: lk}
				ligMap[lk] = lg
				ligOrder = append(ligOrder, lk)
			}
			lg.atoms = append(lg.atoms, pt)
		}
	}

	// Reconcile observed residues with _pdbx_poly_seq_scheme when present so crystallographic missing density is exact.
	var chains []*parsedChainRaw
	for _, cid := range chainOrder {
		pc := chainMap[cid]
		if scheme, ok := schemeByStrand[cid]; ok && len(scheme) >= len(pc.residues) {
			full := make([]*rawResidue, 0, len(scheme))
			for _, se := range scheme {
				if obs, ok := byLabelSeq[cid][se.seqID]; ok {
					full = append(full, obs)
					continue
				}
				rk := resKey{chain: cid, authSeq: se.pdbSeqNum, insCode: se.insCode}
				if obs, ok := resIndex[rk]; ok {
					full = append(full, obs)
					continue
				}
				full = append(full, &rawResidue{
					chainID:     cid,
					labelAsymID: firstNonEmpty(se.asymID, pc.labelAsymID),
					entityID:    firstNonEmpty(se.entityID, pc.entityID),
					labelSeqID:  se.seqID,
					authSeqID:   se.pdbSeqNum,
					insCode:     se.insCode,
					compID:      se.monID,
					observed:    false,
					secondary:   lookupSS(cid, se.pdbSeqNum),
				})
			}
			pc.residues = full
		}
		chains = append(chains, pc)
	}

	var ligands []*rawLigand
	for _, lk := range ligOrder {
		ligands = append(ligands, ligMap[lk])
	}
	return detail, chains, ligands, nil
}

// inferSecondaryFromCA assigns α-helices and extended β-strands from Cα virtual bond geometry (Levitt & Greer P-SEA criteria)
// when header HELIX/SHEET records are omitted.
func inferSecondaryFromCA(residues []*rawResidue) {
	n := len(residues)
	if n < 5 {
		return
	}
	dist := func(a, b *[3]float64) float64 {
		dx := a[0] - b[0]
		dy := a[1] - b[1]
		dz := a[2] - b[2]
		return math.Sqrt(dx*dx + dy*dy + dz*dz)
	}

	marks := make([]string, n)
	for i := 0; i <= n-4; i++ {
		c0, c1, c2, c3 := residues[i].caPos, residues[i+1].caPos, residues[i+2].caPos, residues[i+3].caPos
		if c0 == nil || c1 == nil || c2 == nil || c3 == nil {
			continue
		}
		// Ensure contiguous peptide bond Cα–Cα ~ 3.8 Å
		if dist(c0, c1) > 4.3 || dist(c1, c2) > 4.3 || dist(c2, c3) > 4.3 {
			continue
		}
		d13 := dist(c0, c2)
		d14 := dist(c0, c3)
		tors := dihedralDeg(*c0, *c1, *c2, *c3)

		if d14 >= 4.6 && d14 <= 6.2 && d13 >= 5.0 && d13 <= 6.2 && tors >= 28 && tors <= 82 {
			marks[i+1] = "helix"
			marks[i+2] = "helix"
		} else if d14 >= 8.6 && d14 <= 11.6 && d13 >= 6.2 && (tors <= -85 || tors >= 145) {
			if marks[i+1] == "" {
				marks[i+1] = "sheet"
			}
			if marks[i+2] == "" {
				marks[i+2] = "sheet"
			}
		}
	}
	for i, m := range marks {
		if m != "" {
			residues[i].secondary = m
		}
	}
}

func dihedralDeg(p1, p2, p3, p4 [3]float64) float64 {
	b1 := [3]float64{p2[0] - p1[0], p2[1] - p1[1], p2[2] - p1[2]}
	b2 := [3]float64{p3[0] - p2[0], p3[1] - p2[1], p3[2] - p2[2]}
	b3 := [3]float64{p4[0] - p3[0], p4[1] - p3[1], p4[2] - p3[2]}
	cross := func(u, v [3]float64) [3]float64 {
		return [3]float64{
			u[1]*v[2] - u[2]*v[1],
			u[2]*v[0] - u[0]*v[2],
			u[0]*v[1] - u[1]*v[0],
		}
	}
	dot := func(u, v [3]float64) float64 {
		return u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
	}
	n1 := cross(b1, b2)
	n2 := cross(b2, b3)
	b2Len := math.Sqrt(dot(b2, b2))
	if b2Len < 1e-6 {
		return 0
	}
	m1 := cross([3]float64{b2[0] / b2Len, b2[1] / b2Len, b2[2] / b2Len}, n1)
	x := dot(n1, n2)
	y := dot(m1, n2)
	return math.Atan2(y, x) * 180.0 / math.Pi
}

// Minimal, zero-allocation-friendly CIF loop & key-value parser supporting single/double quotes and multiline ';' blocks.
type cifDoc struct {
	scalars map[string]string
	tables  map[string]cifTable
}

type cifTable struct {
	cols []string
	rows [][]string
}

func (d *cifDoc) scalar(key string) string {
	return cleanCIFVal(d.scalars[key])
}

func (d *cifDoc) firstScalar(keys ...string) string {
	for _, k := range keys {
		if v := d.scalar(k); v != "" {
			return v
		}
	}
	return ""
}

func (d *cifDoc) rows(category string) []map[string]string {
	tbl, ok := d.tables[category]
	if !ok {
		// Check if single-row category was stored as scalars
		prefix := category + "."
		single := make(map[string]string)
		for k, v := range d.scalars {
			if strings.HasPrefix(k, prefix) {
				single[strings.TrimPrefix(k, prefix)] = v
			}
		}
		if len(single) > 0 {
			return []map[string]string{single}
		}
		return nil
	}
	out := make([]map[string]string, 0, len(tbl.rows))
	for _, r := range tbl.rows {
		m := make(map[string]string, len(tbl.cols))
		for i, c := range tbl.cols {
			if i < len(r) {
				m[c] = r[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func parseCIFTables(text string) *cifDoc {
	tokens := tokenizeCIF(text)
	doc := &cifDoc{
		scalars: make(map[string]string),
		tables:  make(map[string]cifTable),
	}

	i := 0
	n := len(tokens)
	for i < n {
		tok := tokens[i]
		if tok == "loop_" {
			i++
			var fullCols []string
			for i < n && strings.HasPrefix(tokens[i], "_") {
				fullCols = append(fullCols, tokens[i])
				i++
			}
			if len(fullCols) == 0 {
				continue
			}
			cat := fullCols[0]
			if dot := strings.IndexByte(cat, '.'); dot > 0 {
				cat = cat[:dot]
			}
			shortCols := make([]string, len(fullCols))
			for idx, fc := range fullCols {
				if dot := strings.IndexByte(fc, '.'); dot > 0 {
					shortCols[idx] = fc[dot+1:]
				} else {
					shortCols[idx] = fc
				}
			}
			// Only materialize categories needed by BioScope to keep memory tiny even on giant mmCIF assemblies.
			keep := cat == "_atom_site" || cat == "_pdbx_poly_seq_scheme" || cat == "_struct_conf" ||
				cat == "_struct_sheet_range" || cat == "_entity"
			var rows [][]string
			width := len(shortCols)
			for i < n && !strings.HasPrefix(tokens[i], "_") && tokens[i] != "loop_" && !strings.HasPrefix(tokens[i], "data_") {
				if i+width > n {
					break
				}
				if keep {
					row := make([]string, width)
					copy(row, tokens[i:i+width])
					rows = append(rows, row)
				}
				i += width
			}
			if keep {
				doc.tables[cat] = cifTable{cols: shortCols, rows: rows}
			}
			continue
		}
		if strings.HasPrefix(tok, "_") {
			if i+1 < n {
				doc.scalars[tok] = tokens[i+1]
				i += 2
				continue
			}
		}
		i++
	}
	return doc
}

func tokenizeCIF(s string) []string {
	tokens := make([]string, 0, len(s)/8)
	n := len(s)
	i := 0
	atLineStart := true

	for i < n {
		c := s[i]
		if c == '\n' || c == '\r' {
			atLineStart = true
			i++
			continue
		}
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if atLineStart && c == ';' {
			// Multiline semicolon text field up to next "\n;"
			i++
			start := i
			for i < n {
				if s[i] == '\n' && i+1 < n && s[i+1] == ';' {
					tokens = append(tokens, strings.TrimSpace(s[start:i]))
					i += 2
					atLineStart = false
					break
				}
				i++
			}
			continue
		}
		atLineStart = false
		if c == '#' {
			for i < n && s[i] != '\n' {
				i++
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote := c
			i++
			start := i
			for i < n {
				if s[i] == quote && (i+1 == n || s[i+1] == ' ' || s[i+1] == '\t' || s[i+1] == '\r' || s[i+1] == '\n') {
					break
				}
				i++
			}
			tokens = append(tokens, "'" + s[start:i] + "'")
			if i < n && s[i] == quote {
				i++
			}
			continue
		}
		start := i
		for i < n && s[i] != ' ' && s[i] != '\t' && s[i] != '\r' && s[i] != '\n' {
			i++
		}
		tokens = append(tokens, s[start:i])
	}
	return tokens
}

func cleanCIFVal(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && ((v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"')) {
		return strings.TrimSpace(v[1 : len(v)-1])
	}
	if v == "." || v == "?" {
		return ""
	}
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
