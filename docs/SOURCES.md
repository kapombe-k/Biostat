# Scientific Data Sources, Attribution & Limitations

BioScope integrates five authoritative open-access molecular biology databases alongside pinned open-source client visualization libraries. Every record displayed or exported by BioScope carries explicit provenance metadata (`source_name`, `record_id`, `record_status`, `endpoint_url`, `retrieved_at`, `delivery_mode`, `citation`, `license`, and `sha256` coordinate digest).

---

## 1. Upstream APIs & Schemas

### 1.1 UniProt Knowledgebase (UniProtKB)

- **Provider:** The UniProt Consortium (EMBL-EBI, SIB, PIR)
- **Default Base URL:** `https://rest.uniprot.org` (`BIOSCOPE_UNIPROT_BASE_URL`)
- **Endpoints Used:**
  - `GET /uniprotkb/{accession}.json` — Full entry lookup by primary accession.
  - `GET /uniprotkb/search?query={query}&format=json&size={limit}` — Keyword, gene symbol, and cross-reference (`(xref:pdb-XXXX)`) discovery.
- **Extracted Fields:**
  - Primary accession, `uniProtkbId`, entry review status (`UniProtKB reviewed (Swiss-Prot)` vs `UniProtKB unreviewed (TrEMBL)`).
  - Recommended full name, alternative names, submitted names, and gene symbols.
  - Organism scientific name, common name, NCBI Taxonomy ID, and full taxonomic lineage.
  - Functional comments (`FUNCTION`, `CATALYTIC ACTIVITY`, `SUBUNIT`, `SUBCELLULAR LOCATION`, `DISEASE` with OMIM identifiers).
  - Sequence features (`Domain`, `Region`, `Chain`, `Signal`, `Propeptide`, `Active site`, `Binding site`, `Metal binding`, `Disulfide bond`, `Site`, `Mutagenesis`).
  - Cross-references (`PDB`, `AlphaFoldDB`, `GO`, `Pfam`, `InterPro`) and canonical amino acid sequence + molecular weight.
- **License & Attribution:** Creative Commons Attribution 4.0 International (**CC BY 4.0**).
- **Citation:** The UniProt Consortium (2025). *UniProt: the Universal Protein Knowledgebase*. Nucleic Acids Research.

---

### 1.2 RCSB Protein Data Bank (RCSB PDB)

- **Provider:** Research Collaboratory for Structural Bioinformatics / wwPDB
- **Default Base URLs:**
  - Search API v2: `https://search.rcsb.org` (`BIOSCOPE_RCSB_SEARCH_URL`)
  - Data API v1: `https://data.rcsb.org` (`BIOSCOPE_RCSB_DATA_URL`)
  - Coordinate Archive: `https://files.rcsb.org` (`BIOSCOPE_RCSB_FILES_URL`)
- **Endpoints Used:**
  - `POST /rcsbsearch/v2/query` — Polymer entity reference sequence lookup by UniProt accession and full-text structure search.
  - `GET /rest/v1/core/entry/{id}` — Experimental method (`X-RAY DIFFRACTION`, `ELECTRON MICROSCOPY`, `SOLUTION NMR`), resolution (`Å`), refinement `R-work` / `R-free`, deposition/release dates, primary citation & DOI, and bound non-polymer heterogen identifiers.
  - `GET /download/{ID}.cif` (with fallback `GET /download/{ID}.pdb`) — Atomic coordinate files.
- **License & Attribution:** **CC0 1.0 Universal Public Domain Dedication** (wwPDB archive data).
- **Citation:** Berman, H.M. et al. (2000). *The Protein Data Bank*. Nucleic Acids Research 28, 235–242; Burley, S.K. et al. (2025). *RCSB Protein Data Bank*. Nucleic Acids Research.

---

### 1.3 AlphaFold Protein Structure Database (AlphaFold DB)

- **Provider:** Google DeepMind & EMBL-EBI
- **Default Base URL:** `https://alphafold.ebi.ac.uk` (`BIOSCOPE_ALPHAFOLD_BASE_URL`)
- **Endpoints Used:**
  - `GET /api/prediction/{uniprotAccession}` — Model metadata (`entryId`, `latestVersion`, `globalMetricValue` mean pLDDT, confidence band fractions, `cifUrl`, `pdbUrl`, `paeImageUrl`).
  - `GET {cifUrl}` / `GET {pdbUrl}` — Predicted atomic coordinate files storing per-residue **pLDDT** (`0–100`) in the isotropic B-factor column (`_atom_site.B_iso_or_equiv` / columns `61–66`).
- **License & Attribution:** Creative Commons Attribution 4.0 International (**CC BY 4.0**).
- **Citation:** Jumper, J. et al. (2021). *Highly accurate protein structure prediction with AlphaFold*. Nature 596, 583–589; Varadi, M. et al. (2024). *AlphaFold Protein Structure Database in 2024*. Nucleic Acids Research.

---

### 1.4 PDB in Europe (PDBe & PDBe-KB / SIFTS)

- **Provider:** EMBL-EBI
- **Default Base URL:** `https://www.ebi.ac.uk/pdbe` (`BIOSCOPE_PDBE_BASE_URL`)
- **Endpoints Used:**
  - `GET /api/validation/global-percentiles/entry/{pdb_id}` — Experimental validation percentiles (`clashscore`, `percent-rama-outliers`, `percent-sidechain-outliers`).
  - `GET /api/mappings/uniprot/{pdb_id}` — **SIFTS** (Structure Integration with Function, Taxonomy and Sequences) residue-level intervals mapping polymer `label_seq_id` and `author_residue_number` ranges to canonical UniProt sequence positions.
