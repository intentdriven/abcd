package relink

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decisions"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func move(t *testing.T, root, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, to)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
		t.Fatal(err)
	}
}

const (
	openA   = "specs/open/a.md"
	closedA = "specs/closed/a.md"
)

// A link to the moved file is repointed from every folder, and the moved file's
// own links, written from the folder it left, are re-relativised against the one
// it is in — while a link that never resolved, an external URL, an anchor-only
// link and an unrelated record are left exactly as written.
func TestRepointRewritesLinksToAndFromTheMovedFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, openA, "[b](b.md) [gone](nope.md) [self](a.md#x) [web](https://example.com/a.md) [top](#top)\n")
	write(t, root, "specs/open/b.md", "[a](a.md) [a again](./a.md#part)\n")
	// Two links name a file called a.md that is not the moved record: one that
	// exists in another folder and one that never resolved. Neither is the
	// move's, so both stay exactly as written.
	write(t, root, "specs/closed/c.md", "[a](../open/a.md) [x](../archive/a.md) [y](../gone/a.md)\n")
	write(t, root, "specs/archive/a.md", "an unrelated record with the same name\n")
	write(t, root, "docs/guide.md", "See [a](../specs/open/a.md \"title\").\n\n[a-ref]: ../specs/open/a.md\n\n```\n[fenced](../specs/open/a.md)\n```\n")
	write(t, root, "docs/other.md", "[b](../specs/open/b.md) mentions a.md only in prose\n")
	move(t, root, openA, closedA)

	got, err := Repoint(root, []Move{{From: openA, To: closedA, MovedNow: true}})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		closedA:             "[b](../open/b.md) [gone](nope.md) [self](a.md#x) [web](https://example.com/a.md) [top](#top)\n",
		"specs/open/b.md":   "[a](../closed/a.md) [a again](../closed/a.md#part)\n",
		"specs/closed/c.md": "[a](a.md) [x](../archive/a.md) [y](../gone/a.md)\n",
		"docs/guide.md":     "See [a](../specs/closed/a.md \"title\").\n\n[a-ref]: ../specs/closed/a.md\n\n```\n[fenced](../specs/closed/a.md)\n```\n",
		"docs/other.md":     "[b](../specs/open/b.md) mentions a.md only in prose\n",
	}
	for rel, w := range want {
		if g := read(t, root, rel); g != w {
			t.Errorf("%s:\n got %q\nwant %q", rel, g, w)
		}
	}

	wantRW := []Rewrite{
		{File: "docs/guide.md", Line: 1, From: "../specs/open/a.md", To: "../specs/closed/a.md"},
		{File: "docs/guide.md", Line: 3, From: "../specs/open/a.md", To: "../specs/closed/a.md"},
		{File: "docs/guide.md", Line: 6, From: "../specs/open/a.md", To: "../specs/closed/a.md"},
		{File: closedA, Line: 1, From: "b.md", To: "../open/b.md"},
		{File: "specs/closed/c.md", Line: 1, From: "../open/a.md", To: "a.md"},
		{File: "specs/open/b.md", Line: 1, From: "a.md", To: "../closed/a.md"},
		{File: "specs/open/b.md", Line: 1, From: "./a.md#part", To: "../closed/a.md#part"},
	}
	if !reflect.DeepEqual(got, wantRW) {
		t.Fatalf("rewrites:\n got %+v\nwant %+v", got, wantRW)
	}

	// A second call over the same moves finds nothing left to do.
	again, err := Repoint(root, []Move{{From: openA, To: closedA, MovedNow: true}})
	if err != nil || len(again) != 0 {
		t.Fatalf("a re-run must be a no-op: %+v, %v", again, err)
	}
}

