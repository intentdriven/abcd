package intent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The review block's one reader and writer: where a block starts (a LIVE
// marker), where it stops (its closing marker, or the known shape of a block
// written before the closing marker existed), and how the record it is written
// into ends.

// shippedWithAuditNotes is a shipped intent whose Audit Notes section body is
// notes, written verbatim: the caller decides how the file ends.
func shippedWithAuditNotes(notes string) string {
	return "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n" +
		"# alpha\n\n## Scope Conditions\n\n" + NullityToken +
		"\n\n## Acceptance Criteria\n\n- ok\n" + groundsSection + "\n## Audit Notes\n\n" + notes
}

// fixtureWithNotes writes a shipped itd-10 (and its closed spc-1) whose Audit
// Notes are notes.
func fixtureWithNotes(t *testing.T, notes string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", shippedWithAuditNotes(notes))
	writeFile(t, root, specsClosed+"/spc-1-alpha.md", specNaming("spc-1", "alpha", "itd-10"))
	return root
}

const blockRcp = "rcp-00000000abcd"

func owedStub(rcp string) string {
	return "<!-- abcd-review: OWED receipt=" + rcp + " -->\nFidelity review OWED (receipt " + rcp + ")."
}

// TestIngestedRecordEndsInANewline: every write of a review block leaves the
// record ending in a newline, as every other record writer does
// (iss-2609231011136579) — including a record that reached the ingest without
// one, the shape the earlier writer left on the tree.
func TestIngestedRecordEndsInANewline(t *testing.T) {
	t.Run("ship then ingest", func(t *testing.T) {
		root := t.TempDir()
		rcp := shipOne(t, root)
		if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(rcp))); err != nil {
			t.Fatal(err)
		}
		if s := shippedIntentBody(t, root); !strings.HasSuffix(s, "\n") {
			t.Fatalf("the ingested record does not end in a newline:\n%q", s[max(0, len(s)-80):])
		}
	})
	t.Run("stub at end of file with no newline", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp))
		if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(blockRcp))); err != nil {
			t.Fatal(err)
		}
		if s := shippedIntentBody(t, root); !strings.HasSuffix(s, "\n") || strings.HasSuffix(s, "\n\n") {
			t.Fatalf("want exactly one final newline:\n%q", s[max(0, len(s)-80):])
		}
	})
	t.Run("dead letter", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp))
		bad := strings.Replace(validVerdict(blockRcp), `"criterion_id": "ac-1"`, `"criterion_id": "ac-9"`, 1)
		res, err := IngestVerdict(root, writeVerdict(t, root, bad))
		if err != nil || res.Status != "dead_letter" {
			t.Fatalf("ingest = %+v %v, want dead_letter", res, err)
		}
		if s := shippedIntentBody(t, root); !strings.HasSuffix(s, "\n") {
			t.Fatalf("the dead-lettered record does not end in a newline:\n%q", s[max(0, len(s)-80):])
		}
	})
}

// TestFirstIngestKeepsLinkRefsBelowTheOwedStub: a stub parked above a trailing
// run of link-reference definitions (the shape appendToAuditNotes writes, per
// iss-2608210737265820) is replaced without taking the definitions with it
// (iss-2609251451432601).
func TestFirstIngestKeepsLinkRefsBelowTheOwedStub(t *testing.T) {
	refs := "[iss-80]: https://example.com/issues/iss-80\n[spc-28]: https://example.com/specs/spc-28\n"
	root := fixtureWithNotes(t, owedStub(blockRcp)+"\n\n"+refs)
	if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(blockRcp))); err != nil {
		t.Fatal(err)
	}
	s := shippedIntentBody(t, root)
	if !strings.HasSuffix(s, "\n\n"+refs) {
		t.Fatalf("the trailing link references did not survive the first ingest, one blank line below the block:\n%s", s)
	}
	if !strings.Contains(s, "abcd-review: INGESTED receipt="+blockRcp) {
		t.Fatalf("not ingested:\n%s", s)
	}
}

