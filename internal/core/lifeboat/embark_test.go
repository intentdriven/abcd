package lifeboat

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

// embarkableSourceFixture builds a git repo whose records are STORE-LOADABLE
// (real frontmatter that intent.Load / spec.Load / capture.List accept): intents
// across multiple buckets, specs open+closed, issues in all three states with
// validateStrict-passing frontmatter, and ADRs. It deliberately carries:
//   - NO abcd marker block in any record (so copyRecord's strip-on-pack is a
//     no-op and packed bytes == source bytes → the P2 self-closure has no
//     exists-differs), and
//   - a CLAUDE.md that already holds the CURRENT marker block (so EnsureMarker is
//     a no-op on a byte-copy of the source, the P2 marker-idempotence leg), and
//   - NO .abcd/work/DECISIONS.md — the abandoned.json re-derivation (P1) must draw
//     only from embarked families (a superseded intent, a wontfix issue, an ADR
//     with `## Alternatives Considered`), never DECISIONS.md, which embark does
//     not write, or L2.abandoned.json would differ from L1's.
func embarkableSourceFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// ADRs — one carrying an Alternatives Considered section (abandoned-worthy).
	write(".abcd/development/decisions/adrs/0001-record-architecture-decisions.md",
		"# 1. Record architecture decisions\n\n## Context\n\nWe need a durable log.\n\n"+
			"## Alternatives Considered\n\nA wiki; a spreadsheet. Both drift.\n")
	write(".abcd/development/decisions/adrs/0002-single-binary.md",
		"# 2. Single binary\n\n## Context\n\nOne binary holds all behaviour.\n")

	// Issues — all three states, schema-valid frontmatter.
	write(".abcd/work/issues/open/iss-1-open-thing.md",
		"---\nschema_version: 1\nid: iss-1\nslug: open-thing\nseverity: minor\ncategory: bug\n"+
			"source: manual-test\nfound_during: testing\n---\nAn open question.\n")
	write(".abcd/work/issues/resolved/iss-2-resolved-thing.md",
		"---\nschema_version: 1\nid: iss-2\nslug: resolved-thing\nseverity: major\ncategory: bug\n"+
			"source: manual-test\nfound_during: testing\nresolution: fixed at source\n---\nA resolved bug.\n")
	write(".abcd/work/issues/wontfix/iss-3-wont-thing.md",
		"---\nschema_version: 1\nid: iss-3\nslug: wont-thing\nseverity: minor\ncategory: process\n"+
			"source: manual-test\nfound_during: testing\nwontfix_reason: out of scope for now\n---\nA wontfix note.\n")

	// Intents — multiple buckets, incl. a superseded (abandoned-worthy).
	write(".abcd/development/intents/drafts/itd-1-alpha.md",
		"---\nid: itd-1\nslug: alpha\nkind: standalone\nspec_id: spc-1\n---\n# itd-1 alpha\n\nA draft intent.\n")
	write(".abcd/development/intents/planned/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nkind: standalone\nspec_id: spc-2\n---\n# itd-2 beta\n\nA planned intent.\n")
	write(".abcd/development/intents/shipped/itd-3-gamma.md",
		"---\nid: itd-3\nslug: gamma\nkind: standalone\nspec_id: null\n---\n# itd-3 gamma\n\nA shipped intent.\n")
	write(".abcd/development/intents/superseded/itd-4-delta.md",
		"---\nid: itd-4\nslug: delta\nkind: standalone\nspec_id: null\n---\n# itd-4 delta\n\nA superseded intent.\n")

	// Specs — open + closed, renderSpec-shaped frontmatter.
	write(".abcd/development/specs/open/spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-1\n---\n# alpha\n\n## Summary\n\nAn open spec.\n")
	write(".abcd/development/specs/closed/spc-2-beta.md",
		"---\nid: spc-2\nslug: beta\nintent: itd-2\n---\n# beta\n\n## Summary\n\nA closed spec.\n")

	// A conventions router with the CURRENT abcd marker block, so a byte-copy of
	// the source needs no marker write (P2).
	write("CLAUDE.md", "# Project\n\nInvariants: the core never writes to stdout.\n")
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	// Install the CURRENT marker block into CLAUDE.md via the shipped one-code-path.
	if _, err := ahoy.EnsureMarker(filepath.Join(repo, "CLAUDE.md"), false); err != nil {
		t.Fatalf("seed CLAUDE.md marker: %v", err)
	}
	run("init", "-q")
	run("add", "-A")
	run("commit", "-q", "-m", "root")
	return repo
}

// contentFingerprint hashes every file's rel path + bytes (excluding .git), so a
// before/after comparison proves a tree is byte-identical, not merely same-sized.
func contentFingerprint(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	var rels []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if rel == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%s\x00%x\n", rel, sha256.Sum256(data))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// copyTree copies src into dst byte-for-byte (files, dirs, and modes), including
// .git — the P2 self-closure needs an identical git history so the git-derived
// lifeboat files re-derive to the same bytes.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil // the fixture plants none; skip defensively
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

// packSource packs the source repo to a fresh lifeboat dir under a temp HOME and
// returns the lifeboat dir. Mirrors packInto but keeps the source as the caller's.
func packSource(t *testing.T, source string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dest := filepath.Join(t.TempDir(), "lifeboat")
	if _, err := Pack(source, dest, okScan); err != nil {
		t.Fatalf("Pack: %v", err)
	}
	return dest
}

func readProvenanceFile(t *testing.T, lifeboatDir string) Provenance {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(lifeboatDir, ProvenanceName))
	if err != nil {
		t.Fatal(err)
	}
	var prov Provenance
	if err := json.Unmarshal(data, &prov); err != nil {
		t.Fatal(err)
	}
	return prov
}

