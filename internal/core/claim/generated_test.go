package claim

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Generated files are shared, and one person's own overlapping reserve warns
// (spc-2610161044461448 step 11; iss-2610152125101468): a path two footprints
// share is no overlap when it is a file origin's default branch declares
// linguist-generated in its root .gitattributes, and a person's own
// reservation over their other copy's goes ahead and is told, where a drain's
// still backs off.

// declare commits attrs as the root .gitattributes (none when "") and each
// file on origin's default branch, and pushes it.
func (w *world) declare(attrs string, files ...string) {
	w.t.Helper()
	if attrs != "" {
		w.seed.Write(".gitattributes", attrs)
	}
	for _, f := range files {
		w.seed.Write(f, "generated or not\n")
	}
	w.seed.Commit("declare")
	w.seed.Git("push", "-q", "origin", "main")
}

// wantBackOff fails unless err is the back-off naming path in id's footprint
// and nothing of want reached origin.
func wantBackOff(t *testing.T, w *world, err error, want, path, id string) {
	t.Helper()
	if !errors.Is(err, ErrBackOff) || !strings.HasPrefix(err.Error(), `"`+path+`" is in the footprint of `+id+",") {
		t.Fatalf("reserving %s = %v; want the back-off naming %q in %s's footprint", want, err, path, id)
	}
	if tip := w.remoteTip(want); tip != "" {
		t.Fatalf("%s reached origin over the overlap: %s", want, tip)
	}
}

// TestAGeneratedFileIsNoOverlap: a declared generated file in two people's
// footprints is no overlap, nor is a directory footprint against the
// generated file inside it; a file not declared is still one, and so are two
// directory footprints.
func TestAGeneratedFileIsNoOverlap(t *testing.T) {
	w := newWorld(t)
	w.declare("docs/ref/commands.md linguist-generated\n", "docs/ref/commands.md", "docs/ref/intro.md", "a.go")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	m := w.clone("77+mira@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "docs/ref/commands.md", "a.go"); err != nil {
		t.Fatal(err)
	}
	if res, err := q.reserve("iss-2", "docs/ref/commands.md", "b.go"); err != nil || res.Act != ActReserved {
		t.Fatalf("the generated file in a second footprint = %+v, %v; want it reserved", res, err)
	}
	if res, err := m.reserve("iss-3", "docs/ref"); err != nil || res.Act != ActReserved {
		t.Fatalf("a directory over the generated file in two footprints = %+v, %v; want it reserved", res, err)
	}
	_, err := m.reserve("iss-4", "a.go")
	wantBackOff(t, w, err, "iss-4", "a.go", "iss-1")
	_, err = q.reserve("iss-5", "docs")
	wantBackOff(t, w, err, "iss-5", "docs/ref", "iss-3")
	_, err = q.reserve("iss-6", "docs/ref/intro.md")
	wantBackOff(t, w, err, "iss-6", "docs/ref/intro.md", "iss-3")
}

// TestADirectoryIsNeverGenerated: a directory is shared as a directory even
// when every file under it is declared generated and a pattern names the
// directory's own path; a generated file inside it is still shared.
func TestADirectoryIsNeverGenerated(t *testing.T) {
	w := newWorld(t)
	w.declare("docs/ref/** linguist-generated\ndocs/* linguist-generated\n", "docs/ref/a.md", "docs/ref/b.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "docs/ref"); err != nil {
		t.Fatal(err)
	}
	_, err := q.reserve("iss-2", "docs/ref")
	wantBackOff(t, w, err, "iss-2", "docs/ref", "iss-1")
	if _, err := q.reserve("iss-3", "docs/ref/a.md"); err != nil {
		t.Fatalf("a generated file inside a reserved directory: %v", err)
	}
	// A file the declaration matches but origin's default branch does not
	// hold is not known to be a file, so it is shared as itself.
	_, err = q.reserve("iss-4", "docs/ref/new.md")
	wantBackOff(t, w, err, "iss-4", "docs/ref/new.md", "iss-1")
}

