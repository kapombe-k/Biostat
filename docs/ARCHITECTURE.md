# BioScope Architecture Guide

BioScope is designed as a fast, deterministic, single-binary **Go + HTMX + Dual-Engine 3D WebGL** application. All upstream network orchestration, rate governance, caching, coordinate parsing, sequence-to-structure mapping, quiz grading, and scene generation happen inside strongly typed Go packages on the server.

---

## 1. High-Level System Architecture

```text
+---------------------------------------------------------------------------------------------------+
|                                   Browser Client (No SPA Build Step)                              |
|                                                                                                   |
|  +---------------------------+    +-------------------------------+    +-----------------------+  |
|  |  Semantic HTML5 + HTMX    |    |  BioScopeViewer (bioscope.js) |    | Bidirectional Seq <-> |  |
|  |  - Live omnibox search    |<-->|  1. Mol* 5.13.1 (MVSJ trees)  |<-->| Structure Matrix      |  |
|  |  - Async BLAST polling    |    |  2. 3Dmol.js 2.5.5 fallback   |    | - Auth chain/res/ins  |  |
|  |  - HX-Trigger scene step  |    |  3. Non-WebGL 2D SVG diagram  |    | - Missing-atom guard  |  |
|  +-------------+-------------+    +---------------+---------------+    +-----------+-----------+  |
+----------------|----------------------------------|--------------------------------|--------------+
                 | HTML / HTMX Partials             | Coordinates & MVS Annotations  | URL State
                 v                                  v                                v
+---------------------------------------------------------------------------------------------------+
|                           Go HTTP Server (`cmd/server`, `internal/handlers`)                      |
|  - Security headers, iframe-compatible CSP, request logging, bounded readers (`MaxBytesReader`)   |
+---------------------------------------------------------------------------------------------------+
         |                                         |                                    |
         v                                         v                                    v
+-------------------------+       +----------------------------------+       +----------------------+
|    ExplorerService      |       |           BlastManager           |       |    Lessons Engine    |
|  (`internal/services`)  |       |       (`internal/services`)      |       | (`internal/lessons`) |
|  - Query classification |       |  - Input FASTA validation        |       |  - 5 guided modules  |
|  - Singleflight dedup   |       |  - Global >=3.5s submit throttle |       |  - Declarative scene |
|  - 4 bounded TTL caches |       |  - <= 1 poll / 60s / RID rule    |       |    step transitions  |
|  - MVS color/tooltips   |       |  - Bounded concurrent job map    |       |  - Server quiz eval  |
+------------+------------+       +----------------+-----------------+       +----------------------+
             |                                     |
             +------------------+------------------+
                                |
                                v
+---------------------------------------------------------------------------------------------------+
|        Scientific Domain Engine (`internal/bio`) & Mapper (`internal/services/mapper.go`)         |
|  - Pure-Go PDB & mmCIF parser (`ParseCoordinates`) extracting chains, insertion codes, B-factors, |
|    pLDDT distributions, HETATM binding pockets (<=4.2 A), and secondary structure                 |
|  - Hybrid SIFTS + affine-gap Smith-Waterman alignment ensuring zero false residue equivalencies   |
+---------------------------------------------------------------------------------------------------+
                                |
                                v
+---------------------------------------------------------------------------------------------------+
|           Resilient Upstream Scientific Clients (`internal/clients` behind interfaces)            |
|  - `SafeHTTPClient`: SSRF outbound host allowlist (incl. redirects), timeouts, exponential        |
|    backoff with `Retry-After`, response byte caps, and 45s per-host circuit breakers              |
|  - Clients: UniProtKB | RCSB Search & Data | AlphaFold DB | PDBe / SIFTS | NCBI BLAST+ XML        |
|  - Transparent Curated Classroom Seed Catalog (`internal/seed`) with explicit provenance flags     |
+---------------------------------------------------------------------------------------------------+
```

---

## 2. Package Layout & Responsibilities

