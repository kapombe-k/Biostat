package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/kapombe-k/Biostat/internal/lessons"
	"github.com/kapombe-k/Biostat/internal/services"
)

type lessonsIndexData struct {
	PageTitle   string
	ActiveNav   string
	NavQuery    string
	LoadViewers bool
	Lessons     []lessons.Lesson
}

type stepPartialData struct {
	Lesson  *lessons.Lesson
	Step    lessons.LessonStep
	HasPrev bool
	PrevIdx int
	HasNext bool
	NextIdx int
}

type lessonViewPageData struct {
	PageTitle      string
	ActiveNav      string
	NavQuery       string
	LoadViewers    bool
	Lesson         *lessons.Lesson
	NextLesson     *lessons.Lesson
	StepView       stepPartialData
	Bundle         *services.WorkspaceBundle
	SceneBootstrap sceneBootstrap
}

func (s *Server) handleLessonsIndex(w http.ResponseWriter, _ *http.Request) {
	data := lessonsIndexData{
		PageTitle:   "Guided Structural Biology Lessons",
		ActiveNav:   "lessons",
		LoadViewers: false,
		Lessons:     lessons.List(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPage(w, "lessons_index", data); err != nil {
		s.logger.Error("render lessons index failed", "err", err)
		http.Error(w, "Template render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleLessonView(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	lesson, ok := lessons.Get(slug)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.renderer.RenderPage(w, "error", errorPageData{
			PageTitle:    "Lesson Not Found",
			ActiveNav:    "lessons",
			ErrorHeading: "Guided lesson module not found",
			ErrorMessage: "Please select one of BioScope's interactive structural biology lessons.",
		})
		return
	}

	stepIdx := 0
	if sp := r.URL.Query().Get("step"); sp != "" {
		if n, err := strconv.Atoi(sp); err == nil && n >= 0 && n < len(lesson.Steps) {
			stepIdx = n
		}
	}
	step := lesson.Steps[stepIdx]

	bundle, err := s.explorer.ResolveWorkspace(r.Context(), step.Scene.ProteinAcc, step.Scene.StructureID, step.Scene.ActiveChain)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all := lessons.List()
	var nextLesson *lessons.Lesson
	for i := range all {
		if all[i].Slug == lesson.Slug && i+1 < len(all) {
			nl := all[i+1]
			nextLesson = &nl
			break
		}
	}

	format := "pdb"
	if bundle.ActiveStructure != nil {
		format = bundle.ActiveStructure.Format
	}

	data := lessonViewPageData{
		PageTitle:   lesson.Title,
		ActiveNav:   "lessons",
		LoadViewers: true,
		Lesson:      lesson,
		NextLesson:  nextLesson,
		StepView:    buildStepData(lesson, stepIdx),
		Bundle:      bundle,
		SceneBootstrap: sceneBootstrap{
			StructureID:     step.Scene.StructureID,
			Format:          format,
			ProteinAcc:      step.Scene.ProteinAcc,
			ActiveChain:     step.Scene.ActiveChain,
			Representation:  step.Scene.Representation,
			ColorScheme:     step.Scene.ColorScheme,
			ShowSurface:     step.Scene.ShowSurface,
			ShowLigands:     step.Scene.ShowLigands,
			FocusLigand:     step.Scene.FocusLigand,
			FocusResidues:   step.Scene.FocusResidues,
			CameraMode:      step.Scene.CameraMode,
			StepBannerTitle: step.Scene.StepBannerTitle,
			Engine:          "molstar",
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPage(w, "lesson_view", data); err != nil {
		s.logger.Error("render lesson view failed", "err", err)
		http.Error(w, "Template render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleLessonStep(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	stepStr := r.PathValue("step")

	lesson, ok := lessons.Get(slug)
	if !ok {
		http.Error(w, "lesson not found", http.StatusNotFound)
		return
	}
	stepIdx, err := strconv.Atoi(stepStr)
	if err != nil || stepIdx < 0 || stepIdx >= len(lesson.Steps) {
		http.Error(w, "invalid step index", http.StatusBadRequest)
		return
	}

	stepData := buildStepData(lesson, stepIdx)
	// Emit HTMX HX-Trigger header carrying declarative SceneSpec so the 3D viewer and sequence panel transition automatically!
	triggerPayload, err := json.Marshal(map[string]any{
		"bioscope:applyScene": stepData.Step.Scene,
	})
	if err == nil {
		w.Header().Set("HX-Trigger", string(triggerPayload))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RenderPartial(w, "lesson_step.html", stepData); err != nil {
		s.logger.Error("render lesson step partial failed", "err", err)
		http.Error(w, "Step render error", http.StatusInternalServerError)
	}
}

func (s *Server) handleLessonQuiz(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	_ = r.ParseForm()
	slug := r.PathValue("slug")
	qID := r.PathValue("questionID")
	opt := r.FormValue("option")

	eval, err := lessons.GradeAnswer(slug, qID, opt)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = s.renderer.RenderPartial(w, "quiz_feedback.html", map[string]any{"Error": err.Error()})
		return
	}
	_ = s.renderer.RenderPartial(w, "quiz_feedback.html", map[string]any{"Eval": eval})
}

func buildStepData(lesson *lessons.Lesson, idx int) stepPartialData {
	return stepPartialData{
		Lesson:  lesson,
		Step:    lesson.Steps[idx],
		HasPrev: idx > 0,
		PrevIdx: idx - 1,
		HasNext: idx+1 < len(lesson.Steps),
		NextIdx: idx + 1,
	}
}
