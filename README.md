---
title: BioScope Interactive Molecular Learning Lab
emoji: 🧬
colorFrom: blue
colorTo: green
sdk: docker
app_port: 7860
license: mit
short_description: Interactive 3D protein structure, sequence & BLAST lab
---

# BioScope: Interactive Molecular Learning Lab

**BioScope** is a production-ready educational structural biology web application built with **Go (`net/http` + `html/template`)**, **HTMX (`2.0.11`)**, **Mol\* (`5.13.1`)**, and **3Dmol.js (`2.5.5`)**. It connects protein sequence biology directly to interactive 3D molecular structures without requiring users to install desktop modeling software or manage complex command-line pipelines.

---

## Core Capabilities

- **Unified Multi-Modal Discovery:** Search seamlessly by **protein or gene name** (`hemoglobin`, `TP53`, `insulin`), **UniProtKB accession** (`P69905`, `P04637`), **experimental PDB ID** (`1GZX`, `1GBT`, `2HYY`), **AlphaFold DB model ID** (`AF-P04637-F1`), or **raw amino acid / FASTA sequence**.
- **Experimental vs. Predicted Structure Exploration:** Compare X-ray diffraction, cryo-EM, and NMR structures (**RCSB PDB** & **PDBe**) side-by-side against **AlphaFold DB** models, keeping physical resolution/validation metrics strictly distinct from per-residue **pLDDT confidence scores (`0–100`)**.
- **Dual-Engine Interactive 3D Stage + Non-WebGL Fallback:**
  - **Primary Engine — Mol\* (`5.13.1`, MIT):** Declarative **MolViewSpec (`MVSJ`)** scenes with custom residue coloring (`pLDDT`, `Secondary Structure`, `Chain`, `B-factor`, `Hydrophobicity`), residue hover tooltips, active-site distance callouts, and surface overlays.
  - **Secondary Engine — 3Dmol.js (`2.5.5`, BSD-3-Clause):** Instant 1-click lighter WebGL viewer with automatic failover.
  - **2D Topology Fallback:** Interactive SVG secondary-structure, confidence, and functional domain diagram when WebGL is disabled.
- **Zero-False-Equivalency Sequence-to-Structure Mapping:** Hybrid **PDBe SIFTS + affine-gap Smith–Waterman** alignment accurately handles cleaved pro-peptides (`4INS`), missing crystallographic loop density (`2HYY`), PDB insertion codes (`1GBT`: `184A`, `188A`, `221A`), and heteromeric binding partners (`1YCR`: `MDM2` bound to `p53`) without ever mis-mapping residue numbers.
- **Rate-Governed Sequence Similarity Search:** Submit sequences asynchronously to **NCBI BLAST+ (`QBlast`)** with strict server-side rate governance (`≤ 1 status poll / 60s / RID`) or run instant classroom Smith–Waterman homology searches.
- **5 Guided Interactive Lessons & Graded Quizzes:** Step-by-step curriculum driving live 3D camera transitions, representations, ligand pocket zoom, and comprehension checks across hemoglobin allostery, serine protease catalysis & kinase inhibition, AlphaFold confidence & disorder, disease variants (`HbS Glu7Val`, `p53–MDM2`), and multi-domain architecture.
- **Reproducible Science Exports:** Share exact exploration states via URL query parameters or download **SHA-256 checksummed JSON provenance records** and **FASTA** files.

---

## 1. Quick Start (Starting the Application)

### Prerequisites

- **Option A (Native Go):** **Go `1.24+`** (zero external Go module dependencies, zero Node/npm build step required; all frontend vendor bundles are pre-pinned in `static/vendor/`).
- **Option B (Docker):** **Docker `20.10+`**.

### Option A: Start Locally with Go

```bash
# 1. Clone the repository
git clone https://github.com/kapombe-k/Biostat.git
cd Biostat

# 2. Start the server (defaults to 0.0.0.0:7860)
go run ./cmd/server
```

Open **`http://localhost:7860`** in your browser.

Or build a standalone static binary (`~17 MB`, includes all HTML templates and curated classroom seed structures):

```bash
make build
./bioscope
```

### Option B: Start with Docker

