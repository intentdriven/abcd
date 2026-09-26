package lint

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// lintWithin runs Lint on a goroutine and fails the test if it has not returned
// within the deadline, so a FIFO the lint blocks on is a failure rather than a
// hung suite.
func lintWithin(t *testing.T, cfg Config, root string) ([]Finding, error) {
	t.Helper()
	type result struct {
		fs  []Finding
		err error
	}
	done := make(chan result, 1)
	go func() {
		fs, err := Lint(cfg, root)
		done <- result{fs, err}
	}()
	select {
	case r := <-done:
		return r.fs, r.err
	case <-time.After(10 * time.Second):
		t.Fatal("Lint did not return: it blocked on a file it should have refused")
		return nil, nil
	}
}

func mkfifo(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
}

func symlinkOut(t *testing.T, root, rel, content string) {
	t.Helper()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
}

// symlinkDirOut makes rel a link to a directory outside the repository holding
// files, so every read below rel crosses the link at an ANCESTOR, not the leaf.
func symlinkDirOut(t *testing.T, root, rel string, files map[string]string) {
	t.Helper()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outside, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
}

// A path the committed config names is read only inside the repository: the
// persona roster symlinked to a file outside the checkout is refused rather than
// read and trusted (iss-2608211914592726).
func TestPersonaRosterSymlinkedOutOfTheRepoIsRefused(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "rec/.keep", "")
	symlinkOut(t, root, "personas.json", `{"personas":[{"name":"Mallory"}]}`)
	cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{
		"persona_registry": {Enabled: true, Severity: "blocker", Registry: "personas.json"},
	}}
	_, err := lintWithin(t, cfg, root)
	if err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Fatalf("want a refusal naming the repository boundary, got %v", err)
	}
}

// A FIFO at a configured path returns instead of hanging the gate.
func TestContextTargetFIFODoesNotHang(t *testing.T) {
	root := t.TempDir()
	mkfifo(t, filepath.Join(root, "CONTEXT.md"))
	cfg := Config{Rules: map[string]RuleConfig{
		"context_status_free": {Enabled: true, Severity: "blocker", Target: "CONTEXT.md"},
	}}
	if _, err := lintWithin(t, cfg, root); err == nil {
		t.Fatal("a FIFO target must be refused, not read as an empty file")
	}
}

// A walk leaf in the issue ledger (outside every root, so the roots walk never
// sees it) that links out of the repository is refused the way the roots walk
// refuses its own leaves.
func TestIssueLedgerLeafSymlinkedOutIsRefused(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "work/issues/open/.keep", "")
	symlinkOut(t, root, "work/issues/open/iss-7-x.md", "---\nid: \"iss-7\"\n---\n")
	cfg := Config{Rules: map[string]RuleConfig{
		"issue_id_unique": {Enabled: true, Severity: "blocker", IssuesDir: "work/issues"},
	}}
	_, err := lintWithin(t, cfg, root)
	if err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Fatalf("want a refusal naming the repository boundary, got %v", err)
	}
}

// record_schema declines a symlinked or oversized record with a finding, as the
// reading walk declines it, instead of following it (iss-2608301203521317).
func TestRecordSchemaDeclinesSymlinkedAndOversizedRecords(t *testing.T) {
	cfg := Config{Rules: map[string]RuleConfig{
		ruleRecordSchema: {Enabled: true, Severity: "blocker", RecordStores: map[string]string{"iss": "work/issues"}},
	}}
	t.Run("symlinked", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "work/issues/open/.keep", "")
		symlinkOut(t, root, "work/issues/open/iss-9-x.md", "---\nid: \"iss-9\"\nseverity: \"SECRET-TARGET\"\n---\n")
		fs, err := lintWithin(t, cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		assertSingleSafeReadFinding(t, fs, "iss-9-x.md")
	})
	t.Run("oversized", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "work/issues/open/iss-9-x.md", "---\nid: \"iss-9\"\n---\n"+strings.Repeat("x", 1<<20+1))
		fs, err := lintWithin(t, cfg, root)
		if err != nil {
			t.Fatal(err)
		}
		assertSingleSafeReadFinding(t, fs, "iss-9-x.md")
	})
}

func assertSingleSafeReadFinding(t *testing.T, fs []Finding, file string) {
	t.Helper()
	n := 0
	for _, f := range fs {
		if strings.Contains(f.Message, "SECRET-TARGET") {
			t.Fatalf("the link target's frontmatter surfaced in lint output: %+v", f)
		}
		if strings.HasSuffix(f.File, file) && strings.Contains(f.Message, "cannot be read safely") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want one cannot-be-read-safely finding on %s, got %+v", file, fs)
	}
}

