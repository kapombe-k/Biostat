# BioScope Testing & Verification Guide

BioScope ships with a fast, deterministic, network-independent test suite covering domain sequence validation, PDB/mmCIF coordinate parsing across 11 real structures, zero-false-equivalency sequence-to-structure alignment, upstream HTTP client mocking, SSRF redirect protection, NCBI BLAST rate-limit governance, lesson coordinate integrity, XSS HTML escaping, and end-to-end HTTP workflows.

---

## 1. Running the Automated Test Suite

```bash
# Run all unit and integration tests
go test -v ./...

# Run with Go's concurrent data-race detector enabled
go test -race -v ./...

# Run static analysis checks
go vet ./...

# Measure cross-package statement coverage across all internal packages
go test -count=1 -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out | tail -n 1
```

All tests use `net/http/httptest` servers and embedded crystallographic/AlphaFold coordinate fixtures (`internal/seed/structures/*`) — **zero external network calls are required** to pass CI or air-gapped verification.

---

## 2. Test Coverage Matrix

| Package | Test File | Verified Behaviors & Invariants | Isolated Coverage |
| :--- | :--- | :--- | :---: |
| `internal/bio` | `sequence_test.go`, `coordparser_test.go` | FASTA parsing & IUPAC validation; query classification (`uniprot`, `pdb`, `alphafold`, `sequence`, `keyword`); PDB & mmCIF coordinate parsing across `1GZX`, `1YCR`, `1GBT`, `1UBQ`, `2HYY`, `4INS`, and `AF-P04637-F1`; insertion code extraction (`A:184A`); pLDDT vs crystallographic B-factor statistics; `<= 4.2 A` HETATM pocket contact calculation; SHA-256 coordinate digest | **81.2%** |
| `internal/lessons` | `lessons_test.go` | Structural integrity of all 5 multi-step lessons; verifies every single `FocusResidue`, `DistanceMeasurement` atom pair, `Label3D` anchor, and `FocusLigand` exists at exact coordinates inside the target structure; quiz grading validation | **85.2%** |
| `internal/cache` | `cache_test.go` | Generic LRU promotion & eviction under capacity pressure; deterministic TTL expiration via injected clock; concurrent `FlightGroup` request deduplication (`50` goroutines -> `1` upstream fetch) | **78.6%** |
| `internal/config` | `config_test.go` | Default port (`7860`) & host (`0.0.0.0`) loading; SSRF allowlist verification rejecting arbitrary hosts (`169.254.169.254`, `evil.example.org`, `http://` MITM downgrade on HTTPS origins) | **78.0%** |
| `internal/handlers` | `handlers_test.go` | End-to-end HTTP integration across `/healthz`, `/`, `/partials/search` (including `<script>` XSS auto-escaping check), `/explore`, `/partials/sequence`, `/api/structures/{id}/{coordinates,annotations}`, `/lessons/{slug}/step/{step}` (`HX-Trigger: {"bioscope:applyScene": ...}`), `/lessons/{slug}/quiz/{id}`, `/partials/blast/submit` (`400` validation recovery + `200` execution), `/api/export/json`, and `/api/export/fasta` | **69.9%** |
| `internal/clients` | `clients_test.go` | Retry with exponential backoff on HTTP `503`; UniProt JSON parsing; RCSB Data v1 summary parsing; AlphaFold DB v4/v6 schema parsing; PDBe SIFTS mapping parsing; NCBI QBlast `CMD=Put`, `SearchInfo`, strict `ErrPollRateLimited` (`< 60s` per `RID`), `BlastOutput` XML HSP parsing; per-host circuit breaker trip & cooldown recovery | **58.3%** |
| `internal/services` | `services_test.go` | Zero-false-equivalency alignment across `1GBT` (PDB insertion codes), `1YCR` (`MDM2` heteromeric partner detection on Chain A vs `p53` Chain B), `4INS` (cleaved two-chain mature insulin vs preproinsulin), and `2HYY` (internal missing electron density loop); MVS annotation generation across all 5 color schemes; `BlastManager` submission & status polling | **58.1%** |
| **Total (`./internal/...`)** | **All Suites Combined** | **Cross-package statement coverage (`-coverpkg=./internal/...`) with `-race` enabled** | **74.3%** |

---

## 3. Recorded Verification Summary

```text
$ go vet ./...
(exit 0, zero warnings)

$ go test -race -v ./...
ok  	github.com/kapombe-k/Biostat/internal/bio       1.082s
ok  	github.com/kapombe-k/Biostat/internal/cache     1.041s
ok  	github.com/kapombe-k/Biostat/internal/clients   1.067s
ok  	github.com/kapombe-k/Biostat/internal/config    1.010s
ok  	github.com/kapombe-k/Biostat/internal/handlers  1.281s
ok  	github.com/kapombe-k/Biostat/internal/lessons   1.296s
ok  	github.com/kapombe-k/Biostat/internal/services  1.272s

$ go test -count=1 -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
total:                                  (statements)            74.3%
```