```bash
# Build the multi-stage production container (runs automated Go tests during build)
docker build -t bioscope:latest .

# Run on port 7860 (stateless, CPU-only, non-root UID 1000)
docker run --rm -p 7860:7860 -e PORT=7860 bioscope:latest
```

Verify container health:

```bash
curl -s http://localhost:7860/healthz
```

### Option C: Deploy to Hugging Face Spaces

This repository includes the required Hugging Face Spaces YAML header at the top of `README.md` (`sdk: docker`, `app_port: 7860`) and runs cleanly as non-root `UID 1000` with ephemeral memory-only state and zero GPU requirements:

1. Create a new **Docker** Space on Hugging Face.
2. Push this repository to your Space remote (`git push space main`).
3. Hugging Face builds `Dockerfile`, exposes port `7860`, and monitors `/healthz` automatically.

### Using BioScope (End-User Walkthrough)

1. **Explore a Showcase or Custom Target:** Click any showcase card on the home page (e.g., *Human Oxyhemoglobin `1GZX`*, *Tumor Suppressor p53 `AF-P04637-F1`*, *ABL1 Kinase + Imatinib `2HYY`*, *Bovine Trypsin `1GBT`*) or type any gene symbol, UniProt accession, PDB ID, or FASTA sequence into the top omnibox.
2. **Inspect & Manipulate 3D Structure:** Drag to rotate, scroll to zoom, right-drag to pan. Use the stage toolbar to switch **Representations** (`Cartoon`, `Ball & Stick`, `Spacefill`, `Backbone`), **Color Schemes** (`pLDDT`, `Secondary Structure`, `Chain`, `B-factor`, `Hydrophobicity`), toggle **Molecular Surface** / **Ligands**, focus individual **HETATM binding pockets**, or switch engines between **Mol\***, **3Dmol.js**, and **2D Topology**.
3. **Link 1D Sequence <-> 3D Atoms:** Click any colored residue tile in the bottom **Sequence-to-Structure Matrix** to zoom the 3D camera onto that exact chain, author sequence ID, and insertion code — or click any residue directly on the 3D model to highlight and scroll its sequence tile into view. Dashed gray tiles denote unmodeled crystallographic loops or cleaved pro-peptides.
4. **Run Sequence Similarity Search:** Open the **BLAST** tab in the right inspector to run either an instant **Classroom Reference Homology Search** or an asynchronous **Live NCBI QBlast** job, then jump directly into any homologous hit's 3D workspace.
5. **Take Guided Lessons:** Click **Guided Lessons** in the top navigation bar to step through interactive modules with automatic 3D camera choreography and instant quiz explanations.

---

## 2. Maintenance & Operations Guide

### Configuration & Environment Variables