// The receipt gate refuses a FIFO receipt and a symlinked one with a fail-closed
// finding: a FIFO once hung the gate, and a link to an out-of-tree forged
// PROMOTE satisfied it (iss-2609012037127981). A link one level up is the same
// forgery — a symlinked commit directory, or a symlinked manifest directory —
// and is refused the same way (iss-2609261016494611).
func TestReceiptGateRefusesUnsafeReceipts(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"
	const gate = "docs-currency-reviewer"
	reviews := filepath.Join(".abcd", "work", "reviews")
	promote := `{"subject":{"digest":{"gitCommit":"` + sha + `"}},"verificationResult":"PROMOTE",` +
		`"judgeModel":"claude-opus-4-8","policy":{"detector":"` + gate + `","version":"1"}}`
	cfg := Config{Rules: map[string]RuleConfig{"receipt_gate": {
		Enabled: true, Severity: "blocker", ReceiptsDir: reviews, Commit: sha, RequiredGates: []string{gate},
	}}}
	receipt := filepath.Join(reviews, sha, gate+".json")
	for name, plant := range map[string]func(t *testing.T, root string){
		"fifo":      func(t *testing.T, root string) { mkfifo(t, filepath.Join(root, receipt)) },
		"symlinked": func(t *testing.T, root string) { symlinkOut(t, root, filepath.ToSlash(receipt), promote) },
		"fifo manifest": func(t *testing.T, root string) {
			writeFile(t, root, receipt, promote)
			mkfifo(t, filepath.Join(root, releaseGateManifestPath))
		},
		// The leaf is a regular file, so O_NOFOLLOW on it refuses nothing: the
		// link is the COMMIT DIRECTORY, which the kernel follows on the way to
		// the leaf (iss-2609261016494611).
		"symlinked commit directory": func(t *testing.T, root string) {
			symlinkDirOut(t, root, filepath.Join(reviews, sha), map[string]string{gate + ".json": promote})
		},
		// The manifest's own directory carried out of the tree: an out-of-tree
		// manifest, and a receipt echoing its hash at the tier it demands.
		"symlinked manifest directory": func(t *testing.T, root string) {
			const manifest = `{"inputs":[]}`
			echo := strings.TrimSuffix(promote, "}") +
				`,"tier":"full","manifestHash":"` + hashManifest([]byte(manifest)) + `"}`
			writeFile(t, root, receipt, echo)
			symlinkDirOut(t, root, filepath.Dir(releaseGateManifestPath),
				map[string]string{filepath.Base(releaseGateManifestPath): manifest})
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			plant(t, root)
			fs, err := lintWithin(t, cfg, root)
			if err != nil {
				t.Fatalf("an unsafe receipt is a finding, not an aborted lint: %v", err)
			}
			if countRule(fs, "receipt_gate") != 1 {
				t.Fatalf("want one fail-closed receipt_gate finding, got %+v", fs)
			}
		})
	}
}

// The prose-citation baseline is an exemption list: every id it names stops
// firing. Its path comes out of the committed config, so a baseline read from
// outside the repository — spelled out with "..", or reached through a
// symlinked directory — would disarm the gate with content the tree does not
// hold. Both are refused, as every other configured path is.
func TestProseCitationBaselineIsReadOnlyInsideTheRepository(t *testing.T) {
	const exempt = `{"schema_version":1,"ids":[{"id":"spc-995","class":"pruned","note":"an exemption the tree does not hold"}]}`
	for name, plant := range map[string]func(t *testing.T, root string) string{
		"climbs out": func(t *testing.T, root string) string {
			if err := os.WriteFile(filepath.Join(filepath.Dir(root), "outside-baseline.json"), []byte(exempt), 0o644); err != nil {
				t.Fatal(err)
			}
			return "../outside-baseline.json"
		},
		"symlinked directory": func(t *testing.T, root string) string {
			symlinkDirOut(t, root, "baselines", map[string]string{"prose.json": exempt})
			return "baselines/prose.json"
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			proseCorpus(t, root)
			writeProseIssue(t, root, "open", "iss-95-ratchet.md", "A newly invented id: spc-995.")
			cfg := proseCfg()
			rc := cfg.Rules[ruleProseCitationResolves]
			rc.Baseline = plant(t, root)
			cfg.Rules[ruleProseCitationResolves] = rc
			fs, err := Lint(cfg, root)
			if err == nil || !strings.Contains(err.Error(), "inside the repository") {
				t.Fatalf("want a containment refusal, got err=%v findings=%+v", err, fs)
			}
		})
	}
}

// The reading walk checks every directory below the issue store for a link, and
// the store root itself on the same terms: a symlinked store root carried the
// whole walk out of the tree, so the outstanding board reported on records the
// repository does not hold.
func TestReadingWalkRefusesASymlinkedStoreRoot(t *testing.T) {
	root := t.TempDir()
	issues := filepath.Join(".abcd", "work", "issues")
	symlinkDirOut(t, root, issues, map[string]string{"README.md": "outside"})
	report, err := ReadReadingOutstanding(root, filepath.ToSlash(issues))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Unsafe) != 1 || report.Unsafe[0].Path != filepath.ToSlash(issues) {
		t.Fatalf("want one unsafe entry naming the store root, got %+v", report.Unsafe)
	}
}
