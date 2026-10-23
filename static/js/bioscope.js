/* BioScope Client Sync & Dual-Engine 3D Molecular Controller (Mol* 5.13.1 Primary + 3Dmol.js 2.5.5 Fallback) */
(function () {
  "use strict";

  const LEGENDS = {
    plddt: [
      { color: "#0053D6", label: "Very High (pLDDT ≥ 90)" },
      { color: "#65CBF3", label: "Confident (70–90)" },
      { color: "#FFDB13", label: "Low (50–70)" },
      { color: "#FF7D45", label: "Very Low (< 50 · Often Disordered)" }
    ],
    secondary: [
      { color: "#F43F5E", label: "α-Helix" },
      { color: "#F59E0B", label: "β-Strand / Sheet" },
      { color: "#94A3B8", label: "Loop / Coil" }
    ],
    chain: [
      { color: "#38BDF8", label: "Chain 1" },
      { color: "#34D399", label: "Chain 2" },
      { color: "#F472B6", label: "Chain 3" },
      { color: "#FBBF24", label: "Chain 4" }
    ],
    bfactor: [
      { color: "#2563EB", label: "Low B-Factor (Rigid Core)" },
      { color: "#F1F5F9", label: "Intermediate" },
      { color: "#EF4444", label: "High B-Factor (Mobile Loop/Tail)" }
    ],
    hydrophobicity: [
      { color: "#F59E0B", label: "Hydrophobic (I,V,L,F,C,M,A,W,Y)" },
      { color: "#06B6D4", label: "Polar / Charged Surface" }
    ]
  };

  function hasWebGLSupport() {
    try {
      const c = document.createElement("canvas");
      return !!(window.WebGLRenderingContext && (c.getContext("webgl2") || c.getContext("webgl") || c.getContext("experimental-webgl")));
    } catch (_) {
      return false;
    }
  }

  const BioScopeApp = {
    state: {
      structureId: "",
      format: "pdb",
      proteinAcc: "",
      activeChain: "A",
      representation: "cartoon",
      colorScheme: "secondary",
      showSurface: false,
      showLigands: true,
      focusLigand: "",
      focusResidues: [],
      cameraMode: "overview",
      stepBannerTitle: "",
      engine: "molstar",
      spinning: false
    },
    molstarViewer: null,
    molstarInitPromise: null,
    viewer3dmol: null,
    coordCache: new Map(),
    annCache: new Map(),
    webglSupported: true,

    init() {
      this.webglSupported = hasWebGLSupport();
      this.bindGlobalDelegation();
      this.bootFromPageJSON();
    },

    bootFromPageJSON() {
      const el = document.getElementById("bioscope-scene-json");
      if (!el) return;
      try {
        const initial = JSON.parse(el.textContent || "{}");
        if (initial && initial.structure_id) {
          Object.assign(this.state, {
            structureId: initial.structure_id,
            format: initial.format || "pdb",
            proteinAcc: initial.protein_accession || "",
            activeChain: initial.active_chain || "A",
            representation: initial.representation || "cartoon",
            colorScheme: initial.color_scheme || "secondary",
            showSurface: !!initial.show_surface,
            showLigands: initial.show_ligands !== false,
            focusLigand: initial.focus_ligand || "",
            focusResidues: Array.isArray(initial.focus_residues) ? initial.focus_residues : [],
            cameraMode: initial.camera_mode || "overview",
            stepBannerTitle: initial.step_banner_title || "",
            engine: !this.webglSupported ? "summary2d" : (initial.engine || "molstar")
          });
          this.syncToolbarControls();
          this.renderScene();
        }
      } catch (err) {
        console.error("[BioScope] Failed to parse initial scene JSON:", err);
      }
    },

    setLoading(visible, message) {
      const box = document.getElementById("viewer-loading");
      const msg = document.getElementById("viewer-loading-msg");
      if (msg && message) msg.textContent = message;
      if (box) box.hidden = !visible;
    },

    updateLegend() {
      const container = document.getElementById("viewer-legend-swatches");
      if (!container) return;
      const items = LEGENDS[this.state.colorScheme] || LEGENDS.secondary;
      container.innerHTML = items.map(it =>
        `<span class="swatch-item"><span class="swatch-dot" style="background:${it.color}"></span>${it.label}</span>`
      ).join("");

      const banner = document.getElementById("viewer-overlay-badge");
      if (banner) {
        const parts = [this.state.structureId];
        if (this.state.stepBannerTitle) {
          parts.push(this.state.stepBannerTitle);
        } else {
          parts.push(this.state.representation.replace(/_/g, " "));
        }
        banner.textContent = parts.join(" · ");
      }
      this.updateShareLink();
    },

    updateShareLink() {
      const shareInput = document.getElementById("share-url-input");
      const jsonExportBtn = document.getElementById("export-json-btn");
      const params = new URLSearchParams();
      if (this.state.proteinAcc) params.set("q", this.state.proteinAcc);
      if (this.state.structureId) params.set("struct", this.state.structureId);
      if (this.state.activeChain) params.set("chain", this.state.activeChain);
      if (this.state.representation) params.set("rep", this.state.representation);
      if (this.state.colorScheme) params.set("color", this.state.colorScheme);
      if (this.state.engine) params.set("engine", this.state.engine);
      if (this.state.focusResidues.length > 0) {
        params.set("focus", this.state.focusResidues.map(r => `${r.chain_id}:${r.auth_seq_id}${r.ins_code || ""}`).join(","));
      }
      const shareURL = `${window.location.origin}/explore?${params.toString()}`;
      if (shareInput) shareInput.value = shareURL;
      if (jsonExportBtn) jsonExportBtn.href = `/api/export/json?${params.toString()}`;
    },

    syncToolbarControls() {
      const repSel = document.getElementById("ctrl-rep");
      if (repSel) repSel.value = this.state.representation;
      const colSel = document.getElementById("ctrl-color");
      if (colSel) colSel.value = this.state.colorScheme;
      const engSel = document.getElementById("ctrl-engine");
      if (engSel) engSel.value = this.state.engine;
      const surfBtn = document.getElementById("ctrl-surface");
      if (surfBtn) surfBtn.classList.toggle("is-active", this.state.showSurface);
      const ligBtn = document.getElementById("ctrl-ligands");
      if (ligBtn) ligBtn.classList.toggle("is-active", this.state.showLigands);
      this.updateLegend();
    },

    coordURL(id) {
      return `${window.location.origin}/api/structures/${encodeURIComponent(id)}/coordinates`;
    },

    annotationsURL(id, scheme) {
      const p = new URLSearchParams({ scheme: scheme || "secondary" });
      if (this.state.proteinAcc) p.set("q", this.state.proteinAcc);
      return `${window.location.origin}/api/structures/${encodeURIComponent(id)}/annotations?${p.toString()}`;
    },

    async fetchAnnotations(id, scheme) {
      const key = `${id}::${scheme}::${this.state.proteinAcc}`;
      if (this.annCache.has(key)) return this.annCache.get(key);
      const resp = await fetch(this.annotationsURL(id, scheme));
      if (!resp.ok) throw new Error(`Annotations HTTP ${resp.status}`);
      const data = await resp.json();
      this.annCache.set(key, data);
      return data;
    },

    async fetchCoordinatesText(id) {
      if (this.coordCache.has(id)) return this.coordCache.get(id);
      const resp = await fetch(this.coordURL(id));
      if (!resp.ok) throw new Error(`Coordinates HTTP ${resp.status}`);
      const formatHdr = resp.headers.get("X-Structure-Format") || (resp.headers.get("Content-Type") || "").includes("cif") ? "mmcif" : "pdb";
      const text = await resp.text();
      const entry = { text, format: text.trimStart().startsWith("data_") ? "mmcif" : formatHdr };
      this.coordCache.set(id, entry);
      return entry;
    },

    async ensureMolstar() {
      if (this.molstarViewer) return this.molstarViewer;
      if (this.molstarInitPromise) return this.molstarInitPromise;
      if (!window.molstar || !window.molstar.Viewer) {
        throw new Error("Mol* bundle unavailable");
      }
      const mount = document.getElementById("bioscope-molstar-mount");
      if (!mount) throw new Error("Mol* mount element missing");

      this.molstarInitPromise = window.molstar.Viewer.create(mount, {
        layoutIsExpanded: false,
        layoutShowControls: false,
        layoutShowRemoteState: false,
        layoutShowSequence: false,
        layoutShowLog: false,
        layoutShowLeftPanel: false,
        viewportShowExpand: true,
        viewportShowSelectionMode: false,
        viewportShowAnimation: false
      }).then(viewer => {
        this.molstarViewer = viewer;
        this.bindMolstarPicking(viewer);
        return viewer;
      });
      return this.molstarInitPromise;
    },

    bindMolstarPicking(viewer) {
      try {
        const plugin = viewer.plugin;
        if (!plugin || !plugin.behaviors || !plugin.behaviors.interaction) return;
        plugin.behaviors.interaction.click.subscribe(ev => {
          const loci = ev && ev.current && ev.current.loci;
          if (!loci || loci.kind !== "element-loci" || !loci.elements || loci.elements.length === 0) return;
          const elem = loci.elements[0];
          const unit = elem.unit;
          if (!unit || !unit.model || !unit.model.atomicHierarchy) return;
          // Resolve first atom index inside clicked OrderedSet
          const indices = elem.indices;
          const unitAtomIdx = typeof indices === "number" ? indices : (indices && indices.length ? indices[0] : 0);
          const atomIdx = unit.elements ? unit.elements[unitAtomIdx] : unitAtomIdx;
          const h = unit.model.atomicHierarchy;
          const rIdx = h.residueAtomSegments.index[atomIdx];
          const cIdx = h.chainAtomSegments.index[atomIdx];
          const authAsym = h.chains.auth_asym_id.value(cIdx);
          const authSeq = h.residues.auth_seq_id.value(rIdx);
          const insCode = h.residues.pdbx_PDB_ins_code.value(rIdx) || "";
          this.onPickedFrom3D(authAsym, authSeq, insCode);
        });
      } catch (err) {
        console.warn("[BioScope] Mol* click binding note:", err);
      }
    },

    buildMVSTree(format) {
      const parseFmt = format === "mmcif" ? "mmcif" : "pdb";
      const annURI = this.annotationsURL(this.state.structureId, this.state.colorScheme);

      const polymerChildren = [
        {
          kind: "representation",
          params: { type: this.state.representation || "cartoon" },
          children: [
            {
              kind: "color_from_uri",
              params: { uri: annURI, format: "json", schema: "auth_residue", field_name: "color" }
            }
          ]
        },
        {
          kind: "tooltip_from_uri",
          params: { uri: annURI, format: "json", schema: "auth_residue", field_name: "tooltip" }
        }
      ];

      const structChildren = [
        {
          kind: "component",
          params: { selector: "polymer" },
          children: polymerChildren
        }
      ];

      if (this.state.showSurface) {
        structChildren.push({
          kind: "component",
          params: { selector: "polymer" },
          children: [
            {
              kind: "representation",
              params: { type: "surface", surface_type: "molecular" },
              children: [
                {
                  kind: "color_from_uri",
                  params: { uri: annURI, format: "json", schema: "auth_residue", field_name: "color" }
                },
                {
                  kind: "opacity",
                  params: { opacity: 0.25 }
                }
              ]
            }
          ]
        });
      }

      if (this.state.showLigands) {
        structChildren.push({
          kind: "component",
          params: { selector: "ligand" },
          children: [
            {
              kind: "representation",
              params: { type: "ball_and_stick", size_factor: 1.3 }
            }
          ]
        });
        structChildren.push({
          kind: "component",
          params: { selector: "ion" },
          children: [
            {
              kind: "representation",
              params: { type: "spacefill", size_factor: 0.85 }
            }
          ]
        });
      }

      const focusExpressions = [];
      for (const fr of this.state.focusResidues) {
        const sel = {
          auth_asym_id: fr.chain_id,
          auth_seq_id: Number(fr.auth_seq_id)
        };
        if (fr.ins_code) sel.pdbx_PDB_ins_code = fr.ins_code;
        focusExpressions.push(sel);

        const resChildren = [
          {
            kind: "representation",
            params: { type: "ball_and_stick", size_factor: 1.35 },
            children: [{ kind: "color", params: { color: "#38BDF8" } }]
          }
        ];
        if (fr.label) {
          resChildren.push({
            kind: "label",
            params: { text: fr.label }
          });
        }
        structChildren.push({
          kind: "component",
          params: { selector: sel },
          children: resChildren
        });
      }

      if (this.state.focusLigand) {
        focusExpressions.push({ label_comp_id: this.state.focusLigand });
      }

      if (this.state.cameraMode === "selection" && focusExpressions.length > 0) {
        structChildren.push({
          kind: "component",
          params: { selector: focusExpressions },
          children: [{ kind: "focus", params: {} }]
        });
      }

      return {
        kind: "root",
        children: [
          { kind: "canvas", params: { background_color: "#0b1120" } },
          {
            kind: "download",
            params: { url: this.coordURL(this.state.structureId) },
            children: [
              {
                kind: "parse",
                params: { format: parseFmt },
                children: [
                  {
                    kind: "structure",
                    params: { type: "model" },
                    children: structChildren
                  }
                ]
              }
            ]
          }
        ]
      };
    },

    async renderScene() {
      if (!this.state.structureId) return;
      const mountMol = document.getElementById("bioscope-molstar-mount");
      const mount3D = document.getElementById("bioscope-3dmol-mount");
      const mount2D = document.getElementById("bioscope-fallback-mount");
      if (!mountMol || !mount3D || !mount2D) return;

      this.updateLegend();
      this.highlightFocusedInStrip();

      if (!this.webglSupported && this.state.engine !== "summary2d") {
        this.state.engine = "summary2d";
        this.syncToolbarControls();
      }

      this.setLoading(true, `Rendering ${this.state.structureId} (${this.state.engine})…`);
      try {
        // Probe coordinate header once to know exact format (mmcif vs pdb) cleanly
        const coordInfo = await this.fetchCoordinatesText(this.state.structureId);
        this.state.format = coordInfo.format;

        if (this.state.engine === "molstar") {
          mountMol.hidden = false;
          mount3D.hidden = true;
          mount2D.hidden = true;
          try {
            const viewer = await this.ensureMolstar();
            const mvs = this.buildMVSTree(coordInfo.format);
            await viewer.loadMvsData(JSON.stringify(mvs), "mvsj", { replaceExisting: true });
            viewer.handleResize();
          } catch (molErr) {
            console.warn("[BioScope] Mol* fallback to 3Dmol.js:", molErr);
            this.state.engine = "3dmol";
            this.syncToolbarControls();
            await this.render3Dmol(coordInfo, mountMol, mount3D, mount2D);
          }
        } else if (this.state.engine === "3dmol") {
          await this.render3Dmol(coordInfo, mountMol, mount3D, mount2D);
        } else {
          await this.renderFallbackSummary(mountMol, mount3D, mount2D);
        }
      } catch (err) {
        console.error("[BioScope] Viewer render failed; showing accessible summary:", err);
        await this.renderFallbackSummary(mountMol, mount3D, mount2D, String(err));
      } finally {
        this.setLoading(false);
      }
    },

    async render3Dmol(coordInfo, mountMol, mount3D, mount2D) {
      mountMol.hidden = true;
      mount3D.hidden = false;
      mount2D.hidden = true;

      if (!window.$3Dmol) throw new Error("3Dmol.js unavailable");
      if (!this.viewer3dmol) {
        this.viewer3dmol = window.$3Dmol.createViewer(mount3D, {
          backgroundColor: "#0b1120",
          antialias: true
        });
      }
      const v = this.viewer3dmol;
      v.clear();

      const fmt = coordInfo.format === "mmcif" ? "cif" : "pdb";
      v.addModel(coordInfo.text, fmt);
      const anns = await this.fetchAnnotations(this.state.structureId, this.state.colorScheme);
      const colorLookup = new Map();
      for (const a of anns) {
        colorLookup.set(`${a.auth_asym_id}:${a.auth_seq_id}:${a.pdbx_PDB_ins_code || ""}`, a.color);
      }

      const colorfunc = atom => colorLookup.get(`${atom.chain}:${atom.resi}:${atom.icode || ""}`) || "#94A3B8";
      const styleObj = {};
      switch (this.state.representation) {
        case "ball_and_stick":
          styleObj.stick = { radius: 0.16, colorfunc };
          styleObj.sphere = { scale: 0.25, colorfunc };
          break;
        case "spacefill":
          styleObj.sphere = { colorfunc };
          break;
        case "backbone":
          styleObj.line = { linewidth: 3.0, colorfunc };
          break;
        default:
          styleObj.cartoon = { colorfunc, arrows: true };
          break;
      }
      v.setStyle({ hetflag: false }, styleObj);
      if (this.state.showLigands) {
        v.setStyle({ hetflag: true, not: { resn: ["HOH", "WAT"] } }, {
          stick: { radius: 0.22 },
          sphere: { scale: 0.3 }
        });
      } else {
        v.setStyle({ hetflag: true }, {});
      }

      if (this.state.showSurface) {
        v.addSurface(window.$3Dmol.SurfaceType.VDW, { opacity: 0.25, color: "#94A3B8" }, { hetflag: false });
      }

      const zoomTargets = [];
      for (const fr of this.state.focusResidues) {
        const sel = { chain: fr.chain_id, resi: Number(fr.auth_seq_id) };
        if (fr.ins_code) sel.icode = fr.ins_code;
        zoomTargets.push(sel);
        v.addStyle(sel, { stick: { radius: 0.24, color: "#38BDF8" }, sphere: { scale: 0.28, color: "#38BDF8" } });
        if (fr.label) {
          v.addLabel(fr.label, {
            backgroundColor: "#0f172a",
            fontColor: "#f8fafc",
            borderColor: "#38bdf8",
            borderThickness: 1,
            fontSize: 12,
            backgroundOpacity: 0.88
          }, sel);
        }
      }
      if (this.state.focusLigand) {
        zoomTargets.push({ resn: this.state.focusLigand });
      }

      v.setClickable({}, true, atom => {
        if (atom) this.onPickedFrom3D(atom.chain, atom.resi, atom.icode || "");
      });

      v.resize();
      if (this.state.cameraMode === "selection" && zoomTargets.length > 0) {
        v.zoomTo({ or: zoomTargets }, 450);
      } else {
        v.zoomTo({}, 350);
      }
      v.spin(this.state.spinning ? "y" : false);
      v.render();
    },

    async renderFallbackSummary(mountMol, mount3D, mount2D, reason) {
      mountMol.hidden = true;
      mount3D.hidden = true;
      mount2D.hidden = false;

      let anns = [];
      try {
        anns = await this.fetchAnnotations(this.state.structureId, this.state.colorScheme);
      } catch (_) {}

      const width = 680;
      const height = 120;
      const count = Math.max(anns.length, 1);
      const barW = Math.max((width - 40) / count, 1.5);
      const rects = anns.map((a, idx) => {
        const x = 20 + (idx / count) * (width - 40);
        return `<rect x="${x.toFixed(1)}" y="36" width="${barW.toFixed(2)}" height="42" fill="${a.color}"><title>${a.tooltip}</title></rect>`;
      }).join("");

      const note = !this.webglSupported
        ? "WebGL hardware/software context was unavailable in this browser environment — rendering accessible 2D secondary/confidence topology bar."
        : (reason ? `Fallback summary active (${reason}).` : "Accessible 2D Topology & Structural Summary View active.");

      mount2D.innerHTML = `
        <div style="padding:1.4rem;color:#e2e8f0;height:100%;overflow:auto;display:flex;flex-direction:column;justify-content:center;gap:0.85rem;">
          <div style="font-size:0.82rem;color:#93c5fd;background:#172554;padding:0.55rem 0.8rem;border-radius:6px;border:1px solid #1d4ed8;">
            ${note}
          </div>
          <div>
            <div style="font-size:0.85rem;font-weight:700;margin-bottom:0.35rem;">Linear Topology & ${this.state.colorScheme.toUpperCase()} Track — ${this.state.structureId} (${anns.length} observed residues)</div>
            <svg viewBox="0 0 ${width} ${height}" style="width:100%;max-height:130px;background:#090f1d;border:1px solid #1e293b;border-radius:8px;">
              <text x="20" y="24" fill="#94a3b8" font-size="11" font-family="monospace">N-terminus</text>
              <text x="${width - 95}" y="24" fill="#94a3b8" font-size="11" font-family="monospace">C-terminus</text>
              ${rects}
              <text x="20" y="102" fill="#64748b" font-size="11">Hover bars or click any residue tile below to inspect coordinates, Secondary Structure, pLDDT/B-factor, and Ligand pockets.</text>
            </svg>
          </div>
        </div>
      `;
    },

    focusSingleResidue(tile) {
      const chain = tile.dataset.chain || "A";
      const auth = parseInt(tile.dataset.auth || "0", 10);
      const ins = tile.dataset.ins || "";
      const observed = tile.dataset.observed === "true";
      const summary = tile.dataset.summary || "";

      document.querySelectorAll(".res-tile.is-selected").forEach(el => el.classList.remove("is-selected"));
      tile.classList.add("is-selected");

      const readout = document.getElementById("live-residue-text");
      if (readout) readout.textContent = summary;

      if (!observed) {
        // Unmodeled missing density — do NOT assert false 3D coordinates!
        return;
      }

      const label = `${tile.dataset.comp} ${auth}${ins} (${chain})`;
      this.state.focusResidues = [{ chain_id: chain, auth_seq_id: auth, ins_code: ins, label }];
      this.state.cameraMode = "selection";
      this.updateShareLink();

      if (this.state.engine === "molstar" && this.molstarViewer) {
        const schemaItem = { auth_asym_id: chain, auth_seq_id: auth };
        if (ins) schemaItem.pdbx_PDB_ins_code = ins;
        this.molstarViewer.structureInteractivity({
          elements: { items: [schemaItem] },
          action: ["select", "focus"]
        });
      } else if (this.state.engine === "3dmol" && this.viewer3dmol) {
        const sel = { chain, resi: auth };
        if (ins) sel.icode = ins;
        this.viewer3dmol.zoomTo(sel, 380);
        this.viewer3dmol.render();
      }
    },

    onPickedFrom3D(chain, authSeq, insCode) {
      const cleanIns = insCode === "?" || insCode === "." ? "" : (insCode || "");
      const selector = `.res-tile[data-chain="${CSS.escape(String(chain))}"][data-auth="${CSS.escape(String(authSeq))}"][data-ins="${CSS.escape(cleanIns)}"]`;
      const tile = document.querySelector(selector);
      if (tile) {
        document.querySelectorAll(".res-tile.is-selected").forEach(el => el.classList.remove("is-selected"));
        tile.classList.add("is-selected");
        tile.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
        const readout = document.getElementById("live-residue-text");
        if (readout) readout.textContent = tile.dataset.summary || `Selected ${chain}:${authSeq}${cleanIns}`;
      } else {
        const readout = document.getElementById("live-residue-text");
        if (readout) {
          readout.textContent = `3D Pick: Chain ${chain} · Residue ${authSeq}${cleanIns} (switch to Chain ${chain} tab below to view full sequence strip)`;
        }
      }
    },

    highlightFocusedInStrip() {
      document.querySelectorAll(".res-tile.is-selected").forEach(el => el.classList.remove("is-selected"));
      let scrolled = false;
      for (const fr of this.state.focusResidues) {
        const sel = `.res-tile[data-chain="${CSS.escape(String(fr.chain_id))}"][data-auth="${CSS.escape(String(fr.auth_seq_id))}"][data-ins="${CSS.escape(fr.ins_code || "")}"]`;
        const tile = document.querySelector(sel);
        if (tile) {
          tile.classList.add("is-selected");
          if (!scrolled) {
            tile.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
            const readout = document.getElementById("live-residue-text");
            if (readout && tile.dataset.summary) readout.textContent = tile.dataset.summary;
            scrolled = true;
          }
        }
      }
    },

    bindGlobalDelegation() {
      document.addEventListener("change", ev => {
        const t = ev.target;
        if (!t) return;
        if (t.id === "ctrl-rep") {
          this.state.representation = t.value;
          this.renderScene();
        } else if (t.id === "ctrl-color") {
          this.state.colorScheme = t.value;
          this.renderScene();
        } else if (t.id === "ctrl-engine") {
          this.state.engine = t.value;
          this.renderScene();
        }
      });

      document.addEventListener("click", ev => {
        const tile = ev.target.closest(".res-tile");
        if (tile) {
          this.focusSingleResidue(tile);
          return;
        }

        const tabBtn = ev.target.closest("[data-tab-target]");
        if (tabBtn) {
          const targetId = tabBtn.dataset.tabTarget;
          const card = tabBtn.closest(".inspector-card");
          if (card) {
            card.querySelectorAll(".tab-btn").forEach(b => b.classList.toggle("is-active", b === tabBtn));
            card.querySelectorAll(".tab-pane").forEach(p => {
              p.hidden = p.id !== targetId;
            });
          }
          return;
        }

        const focusBtn = ev.target.closest("[data-focus-residue]");
        if (focusBtn) {
          const chain = focusBtn.dataset.focusChain || this.state.activeChain || "A";
          const resi = parseInt(focusBtn.dataset.focusResidue, 10);
          const ins = focusBtn.dataset.focusIns || "";
          const lbl = focusBtn.dataset.focusLabel || `${chain}:${resi}${ins}`;
          this.state.focusResidues = [{ chain_id: chain, auth_seq_id: resi, ins_code: ins, label: lbl }];
          this.state.cameraMode = "selection";
          this.renderScene();
          return;
        }

        const ligFocusBtn = ev.target.closest("[data-focus-ligand]");
        if (ligFocusBtn) {
          this.state.focusLigand = ligFocusBtn.dataset.focusLigand;
          this.state.showLigands = true;
          this.state.cameraMode = "selection";
          this.syncToolbarControls();
          this.renderScene();
          return;
        }

        const sampleBtn = ev.target.closest("[data-fill-query]");
        if (sampleBtn) {
          const box = document.getElementById("omnibox-input");
          if (box) {
            box.value = sampleBtn.dataset.fillQuery;
            box.dispatchEvent(new Event("input", { bubbles: true }));
            box.focus();
          }
          return;
        }

        if (ev.target.closest("#ctrl-surface")) {
          this.state.showSurface = !this.state.showSurface;
          this.syncToolbarControls();
          this.renderScene();
        } else if (ev.target.closest("#ctrl-ligands")) {
          this.state.showLigands = !this.state.showLigands;
          this.syncToolbarControls();
          this.renderScene();
        } else if (ev.target.closest("#ctrl-reset")) {
          this.state.focusResidues = [];
          this.state.focusLigand = "";
          this.state.cameraMode = "overview";
          this.renderScene();
        } else if (ev.target.closest("#ctrl-spin")) {
          this.state.spinning = !this.state.spinning;
          ev.target.closest("#ctrl-spin").classList.toggle("is-active", this.state.spinning);
          if (this.viewer3dmol && this.state.engine === "3dmol") {
            this.viewer3dmol.spin(this.state.spinning ? "y" : false);
          }
        } else if (ev.target.closest("#copy-share-btn")) {
          const inp = document.getElementById("share-url-input");
          const btn = ev.target.closest("#copy-share-btn");
          if (inp && navigator.clipboard) {
            navigator.clipboard.writeText(inp.value).then(() => {
              const orig = btn.textContent;
              btn.textContent = "Copied!";
              setTimeout(() => { btn.textContent = orig; }, 1600);
            });
          }
        }
      });

      // Listen for server-triggered HTMX declarative scene updates (e.g. Lesson Steps & Proof Buttons!)
      document.body.addEventListener("bioscope:applyScene", ev => {
        const scene = ev.detail || {};
        if (!scene.structure_id) return;
        const structChanged = scene.structure_id !== this.state.structureId || scene.active_chain !== this.state.activeChain;
        Object.assign(this.state, {
          structureId: scene.structure_id,
          proteinAcc: scene.protein_accession || this.state.proteinAcc,
          activeChain: scene.active_chain || "A",
          representation: scene.representation || "cartoon",
          colorScheme: scene.color_scheme || "secondary",
          showSurface: !!scene.show_surface,
          showLigands: scene.show_ligands !== false,
          focusLigand: scene.focus_ligand || "",
          focusResidues: Array.isArray(scene.focus_residues) ? scene.focus_residues : [],
          cameraMode: scene.camera_mode || "overview",
          stepBannerTitle: scene.step_banner_title || ""
        });
        this.syncToolbarControls();
        if (structChanged && window.htmx && document.getElementById("sequence-panel-wrapper")) {
          window.htmx.ajax("GET",
            `/partials/sequence?q=${encodeURIComponent(this.state.proteinAcc)}&struct=${encodeURIComponent(this.state.structureId)}&chain=${encodeURIComponent(this.state.activeChain)}`,
            { target: "#sequence-panel-wrapper", swap: "innerHTML" }
          ).then(() => this.highlightFocusedInStrip());
        }
        this.renderScene();
      });

      document.body.addEventListener("htmx:afterSwap", ev => {
        if (ev.target && ev.target.id === "sequence-panel-wrapper") {
          this.highlightFocusedInStrip();
        }
      });
    }
  };

  window.BioScopeApp = BioScopeApp;
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => BioScopeApp.init());
  } else {
    BioScopeApp.init();
  }
})();