BioScope follows 12-factor configuration via environment variables (`internal/config/config.go`). Every setting has safe production defaults:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `BIOSCOPE_HOST` | `0.0.0.0` | Network interface bind address |
| `PORT` / `BIOSCOPE_PORT` | `7860` | HTTP listening port (`PORT` takes precedence for container platforms) |
| `BIOSCOPE_LOG_LEVEL` | `info` | Structured JSON log verbosity (`debug`, `info`, `warn`, `error`) |
| `BIOSCOPE_CONTACT_EMAIL` | `bioscope-maintainer@example.org` | Polite contact header sent in `User-Agent` and NCBI BLAST `EMAIL` parameter (**set this to your institutional email in public deployments**) |
| `BIOSCOPE_OFFLINE_FALLBACK` | `true` | Enables curated classroom snapshot fallback (`delivery_mode: "curated-seed-fallback"`) if live upstream hosts are unreachable |
| `BIOSCOPE_HTTP_TIMEOUT_SECONDS` | `12` | Per-request timeout (`1–120s`) for outbound scientific API requests |
| `BIOSCOPE_MAX_RETRIES` | `2` | Maximum exponential-backoff retries on transient HTTP `429` / `5xx` errors |
| `BIOSCOPE_MAX_RESPONSE_MB` | `32` | Upper safety limit (MiB) on downloaded coordinate/API payloads |
| `BIOSCOPE_CACHE_MAX_ITEMS` | `512` | Max entries per bounded in-memory LRU cache (`proteins`, `searches`, `structures`, `coordinates`) |
| `BIOSCOPE_CACHE_TTL_MINUTES` | `60` | Cache entry time-to-live in minutes |
| `BIOSCOPE_BLAST_POLL_MIN_SECONDS` | `60` | Minimum interval between upstream NCBI status polls for the same `RID` (**never set `< 60` against public NCBI servers**) |
| `BIOSCOPE_BLAST_SUBMIT_MIN_INTERVAL_MS` | `3500` | Global minimum spacing (ms) between outbound BLAST job submissions |
| `BIOSCOPE_BLAST_MAX_JOBS` | `32` | Maximum retained concurrent/recent BLAST jobs in memory |
| `BIOSCOPE_EXTRA_ALLOWED_HOSTS` | *(empty)* | Optional comma-separated hostnames added to the outbound SSRF allowlist when using self-hosted mirrors |
| `BIOSCOPE_UNIPROT_BASE_URL` | `https://rest.uniprot.org` | Override UniProt REST base URL |
| `BIOSCOPE_RCSB_SEARCH_URL` | `https://search.rcsb.org` | Override RCSB Search v2 base URL |
| `BIOSCOPE_RCSB_DATA_URL` | `https://data.rcsb.org` | Override RCSB Data v1 base URL |
| `BIOSCOPE_RCSB_FILES_URL` | `https://files.rcsb.org` | Override RCSB coordinate archive base URL |
| `BIOSCOPE_ALPHAFOLD_BASE_URL` | `https://alphafold.ebi.ac.uk` | Override AlphaFold DB API base URL |
| `BIOSCOPE_PDBE_BASE_URL` | `https://www.ebi.ac.uk/pdbe` | Override PDBe / SIFTS base URL |
| `BIOSCOPE_BLAST_BASE_URL` | `https://blast.ncbi.nlm.nih.gov/Blast.cgi` | Override NCBI QBlast CGI endpoint |

### Health Monitoring & Observability

- **Liveness / Readiness Probe:** `GET /healthz` returns HTTP `200 OK` with live cache utilization (`items`, `max_items`, `hits`, `misses`, `evictions`), version, and process uptime in seconds:
  ```json
  {
    "service": "bioscope",
    "status": "ok",
    "uptime_sec": 420,
    "version": "1.0.0",
    "cache": {
      "coordinates": {"items": 6, "max_items": 272, "hits": 38, "misses": 6, "evictions": 0},
      "proteins": {"items": 8, "max_items": 512, "hits": 44, "misses": 8, "evictions": 0}
    }
  }
  ```
- **Structured Logs:** All requests and upstream events are emitted as newline-delimited JSON to `stdout` via Go's standard `log/slog`.

### Updating Pinned Frontend Vendor Assets

Frontend assets are intentionally self-hosted under `static/vendor/` with explicit version filenames so upgrades are auditable and reproducible:

1. Download the verified release tarball from `npmjs.org`:
   ```bash
   mkdir -p /tmp/vendor-update && cd /tmp/vendor-update
   npm pack molstar@5.13.1 3dmol@2.5.5 htmx.org@2.0.11
   ```
2. Extract the production bundle + license file into `static/vendor/`, update script/stylesheet references in `internal/templates/layout.html` if bumping version numbers, and run `make race` before committing.

### Operational Troubleshooting

- **3D Viewer Falls Back to 2D SVG Diagram:** Browser hardware acceleration or WebGL is disabled. Enable `"Use hardware acceleration when available"` in Chrome/Firefox/Edge settings, or click **3Dmol.js** in the viewer toolbar for lower-memory GPU environments.
- **Live NCBI BLAST Takes 30–90 Seconds:** NCBI's public QBlast cluster queues `blastp` jobs asynchronously during peak hours and enforces `≤ 1 poll / 60s / RID`. For instant zero-queue classroom demonstrations, select **Classroom Reference Search** in the BLAST panel.
- **Air-Gapped / Restricted Network Deployments:** Keep `BIOSCOPE_OFFLINE_FALLBACK=true` (default). After 2 failed outbound connection attempts to blocked hosts, BioScope's per-host circuit breaker trips automatically so subsequent classroom queries resolve locally in `< 5 ms`.