- **License & Attribution:** **CC0 1.0 Public Domain** / EMBL-EBI Terms of Use.

---

### 1.5 NCBI BLAST+ URL API (QBlast)

- **Provider:** National Center for Biotechnology Information (NLM / NIH)
- **Default Base URL:** `https://blast.ncbi.nlm.nih.gov/Blast.cgi` (`BIOSCOPE_BLAST_BASE_URL`)
- **Operations Used:**
  - `POST ?CMD=Put` (`PROGRAM=blastp`, `DATABASE=swissprot|pdb|refseq_protein|nr`, `FORMAT_TYPE=XML`, `TOOL=BioScope`, `EMAIL=<ContactEmail>`) — Asynchronous job submission returning `RID` and estimated runtime `RTOE`.
  - `GET ?CMD=Get&FORMAT_OBJECT=SearchInfo&RID={RID}` — Asynchronous job status polling (`WAITING`, `READY`, `FAILED`, `UNKNOWN`).
  - `GET ?CMD=Get&FORMAT_TYPE=XML&RID={RID}` — Retrieval of `BlastOutput` XML alignments (`BitScore`, `E-value`, `Identity`, `QueryFrom..QueryTo`, `HitFrom..HitTo`, aligned sequences).
- **Strict Rate-Policy Enforcement (`internal/clients/blast.go` & `internal/services/blast_manager.go`):**
  1. Global minimum delay of **≥ 3.5 seconds** between submission requests (`BIOSCOPE_BLAST_SUBMIT_MIN_INTERVAL_MS`).
  2. Strict per-RID polling lock enforcing **at most 1 status poll per 60 seconds per job ID** (`BIOSCOPE_BLAST_POLL_MIN_SECONDS=60`), returning cached job progress + exact countdown seconds to browser HTMX polls in between.
  3. Bounded concurrent background job registry (`BIOSCOPE_BLAST_MAX_JOBS=32`).
- **Citation:** Camacho, C. et al. (2009). *BLAST+: architecture and applications*. BMC Bioinformatics 10, 421.

---

## 2. Pinned Client Visualization Dependencies

All browser libraries are version-pinned and vendored directly inside `static/vendor/` with accompanying upstream license texts (zero unversioned external CDN links):

| Library | Pinned Version | File Path | License | Role in BioScope |
| :--- | :--- | :--- | :--- | :--- |
| **Mol\*** (`molstar`) | `5.13.1` | `static/vendor/molstar-5.13.1.{js,css}` | MIT | Primary WebGL 3D molecular viewer + native `MolViewSpec` (`MVSJ`) declarative states |
| **3Dmol.js** (`3dmol`) | `2.5.5` | `static/vendor/3Dmol-2.5.5-min.js` | BSD-3-Clause | Lightweight alternative WebGL 3D molecular viewer & automatic fallback engine |
| **HTMX** (`htmx.org`) | `2.0.11` | `static/vendor/htmx-2.0.11.min.js` | Zero-Clause BSD | Server-rendered partial swaps, live omnibox search, async BLAST job polling, lesson steps & quizzes |

---

## 3. Curated Classroom Seed Snapshot & Scientific Caveats

### 3.1 Transparent Offline / Classroom Fallback (`BIOSCOPE_OFFLINE_FALLBACK`)
When deployed in air-gapped classrooms, restricted sandboxes, or during transient upstream maintenance windows, BioScope can fall back seamlessly (`BIOSCOPE_OFFLINE_FALLBACK=true`) to authentic embedded RCSB PDB and EMBL-EBI AlphaFold DB coordinate structures (`1GZX`, `AF-P69905-F1`, `1YCR`, `AF-P04637-F1`, `4INS`, `AF-P01308-F1`, `1LYZ`, `AF-P00698-F1`, `1GBT`, `1UBQ`, `2HYY`). Every UI badge and JSON export explicitly stamps `delivery_mode: "curated-seed-fallback"` vs `"live"` so data provenance is never ambiguous. Set `BIOSCOPE_OFFLINE_FALLBACK=false` to require strictly live upstream responses.

### 3.2 Key Structural Biology Caveats Enforced in the UI
1. **Predictions vs. Experimental Models Are Never Conflated:** Experimental structures report physical diffraction/microscopy metrics (`Resolution Å`, `R-free`, `B-factor Å²`, `Ramachandran outliers`), whereas AlphaFold predictions report **per-residue pLDDT confidence (`0–100`)**.
2. **Low-pLDDT Regions (`< 50`):** Indicate probable intrinsic disorder in isolated solution rather than static 3D coordinates, and inter-domain orientations require PAE validation.
3. **Missing Cofactors & Processing in Monomer Predictions:** Standard AlphaFold DB monomer entries omit non-protein prosthetic groups (such as `HEM` in `AF-P69905-F1`), quaternary partners, and proteolytic cleavages (such as uncleaved preproinsulin signal/C-peptides in `AF-P01308-F1`).
4. **Zero False Residue Equivalencies:** Author PDB residue numbers frequently diverge from canonical UniProt positions due to initiator `Met1` cleavage (`1GZX`), excised pro-peptides (`4INS`), crystallographic missing loop density (`2HYY`), heteromeric binding partners (`1YCR`), or historical homology insertion codes (`1GBT`: `65A`, `184A`, `188A`, `221A`). BioScope resolves and displays these explicitly rather than assuming `PDB index == UniProt index`.
