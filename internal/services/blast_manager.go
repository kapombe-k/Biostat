package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/clients"
	"github.com/kapombe-k/Biostat/internal/config"
	"github.com/kapombe-k/Biostat/internal/seed"
)

// ErrTooManyBlastJobs indicates the bounded active BLAST queue is full.
var ErrTooManyBlastJobs = errors.New("maximum concurrent BLAST background jobs reached; please wait for active searches to finish")

// BlastManager coordinates asynchronous sequence validation, job submission, rate-limited polling, and bounded retention.
type BlastManager struct {
	cfg    *config.Config
	client clients.BlastClient
	logger *slog.Logger

	mu        sync.Mutex
	jobs      map[string]*bio.BlastJob
	jobOrder  []string
	pollLocks map[string]*sync.Mutex
	nowFn     func() time.Time
}

// NewBlastManager constructs a bounded BLAST job orchestrator.
func NewBlastManager(cfg *config.Config, client clients.BlastClient, logger *slog.Logger) *BlastManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &BlastManager{
		cfg:       cfg,
		client:    client,
		logger:    logger,
		jobs:      make(map[string]*bio.BlastJob),
		pollLocks: make(map[string]*sync.Mutex),
		nowFn:     time.Now,
	}
}

// Submit validates rawFASTA and initiates either a live NCBI QBlast job or a classroom reference search.
func (m *BlastManager) Submit(ctx context.Context, rawFASTA, database, mode string) (*bio.BlastJob, []string, error) {
	parsed, err := bio.ParseAndValidateSequence(rawFASTA, m.cfg.MinSequenceLength, m.cfg.MaxSequenceLength)
	if err != nil {
		return nil, nil, err
	}

	if database == "" {
		database = "swissprot"
	}

	// Enforce bounded active job capacity before contacting upstream APIs.
	m.mu.Lock()
	activeRunning := 0
	for _, j := range m.jobs {
		if j.Status == bio.BlastQueued || j.Status == bio.BlastRunning {
			activeRunning++
		}
	}
	if activeRunning >= m.cfg.BlastMaxJobs {
		m.mu.Unlock()
		return nil, parsed.Warnings, ErrTooManyBlastJobs
	}
	m.mu.Unlock()

	now := m.nowFn().UTC()
	jobID := randomJobID()

	if mode == "reference" {
		job := m.runClassroomReferenceJob(jobID, parsed, database, now, "Requested instant BioScope reference homology search")
		m.storeJob(job)
		return job, parsed.Warnings, nil
	}

	rid, rtoe, endpoint, submitErr := m.client.SubmitSearch(ctx, database, parsed.Sequence)
	if submitErr != nil {
		m.logger.Warn("live NCBI BLAST submission failed", "err", submitErr, "fallback_enabled", m.cfg.OfflineFallback)
		if m.cfg.OfflineFallback {
			job := m.runClassroomReferenceJob(jobID, parsed, database, now,
				"Live NCBI BLAST unreachable from current network; executed transparent local Smith-Waterman alignment against BioScope's curated Swiss-Prot reference catalog.")
			m.storeJob(job)
			return job, parsed.Warnings, nil
		}
		return nil, parsed.Warnings, fmt.Errorf("NCBI BLAST submission failed: %w", submitErr)
	}

	job := &bio.BlastJob{
		JobID:             jobID,
		RID:               rid,
		Program:           "blastp",
		Database:          database,
		QueryHeader:       parsed.Header,
		QuerySequence:     parsed.Sequence,
		QueryLength:       parsed.Length,
		Status:            bio.BlastQueued,
		DeliveryMode:      bio.DeliveryLive,
		SubmittedAt:       now,
		LastPolledAt:      now,
		NextAllowedPollAt: now.Add(m.cfg.BlastPollMinInterval),
		EstimatedWaitSec:  rtoe,
		Provenance: bio.ProvenanceRecord{
			SourceName:   "NCBI BLAST+ URL API",
			RecordID:     rid,
			RecordStatus: "Asynchronous Remote Job",
			EndpointURL:  endpoint,
			RetrievedAt:  now,
			DeliveryMode: bio.DeliveryLive,
			Citation:     "Camacho, C. et al. (2009) BLAST+: architecture and applications. BMC Bioinformatics 10, 421.",
			License:      "NCBI Public Domain / Fair Usage Policy (max 1 poll/min/RID)",
		},
	}
	m.storeJob(job)
	return m.cloneJob(job), parsed.Warnings, nil
}