---

## 3. Developer Guide

### Repository Structure

```text
Biostat/
├── cmd/server/main.go             # HTTP server entrypoint & graceful shutdown
├── internal/
│   ├── bio/                       # Domain models, query classifier, FASTA validator, PDB/mmCIF parser
│   ├── cache/                     # Generic bounded LRU+TTL cache & singleflight deduplicator
│   ├── clients/                   # Upstream client interfaces, SSRF-safe HTTP client & adapters
│   ├── config/                    # Env configuration & outbound host allowlist
│   ├── handlers/                  # HTTP mux, CSP/security headers, HTMX partials, JSON/FASTA exports
│   ├── lessons/                   # 5 interactive structural biology lessons & quiz evaluator
│   ├── seed/                      # Embedded PDB/mmCIF structures & curated classroom catalog
│   ├── services/                  # Explorer orchestrator, SIFTS+SW mapper, async BlastManager
│   └── templates/                 # Embedded Go html/template pages & partials
├── static/
│   ├── css/bioscope.css           # Scientific workstation UI styles
│   ├── js/bioscope.js             # Mol* MVSJ + 3Dmol.js + bidirectional sequence controller
│   └── vendor/                    # Pinned Mol* 5.13.1, 3Dmol.js 2.5.5, HTMX 2.0.11 + licenses
├── docs/
│   ├── ARCHITECTURE.md            # Detailed system design, data flows & alignment algorithms
│   ├── SOURCES.md                 # Upstream API schemas, citations, licenses & scientific caveats
│   └── TESTING.md                 # Test suite matrix & coverage report
├── Dockerfile                     # Multi-stage production container build (PORT=7860, UID 1000)
├── Makefile                       # Common dev, test, build & Docker targets
└── README.md                      # User, maintainer & developer documentation
```

### Common Development Commands

```bash
make run          # Start local dev server on http://localhost:7860
make vet          # Run static analysis (`go vet ./...`)
make test         # Run unit + integration tests
make race         # Run all tests with Go's race detector (`go test -race -v ./...`)
make cover        # Generate coverage profile across packages
make build        # Compile stripped static binary `./bioscope`
make docker-run   # Build & launch production container on port 7860
```

### Extending Guided Lessons (`internal/lessons/lessons.go`)

1. Append a new `Lesson` entry in `internal/lessons/lessons.go` containing `LessonStep` entries (with a declarative `SceneSpec`: `ProteinAcc`, `StructureID`, `ActiveChain`, `Representation`, `ColorScheme`, `FocusResidues`, `Distances`, `Labels`) and `QuizQuestion` items.
2. Run `go test -v ./internal/lessons`: `TestLessonsIntegrityAndExactCoordinateAnchors` automatically parses every referenced PDB/mmCIF structure and verifies that every `FocusResidue`, `FocusLigand`, `DistanceMeasurement` atom pair, and `Label3D` anchor exists in the atomic coordinates!

### Adding or Modifying Upstream Scientific Clients (`internal/clients/`)

1. Define or extend the typed interface in `internal/clients/interfaces.go`.
2. Route all outbound requests through `*SafeHTTPClient` (`internal/clients/httpclient.go`) so SSRF host validation, timeouts, `MaxResponseBytes`, exponential retries, and per-host circuit breakers apply automatically.
3. Add recorded JSON/XML fixture unit tests in `internal/clients/clients_test.go` using `httptest.NewServer`.

---

## 4. Further Documentation & Scientific Attribution

- **[`docs/SOURCES.md`](docs/SOURCES.md):** Complete documentation of UniProtKB (CC BY 4.0), RCSB PDB (CC0 1.0), AlphaFold DB (CC BY 4.0), PDBe / SIFTS (CC0 1.0), NCBI BLAST+ fair-usage policies, vendored library licenses, and structural biology caveats.
- **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md):** Deep dive into server architecture, hybrid SIFTS + Smith–Waterman residue mapping, declarative `MolViewSpec` synchronization, security posture, and caching.
- **[`docs/TESTING.md`](docs/TESTING.md):** Full test suite breakdown and recorded coverage metrics.