// TestReplacementKeepsLinkRefsBelowALegacyBlock: a block written before the
// closing marker existed runs to the next marker, heading or end of file, and a
// replacement of it keeps a trailing run of link-reference definitions, which
// the renderer never writes (iss-2609251451432601, "and any replacement").
func TestReplacementKeepsLinkRefsBelowALegacyBlock(t *testing.T) {
	refs := "[iss-80]: https://example.com/issues/iss-80\n"
	legacy := "<!-- abcd-review: INGESTED receipt=" + blockRcp + " -->\nFidelity review — receipt " + blockRcp + " (verifier old v0).\n\nGap audit:\n- honoured: (none)"
	content := shippedWithAuditNotes(legacy + "\n\n" + refs)
	out := upsertReviewBlock(content, blockRcp, "<!-- abcd-review: INGESTED receipt="+blockRcp+" -->\nNEW")
	if !strings.HasSuffix(out, "NEW\n\n"+refs) {
		t.Fatalf("the link reference did not survive the replacement:\n%s", out)
	}
	if strings.Contains(out, "verifier old v0") {
		t.Fatalf("the legacy block was not replaced:\n%s", out)
	}
}

// TestReingestKeepsAHumanNoteBelowTheBlock: prose a human writes under an
// ingested block is not part of the block, so an identical re-ingest is the
// documented noop and a replacing one keeps the note (iss-2609251451434656).
func TestReingestKeepsAHumanNoteBelowTheBlock(t *testing.T) {
	root := t.TempDir()
	rcp := shipOne(t, root)
	vp := writeVerdict(t, root, validVerdict(rcp))
	if _, err := IngestVerdict(root, vp); err != nil {
		t.Fatal(err)
	}
	const note = "A note the product thinker wrote under the review."
	path := filepath.Join(root, shippedDir, "itd-10-alpha.md")
	withNote := shippedIntentBody(t, root) + "\n" + note + "\n"
	if err := os.WriteFile(path, []byte(withNote), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := IngestVerdict(root, vp)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "noop" || res.Replaced {
		t.Fatalf("identical re-ingest = %+v, want noop", res)
	}
	if got := shippedIntentBody(t, root); got != withNote {
		t.Fatalf("an identical re-ingest changed the record:\n%s", got)
	}

	changed := strings.Replace(validVerdict(rcp), "the ship-move writes the OWED stub and request file",
		"the ship-move writes the OWED stub, and the request file beside it", 1)
	res, err = IngestVerdict(root, writeVerdict(t, root, changed))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "ingested" || !res.Replaced {
		t.Fatalf("changed re-ingest = %+v, want ingested and replaced", res)
	}
	s := shippedIntentBody(t, root)
	if !strings.Contains(s, "and the request file beside it") {
		t.Fatalf("the replacement did not land:\n%s", s)
	}
	if !strings.HasSuffix(s, "\n\n"+note+"\n") {
		t.Fatalf("the human's note did not survive the replacement, one blank line below the block:\n%s", s)
	}
}

// TestOnlyALiveMarkerCounts: the review marker is the ledger's own state, so
// only a marker a markdown reader would parse as one counts — a line of the
// Audit Notes, outside any fenced block or HTML comment span. A marker-shaped
// line in a fence, in a comment, or in another section is an example, not
// state (iss-2609020529185438).
func TestOnlyALiveMarkerCounts(t *testing.T) {
	marker := "<!-- abcd-review: OWED receipt=" + blockRcp + " -->"
	for name, body := range map[string]string{
		"backtick fence": "# a\n\n## Audit Notes\n\n```markdown\n" + marker + "\n```\n",
		"tilde fence":    "# a\n\n## Audit Notes\n\n~~~\n" + marker + "\n~~~\n",
		"comment span":   "# a\n\n## Audit Notes\n\n<!--\nparked example:\n" + marker + "\n",
		"other section":  "# a\n\n## Notes\n\n" + marker + "\n\n## Audit Notes\n\nnothing yet\n",
		"no section":     "# a\n\n" + marker + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if state, ok := markerState(body, blockRcp); ok {
				t.Errorf("markerState counted a %s marker that is not live", state)
			}
			if rcp, state, ok := existingMarker(body); ok {
				t.Errorf("existingMarker found %s %s, which is not live", state, rcp)
			}
		})
	}
	live := "# a\n\n## Audit Notes\n\n```\n<!-- abcd-review: INGESTED receipt=rcp-ffffffffffff -->\n```\n\n" + marker + "\nFidelity review OWED.\n"
	if state, ok := markerState(live, blockRcp); !ok || state != "OWED" {
		t.Fatalf("the live marker below a fenced example must count: state=%q ok=%v", state, ok)
	}
	if rcp, _, ok := existingMarker(live); !ok || rcp != blockRcp {
		t.Fatalf("existingMarker = %q, want the live %s, not the fenced example", rcp, blockRcp)
	}
}