// GetAndPoll returns the current state of jobID, querying NCBI at most once per configured BlastPollMinInterval (60s).
func (m *BlastManager) GetAndPoll(ctx context.Context, jobID string) (*bio.BlastJob, int, error) {
	m.mu.Lock()
	job, ok := m.jobs[jobID]
	if !ok {
		m.mu.Unlock()
		return nil, 0, clients.ErrNotFound
	}
	lock := m.pollLocks[jobID]
	m.mu.Unlock()

	lock.Lock()
	defer lock.Unlock()

	m.mu.Lock()
	job = m.jobs[jobID]
	now := m.nowFn().UTC()
	if job.Status == bio.BlastReady || job.Status == bio.BlastFailed || job.DeliveryMode == bio.DeliverySeedFallback {
		cp := m.cloneJob(job)
		m.mu.Unlock()
		return cp, 0, nil
	}
	if now.Before(job.NextAllowedPollAt) {
		rem := int(job.NextAllowedPollAt.Sub(now).Seconds()) + 1
		cp := m.cloneJob(job)
		m.mu.Unlock()
		return cp, rem, nil
	}
	rid := job.RID
	qLen := job.QueryLength
	m.mu.Unlock()

	status, msg, err := m.client.PollStatus(ctx, rid)
	pollTime := m.nowFn().UTC()

	m.mu.Lock()
	job.LastPolledAt = pollTime
	job.NextAllowedPollAt = pollTime.Add(m.cfg.BlastPollMinInterval)
	job.PollCount++
	m.mu.Unlock()

	if err != nil {
		if errors.Is(err, clients.ErrPollRateLimited) {
			m.mu.Lock()
			rem := int(job.NextAllowedPollAt.Sub(pollTime).Seconds()) + 1
			cp := m.cloneJob(job)
			m.mu.Unlock()
			return cp, rem, nil
		}
		m.logger.Warn("BLAST status poll error", "rid", rid, "err", err)
		m.mu.Lock()
		cp := m.cloneJob(job)
		m.mu.Unlock()
		return cp, int(m.cfg.BlastPollMinInterval.Seconds()), nil
	}

	if status == bio.BlastReady {
		hits, fetchErr := m.client.FetchResults(ctx, rid, qLen)
		m.mu.Lock()
		defer m.mu.Unlock()
		if fetchErr != nil {
			job.Status = bio.BlastFailed
			job.ErrorMessage = fmt.Sprintf("Failed to parse BLAST results: %v", fetchErr)
			return m.cloneJob(job), 0, nil
		}
		job.Status = bio.BlastReady
		job.Hits = hits
		job.Provenance.RetrievedAt = pollTime
		return m.cloneJob(job), 0, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	job.Status = status
	if status == bio.BlastFailed {
		job.ErrorMessage = msg
		return m.cloneJob(job), 0, nil
	}
	return m.cloneJob(job), int(m.cfg.BlastPollMinInterval.Seconds()), nil
}

func (m *BlastManager) runClassroomReferenceJob(jobID string, parsed *bio.ParsedSequence, db string, now time.Time, note string) *bio.BlastJob {
	hits := seed.LocalSimilaritySearch(parsed.Sequence)
	return &bio.BlastJob{
		JobID:             jobID,
		RID:               "REF-" + jobID,
		Program:           "Smith-Waterman (Reference Swiss-Prot Subset)",
		Database:          db,
		QueryHeader:       parsed.Header,
		QuerySequence:     parsed.Sequence,
		QueryLength:       parsed.Length,
		Status:            bio.BlastReady,
		DeliveryMode:      bio.DeliverySeedFallback,
		SubmittedAt:       now,
		LastPolledAt:      now,
		NextAllowedPollAt: now,
		ErrorMessage:      note,
		Hits:              hits,
		Provenance: bio.ProvenanceRecord{
			SourceName:   "BioScope Reference Homology Engine (Curated Swiss-Prot Subset)",
			RecordID:     "REF-" + jobID,
			RecordStatus: "Completed Local Alignment",
			EndpointURL:  "bioscope://seed/local-smith-waterman",
			RetrievedAt:  now,
			DeliveryMode: bio.DeliverySeedFallback,
			Citation:     "Smith, T.F. & Waterman, M.S. (1981) Identification of common molecular subsequences. J. Mol. Biol. 147, 195-197.",
			License:      "CC BY 4.0 (Swiss-Prot Reference Sequences)",
		},
	}
}

func (m *BlastManager) storeJob(job *bio.BlastJob) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Evict oldest finished job when total retained jobs reaches 2*BlastMaxJobs.
	maxRetained := m.cfg.BlastMaxJobs * 2
	if maxRetained < 8 {
		maxRetained = 8
	}
	if len(m.jobOrder) >= maxRetained {
		for idx, id := range m.jobOrder {
			if j, exists := m.jobs[id]; exists && (j.Status == bio.BlastReady || j.Status == bio.BlastFailed) {
				delete(m.jobs, id)
				delete(m.pollLocks, id)
				m.jobOrder = append(m.jobOrder[:idx], m.jobOrder[idx+1:]...)
				break
			}
		}
	}

	m.jobs[job.JobID] = job
	m.pollLocks[job.JobID] = &sync.Mutex{}
	m.jobOrder = append(m.jobOrder, job.JobID)
}

func (m *BlastManager) cloneJob(j *bio.BlastJob) *bio.BlastJob {
	cp := *j
	if len(j.Hits) > 0 {
		cp.Hits = make([]bio.BlastHit, len(j.Hits))
		copy(cp.Hits, j.Hits)
	}
	return &cp
}

func randomJobID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
