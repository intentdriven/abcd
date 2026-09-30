package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/gittest"
)

// armLedgerProseCitations writes a record-lint configuration arming
// prose_citation_resolves over the ledger's stores, as this repository's does.
func armLedgerProseCitations(t *testing.T, repo string) {
	t.Helper()
	writeTree(t, repo, ".abcd/record-lint.json", `{
  "roots": [".abcd/work"],
  "rules": {
    "prose_citation_resolves": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"iss": ".abcd/work/issues", "rdi": ".abcd/work/issues/readings"}
    }
  }
}
`)
}

// withNoProseGate unregisters the gate for one test.
func withNoProseGate(t *testing.T) {
	t.Helper()
	intent.SetProseCitationGate(nil)
	t.Cleanup(func() { intent.SetProseCitationGate(lintProseGate) })
}

// ledgerFiles lists every file under the ledger, repo-relative, so a test can
// prove a refusal left it exactly as it was.
func ledgerFiles(t *testing.T, repo string) []string {
	t.Helper()
	var out []string
	root := filepath.Join(repo, LedgerRelPath)
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(repo, p)
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A reading's items are the host's words, copied into records record-lint's
// prose_citation_resolves reads; an item citing a record id that names no
// record is refused before anything is written, naming the item, the field and
// the id (iss-2609261835118276). A resolving citation lands, and a repository
// that does not arm the rule gates nothing.
func TestIngestReadingRefusesAnUnresolvedCitation(t *testing.T) {
	const dangling = "iss-2609999999999999"
	req := func(repo, ir, pattern string) IngestReadingRequest {
		return IngestReadingRequest{
			RepoRoot: repo, IssuesRoot: ir, Run: "rdg-2608300000000001", Manifest: "sha256:beef",
			Position: "detection", Regime: "registrative",
			Items: []ReadingItem{
				{Pattern: "a clean constraint", Body: bodyFor("detection")},
				{Pattern: pattern, Body: bodyFor("detection")},
			},
		}
	}

	t.Run("armed: refused, nothing written", func(t *testing.T) {
		repo, ir := ledger(t)
		armLedgerProseCitations(t, repo)
		_, err := IngestReading(req(repo, ir, "the constraint "+dangling+" states"))
		if !errors.Is(err, ErrUnresolvedCitation) || !strings.Contains(err.Error(), dangling) ||
			!strings.Contains(err.Error(), "reading item 2's pattern") || !strings.Contains(err.Error(), "prose_citation_resolves") {
			t.Fatalf("err = %v, want a refusal naming item 2's pattern, %s and the rule", err, dangling)
		}
		if got := ledgerFiles(t, repo); len(got) != 0 {
			t.Fatalf("a refused ingest wrote %v", got)
		}
	})
	t.Run("armed: a body field is judged too", func(t *testing.T) {
		repo, ir := ledger(t)
		armLedgerProseCitations(t, repo)
		r := req(repo, ir, "a clean pattern")
		body := bodyFor("detection")
		var field string
		for field = range body {
			break
		}
		body[field] = "see " + dangling
		r.Items[1].Body = body
		_, err := IngestReading(r)
		if !errors.Is(err, ErrUnresolvedCitation) || !strings.Contains(err.Error(), "reading item 2's "+field) {
			t.Fatalf("err = %v, want a refusal naming item 2's %s", err, field)
		}
	})
	t.Run("armed: a resolving citation ingests", func(t *testing.T) {
		repo, ir := ledger(t)
		armLedgerProseCitations(t, repo)
		held, err := testCapture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
			Category: "bug", Source: "user-observation", FoundDuring: "t"})
		if err != nil {
			t.Fatal(err)
		}
		res, err := IngestReading(req(repo, ir, "the constraint "+held.ID+" states"))
		if err != nil || len(res.Records) != 2 {
			t.Fatalf("ingest = %+v %v, want both items written", res, err)
		}
	})
	t.Run("unarmed: the repository does not gate prose citations", func(t *testing.T) {
		repo, ir := ledger(t)
		res, err := IngestReading(req(repo, ir, "the constraint "+dangling+" states"))
		if err != nil || len(res.Records) != 2 {
			t.Fatalf("ingest = %+v %v, want both items written", res, err)
		}
	})
	t.Run("no gate registered: refused, nothing written", func(t *testing.T) {
		withNoProseGate(t)
		repo, ir := ledger(t)
		_, err := IngestReading(req(repo, ir, "a clean pattern"))
		if err == nil || !strings.Contains(err.Error(), "no prose-citation gate is registered") {
			t.Fatalf("err = %v, want the unregistered gate refused", err)
		}
		if _, err := os.Stat(filepath.Join(ir, issueschema.ReadingsDir)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a refused ingest provisioned the readings store (%v)", err)
		}
	})
}

// consistencyArmedRepo is consistencyLedgerRepo with the ledger's prose
// citations armed, committed with the fixture so the emit reads a clean tree.
func consistencyArmedRepo(t *testing.T) string {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(cxA, "---\nid: itd-10\nslug: one-spec\nkind: standalone\nspec_id: spc-1\n---\n\n# One spec\n\n## Press Release\n\n"+cxQuoteA+"\n")
	r.Write(cxB, "---\nid: itd-11\nslug: many-specs\nkind: standalone\nspec_id: spc-2\n---\n\n# Many specs\n\n## Decisions\n\n1. "+cxQuoteB+"\n")
	armLedgerProseCitations(t, r.Root())
	r.Commit("fixture")
	return r.Root()
}

// The consistency pass files each finding as an issue carrying the host's
// summary, quotes and explanation; a finding citing a record id that names no
// record is refused before the first finding is filed and before the report is
// written, naming the finding and the id (iss-2609261835118276).
func TestConsistencyIngestRefusesAnUnresolvedCitation(t *testing.T) {
	const dangling = "adr-2609999999999999"
	cite := func(payload []byte) []byte {
		return []byte(strings.Replace(string(payload), "the other says one or more.",
			"the other says one or more, as "+dangling+" ruled.", 1))
	}

	t.Run("armed: refused, nothing written", func(t *testing.T) {
		root := consistencyArmedRepo(t)
		_, err := IngestConsistency(root, cite(consistencyPayload(t, root)), "2026-09-26", nil)
		if !errors.Is(err, ErrUnresolvedCitation) || !strings.Contains(err.Error(), dangling) ||
			!strings.Contains(err.Error(), "consistency finding 1") {
			t.Fatalf("err = %v, want a refusal naming finding 1 and %s", err, dangling)
		}
		for _, f := range ledgerFiles(t, root) {
			if filepath.Base(f) != lockFilename {
				t.Fatalf("a refused ingest wrote %s", f)
			}
		}
		if _, err := os.Stat(filepath.Join(root, intent.ReviewsShelfRelDir)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a refused ingest wrote a report (%v)", err)
		}
	})
	t.Run("armed: a clean finding is filed", func(t *testing.T) {
		root := consistencyArmedRepo(t)
		res, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
		if err != nil || len(res.Filed) != 1 {
			t.Fatalf("ingest = %+v %v, want the finding filed", res, err)
		}
	})
	t.Run("no gate registered: refused", func(t *testing.T) {
		withNoProseGate(t)
		root := consistencyLedgerRepo(t)
		_, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26", nil)
		if err == nil || !strings.Contains(err.Error(), "no prose-citation gate is registered") {
			t.Fatalf("err = %v, want the unregistered gate refused", err)
		}
	})
}
