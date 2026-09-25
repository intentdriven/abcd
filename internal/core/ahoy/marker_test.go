package ahoy

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkerInsertIntoAbsentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	wrote, ok := installMarkerFile(path)
	if !ok || !wrote {
		t.Fatalf("install into absent file: wrote=%v ok=%v", wrote, ok)
	}
	if classifyMarker(path) != markerCurrent {
		t.Errorf("state after install = %q, want current", classifyMarker(path))
	}
}

func TestMarkerInsertAfterFrontmatterAndH1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	original := "---\ntitle: x\n---\n# Heading\n\nbody text\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	wrote, ok := installMarkerFile(path)
	if !ok || !wrote {
		t.Fatalf("install: wrote=%v ok=%v", wrote, ok)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Block must land after the frontmatter close, before the heading, and
	// preserve the body.
	if !bytes.Contains(got, []byte("body text")) {
		t.Errorf("body text lost:\n%s", got)
	}
	fmClose := bytes.Index(got, []byte("---\n\n")) // frontmatter's closing fence
	blockAt := bytes.Index(got, markerBegin)
	headingAt := bytes.Index(got, []byte("# Heading"))
	if blockAt == -1 || fmClose == -1 || headingAt == -1 {
		t.Fatalf("missing landmark (block=%d fmClose=%d heading=%d):\n%s", blockAt, fmClose, headingAt, got)
	}
	if !(fmClose < blockAt && blockAt < headingAt) {
		t.Errorf("block not placed between frontmatter and heading:\n%s", got)
	}
	if classifyMarker(path) != markerCurrent {
		t.Errorf("state = %q, want current", classifyMarker(path))
	}
}

func TestMarkerInsertSkipsFencedH1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	// A '# ' shell comment inside a fenced snippet precedes the real H1. The
	// block must not split the fence; it belongs after the real heading.
	original := "```bash\n# install deps\nmake build\n```\n# Real Title\n\nbody text\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := installMarkerFile(path); !ok {
		t.Fatal("install failed")
	}
	got, _ := os.ReadFile(path)
	// The fenced snippet must stay contiguous (block did not land inside it).
	if !bytes.Contains(got, []byte("# install deps\nmake build")) {
		t.Errorf("marker split the fenced snippet:\n%s", got)
	}
	blockAt := bytes.Index(got, markerBegin)
	titleAt := bytes.Index(got, []byte("# Real Title"))
	fenceCommentAt := bytes.Index(got, []byte("# install deps"))
	if blockAt == -1 || titleAt == -1 {
		t.Fatalf("missing landmark (block=%d title=%d):\n%s", blockAt, titleAt, got)
	}
	if !(fenceCommentAt < blockAt && titleAt < blockAt) {
		t.Errorf("block not placed after the real H1:\n%s", got)
	}
	if classifyMarker(path) != markerCurrent {
		t.Errorf("state = %q, want current", classifyMarker(path))
	}
}

// The fence rule is CommonMark's (mdrecord.Mask): a fence closes only on a
// bare run of its own marker, at least as long as the opener. A three-backtick
// line quoted inside a four-backtick example does not close it, and a `# `
// line parked in an HTML comment is not the title. Read with a private toggle,
// the block was written inside the example, above the real H1
// (iss-2609250955219864).
func TestMarkerInsertFollowsTheCommonMarkFenceRule(t *testing.T) {
	for name, original := range map[string]string{
		"quoted fence":    "````markdown\n```sh\n# not the title\n```\n````\n\n# Real Title\n\nbody text\n",
		"tilde in ticks":  "```text\n~~~\n# not the title\n~~~\n```\n\n# Real Title\n\nbody text\n",
		"commented title": "<!--\n# not the title\n-->\n\n# Real Title\n\nbody text\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "CLAUDE.md")
			if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, ok := installMarkerFile(path); !ok {
				t.Fatal("install failed")
			}
			got, _ := os.ReadFile(path)
			blockAt := bytes.Index(got, markerBegin)
			titleAt := bytes.Index(got, []byte("# Real Title"))
			if blockAt == -1 || titleAt == -1 || blockAt < titleAt {
				t.Fatalf("the block belongs after the real H1 (block=%d title=%d):\n%s", blockAt, titleAt, got)
			}
			if !bytes.HasPrefix(got, []byte(strings.SplitN(original, "# Real Title", 2)[0])) {
				t.Errorf("the example above the title must be left whole:\n%s", got)
			}
		})
	}
}

