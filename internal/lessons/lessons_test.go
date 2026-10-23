package lessons

import (
	"testing"

	"github.com/kapombe-k/Biostat/internal/bio"
	"github.com/kapombe-k/Biostat/internal/seed"
)

func TestLessonsIntegrityAndExactCoordinateAnchors(t *testing.T) {
	all := List()
	if len(all) < 5 {
		t.Fatalf("expected >=5 curriculum lessons, got %d", len(all))
	}

	for _, l := range all {
		if len(l.Steps) < 3 {
			t.Errorf("lesson %s should have >=3 steps", l.Slug)
		}
		if len(l.Quiz) < 2 {
			t.Errorf("lesson %s should have >=2 quiz questions", l.Slug)
		}

		// Grade every quiz question positively and negatively
		for _, q := range l.Quiz {
			evRight, err := GradeAnswer(l.Slug, q.ID, q.CorrectOption)
			if err != nil || !evRight.IsCorrect {
				t.Errorf("expected correct evaluation for %s/%s: err=%v", l.Slug, q.ID, err)
			}
			for _, o := range q.Options {
				if o.ID != q.CorrectOption {
					evWrong, err := GradeAnswer(l.Slug, q.ID, o.ID)
					if err != nil || evWrong.IsCorrect {
						t.Errorf("expected incorrect evaluation for %s/%s/%s", l.Slug, q.ID, o.ID)
					}
				}
			}
		}

		// Verify every single 3D FocusResidue in every step physically exists in the parsed atomic coordinates!
		for _, step := range l.Steps {
			raw, _, meta, err := seed.ReadStructureCoordinates(step.Scene.StructureID)
			if err != nil {
				t.Fatalf("lesson %s step %d missing seed structure %s: %v", l.Slug, step.Index, step.Scene.StructureID, err)
			}
			det, err := bio.ParseCoordinates(step.Scene.StructureID, meta.Kind, raw)
			if err != nil {
				t.Fatalf("parse %s: %v", step.Scene.StructureID, err)
			}
			for _, fr := range step.Scene.FocusResidues {
				found := false
				for _, ch := range det.Chains {
					if ch.AuthAsymID != fr.ChainID {
						continue
					}
					for _, r := range ch.Residues {
						if r.Observed && r.AuthSeqID == fr.AuthSeqID && r.InsCode == fr.InsCode {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("lesson %s step %d focus residue %s:%d%s NOT found in observed coordinates of %s",
						l.Slug, step.Index, fr.ChainID, fr.AuthSeqID, fr.InsCode, step.Scene.StructureID)
				}
			}
		}
	}
}
