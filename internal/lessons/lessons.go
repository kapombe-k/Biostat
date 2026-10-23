package lessons

import (
	"fmt"
	"strings"
)

// FocusResidue identifies a specific structural residue to highlight, ball-and-stick, and optionally label in 3D.
type FocusResidue struct {
	ChainID   string `json:"chain_id"`
	AuthSeqID int    `json:"auth_seq_id"`
	InsCode   string `json:"ins_code,omitempty"`
	Label     string `json:"label,omitempty"`
}

// SceneSpec is a declarative 3D molecular viewer state synchronized across Mol* (MVSJ) and 3Dmol.js.
type SceneSpec struct {
	StructureID     string         `json:"structure_id"`
	ProteinAcc      string         `json:"protein_accession"`
	ActiveChain     string         `json:"active_chain"`
	Representation  string         `json:"representation"` // "cartoon", "backbone", "ball_and_stick", "spacefill", "surface"
	ColorScheme     string         `json:"color_scheme"`   // "plddt", "secondary", "chain", "bfactor", "hydrophobicity"
	ShowSurface     bool           `json:"show_surface"`
	ShowLigands     bool           `json:"show_ligands"`
	FocusLigand     string         `json:"focus_ligand,omitempty"`
	FocusResidues   []FocusResidue `json:"focus_residues,omitempty"`
	CameraMode      string         `json:"camera_mode"` // "overview", "selection", "ligand"
	StepBannerTitle string         `json:"step_banner_title"`
}

// LessonStep represents one structured interactive teaching step within a lesson.
type LessonStep struct {
	Index           int       `json:"index"`
	Title           string    `json:"title"`
	ConceptTag      string    `json:"concept_tag"`
	BodyParagraphs  []string  `json:"body_paragraphs"`
	WhatToObserve   []string  `json:"what_to_observe"`
	ScientificNote  string    `json:"scientific_note"`
	Scene           SceneSpec `json:"scene"`
}

// QuizOption represents one selectable answer choice in an interactive check-for-understanding question.
type QuizOption struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Explanation string `json:"explanation"`
}

// QuizQuestion represents a structured multiple-choice question with immediate pedagogical feedback.
type QuizQuestion struct {
	ID            string       `json:"id"`
	Prompt        string       `json:"prompt"`
	Options       []QuizOption `json:"options"`
	CorrectOption string       `json:"correct_option"`
	ProofStepIdx  int          `json:"proof_step_idx"`
}

// QuizEvaluation contains server-rendered feedback for a submitted answer.
type QuizEvaluation struct {
	LessonSlug      string
	Question        QuizQuestion
	SelectedOption  string
	IsCorrect       bool
	SelectedExplain string
	CorrectExplain  string
}

// Lesson represents a complete guided computational structural biology module.
type Lesson struct {
	Slug              string         `json:"slug"`
	Order             int            `json:"order"`
	Title             string         `json:"title"`
	Subtitle          string         `json:"subtitle"`
	Category          string         `json:"category"`
	Difficulty        string         `json:"difficulty"`
	DurationMinutes   int            `json:"duration_minutes"`
	FeaturedStructure string         `json:"featured_structure"`
	FeaturedProtein   string         `json:"featured_protein"`
	LearningGoals     []string       `json:"learning_goals"`
	Steps             []LessonStep   `json:"steps"`
	Quiz              []QuizQuestion `json:"quiz"`
}