// ---------------------------------------------------------------------------
// resolveTarget — the inverse mapping table (§5)
// ---------------------------------------------------------------------------

func TestResolveTargetMapsEveryFamily(t *testing.T) {
	cases := []struct {
		name    string
		rel     string
		wantFam string
		wantTgt string
		wantDsp disposition
	}{
		{"adr flat", "docs/adrs/0001-x.md", "adrs", ".abcd/development/decisions/adrs/0001-x.md", dispPlanned},
		{"issue open", "activity/issues/open/iss-1-x.md", "issues", ".abcd/work/issues/open/iss-1-x.md", dispPlanned},
		{"issue resolved", "activity/issues/resolved/iss-2-x.md", "issues", ".abcd/work/issues/resolved/iss-2-x.md", dispPlanned},
		{"issue wontfix", "activity/issues/wontfix/iss-3-x.md", "issues", ".abcd/work/issues/wontfix/iss-3-x.md", dispPlanned},
		{"intent bucketed", "rescue/intents/planned/itd-2-x.md", "intents", ".abcd/development/intents/planned/itd-2-x.md", dispPlanned},
		{"intent bucket-less -> drafts", "rescue/intents/itd-9-x.md", "intents", ".abcd/development/intents/drafts/itd-9-x.md", dispPlanned},
		{"spec open", "rescue/specs/open/spc-1-x.md", "specs", ".abcd/development/specs/open/spc-1-x.md", dispPlanned},
		{"spec closed", "rescue/specs/closed/spc-2-x.md", "specs", ".abcd/development/specs/closed/spc-2-x.md", dispPlanned},
		// Unmapped: unknown bucket for a load-bearing status family.
		{"issue unknown bucket", "activity/issues/bogus/iss-9-x.md", "issues", "", dispUnmapped},
		{"spec bucket-less", "rescue/specs/spc-9-x.md", "specs", "", dispUnmapped},
		// Report-only + unknown.
		{"brief report-only", "brief/01-product/README.md", "", "", dispReportOnly},
		{"coverage report-only", "coverage.json", "", "", dispReportOnly},
		{"graveyard report-only", "graveyard/archaeology.json", "", "", dispReportOnly},
		{"spine report-only", "rescue/spine.md", "", "", dispReportOnly},
		{"provenance report-only", "_provenance.json", "", "", dispReportOnly},
		{"foreign unknown", "foo/bar.md", "", "", dispUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fam, tgt, disp, _ := resolveTarget(tc.rel)
			if disp != tc.wantDsp {
				t.Fatalf("disposition = %v, want %v", disp, tc.wantDsp)
			}
			if disp == dispPlanned {
				if fam != tc.wantFam || tgt != tc.wantTgt {
					t.Errorf("planned = (%q,%q), want (%q,%q)", fam, tgt, tc.wantFam, tc.wantTgt)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// classifyEmbark — conflict classification (§7 step 7)
// ---------------------------------------------------------------------------

func TestClassifyEmbarkConflicts(t *testing.T) {
	content := []byte("hello\n")

	t.Run("absent -> create", func(t *testing.T) {
		tgt := t.TempDir()
		pe, cf := classifyEmbark(tgt, "docs/adrs/x.md", "sub/x.md", "adrs", content)
		if cf != nil {
			t.Fatalf("unexpected conflict: %+v", cf)
		}
		if pe.Action != ActionCreate {
			t.Errorf("action = %q, want create", pe.Action)
		}
	})

	t.Run("identical -> unchanged", func(t *testing.T) {
		tgt := t.TempDir()
		mustWrite(t, filepath.Join(tgt, "x.md"), content)
		pe, cf := classifyEmbark(tgt, "docs/adrs/x.md", "x.md", "adrs", content)
		if cf != nil {
			t.Fatalf("unexpected conflict: %+v", cf)
		}
		if pe.Action != ActionUnchanged {
			t.Errorf("action = %q, want unchanged", pe.Action)
		}
	})

	t.Run("differing -> exists-differs", func(t *testing.T) {
		tgt := t.TempDir()
		mustWrite(t, filepath.Join(tgt, "x.md"), []byte("other\n"))
		_, cf := classifyEmbark(tgt, "docs/adrs/x.md", "x.md", "adrs", content)
		if cf == nil || cf.Kind != ConflictExistsDiffers {
			t.Fatalf("want exists-differs, got %+v", cf)
		}
	})

	t.Run("target is a dir -> target-not-regular", func(t *testing.T) {
		tgt := t.TempDir()
		if err := os.Mkdir(filepath.Join(tgt, "x.md"), 0o755); err != nil {
			t.Fatal(err)
		}
		_, cf := classifyEmbark(tgt, "docs/adrs/x.md", "x.md", "adrs", content)
		if cf == nil || cf.Kind != ConflictTargetNotRegular {
			t.Fatalf("want target-not-regular, got %+v", cf)
		}
	})

	t.Run("target is a symlink -> target-not-regular", func(t *testing.T) {
		tgt := t.TempDir()
		real := filepath.Join(tgt, "real")
		mustWrite(t, real, content)
		if err := os.Symlink(real, filepath.Join(tgt, "x.md")); err != nil {
			t.Skipf("cannot symlink: %v", err)
		}
		_, cf := classifyEmbark(tgt, "docs/adrs/x.md", "x.md", "adrs", content)
		if cf == nil || cf.Kind != ConflictTargetNotRegular {
			t.Fatalf("want target-not-regular, got %+v", cf)
		}
	})

	t.Run("parent is a file -> parent-not-dir", func(t *testing.T) {
		tgt := t.TempDir()
		mustWrite(t, filepath.Join(tgt, "sub"), content) // "sub" is a file, not a dir
		_, cf := classifyEmbark(tgt, "docs/adrs/x.md", "sub/x.md", "adrs", content)
		if cf == nil || cf.Kind != ConflictParentNotDir {
			t.Fatalf("want parent-not-dir, got %+v", cf)
		}
	})

	t.Run("parent is a symlink -> parent-not-dir", func(t *testing.T) {
		tgt := t.TempDir()
		realDir := filepath.Join(tgt, "realdir")
		if err := os.Mkdir(realDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realDir, filepath.Join(tgt, "sub")); err != nil {
			t.Skipf("cannot symlink: %v", err)
		}
		_, cf := classifyEmbark(tgt, "docs/adrs/x.md", "sub/x.md", "adrs", content)
		if cf == nil || cf.Kind != ConflictParentNotDir {
			t.Fatalf("want parent-not-dir, got %+v", cf)
		}
	})
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// VerifyManifest
// ---------------------------------------------------------------------------

func TestVerifyManifestIntactLifeboat(t *testing.T) {
	repo := packFixture(t)
	dest, _ := packInto(t, repo, okScan)
	if err := VerifyManifest(dest); err != nil {
		t.Errorf("intact lifeboat failed verification: %v", err)
	}
}

func TestVerifyManifestCatchesTampering(t *testing.T) {
	t.Run("flipped record byte", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		adr := filepath.Join(dest, "docs/adrs/0001-example.md")
		data, _ := os.ReadFile(adr)
		if err := os.WriteFile(adr, append(data, '!'), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := VerifyManifest(dest); err == nil {
			t.Error("a flipped record byte must fail verification")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		if err := os.Remove(filepath.Join(dest, "docs/adrs/0001-example.md")); err != nil {
			t.Fatal(err)
		}
		if err := VerifyManifest(dest); err == nil {
			t.Error("a missing manifest file must fail verification")
		}
	})

	t.Run("extra file", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		if err := os.WriteFile(filepath.Join(dest, "docs/adrs/planted.md"), []byte("foreign\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := VerifyManifest(dest); err == nil {
			t.Error("an extra manifest-relevant file must fail verification")
		}
	})

	t.Run("symlink inside the lifeboat", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		if err := os.Symlink(filepath.Join(dest, "coverage.json"), filepath.Join(dest, "link.json")); err != nil {
			t.Skipf("cannot symlink: %v", err)
		}
		if err := VerifyManifest(dest); err == nil {
			t.Error("a symlink inside the lifeboat must fail verification")
		}
	})

	t.Run("oversize file", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		big := make([]byte, maxEmbarkFileBytes+1)
		if err := os.WriteFile(filepath.Join(dest, "docs/adrs/huge.md"), big, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := VerifyManifest(dest); err == nil {
			t.Error("an oversize file must fail verification")
		}
	})
}

func TestVerifyManifestToleratesLayer3(t *testing.T) {
	repo := packFixture(t)
	dest, _ := packInto(t, repo, okScan)
	// The post-pack layer-3 interpretation is deliberately excluded from the
	// manifest; its presence must NOT break verification.
	mustWrite(t, filepath.Join(dest, "graveyard/lessons.json"), []byte(`{"schema_version":1,"lessons":[]}`+"\n"))
	mustWrite(t, filepath.Join(dest, "graveyard/low-confidence/x.json"), []byte(`{"schema_version":1,"lessons":[]}`+"\n"))
	if err := VerifyManifest(dest); err != nil {
		t.Errorf("layer-3 files broke verification: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EmbarkProbe — gates + mapping
// ---------------------------------------------------------------------------

func TestEmbarkProbeGates(t *testing.T) {
	t.Run("non-lifeboat dir -> error", func(t *testing.T) {
		lb := t.TempDir() // real dir, no _provenance.json
		if _, err := EmbarkProbe(lb, t.TempDir()); err == nil {
			t.Error("a non-lifeboat dir must be a structural error")
		}
	})

	t.Run("schema-too-new -> upgrade error", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		bumpProvenanceSchema(t, dest, 99)
		_, err := EmbarkProbe(dest, t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "update abcd:") {
			t.Errorf("schema-too-new must return the update message, got: %v", err)
		}
	})

	t.Run("symlinked target -> error", func(t *testing.T) {
		repo := packFixture(t)
		dest, _ := packInto(t, repo, okScan)
		realTgt := t.TempDir()
		link := filepath.Join(t.TempDir(), "tgt")
		if err := os.Symlink(realTgt, link); err != nil {
			t.Skipf("cannot symlink: %v", err)
		}
		if _, err := EmbarkProbe(dest, link); err == nil {
			t.Error("a symlinked target must be a structural error")
		}
	})
}

func TestEmbarkProbeMapsFamilies(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	plan, err := EmbarkProbe(dest, t.TempDir())
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	if !plan.ManifestVerified {
		t.Error("plan.ManifestVerified = false, want true")
	}
	// Every family maps to its canonical target, all creates into a fresh target.
	wantTargets := map[string]bool{
		".abcd/development/decisions/adrs/0001-record-architecture-decisions.md": false,
		".abcd/development/decisions/adrs/0002-single-binary.md":                 false,
		".abcd/work/issues/open/iss-1-open-thing.md":                             false,
		".abcd/work/issues/resolved/iss-2-resolved-thing.md":                     false,
		".abcd/work/issues/wontfix/iss-3-wont-thing.md":                          false,
		".abcd/development/intents/drafts/itd-1-alpha.md":                        false,
		".abcd/development/intents/planned/itd-2-beta.md":                        false,
		".abcd/development/intents/shipped/itd-3-gamma.md":                       false,
		".abcd/development/intents/superseded/itd-4-delta.md":                    false,
		".abcd/development/specs/open/spc-1-alpha.md":                            false,
		".abcd/development/specs/closed/spc-2-beta.md":                           false,
	}
	for _, p := range plan.Planned {
		if p.Action != ActionCreate {
			t.Errorf("%s: action %q, want create", p.TargetPath, p.Action)
		}
		if _, ok := wantTargets[p.TargetPath]; ok {
			wantTargets[p.TargetPath] = true
		}
	}
	for tgt, seen := range wantTargets {
		if !seen {
			t.Errorf("planned target missing: %s", tgt)
		}
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("fresh target must have no conflicts, got %d", len(plan.Conflicts))
	}
}

// ---------------------------------------------------------------------------
// EmbarkFrom — the round-trip through the real stores
// ---------------------------------------------------------------------------

func TestEmbarkFromRoundTrip(t *testing.T) {
	source := embarkableSourceFixture(t)
	srcFP := contentFingerprint(t, source)
	dest := packSource(t, source)

	target := t.TempDir()
	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	if res.Written == 0 || len(res.Conflicts) != 0 {
		t.Fatalf("unexpected result: written=%d conflicts=%d", res.Written, len(res.Conflicts))
	}

	// intent.Load semantic equality (source vs target).
	srcIntents, err := intent.Load(source)
	if err != nil {
		t.Fatal(err)
	}
	tgtIntents, err := intent.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sortIntents(srcIntents.Intents), sortIntents(tgtIntents.Intents)) {
		t.Errorf("intent corpus differs:\n src=%+v\n tgt=%+v", srcIntents.Intents, tgtIntents.Intents)
	}

	// spec.Load semantic equality.
	srcSpecs, err := spec.Load(source)
	if err != nil {
		t.Fatal(err)
	}
	tgtSpecs, err := spec.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sortSpecs(srcSpecs.Specs), sortSpecs(tgtSpecs.Specs)) {
		t.Errorf("spec store differs:\n src=%+v\n tgt=%+v", srcSpecs.Specs, tgtSpecs.Specs)
	}

	// capture.List semantic equality (Path is absolute; compare id-keyed fields).
	if src, tgt := listIssues(t, source), listIssues(t, target); !reflect.DeepEqual(src, tgt) {
		t.Errorf("issue ledger differs:\n src=%+v\n tgt=%+v", src, tgt)
	}

	// ADRs byte-identical.
	for _, rel := range []string{
		".abcd/development/decisions/adrs/0001-record-architecture-decisions.md",
		".abcd/development/decisions/adrs/0002-single-binary.md",
	} {
		s, _ := os.ReadFile(filepath.Join(source, rel))
		g, _ := os.ReadFile(filepath.Join(target, rel))
		if string(s) != string(g) {
			t.Errorf("%s not byte-identical after the trip", rel)
		}
	}

	// Target CLAUDE.md carries the CURRENT marker block (dry-run predicts no change).
	changed, err := ahoy.EnsureMarker(filepath.Join(target, "CLAUDE.md"), true)
	if err != nil || changed {
		t.Errorf("target CLAUDE.md not current after embark: changed=%v err=%v", changed, err)
	}

	// Source tree byte-identical after the whole trip.
	if after := contentFingerprint(t, source); after != srcFP {
		t.Error("embark mutated the source tree")
	}
}

func TestEmbarkFromReRunIsIdempotent(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	target := t.TempDir()
	if _, err := EmbarkFrom(dest, target); err != nil {
		t.Fatalf("first embark: %v", err)
	}
	res, err := EmbarkFrom(dest, target)
	if err != nil {
		t.Fatalf("second embark: %v", err)
	}
	if res.Written != 0 {
		t.Errorf("re-embark wrote %d files, want 0 (all unchanged)", res.Written)
	}
	if res.Unchanged == 0 {
		t.Errorf("re-embark reported 0 unchanged, want every record unchanged")
	}
	if res.Marker.Changed {
		t.Errorf("re-embark marker changed, want current")
	}
}

func TestEmbarkFromConflictWritesNothing(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	target := t.TempDir()

	// Plant a differing file at one canonical target location.
	adr := filepath.Join(target, ".abcd/development/decisions/adrs/0001-record-architecture-decisions.md")
	mustWrite(t, adr, []byte("DIFFERENT CONTENT\n"))
	before := contentFingerprint(t, target)

	res, err := EmbarkFrom(dest, target)
	if !errors.Is(err, ErrEmbarkConflicts) {
		t.Fatalf("want ErrEmbarkConflicts, got: %v", err)
	}
	if res.Written != 0 {
		t.Errorf("conflict refusal wrote %d files, want 0", res.Written)
	}
	if len(res.Conflicts) == 0 {
		t.Error("conflict refusal returned no conflicts")
	}
	if after := contentFingerprint(t, target); after != before {
		t.Error("conflict refusal mutated the target (must write nothing)")
	}
}

// ---------------------------------------------------------------------------
// Ignored classification, symlink refusal, coverage handoff
// ---------------------------------------------------------------------------

func TestEmbarkProbeIgnoredClassification(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	// Plant an unknown foreign file and an unknown-bucket issue INTO the lifeboat,
	// then re-seal so VerifyManifest still passes.
	mustWrite(t, filepath.Join(dest, "foo/bar.md"), []byte("foreign\n"))
	mustWrite(t, filepath.Join(dest, "activity/issues/bogus/iss-9-x.md"), []byte("bad bucket\n"))
	reseal(t, dest)

	plan, err := EmbarkProbe(dest, t.TempDir())
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	reasons := map[string]IgnoredReason{}
	for _, ig := range plan.Ignored {
		reasons[ig.LifeboatPath] = ig.Reason
	}
	checks := map[string]IgnoredReason{
		"brief/01-product/README.md":       "", // report-only presence not guaranteed; checked below
		"coverage.json":                    IgnoredReportOnly,
		"graveyard/archaeology.json":       IgnoredReportOnly,
		"_provenance.json":                 IgnoredReportOnly,
		"foo/bar.md":                       IgnoredUnknown,
		"activity/issues/bogus/iss-9-x.md": IgnoredUnmapped,
	}
	for p, want := range checks {
		if want == "" {
			continue
		}
		if got := reasons[p]; got != want {
			t.Errorf("ignored[%s] = %q, want %q", p, got, want)
		}
	}
}

func TestEmbarkRefusesSymlinkedLifeboatFile(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	// Replace a record with a symlink; VerifyManifest (and the walk) must refuse it.
	adr := filepath.Join(dest, "docs/adrs/0001-record-architecture-decisions.md")
	if err := os.Remove(adr); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dest, "coverage.json"), adr); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	if _, err := EmbarkProbe(dest, t.TempDir()); err == nil {
		t.Error("a symlinked lifeboat file must be refused")
	}
}

func TestReadPackedCoverage(t *testing.T) {
	t.Run("absent -> not present", func(t *testing.T) {
		h := readPackedCoverage(t.TempDir())
		if h.Present || h.Degraded {
			t.Errorf("absent coverage: %+v, want Present:false", h)
		}
	})

	t.Run("garbage -> degraded", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "coverage.json"), []byte("{not json"))
		h := readPackedCoverage(dir)
		if !h.Present || !h.Degraded {
			t.Errorf("garbage coverage: %+v, want Present+Degraded", h)
		}
	})

	t.Run("valid -> blanks extracted", func(t *testing.T) {
		dir := t.TempDir()
		cov := Coverage{
			SchemaVersion: SchemaVersion,
			Summary:       Summary{Grounded: 1, Partial: 0, Blank: 2},
			Sections: []SectionCoverage{
				{Name: "product/context", Status: StatusGrounded},
				{Name: "product/personas", Kind: KindHumanOwned, Status: StatusBlank,
					Question: "Who is this for?", Searched: []string{"personas registry"}},
				{Name: "evidence/tradeoffs", Status: StatusBlank, Question: "What was weighed?"},
			},
		}
		data, _ := json.Marshal(cov)
		mustWrite(t, filepath.Join(dir, "coverage.json"), data)
		h := readPackedCoverage(dir)
		if !h.Present || h.Degraded {
			t.Fatalf("valid coverage: %+v", h)
		}
		if len(h.Blanks) != 2 {
			t.Fatalf("blanks = %d, want 2 (blank sections only)", len(h.Blanks))
		}
		if h.Blanks[0].Section != "product/personas" || h.Blanks[0].Question != "Who is this for?" {
			t.Errorf("blank[0] = %+v", h.Blanks[0])
		}
	})
}

// ---------------------------------------------------------------------------
// P1 — record-derived sub-manifest closure
// ---------------------------------------------------------------------------

func TestP1RecordManifestClosure(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)

	l1, err := Plan(source)
	if err != nil {
		t.Fatal(err)
	}
	prov1 := readProvenanceFile(t, dest)

	target := t.TempDir()
	if _, err := EmbarkFrom(dest, target); err != nil {
		t.Fatalf("EmbarkFrom: %v", err)
	}
	l2, err := Plan(target)
	if err != nil {
		t.Fatal(err)
	}

	h1 := RecordManifestSHA256(l1.Files)
	h2 := RecordManifestSHA256(l2.Files)
	if h1 != h2 {
		t.Errorf("record manifest hash not closed: L1=%s L2=%s", h1, h2)
		// Diagnose which record family diverged.
		diffRecordFiles(t, l1.Files, l2.Files)
	}
	if prov1.RecordManifestSHA256 != h1 {
		t.Errorf("packed provenance record hash %s != Plan(source) %s", prov1.RecordManifestSHA256, h1)
	}
	if prov1.RecordManifestSHA256 != h2 {
		t.Errorf("packed provenance record hash %s != re-pack %s", prov1.RecordManifestSHA256, h2)
	}
}

