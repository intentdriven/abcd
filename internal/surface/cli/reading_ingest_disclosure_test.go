package cli

// reading_ingest_disclosure_test.go: what the ingest front door says on the
// refusal path about the committed tier (iss-2608311517509690).
//
// The core reports what it found, cleared and rolled back; the surface used to
// render those fields only when a refusal RECORD existed, so a refusal before
// the run's identity was proven — the type check — printed the error alone. The
// text and the JSON now carry the disclosure on every refusal path.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/reading"
)

// plantOrphanedStage writes an orphaned stage and one reading record its run
// had landed into repo, returning the record's absolute path and its bytes.
func plantOrphanedStage(t *testing.T, repo, runID, itemID string) (string, []byte) {
	t.Helper()
	write := func(rel string, body []byte) string {
		abs := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			t.Fatal(err)
		}
		return abs
	}
	write(reading.IngestStageDir+"/"+runID+"/stage.json",
		[]byte(`{"_type":"`+reading.StageType+`","run_id":"`+runID+`","records":["`+itemID+`"]}`))
	body := []byte("---\nid: " + itemID + "\nrun: " + runID + "\n---\n\nthe committed body\n")
	return write(".abcd/work/issues/readings/"+runID+"/"+itemID+".md", body), body
}

// parkDetectionRun assembles one run through the sibling verb and returns a
// LEGAL payload for it, with the shipped detection definition copied into repo
// so the ingest can resolve the instrument. The working directory must already
// be repo.
func parkDetectionRun(t *testing.T, srcRoot, repo string) map[string]any {
	t.Helper()
	out := runCLI(t, "reading", "assemble", "--position", "detection", "--target", "HEAD", "--json")
	var assembled struct {
		RunID        string `json:"run_id"`
		ManifestHash string `json:"manifest_hash"`
	}
	if err := json.Unmarshal(out, &assembled); err != nil {
		t.Fatalf("decode the assemble render: %v\n%s", err, out)
	}
	def, err := reading.LoadDefinition(srcRoot, "detection")
	if err != nil {
		t.Fatal(err)
	}
	defRaw, err := os.ReadFile(filepath.Join(srcRoot, filepath.FromSlash(def.Path)))
	if err != nil {
		t.Fatal(err)
	}
	defPath := filepath.Join(repo, filepath.FromSlash(def.Path))
	if err := os.MkdirAll(filepath.Dir(defPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defPath, defRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"_type": "abcd.reading.output/1", "run_id": assembled.RunID,
		"position": "detection", "regime": def.Regime,
		"manifest_sha256": assembled.ManifestHash,
		"instrument": map[string]any{
			"model": "a-model", "definition_sha256": def.SHA256,
			"assembler_version": reading.AssemblerVersion(),
		},
		"items": []any{map[string]any{
			"pattern": "the pattern this reading read under",
			"tension": "the record says one thing", "constraint_in_play": "a stated constraint",
			"why_a_tension": "the two cannot both hold",
		}},
	}
}

// writePayloadFile renders a payload to a file the verb can be pointed at.
func writePayloadFile(t *testing.T, doc any) string {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "output.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestIngestDisclosesTheOrphanItLeftOnRefusal: with an orphaned stage present
// and its record in the ledger, a refused ingest leaves the record byte for
// byte, reports the named error, and — in the text and in the JSON alike —
// says the orphan was seen and left in place. Both kinds of refusal are
// driven: one before the run's identity is proven (no refusal record) and one
// after (a refusal record), because the surface used to key the render on the
// record alone.
func TestIngestDisclosesTheOrphanItLeftOnRefusal(t *testing.T) {
	const orphan, item = "rdg-2608310000000038", "rdi-2608310000000039"
	srcRoot := repoRootFromTest(t)

	cases := []struct {
		name    string
		payload func(t *testing.T, repo string) map[string]any
		wants   string
	}{
		{"at the type check", func(t *testing.T, _ string) map[string]any {
			return map[string]any{"_type": "not-the-output-type", "run_id": "rdg-2608310000000040",
				"position": "detection", "regime": "registrative", "manifest_sha256": strings.Repeat("0", 64),
				"instrument": map[string]any{"model": "m", "definition_sha256": "d", "assembler_version": "v"},
				"items":      []any{map[string]any{"pattern": "p"}}}
		}, "_type"},
		{"at the regime check", func(t *testing.T, repo string) map[string]any {
			doc := parkDetectionRun(t, srcRoot, repo)
			doc["regime"] = "generative"
			return doc
		}, "generative"},
	}
	for _, tc := range cases {
		for _, asJSON := range []bool{true, false} {
			mode := "text"
			if asJSON {
				mode = "json"
			}
			t.Run(tc.name+" in "+mode, func(t *testing.T) {
				repo := readingRepo(t)
				t.Chdir(repo)
				recordPath, body := plantOrphanedStage(t, repo, orphan, item)
				outPath := writePayloadFile(t, tc.payload(t, repo))

				args := []string{"reading", "ingest", "--reading-json", outPath}
				if asJSON {
					args = append(args, "--json")
				}
				rendered, err := runCLIErr(t, args...)
				if err == nil {
					t.Fatalf("the illegal payload exited 0:\n%s", rendered)
				}
				if code := exitCodeOf(err); code != 2 {
					t.Errorf("a refusal exited %d, want 2", code)
				}
				if !strings.Contains(err.Error(), tc.wants) {
					t.Errorf("the refusal does not name the rule: %v", err)
				}

				// (a) the committed record is untouched, byte for byte.
				got, readErr := os.ReadFile(recordPath)
				if readErr != nil {
					t.Fatalf("a refused ingest deleted the committed record: %v", readErr)
				}
				if !bytes.Equal(got, body) {
					t.Errorf("the committed record changed under a refused ingest:\n%s", got)
				}

				// (c) the disclosure is on the surface.
				if asJSON {
					var res reading.IngestResult
					if jsonErr := json.Unmarshal(rendered, &res); jsonErr != nil {
						t.Fatalf("the refusal rendered no JSON result: %v\n%s", jsonErr, rendered)
					}
					if len(res.PendingStages) != 1 || res.PendingStages[0] != orphan {
						t.Errorf("the JSON reports pending stages %v, want exactly [%s]:\n%s", res.PendingStages, orphan, rendered)
					}
					if len(res.ClearedStages) != 0 || len(res.RolledBack) != 0 {
						t.Errorf("the JSON reports a mutation on a refused run:\n%s", rendered)
					}
					return
				}
				text := string(rendered)
				if !strings.Contains(text, orphan) || !strings.Contains(text, "left in place") {
					t.Errorf("the text render does not say the orphan was left in place:\n%s", text)
				}
				if strings.Contains(text, "cleared:") || strings.Contains(text, "rolled back:") {
					t.Errorf("the text render reports a mutation on a refused run:\n%s", text)
				}
			})
		}
	}
}

// TestTheTextRenderDisclosesWithoutAProvenRun: a refusal before the run's
// identity is proven has no run id, position or regime to head the render
// with, and the header must not print an empty one as if it were a run.
func TestTheTextRenderDisclosesWithoutAProvenRun(t *testing.T) {
	var buf bytes.Buffer
	renderIngestResult(&buf, reading.IngestResult{
		PendingStages: []string{"rdg-2608310000000041"},
	})
	out := buf.String()
	if strings.Contains(out, "0 record(s)") || strings.Contains(out, "at the  position") {
		t.Errorf("the render heads a refusal with an empty run line:\n%s", out)
	}
	if !strings.Contains(out, "rdg-2608310000000041") || !strings.Contains(out, "left in place") {
		t.Errorf("the render does not disclose the pending stage:\n%s", out)
	}
}
