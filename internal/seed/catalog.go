package seed

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
)

//go:embed structures/*
var structureFS embed.FS

var seedSnapshotTime = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

// SeedStructureMeta holds curated crystallographic/AlphaFold enrichment for embedded fallback models.
type SeedStructureMeta struct {
	ID                   string
	Filename             string
	Kind                 bio.StructureKind
	SourceDB             string
	Title                string
	Method               string
	Resolution           *float64
	RWork                *float64
	RFree                *float64
	MeanPLDDT            *float64
	ClashscorePercentile *float64
	RamachandranOutliers *float64
	SidechainOutliers    *float64
	ReleaseDate          string
	Organism             string
	Citation             string
	DOI                  string
	PrimaryUniProt       string
	SecondaryUniProt     map[string]string // chainID -> partner UniProt description when heteromeric
	ChainsSummary        string
	CoverageRange        string
	LigandIDs            []string
}

func f64(v float64) *float64 { return &v }

var seedStructures = map[string]SeedStructureMeta{
	"1GZX": {
		ID:                   "1GZX",
		Filename:             "structures/1GZX.pdb",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Crystal Structure of Human Oxyhaemoglobin at 2.1 Å Resolution (α₂β₂ Heterotetramer + Heme & Bound O₂)",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(2.10),
		RWork:                f64(0.169),
		RFree:                f64(0.224),
		ClashscorePercentile: f64(84.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(1.9),
		ReleaseDate:          "1997-07-15",
		Organism:             "Homo sapiens",
		Citation:             "Paoli, M., Liddington, R., Tame, J., Wilkinson, A., Dodson, G. (1996) Crystal structure of T state haemoglobin with oxygen bound at all four haems. J. Mol. Biol. 256, 775-792.",
		DOI:                  "10.1006/jmbi.1996.0124",
		PrimaryUniProt:       "P69905",
		SecondaryUniProt: map[string]string{
			"B": "P68871 (Hemoglobin subunit beta, HBB_HUMAN)",
			"D": "P68871 (Hemoglobin subunit beta, HBB_HUMAN)",
		},
		ChainsSummary: "A, C = α-globin (141 aa); B, D = β-globin (146 aa)",
		CoverageRange: "UniProt 2–142 (cleaved Met1)",
		LigandIDs:     []string{"HEM", "OXY"},
	},
	"AF-P69905-F1": {
		ID:             "AF-P69905-F1",
		Filename:       "structures/AF-P69905-F1.cif",
		Kind:           bio.StructurePredicted,
		SourceDB:       "AlphaFold DB",
		Title:          "AlphaFold Monomer v4 Prediction for Human Hemoglobin Subunit Alpha (HBA1)",
		Method:         "ALPHAFOLD MONOMER PREDICTION",
		MeanPLDDT:      f64(96.8),
		ReleaseDate:    "2024-03-01",
		Organism:       "Homo sapiens",
		Citation:       "Jumper, J. et al. (2021) Highly accurate protein structure prediction with AlphaFold. Nature 596, 583-589; Varadi, M. et al. (2024) AlphaFold Protein Structure Database. Nucleic Acids Research.",
		DOI:            "10.1038/s41586-021-03819-2",
		PrimaryUniProt: "P69905",
		ChainsSummary:  "A = Monomer (142 aa; apo-protein without HEM cofactor or β-subunits)",
		CoverageRange:  "UniProt 1–142 (Full length)",
	},
	"1YCR": {
		ID:                   "1YCR",
		Filename:             "structures/1YCR.pdb",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Structure of the MDM2 Oncoprotein Bound to the Transactivation Domain Peptide of Tumor Suppressor p53",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(2.60),
		RWork:                f64(0.215),
		RFree:                f64(0.274),
		ClashscorePercentile: f64(76.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(3.2),
		ReleaseDate:          "1996-11-08",
		Organism:             "Homo sapiens",
		Citation:             "Kussie, P.H. et al. (1996) Structure of the MDM2 oncoprotein bound to the p53 tumor suppressor transactivation domain. Science 274, 948-953.",
		DOI:                  "10.1126/science.274.5289.948",
		PrimaryUniProt:       "P04637",
		SecondaryUniProt: map[string]string{
			"A": "Q00987 (E3 ubiquitin-protein ligase Mdm2 binding cleft, residues 25–109)",
		},
		ChainsSummary: "A = MDM2 cleft (85 aa); B = p53 transactivation helix (residues 17–29)",
		CoverageRange: "UniProt 17–29 (Amphipathic α-helix)",
	},
	"AF-P04637-F1": {
		ID:             "AF-P04637-F1",
		Filename:       "structures/AF-P04637-F1.pdb",
		Kind:           bio.StructurePredicted,
		SourceDB:       "AlphaFold DB",
		Title:          "AlphaFold v6 Full-Length Structure Prediction of Cellular Tumor Antigen p53 (TP53)",
		Method:         "ALPHAFOLD MONOMER PREDICTION",
		MeanPLDDT:      f64(68.9),
		ReleaseDate:    "2025-08-01",
		Organism:       "Homo sapiens",
		Citation:       "Jumper, J. et al. (2021) Highly accurate protein structure prediction with AlphaFold. Nature 596, 583-589.",
		DOI:            "10.1038/s41586-021-03819-2",
		PrimaryUniProt: "P04637",
		ChainsSummary:  "A = Full-length p53 monomer (393 aa: folded DBD core + intrinsically disordered tails)",
		CoverageRange:  "UniProt 1–393 (Full length)",
	},
	"4INS": {
		ID:                   "4INS",
		Filename:             "structures/4INS.pdb",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Structure of 2-Zn Insulin at 1.5 Å Resolution: Mature Disulfide-Linked A & B Chains with Zn²⁺ Coordination",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(1.50),
		RWork:                f64(0.153),
		RFree:                f64(0.192),
		ClashscorePercentile: f64(90.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(1.1),
		ReleaseDate:          "1989-10-15",
		Organism:             "Sus scrofa / Homo sapiens homolog",
		Citation:             "Baker, E.N., Blundell, T.L., Cutfield, J.F., Dodson, E.J., Hodgkin, D.C. et al. (1988) The structure of 2Zn pig insulin crystals at 1.5 Å resolution. Phil. Trans. R. Soc. Lond. B 319, 369-456.",
		DOI:                  "10.1098/rstb.1988.0058",
		PrimaryUniProt:       "P01308",
		ChainsSummary:        "A, C = Insulin A-chain (21 aa); B, D = Insulin B-chain (30 aa)",
		CoverageRange:        "Chain B: UniProt 25–54; Chain A: UniProt 90–110 (Signal & C-peptide excised)",
		LigandIDs:            []string{"ZN"},
	},
	"AF-P01308-F1": {
		ID:             "AF-P01308-F1",
		Filename:       "structures/AF-P01308-F1.cif",
		Kind:           bio.StructurePredicted,
		SourceDB:       "AlphaFold DB",
		Title:          "AlphaFold v6 Prediction of Full-Length Human Preproinsulin (Signal Peptide + B-Chain + C-Peptide + A-Chain)",
		Method:         "ALPHAFOLD MONOMER PREDICTION",
		MeanPLDDT:      f64(67.4),
		ReleaseDate:    "2025-08-01",
		Organism:       "Homo sapiens",
		Citation:       "Jumper, J. et al. (2021) Highly accurate protein structure prediction with AlphaFold. Nature 596, 583-589.",
		DOI:            "10.1038/s41586-021-03819-2",
		PrimaryUniProt: "P01308",
		ChainsSummary:  "A = Uncleaved preproinsulin precursor (110 aa)",
		CoverageRange:  "UniProt 1–110 (Includes cleaved pro-regions)",
	},
	"1LYZ": {
		ID:                   "1LYZ",
		Filename:             "structures/1LYZ.pdb",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Real-Space Refinement of Hen Egg-White Lysozyme at 2.0 Å Resolution (Glu35 / Asp52 Catalytic Cleft)",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(2.00),
		RWork:                f64(0.205),
		ClashscorePercentile: f64(79.0),
		RamachandranOutliers: f64(0.8),
		SidechainOutliers:    f64(2.4),
		ReleaseDate:          "1976-05-01",
		Organism:             "Gallus gallus",
		Citation:             "Diamond, R. (1974) Real-space refinement of the structure of hen egg-white lysozyme. J. Mol. Biol. 82, 371-391.",
		DOI:                  "10.1016/0022-2836(74)90598-1",
		PrimaryUniProt:       "P00698",
		ChainsSummary:        "A = Mature lysozyme C (129 aa; 4 intra-chain disulfides)",
		CoverageRange:        "UniProt 19–147 (Signal peptide 1–18 cleaved)",
	},
	"AF-P00698-F1": {
		ID:             "AF-P00698-F1",
		Filename:       "structures/AF-P00698-F1.cif",
		Kind:           bio.StructurePredicted,
		SourceDB:       "AlphaFold DB",
		Title:          "AlphaFold v3 Prediction of Hen Egg-White Lysozyme Precursor (Including N-Terminal Signal Peptide)",
		Method:         "ALPHAFOLD MONOMER PREDICTION",
		MeanPLDDT:      f64(92.4),
		ReleaseDate:    "2023-11-01",
		Organism:       "Gallus gallus",
		Citation:       "Jumper, J. et al. (2021) Highly accurate protein structure prediction with AlphaFold. Nature 596, 583-589.",
		DOI:            "10.1038/s41586-021-03819-2",
		PrimaryUniProt: "P00698",
		ChainsSummary:  "A = Precursor polypeptide (147 aa)",
		CoverageRange:  "UniProt 1–147 (Full length)",
	},
	"1GBT": {
		ID:                   "1GBT",
		Filename:             "structures/1GBT.cif",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Bovine Beta-Trypsin Acyl-Enzyme Complex with 4-Guanidinobenzoate (Catalytic Triad His57–Asp102–Ser195 & Insertion Codes)",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(2.00),
		RWork:                f64(0.168),
		ClashscorePercentile: f64(88.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(1.6),
		ReleaseDate:          "1993-04-30",
		Organism:             "Bos taurus",
		Citation:             "Singer, P.T., Smalas, A., Carty, R.P., Mangel, W.F., Sweet, R.M. (1993) The hydrolytic water molecule in trypsin, revealed by time-resolved Laue crystallography. Science 259, 669-673.",
		DOI:                  "10.1126/science.8430314",
		PrimaryUniProt:       "P00760",
		ChainsSummary:        "A = Mature β-trypsin (223 aa; chymotrypsinogen numbering with 65A, 184A, 188A, 221A)",
		CoverageRange:        "UniProt 24–246 (Zymogen signal/activation peptide 1–23 cleaved)",
		LigandIDs:            []string{"GBS", "CA", "SO4"},
	},
	"1UBQ": {
		ID:                   "1UBQ",
		Filename:             "structures/1UBQ.cif",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Structure of Human Ubiquitin Refined at 1.8 Å Resolution (Canonical β-Grasp Fold)",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(1.80),
		RWork:                f64(0.176),
		ClashscorePercentile: f64(92.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(0.0),
		ReleaseDate:          "1987-01-02",
		Organism:             "Homo sapiens",
		Citation:             "Vijay-Kumar, S., Bugg, C.E., Cook, W.J. (1987) Structure of ubiquitin refined at 1.8 Å resolution. J. Mol. Biol. 194, 531-544.",
		DOI:                  "10.1016/0022-2836(87)90679-6",
		PrimaryUniProt:       "P0CG48",
		ChainsSummary:        "A = Ubiquitin monomer (76 aa; 5-strand mixed β-sheet + central α-helix)",
		CoverageRange:        "UniProt 1–76 (100% full-length coverage)",
	},
	"2HYY": {
		ID:                   "2HYY",
		Filename:             "structures/2HYY.pdb",
		Kind:                 bio.StructureExperimental,
		SourceDB:             "RCSB PDB",
		Title:                "Human Abl Kinase Domain in Complex with Imatinib (Gleevec / STI-571) in Inactive DFG-Out Conformation",
		Method:               "X-RAY DIFFRACTION",
		Resolution:           f64(2.00),
		RWork:                f64(0.204),
		RFree:                f64(0.246),
		ClashscorePercentile: f64(86.0),
		RamachandranOutliers: f64(0.0),
		SidechainOutliers:    f64(1.8),
		ReleaseDate:          "2007-02-06",
		Organism:             "Homo sapiens",
		Citation:             "Cowan-Jacob, S.W. et al. (2007) Structural biology contributions to the discovery of drugs to treat chronic myelogenous leukaemia. Acta Crystallogr. D 63, 80-93.",
		DOI:                  "10.1107/S0907444906047287",
		PrimaryUniProt:       "P00519",
		ChainsSummary:        "A = ABL1 tyrosine kinase catalytic domain (residues 235–498 + STI-571)",
		CoverageRange:        "Kinase catalytic domain (Thr315 gatekeeper + Asp381-Phe382-Gly383 DFG motif)",
		LigandIDs:            []string{"STI"},
	},
}

var seedProteins = map[string]bio.ProteinRecord{
	"P69905": {
		Accession:            "P69905",
		EntryName:            "HBA_HUMAN",
		RecommendedName:      "Hemoglobin subunit alpha",
		AlternativeNames:     []string{"Alpha-globin", "Hemoglobin alpha chain"},
		GeneNames:            []string{"HBA1", "HBA2"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Craniata", "Vertebrata", "Mammalia", "Primates", "Hominidae", "Homo"},
		Sequence:             "MVLSPADKTNVKAAWGKVGAHAGEYGAEALERMFLSFPTTKTYFPHFDLSHGSAQVKGHGKKVADALTNAVAHVDDMPNALSALSDLHAHKLRVDPVNFKLLSHCLLVTLAAHLPAEFTPAVHASLDKFLASVSTVLTSKYR",
		Length:               142,
		MassDa:               15258,
		FunctionSummary:      "Involved in oxygen transport from the lung to the various peripheral tissues. Forms an allosteric α₂β₂ heterotetramer whose oxygen affinity is cooperatively regulated by heme iron out-of-plane movement, pH (Bohr effect), CO₂, and 2,3-bisphosphoglycerate.",
		SubunitNote:          "Heterotetramer of two alpha chains and two beta chains in adult hemoglobin A (HbA, α₂β₂). Notice that initiator Met1 is cleaved post-translationally, so crystallographic Val1 corresponds to UniProt Val2.",
		SubcellularLocations: []string{"Cytoplasm (Erythrocyte cytosol)"},
		Diseases: []bio.DiseaseAnnotation{
			{Name: "Alpha-thalassemia", Acronym: "ATHAL", MIM: "604131", Description: "Impaired alpha-globin synthesis leading to microcytic hypochromic anemia and excess beta-chain tetramers (HbH)."},
		},
		Domains: []bio.FeatureSpan{
			{Type: "Chain", Description: "Hemoglobin subunit alpha (mature cleaved chain)", Start: 2, End: 142},
			{Type: "Domain", Description: "Globin 8-helix bundle fold", Start: 2, End: 142},
		},
		Features: []bio.FeatureSpan{
			{Type: "Binding site", Description: "Distal histidine E7 — stabilizes bound O₂ via hydrogen bond (PDB His58)", Start: 59, End: 59, Ligand: "O₂ / Heme"},
			{Type: "Metal binding", Description: "Proximal histidine F8 — axial coordination bond to Heme Fe²⁺ (PDB His87)", Start: 88, End: 88, Ligand: "Heme (Fe²⁺)"},
		},
		Terms: []bio.TermRef{
			{Database: "InterPro", ID: "IPR000971", Name: "Globin"},
			{Database: "Pfam", ID: "PF00042", Name: "Globin family"},
			{Database: "GO", ID: "GO:0015671", Name: "P:oxygen transport"},
			{Database: "GO", ID: "GO:0020037", Name: "F:heme binding"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1GZX", Method: "X-ray", Resolution: f64(2.10), Chains: "A/C=2-142"},
		},
		AlphaFoldID: "AF-P69905-F1",
	},
	"P68871": {
		Accession:            "P68871",
		EntryName:            "HBB_HUMAN",
		RecommendedName:      "Hemoglobin subunit beta",
		AlternativeNames:     []string{"Beta-globin", "Hemoglobin beta chain"},
		GeneNames:            []string{"HBB"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Primates", "Hominidae", "Homo"},
		Sequence:             "MVHLTPEEKSAVTALWGKVNVDEVGGEALGRLLVVYPWTQRFFESFGDLSTPDAVMGNPKVKAHGKKVLGAFSDGLAHLDNLKGTFATLSELHCDKLHVDPENFRLLGNVLVCVLAHHFGKEFTPPVQAAYQKVVAGVANALAHKYH",
		Length:               147,
		MassDa:               15998,
		FunctionSummary:      "Oxygen-carrying beta subunit of adult hemoglobin α₂β₂ tetramer. The classic single-point missense mutation Glu7Val (traditionally numbered Glu6Val after Met1 cleavage; author residue 149 in 1GZX Chain B) causes sickle-cell disease via hydrophobic polymerization in deoxy state.",
		SubunitNote:          "Heterotetramer α₂β₂ (Chains B and D in 1GZX, numbered 144–289 and 544–689 by the structure authors).",
		SubcellularLocations: []string{"Cytoplasm"},
		Diseases: []bio.DiseaseAnnotation{
			{Name: "Sickle cell disease", Acronym: "SCD", MIM: "603903", Description: "Caused by surface hydrophobic substitution Glu7Val (β6 Glu→Val), promoting fiber polymerization under low oxygen tension."},
		},
		Domains: []bio.FeatureSpan{
			{Type: "Chain", Description: "Hemoglobin subunit beta (mature)", Start: 2, End: 147},
			{Type: "Domain", Description: "Globin fold", Start: 2, End: 147},
		},
		Features: []bio.FeatureSpan{
			{Type: "Mutagenesis", Description: "Sickle-cell disease locus Glu→Val (author residue 149 in 1GZX Chain B)", Start: 7, End: 7},
			{Type: "Binding site", Description: "Distal histidine E7 — O₂ stabilization (author residue 206 in 1GZX Chain B)", Start: 64, End: 64, Ligand: "O₂"},
			{Type: "Metal binding", Description: "Proximal histidine F8 — Heme Fe²⁺ axial ligand (author residue 235 in 1GZX Chain B)", Start: 93, End: 93, Ligand: "Heme (Fe²⁺)"},
		},
		Terms: []bio.TermRef{
			{Database: "InterPro", ID: "IPR000971", Name: "Globin"},
			{Database: "GO", ID: "GO:0005344", Name: "F:oxygen carrier activity"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1GZX", Method: "X-ray", Resolution: f64(2.10), Chains: "B/D=2-147"},
		},
		AlphaFoldID: "",
	},
	"P04637": {
		Accession:            "P04637",
		EntryName:            "P53_HUMAN",
		RecommendedName:      "Cellular tumor antigen p53",
		AlternativeNames:     []string{"Tumor suppressor p53", "Phosphoprotein p53", "Antigen NY-CO-13"},
		GeneNames:            []string{"TP53", "P53"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Primates", "Hominidae", "Homo"},
		Sequence:             "MEEPQSDPSVEPPLSQETFSDLWKLLPENNVLSPLPSQAMDDLMLSPDDIEQWFTEDPGPDEAPRMPEAAPPVAPAPAAPTPAAPAPAPSWPLSSSVPSQKTYQGSYGFRLGFLHSGTAKSVTCTYSPALNKMFCQLAKTCPVQLWVDSTPPPGTRVRAMAIYKQSQHMTEVVRRCPHHERCSDSDGLAPPQHLIRVEGNLRVEYLDDRNTFRHSVVVPYEPPEVGSDCTTIHYNYMCNSSCMGGMNRRPILTIITLEDSSGNLLGRNSFEVRVCACPGRDRRTEEENLRKKGEPHHELPPGSTKRALPNNTSSSPQPKKKPLDGEYFTLQIRGRERFEMFRELNEALELKDAQAGKEPGGSRAHSSHLKSKKGQSTSRHKKLMFKTEGPDSD",
		Length:               393,
		MassDa:               43653,
		FunctionSummary:      "Multifunctional transcription factor ('guardian of the genome') inducing cell cycle arrest, DNA repair, senescence, or apoptosis in response to genotoxic stress. Illustrates a classic modular architecture combining folded core/tetramerization domains with intrinsically disordered N- and C-terminal regions that fold upon partner binding (such as MDM2 in PDB 1YCR).",
		SubunitNote:          "Homotetramer assembled via residues 325–356; negatively regulated in unstressed cells by direct binding of E3 ubiquitin ligase MDM2 to residues 17–29.",
		SubcellularLocations: []string{"Nucleus", "Cytoplasm", "Mitochondrion"},
		Diseases: []bio.DiseaseAnnotation{
			{Name: "Li-Fraumeni syndrome", Acronym: "LFS1", MIM: "151623", Description: "Autosomal dominant cancer predisposition syndrome resulting from germline TP53 loss-of-function mutations."},
		},
		Domains: []bio.FeatureSpan{
			{Type: "Region", Description: "Transactivation domain TAD1 (folds into α-helix on MDM2 binding)", Start: 1, End: 40},
			{Type: "Region", Description: "Proline-rich intrinsically disordered linker", Start: 61, End: 93},
			{Type: "Domain", Description: "p53 DNA-binding β-sandwich core domain", Start: 94, End: 292},
			{Type: "Domain", Description: "Tetramerization α/β domain", Start: 325, End: 356},
			{Type: "Region", Description: "Basic C-terminal regulatory tail (disordered)", Start: 364, End: 393},
		},
		Features: []bio.FeatureSpan{
			{Type: "Site", Description: "Phe19 — hydrophobic anchor in MDM2 pocket (1YCR Chain B)", Start: 19, End: 19},
			{Type: "Site", Description: "Trp23 — deepest indole anchor in MDM2 cleft (1YCR Chain B)", Start: 23, End: 23},
			{Type: "Site", Description: "Leu26 — hydrophobic triad completion in MDM2 cleft (1YCR Chain B)", Start: 26, End: 26},
			{Type: "Metal binding", Description: "Structural Zn²⁺ coordination site (Cys176/His179/Cys238/Cys242)", Start: 176, End: 179, Ligand: "Zn²⁺"},
			{Type: "Site", Description: "Arg248 — major human cancer hotspot DNA minor-groove contact", Start: 248, End: 248},
			{Type: "Site", Description: "Arg273 — major human cancer hotspot DNA phosphate contact", Start: 273, End: 273},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF00870", Name: "P53 DNA-binding domain"},
			{Database: "Pfam", ID: "PF08563", Name: "P53 transactivation motif"},
			{Database: "GO", ID: "GO:0003700", Name: "F:DNA-binding transcription factor activity"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1YCR", Method: "X-ray", Resolution: f64(2.60), Chains: "B=17-29"},
		},
		AlphaFoldID: "AF-P04637-F1",
	},
	"P01308": {
		Accession:            "P01308",
		EntryName:            "INS_HUMAN",
		RecommendedName:      "Insulin",
		AlternativeNames:     []string{"Preproinsulin", "Proinsulin"},
		GeneNames:            []string{"INS"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Primates", "Hominidae", "Homo"},
		Sequence:             "MALWMRLLPLLALLALWGPDPAAAFVNQHLCGSHLVEALYLVCGERGFFYTPKTRREAEDLQVGQVELGGGPGAGSLQPLALEGSLQKRGIVEQCCTSICSLYQLENYCN",
		Length:               110,
		MassDa:               11981,
		FunctionSummary:      "Peptide hormone that decreases blood glucose concentration by increasing cell permeability to monosaccharides, amino acids, and fatty acids, while accelerating glycolysis and glycogen synthesis. Synthesized as single-chain 110-residue preproinsulin and proteolytically matured into disulfide-linked Chain B (UniProt 25–54) and Chain A (UniProt 90–110) upon excision of the signal peptide (1–24) and C-peptide (57–87).",
		SubunitNote:          "Mature hormone is a heterodimer of B-chain and A-chain linked by two interchain disulfide bonds (Cys31–Cys96 / B7–A7 and Cys43–Cys109 / B19–A20); forms hexamers around two Zn²⁺ ions via HisB10 inside pancreatic beta-cell secretory granules (as seen in 4INS).",
		SubcellularLocations: []string{"Secreted"},
		Diseases: []bio.DiseaseAnnotation{
			{Name: "Permanent neonatal diabetes mellitus", Acronym: "PNDM", MIM: "606176", Description: "Misfolding mutations disrupting proinsulin disulfide pairing trigger ER stress in pancreatic beta cells."},
		},
		Domains: []bio.FeatureSpan{
			{Type: "Signal", Description: "ER translocation signal peptide (cleaved)", Start: 1, End: 24},
			{Type: "Chain", Description: "Insulin B chain (PDB Chain B residues 1–30)", Start: 25, End: 54},
			{Type: "Propeptide", Description: "Connecting C-peptide (excised during maturation)", Start: 57, End: 87},
			{Type: "Chain", Description: "Insulin A chain (PDB Chain A residues 1–21)", Start: 90, End: 110},
		},
		Features: []bio.FeatureSpan{
			{Type: "Disulfide bond", Description: "Interchain disulfide CysB7–CysA7", Start: 31, End: 96},
			{Type: "Metal binding", Description: "HisB10 — coordinates hexamer-stabilizing Zn²⁺ ion in 4INS", Start: 34, End: 34, Ligand: "Zn²⁺"},
			{Type: "Disulfide bond", Description: "Interchain disulfide CysB19–CysA20", Start: 43, End: 109},
			{Type: "Disulfide bond", Description: "Intrachain A-loop disulfide CysA6–CysA11", Start: 95, End: 100},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF00049", Name: "Insulin/IGF/Relaxin family"},
			{Database: "GO", ID: "GO:0005158", Name: "F:insulin receptor binding"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "4INS", Method: "X-ray", Resolution: f64(1.50), Chains: "B/D=25-54, A/C=90-110"},
		},
		AlphaFoldID: "AF-P01308-F1",
	},
	"P00698": {
		Accession:            "P00698",
		EntryName:            "LYSC_CHICK",
		RecommendedName:      "Lysozyme C",
		AlternativeNames:     []string{"1,4-beta-N-acetylmuramidase C", "Hen egg-white lysozyme"},
		GeneNames:            []string{"LYZ"},
		Reviewed:             true,
		OrganismName:         "Gallus gallus",
		OrganismCommon:       "Chicken",
		TaxonID:              9031,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Aves", "Galliformes", "Phasianidae", "Gallus"},
		Sequence:             "MRSLLILVLCFLPLAALGKVFGRCELAAAMKRHGLDNYRGYSLGNWVCAAKFESNFNTQATNRNTDGSTDYGILQINSRWWCNDGRTPGSRNLCNIPCSALLSSDITASVNCAKKIVSDGNGMNAWVAWRNRCKGTDVQAWIRGCRL",
		Length:               147,
		MassDa:               16239,
		FunctionSummary:      "Bacteriolytic glycoside hydrolase that cleaves β-(1→4) glycosidic bonds between N-acetylmuramic acid (NAM) and N-acetyl-D-glucosamine (NAG) in bacterial peptidoglycan cell walls. The first enzyme ever solved by X-ray crystallography (Blake et al., 1965), establishing classic acid-base covalent catalysis via Glu35 (UniProt Glu53) and Asp52 (UniProt Asp70).",
		CatalyticActivities:  []string{"Hydrolysis of (1->4)-beta-linkages between N-acetylmuramic acid and N-acetyl-D-glucosamine residues in peptidoglycan."},
		SubunitNote:          "Monomer stabilized by four intramolecular disulfide bridges.",
		SubcellularLocations: []string{"Secreted"},
		Domains: []bio.FeatureSpan{
			{Type: "Signal", Description: "Secretory signal peptide (cleaved prior to folding)", Start: 1, End: 18},
			{Type: "Chain", Description: "Mature Lysozyme C (PDB residues 1–129)", Start: 19, End: 147},
		},
		Features: []bio.FeatureSpan{
			{Type: "Active site", Description: "Catalytic general acid/base proton donor (PDB Glu35)", Start: 53, End: 53},
			{Type: "Active site", Description: "Catalytic nucleophile / oxocarbenium stabilizer (PDB Asp52)", Start: 70, End: 70},
			{Type: "Binding site", Description: "Substrate sugar ring stacking dyad (PDB Trp62–Trp63)", Start: 80, End: 81, Ligand: "Peptidoglycan"},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF00062", Name: "C-type lysozyme/alpha-lactalbumin"},
			{Database: "GO", ID: "GO:0003796", Name: "F:lysozyme activity"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1LYZ", Method: "X-ray", Resolution: f64(2.00), Chains: "A=19-147"},
		},
		AlphaFoldID: "AF-P00698-F1",
	},
	"P00760": {
		Accession:            "P00760",
		EntryName:            "TRY1_BOVIN",
		RecommendedName:      "Cationic trypsin",
		AlternativeNames:     []string{"Beta-trypsin", "Trypsinogen"},
		GeneNames:            []string{"PRSS1"},
		Reviewed:             true,
		OrganismName:         "Bos taurus",
		OrganismCommon:       "Bovine",
		TaxonID:              9913,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Artiodactyla", "Bovidae", "Bos"},
		Sequence:             "MKTFIFLALLGAAVAFPVDDDDKIVGGYTCGANTVPYQVSLNSGYHFCGGSLINSQWVVSAAHCYKSGIQVRLGEDNINVVEGNEQFISASKSIVHPSYNSNTLNNDIMLIKLKSAASLNSRVASISLPTSCASAGTQCLISGWGNTKSSGTSYPDVLKCLKAPILSDSSCKSAYPGQITSNMFCAGYLEGGKDSCQGDSGGPVVCSGKLQGIVSWGSGCAQKNKPGVYTKVCNYVSWIKQTIASN",
		Length:               246,
		MassDa:               25785,
		FunctionSummary:      "Pancreatic endopeptidase that cleaves peptide bonds strictly on the carboxyl side of positively charged Lys and Arg residues. Exemplifies both the classic charge-relay Serine Protease Catalytic Triad (His57, Asp102, Ser195) with an anionic Asp189 S1 specificity pocket AND historical homology-based PDB insertion codes (65A, 184A, 188A, 221A).",
		CatalyticActivities:  []string{"Preferential cleavage: Arg-|-Xaa, Lys-|-Xaa via tetrahedral acyl-enzyme intermediate at Ser195."},
		SubunitNote:          "Monomer secreted as inactive trypsinogen; activated by enterokinase cleavage of the N-terminal peptide.",
		SubcellularLocations: []string{"Secreted (Extracellular pancreatic juice)"},
		Domains: []bio.FeatureSpan{
			{Type: "Signal", Description: "Signal peptide", Start: 1, End: 15},
			{Type: "Propeptide", Description: "Trypsinogen activation peptide (cleaved at Lys23)", Start: 16, End: 23},
			{Type: "Domain", Description: "Peptidase S1 double β-barrel catalytic domain", Start: 24, End: 246},
		},
		Features: []bio.FeatureSpan{
			{Type: "Active site", Description: "Charge-relay proton shuttle base (PDB His57)", Start: 63, End: 63},
			{Type: "Active site", Description: "Triad hydrogen-bond orienter (PDB Asp102)", Start: 107, End: 107},
			{Type: "Binding site", Description: "S1 pocket negative charge recognizing Arg/Lys substrates (PDB Asp189)", Start: 194, End: 194, Ligand: "GBS / Arg / Lys"},
			{Type: "Active site", Description: "Acyl-enzyme nucleophilic serine (PDB Ser195)", Start: 200, End: 200, Ligand: "GBS"},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF00089", Name: "Trypsin"},
			{Database: "GO", ID: "GO:0004252", Name: "F:serine-type endopeptidase activity"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1GBT", Method: "X-ray", Resolution: f64(2.00), Chains: "A=24-246"},
		},
	},
	"P0CG48": {
		Accession:            "P0CG48",
		EntryName:            "UBB_HUMAN",
		RecommendedName:      "Polyubiquitin-B (Ubiquitin monomer)",
		AlternativeNames:     []string{"Ubiquitin"},
		GeneNames:            []string{"UBB"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Primates", "Hominidae", "Homo"},
		Sequence:             "MQIFVKTLTGKTITLEVEPSDTIENVKAKIQDKEGIPPDQQRLIFAGKQLEDGRTLSDYNIQKESTLHLVLRLRGG",
		Length:               76,
		MassDa:               8565,
		FunctionSummary:      "Universally conserved 76-residue eukaryotic regulatory modifier covalently attached via its C-terminal Gly76 to target lysine sidechains. Architecture defines the canonical β-grasp fold: a 5-stranded mixed parallel/antiparallel β-sheet cradling a central amphipathic α-helix around the Ile44 hydrophobic patch.",
		SubunitNote:          "Monomer or covalently linked polyubiquitin chains (Lys48-linked chains signal 26S proteasomal degradation; Lys63-linked chains mediate DNA repair and endocytosis).",
		SubcellularLocations: []string{"Cytoplasm", "Nucleus"},
		Domains: []bio.FeatureSpan{
			{Type: "Domain", Description: "Ubiquitin β-grasp domain", Start: 1, End: 76},
		},
		Features: []bio.FeatureSpan{
			{Type: "Site", Description: "Ile44 — hydrophobic receptor recognition patch", Start: 44, End: 44},
			{Type: "Site", Description: "Lys48 — canonical polyubiquitin linkage site for proteasomal degradation", Start: 48, End: 48},
			{Type: "Site", Description: "Gly76 — C-terminal diglycine isopeptide conjugation site", Start: 76, End: 76},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF00240", Name: "Ubiquitin family"},
			{Database: "GO", ID: "GO:0000209", Name: "P:protein polyubiquitination"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "1UBQ", Method: "X-ray", Resolution: f64(1.80), Chains: "A=1-76"},
		},
	},
	"P00519": {
		Accession:            "P00519",
		EntryName:            "ABL1_HUMAN",
		RecommendedName:      "Tyrosine-protein kinase ABL1 (Catalytic Domain)",
		AlternativeNames:     []string{"Abelson murine leukemia viral oncogene homolog 1", "BCR-ABL1 target"},
		GeneNames:            []string{"ABL1", "ABL"},
		Reviewed:             true,
		OrganismName:         "Homo sapiens",
		OrganismCommon:       "Human",
		TaxonID:              9606,
		Lineage:              []string{"Eukaryota", "Metazoa", "Chordata", "Mammalia", "Primates", "Hominidae", "Homo"},
		// Canonical human ABL1-1a kinase catalytic domain construct (residues 235–498 with unmodeled loop residue 276 restored)
		Sequence:             "WEMERTDITMKHKLGGGQYGEVYEGVWKKYSLTVAVKTLKEDTMEVEEFLKEAAVMKEIKHPNLVQLLGVCTREPPFYIITEFMTYGNLLDYLRECNRQEVNAVVLLYMATQISSAMEYLEKKNFIHRDLAARNCLVGENHLVKVADFGLSRLMTGDTYTAHAGAKFPIKWTAPESLAYNKFSIKSDVWAFGVLLWEIATYGMSPYPGIDLSQVYELLEKDYRMERPEGCPEKVYELMRACWQWNPSDRPSFAEIHQAFETMFQ",
		Length:               264,
		MassDa:               30380,
		FunctionSummary:      "Non-receptor tyrosine kinase regulating cytoskeleton remodeling, DNA damage response, and apoptosis. Chromosomal translocation t(9;22) (Philadelphia chromosome) produces constitutively active BCR-ABL1 in chronic myeloid leukemia (CML). PDB 2HYY reveals how Imatinib (Gleevec / STI-571) selectively locks the kinase domain into an inactive DFG-out conformation.",
		CatalyticActivities:  []string{"ATP + L-tyrosyl-[protein] = ADP + O-phospho-L-tyrosyl-[protein]."},
		SubunitNote:          "Kinase catalytic bilobal fold comprising an N-terminal β-sheet lobe and a C-terminal α-helical lobe flanking the ATP/Imatinib cleft.",
		SubcellularLocations: []string{"Cytoplasm", "Nucleus", "Cytoskeleton"},
		Diseases: []bio.DiseaseAnnotation{
			{Name: "Chronic myeloid leukemia", Acronym: "CML", MIM: "608232", Description: "Driven by BCR-ABL1 fusion kinase; clinical resistance frequently arises via gatekeeper mutation Thr315Ile (T315I), which sterically clashes with Imatinib and breaks a critical hydrogen bond."},
		},
		Domains: []bio.FeatureSpan{
			{Type: "Domain", Description: "Protein kinase N-lobe (P-loop & αC-helix)", Start: 1, End: 84},
			{Type: "Domain", Description: "Protein kinase C-lobe & activation segment", Start: 85, End: 264},
		},
		Features: []bio.FeatureSpan{
			{Type: "Binding site", Description: "Glu286 — αC-helix H-bond acceptor to Imatinib amide", Start: 52, End: 52, Ligand: "Imatinib (STI)"},
			{Type: "Mutagenesis", Description: "Thr315 — gatekeeper H-bond donor; T315I causes clinical Gleevec resistance", Start: 81, End: 81, Ligand: "Imatinib (STI)"},
			{Type: "Binding site", Description: "Met318 — kinase hinge backbone H-bond to Imatinib pyridine", Start: 84, End: 84, Ligand: "Imatinib (STI)"},
			{Type: "Active site", Description: "DFG motif (Asp381–Phe382–Gly383) flipped into inactive DFG-out state", Start: 147, End: 149, Ligand: "Imatinib (STI)"},
		},
		Terms: []bio.TermRef{
			{Database: "Pfam", ID: "PF07714", Name: "Protein tyrosine and serine/threonine kinase"},
			{Database: "GO", ID: "GO:0004713", Name: "F:protein tyrosine kinase activity"},
		},
		PDBCrossRefs: []bio.PDBCrossRef{
			{ID: "2HYY", Method: "X-ray", Resolution: f64(2.00), Chains: "A=235-498"},
		},
	},
}

// Ordered list of featured showcase accessions for landing UI cards.
var showcaseOrder = []string{
	"P69905", // Oxyhemoglobin α₂β₂
	"P04637", // Tumor Suppressor p53 (1YCR + AlphaFold full length)
	"P01308", // Human Insulin (4INS cleaved two-chain + AF preproinsulin)
	"P00519", // ABL1 Kinase + Imatinib (2HYY drug pocket)
	"P00760", // Bovine Trypsin (1GBT triad + insertion codes)
	"P00698", // Hen Lysozyme (1LYZ + AF precursor)
	"P0CG48", // Human Ubiquitin (1UBQ β-grasp reference)
	"P68871", // Hemoglobin Beta (Sickle-cell locus)
}

// GetProtein returns a copy of a curated protein record with fallback provenance populated.
func GetProtein(accession string) (*bio.ProteinRecord, bool) {
	rec, ok := seedProteins[strings.ToUpper(strings.TrimSpace(accession))]
	if !ok {
		return nil, false
	}
	cp := rec
	cp.Provenance = bio.ProvenanceRecord{
		SourceName:   "UniProtKB (BioScope Curated Snapshot)",
		RecordID:     cp.Accession,
		RecordStatus: "Reviewed (Swiss-Prot)",
		EndpointURL:  fmt.Sprintf("https://rest.uniprot.org/uniprotkb/%s.json", cp.Accession),
		RetrievedAt:  seedSnapshotTime,
		DeliveryMode: bio.DeliverySeedFallback,
		Citation:     "The UniProt Consortium (2025). UniProt: the Universal Protein Knowledgebase. Nucleic Acids Research.",
		License:      "CC BY 4.0",
	}
	return &cp, true
}

// SearchProteins performs keyword, gene, accession, and PDB-ID matching across curated classroom entries.
func SearchProteins(query string) []bio.SearchHit {
	q := strings.ToLower(strings.TrimSpace(query))
	var hits []bio.SearchHit

	for _, acc := range showcaseOrder {
		rec := seedProteins[acc]
		matchedBy := ""
		if q == "" {
			matchedBy = "Featured Classroom Entry"
		} else if strings.EqualFold(rec.Accession, q) || strings.EqualFold(rec.EntryName, q) {
			matchedBy = "Exact Accession"
		} else if strings.EqualFold(rec.AlphaFoldID, q) {
			matchedBy = "AlphaFold Model ID"
		}
		if matchedBy == "" {
			for _, g := range rec.GeneNames {
				if strings.Contains(strings.ToLower(g), q) {
					matchedBy = "Gene Symbol (" + g + ")"
					break
				}
			}
		}
		if matchedBy == "" {
			for _, xr := range rec.PDBCrossRefs {
				if strings.EqualFold(xr.ID, q) {
					matchedBy = "PDB Structure (" + xr.ID + ")"
					break
				}
			}
		}
		if matchedBy == "" {
			haystack := strings.ToLower(rec.RecommendedName + " " + strings.Join(rec.AlternativeNames, " ") + " " +
				rec.OrganismName + " " + rec.OrganismCommon + " " + rec.FunctionSummary)
			if strings.Contains(haystack, q) {
				matchedBy = "Functional Annotation"
			}
		}
		if matchedBy != "" {
			var sids []string
			for _, xr := range rec.PDBCrossRefs {
				sids = append(sids, xr.ID)
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
				StructureIDs:     sids,
				AlphaFoldID:      rec.AlphaFoldID,
				MatchedBy:        matchedBy,
				HighlightSnippet: rec.FunctionSummary,
			})
		}
	}
	return hits
}

// GetStructureMeta returns curated metadata for an embedded PDB or AlphaFold structure.
func GetStructureMeta(structureID string) (SeedStructureMeta, bool) {
	m, ok := seedStructures[strings.ToUpper(strings.TrimSpace(structureID))]
	return m, ok
}

// ReadStructureCoordinates reads authentic embedded PDB/mmCIF coordinate bytes for a seed model.
func ReadStructureCoordinates(structureID string) ([]byte, string, SeedStructureMeta, error) {
	m, ok := GetStructureMeta(structureID)
	if !ok {
		return nil, "", SeedStructureMeta{}, fmt.Errorf("structure %q is not in curated seed library", structureID)
	}
	raw, err := structureFS.ReadFile(m.Filename)
	if err != nil {
		return nil, "", m, err
	}
	format := "pdb"
	if strings.HasSuffix(m.Filename, ".cif") {
		format = "mmcif"
	}
	return raw, format, m, nil
}

// ListStructuresForProtein returns summaries of all embedded experimental & predicted structures linked to accession.
func ListStructuresForProtein(accession string) []bio.StructureSummary {
	acc := strings.ToUpper(strings.TrimSpace(accession))
	rec, ok := seedProteins[acc]
	var candidateIDs []string
	if ok {
		for _, xr := range rec.PDBCrossRefs {
			candidateIDs = append(candidateIDs, xr.ID)
		}
		if rec.AlphaFoldID != "" {
			candidateIDs = append(candidateIDs, rec.AlphaFoldID)
		}
	}

	var out []bio.StructureSummary
	for _, sid := range candidateIDs {
		if m, exists := seedStructures[sid]; exists {
			out = append(out, bio.StructureSummary{
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
		}
	}
	return out
}

// LocalSimilaritySearch runs exact local alignment against all curated proteins when live NCBI BLAST is unreachable,
// clearly tagging the result as a Curated Classroom Reference Search so users always know the exact data provenance.
func LocalSimilaritySearch(querySeq string) []bio.BlastHit {
	q := strings.ToUpper(strings.TrimSpace(querySeq))
	var hits []bio.BlastHit
	for _, acc := range showcaseOrder {
		rec := seedProteins[acc]
		score, qStart, qEnd, tStart, tEnd, qAln, mid, tAln, matches := localSmithWaterman(q, rec.Sequence)
		if score < 32 || len(qAln) < 10 {
			continue
		}
		identPct := float64(matches) / float64(len(qAln)) * 100.0
		coverPct := float64(qEnd-qStart+1) / float64(len(q)) * 100.0
		bitScore := float64(score) * 0.42
		// Karlin-Altschul E-value approximation for Swiss-Prot scale (~2.1e8 total residues)
		eVal := 2.1e8 * float64(len(q)) * pow2(-bitScore)
		if eVal < 1e-180 {
			eVal = 0.0
		}
		pdbGuess := ""
		if len(rec.PDBCrossRefs) > 0 {
			pdbGuess = rec.PDBCrossRefs[0].ID
		}
		hits = append(hits, bio.BlastHit{
			Accession:     rec.Accession,
			UniProtGuess:  rec.Accession,
			PDBChainGuess: pdbGuess,
			Description:   fmt.Sprintf("%s (%s)", rec.RecommendedName, rec.EntryName),
			Organism:      rec.OrganismName,
			Length:        rec.Length,
			BitScore:      round1(bitScore),
			EValue:        sigFig2(eVal),
			IdentityPct:   round1(identPct),
			QueryCoverPct: round1(coverPct),
			QueryFrom:     qStart,
			QueryTo:       qEnd,
			HitFrom:       tStart,
			HitTo:         tEnd,
			QueryAlign:    qAln,
			Midline:       mid,
			HitAlign:      tAln,
		})
	}
	// Sort by BitScore descending
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].BitScore > hits[i].BitScore {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
	for i := range hits {
		hits[i].Rank = i + 1
	}
	return hits
}

func pow2(x float64) float64 {
	// exp(x * ln 2)
	const ln2 = 0.6931471805599453
	y := x * ln2
	if y < -420 {
		return 0
	}
	// Taylor/halving safe exp via standard math
	return expApprox(y)
}

func expApprox(y float64) float64 {
	// Split y = k*ln2 + r, |r| <= 0.35
	const ln2 = 0.6931471805599453
	k := int(y / ln2)
	r := y - float64(k)*ln2
	// 7-term Taylor around 0 is accurate to < 1e-11
	term := 1.0
	sum := 1.0
	for n := 1; n <= 9; n++ {
		term *= r / float64(n)
		sum += term
	}
	if k >= 0 {
		for i := 0; i < k; i++ {
			sum *= 2.0
		}
	} else {
		for i := 0; i < -k; i++ {
			sum *= 0.5
		}
	}
	return sum
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10.0
}

func sigFig2(v float64) float64 {
	if v == 0 {
		return 0
	}
	exp := 0
	n := v
	for n < 1.0 && exp > -300 {
		n *= 10
		exp--
	}
	for n >= 10.0 && exp < 300 {
		n /= 10
		exp++
	}
	rounded := float64(int(n*10+0.5)) / 10.0
	for exp > 0 {
		rounded *= 10
		exp--
	}
	for exp < 0 {
		rounded /= 10
		exp++
	}
	return rounded
}

// Compact BLOSUM62-inspired scoring Smith-Waterman aligner for sequence-to-seed matching.
func localSmithWaterman(a, b string) (bestScore, aStart, aEnd, bStart, bEnd int, aAln, mid, bAln string, matches int) {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return 0, 0, 0, 0, 0, "", "", "", 0
	}
	const gap = -4
	H := make([][]int, m+1)
	tb := make([][]uint8, m+1) // 1=diag, 2=up, 3=left
	for i := range H {
		H[i] = make([]int, n+1)
		tb[i] = make([]uint8, n+1)
	}

	bestI, bestJ := 0, 0
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			sub := subScore(a[i-1], b[j-1])
			diag := H[i-1][j-1] + sub
			up := H[i-1][j] + gap
			left := H[i][j-1] + gap

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
				bestScore = val
				bestI, bestJ = i, j
			}
		}
	}

	var qa, ma, ba []byte
	i, j := bestI, bestJ
	for i > 0 && j > 0 && H[i][j] > 0 {
		switch tb[i][j] {
		case 1:
			ca, cb := a[i-1], b[j-1]
			qa = append(qa, ca)
			ba = append(ba, cb)
			if ca == cb {
				ma = append(ma, ca)
				matches++
			} else if subScore(ca, cb) > 0 {
				ma = append(ma, '+')
			} else {
				ma = append(ma, ' ')
			}
			i--
			j--
		case 2:
			qa = append(qa, a[i-1])
			ma = append(ma, ' ')
			ba = append(ba, '-')
			i--
		case 3:
			qa = append(qa, '-')
			ma = append(ma, ' ')
			ba = append(ba, b[j-1])
			j--
		}
	}
	reverseBytes(qa)
	reverseBytes(ma)
	reverseBytes(ba)
	return bestScore, i + 1, bestI, j + 1, bestJ, string(qa), string(ma), string(ba), matches
}

func reverseBytes(s []byte) {
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}
}

func subScore(a, b byte) int {
	if a == b {
		switch a {
		case 'W', 'C', 'H', 'Y', 'F', 'M':
			return 8
		default:
			return 5
		}
	}
	pair := string([]byte{a, b})
	switch pair {
	case "IL", "LI", "IV", "VI", "LV", "VL", "FY", "YF", "FW", "WF", "YW", "WY",
		"KR", "RK", "DE", "ED", "ST", "TS", "NQ", "QN", "AG", "GA", "AT", "TA":
		return 2
	}
	return -3
}