// Two records moved in one operation: a link from one to the other follows both.
func TestRepointFollowsTwoMovesAtOnce(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/open/s.md", "[i](../../intents/planned/i.md)\n")
	write(t, root, "intents/planned/i.md", "[s](../../specs/open/s.md)\n")
	move(t, root, "specs/open/s.md", "specs/closed/s.md")
	move(t, root, "intents/planned/i.md", "intents/shipped/i.md")

	if _, err := Repoint(root, []Move{
		{From: "specs/open/s.md", To: "specs/closed/s.md", MovedNow: true},
		{From: "intents/planned/i.md", To: "intents/shipped/i.md", MovedNow: true},
	}); err != nil {
		t.Fatal(err)
	}
	if g := read(t, root, "specs/closed/s.md"); g != "[i](../../intents/shipped/i.md)\n" {
		t.Errorf("spec: %q", g)
	}
	if g := read(t, root, "intents/shipped/i.md"); g != "[s](../../specs/closed/s.md)\n" {
		t.Errorf("intent: %q", g)
	}
}

// A move an earlier call made (MovedNow false — a verb's re-run finding the
// record already in its new folder) still repoints every OTHER file's link to
// the old path, so a re-run finishes an interrupted repoint; but the moved
// file's own links are read from the folder it is in, because they may have
// been written there since. Re-reading them from the folder it left would
// resolve a bare link to a different file of the same name.
func TestRepointReadsAnEarlierMovesOwnLinksWhereTheFileIs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/open/README.md", "# open\n")
	write(t, root, "specs/closed/README.md", "# closed\n")
	own := "[index](README.md) [b](../open/b.md)\n"
	write(t, root, closedA, own)
	write(t, root, "specs/open/b.md", "[a](a.md)\n")

	got, err := Repoint(root, []Move{{From: openA, To: closedA}})
	if err != nil {
		t.Fatal(err)
	}
	if g := read(t, root, closedA); g != own {
		t.Errorf("the moved file's own links were re-read from the folder it left: %q", g)
	}
	if g := read(t, root, "specs/open/b.md"); g != "[a](../closed/a.md)\n" {
		t.Errorf("a stale link to the old path was not repointed: %q", g)
	}
	want := []Rewrite{{File: "specs/open/b.md", Line: 1, From: "a.md", To: "../closed/a.md"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrites:\n got %+v\nwant %+v", got, want)
	}
}

// The walk stays out of git's directory, the local tier and a nested checkout,
// whose files belong to another working tree, and writes nothing into the
// history no change may edit (a frozen decision log, the reviews folder); a
// decision record is relinked like any other record.
func TestRepointStaysOutOfForeignTreesAndAppendOnlyLogs(t *testing.T) {
	root := t.TempDir()
	write(t, root, openA, "a\n")
	link := "[a](../../specs/open/a.md)\n"
	write(t, root, ".abcd/.work.local/x.md", link)
	write(t, root, "nested/.git", "gitdir: elsewhere\n")
	write(t, root, "nested/sub/x.md", link)
	write(t, root, ".git/x/y.md", link)
	write(t, root, decisions.FrozenLog, link)
	write(t, root, ".abcd/work/decisions/dcn-2610100000000001-x.md", "[a](../../../specs/open/a.md)\n")
	write(t, root, ".abcd/work/reviews/2026-01-01-x/00-summary.md", link)
	move(t, root, openA, closedA)

	got, err := Repoint(root, []Move{{From: openA, To: closedA, MovedNow: true}})
	if err != nil {
		t.Fatal(err)
	}
	// A decision record is relinked like any other record (spc-2610110931381316,
	// decision 13); nothing else here may be rewritten.
	if len(got) != 1 || got[0].File != ".abcd/work/decisions/dcn-2610100000000001-x.md" {
		t.Fatalf("only the decision record may be rewritten: %+v", got)
	}
	for _, rel := range []string{".abcd/.work.local/x.md", "nested/sub/x.md", ".git/x/y.md",
		decisions.FrozenLog, ".abcd/work/reviews/2026-01-01-x/00-summary.md"} {
		if g := read(t, root, rel); g != link {
			t.Errorf("%s rewritten to %q", rel, g)
		}
	}
}

