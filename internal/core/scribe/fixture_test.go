package scribe

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// positionDetection is the registrative position, whose items meet no ordering
// gate: the fixture a test uses when it is not about the gate.
const positionDetection = "detection"

// fixtureRun is the run every fixture ingests; otherRun is a second run whose
// records sit beside it in the store.
const (
	fixtureRun = "rdg-2609250000000001"
	otherRun   = "rdg-2609250000000002"
)

// Sentinels planted OUTSIDE the ledger. None may reach a scribe context: each
// names the class of material it stands for, so a leak names itself.
var outsideSentinels = map[string]string{
	"internal/core/thing.go":                          "SENTINEL-SHIPPED-SOURCE",
	"docs/guide.md":                                   "SENTINEL-SHIPPED-DOCS",
	"commands/scribe.md":                              "SENTINEL-COMMAND-SURFACE",
	".abcd/development/brief/01-x.md":                 "SENTINEL-BRIEF",
	".abcd/development/intents/planned/itd-1-x.md":    "SENTINEL-INTENT",
	".abcd/development/specs/open/spc-1-x.md":         "SENTINEL-SPEC",
	".abcd/development/decisions/adrs/0001-x.md":      "SENTINEL-DECISION",
	".abcd/work/DECISIONS.md":                         "SENTINEL-WORKING-DECISIONS",
	".abcd/.work.local/NEXT.md":                       "SENTINEL-LOCAL-TIER",
	".abcd/.work.local/transcripts/aaaa/records/x.md": "SENTINEL-LOCAL-TRANSCRIPT",
}

// fixture is one repository holding an ingested run of n items at position, a
// second run, an open issue, the scribe definition, and every sentinel above.
type fixture struct {
	repo  string
	home  string
	items []string
	other []string
}

func newFixture(t *testing.T, position string, n int) fixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	f := fixture{repo: repo, home: home}
	f.items = ingestRun(t, repo, fixtureRun, position, n)
	f.other = ingestRun(t, repo, otherRun, positionDetection, 1)
	for _, run := range []string{fixtureRun, otherRun} {
		writeFile(t, repo, filepath.Join(issueschema.ReadingsRecordDir, run, issueschema.RunRecordFileName),
			`{"run_id":"`+run+`","position":"`+position+`"}`)
	}
	writeFile(t, repo, ".abcd/work/issues/open/iss-1-an-open-issue.md", "---\nid: iss-1\n---\nAn open issue.\n")
	writeFile(t, repo, DefinitionPath, "---\nname: scribe\nprompt_version: 0.2.0\n---\n\nThe scribe.\n")
	for rel, sentinel := range outsideSentinels {
		writeFile(t, repo, rel, sentinel+"\n")
	}
	writeFile(t, home, ".abcd/transcripts/aaaa/records/y.md", "SENTINEL-TRANSCRIPT-STORE\n")
	return f
}

// ingestRun writes n reading records for run through the ledger's own writer.
func ingestRun(t *testing.T, repo, run, position string, n int) []string {
	t.Helper()
	items := make([]capture.ReadingItem, 0, n)
	for i := 0; i < n; i++ {
		body := map[string]string{}
		for _, field := range issueschema.ReadingBodyFields[position] {
			body[field] = "text for " + field
		}
		items = append(items, capture.ReadingItem{Pattern: "a stated constraint", Body: body})
	}
	res, err := capture.IngestReading(capture.IngestReadingRequest{
		RepoRoot: repo, Run: run, Manifest: "sha256:" + strings.Repeat("a", 64),
		Position: position, Regime: issueschema.ReadingRegime(position), Items: items,
	})
	if err != nil {
		t.Fatalf("IngestReading %s: %v", run, err)
	}
	ids := make([]string, 0, len(res.Records))
	for _, r := range res.Records {
		ids = append(ids, r.ID)
	}
	return ids
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// supply writes the researcher's dispositions text outside the repository and
// returns its path.
func supply(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "dispositions.md")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// treeDigest hashes every file under dir, keyed by relative path, so a test can
// prove a tier was left byte-identical.
func treeDigest(t *testing.T, dir string) string {
	t.Helper()
	var lines []string
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		sum := sha256.Sum256(raw)
		rel, _ := filepath.Rel(dir, p)
		lines = append(lines, filepath.ToSlash(rel)+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
