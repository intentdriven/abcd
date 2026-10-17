package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The closed disposition vocabulary (the deferred-finding discipline): from
// release-gate manifest schemaVersion 2, a release receipt's finding is exactly
// fixed, invalid or deferred, and a deferred one names its own open issue record.
// The rule is armed by ArmReceiptGate alone, so the doc-fidelity gate, which
// reads receipts through CheckGateReceipt with its own words, never meets it.

const (
	dispSha   = "0123456789abcdef0123456789abcdef01234567"
	dispGateA = "iss35-brief-surface-crosscheck"
	dispGateB = "docs-currency-reviewer"
)

type dispFixture struct {
	t    *testing.T
	root string
	cfg  RuleConfig
	hash string
}

// newDispFixture is a content tree with a release-gate manifest at the given
// schemaVersion, both gates holding a conforming receipt with no findings, and
// the receipt gate armed the way a release arms it.
func newDispFixture(t *testing.T, schemaVersion string) *dispFixture {
	t.Helper()
	root := t.TempDir()
	manifest := `{"schemaVersion":` + schemaVersion + `,"detector":"` + dispGateA + `"}` + "\n"
	writeFile(t, root, releaseGateManifestPath, manifest)
	f := &dispFixture{t: t, root: root, hash: hashManifest([]byte(manifest))}
	f.cfg = ArmReceiptGate(Config{}, dispSha, []string{dispGateA, dispGateB}).Rules["receipt_gate"]
	f.put(dispGateA, `[]`)
	f.put(dispGateB, `[]`)
	return f
}

// manifest replaces the tree's manifest and re-pins both receipts to it, so the
// only defect left is the one the manifest carries.
func (f *dispFixture) manifest(body string) {
	f.t.Helper()
	writeFile(f.t, f.root, releaseGateManifestPath, body)
	f.hash = hashManifest([]byte(body))
	f.put(dispGateA, `[]`)
	f.put(dispGateB, `[]`)
}

func (f *dispFixture) put(gate, failing string) {
	f.t.Helper()
	writeFile(f.t, f.root, filepath.Join(reviewsSubdir, dispSha, gate+".json"),
		manifestReceipt(dispSha, gate, releaseTierFull, f.hash, failing))
}

// issue writes a record the ledger reader accepts into the status directory.
func (f *dispFixture) issue(status, id string) string {
	f.t.Helper()
	rel := filepath.Join(".abcd", "work", "issues", status, id+"-a-slug.md")
	writeFile(f.t, f.root, rel, validIssue(id, "a-slug"))
	return rel
}

func (f *dispFixture) run() []Finding {
	f.t.Helper()
	return runReceiptGate(f.t, f.root, f.cfg)
}