// TestTheDeclarationIsReadFromOriginsDefaultBranch: a .gitattributes only in
// this copy, committed here or pushed to another branch of origin, declares
// nothing; once origin's default branch declares it, the next act shares it.
func TestTheDeclarationIsReadFromOriginsDefaultBranch(t *testing.T) {
	w := newWorld(t)
	w.declare("", "gen.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "gen.md"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(q.dir, ".gitattributes"), "gen.md linguist-generated\n")
	q.git("add", ".gitattributes")
	q.git("commit", "-q", "-m", "declare here only")
	q.git("push", "-q", "origin", "HEAD:refs/heads/elsewhere")
	_, err := q.reserve("iss-2", "gen.md")
	wantBackOff(t, w, err, "iss-2", "gen.md", "iss-1")
	w.declare("gen.md linguist-generated\n")
	if res, err := q.reserve("iss-2", "gen.md"); err != nil || res.GeneratedUnread != "" {
		t.Fatalf("the file once origin's default branch declares it = %+v, %v; want it reserved", res, err)
	}
}

// TestAnUnreadableDeclarationIsStrictAndSaysWhy: a declaration past the cap,
// or a .gitattributes that is not a regular file, recognises no generated
// file, so the overlap backs off, and both the back-off and a reservation
// that overlaps nothing say why.
func TestAnUnreadableDeclarationIsStrictAndSaysWhy(t *testing.T) {
	big := "gen.md linguist-generated\n#" + strings.Repeat("x", attributesCap) + "\n"
	for name, c := range map[string]struct {
		write func(w *world)
		why   string
	}{
		"past the cap": {func(w *world) {
			w.declare(big, "gen.md")
		}, fmt.Sprintf(".gitattributes on origin/main is %d bytes, past the 256 KiB abcd reads", len(big))},
		"not a regular file": {func(w *world) {
			w.seed.Write(".gitattributes/inner", "gen.md linguist-generated\n")
			w.declare("", "gen.md")
		}, ".gitattributes on origin/main is not a regular file"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			c.write(w)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			q := w.clone("99+quinn@users.noreply.github.com", "")
			if _, err := p.reserve("iss-1", "gen.md"); err != nil {
				t.Fatal(err)
			}
			_, err := q.reserve("iss-2", "gen.md")
			wantBackOff(t, w, err, "iss-2", "gen.md", "iss-1")
			if !strings.Contains(err.Error(), "(generated files were not recognised: "+c.why+")") {
				t.Fatalf("the back-off = %v; want it to say why no generated file was recognised: %s", err, c.why)
			}
			res, err := q.reserve("iss-3", "other.go")
			if err != nil || res.GeneratedUnread != c.why {
				t.Fatalf("a reservation overlapping nothing = %+v, %v; want it made, saying %q", res, err, c.why)
			}
		})
	}
}