// A move the tree does not show — its source still present, or its destination
// absent — is ignored, and a path that is not clean and repo-relative is
// refused before anything is read.
func TestRepointIgnoresMovesThatDidNotHappenAndRefusesUnsafePaths(t *testing.T) {
	root := t.TempDir()
	write(t, root, openA, "a\n")
	write(t, root, "specs/open/b.md", "[a](a.md)\n")

	got, err := Repoint(root, []Move{{From: openA, To: closedA, MovedNow: true}})
	if err != nil || len(got) != 0 {
		t.Fatalf("a move that did not happen must rewrite nothing: %+v, %v", got, err)
	}
	for _, m := range []Move{{From: "../x.md", To: closedA}, {From: openA, To: "/abs.md"}, {From: "", To: closedA}} {
		if _, err := Repoint(root, []Move{m}); err == nil || !strings.Contains(err.Error(), "repo-relative") {
			t.Errorf("move %+v must be refused, got %v", m, err)
		}
	}
	if g := read(t, root, "specs/open/b.md"); g != "[a](a.md)\n" {
		t.Errorf("b.md rewritten: %q", g)
	}
}

// Rewritten says what Repoint writes over a file that did not itself move, whatever form
// the link takes and wherever the file is, and leaves what Repoint's walk does not reach.
func TestRewrittenIsWhatRepointWrites(t *testing.T) {
	const (
		from = ".abcd/development/intents/planned/itd-10-a.md"
		to   = ".abcd/development/intents/shipped/itd-10-a.md"
	)
	for name, c := range map[string]struct {
		rel, content string
		writes       bool
	}{
		"a link from outside the store":   {"docs/guide.md", "See [it](../" + from + "#f \"t\").\nMore.\n", true},
		"a reference definition":          {"docs/ref.md", "[it]: ../" + from + "\n", true},
		"a store's README":                {".abcd/development/intents/README.md", "The [it](planned/itd-10-a.md).\n", true},
		"a file beside the record":        {".abcd/development/intents/planned/notes.md", "See [it](itd-10-a.md).\n", true},
		"a file in the record's new home": {".abcd/development/intents/shipped/notes.md", "See [it](../planned/itd-10-a.md).\n", true},
		"a link not in its shortest form": {"docs/long.md", "See [it](../docs/../" + from + ").\n", true},
		"a link spelled from here":        {".abcd/development/intents/here.md", "See [it](./planned/itd-10-a.md).\n", true},
		"the path as plain text":          {"docs/plain.md", "planned/itd-10-a.md and [it](../" + from + ").\n", true},
		"only plain text":                 {"docs/text.md", "`planned/itd-10-a.md` by name.\n", false},
		"another record's link":           {"docs/other.md", "See [it](../.abcd/development/intents/planned/itd-11-b.md).\n", false},
		"a dependency tree":               {"node_modules/x/a.md", "See [it](../../" + from + ").\n", false},
		"the local tier":                  {".abcd/.work.local/a.md", "See [it](../development/intents/planned/itd-10-a.md).\n", false},
		"the reviews folder":              {".abcd/work/reviews/a.md", "See [it](../../development/intents/planned/itd-10-a.md).\n", false},
		"a file that is not markdown":     {"docs/a.txt", "See [it](../" + from + ").\n", false},
		"markdown in capitals":            {"docs/a.MD", "See [it](../" + from + ").\n", true},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, c.rel, c.content)
			write(t, root, to, "# moved\n")
			wrote, err := Repoint(root, []Move{{From: from, To: to}})
			if err != nil {
				t.Fatal(err)
			}
			if (len(wrote) > 0) != c.writes {
				t.Fatalf("premise: Repoint wrote %+v", wrote)
			}
			if got, want := Rewritten(c.rel, c.content, []Move{{From: from, To: to}}), read(t, root, c.rel); got != want {
				t.Fatalf("Rewritten = %q, Repoint wrote %q", got, want)
			}
		})
	}
}
