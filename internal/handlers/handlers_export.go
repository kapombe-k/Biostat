package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kapombe-k/Biostat/internal/bio"
)

func (s *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	structParam := strings.TrimSpace(r.URL.Query().Get("struct"))
	chainParam := strings.TrimSpace(r.URL.Query().Get("chain"))

	bundle, err := s.explorer.ResolveWorkspace(r.Context(), q, structParam, chainParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	sid := ""
	if bundle.ActiveStructure != nil {
		sid = bundle.ActiveStructure.ID
	}
	cid := ""
	if bundle.ActiveChain != nil {
		cid = bundle.ActiveChain.AuthAsymID
	}

	state := bio.ExplorationState{
		Query:          q,
		Accession:      bundle.Protein.Accession,
		StructureID:    sid,
		ChainID:        cid,
		Representation: sanitizeChoice(r.URL.Query().Get("rep"), []string{"cartoon", "ball_and_stick", "spacefill", "backbone"}, "cartoon"),
		ColorScheme:    sanitizeChoice(r.URL.Query().Get("color"), []string{"plddt", "secondary", "chain", "bfactor", "hydrophobicity"}, bundle.DefaultColorScheme),
		FocusResidues:  strings.TrimSpace(r.URL.Query().Get("focus")),
		ViewerEngine:   sanitizeChoice(r.URL.Query().Get("engine"), []string{"molstar", "3dmol", "summary2d"}, "molstar"),
	}

	export := s.explorer.BuildExport(bundle, state, requestOrigin(r))
	filename := fmt.Sprintf("bioscope-%s", strings.ToLower(bundle.Protein.Accession))
	if sid != "" {
		filename += "-" + strings.ToLower(sid)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, filename))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(export)
}

func (s *Server) handleExportFASTA(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	bundle, err := s.explorer.ResolveWorkspace(r.Context(), q, "", "")
	if err != nil || bundle.Protein == nil || bundle.Protein.Sequence == "" {
		http.Error(w, "Protein sequence not found", http.StatusNotFound)
		return
	}

	prot := bundle.Protein
	hdr := fmt.Sprintf("sp|%s|%s %s OS=%s OX=%d", prot.Accession, prot.EntryName, prot.RecommendedName, prot.OrganismName, prot.TaxonID)
	fasta := bio.FormatFASTA(hdr, prot.Sequence)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.fasta"`, strings.ToLower(prot.Accession)))
	_, _ = w.Write([]byte(fasta))
}