var catalog = []Lesson{
	{
		Slug:              "protein-structure-hierarchy",
		Order:             1,
		Title:             "The Four Levels of Protein Structure",
		Subtitle:          "From linear amino acid sequence to human oxyhemoglobin's cooperative α₂β₂ tetramer",
		Category:          "Foundations of Structural Biology",
		Difficulty:        "Beginner",
		DurationMinutes:   8,
		FeaturedStructure: "1GZX",
		FeaturedProtein:   "P69905",
		LearningGoals: []string{
			"Distinguish primary, secondary, tertiary, and quaternary structure directly in atomic 3D coordinates.",
			"Understand why PDB residue numbers differ from canonical UniProt positions after N-terminal Met1 cleavage.",
			"Inspect how proximal (His87) and distal (His58) histidines tune heme iron reactivity for reversible O₂ transport.",
		},
		Steps: []LessonStep{
			{
				Index:      0,
				Title:      "1. Primary Structure & Post-Translational Numbering",
				ConceptTag: "Primary Structure (1°)",
				BodyParagraphs: []string{
					"A protein's primary structure is the linear sequence of amino acids linked by planar covalent peptide bonds from the N-terminus to the C-terminus. Even a single amino acid change can alter physiology dramatically.",
					"In human adult hemoglobin (PDB 1GZX), compare the sequence strip below the viewer with UniProt P69905: translation starts with Met1, which is enzymatically cleaved before maturation, so crystal residue Val1 maps to UniProt Val2 (+1 offset). On Chain B (β-globin, numbered 144–289 by the authors), residue Glu149 corresponds to the classic β6 Glu→Val sickle-cell locus!",
				},
				WhatToObserve: []string{
					"Trace the continuous covalent Cα trace of Chain A without sidechain clutter.",
					"Locate Val1 (N-terminus of Chain A) and compare its PDB author number (1) against its UniProt index (V2) in the sequence strip.",
				},
				ScientificNote: "Never assume PDB residue k equals UniProt sequence position k—always inspect explicit sequence-to-structure alignments.",
				Scene: SceneSpec{
					StructureID:     "1GZX",
					ProteinAcc:      "P69905",
					ActiveChain:     "A",
					Representation:  "backbone",
					ColorScheme:     "chain",
					ShowLigands:     false,
					CameraMode:      "selection",
					StepBannerTitle: "Primary Covalent Backbone Trace (Chain A)",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 1, Label: "Val1 (N-terminus · UniProt V2)"},
						{ChainID: "B", AuthSeqID: 149, Label: "Glu149 (β6 Sickle-Cell Locus · UniProt E7)"},
					},
				},
			},
			{
				Index:      1,
				Title:      "2. Secondary Structure: The 8-Helix Globin Bundle",
				ConceptTag: "Secondary Structure (2°)",
				BodyParagraphs: []string{
					"Local backbone hydrogen bonds between main-chain carbonyl oxygens (C=O at residue i) and amide hydrogens (N–H at residue i+4) fold ~75% of each globin chain into right-handed α-helices (labeled A through H in Perutz's classic nomenclature).",
					"Notice that hemoglobin contains zero β-sheets: short non-helical loops connect adjacent helices so sidechains project outward from rigid cylindrical scaffolds.",
				},
				WhatToObserve: []string{
					"Observe rose-colored right-handed α-helices connected by slate-gray turns.",
					"Inspect Helix E (Distal His58) and Helix F (Proximal His87) flanking the central cavity.",
				},
				ScientificNote: "α-Helices contain 3.6 residues per turn with a pitch of 5.4 Å, aligning peptide dipoles along the helix axis.",
				Scene: SceneSpec{
					StructureID:     "1GZX",
					ProteinAcc:      "P69905",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowLigands:     true,
					CameraMode:      "overview",
					StepBannerTitle: "All-α Globin Secondary Architecture",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 58, Label: "His58 (Helix E7 Distal)"},
						{ChainID: "A", AuthSeqID: 87, Label: "His87 (Helix F8 Proximal)"},
					},
				},
			},
			{
				Index:      2,
				Title:      "3. Tertiary Fold & Prosthetic Heme-O₂ Microenvironment",
				ConceptTag: "Tertiary Structure (3°)",
				BodyParagraphs: []string{
					"Tertiary structure describes the 3D packing of secondary elements plus non-polymer prosthetic groups required for function. Free heme Fe²⁺ in water rapidly oxidizes irreversibly to Fe³⁺ (incapable of binding O₂) and binds toxic CO ~20,000× tighter than O₂.",
					"Globin's hydrophobic pocket prevents Fe²⁺ oxidation, Proximal His87 coordinates axially to iron (~2.1 Å), and Distal His58 forms a selective hydrogen bond with bent end-on O₂ (OXY) while sterically disfavoring linear CO.",
				},
				WhatToObserve: []string{
					"Zoom directly into the Heme (HEM) porphyrin ring and bound dioxygen (OXY) inside Chain A.",
					"Trace the axial bond from His87 sidechain Nε2 to iron on one face and O₂ stabilized near His58 on the opposite face.",
				},
				ScientificNote: "Non-protein prosthetic groups (HEM, FAD, Zn²⁺) are essential functional components captured directly in experimental coordinates.",
				Scene: SceneSpec{
					StructureID:     "1GZX",
					ProteinAcc:      "P69905",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowLigands:     true,
					FocusLigand:     "HEM",
					CameraMode:      "selection",
					StepBannerTitle: "Heme Fe²⁺ Coordination & Bound Dioxygen Pocket",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 87, Label: "His87 (Proximal Fe²⁺ Ligand)"},
						{ChainID: "A", AuthSeqID: 58, Label: "His58 (Distal O₂ H-Bond)"},
					},
				},
			},
			{
				Index:      3,
				Title:      "4. Quaternary Assembly: Cooperative α₂β₂ Heterotetramer",
				ConceptTag: "Quaternary Structure (4°)",
				BodyParagraphs: []string{
					"Quaternary structure emerges when multiple folded polypeptide subunits assemble into a functional complex. Adult hemoglobin comprises two α-chains (Chains A, C) and two β-chains (Chains B, D), carrying four heme groups ~25–40 Å apart.",
					"When O₂ binds one heme iron and pulls Fe²⁺ into the porphyrin plane, Helix F shifts by ~0.6 Å, reorganizing inter-subunit salt bridges across the α₁β₂ interface and increasing oxygen affinity at remaining subunits—producing hemoglobin's sigmoidal oxygen-dissociation curve.",
				},
				WhatToObserve: []string{
					"Rotate the complete 4-subunit α₂β₂ assembly with translucent molecular surface enabled.",
					"Identify the four distinct subunit colors and observe how all four heme pockets remain solvent-accessible yet insulated from one another.",
				},
				ScientificNote: "Allostery enables hemoglobin to load to >97% saturation in pulmonary capillaries and unload efficiently in metabolically active tissues.",
				Scene: SceneSpec{
					StructureID:     "1GZX",
					ProteinAcc:      "P69905",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "chain",
					ShowSurface:     true,
					ShowLigands:     true,
					CameraMode:      "overview",
					StepBannerTitle: "Complete α₂β₂ Heterotetramer & Subunit Interfaces",
				},
			},
		},
		Quiz: []QuizQuestion{
			{
				ID:     "q1-hierarchy",
				Prompt: "In oxyhemoglobin (PDB 1GZX), why does proximal histidine appear at residue 87 in Chain A's crystal structure but at position 88 in canonical UniProt entry P69905?",
				Options: []QuizOption{
					{ID: "a", Text: "X-ray diffraction systematically skips every N-terminal residue due to phase errors.", Explanation: "Incorrect: phase determination does not shift sequence numbering; N-terminal methionine is biologically cleaved inside red blood cell precursors."},
					{ID: "b", Text: "Initiator Met1 is cleaved post-translationally prior to crystallization, shifting mature PDB numbering by -1 relative to translated mRNA/UniProt sequence.", Explanation: "Correct! Methionine aminopeptidase removes Met1, making mature Val1 equal to UniProt Val2 and PDB His87 equal to UniProt His88."},
					{ID: "c", Text: "Chain A comes from a different species than Homo sapiens.", Explanation: "Incorrect: 1GZX is 100% human adult hemoglobin α₂β₂."},
				},
				CorrectOption: "b",
				ProofStepIdx:  0,
			},
			{
				ID:     "q2-heme",
				Prompt: "Which structural level best describes the assembly of two α-globin chains and two β-globin chains into a cooperative oxygen-transport complex?",
				Options: []QuizOption{
					{ID: "a", Text: "Secondary structure", Explanation: "Secondary structure refers only to local backbone hydrogen-bonding motifs (α-helices and β-strands) within a polypeptide."},
					{ID: "b", Text: "Tertiary structure", Explanation: "Tertiary structure describes the 3D fold of an individual subunit and its bound cofactor (comparable to monomeric myoglobin)."},
					{ID: "c", Text: "Quaternary structure", Explanation: "Correct! Multi-subunit stoichiometry (α₂β₂) and inter-chain allosteric interfaces define quaternary structure."},
				},
				CorrectOption: "c",
				ProofStepIdx:  3,
			},
		},
	},
	{
		Slug:              "secondary-structure-motifs",
		Order:             2,
		Title:             "Secondary Structures & Hydrophobic Cores",
		Subtitle:          "Dissecting α-helices, mixed β-sheets, reverse turns, and ubiquitin's signaling surface in 1UBQ",
		Category:          "Protein Folding Principles",
		Difficulty:        "Beginner",
		DurationMinutes:   7,
		FeaturedStructure: "1UBQ",
		FeaturedProtein:   "P0CG48",
		LearningGoals: []string{
			"Compare intra-helical (i → i+4) vs inter-strand β-sheet hydrogen-bonding architectures.",
			"Visualize how hydrophobic sidechains pack into an anhydrous core while polar sidechains face solvent.",
			"Locate Ubiquitin's Ile44 patch, Lys48 chain-linkage residue, and flexible C-terminal Gly76 diglycine tail.",
		},
		Steps: []LessonStep{
			{
				Index:      0,
				Title:      "1. Architecture of the Canonical β-Grasp Fold",
				ConceptTag: "α/β Topology",
				BodyParagraphs: []string{
					"Human ubiquitin (76 amino acids, 1.8 Å X-ray structure 1UBQ) is conserved across all eukaryotes with >96% sequence identity from yeast to humans. It defines the classic β-grasp superfold.",
					"Switch your gaze between the ribbon diagram and the sequence strip below: five β-strands assemble into a curved sheet that 'grasps' a diagonal 12-residue α-helix (Ile23–Glu34), yielding extreme thermal and proteolytic resilience.",
				},
				WhatToObserve: []string{
					"Count the five golden directional arrows (β-strands pointing N → C) cradling the rose-magenta α-helix.",
					"Notice 100% of all 76 UniProt residues are experimentally observed in 1UBQ.",
				},
				ScientificNote: "Arrowheads on β-ribbons always point toward the C-terminus, immediately revealing parallel vs antiparallel strand pairs!",
				Scene: SceneSpec{
					StructureID:     "1UBQ",
					ProteinAcc:      "P0CG48",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					CameraMode:      "overview",
					StepBannerTitle: "Ubiquitin 1.8 Å β-Grasp Architecture",
				},
			},
			{
				Index:      1,
				Title:      "2. N-Terminal β-Hairpin & Tight Reverse Turn",
				ConceptTag: "β-Sheets & Turns",
				BodyParagraphs: []string{
					"Look closely at residues Met1–Val17: Strand β1 (Gln2–Phe4–Val5–Lys6) reverses 180° across a compact 4-residue β-turn (Thr7–Leu8–Thr9–Gly10) into antiparallel Strand β2 (Lys11–Ile13–Thr15).",
					"Glycine (Gly10) lacks a β-carbon sidechain, granting Ramachandran φ/ψ torsional freedom essential for sharp polypeptide chain reversals without steric clash.",
				},
				WhatToObserve: []string{
					"Notice how Strands β1 and β2 point in opposite directions (antiparallel), whereas Strands β1 and β5 run in the same direction (parallel)!",
					"Inspect Gly10 sitting right at the apex of the tight hairpin bend.",
				},
				ScientificNote: "Mixed β-sheets combine both crisp linear H-bonds (antiparallel) and slightly angled H-bonds (parallel) in a single cooperative sheet.",
				Scene: SceneSpec{
					StructureID:     "1UBQ",
					ProteinAcc:      "P0CG48",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					CameraMode:      "selection",
					StepBannerTitle: "Antiparallel β1–β2 Hairpin & Gly10 Turn Apex",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 4, Label: "Phe4 (Strand β1)"},
						{ChainID: "A", AuthSeqID: 10, Label: "Gly10 (Flexible β-Turn Apex)"},
						{ChainID: "A", AuthSeqID: 13, Label: "Ile13 (Strand β2)"},
					},
				},
			},
			{
				Index:      2,
				Title:      "3. The Hydrophobic Effect & Amphipathic Core Packing",
				ConceptTag: "Thermodynamics of Folding",
				BodyParagraphs: []string{
					"Coloring every residue by Kyte-Doolittle hydrophobicity (amber = nonpolar hydrophobic; cyan = polar/charged) reveals the primary thermodynamic driver of globular folding: burying nonpolar sidechains away from water maximizes solvent entropy.",
					"Notice that every fourth residue along the central α-helix (Ile23, Val26, Ile30) faces inward into the dry hydrophobic core against Ile3, Val5, and Ile13, while charged Lys27, Lys29, Asp32, and Glu34 face outward into water!",
				},
				WhatToObserve: []string{
					"Rotate around the α-helix to see its two distinct faces: amber nonpolar interior vs cyan charged exterior (amphipathicity).",
					"Locate solvent-exposed Ile44 on the β-sheet outer surface—because exposed hydrophobic patches are energetically costly, evolution preserves them specifically as high-affinity binding interfaces!",
				},
				ScientificNote: "Exposed hydrophobic patches on soluble proteins strongly predict protein-protein interaction hot spots (like Ubiquitin's Ile44 patch).",
				Scene: SceneSpec{
					StructureID:     "1UBQ",
					ProteinAcc:      "P0CG48",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "hydrophobicity",
					CameraMode:      "selection",
					StepBannerTitle: "Amphipathic α-Helix Core vs Exposed Ile44 Patch",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 23, Label: "Ile23 (Buried Helix Core)"},
						{ChainID: "A", AuthSeqID: 26, Label: "Val26 (Buried Helix Core)"},
						{ChainID: "A", AuthSeqID: 30, Label: "Ile30 (Buried Helix Core)"},
						{ChainID: "A", AuthSeqID: 44, Label: "Ile44 (Exposed Recognition Patch)"},
					},
				},
			},
			{
				Index:      3,
				Title:      "4. Functional Mobility: Lys48 & C-Terminal Gly76 Tail",
				ConceptTag: "Crystallographic B-Factors",
				BodyParagraphs: []string{
					"Coloring by experimental crystallographic B-factor (blue = rigid well-ordered atoms; red = mobile high-displacement atoms) turns a static crystal model into a map of local conformational dynamics!",
					"Notice that the β-grasp core is deep blue (<12 Å²), whereas the C-terminal tail (Leu73–Arg74–Gly75–Gly76) glows bright coral-red: this flexible tether must reach E1/E2/E3 ligase active sites to form an isopeptide bond with substrate lysines such as Lys48!",
				},
				WhatToObserve: []string{
					"Compare the rigid blue core against the mobile red C-terminal diglycine tail ending at Gly76.",
					"Locate surface-accessible Lys48, whose ε-amino group forms proteasomal degradation polyubiquitin chains.",
				},
				ScientificNote: "In experimental X-ray structures, column 61–66 stores B-factors (atomic displacement in Å²)—higher values mean GREATER thermal/conformational mobility (the exact opposite direction of AlphaFold pLDDT!).",
				Scene: SceneSpec{
					StructureID:     "1UBQ",
					ProteinAcc:      "P0CG48",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "bfactor",
					CameraMode:      "overview",
					StepBannerTitle: "B-Factor Dynamics: Rigid Core vs Flexible Gly76 Tail",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 48, Label: "Lys48 (Proteasomal Chain Site)"},
						{ChainID: "A", AuthSeqID: 76, Label: "Gly76 (Flexible C-Terminal Tail)"},
					},
				},
			},
		},
		Quiz: []QuizQuestion{
			{
				ID:     "q1-ss",
				Prompt: "Why are residues Ile23, Val26, and Ile30 spaced roughly 3–4 positions apart in Ubiquitin's primary sequence yet aligned along the exact same buried face in 3D?",
				Options: []QuizOption{
					{ID: "a", Text: "Because an α-helix completes one 360° turn every ~3.6 amino acids, placing i, i+3, and i+4 on the same radial side of the helix cylinder.", Explanation: "Correct! Amphipathic helices alternate hydrophobic residues every 3–4 positions to match the 3.6-residues-per-turn helical periodicity."},
					{ID: "b", Text: "Because β-strands coil tightly with 3.6 residues per turn.", Explanation: "Incorrect: β-strands are extended (~2 residues per repeat alternating 180° opposite faces i, i+2, i+4)."},
					{ID: "c", Text: "Disulfide bonds pull Ile23 and Ile30 together.", Explanation: "Incorrect: Ubiquitin contains zero cysteines and zero disulfide bonds."},
				},
				CorrectOption: "a",
				ProofStepIdx:  2,
			},
			{
				ID:     "q2-bfactor",
				Prompt: "In an experimental X-ray structure like 1UBQ, what does a HIGH crystallographic B-factor at C-terminal Gly76 indicate?",
				Options: []QuizOption{
					{ID: "a", Text: "Highest machine-learning prediction confidence (>90%).", Explanation: "Careful! That describes AlphaFold pLDDT—in X-ray crystallography, high B-factors mean greater atomic positional uncertainty/mobility."},
					{ID: "b", Text: "Elevated local thermal vibration / conformational flexibility of the C-terminal tail.", Explanation: "Correct! High B-factors reflect smeared electron density where flexible loops or tails sample multiple conformations in the crystal lattice."},
				},
				CorrectOption: "b",
				ProofStepIdx:  3,
			},
		},
	},
	{
		Slug:              "enzyme-active-sites-and-inhibitors",
		Order:             3,
		Title:             "Active Sites, Insertion Codes & Drug Resistance",
		Subtitle:          "Catalytic triads in Trypsin (1GBT) and Imatinib resistance in BCR-ABL1 Kinase (2HYY)",
		Category:          "Enzymology & Structure-Based Drug Design",
		Difficulty:        "Intermediate",
		DurationMinutes:   10,
		FeaturedStructure: "1GBT",
		FeaturedProtein:   "P00760",
		LearningGoals: []string{
			"Identify how residues distant in linear sequence converge in 3D to form Trypsin's His57–Asp102–Ser195 catalytic triad.",
			"Handle historical PDB insertion codes (65A, 184A, 188A, 221A) cleanly without misnumbering downstream residues.",
			"Inspect the atomic mechanism of Imatinib (Gleevec) binding and T315I clinical resistance in ABL1 kinase (PDB 2HYY).",
		},
		Steps: []LessonStep{
			{
				Index:      0,
				Title:      "1. Convergent 3D Geometry of the Serine Protease Triad",
				ConceptTag: "Enzyme Catalysis",
				BodyParagraphs: []string{
					"Enzyme active sites assemble sidechains separated by dozens or hundreds of positions in primary sequence into sub-Ångström chemical alignment. In bovine β-trypsin (PDB 1GBT), His57, Asp102, and Ser195 sit far apart on the sequence strip yet form a single hydrogen-bonded charge-relay network.",
					"Asp102 electrostatically orients His57 so His57 acts as a general base abstracting the hydroxyl proton from Ser195, turning Ser195 Oγ into a potent alkoxide nucleophile that attacks peptide carbonyl carbons—trapped here covalently by acyl-enzyme inhibitor 4-guanidinobenzoate (GBS)!",
				},
				WhatToObserve: []string{
					"Trace Asp102 → His57 → Ser195 → covalent inhibitor GBS right inside the active-site cleft.",
					"Notice on the sequence strip below that His57, Asp102, and Ser195 sit in completely different regions of the linear chain.",
				},
				ScientificNote: "The exact same Asp–His–Ser triad geometry evolved independently in trypsin, subtilisin, and α/β-hydrolases—a textbook case of convergent evolution at the atomic scale.",
				Scene: SceneSpec{
					StructureID:     "1GBT",
					ProteinAcc:      "P00760",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowLigands:     true,
					FocusLigand:     "GBS",
					CameraMode:      "selection",
					StepBannerTitle: "Trypsin Charge-Relay Catalytic Triad + Acyl-Enzyme Inhibitor",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 57, Label: "His57 (General Base)"},
						{ChainID: "A", AuthSeqID: 102, Label: "Asp102 (Triad Orienter)"},
						{ChainID: "A", AuthSeqID: 195, Label: "Ser195 (Acyl-Enzyme Nucleophile)"},
					},
				},
			},
			{
				Index:      1,
				Title:      "2. S1 Specificity Pocket (Asp189) & Homology Insertion Codes",
				ConceptTag: "Coordinate Numbering Rigor",
				BodyParagraphs: []string{
					"Why does trypsin cleave strictly after positively charged Lys/Arg, whereas chymotrypsin cleaves bulky aromatics? At the bottom of the deep S1 pocket sits negatively charged Asp189 forming a salt bridge directly with the positive guanidinium headgroup of GBS!",
					"Look right next to Asp189 on the sequence strip: you will see residues Tyr184A and Lys188A carrying letter insertion codes! Structural biologists number serine proteases against chymotrypsinogen so catalytic residues ALWAYS stay His57/Asp102/Ser195; extra loop residues in homologs receive insertion letters (184, 184A, 185).",
				},
				WhatToObserve: []string{
					"Locate Asp189 directly coordinating the distal nitrogen atoms of inhibitor GBS at the base of the pocket.",
					"Inspect Tyr184A and Lys188A in both 3D space and on the sequence strip (marked with amber insertion badges).",
				},
				ScientificNote: "Treating PDB residue IDs strictly as integers discards insertion codes and merges distinct amino acids (184 and 184A)—always key structural residues by (chain, auth_seq_id, ins_code)!",
				Scene: SceneSpec{
					StructureID:     "1GBT",
					ProteinAcc:      "P00760",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowLigands:     true,
					FocusLigand:     "GBS",
					CameraMode:      "selection",
					StepBannerTitle: "Asp189 S1 Specificity Pocket & Insertion Codes 184A / 188A",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 189, Label: "Asp189 (Anionic S1 Specificity)"},
						{ChainID: "A", AuthSeqID: 184, InsCode: "A", Label: "Tyr184A (Insertion Code Residue)"},
						{ChainID: "A", AuthSeqID: 188, InsCode: "A", Label: "Lys188A (Insertion Code Residue)"},
					},
				},
			},
			{
				Index:      2,
				Title:      "3. Targeted Kinase Inhibition: Imatinib (Gleevec) in ABL1",
				ConceptTag: "Rational Drug Discovery",
				BodyParagraphs: []string{
					"Now switch live to human ABL1 tyrosine kinase bound to Imatinib/Gleevec (STI, PDB 2HYY at 2.0 Å)—the landmark rational cancer drug for BCR-ABL1+ chronic myeloid leukemia.",
					"Active kinases require their conserved Asp381–Phe382–Gly383 ('DFG') motif pointing into the cleft to chelate ATP-Mg²⁺. Imatinib threads deep into an allosteric back pocket, trapping ABL1 in a catalytically dead 'DFG-out' conformation that closely related SRC kinases resist adopting—explaining Imatinib's remarkable selectivity!",
				},
				WhatToObserve: []string{
					"Observe drug ligand STI (ball-and-stick) spanning from the kinase hinge all the way past Glu286 into the hydrophobic DFG-out pocket.",
					"Locate Phe382 flipped outward across the cleft to make room for Imatinib's piperazinyl-benzamide tail.",
				},
				ScientificNote: "Also inspect residue 276 on the sequence strip: notice its dashed tile ('Unmodeled') where flexible β3–αC loop density was unresolvable in the crystal.",
				Scene: SceneSpec{
					StructureID:     "2HYY",
					ProteinAcc:      "P00519",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowSurface:     true,
					ShowLigands:     true,
					FocusLigand:     "STI",
					CameraMode:      "selection",
					StepBannerTitle: "ABL1 Kinase + Imatinib (STI-571) DFG-Out Pocket",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 286, Label: "Glu286 (αC-Helix H-Bond)"},
						{ChainID: "A", AuthSeqID: 381, Label: "Asp381 (DFG Motif)"},
						{ChainID: "A", AuthSeqID: 382, Label: "Phe382 (Flipped Out)"},
					},
				},
			},
			{
				Index:      3,
				Title:      "4. Clinical Resistance Atom-by-Atom: Gatekeeper T315I",
				ConceptTag: "Mutational Escape",
				BodyParagraphs: []string{
					"Zoom directly onto Thr315—the kinase 'gatekeeper' residue guarding entrance to the deep hydrophobic pocket in PDB 2HYY. Thr315's hydroxyl group donates a crucial short hydrogen bond directly to Imatinib's secondary amine nitrogen (<3.0 Å).",
					"In relapsed CML patients carrying the Thr315Ile (T315I) point mutation, replacing compact polar Threonine with bulky branched Isoleucine simultaneously deletes that stabilizing hydrogen bond AND creates a hard van der Waals steric collision right through Imatinib's scaffold—inspiring third-generation inhibitor Ponatinib designed with a slim carbon-carbon triple bond to clear Ile315!",
				},
				WhatToObserve: []string{
					"Inspect how tightly Thr315 packs right against the central ring system of Imatinib (STI), leaving zero room for an extra methyl-ethyl sidechain.",
					"Compare nearby hinge residue Met318 anchoring Imatinib's pyridine ring via main-chain hydrogen bonding.",
				},
				ScientificNote: "3D structural inspection explains clinical resistance mechanisms that linear sequence tables alone could never reveal.",
				Scene: SceneSpec{
					StructureID:     "2HYY",
					ProteinAcc:      "P00519",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					ShowLigands:     true,
					FocusLigand:     "STI",
					CameraMode:      "selection",
					StepBannerTitle: "Gatekeeper Thr315 Steric & H-Bond Contact with Imatinib",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 315, Label: "Thr315 (Gatekeeper · T315I Site)"},
						{ChainID: "A", AuthSeqID: 318, Label: "Met318 (Hinge H-Bond)"},
					},
				},
			},
		},
		Quiz: []QuizQuestion{
			{
				ID:     "q1-insertion",
				Prompt: "Why do crystal structures of serine proteases like bovine trypsin (1GBT) contain residue labels such as '184A' and '188A'?",
				Options: []QuizOption{
					{ID: "a", Text: "They denote alternate conformations of a single amino acid sidechain.", Explanation: "Alternate sidechain rotamers use altLoc column 17 (A/B), whereas insertion codes live in column 27 and denote additional consecutive sequence residues."},
					{ID: "b", Text: "They preserve conserved family numbering (chymotrypsinogen system) across loop insertions so catalytic His57, Asp102, and Ser195 keep identical numbers.", Explanation: "Correct! Insertion codes allow homologous active-site residues to retain consistent literature numbering despite variable loop lengths."},
				},
				CorrectOption: "b",
				ProofStepIdx:  1,
			},
			{
				ID:     "q2-t315i",
				Prompt: "Based on crystal structure 2HYY, why does the gatekeeper substitution Thr315Ile (T315I) confer clinical resistance to Imatinib?",
				Options: []QuizOption{
					{ID: "a", Text: "Thr315 sits 40 Å away on the exterior surface and blocks nuclear import.", Explanation: "Incorrect: Thr315 directly contacts Imatinib inside the ATP/back-pocket cleft (<3.1 Å)."},
					{ID: "b", Text: "Replacing polar Thr315 with bulkier nonpolar Ile eliminates a key hydrogen bond to Imatinib and introduces a direct steric clash in the gatekeeper channel.", Explanation: "Correct! Both loss of the Thr315 Oγ1 hydrogen bond and steric protrusion of Ile's Cδ1 methyl block Imatinib binding."},
				},
				CorrectOption: "b",
				ProofStepIdx:  3,
			},
		},
	},
	{
		Slug:              "blast-homology-and-evalues",
		Order:             4,
		Title:             "BLAST E-Values, Coverage & Homology Transfer",
		Subtitle:          "How statistical sequence alignment connects unknown sequences to 3D structures without over-interpreting hits",
		Category:          "Sequence Bioinformatics",
		Difficulty:        "Intermediate",
		DurationMinutes:   8,
		FeaturedStructure: "1LYZ",
		FeaturedProtein:   "P00698",
		LearningGoals: []string{
			"Interpret BLAST E-values vs Bit Scores vs Percent Identity using Karlin-Altschul statistics.",
			"Recognize why structural folds evolve far more slowly than primary sequences ('twilight zone' homology).",
			"Avoid the domain-chimerism trap by always checking Query Coverage % and catalytic site conservation in 3D.",
		},
		Steps: []LessonStep{
			{
				Index:      0,
				Title:      "1. Percent Identity Is Not Statistics: Why E-Values Matter",
				ConceptTag: "Karlin-Altschul Theory",
				BodyParagraphs: []string{
					"A common beginner mistake is judging BLAST hits solely by Percent Identity ('90% identity must be homologous!'). An alignment of 9 out of 10 amino acids gives 90% identity yet easily occurs dozens of times purely by chance in a 200-million-residue database!",
					"BLAST's Expect value (E = K · m · n · 2^{-S'}) estimates how many alignments with Bit Score ≥ S' would appear randomly given query length m and database size n. E = 1e-45 means practically zero probability of chance similarity; E = 2.4 means two or more random hits are expected by pure luck!",
				},
				WhatToObserve: []string{
					"Inspect Hen Egg-White Lysozyme (1LYZ, 129 mature residues): full-length alignment yields Bit Score >250 and E < 1e-80, whereas a 12-residue motif yields larger E-values even at 100% identity.",
					"Try running a live or reference search on the BLAST tab of any workspace using full vs truncated sequences!",
				},
				ScientificNote: "Because E scales linearly with database size n, searching Swiss-Prot (~570k curated proteins) produces ~100× smaller E-values for identical alignments than searching massive uncurated nr!",
				Scene: SceneSpec{
					StructureID:     "1LYZ",
					ProteinAcc:      "P00698",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					CameraMode:      "overview",
					StepBannerTitle: "Hen Egg-White Lysozyme (1LYZ) — Conserved Glycosidase Fold",
				},
			},
			{
				Index:      1,
				Title:      "2. Verifying Enzyme Function: Catalytic Residue Conservation",
				ConceptTag: "Functional Annotation",
				BodyParagraphs: []string{
					"Significant E-values (< 1e-5) establish evolutionary homology (common ancestry), NOT necessarily identical catalytic chemistry! For example, mammalian α-lactalbumin shares ~38% identity and an almost identical 3D fold with lysozyme (1LYZ), yet lost bacteriolytic activity because catalytic Glu35 was mutated!",
					"Zoom into Lysozyme's deep peptidoglycan cleft below: true muramidase catalysis requires BOTH Glu35 (proton donor in a hydrophobic pKa-elevated environment) AND Asp52 (ionized nucleophile/stabilizer across the cleft). Always inspect whether active-site residues are conserved in your alignment!",
				},
				WhatToObserve: []string{
					"Locate Glu35 and Asp52 facing each other across the substrate-binding groove alongside sugar-stacking Trp62 and Trp63.",
					"Notice how mapping BLAST alignments onto 3D active sites distinguishes true isofunctional enzymes from non-catalytic paralogs.",
				},
				ScientificNote: "Homology is binary (shared ancestor or not); function transfer requires verifying active-site residue preservation.",
				Scene: SceneSpec{
					StructureID:     "1LYZ",
					ProteinAcc:      "P00698",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "secondary",
					CameraMode:      "selection",
					StepBannerTitle: "Lysozyme Catalytic Dyad (Glu35 & Asp52) + Trp62/Trp63 Cleft",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 35, Label: "Glu35 (General Acid Proton Donor)"},
						{ChainID: "A", AuthSeqID: 52, Label: "Asp52 (Catalytic Stabilizer)"},
						{ChainID: "A", AuthSeqID: 62, Label: "Trp62 (Sugar Ring Stacking)"},
						{ChainID: "A", AuthSeqID: 63, Label: "Trp63 (Sugar Ring Stacking)"},
					},
				},
			},
			{
				Index:      2,
				Title:      "3. Structure Evolves More Slowly Than Sequence",
				ConceptTag: "Evolutionary Structural Conservation",
				BodyParagraphs: []string{
					"Switch back to Hemoglobin (1GZX) and compare Chain A (α-globin, sky blue) against Chain B (β-globin, emerald green): after ~450 million years of divergence since an ancestral globin gene duplication, human α- and β-globin share only ~43% amino acid identity—and sperm whale myoglobin shares only ~25%!",
					"Yet in 3D space, their 8-helix tertiary folds superpose within ~1.4 Å RMSD! Surface residues tolerate frequent neutral substitutions as long as core packing volume and heme histidines remain intact.",
				},
				WhatToObserve: []string{
					"Compare the nearly mirror-symmetric 3D folds of Chain A (sky blue) and Chain B (emerald green) despite >80 amino acid differences between them.",
				},
				ScientificNote: "Below ~25% sequence identity ('the twilight zone'), pairwise BLAST struggles to detect remote homologs even when 3D structures remain unmistakably related.",
				Scene: SceneSpec{
					StructureID:     "1GZX",
					ProteinAcc:      "P69905",
					ActiveChain:     "B",
					Representation:  "cartoon",
					ColorScheme:     "chain",
					ShowLigands:     true,
					CameraMode:      "overview",
					StepBannerTitle: "Divergent Sequences (~43% Identity), Conserved 3D Globin Fold",
				},
			},
		},
		Quiz: []QuizQuestion{
			{
				ID:     "q1-evalue",
				Prompt: "You align an identical 20-residue peptide first against Swiss-Prot (570k proteins) and then against nr (500M proteins). How do Bit Score and E-value behave?",
				Options: []QuizOption{
					{ID: "a", Text: "Bit Score stays constant (normalized by scoring matrix parameters), while the E-value increases roughly ~800-fold in nr because random background hits scale with database size.", Explanation: "Correct! Bit Score depends only on the alignment and scoring system, whereas E = K·m·n·2^(-S') scales proportionally with search space n."},
					{ID: "b", Text: "Both Bit Score and E-value stay completely unchanged across all databases.", Explanation: "Incorrect: larger databases always inflate the expected number of chance matches at a fixed raw score."},
				},
				CorrectOption: "a",
				ProofStepIdx:  0,
			},
			{
				ID:     "q2-homolog",
				Prompt: "Your query protein hits Lysozyme (1LYZ) with E = 3e-28 and 39% identity, but sequence alignment shows Glu35→Thr and Asp52→Ser substitutions. What is the scientifically responsible conclusion?",
				Options: []QuizOption{
					{ID: "a", Text: "The proteins are evolutionary homologs sharing the C-type lysozyme fold, but muramidase catalytic activity is likely lost or altered because both essential catalytic carboxylates are missing.", Explanation: "Correct! Just like mammalian α-lactalbumin, homologous scaffold conservation does not guarantee retention of active-site catalysis."},
					{ID: "b", Text: "Any hit below E = 1e-10 automatically catalyzes the exact same reaction.", Explanation: "Incorrect: paralogous gene families routinely evolve non-enzymatic regulatory roles while retaining overall 3D fold."},
				},
				CorrectOption: "a",
				ProofStepIdx:  1,
			},
		},
	},
	{
		Slug:              "alphafold-confidence-and-disorder",
		Order:             5,
		Title:             "Interpreting AlphaFold: pLDDT, Disorder & Cofactors",
		Subtitle:          "Comparing full-length predicted p53 (AF-P04637-F1) against experimental MDM2 complex (1YCR)",
		Category:          "AI Structure Prediction Literacy",
		Difficulty:        "Intermediate",
		DurationMinutes:   10,
		FeaturedStructure: "AF-P04637-F1",
		FeaturedProtein:   "P04637",
		LearningGoals: []string{
			"Interpret AlphaFold's four per-residue pLDDT bands (>=90, 70–90, 50–70, <50) accurately.",
			"Avoid treating low-confidence (<50) intrinsically disordered ribbons as static physical conformations.",
			"Contrast AI monomer predictions against experimental structures capturing partner-induced folding and cofactors.",
		},
		Steps: []LessonStep{
			{
				Index:      0,
				Title:      "1. Reading AlphaFold's Per-Residue pLDDT Confidence Map",
				ConceptTag: "pLDDT Confidence Scale",
				BodyParagraphs: []string{
					"Unlike X-ray crystallography or cryo-EM, AlphaFold produces computational predictions rather than physical measurements. Every residue carries a predicted Local Distance Difference Test score (pLDDT, 0–100) stored in the PDB B-factor column.",
					"Inspect full-length human tumor suppressor p53 (AF-P04637-F1, 393 aa) colored by official AlphaFold confidence: Dark Blue (≥90, Very High—reliable backbone & sidechain rotamers) marks the central β-sandwich DNA-binding core (94–292) and tetramerization helix (325–356), whereas Orange (<50, Very Low) marks long N- and C-terminal tails!",
				},
				WhatToObserve: []string{
					"Notice the stark visual contrast between the compact dark-blue central core and the extended orange ribbons looping around it.",
					"Compare the pLDDT distribution metrics in the Quality card: ~39% Very High vs ~39% Very Low within a single famous protein!",
				},
				ScientificNote: "Always report per-domain confidence rather than quoting a single global mean pLDDT across multi-domain disordered proteins.",
				Scene: SceneSpec{
					StructureID:     "AF-P04637-F1",
					ProteinAcc:      "P04637",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "plddt",
					CameraMode:      "overview",
					StepBannerTitle: "Full-Length p53 AlphaFold v6 Model (Colored by pLDDT)",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 248, Label: "Arg248 (Core DBD · pLDDT > 94)"},
						{ChainID: "A", AuthSeqID: 273, Label: "Arg273 (Core DBD · pLDDT > 95)"},
					},
				},
			},
			{
				Index:      1,
				Title:      "2. Crucial Pitfall: Orange Ribbons Are Dynamic Ensembles, Not Rigid Shapes!",
				ConceptTag: "Intrinsic Disorder (IDRs)",
				BodyParagraphs: []string{
					"When AlphaFold renders residues 1–93 as wide sweeping orange ribbons wrapping loosely around the core, novices often hypothesize that residue 23 touches residue 180! DO NOT MAKE THIS MISTAKE: pLDDT < 50 is a strong predictor of biophysical intrinsic disorder in isolated solution.",
					"Notice an inspiring flash of signal inside that disordered tail: zoom onto residues Glu17–Asn29 (Phe19, Trp23, Leu26)—AlphaFold tentatively sketches a local helical turn at the exact transactivation motif, even though its global position relative to the core DBD is completely unfixed!",
				},
				WhatToObserve: []string{
					"Zoom onto Phe19, Trp23, and Leu26 inside AF-P04637-F1 and inspect their low/moderate pLDDT scores on the sequence strip.",
				},
				ScientificNote: "To judge whether two domains have fixed relative orientation in AlphaFold, you must check Predicted Aligned Error (PAE), not visual proximity in 3D!",
				Scene: SceneSpec{
					StructureID:     "AF-P04637-F1",
					ProteinAcc:      "P04637",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "plddt",
					CameraMode:      "selection",
					StepBannerTitle: "p53 N-Terminal Transactivation Motif in Monomer Prediction",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 19, Label: "Phe19 (TAD1 Triad)"},
						{ChainID: "A", AuthSeqID: 23, Label: "Trp23 (TAD1 Triad)"},
						{ChainID: "A", AuthSeqID: 26, Label: "Leu26 (TAD1 Triad)"},
					},
				},
			},
			{
				Index:      2,
				Title:      "3. Disorder-to-Order Folding in Experimental Structure 1YCR",
				ConceptTag: "Partner-Induced Folding",
				BodyParagraphs: []string{
					"Now switch directly to experimental 2.6 Å crystal structure PDB 1YCR (Kussie et al., Science 1996): here, p53 peptide Glu17–Asn29 (Chain B, emerald green) was co-crystallized bound to its negative regulator MDM2 (Chain A, sky blue)!",
					"Upon binding MDM2's deep hydrophobic groove, p53's disordered transactivation segment snaps into a crisp amphipathic α-helix burying Phe19, Trp23, and Leu26 tightly into MDM2—the exact pocket targeted by Nutlin anticancer drugs designed to reactivate p53!",
				},
				WhatToObserve: []string{
					"Observe Phe19, Trp23, and Leu26 projecting along one face of Chain B's helix deep inside MDM2's translucent surface pocket.",
					"Notice that Chain A (MDM2) is explicitly marked as a heteromeric partner chain in BioScope rather than falsely mapped to p53's sequence!",
				},
				ScientificNote: "Intrinsically disordered regions frequently evolve to bind multiple signaling partners cleanly via coupled folding and binding.",
				Scene: SceneSpec{
					StructureID:     "1YCR",
					ProteinAcc:      "P04637",
					ActiveChain:     "B",
					Representation:  "cartoon",
					ColorScheme:     "chain",
					ShowSurface:     true,
					CameraMode:      "selection",
					StepBannerTitle: "Experimental 1YCR: p53 Helix (Phe19/Trp23/Leu26) Inside MDM2 Cleft",
					FocusResidues: []FocusResidue{
						{ChainID: "B", AuthSeqID: 19, Label: "p53 Phe19"},
						{ChainID: "B", AuthSeqID: 23, Label: "p53 Trp23"},
						{ChainID: "B", AuthSeqID: 26, Label: "p53 Leu26"},
					},
				},
			},
			{
				Index:      3,
				Title:      "4. Missing Cofactors & Propeptides: AF-P69905-F1 vs Experimental 1GZX",
				ConceptTag: "Model Scope & Limitations",
				BodyParagraphs: []string{
					"Finally, inspect AlphaFold's monomer model for human α-globin (AF-P69905-F1): its mean pLDDT is an impressive 96.8 ('Very High'), and sidechains around His88 (PDB His87) pre-organize into an unmistakable heme-shaped cavity.",
					"Yet three biological realities are absent from monomer AlphaFold DB entries: (1) Heme Fe²⁺ and O₂ are absent, (2) initiator Met1 and cleaved pro-peptides (like preproinsulin's signal/C-peptide in AF-P01308-F1!) remain attached, and (3) α₂β₂ quaternary allostery is missing. Combining predicted and experimental structures gives you the best of both worlds!",
				},
				WhatToObserve: []string{
					"Inspect the empty cavity next to His88 in AF-P69905-F1 where Heme sits in experimental crystal structure 1GZX.",
				},
				ScientificNote: "Scientific best practice: always distinguish predictions from experimental structures in figures, citations, and downstream docking studies.",
				Scene: SceneSpec{
					StructureID:     "AF-P69905-F1",
					ProteinAcc:      "P69905",
					ActiveChain:     "A",
					Representation:  "cartoon",
					ColorScheme:     "plddt",
					CameraMode:      "selection",
					StepBannerTitle: "AlphaFold Monomer α-Globin (pLDDT 96.8 · Apo Cavity at His88)",
					FocusResidues: []FocusResidue{
						{ChainID: "A", AuthSeqID: 88, Label: "His88 (Pre-Formed Apo Heme Pocket)"},
						{ChainID: "A", AuthSeqID: 59, Label: "His59 (Distal Histidine)"},
					},
				},
			},
		},
		Quiz: []QuizQuestion{
			{
				ID:     "q1-plddt",
				Prompt: "In full-length p53 (AF-P04637-F1), residues 1–60 display pLDDT scores around 32–45 (colored orange). How should a researcher interpret those coordinates?",
				Options: []QuizOption{
					{ID: "a", Text: "As a high-precision rigid loop suitable for direct small-molecule virtual screening.", Explanation: "Incorrect: pLDDT < 50 regions should never be used directly as rigid docking pockets."},
					{ID: "b", Text: "As a strong indication that the region is intrinsically disordered in isolation, though local motifs (like 17–29) may fold upon binding partners such as MDM2.", Explanation: "Correct! Orange <50 ribbons depict an unfixed conformational ensemble rather than a single physical fold, until stabilized by binding partners as shown in PDB 1YCR."},
				},
				CorrectOption: "b",
				ProofStepIdx:  1,
			},
			{
				ID:     "q2-af-limits",
				Prompt: "Why does AlphaFold DB entry AF-P01308-F1 (Human Insulin) contain 110 amino acids in a single chain whereas experimental crystal structure 4INS contains separate 21-aa and 30-aa chains?",
				Options: []QuizOption{
					{ID: "a", Text: "AlphaFold DB predicts full canonical translated UniProt sequences (uncleaved preproinsulin), whereas mature experimental insulin has undergone signal peptide and C-peptide proteolytic cleavage.", Explanation: "Correct! Canonical AlphaFold DB models fold full-length translated sequences without post-translational proteolytic processing."},
					{ID: "b", Text: "X-ray crystallography deletes 59 random amino acids during crystallization.", Explanation: "Incorrect: mature biological insulin secreted by pancreatic β-cells genuinely consists only of disulfide-linked A and B chains."},
				},
				CorrectOption: "a",
				ProofStepIdx:  3,
			},
		},
	},
}

