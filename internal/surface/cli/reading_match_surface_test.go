package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/reading"
)

// reading_match_surface_test.go is the wiring proof for ruling DQ2b
// (adr-2609300821558671): `reading ingest` matches every finding it stores and
// shows the likely repeats, printed and in --json, and `capture promote rdi-N`
// matches the draft it mints.

// The reading's finding, doubling fmHeld in the instrument's own words.
var fmReadingItem = map[string]any{
	"pattern":            "ledger reader skips records with duplicated keys",
	"tension":            fmDouble,
	"constraint_in_play": "every finding appears in every listing of the capture ledger",
	"why_a_tension":      "a record whose frontmatter carries a duplicated key disappears from every listing silently",
}

func fmCapture(t *testing.T, repo, text string) capture.CaptureResult {
	t.Helper()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: text, Severity: capture.SeverityMinor, Category: "bug",
		Source: "user-observation", FoundDuring: "fixture", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// detectionPayloadWith writes a legal detection payload carrying the items
// given, and returns the path the verb reads.
func detectionPayloadWith(t *testing.T, runID, manifestHash string, def reading.Definition, items ...map[string]any) string {
	t.Helper()
	list := make([]any, 0, len(items))
	for _, it := range items {
		list = append(list, it)
	}
	raw, err := json.Marshal(map[string]any{
		"_type": "abcd.reading.output/1", "run_id": runID,
		"position": "detection", "regime": def.Regime,
		"manifest_sha256": manifestHash,
		"instrument": map[string]any{
			"model": "a-model", "definition_sha256": def.SHA256,
			"assembler_version": reading.AssemblerVersion(),
		},
		"items": list,
	})
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "output.json")
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return outPath
}

// TestReadingIngestShowsTheLikelyRepeats: the ingest reports the match for
// every record in --json and prints it; a later reading returning the same
// finding is linked to the earlier item as well as to the issue.
func TestReadingIngestShowsTheLikelyRepeats(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Chdir(repo)
	held := fmCapture(t, repo, fmHeld)
	fmCapture(t, repo, "The site builder renders a stale anchor for a heading renamed since the last build.")
	fmCapture(t, repo, "The history store drops a transcript that exceeds its byte budget without saying so.")

	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	raw := runCLI(t, "reading", "ingest", "--reading-json", detectionPayloadWith(t, runID, manifestHash, def, fmReadingItem), "--json")
	var res struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
		Matches []struct {
			ID    string `json:"id"`
			Match *struct {
				Matches []struct {
					ID     string `json:"id"`
					Linked bool   `json:"linked"`
				} `json:"matches"`
			} `json:"match"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("decode: %v\n%s", err, raw)
	}
	if len(res.Records) != 1 || len(res.Matches) != 1 || res.Matches[0].ID != res.Records[0].ID ||
		res.Matches[0].Match == nil || len(res.Matches[0].Match.Matches) == 0 ||
		res.Matches[0].Match.Matches[0].ID != held.ID || !res.Matches[0].Match.Matches[0].Linked {
		t.Fatalf("ingest --json does not carry the match on %s:\n%s", held.ID, raw)
	}
	first := res.Records[0].ID

	runID2, manifestHash2, def2 := parkedRunForIngest(t, srcRoot, repo, "detection")
	out := string(runCLI(t, "reading", "ingest", "--reading-json", detectionPayloadWith(t, runID2, manifestHash2, def2, fmReadingItem)))
	if !strings.Contains(out, "matched "+first) || !strings.Contains(out, "link written") {
		t.Fatalf("the ingest does not print the repeat of %s:\n%s", first, out)
	}
}

// TestCapturePromoteReadingItemReportsTheMatch: promoting an accepted reading
// item matches the draft it mints, prints the match and carries it in --json.
func TestCapturePromoteReadingItemReportsTheMatch(t *testing.T) {
	repo, _ := gitRepoNoStore(t)
	t.Chdir(repo)
	held := fmCapture(t, repo, fmHeld)
	fmCapture(t, repo, "The site builder renders a stale anchor for a heading renamed since the last build.")
	body := map[string]string{}
	for k, v := range fmReadingItem {
		if k != "pattern" {
			body[k] = v.(string)
		}
	}
	res, err := capture.IngestReading(capture.IngestReadingRequest{
		RepoRoot: repo, Run: "rdg-2609300000000001", Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: "detection", Regime: issueschema.ReadingRegime("detection"),
		Items: []capture.ReadingItem{
			{Pattern: fmReadingItem["pattern"].(string), Body: body},
			{Pattern: "the capture ledger reader skips records carrying duplicated keys", Body: body},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Records {
		if _, err := capture.Disposition(capture.DispositionRequest{
			RepoRoot: repo, Item: r.ID, State: issueschema.DispositionAccepted,
			Grounds: "the tension is real and worth acting on",
		}); err != nil {
			t.Fatal(err)
		}
	}
	item := res.Records[0].ID

	raw := runCLI(t, "capture", "promote", item, "--json")
	var p struct {
		Match *struct {
			Matches []struct {
				ID string `json:"id"`
			} `json:"matches"`
		} `json:"match"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("decode: %v\n%s", err, raw)
	}
	if p.Match == nil || len(p.Match.Matches) == 0 || p.Match.Matches[0].ID != held.ID {
		t.Fatalf("promote --json does not carry the match on %s:\n%s", held.ID, raw)
	}
	out := string(runCLI(t, "capture", "promote", res.Records[1].ID))
	if !strings.Contains(out, "matched "+held.ID) {
		t.Fatalf("promote does not print the match on %s:\n%s", held.ID, out)
	}
}
