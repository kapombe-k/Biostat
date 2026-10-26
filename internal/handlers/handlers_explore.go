package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/lessons"
	"github.com/kapombe-k/Biostat/internal/services"
)

type searchViewData struct {
	RawQuery    string
	Hits        []bio.SearchHit
	ParsedFASTA *bio.ParsedSequence
	Error       string
}

type homePageData struct {
	PageTitle   string
	ActiveNav   string
	NavQuery    string
	LoadViewers bool
	SearchData  searchViewData
	Lessons     []lessons.Lesson
}

type sceneBootstrap struct {
	StructureID     string                 `json:"structure_id"`
	Format          string                 `json:"format"`
	ProteinAcc      string                 `json:"protein_accession"`
	ActiveChain     string                 `json:"active_chain"`
	Representation  string                 `json:"representation"`
	ColorScheme     string                 `json:"color_scheme"`
	ShowSurface     bool                   `json:"show_surface"`
	ShowLigands     bool                   `json:"show_ligands"`
	FocusLigand     string                 `json:"focus_ligand,omitempty"`
	FocusResidues   []lessons.FocusResidue `json:"focus_residues,omitempty"`
	CameraMode      string                 `json:"camera_mode"`
	StepBannerTitle string                 `json:"step_banner_title,omitempty"`
	Engine          string                 `json:"engine"`
}

type workspacePageData struct {
	PageTitle      string
	ActiveNav      string
	NavQuery       string
	LoadViewers    bool
	Bundle         *services.WorkspaceBundle
	SceneBootstrap sceneBootstrap
	ShareURL       string
}