| Directory | Responsibility |
| :--- | :--- |
| `cmd/server/` | Entrypoint (`main.go`): loads validated configuration, wires concrete clients into service interfaces, binds `0.0.0.0:$PORT`, and handles graceful `SIGINT`/`SIGTERM` shutdown. |
| `internal/config/` | Environment parsing, validation, defaults (`PORT=7860`), BLAST rate-limit thresholds, cache limits, and SSRF allowlist verification (`IsAllowedURL`). |
| `internal/bio/` | Core domain structs (`Protein`, `StructureSummary`, `ParsedStructure`, `ResidueMapping`, `BlastJob`, `ExplorationExport`), query classifier (`ClassifyQuery`), sequence validator, and pure-Go PDB + mmCIF coordinate parser (`coordparser.go`). |
| `internal/cache/` | Generic thread-safe bounded **LRU + TTL** cache (`Cache[K, V]`) with hit/miss/eviction metrics and generic in-flight request deduplicator (`FlightGroup[V]`). |
| `internal/clients/` | Service interfaces (`UniProtClient`, `RCSBClient`, `AlphaFoldClient`, `PDBeClient`, `BlastClient`), `SafeHTTPClient`, and concrete HTTP adapters. |
| `internal/seed/` | Embedded classroom structural snapshot (`embed.FS` over 11 authentic PDB/mmCIF coordinate files + curated protein metadata + local Smith-Waterman similarity search). |
| `internal/services/` | `ExplorerService` (search, workspace orchestration, MVS annotation generation, JSON export), `BuildResidueMapping` (SIFTS + sequence alignment), and `BlastManager` (async rate-governed job execution). |
| `internal/lessons/` | Five multi-step interactive structural biology lessons with exact 3D coordinate presets, distance measurements, labels, and graded comprehension quizzes. |
| `internal/templates/` | `embed.FS` Go `html/template` compiler pairing each top-level page with `layout.html` and shared HTMX partials (`partials/*.html`), auto-escaping untrusted text by default. |
| `internal/handlers/` | `net/http` `ServeMux` routes, security/CSP/logging middleware, HTMX fragment endpoints, MVS JSON annotation feeds, and export download handlers. |
| `static/` | Workstation stylesheet (`css/bioscope.css`), dual-engine viewer & sequence sync controller (`js/bioscope.js`), and version-pinned vendor bundles (`vendor/`). |

---

## 3. Sequence-to-Structure Mapping Algorithm (`internal/services/mapper.go`)

Real-world structural biology structures rarely share 1:1 residue numbering with canonical UniProt sequences. BioScope resolves every polymer chain through a strict two-stage pipeline:

1. **Stage 1 — Authoritative SIFTS Interval Anchoring (`PDBe /api/mappings/uniprot/{id}`):**
   - Matches polymer `entity_id` / chain segments against reported SIFTS `(uniprot_start..uniprot_end) <-> (label_seq_id start..end)` intervals.
   - Detects heteromeric non-self chains immediately when a polymer entity maps to a distinct UniProt accession (for example, `Chain A` in `1YCR` maps to `Q00987` MDM2 while `Chain B` maps to `P04637` p53), labeling the partner explicitly instead of force-aligning unrelated sequences.
2. **Stage 2 — Structure-Aware Local Smith-Waterman Verification:**
   - Aligns observed three-letter-decoded coordinate residues (preserving exact `auth_asym_id`, `auth_seq_id`, and `pdbx_PDB_ins_code`, e.g., `184` vs `184A` in `1GBT`) against canonical UniProt residues using BLOSUM62-style substitution scoring with gap penalties calibrated for crystallographic missing-loop density (`2HYY` `Glu276`) and excised pro-peptides (`4INS` C-peptide `55..89`).
   - Enforces a minimum identity threshold (`>= 45%` across `>= 8` residues) before asserting homology; chains falling below threshold are flagged `is_heteromeric_partner = true` with zero false residue links.
3. **Client Guardrails (`static/js/bioscope.js`):**
   - Clicking an observed residue tile calls `BioScopeViewer.focusResidue(chain, authSeq, insCode)` and synchronizes the URL (`?focus=A:184A`).
   - Clicking an unmodeled residue tile (`observed == false`) displays an explicit scientific explanation banner explaining why crystallographic electron density or processed peptide segments are absent rather than jumping to a wrong residue number.

---

## 4. Declarative Dual-Engine 3D Synchronization

- **Primary Viewer (`Mol* 5.13.1`):** Consumes declarative **MolViewSpec (`MVSJ`)** JSON trees built dynamically from the active `SceneSpec`. Custom coloring (`plddt`, `secondary`, `chain`, `bfactor`, `hydrophobicity`) and rich residue-level hover tooltips (`UniProt index + Author PDB index + insertion code + pLDDT/B-factor + UniProt functional features`) are streamed directly from `/api/structures/{id}/annotations?scheme=...` (`schema: "auth_residue"`), alongside 3D `distance_measurement` and `label` primitives.
- **Fallback Viewer (`3Dmol.js 2.5.5`):** Renders the identical coordinate stream, color scheme, surface overlays, ligand sticks, and clickable residue callbacks if switched manually by the user or automatically upon primary viewer failure.
- **Zero-WebGL Fallback (`summary2d`):** Renders an interactive 2D SVG structural domain, secondary-structure, confidence/B-factor, and active-site topology diagram whenever WebGL is unavailable in headless/restricted browsers.
- **HTMX Lesson Step Driving:** Navigating between steps (`GET /lessons/{slug}/step/{step}`) returns the rendered step HTML AND an HTTP `HX-Trigger: {"bioscope:applyScene": {...}}` header. `bioscope.js` listens for `bioscope:applyScene` on `document.body`, seamlessly reloading structures if needed (e.g., transitioning live from `1GBT` Trypsin to `2HYY` ABL1 Kinase–Imatinib), updating representations/surfaces, drawing 3D distance callouts, and refreshing the Sequence-to-Structure panel.