// TestOnesOwnReserveOverlappingOnesOtherCopyWarns: a person's own reservation
// (Request.WarnOwnOverlap, the reserve verb's) sharing a path with their live
// reservation from another copy is made, and names the path and the record; a
// drain's reservation over the same path still backs off, so does an act that
// does not ask for the warning, and so does another person's.
func TestOnesOwnReserveOverlappingOnesOtherCopyWarns(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "internal/core/claim"); err != nil {
		t.Fatal(err)
	}
	res, err := laptop.do(Request{Record: "iss-5", Footprint: []string{"internal/core/claim/view.go", "x.go"}, WarnOwnOverlap: true})
	want := []OwnOverlap{{Path: "internal/core/claim/view.go", Record: "iss-1", Expiry: t0.Add(2 * time.Hour)}}
	if err != nil || res.Act != ActReserved || !slices.Equal(res.OwnOverlap, want) {
		t.Fatalf("one's own reserve over one's other copy = %+v, %v; want it reserved, naming %+v", res, err, want)
	}
	if w.remoteTip("iss-5") == "" {
		t.Fatal("the reservation that warned did not reach origin")
	}
	if got := res.OwnOverlap[0].Warning(); got != `"internal/core/claim/view.go" is also in the footprint of iss-1, which you hold from your other copy until 2026-10-03T16:00:00Z; reserved anyway, since both are yours` {
		t.Fatalf("the warning = %q", got)
	}
	_, err = laptop.do(Request{Record: "iss-6", Footprint: []string{"internal/core/claim/claim.go"}, Drain: true})
	wantBackOff(t, w, err, "iss-6", "internal/core/claim/claim.go", "iss-1")
	_, err = q.reserve("iss-7", "internal/core/claim/claim.go")
	wantBackOff(t, w, err, "iss-7", "internal/core/claim/claim.go", "iss-1")
	// An act that does not ask for the warning (a build's) backs off it too.
	_, err = laptop.reserve("iss-9", "internal/core/claim/crowd.go")
	wantBackOff(t, w, err, "iss-9", "internal/core/claim/crowd.go", "iss-1")
	// A reservation that shares nothing names nothing.
	if res, err := laptop.do(Request{Record: "iss-8", Footprint: []string{"y.go"}, WarnOwnOverlap: true}); err != nil || res.OwnOverlap != nil {
		t.Fatalf("a reserve sharing nothing = %+v, %v; want no overlap named", res, err)
	}
}

// TestContentionAgreesWithCrowdingOnGeneratedFiles: the drain's pass-over,
// read from the view alone, shares a generated file exactly as the
// reservation act does.
func TestContentionAgreesWithCrowdingOnGeneratedFiles(t *testing.T) {
	w := newWorld(t)
	w.declare("commands.md linguist-generated\n", "docs/commands.md", "a.go")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.reserve("iss-1", "docs/commands.md", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(q.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Generated.Is("docs/commands.md") || snap.Generated.Is("docs") || snap.Generated.Is("a.go") || snap.GeneratedUnread != "" {
		t.Fatalf("the view's generated files = %+v (%q); want docs/commands.md alone", snap.Generated, snap.GeneratedUnread)
	}
	for _, c := range []struct {
		id string
		fp []string
	}{
		{"iss-2", []string{"docs/commands.md"}},
		{"iss-3", []string{"docs"}},
		{"iss-4", []string{"a.go"}},
	} {
		why := snap.Contention(c.id, c.fp, w.clock())
		_, err := q.reserve(c.id, c.fp...)
		if (why == "") != (err == nil) {
			t.Fatalf("%s %v: Contention = %q, but the reservation act = %v", c.id, c.fp, why, err)
		}
	}
	if why := snap.Contention("iss-4", []string{"a.go"}, w.clock()); !strings.HasPrefix(why, `"a.go" is in the footprint of iss-1`) {
		t.Fatalf("Contention over a.go = %q", why)
	}
}

// wantGenerated fails unless GeneratedFiles in dir answers want for each path,
// with why as its reason.
func wantGenerated(t *testing.T, dir string, want map[string]bool, why string) {
	t.Helper()
	g, got := GeneratedFiles(dir)
	if got != why {
		t.Fatalf("GeneratedFiles reason = %q; want %q", got, why)
	}
	for p, w := range want {
		if g.Is(p) != w {
			t.Errorf("Is(%q) = %v; want %v", p, !w, w)
		}
	}
}

// TestAFileUnderANestedDeclarationIsNeverGenerated: only the root
// .gitattributes is read, so a file at or below a directory holding its own
// .gitattributes, which could unset the attribute, is never generated.
func TestAFileUnderANestedDeclarationIsNeverGenerated(t *testing.T) {
	w := newWorld(t)
	w.seed.Write("docs/guides/.gitattributes", "* -linguist-generated\n")
	w.declare("*.md linguist-generated\n", "docs/a.md", "docs/guides/intro.md", "docs/guides/deep/x.md", "docs/guidesx.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"docs/a.md": true, "docs/guidesx.md": true, "README.md": true,
		"docs/guides/intro.md": false, "docs/guides/deep/x.md": false}, "")
}

// TestANestedDeclarationIsFoundWhateverItsCapitals: git on a case-insensitive
// checkout reads a nested .gitattributes spelled in other capitals, so it
// leaves the files below it out too.
func TestANestedDeclarationIsFoundWhateverItsCapitals(t *testing.T) {
	w := newWorld(t)
	w.seed.Write("docs/.GITATTRIBUTES", "* -linguist-generated\n")
	w.declare("*.md linguist-generated\n", "docs/a.md", "other/b.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"docs/a.md": false, "other/b.md": true, "README.md": true}, "")
}

// TestADeclarationHoldingANULDeclaresNothing: git stops reading an attributes
// file at a NUL byte, so a declaration holding one declares nothing, and says
// why.
func TestADeclarationHoldingANULDeclaresNothing(t *testing.T) {
	w := newWorld(t)
	w.declare("*.md linguist-generated\nREADME.md -linguist-generated\x00\n", "a.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"a.md": false, "README.md": false}, ".gitattributes on origin/main holds a NUL byte")
}

// TestOnlyARegularFileIsGenerated: a symbolic link the declaration matches is
// not generated; a regular file, executable or not, is.
func TestOnlyARegularFileIsGenerated(t *testing.T) {
	w := newWorld(t)
	w.seed.Write("tool.md", "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(w.seed.Root(), "tool.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.md", filepath.Join(w.seed.Root(), "link.md")); err != nil {
		t.Fatal(err)
	}
	w.declare("*.md linguist-generated\n", "a.md")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"a.md": true, "tool.md": true, "link.md": false}, "")
}

// TestADeclarationPastTheTotalBudgetDeclaresNothing: the matching over every
// file together is bounded too (generatedSteps); past it no file is generated,
// and the reason says so.
func TestADeclarationPastTheTotalBudgetDeclaresNothing(t *testing.T) {
	w := newWorld(t)
	var files []string
	for i := range 40 {
		files = append(files, fmt.Sprintf("gen/f%d.md", i))
	}
	w.declare("*.md linguist-generated\n", files...)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"gen/f1.md": true}, "")
	defer func(was int) { generatedSteps = was }(generatedSteps)
	generatedSteps = 60
	wantGenerated(t, p.dir, map[string]bool{"gen/f1.md": false, "gen/f39.md": false},
		".gitattributes on origin/main takes too long to match against the files there")
}

