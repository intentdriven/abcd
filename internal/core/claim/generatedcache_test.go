package claim

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The generated set is kept between reads (iss-2610170833428011): a read
// under the key of the last one takes the kept answer and neither lists nor
// matches again; a move of origin's default branch to another tree, or a
// change to the matching budget, reads afresh; and a kept file that is not
// an answer for the key in hand, or is not a plain untracked regular file
// within the cap, reads as nothing kept.

// countMatches counts the matches generatedFilesAt makes from here on.
func countMatches(t *testing.T) *int {
	t.Helper()
	n := new(int)
	was := generatedMatch
	generatedMatch = func(rs attrRules, paths []string) (map[string]bool, bool) {
		*n++
		return was(rs, paths)
	}
	t.Cleanup(func() { generatedMatch = was })
	return n
}

// keptPath is the kept file in the checkout at dir.
func keptPath(dir string) string {
	return filepath.Join(dir, filepath.FromSlash(generatedKeptRel))
}

// keyOn is the key the clone p's read of origin's default branch keeps its
// answer under.
func keyOn(t *testing.T, p *participant) string {
	t.Helper()
	tree := strings.TrimSpace(p.git("rev-parse", "refs/remotes/origin/main^{tree}"))
	key := generatedKey("refs/remotes/origin/main", tree)
	if key == "" {
		t.Fatal("the running binary cannot be named, so nothing is kept; the test proves nothing")
	}
	return key
}

// generatedWorld is a clone of a world whose default branch declares *.gen
// generated and holds a.gen, b.gen and c.txt.
func generatedWorld(t *testing.T) *participant {
	t.Helper()
	w := newWorld(t)
	w.declare("*.gen linguist-generated\n", "a.gen", "b.gen", "c.txt")
	return w.clone("1234+octo@users.noreply.github.com", "")
}

// TestASecondReadTakesTheKeptSet: the second read neither matches nor lists
// the tree again, and answers what the first did; the kept file is the
// owner's alone.
func TestASecondReadTakesTheKeptSet(t *testing.T) {
	p := generatedWorld(t)
	n := countMatches(t)
	want := map[string]bool{"a.gen": true, "b.gen": true, "c.txt": false, "README.md": false}
	wantGenerated(t, p.dir, want, "")
	if *n != 1 {
		t.Fatalf("the first read matched %d time(s); want 1", *n)
	}
	fi, err := os.Lstat(keptPath(p.dir))
	if err != nil {
		t.Fatalf("nothing was kept: %v", err)
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
		t.Fatalf("the kept file is %v; want a regular file at 0600", fi.Mode())
	}
	log := traced(t)
	wantGenerated(t, p.dir, want, "")
	if *n != 1 {
		t.Fatalf("the second read matched again (%d matches); want the kept set taken", *n)
	}
	for _, cmd := range log.cmds {
		if len(cmd) > 1 && cmd[0] == "ls-tree" && cmd[1] == "-r" {
			t.Fatalf("the second read listed the tree again: %q", cmd)
		}
	}
}

// TestASecondViewDoesNotMatchAgain: a view with a live footprint matches
// once, and the next view on the same default branch takes the kept set.
func TestASecondViewDoesNotMatchAgain(t *testing.T) {
	p := generatedWorld(t)
	if _, err := p.reserve("iss-1", "a.gen", "c.txt"); err != nil {
		t.Fatal(err)
	}
	dropKeptGenerated(t, p.dir)
	n := countMatches(t)
	for i := range 2 {
		snap, err := View(p.dir, p.w.clock())
		if err != nil {
			t.Fatal(err)
		}
		if !snap.Generated.Is("a.gen") || snap.Generated.Is("c.txt") || snap.GeneratedUnread != "" {
			t.Fatalf("view %d: generated a.gen %v, c.txt %v, unread %q; want a.gen alone",
				i+1, snap.Generated.Is("a.gen"), snap.Generated.Is("c.txt"), snap.GeneratedUnread)
		}
		if *n != 1 {
			t.Fatalf("after view %d the match ran %d time(s); want once", i+1, *n)
		}
	}
}