type errorPageData struct {
	PageTitle    string
	ActiveNav    string
	NavQuery     string
	LoadViewers  bool
	ErrorHeading string
	ErrorMessage string
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	data := homePageData{
		PageTitle:   "Explore Biomolecules in 3D",
		ActiveNav:   "home",
		LoadViewers: false,
		SearchData: searchViewData{
			Hits: s.explorer.FeaturedProteins(),
		},
		Lessons: lessons.List(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPage(w, "home", data); err != nil {
		s.logger.Error("render home failed", "err", err)
		http.Error(w, "Template render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleSearchPartial(w http.ResponseWriter, r *http.Request) {
	rawQ := strings.TrimSpace(r.URL.Query().Get("q"))
	_, hits, parsed, err := s.explorer.Search(r.Context(), rawQ)
	vd := searchViewData{
		RawQuery:    rawQ,
		Hits:        hits,
		ParsedFASTA: parsed,
	}
	if err != nil {
		vd.Error = err.Error()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "search_results.html", vd); err != nil {
		s.logger.Error("render search partial failed", "err", err)
		http.Error(w, "Search render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleExplore(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	structParam := strings.TrimSpace(r.URL.Query().Get("struct"))
	chainParam := strings.TrimSpace(r.URL.Query().Get("chain"))
	repParam := sanitizeChoice(r.URL.Query().Get("rep"), []string{"cartoon", "ball_and_stick", "spacefill", "backbone"}, "cartoon")
	colorParam := sanitizeChoice(r.URL.Query().Get("color"), []string{"plddt", "secondary", "chain", "bfactor", "hydrophobicity"}, "")
	engineParam := sanitizeChoice(r.URL.Query().Get("engine"), []string{"molstar", "3dmol", "summary2d"}, "molstar")
	focusParam := strings.TrimSpace(r.URL.Query().Get("focus"))

	bundle, err := s.explorer.ResolveWorkspace(r.Context(), q, structParam, chainParam)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.renderer.RenderPage(w, "error", errorPageData{
			PageTitle:    "Record Not Found",
			ActiveNav:    "explore",
			NavQuery:     q,
			ErrorHeading: "Could not resolve biomolecule or structure",
			ErrorMessage: err.Error(),
		})
		return
	}

	if colorParam == "" {
		colorParam = bundle.DefaultColorScheme
	}

	activeChainID := "A"
	if bundle.ActiveChain != nil {
		activeChainID = bundle.ActiveChain.AuthAsymID
	}
	focusList := parseFocusParam(focusParam, activeChainID)
	camMode := "overview"
	if len(focusList) > 0 {
		camMode = "selection"
	}

	var sid, sfmt string
	if bundle.ActiveStructure != nil {
		sid = bundle.ActiveStructure.ID
		sfmt = bundle.ActiveStructure.Format
	}

	origin := requestOrigin(r)
	shareState := bio.ExplorationState{
		Query:          q,
		Accession:      bundle.Protein.Accession,
		StructureID:    sid,
		ChainID:        activeChainID,
		Representation: repParam,
		ColorScheme:    colorParam,
		FocusResidues:  focusParam,
		ViewerEngine:   engineParam,
	}

	data := workspacePageData{
		PageTitle:   fmt.Sprintf("%s (%s)", bundle.Protein.RecommendedName, bundle.Protein.Accession),
		ActiveNav:   "explore",
		NavQuery:    q,
		LoadViewers: true,
		Bundle:      bundle,
		ShareURL:    services.BuildShareURL(origin, shareState),
		SceneBootstrap: sceneBootstrap{
			StructureID:    sid,
			Format:         sfmt,
			ProteinAcc:     bundle.Protein.Accession,
			ActiveChain:    activeChainID,
			Representation: repParam,
			ColorScheme:    colorParam,
			ShowSurface:    false,
			ShowLigands:    true,
			FocusResidues:  focusList,
			CameraMode:     camMode,
			Engine:         engineParam,
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPage(w, "workspace", data); err != nil {
		s.logger.Error("render workspace failed", "err", err)
		http.Error(w, "Workspace render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleSequencePartial(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	structParam := strings.TrimSpace(r.URL.Query().Get("struct"))
	chainParam := strings.TrimSpace(r.URL.Query().Get("chain"))

	bundle, err := s.explorer.ResolveWorkspace(r.Context(), q, structParam, chainParam)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "sequence_panel.html", bundle); err != nil {
		s.logger.Error("render sequence partial failed", "err", err)
		http.Error(w, "Sequence partial render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleStructureCoordinates(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	payload, err := s.explorer.GetStructureCoordinates(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ext := "pdb"
	ct := "chemical/x-pdb; charset=utf-8"
	if payload.Format == "mmcif" {
		ext = "cif"
		ct = "chemical/x-cif; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Structure-Format", payload.Format)
	w.Header().Set("Cache-Control", "public, max-age=1800")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, payload.ID, ext))
	}
	_, _ = w.Write(payload.Data)
}

func (s *Server) handleStructureAnnotations(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scheme := r.URL.Query().Get("scheme")
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var prot *bio.ProteinRecord
	if q != "" {
		prot, _ = s.explorer.GetProtein(r.Context(), q)
	}
	detail, err := s.explorer.GetStructureDetail(r.Context(), id, prot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	anns := services.BuildMVSAnnotations(detail, scheme)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(anns)
}

func sanitizeChoice(val string, allowed []string, fallback string) string {
	v := strings.ToLower(strings.TrimSpace(val))
	for _, a := range allowed {
		if v == a {
			return a
		}
	}
	return fallback
}

func parseFocusParam(raw, defaultChain string) []lessons.FocusResidue {
	if raw == "" {
		return nil
	}
	var out []lessons.FocusResidue
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		ch := defaultChain
		resPart := tok
		if idx := strings.IndexByte(tok, ':'); idx > 0 {
			ch = tok[:idx]
			resPart = tok[idx+1:]
		}
		// Extract optional trailing letter insertion code (e.g. "184A")
		ins := ""
		if n := len(resPart); n > 1 && resPart[n-1] >= 'A' && resPart[n-1] <= 'Z' {
			ins = resPart[n-1:]
			resPart = resPart[:n-1]
		}
		if num, err := strconv.Atoi(resPart); err == nil {
			out = append(out, lessons.FocusResidue{
				ChainID:   ch,
				AuthSeqID: num,
				InsCode:   ins,
				Label:     fmt.Sprintf("%s:%d%s", ch, num, ins),
			})
		}
	}
	return out
}

func requestOrigin(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", proto, host)
}