// refusal reports the receipt_gate finding whose message carries every part.
func refusal(fs []Finding, parts ...string) bool {
	for _, f := range fs {
		if f.RuleID != "receipt_gate" {
			continue
		}
		all := true
		for _, p := range parts {
			if !strings.Contains(f.Message, p) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func TestReleaseDispositionsAdmitAConformingReceiptPair(t *testing.T) {
	f := newDispFixture(t, "2")
	f.issue("open", "iss-5")
	f.issue("open", "iss-6")
	f.put(dispGateA, `[{"disposition":"fixed"},{"disposition":"invalid"},{"disposition":"deferred","record":"iss-5"}]`)
	f.put(dispGateB, `[{"disposition":"deferred","record":"iss-6"}]`)
	if fs := f.run(); countRule(fs, "receipt_gate") != 0 {
		t.Fatalf("a conforming receipt pair must pass: %+v", fs)
	}
}

func TestReleaseDispositionsRefuseEachMalformedFinding(t *testing.T) {
	cases := []struct {
		name    string
		failing string
		want    []string
	}{
		{"free-text acceptance", `[{"disposition":"accepted: knowingly shipped unfixed"}]`,
			[]string{"'" + dispGateA + "'", "failing[0]", `"accepted: knowingly shipped unfixed"`, "fixed, invalid or deferred"}},
		{"capitalised word", `[{"disposition":"Deferred","record":"iss-5"}]`,
			[]string{"failing[0]", `"Deferred"`, "fixed, invalid or deferred"}},
		{"trailing space", `[{"disposition":"fixed"},{"disposition":"deferred ","record":"iss-5"}]`,
			[]string{"failing[1]", `"deferred "`, "fixed, invalid or deferred"}},
		{"empty disposition", `[{"where":"x"}]`,
			[]string{"failing[0]", `""`, "fixed, invalid or deferred"}},
		{"deferred with no record", `[{"disposition":"deferred"}]`,
			[]string{"failing[0]", "deferred", "names no record"}},
		{"fixed with a record", `[{"disposition":"fixed","record":"iss-5"}]`,
			[]string{"failing[0]", "fixed", `"iss-5"`, "only a deferred finding names a record"}},
		{"invalid with a record", `[{"disposition":"invalid","record":"iss-5"}]`,
			[]string{"failing[0]", "invalid", `"iss-5"`, "only a deferred finding names a record"}},
		{"zero-padded id", `[{"disposition":"deferred","record":"iss-0123"}]`,
			[]string{"failing[0]", `"iss-0123"`, "not an issue id"}},
		{"padded record value", `[{"disposition":"deferred","record":" iss-5"}]`,
			[]string{"failing[0]", `" iss-5"`, "not an issue id"}},
		{"another family's id", `[{"disposition":"deferred","record":"itd-5"}]`,
			[]string{"failing[0]", `"itd-5"`, "not an issue id"}},
		{"missing record", `[{"disposition":"deferred","record":"iss-7"}]`,
			[]string{"failing[0]", "iss-7", "no issue record"}},
		{"resolved record", `[{"disposition":"deferred","record":"iss-8"}]`,
			[]string{"failing[0]", "iss-8", "resolved/", "not open/"}},
		{"record in open and resolved", `[{"disposition":"deferred","record":"iss-9"}]`,
			[]string{"failing[0]", "iss-9", "2 files"}},
		{"record the ledger reader refuses", `[{"disposition":"deferred","record":"iss-10"}]`,
			[]string{"failing[0]", "iss-10", "ledger reader refuses"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newDispFixture(t, "2")
			f.issue("open", "iss-5")
			f.issue("open", "iss-123")
			f.issue("resolved", "iss-8")
			f.issue("open", "iss-9")
			f.issue("resolved", "iss-9")
			writeFile(t, f.root, filepath.Join(".abcd", "work", "issues", "open", "iss-10-a-slug.md"),
				strings.Replace(validIssue("iss-10", "a-slug"), "severity: minor", "severity: 'minor'", 1))
			f.put(dispGateA, c.failing)
			fs := f.run()
			if !refusal(fs, c.want...) {
				t.Fatalf("want a refusal naming %q, got %+v", c.want, fs)
			}
		})
	}
}

func TestReleaseDispositionsRefuseASharedRecord(t *testing.T) {
	t.Run("within one receipt", func(t *testing.T) {
		f := newDispFixture(t, "2")
		f.issue("open", "iss-5")
		f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"},{"disposition":"fixed"},{"disposition":"deferred","record":"iss-5"}]`)
		fs := f.run()
		if !refusal(fs, "iss-5", "2 deferred findings", "failing[0]", "failing[2]") {
			t.Fatalf("a record shared by two findings of one receipt must be refused with its count: %+v", fs)
		}
	})
	t.Run("across two gates", func(t *testing.T) {
		f := newDispFixture(t, "2")
		f.issue("open", "iss-5")
		f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"}]`)
		f.put(dispGateB, `[{"disposition":"deferred","record":"iss-5"}]`)
		fs := f.run()
		if !refusal(fs, "iss-5", "2 deferred findings", "'"+dispGateA+"' failing[0]", "'"+dispGateB+"' failing[0]") {
			t.Fatalf("a record shared across two gates' receipts must be refused with its count: %+v", fs)
		}
	})
	t.Run("a refused receipt does not count", func(t *testing.T) {
		f := newDispFixture(t, "2")
		f.issue("open", "iss-5")
		f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"}]`)
		writeFile(t, f.root, filepath.Join(reviewsSubdir, dispSha, dispGateB+".json"),
			strings.Replace(manifestReceipt(dispSha, dispGateB, releaseTierFull, f.hash, `[{"disposition":"deferred","record":"iss-5"}]`),
				"PROMOTE", "HOLD", 1))
		fs := f.run()
		if refusal(fs, "deferred findings") {
			t.Fatalf("a receipt refused on its verdict must not count toward a shared record: %+v", fs)
		}
	})
}

func TestReleaseDispositionsReadTheRecordGuarded(t *testing.T) {
	t.Run("symlinked record", func(t *testing.T) {
		f := newDispFixture(t, "2")
		writeFile(t, f.root, "elsewhere/iss-5-a-slug.md", validIssue("iss-5", "a-slug"))
		open := filepath.Join(f.root, ".abcd", "work", "issues", "open")
		if err := os.MkdirAll(open, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(f.root, "elsewhere", "iss-5-a-slug.md"), filepath.Join(open, "iss-5-a-slug.md")); err != nil {
			t.Fatal(err)
		}
		f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"}]`)
		fs := f.run()
		if !refusal(fs, "failing[0]", "iss-5", "cannot be read safely") {
			t.Fatalf("a symlinked record must be refused, not followed: %+v", fs)
		}
	})
	t.Run("symlinked status directory", func(t *testing.T) {
		f := newDispFixture(t, "2")
		writeFile(t, f.root, "elsewhere/iss-5-a-slug.md", validIssue("iss-5", "a-slug"))
		issues := filepath.Join(f.root, ".abcd", "work", "issues")
		if err := os.MkdirAll(issues, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(f.root, "elsewhere"), filepath.Join(issues, "open")); err != nil {
			t.Fatal(err)
		}
		f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"}]`)
		fs := f.run()
		if !refusal(fs, "failing[0]", "iss-5", "cannot be read safely") {
			t.Fatalf("a record reached through a symlinked ancestor must be refused: %+v", fs)
		}
	})
}

// A schemaVersion-1 manifest in a v0.14.1 tree is an older content tree
// re-gated: its receipts were written to the old rules, and they stay valid
// for their own commit.
func TestReleaseDispositionsLeaveASchemaVersionOneTreeAlone(t *testing.T) {
	f := newDispFixture(t, "1")
	writeFile(t, f.root, "CHANGELOG.md", "# Changelog\n\n## [0.14.1] - 2026-10-14\n")
	f.put(dispGateA, `[{"disposition":"accepted: knowingly shipped unfixed"},{"disposition":"deferred to iss-2609091956001547"}]`)
	f.put(dispGateB, `[{"disposition":"Deferred","record":"iss-0123"}]`)
	if fs := f.run(); countRule(fs, "receipt_gate") != 0 {
		t.Fatalf("a schemaVersion-1 tree must keep the old rules: %+v", fs)
	}
}

// The rule is armed by the release's arming alone: the same tree read through
// an unarmed RuleConfig — the doc-fidelity gate's reader — meets none of it.
func TestReleaseDispositionsLeaveTheDocFidelityGateAlone(t *testing.T) {
	f := newDispFixture(t, "2")
	f.put(dispGateA, `[{"doc":"brief","sentence":"x","disposition":"confirmed"}]`)
	got, err := CheckGateReceipt(f.root, reviewsSubdir, dispSha, dispGateA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Problems) != 0 || !got.Parsed || len(got.Failing) != 1 || got.Failing[0].Disposition != "confirmed" {
		t.Fatalf("a doc-fidelity receipt must be read as before: %+v", got)
	}
}

func TestReleaseDispositionsRefuseAnUnreadableSchemaVersion(t *testing.T) {
	for name, v := range map[string]string{"string": `"2"`, "fraction": `2.5`, "null": `null`} {
		t.Run(name, func(t *testing.T) {
			f := newDispFixture(t, v)
			fs := f.run()
			if !refusal(fs, "schemaVersion") {
				t.Fatalf("a manifest whose schemaVersion is %s must fail closed: %+v", v, fs)
			}
		})
	}
	t.Run("absent", func(t *testing.T) {
		f := newDispFixture(t, "2")
		f.manifest(`{"detector":"` + dispGateA + `"}` + "\n")
		if fs := f.run(); !refusal(fs, "schemaVersion") {
			t.Fatalf("a manifest with no schemaVersion must fail closed: %+v", fs)
		}
	})
	t.Run("repeated", func(t *testing.T) {
		f := newDispFixture(t, "2")
		f.manifest(`{"schemaVersion":2,"schemaVersion":1}` + "\n")
		if fs := f.run(); !refusal(fs, "manifest", "duplicate key") {
			t.Fatalf("a manifest naming schemaVersion twice must fail closed: %+v", fs)
		}
	})
}

// The ledger a deferred record is read from is the canonical one, pinned by
// the release's arming: a committer-editable issues_dir pointing elsewhere is
// ignored, so records hand-made under a shadow ledger never satisfy the gate
// while the real ledger, its rules and the capture verbs never see them.
func TestReleaseDispositionsReadTheCanonicalLedgerOnly(t *testing.T) {
	f := newDispFixture(t, "2")
	in := Config{Rules: map[string]RuleConfig{"receipt_gate": {IssuesDir: filepath.Join(".abcd", "work", "shadow")}}}
	f.cfg = ArmReceiptGate(in, dispSha, []string{dispGateA, dispGateB}).Rules["receipt_gate"]
	if f.cfg.IssuesDir != recordid.IssuesRelDir {
		t.Errorf("the release's arming must pin the ledger to %s, got %q", recordid.IssuesRelDir, f.cfg.IssuesDir)
	}
	writeFile(t, f.root, filepath.Join(".abcd", "work", "shadow", "open", "iss-5-a-slug.md"), validIssue("iss-5", "a-slug"))
	f.put(dispGateA, `[{"disposition":"deferred","record":"iss-5"}]`)
	if fs := f.run(); !refusal(fs, "failing[0]", "iss-5", "no issue record in "+recordid.IssuesRelDir) {
		t.Fatalf("a record in a shadow ledger must not satisfy the gate: %+v", fs)
	}
}

// The receipts the release gate judges are read from the canonical reviews
// directory, the one release.yml's --derive-content-sha and the local
// uncommitted scan read: a committer-editable receipts_dir pointing elsewhere
// is ignored, so receipts planted in a shadow directory satisfy nothing.
func TestReleaseGateReadsTheCanonicalReceiptsOnly(t *testing.T) {
	f := newDispFixture(t, "2")
	shadow := filepath.Join(".abcd", "work", "shadow-reviews")
	in := Config{Rules: map[string]RuleConfig{"receipt_gate": {ReceiptsDir: shadow}}}
	f.cfg = ArmReceiptGate(in, dispSha, []string{dispGateA, dispGateB}).Rules["receipt_gate"]
	if f.cfg.ReceiptsDir != reviewsSubdir {
		t.Errorf("the release's arming must pin the receipts to %s, got %q", reviewsSubdir, f.cfg.ReceiptsDir)
	}
	// The canonical receipts defer to a record that does not exist; the
	// shadow ones are clean.
	f.put(dispGateA, `[{"disposition":"deferred","record":"iss-7"}]`)
	for _, g := range []string{dispGateA, dispGateB} {
		writeFile(t, f.root, filepath.Join(shadow, dispSha, g+".json"), manifestReceipt(dispSha, g, releaseTierFull, f.hash, `[]`))
	}
	if fs := f.run(); !refusal(fs, "failing[0]", "iss-7", "no issue record") {
		t.Fatalf("receipts in a shadow directory must not stand in for the canonical ones: %+v", fs)
	}
}