// TestAMoveOfTheDefaultBranchReadsAfresh: once origin's default branch moves
// to another tree, the read matches again and answers the new declaration;
// a change to the matching budget reads afresh too.
func TestAMoveOfTheDefaultBranchReadsAfresh(t *testing.T) {
	w := newWorld(t)
	w.declare("*.gen linguist-generated\n", "a.gen", "c.txt")
	p := w.clone("1234+octo@users.noreply.github.com", "")
	n := countMatches(t)
	wantGenerated(t, p.dir, map[string]bool{"a.gen": true, "c.txt": false}, "")
	w.declare("*.txt linguist-generated\n")
	p.git("fetch", "-q", "origin")
	wantGenerated(t, p.dir, map[string]bool{"a.gen": false, "c.txt": true}, "")
	if *n != 2 {
		t.Fatalf("matched %d time(s) over two trees; want 2", *n)
	}
	defer func(was int) { matchSteps = was }(matchSteps)
	matchSteps++
	wantGenerated(t, p.dir, map[string]bool{"a.gen": false, "c.txt": true}, "")
	if *n != 3 {
		t.Fatalf("matched %d time(s) after the budget changed; want 3", *n)
	}
}

// TestAMatchPastItsBudgetIsKept: a match that ran past its budget is as sure
// an answer for its key as a set, so it is kept and its reason given again
// without matching; the budget restored reads the set afresh.
func TestAMatchPastItsBudgetIsKept(t *testing.T) {
	p := generatedWorld(t)
	n := countMatches(t)
	was := generatedSteps
	defer func() { generatedSteps = was }()
	generatedSteps = 10
	const why = ".gitattributes on origin/main takes too long to match against the files there"
	for range 2 {
		wantGenerated(t, p.dir, map[string]bool{"a.gen": false}, why)
	}
	if *n != 1 {
		t.Fatalf("matched %d time(s) past the budget; want once", *n)
	}
	generatedSteps = was
	wantGenerated(t, p.dir, map[string]bool{"a.gen": true}, "")
	if *n != 2 {
		t.Fatalf("matched %d time(s) once the budget was restored; want 2", *n)
	}
}

// TestNothingIsKeptWithoutTheLocalTier: a checkout with no local tier keeps
// nothing and creates no tier, and so matches on every read.
func TestNothingIsKeptWithoutTheLocalTier(t *testing.T) {
	p := generatedWorld(t)
	tier := filepath.Join(p.dir, ".abcd")
	if err := os.RemoveAll(tier); err != nil {
		t.Fatal(err)
	}
	n := countMatches(t)
	for range 2 {
		wantGenerated(t, p.dir, map[string]bool{"a.gen": true}, "")
	}
	if *n != 2 {
		t.Fatalf("matched %d time(s) without a local tier; want 2", *n)
	}
	if _, err := os.Lstat(tier); !os.IsNotExist(err) {
		t.Fatalf("a read made %s: %v", tier, err)
	}
}

