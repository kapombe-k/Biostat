package templates

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/kapombe-k/Biostat/internal/bio"
)

//go:embed *.html partials/*.html
var templateFS embed.FS

// Renderer compiles and renders full HTML pages and HTMX partial fragments safely.
type Renderer struct {
	pages    map[string]*template.Template
	partials *template.Template
}

// NewRenderer parses all embedded templates with auto-escaping enabled and strict startup verification.
func NewRenderer() (*Renderer, error) {
	funcs := template.FuncMap{
		"toJSON": func(v any) template.JS {
			// encoding/json escapes '<', '>', '&', U+2028, and U+2029 by default in Go stdlib,
			// making application/json script data islands completely immune to tag-breakout injection.
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"derefFloat": func(p *float64, format string) string {
			if p == nil {
				return "—"
			}
			return fmt.Sprintf(format, *p)
		},
		"join": strings.Join,
		"upper": strings.ToUpper,
		"add": func(a, b int) int { return a + b },
		"formatEVal": func(v float64) string {
			if v == 0 {
				return "0.0"
			}
			if v < 0.001 {
				return fmt.Sprintf("%.2e", v)
			}
			return fmt.Sprintf("%.3g", v)
		},
		"resSummary": func(ch *bio.ChainModel, r bio.ResidueNode, kind bio.StructureKind) string {
			parts := []string{fmt.Sprintf("Chain %s · %s %s (label #%d)", r.ChainID, r.CompID, r.AuthLabel, r.LabelSeqID)}
			if !r.Observed {
				parts = append(parts, "UNMODELED IN 3D (missing crystallographic density)")
			}
			if r.UniProtPos != nil && ch != nil && ch.UniProtAccession != "" {
				parts = append(parts, fmt.Sprintf("UniProt %s %s%d", ch.UniProtAccession, r.UniProtAA, *r.UniProtPos))
			} else {
				parts = append(parts, "Unmapped to primary UniProt entry")
			}
			if r.Observed {
				parts = append(parts, "2° "+r.Secondary)
				if kind == bio.StructurePredicted {
					parts = append(parts, fmt.Sprintf("pLDDT %.1f (%s)", r.MetricValue, r.ConfidenceBand))
				} else {
					parts = append(parts, fmt.Sprintf("B-factor %.1f Å²", r.MetricValue))
				}
			}
			if len(r.FeatureLabels) > 0 {
				parts = append(parts, r.FeatureLabels[0])
			}
			return strings.Join(parts, "  |  ")
		},
	}

	pageNames := []string{"home", "workspace", "lessons_index", "lesson_view", "error"}
	pages := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(templateFS, "layout.html", "partials/*.html", name+".html")
		if err != nil {
			return nil, fmt.Errorf("parse page template %s: %w", name, err)
		}
		pages[name] = t
	}

	partials, err := template.New("partials").Funcs(funcs).ParseFS(templateFS, "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse partial templates: %w", err)
	}

	return &Renderer{
		pages:    pages,
		partials: partials,
	}, nil
}

// RenderPage renders a full HTML document buffered cleanly so template errors never emit partial broken markup.
func (r *Renderer) RenderPage(w io.Writer, page string, data any) error {
	t, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("unknown page template %q", page)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

// RenderPartial renders an HTMX HTML partial cleanly.
func (r *Renderer) RenderPartial(w io.Writer, partialName string, data any) error {
	var buf bytes.Buffer
	if err := r.partials.ExecuteTemplate(&buf, partialName, data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
