package bio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRealSeedStructures(t *testing.T) {
	// 1. Verify mmCIF parsing, PDB insertion codes, and ligand contacts on Bovine Trypsin (1GBT.cif)
	rawGBT, err := os.ReadFile(filepath.Join("..", "seed", "structures", "1GBT.cif"))
	if err != nil {
		t.Fatalf("read 1GBT.cif: %v", err)
	}
	gbt, err := ParseCoordinates("1GBT", StructureExperimental, rawGBT)
	if err != nil {
		t.Fatalf("ParseCoordinates(1GBT): %v", err)
	}
	if gbt.Format != "mmcif" || len(gbt.Chains) != 1 {
		t.Fatalf("expected 1 mmcif chain for 1GBT, got format=%s chains=%d", gbt.Format, len(gbt.Chains))
	}
	if gbt.Chains[0].InsertionCodeCount < 4 {
		t.Fatalf("expected >=4 insertion codes in 1GBT (65A, 184A, 188A, 221A), got %d", gbt.Chains[0].InsertionCodeCount)
	}
	found184A := false
	for _, r := range gbt.Chains[0].Residues {
		if r.AuthLabel == "184A" && r.InsCode == "A" {
			found184A = true
		}
	}
	if !found184A {
		t.Fatalf("expected insertion code residue 184A in 1GBT")
	}
	if len(gbt.Ligands) == 0 {
		t.Fatalf("expected GBS/CA ligands in 1GBT")
	}

	// 2. Verify multi-chain tetramer, heme/oxygen contacts, and CA-inferred helices on Oxyhemoglobin (1GZX.pdb)
	rawGZX, err := os.ReadFile(filepath.Join("..", "seed", "structures", "1GZX.pdb"))
	if err != nil {
		t.Fatalf("read 1GZX.pdb: %v", err)
	}
	gzx, err := ParseCoordinates("1GZX", StructureExperimental, rawGZX)
	if err != nil {
		t.Fatalf("ParseCoordinates(1GZX): %v", err)
	}
	if len(gzx.Chains) != 4 {
		t.Fatalf("expected 4 chains in 1GZX tetramer, got %d", len(gzx.Chains))
	}
	helices := 0
	for _, r := range gzx.Chains[0].Residues {
		if r.Secondary == "helix" {
			helices++
		}
	}
	if helices < 70 {
		t.Fatalf("expected globin fold to contain rich helical structure, got %d helical residues", helices)
	}

	// 3. Verify AlphaFold pLDDT confidence breakdown on full-length p53 (AF-P04637-F1.pdb)
	rawAF, err := os.ReadFile(filepath.Join("..", "seed", "structures", "AF-P04637-F1.pdb"))
	if err != nil {
		t.Fatalf("read AF-P04637-F1.pdb: %v", err)
	}
	afP53, err := ParseCoordinates("AF-P04637-F1", StructurePredicted, rawAF)
	if err != nil {
		t.Fatalf("ParseCoordinates(AF-P04637-F1): %v", err)
	}
	if afP53.Chains[0].ObservedCount != 393 {
		t.Fatalf("expected 393 residues in AF-P04637-F1, got %d", afP53.Chains[0].ObservedCount)
	}
	// p53 famously has both high-confidence core DNA-binding domain AND very-low confidence disordered tails!
	if afP53.Quality.PctVeryHigh < 25 || afP53.Quality.PctVeryLow < 25 {
		t.Fatalf("expected bimodal pLDDT distribution on p53, got %+v", afP53.Quality)
	}
}
