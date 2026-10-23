package handlers

import (
	"errors"
	"net/http"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/services"
)

type blastStatusViewData struct {
	Job             *bio.BlastJob
	Warnings        []string
	NextPollSeconds int
	Error           string
}

func (s *Server) handleBlastSubmit(w http.ResponseWriter, r *http.Request) {
	// Limit POST form size strictly to 128 KB to prevent memory exhaustion attacks.
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = s.renderer.RenderPartial(w, "blast_status.html", blastStatusViewData{
			Error: "Request payload exceeded safety limit or malformed form submission.",
		})
		return
	}

	rawSeq := r.FormValue("sequence")
	db := r.FormValue("database")
	mode := sanitizeChoice(r.FormValue("mode"), []string{"ncbi", "reference"}, "ncbi")

	job, warnings, err := s.blast.Submit(r.Context(), rawSeq, db, mode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrTooManyBlastJobs) {
			status = http.StatusTooManyRequests
		}
		w.WriteHeader(status)
		_ = s.renderer.RenderPartial(w, "blast_status.html", blastStatusViewData{
			Warnings: warnings,
			Error:    err.Error(),
		})
		return
	}

	_ = s.renderer.RenderPartial(w, "blast_status.html", blastStatusViewData{
		Job:             job,
		Warnings:        warnings,
		NextPollSeconds: int(s.cfg.BlastPollMinInterval.Seconds()),
	})
}

func (s *Server) handleBlastStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobID")
	job, nextSec, err := s.blast.GetAndPoll(r.Context(), jobID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = s.renderer.RenderPartial(w, "blast_status.html", blastStatusViewData{
			Error: "BLAST job not found or expired from bounded session memory.",
		})
		return
	}
	_ = s.renderer.RenderPartial(w, "blast_status.html", blastStatusViewData{
		Job:             job,
		NextPollSeconds: nextSec,
	})
}