// TestAFencedMarkerIsNotASolicitation is the end-to-end form: a record quoting
// a marker in a fenced example does not solicit a verdict for that receipt.
func TestAFencedMarkerIsNotASolicitation(t *testing.T) {
	root := fixtureWithNotes(t, "```\n<!-- abcd-review: OWED receipt="+blockRcp+" -->\n```\n")
	before := shippedIntentBody(t, root)
	if _, err := IngestVerdict(root, writeVerdictRaw(t, root, validVerdict(blockRcp))); err == nil ||
		!strings.Contains(err.Error(), "unsolicited") {
		t.Fatalf("err = %v, want the receipt refused as unsolicited", err)
	}
	if shippedIntentBody(t, root) != before {
		t.Fatal("a refused ingest changed the record")
	}
}

// armProseCitations writes a record-lint configuration arming
// prose_citation_resolves over the intent store, as this repository's does.
func armProseCitations(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, ".abcd/record-lint.json", `{
  "roots": [".abcd/development"],
  "rules": {
    "prose_citation_resolves": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"itd": ".abcd/development/intents", "spc": ".abcd/development/specs"}
    }
  }
}
`)
}

// TestIngestRefusesAnUnresolvableCitation: the ingest writes a committed record
// that record-lint's prose_citation_resolves reads, so a verdict whose prose
// names a record id that resolves to nothing is refused before anything is
// written, naming the id — a valid verdict must never produce an uncommittable
// record (iss-2609231036448320).
func TestIngestRefusesAnUnresolvableCitation(t *testing.T) {
	const dangling = "spc-2609999999999999"
	cite := func(rcp, id string) string {
		return strings.Replace(validVerdict(rcp), "the ship-move writes the OWED stub and request file",
			"the fixture dangles its target to "+id+" and the check refuses it", 1)
	}

	t.Run("armed: refused, nothing written", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp)+"\n")
		armProseCitations(t, root)
		before := shippedIntentBody(t, root)
		_, err := IngestVerdict(root, writeVerdict(t, root, cite(blockRcp, dangling)))
		if err == nil || !strings.Contains(err.Error(), dangling) || !strings.Contains(err.Error(), "prose_citation_resolves") {
			t.Fatalf("err = %v, want a refusal naming %s and the rule", err, dangling)
		}
		if shippedIntentBody(t, root) != before {
			t.Fatal("a refused ingest changed the record")
		}
	})
	t.Run("armed: a resolving citation ingests", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp)+"\n")
		armProseCitations(t, root)
		res, err := IngestVerdict(root, writeVerdict(t, root, cite(blockRcp, "spc-1")))
		if err != nil || res.Status != "ingested" {
			t.Fatalf("ingest = %+v %v, want ingested", res, err)
		}
	})
	t.Run("unarmed: the repository does not gate prose citations", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp)+"\n")
		res, err := IngestVerdict(root, writeVerdict(t, root, cite(blockRcp, dangling)))
		if err != nil || res.Status != "ingested" {
			t.Fatalf("ingest = %+v %v, want ingested", res, err)
		}
	})
	t.Run("armed: a re-ingest is refused the same way", func(t *testing.T) {
		root := fixtureWithNotes(t, owedStub(blockRcp)+"\n")
		armProseCitations(t, root)
		if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(blockRcp))); err != nil {
			t.Fatal(err)
		}
		before := shippedIntentBody(t, root)
		_, err := IngestVerdict(root, writeVerdict(t, root, cite(blockRcp, dangling)))
		if err == nil || !strings.Contains(err.Error(), dangling) {
			t.Fatalf("err = %v, want a refusal naming %s", err, dangling)
		}
		if shippedIntentBody(t, root) != before {
			t.Fatal("a refused re-ingest changed the record")
		}
	})
}