func TestClassifySymlinkedMarkerIsNotResolvableGap(t *testing.T) {
	dir := t.TempDir()
	// docs.target=claude_md so detection checks only the symlinked CLAUDE.md.
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "config.json"),
		[]byte(`{"docs":{"target":"claude_md"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A symlinked CLAUDE.md whose target lacks the block: classifyMarker must
	// report it as a symlink (non-resolvable), not "missing", so detection does
	// not emit a resolvable gap that install can never close.
	real := filepath.Join(t.TempDir(), "real.md")
	if err := os.WriteFile(real, []byte("# Title\n\nno block here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "CLAUDE.md")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if got := classifyMarker(link); got != markerSymlink {
		t.Fatalf("classifyMarker on symlink = %q, want %q", got, markerSymlink)
	}
	// install refuses to write through the symlink, so the two must agree: no
	// resolvable gap paired with a silent no-op.
	if wrote, ok := installMarkerFile(link); wrote || ok {
		t.Fatalf("installMarkerFile through symlink: wrote=%v ok=%v, want false/false", wrote, ok)
	}
	// detectMarkerDrift must not emit an actionable (required+resolvable) gap.
	for _, g := range detectMarkerDrift(dir) {
		if g.Required && g.Resolvable {
			t.Errorf("symlinked marker produced an actionable gap: %+v", g)
		}
	}
}

func TestMarkerInstallIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(path, []byte("# Title\n\nprose\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := installMarkerFile(path); !ok {
		t.Fatal("first install failed")
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wrote, ok := installMarkerFile(path)
	if !ok {
		t.Fatal("second install failed")
	}
	if wrote {
		t.Errorf("second install rewrote a current block (not byte-stable)")
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("marker install not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestMarkerOutdatedBlockIsRewritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	stale := "# Title\n\n<!-- BEGIN ABCD -->\nOLD CONTENT\n<!-- END ABCD -->\n\nafter\n"
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if classifyMarker(path) != markerOutdated {
		t.Fatalf("precondition: expected outdated, got %q", classifyMarker(path))
	}
	wrote, ok := installMarkerFile(path)
	if !ok || !wrote {
		t.Fatalf("rewrite: wrote=%v ok=%v", wrote, ok)
	}
	if classifyMarker(path) != markerCurrent {
		t.Errorf("state after rewrite = %q, want current", classifyMarker(path))
	}
	got, _ := os.ReadFile(path)
	if bytes.Contains(got, []byte("OLD CONTENT")) {
		t.Errorf("stale content survived rewrite:\n%s", got)
	}
	if !bytes.Contains(got, []byte("after")) {
		t.Errorf("trailing content lost:\n%s", got)
	}
}

func TestMarkerMultiBlockCollapsesToOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	dup := "# T\n\n<!-- BEGIN ABCD -->\na\n<!-- END ABCD -->\n\nmid\n\n<!-- BEGIN ABCD -->\nb\n<!-- END ABCD -->\n"
	if err := os.WriteFile(path, []byte(dup), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := installMarkerFile(path); !ok {
		t.Fatal("install failed")
	}
	got, _ := os.ReadFile(path)
	if n := bytes.Count(got, markerBegin); n != 1 {
		t.Errorf("expected exactly one block after collapse, got %d:\n%s", n, got)
	}
	if !bytes.Contains(got, []byte("mid")) {
		t.Errorf("inter-block content lost:\n%s", got)
	}
}

func TestMarkerRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	original := "# Title\n\nprose here\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := installMarkerFile(path); !ok {
		t.Fatal("install failed")
	}
	if _, ok := removeMarkerFile(path); !ok {
		t.Fatal("remove failed")
	}
	got, _ := os.ReadFile(path)
	if bytes.Contains(got, markerBegin) {
		t.Errorf("block survived removal:\n%s", got)
	}
	if !bytes.Equal(got, []byte(original)) {
		t.Errorf("round-trip not byte-identical:\nwant:\n%q\ngot:\n%q", original, got)
	}
}

func TestMarkerCRLFPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	crlf := "# Title\r\n\r\nprose\r\n"
	if err := os.WriteFile(path, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := installMarkerFile(path); !ok {
		t.Fatal("install failed")
	}
	got, _ := os.ReadFile(path)
	if bytes.Contains(got, []byte("\n")) && !bytes.Contains(got, []byte("\r\n")) {
		t.Errorf("CRLF flavour lost")
	}
	// The block itself must be CRLF so the file has no mixed EOLs.
	if bytes.Contains(got, append([]byte("<!-- BEGIN ABCD -->"), '\n')) &&
		!bytes.Contains(got, append([]byte("<!-- BEGIN ABCD -->"), []byte("\r\n")...)) {
		t.Errorf("block wrapped with LF in a CRLF file:\n%q", got)
	}
	if classifyMarker(path) != markerCurrent {
		t.Errorf("state = %q, want current", classifyMarker(path))
	}
}

// TestStripMarkerBlock covers the pure exported strip used by the lifeboat
// packer: a balanced block is removed and reported, content without one is
// returned unchanged, and an unbalanced fence (or a literal delimiter mention
// outside a pair) is left intact.
func TestStripMarkerBlock(t *testing.T) {
	withBlock := []byte("# Title\n\n<!-- BEGIN ABCD -->\nloader\n<!-- END ABCD -->\n\nBody.\n")
	out, changed := StripMarkerBlock(withBlock)
	if !changed {
		t.Fatal("expected a balanced block to be stripped")
	}
	if bytes.Contains(out, []byte("BEGIN ABCD")) || bytes.Contains(out, []byte("loader")) {
		t.Errorf("block not fully removed:\n%s", out)
	}
	if !bytes.Contains(out, []byte("# Title")) || !bytes.Contains(out, []byte("Body.")) {
		t.Errorf("strip damaged surrounding content:\n%s", out)
	}

	clean := []byte("# Title\n\nNo markers here.\n")
	out2, changed2 := StripMarkerBlock(clean)
	if changed2 {
		t.Error("clean content reported as changed")
	}
	if !bytes.Equal(out2, clean) {
		t.Error("clean content was modified")
	}

	// An unbalanced fence is not a block: leave it intact.
	unbalanced := []byte("# Title\n\n<!-- BEGIN ABCD -->\nno end fence\n")
	out3, changed3 := StripMarkerBlock(unbalanced)
	if changed3 || !bytes.Equal(out3, unbalanced) {
		t.Errorf("unbalanced fence must be left intact, got changed=%v:\n%s", changed3, out3)
	}
}

// TestDefaultAdoptionWritesNoNameIntoConventionsFiles pins the 2026-09-11 ruling
// (iss-2609110944498549): an adoption with default options leaves the repository's
// committed conventions files as the repository wrote them. The managed block names
// the tool and documents its internals, so it is planted only where the adopting
// project chose a docs target; the default target is skip. The adopted repo must
// still classify as managed, on its registry entry rather than on a marker.
func TestDefaultAdoptionWritesNoNameIntoConventionsFiles(t *testing.T) {
	setupHermetic(t)
	repo := committedRepo(t)
	claude := filepath.Join(repo, "CLAUDE.md")
	own := []byte("# Project\n\nOur own notes.\n")
	if err := os.WriteFile(claude, own, 0o644); err != nil {
		t.Fatal(err)
	}

	// The preview must not promise a block the default will not plant.
	pre, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(pre.Gaps, "marker.missing") {
		t.Errorf("dry-run on a fresh repo reports marker.missing, but the default docs target plants no block: %+v", pre.Gaps)
	}

	opts := installOpts()
	delete(opts.ValueOverrides, "docs_target") // the default, exactly as a user who never passed the flag gets it
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("default adoption status = %q (remaining %v, notes %v), want clean", res.Status, res.Remaining, res.Notes)
	}

	got, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, own) {
		t.Errorf("default adoption rewrote CLAUDE.md:\n%s", got)
	}
	if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf("default adoption created AGENTS.md (err=%v)", err)
	}

	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := stringVal(subMap(cfg, "docs"), "target"); v != "skip" {
		t.Errorf("persisted docs.target = %q, want skip", v)
	}

	post, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if post.FolderKind != ManagedRepo {
		t.Errorf("after a default adoption the repo classifies as %q, want %q", post.FolderKind, ManagedRepo)
	}
	if hasGap(post.Gaps, "marker.missing") {
		t.Errorf("after a default adoption detection still asks for a marker block: %+v", post.Gaps)
	}
}

// TestFirstInstallPlantsTheBlockWhereTheProjectChoseIt is the other half of the
// skip default: a project that names a docs target on its first install gets the
// block there. At the default, detection previews no marker gap, so the
// plugin-owned category is never offered — the chosen target is the approval, or
// the first install would persist the target and plant nothing.
func TestFirstInstallPlantsTheBlockWhereTheProjectChoseIt(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   []string
		absent []string
	}{
		{"both", []string{"CLAUDE.md", "AGENTS.md"}, nil},
		{"claude_md", []string{"CLAUDE.md"}, []string{"AGENTS.md"}},
		{"agents_md", []string{"AGENTS.md"}, []string{"CLAUDE.md"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			setupHermetic(t)
			repo := committedRepo(t)
			opts := installOpts()
			opts.ValueOverrides["docs_target"] = tc.target
			res, err := Install(repo, opts, RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != "clean" {
				t.Fatalf("status = %q (remaining %v, notes %v), want clean", res.Status, res.Remaining, res.Notes)
			}
			for _, name := range tc.want {
				if got := classifyMarker(filepath.Join(repo, name)); got != markerCurrent {
					t.Errorf("%s marker = %q, want current", name, got)
				}
			}
			for _, name := range tc.absent {
				if _, err := os.Lstat(filepath.Join(repo, name)); !os.IsNotExist(err) {
					t.Errorf("%s written for docs target %s (err=%v)", name, tc.target, err)
				}
			}
		})
	}
}