// List returns all lessons in pedagogical order.
func List() []Lesson {
	out := make([]Lesson, len(catalog))
	copy(out, catalog)
	return out
}

// Get returns a single lesson by slug.
func Get(slug string) (*Lesson, bool) {
	s := strings.ToLower(strings.TrimSpace(slug))
	for i := range catalog {
		if catalog[i].Slug == s {
			cp := catalog[i]
			return &cp, true
		}
	}
	return nil, false
}

// GradeAnswer evaluates a user's selected option for questionID within lesson slug.
func GradeAnswer(slug, questionID, optionID string) (*QuizEvaluation, error) {
	lesson, ok := Get(slug)
	if !ok {
		return nil, fmt.Errorf("lesson %q not found", slug)
	}
	optNorm := strings.ToLower(strings.TrimSpace(optionID))
	for _, q := range lesson.Quiz {
		if q.ID == questionID {
			var selExp, corExp string
			valid := false
			for _, o := range q.Options {
				if o.ID == optNorm {
					selExp = o.Explanation
					valid = true
				}
				if o.ID == q.CorrectOption {
					corExp = o.Explanation
				}
			}
			if !valid {
				return nil, fmt.Errorf("please choose an answer option before submitting")
			}
			return &QuizEvaluation{
				LessonSlug:      lesson.Slug,
				Question:        q,
				SelectedOption:  optNorm,
				IsCorrect:       optNorm == q.CorrectOption,
				SelectedExplain: selExp,
				CorrectExplain:  corExp,
			}, nil
		}
	}
	return nil, fmt.Errorf("question %q not found in lesson %q", questionID, slug)
}