// ---------------------------------------------------------------------------
// P2 — literal self-closure
// ---------------------------------------------------------------------------

func TestP2SelfClosure(t *testing.T) {
	source := embarkableSourceFixture(t)
	dest := packSource(t, source)
	origManifest := readProvenanceFile(t, dest).ManifestSHA256

	// A byte-copy of the source repo, SAME basename (Repo.Name is the basename) and
	// SAME .git (so git-derived lifeboat files re-derive identically).
	parent := t.TempDir()
	copyDir := filepath.Join(parent, filepath.Base(source))
	copyTree(t, source, copyDir)
	before := contentFingerprint(t, copyDir)

	res, err := EmbarkFrom(dest, copyDir)
	if err != nil {
		t.Fatalf("EmbarkFrom into a self-copy: %v", err)
	}
	if res.Written != 0 {
		t.Errorf("self-copy embark wrote %d files, want 0 (all unchanged)", res.Written)
	}
	if res.Marker.Changed {
		t.Errorf("self-copy marker changed, want current (idempotent)")
	}
	if after := contentFingerprint(t, copyDir); after != before {
		t.Error("self-copy embark mutated the copy (must be a pure no-op)")
	}

	// Re-pack the copy reproduces the EXACT original manifest_sha256.
	lb2, err := Plan(copyDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := ManifestSHA256(lb2.Files); got != origManifest {
		t.Errorf("self-closure manifest mismatch: re-pack=%s original=%s", got, origManifest)
	}
}

// ---------------------------------------------------------------------------
// Drift guards — the family table cannot silently diverge from the stores.
// ---------------------------------------------------------------------------

func TestFamilyTableDriftGuards(t *testing.T) {
	if !reflect.DeepEqual(intentEmbarkBuckets, intent.Buckets) {
		t.Errorf("intentEmbarkBuckets %v != intent.Buckets %v", intentEmbarkBuckets, intent.Buckets)
	}
	if !reflect.DeepEqual(specEmbarkBuckets, []string{spec.StatusOpen, spec.StatusClosed}) {
		t.Errorf("specEmbarkBuckets %v != {open, closed}", specEmbarkBuckets)
	}
	prefixes := map[string]string{}
	for _, f := range embarkFamilies {
		prefixes[f.Name] = f.TargetPrefix
	}
	wants := map[string]string{
		"intents": intent.IntentsRelDir + "/",
		"specs":   spec.SpecsRelDir + "/",
		"issues":  capture.LedgerRelPath + "/",
		"adrs":    nativeADRDir + "/",
	}
	for name, want := range wants {
		if got := prefixes[name]; got != want {
			t.Errorf("family %s TargetPrefix = %q, want %q", name, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func sortIntents(in []intent.Intent) []intent.Intent {
	out := append([]intent.Intent(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortSpecs(in []spec.Spec) []spec.Spec {
	out := append([]spec.Spec(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// issueKey is the Path-independent comparable projection of an issue.
type issueKey struct {
	ID, Slug, Sev, Cat, Src, FoundDuring, Resolution, Wontfix, Body string
	Status                                                          string
}

func listIssues(t *testing.T, repoRoot string) []issueKey {
	t.Helper()
	res, err := capture.List(capture.ListRequest{RepoRoot: repoRoot})
	if err != nil {
		t.Fatal(err)
	}
	var keys []issueKey
	for _, iss := range res.Issues {
		keys = append(keys, issueKey{
			ID: iss.ID, Slug: iss.Slug, Sev: string(iss.Severity), Cat: string(iss.Category),
			Src: string(iss.Source), FoundDuring: iss.FoundDuring, Resolution: iss.Resolution,
			Wontfix: iss.WontfixReason, Body: iss.Body, Status: string(iss.Status),
		})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].ID < keys[j].ID })
	return keys
}

// bumpProvenanceSchema rewrites _provenance.json's schema_version in place. It
// does not re-hash: the schema gate runs before manifest verification, so the
// too-new version is caught first.
func bumpProvenanceSchema(t *testing.T, dest string, v int) {
	t.Helper()
	prov := readProvenanceFile(t, dest)
	prov.SchemaVersion = v
	data, err := json.MarshalIndent(prov, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, ProvenanceName), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reseal recomputes manifest_sha256 over the current on-disk (non-excluded) tree
// and rewrites _provenance.json, so a test that plants files into a packed
// lifeboat keeps VerifyManifest passing. It mirrors the manifest construction.
func reseal(t *testing.T, dest string) {
	t.Helper()
	root, err := os.OpenRoot(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	rels, err := walkLifeboatFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	var files []PlannedFile
	for _, rel := range rels {
		if isManifestExcluded(rel) {
			continue
		}
		data, err := readLifeboatFile(root, rel)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, PlannedFile{Path: rel, Content: data})
	}
	prov := readProvenanceFile(t, dest)
	prov.ManifestSHA256 = ManifestSHA256(files)
	data, err := json.MarshalIndent(prov, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, ProvenanceName), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// diffRecordFiles reports which record-derived files differ between two plans.
func diffRecordFiles(t *testing.T, a, b []PlannedFile) {
	t.Helper()
	index := func(fs []PlannedFile) map[string][]byte {
		m := map[string][]byte{}
		for _, f := range fs {
			if isRecordDerived(f.Path) {
				m[f.Path] = f.Content
			}
		}
		return m
	}
	ma, mb := index(a), index(b)
	for p, ca := range ma {
		cb, ok := mb[p]
		if !ok {
			t.Logf("record %s present in L1, absent in L2", p)
			continue
		}
		if string(ca) != string(cb) {
			t.Logf("record %s differs:\n L1=%q\n L2=%q", p, ca, cb)
		}
	}
	for p := range mb {
		if _, ok := ma[p]; !ok {
			t.Logf("record %s present in L2, absent in L1", p)
		}
	}
}

// TestEmbarkProbeRenderSanitisesProvenanceHash: record_manifest_sha256 comes
// verbatim from the untrusted _provenance.json (which manifest verification
// deliberately excludes), so the human render must sanitise it like every
// other lifeboat-derived string — an ANSI escape must never reach the terminal.
func TestEmbarkProbeRenderSanitisesProvenanceHash(t *testing.T) {
	source := embarkableSourceFixture(t)
	lifeboat := packSource(t, source)

	provPath := filepath.Join(lifeboat, ProvenanceName)
	raw, err := os.ReadFile(provPath)
	if err != nil {
		t.Fatal(err)
	}
	var prov map[string]any
	if err := json.Unmarshal(raw, &prov); err != nil {
		t.Fatal(err)
	}
	prov["record_manifest_sha256"] = "\x1b[31mVERIFIED CLEAN\x1b[0m"
	tampered, err := json.Marshal(prov)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, provPath, tampered)

	target := t.TempDir()
	plan, err := EmbarkProbe(lifeboat, target)
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	if out := plan.Render(); strings.ContainsRune(out, 0x1b) {
		t.Fatalf("render leaks a raw control byte from provenance:\n%q", out)
	}
}

// TestEmbarkDuplicateTargetIsConflict: two lifeboat files that resolve to the
// same embark target (a bucket-less rescue/intents/<leaf> and the bucketed
// rescue/intents/drafts/<leaf>) must surface as a conflict, never a silent
// last-writer-wins overwrite.
func TestEmbarkDuplicateTargetIsConflict(t *testing.T) {
	source := embarkableSourceFixture(t)
	// A bucket-less intent alongside a drafts intent with the same leaf: both
	// map to .abcd/development/intents/drafts/<leaf> on embark.
	mustWrite(t, filepath.Join(source, ".abcd/development/intents/drafts/itd-77-dup.md"),
		[]byte("---\nid: itd-77\nslug: dup\nkind: feature\n---\n# drafts copy\n"))
	mustWrite(t, filepath.Join(source, ".abcd/development/intents/itd-77-dup.md"),
		[]byte("---\nid: itd-77\nslug: dup\nkind: feature\n---\n# bucket-less copy\n"))
	lifeboat := packSource(t, source)

	target := t.TempDir()
	plan, err := EmbarkProbe(lifeboat, target)
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	dup := false
	for _, c := range plan.Conflicts {
		if c.Kind == ConflictDuplicateTarget {
			dup = true
		}
	}
	if !dup {
		t.Fatalf("no duplicate-target conflict reported; conflicts=%+v planned=%d", plan.Conflicts, len(plan.Planned))
	}

	before := contentFingerprint(t, target)
	if _, err := EmbarkFrom(lifeboat, target); err == nil {
		t.Fatal("EmbarkFrom succeeded despite a duplicate-target conflict")
	}
	if contentFingerprint(t, target) != before {
		t.Fatal("EmbarkFrom wrote into the target on the conflict path")
	}
}

// TestEmbarkCaseVariantTargetIsConflict is the case-folding detector over the
// duplicate-target guard (iss-2608270500193735). The guard keys a plain Go map
// on the resolved target string, so two lifeboat files whose targets differ only
// in case are two distinct keys, no conflict fires, and both plan as Create
// (planning precedes writing, so neither target exists yet). On a case-folding
// target filesystem writeEmbark's overwriting rename then lands the second on
// the first — one record silently lost — while the result reports both as
// written. The lifeboat is untrusted content that names both files, so the
// attacker chooses which body survives.
//
// The two lifeboat paths differ by DIRECTORY, not only by case (a bucket-less
// intent and a bucketed one), so the fixture builds on a case-folding host and a
// case-sensitive one alike, and the fold predicate is substituted rather than
// read off the host — the refusal is proven either way.
func TestEmbarkCaseVariantTargetIsConflict(t *testing.T) {
	build := func(t *testing.T) string {
		t.Helper()
		source := embarkableSourceFixture(t)
		lifeboat := packSource(t, source)
		// Both resolve under .abcd/development/intents/drafts/: the bucket-less
		// file through the default bucket, the bucketed one directly. The leaves
		// differ only in case.
		mustWrite(t, filepath.Join(lifeboat, "rescue/intents/itd-77-dup.md"),
			[]byte("---\nid: itd-77\nslug: dup\nkind: feature\n---\n# bucket-less copy\n"))
		mustWrite(t, filepath.Join(lifeboat, "rescue/intents/drafts/ITD-77-DUP.md"),
			[]byte("---\nid: itd-77\nslug: dup\nkind: feature\n---\n# re-cased copy\n"))
		reseal(t, lifeboat)
		return lifeboat
	}
	hasDup := func(t *testing.T, conflicts []Conflict) bool {
		t.Helper()
		for _, c := range conflicts {
			if c.Kind == ConflictDuplicateTarget {
				return true
			}
		}
		return false
	}

	restore := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = restore })

	t.Run("case-folding filesystem refuses the whole write", func(t *testing.T) {
		caseFoldingFS = func() bool { return true }
		lifeboat := build(t)
		target := t.TempDir()

		plan, err := EmbarkProbe(lifeboat, target)
		if err != nil {
			t.Fatalf("EmbarkProbe: %v", err)
		}
		if !hasDup(t, plan.Conflicts) {
			t.Fatalf("no duplicate-target conflict for two case-variant targets; conflicts=%+v planned=%d",
				plan.Conflicts, len(plan.Planned))
		}

		before := contentFingerprint(t, target)
		res, err := EmbarkFrom(lifeboat, target)
		if err == nil {
			t.Fatalf("EmbarkFrom succeeded on a case-variant collision; Written=%d Families=%v", res.Written, res.Families)
		}
		if res.Written != 0 {
			t.Errorf("EmbarkFrom reported %d written on the conflict path, want 0", res.Written)
		}
		if contentFingerprint(t, target) != before {
			t.Fatal("EmbarkFrom wrote into the target on the conflict path")
		}
	})

	t.Run("case-sensitive filesystem keeps both", func(t *testing.T) {
		caseFoldingFS = func() bool { return false }
		lifeboat := build(t)
		target := t.TempDir()

		plan, err := EmbarkProbe(lifeboat, target)
		if err != nil {
			t.Fatalf("EmbarkProbe: %v", err)
		}
		if hasDup(t, plan.Conflicts) {
			t.Fatalf("two case-variant targets are distinct files on a case-sensitive filesystem; conflicts=%+v", plan.Conflicts)
		}
	})
}

// stripPassBExemption rewrites a packed lifeboat's provenance in place without
// the exemption marker — the shape every lifeboat packed before the field
// existed has.
func stripPassBExemption(t *testing.T, lifeboat string) {
	t.Helper()
	provPath := filepath.Join(lifeboat, ProvenanceName)
	raw, err := os.ReadFile(provPath)
	if err != nil {
		t.Fatal(err)
	}
	var prov map[string]any
	if err := json.Unmarshal(raw, &prov); err != nil {
		t.Fatal(err)
	}
	if _, ok := prov["pass_b_exemption"]; !ok {
		t.Fatal("the packed lifeboat carries no exemption to strip; the test is not testing what it says")
	}
	delete(prov, "pass_b_exemption")
	out, err := json.Marshal(prov)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, provPath, out)
}

// TestEmbarkReportsADeclaredPassBExemption is the consumer half of the promise.
// A lifeboat that declares Pass B exempt must be READ as exempt: the reader is
// told the pass did not run and why, in the same blanks-first handoff that lists
// what they have to answer.
func TestEmbarkReportsADeclaredPassBExemption(t *testing.T) {
	source := embarkableSourceFixture(t)
	lifeboat := packSource(t, source)

	plan, err := EmbarkProbe(lifeboat, t.TempDir())
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	if plan.Coverage == nil || plan.Coverage.PassBExemption == nil {
		t.Fatal("the handoff does not carry the declared exemption")
	}
	out := plan.Render()
	if !strings.Contains(out, "pass B") || !strings.Contains(out, plan.Coverage.PassBExemption.Reason) {
		t.Errorf("the report does not name the declared exemption:\n%s", out)
	}
}

// TestEmbarkUnmarkedProvenanceReadsAsBefore is the compatibility half: a lifeboat
// with no marker is a lifeboat with no exemption, and every other line of the
// report is what it always was. The two renders are compared directly, so the
// only difference the marker may make is the line that declares it.
func TestEmbarkUnmarkedProvenanceReadsAsBefore(t *testing.T) {
	source := embarkableSourceFixture(t)
	target := t.TempDir()

	marked := packSource(t, source)
	markedPlan, err := EmbarkProbe(marked, target)
	if err != nil {
		t.Fatalf("EmbarkProbe (marked): %v", err)
	}

	unmarked := packSource(t, source)
	stripPassBExemption(t, unmarked)
	plan, err := EmbarkProbe(unmarked, target)
	if err != nil {
		t.Fatalf("EmbarkProbe (unmarked): %v", err)
	}
	if plan.Coverage != nil && plan.Coverage.PassBExemption != nil {
		t.Fatalf("an unmarked lifeboat read as exempt: %+v", plan.Coverage.PassBExemption)
	}
	out := plan.Render()
	if strings.Contains(out, "pass B") {
		t.Errorf("an unmarked lifeboat's report claims an exemption:\n%s", out)
	}

	// Every other line is unchanged. The lifeboat directory differs between the
	// two packs, so the paths are normalised before the comparison.
	want := strings.ReplaceAll(markedPlan.Render(), marked, "<lifeboat>")
	got := strings.ReplaceAll(out, unmarked, "<lifeboat>")
	var kept []string
	for _, line := range strings.Split(want, "\n") {
		if strings.Contains(line, "pass B") {
			// The declaration takes its separator blank line with it.
			kept = kept[:len(kept)-1]
			continue
		}
		kept = append(kept, line)
	}
	if strings.Join(kept, "\n") != got {
		t.Errorf("the marker changed more than the line that declares it:\ngot:\n%s\nwant:\n%s", got, strings.Join(kept, "\n"))
	}
}

// TestEmbarkSanitisesTheExemptionReason: pass_b_exemption.reason comes verbatim
// from the untrusted _provenance.json, which manifest verification deliberately
// excludes, so it is attacker-controlled in a lifeboat anyone can hand a user.
// Every other string in the handoff is sanitised where the handoff is BUILT
// (Question, Searched), not only where it is printed — because the handoff is
// also what `--json` emits, and a consumer piping that to a terminal never sees
// the render's own masking.
func TestEmbarkSanitisesTheExemptionReason(t *testing.T) {
	source := embarkableSourceFixture(t)
	lifeboat := packSource(t, source)

	provPath := filepath.Join(lifeboat, ProvenanceName)
	raw, err := os.ReadFile(provPath)
	if err != nil {
		t.Fatal(err)
	}
	var prov map[string]any
	if err := json.Unmarshal(raw, &prov); err != nil {
		t.Fatal(err)
	}
	prov["pass_b_exemption"] = map[string]any{
		"reason": "exempt\x9b31mEVIL‮​\x7f\nlifeboat verified",
	}
	tampered, err := json.Marshal(prov)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, provPath, tampered)

	plan, err := EmbarkProbe(lifeboat, t.TempDir())
	if err != nil {
		t.Fatalf("EmbarkProbe: %v", err)
	}
	if plan.Coverage == nil || plan.Coverage.PassBExemption == nil {
		t.Fatal("the handoff dropped the exemption")
	}
	got := plan.Coverage.PassBExemption.Reason
	for _, r := range got {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == 0x202e || r == 0x200b {
			t.Fatalf("the handoff carries an unmasked %U from an untrusted lifeboat: %q", r, got)
		}
	}
}