// treeRoot is the repository this package is tested in.
func treeRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("the repository root is not where the test expects it: %v", err)
	}
	return root
}

// rawMarkerRe is the byte pattern the review marker was read with before the
// live-marker reader: the migration guard below holds the reader to it on every
// record the tree carries.
var rawMarkerRe = regexp.MustCompile(`(?m)^<!-- abcd-review: (OWED|INGESTED|DEAD_LETTER) receipt=(rcp-[0-9a-f]+) -->\r?$`)

// TestEveryTreeReviewBlockRoundTrips runs every intent record in this
// repository through the review block's reader and writer: the reader finds the
// marker the byte pattern found, and writing each block back over itself leaves
// the record byte-identical, so the bounded extent reads every block already on
// the tree exactly as the unbounded one did.
func TestEveryTreeReviewBlockRoundTrips(t *testing.T) {
	root := treeRoot(t)
	files, err := filepath.Glob(filepath.Join(root, ".abcd", "development", "intents", "*", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	blocks := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		rel, _ := filepath.Rel(root, f)
		raw := rawMarkerRe.FindAllStringSubmatch(content, -1)
		if rcp, state, ok := existingMarker(content); ok != (len(raw) > 0) || (ok && (rcp != raw[0][2] || state != raw[0][1])) {
			t.Errorf("%s: existingMarker = %s %s %v, the byte pattern read %v", rel, rcp, state, ok, raw)
		}
		for _, m := range raw {
			blocks++
			if state, ok := markerState(content, m[2]); !ok || state != m[1] {
				t.Errorf("%s: markerState(%s) = %s %v, want %s", rel, m[2], state, ok, m[1])
			}
			text, ok := reviewBlockText(content, m[2])
			if !ok {
				t.Errorf("%s: no review block read for %s", rel, m[2])
				continue
			}
			if got := upsertReviewBlock(content, m[2], text); got != content {
				t.Errorf("%s: writing the %s block back over itself changed the record", rel, m[2])
			}
		}
	}
	if blocks == 0 {
		t.Fatal("the tree carries no review block; the guard read nothing")
	}
}

// TestAppendLandsInTheLiveAuditNotes: the writer appends into the section the
// reader reads, so a fenced `## Audit Notes` example above the real section
// never receives the block (iss-2609020529185438).
func TestAppendLandsInTheLiveAuditNotes(t *testing.T) {
	content := "# a\n\n## Example\n\n```markdown\n## Audit Notes\n\nquoted\n```\n\n## Audit Notes\n\nlive\n"
	out := upsertReviewBlock(content, blockRcp, owedStub(blockRcp))
	want := "# a\n\n## Example\n\n```markdown\n## Audit Notes\n\nquoted\n```\n\n## Audit Notes\n\nlive\n\n" + owedStub(blockRcp) + "\n"
	if out != want {
		t.Fatalf("the block did not land in the live section:\n got %q\nwant %q", out, want)
	}
	if state, ok := markerState(out, blockRcp); !ok || state != "OWED" {
		t.Fatalf("the reader cannot find the block the writer wrote: %q %v", state, ok)
	}
}

// TestIngestRefusesWithNoGateRegistered: an ingest the prose-citation check
// cannot be made for writes nothing, rather than a record the gate never saw.
func TestIngestRefusesWithNoGateRegistered(t *testing.T) {
	saved := proseCitationGate
	SetProseCitationGate(nil)
	t.Cleanup(func() { SetProseCitationGate(saved) })

	root := fixtureWithNotes(t, owedStub(blockRcp)+"\n")
	before := shippedIntentBody(t, root)
	if _, err := IngestVerdict(root, writeVerdict(t, root, validVerdict(blockRcp))); err == nil ||
		!strings.Contains(err.Error(), "no prose-citation gate is registered") {
		t.Fatalf("err = %v, want the unregistered gate refused", err)
	}
	if shippedIntentBody(t, root) != before {
		t.Fatal("a refused ingest changed the record")
	}
}