// blobEntry is one ls-tree -r -z entry for a regular file at p.
func blobEntry(p string) string {
	return "100644 blob 0123456789abcdef0123456789abcdef01234567\t" + p
}

// TestANestedDeclarationIsLookedUpWhateverTheCapitals: a directory holding a
// .gitattributes is the same directory to git on a case-insensitive checkout
// whatever the capitals of its name or a file's path under it.
func TestANestedDeclarationIsLookedUpWhateverTheCapitals(t *testing.T) {
	got, _ := candidatesOf([]string{blobEntry("Docs/.gitattributes"), blobEntry("docs/x.gen"), blobEntry("DOCS/deep/y.gen"), blobEntry("other/z.gen")})
	if !slices.Equal(got, []string{"other/z.gen"}) {
		t.Fatalf("candidates = %v; want other/z.gen alone", got)
	}
}

// TestAVeryDeepPathIsJudgedQuickly: looking a path's directories up among
// the nested declarations reads the path once, however deep it is.
func TestAVeryDeepPathIsJudgedQuickly(t *testing.T) {
	deep := strings.Repeat("d/", 2049)
	entries := []string{blobEntry("n/.gitattributes")}
	for i := range 500 {
		entries = append(entries, blobEntry(fmt.Sprintf("%sf%d", deep, i)))
	}
	done := make(chan int, 1)
	start := time.Now()
	go func() {
		got, _ := candidatesOf(entries)
		done <- len(got)
	}()
	select {
	case n := <-done:
		if n != 500 {
			t.Fatalf("candidates = %d; want the 500 deep files", n)
		}
		if took := time.Since(start); took > 2*time.Second {
			t.Fatalf("500 paths 2,049 directories deep took %s; want well under a second", took)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("500 paths 2,049 directories deep did not finish in 10s")
	}
}

// treeOnMain commits each path with body as a blob straight into the index of
// origin's default branch, whatever this file system makes of the name, and
// pushes it.
func (w *world) treeOnMain(files map[string]string) {
	w.t.Helper()
	for p, body := range files {
		cmd := exec.Command("git", "-C", w.seed.Root(), "hash-object", "-w", "--stdin")
		cmd.Env = w.seed.Env()
		cmd.Stdin = strings.NewReader(body)
		out, err := cmd.Output()
		if err != nil {
			w.t.Fatal(err)
		}
		w.seed.Git("update-index", "--add", "--cacheinfo", "100644,"+strings.TrimSpace(string(out))+","+p)
	}
	w.seed.Git("commit", "-q", "-m", "tree")
	w.seed.Git("push", "-q", "origin", "main")
}

// TestOnlyAPlainDeclarationIsShared: abcd shares a generated file only on a
// plain declaration; otherwise it declares nothing and names why.
func TestOnlyAPlainDeclarationIsShared(t *testing.T) {
	const decl = "*.md linguist-generated\n"
	for name, c := range map[string]struct {
		files map[string]string
		why   string
	}{
		"a byte outside ASCII in the declaration": {map[string]string{".gitattributes": decl + "# r\xc3\xa9sum\xc3\xa9\n"},
			".gitattributes on origin/main holds a byte outside ASCII"},
		"a quoted pattern": {map[string]string{".gitattributes": decl + "\"b.md\" text\n"},
			".gitattributes on origin/main quotes a pattern"},
		"a quoted macro": {map[string]string{".gitattributes": decl + "\"[attr]x\" -linguist-generated\nb.md x\n"},
			".gitattributes on origin/main quotes a pattern"},
		"a macro": {map[string]string{".gitattributes": decl + "[attr]x text\n"},
			".gitattributes on origin/main names a macro"},
		"a bare [attr]": {map[string]string{".gitattributes": decl + "[attr] text\n"},
			".gitattributes on origin/main names a macro"},
		"a .gitattributes named outside ASCII": {map[string]string{".gitattributes": decl, "d/.gitattribute\xc5\xbf": "* -linguist-generated\n"},
			"origin/main holds a .gitattributes named outside ASCII"},
		"a root .gitattributes named outside ASCII": {map[string]string{".gitattributes": decl, ".gitattribute\xc5\xbf": "* -linguist-generated\n"},
			"origin/main holds a .gitattributes named outside ASCII"},
		"a .gitattributes under a directory named outside ASCII": {map[string]string{".gitattributes": decl, "\xc3\x89/.gitattributes": "* -linguist-generated\n", "\xc3\xa9/a.md": "x\n"},
			"origin/main holds a .gitattributes in a directory named outside ASCII"},
		"a second root .gitattributes in other capitals": {map[string]string{".gitattributes": decl, ".GITATTRIBUTES": "* -linguist-generated\n"},
			"origin/main holds a second root .gitattributes in other capitals"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			c.files["a.md"] = "x\n"
			w.treeOnMain(c.files)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			wantGenerated(t, p.dir, map[string]bool{"a.md": false, "README.md": false}, c.why)
		})
	}
}

// TestAFileNamedOutsideASCIIIsNeverGenerated: a file whose path holds a byte
// outside ASCII is never generated; the rest still are.
func TestAFileNamedOutsideASCIIIsNeverGenerated(t *testing.T) {
	w := newWorld(t)
	w.treeOnMain(map[string]string{".gitattributes": "*.md linguist-generated\n", "a.md": "x\n", "r\xc3\xa9sum\xc3\xa9.md": "x\n", "\xc3\xa9/b.md": "x\n"})
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wantGenerated(t, p.dir, map[string]bool{"a.md": true, "r\xc3\xa9sum\xc3\xa9.md": false, "\xc3\xa9/b.md": false}, "")
}