// TestAKeptFileThatIsNotAnAnswerIsReadAfresh: a kept file that is corrupt,
// past the cap, of another form, key or outcome, names a path no listing
// gives, is a symbolic link, or is tracked by git reads as nothing kept: the
// read matches afresh, answers what the tree declares, never the planted
// path, and keeps the right answer in its place.
func TestAKeptFileThatIsNotAnAnswerIsReadAfresh(t *testing.T) {
	planted := map[string]bool{"a.gen": true, "c.txt": true}
	for name, c := range map[string]struct {
		plant func(t *testing.T, p *participant, key string)
		// tracked is true when the plant makes git track the kept file,
		// which is then never rewritten.
		tracked bool
	}{
		"corrupt": {plant: func(t *testing.T, p *participant, key string) {
			writeFile(t, keptPath(p.dir), "not a kept set\x00")
		}},
		"truncated": {plant: func(t *testing.T, p *participant, key string) {
			data := encodeKept(key, planted, false)
			writeFile(t, keptPath(p.dir), string(data[:len(data)-1]))
		}},
		"past the cap": {plant: func(t *testing.T, p *participant, key string) {
			data := encodeKept(key, planted, false)
			pad := strings.Repeat("x", generatedKeptCap+1-len(data))
			writeFile(t, keptPath(p.dir), string(data)+pad)
		}},
		"another format": {plant: func(t *testing.T, p *participant, key string) {
			data := strings.Replace(string(encodeKept(key, planted, false)), "\x00"+generatedKeptFormat+"\x00", "\x009\x00", 1)
			writeFile(t, keptPath(p.dir), data)
		}},
		"another key": {plant: func(t *testing.T, p *participant, key string) {
			writeFile(t, keptPath(p.dir), string(encodeKept(strings.Repeat("0", len(key)), planted, false)))
		}},
		"an unknown outcome": {plant: func(t *testing.T, p *participant, key string) {
			data := strings.Replace(string(encodeKept(key, planted, false)), "\x00"+keptSet+"\x00", "\x00all\x00", 1)
			writeFile(t, keptPath(p.dir), data)
		}},
		"spent with paths": {plant: func(t *testing.T, p *participant, key string) {
			data := strings.Replace(string(encodeKept(key, planted, false)), "\x00"+keptSet+"\x00", "\x00"+keptSpent+"\x00", 1)
			writeFile(t, keptPath(p.dir), data)
		}},
		"a path outside the tree": {plant: func(t *testing.T, p *participant, key string) {
			writeFile(t, keptPath(p.dir), string(encodeKept(key, map[string]bool{"a.gen": true, "../c.txt": true}, false)))
		}},
		"an unclean path": {plant: func(t *testing.T, p *participant, key string) {
			writeFile(t, keptPath(p.dir), string(encodeKept(key, map[string]bool{"a.gen": true, "./c.txt": true}, false)))
		}},
		"a symbolic link": {plant: func(t *testing.T, p *participant, key string) {
			elsewhere := filepath.Join(t.TempDir(), "kept")
			writeFile(t, elsewhere, string(encodeKept(key, planted, false)))
			if err := os.Remove(keptPath(p.dir)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(elsewhere, keptPath(p.dir)); err != nil {
				t.Fatal(err)
			}
		}},
		"tracked by git": {tracked: true, plant: func(t *testing.T, p *participant, key string) {
			writeFile(t, keptPath(p.dir), string(encodeKept(key, planted, false)))
			p.git("add", "-f", generatedKeptRel)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			p := generatedWorld(t)
			want := map[string]bool{"a.gen": true, "b.gen": true, "c.txt": false}
			wantGenerated(t, p.dir, want, "")
			key := keyOn(t, p)
			c.plant(t, p, key)
			n := countMatches(t)
			wantGenerated(t, p.dir, want, "")
			if *n != 1 {
				t.Fatalf("matched %d time(s) over the planted file; want it read afresh once", *n)
			}
			fi, err := os.Lstat(keptPath(p.dir))
			if err != nil {
				t.Fatal(err)
			}
			if c.tracked {
				raw, err := os.ReadFile(keptPath(p.dir))
				if err != nil {
					t.Fatal(err)
				}
				if string(raw) != string(encodeKept(key, planted, false)) {
					t.Fatal("the tracked kept file was rewritten")
				}
				return
			}
			if !fi.Mode().IsRegular() {
				t.Fatalf("the kept file is %v after the read; want it replaced by a regular file", fi.Mode())
			}
			files, spent, ok := keptGenerated(viewGit{}, p.dir, key)
			if !ok || spent || !sameSet(files, want) {
				t.Fatalf("kept after the read = %v, spent %v, ok %v; want the tree's own answer", files, spent, ok)
			}
		})
	}
}

// sameSet reports whether got holds exactly the paths want marks true.
func sameSet(got, want map[string]bool) bool {
	var g, w []string
	for p, in := range got {
		if in {
			g = append(g, p)
		}
	}
	for p, in := range want {
		if in {
			w = append(w, p)
		}
	}
	slices.Sort(g)
	slices.Sort(w)
	return slices.Equal(g, w)
}

// TestTheKeptFileRoundTrips: what encodeKept writes decodeKept reads under
// the same key, an empty set and a spent outcome included, and under no
// other key.
func TestTheKeptFileRoundTrips(t *testing.T) {
	key := strings.Repeat("ab", 32)
	for _, c := range []struct {
		files map[string]bool
		spent bool
	}{
		{map[string]bool{"a.gen": true, "dir/b.gen": true}, false},
		{map[string]bool{}, false},
		{nil, true},
	} {
		data := encodeKept(key, c.files, c.spent)
		files, spent, ok := decodeKept(data, key)
		if !ok || spent != c.spent || !sameSet(files, c.files) {
			t.Errorf("round trip of %v (spent %v) = %v, %v, %v", c.files, c.spent, files, spent, ok)
		}
		if !c.spent && files == nil {
			t.Errorf("round trip of %v gave a nil set; want an empty one, as a match gives", c.files)
		}
		if _, _, ok := decodeKept(data, strings.Repeat("cd", 32)); ok {
			t.Errorf("the file for %v read under another key", c.files)
		}
	}
}
