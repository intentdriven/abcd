package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// readmePart renders one part with LF line endings, for building fixtures.
func readmePart(name, body string) string {
	return string(renderReadmePart(name, body, []byte("\n")))
}

// handEdit returns a rendered part whose body was changed by hand: the fence
// still carries the hash of body, but the text between the fences is edited.
func handEdit(name, body, edited string) string {
	return strings.Replace(readmePart(name, body), "\n"+body+"\n", "\n"+edited+"\n", 1)
}

func writeReadme(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readReadme(t *testing.T, path string) string {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

// backdate sets path's modification time an hour into the past, so a write the
// run should not make would show as a changed time.
func backdate(t *testing.T, path string) time.Time {
	t.Helper()
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return old
}

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}

func notes(res readmePartsResult) []string {
	var out []string
	for _, p := range res.Parts {
		out = append(out, p.Note)
	}
	return out
}

func hasNote(res readmePartsResult, want string) bool {
	for _, n := range notes(res) {
		if n == want {
			return true
		}
	}
	return false
}

const licenceMIT = "Licensed under the MIT License; see [LICENSE](LICENSE)."

// TestReadmePartStates holds the six states the spec's table names, each read
// by the parser and then met by a write: absent is written, current is left
// with the file's bytes and modification time unchanged, stale is rewritten,
// a part edited by hand is named and left while the other parts are still
// written, and a duplicated or unclosed mark refuses the whole write and names
// the line. Removal (uninstall's file half) takes away only what still matches
// its hash.
func TestReadmePartStates(t *testing.T) {
	owner := "# Project\n\nThe owner's own words.\n"
	edited := handEdit("licence", licenceMIT, "Licensed under whatever I like.")
	cases := []struct {
		name    string
		content string
		fresh   string
		want    readmePartState
		lines   []int
	}{
		{"absent", owner, licenceMIT, readmePartAbsent, nil},
		{"current", "# Project\n\n" + readmePart("licence", licenceMIT) + "\n\nThe owner's own words.\n", licenceMIT, readmePartCurrent, nil},
		{"stale", "# Project\n\n" + readmePart("licence", "Licensed under the ISC License.") + "\n\nThe owner's own words.\n", licenceMIT, readmePartStale, nil},
		{"edited by hand", "# Project\n\n" + edited + "\n\nThe owner's own words.\n", licenceMIT, readmePartEdited, []int{3}},
		{"duplicated", "# Project\n\n" + readmePart("licence", licenceMIT) + "\n\nwords\n\n" + readmePart("licence", licenceMIT) + "\n", licenceMIT, readmePartDuplicated, []int{3, 9}},
		{"unclosed opening", "# Project\n\n" + strings.SplitN(readmePart("licence", licenceMIT), "\n", 2)[0] + "\nbody\n", licenceMIT, readmePartUnclosed, []int{3}},
		{"unclosed closing", "# Project\n\nwords\n<!-- END ABCD:licence -->\n", licenceMIT, readmePartUnclosed, []int{4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state, lines := parseReadmeParts([]byte(c.content)).classify("licence", c.fresh)
			if state != c.want || !reflect.DeepEqual(lines, c.lines) {
				t.Fatalf("classify = %q at %v, want %q at %v", state, lines, c.want, c.lines)
			}
		})
	}

	want := []readmePartWant{{Name: "licence", Body: licenceMIT}}

	t.Run("absent is written after the heading, owner lines unchanged", func(t *testing.T) {
		path := writeReadme(t, owner)
		res, err := syncReadmeParts(path, want)
		if err != nil || !res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		wantText := "# Project\n\n" + readmePart("licence", licenceMIT) + "\n\nThe owner's own words.\n"
		if got := readReadme(t, path); got != wantText {
			t.Fatalf("README after write:\n%q\nwant\n%q", got, wantText)
		}
		if !hasNote(res, "licence part written") {
			t.Errorf("notes = %q, want one saying the licence part was written", notes(res))
		}
	})

	t.Run("current writes nothing, bytes and modification time unchanged", func(t *testing.T) {
		path := writeReadme(t, owner)
		if _, err := syncReadmeParts(path, want); err != nil {
			t.Fatal(err)
		}
		before := readReadme(t, path)
		old := backdate(t, path)
		res, err := syncReadmeParts(path, want)
		if err != nil || res.Wrote {
			t.Fatalf("second sync: wrote=%v err=%v", res.Wrote, err)
		}
		if got := readReadme(t, path); got != before {
			t.Errorf("second run changed the bytes:\n%q\nwant\n%q", got, before)
		}
		if mt := modTime(t, path); !mt.Equal(old) {
			t.Errorf("second run changed the modification time: %v, want %v", mt, old)
		}
	})

	t.Run("stale is rewritten where it stands", func(t *testing.T) {
		path := writeReadme(t, cases[2].content)
		res, err := syncReadmeParts(path, want)
		if err != nil || !res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		if got := readReadme(t, path); got != cases[1].content {
			t.Fatalf("README after rewrite:\n%q\nwant\n%q", got, cases[1].content)
		}
		if !hasNote(res, "licence part rewritten") {
			t.Errorf("notes = %q", notes(res))
		}
	})

	t.Run("edited by hand is left, byte-identical, and named with its line", func(t *testing.T) {
		path := writeReadme(t, cases[3].content)
		res, err := syncReadmeParts(path, want)
		if err != nil || res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		if got := readReadme(t, path); got != cases[3].content {
			t.Fatalf("an edited part was touched:\n%q", got)
		}
		if !hasNote(res, "licence part edited by hand at line 3; left as it is") {
			t.Errorf("notes = %q", notes(res))
		}
	})

	t.Run("edited by hand leaves only that part; the others are written", func(t *testing.T) {
		path := writeReadme(t, cases[3].content)
		res, err := syncReadmeParts(path, append(want, readmePartWant{Name: "links", Body: "- [Changelog](CHANGELOG.md)"}))
		if err != nil || !res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		wantText := "# Project\n\n" + edited + "\n\n" + readmePart("links", "- [Changelog](CHANGELOG.md)") + "\n\nThe owner's own words.\n"
		if got := readReadme(t, path); got != wantText {
			t.Fatalf("README:\n%q\nwant\n%q", got, wantText)
		}
		if !hasNote(res, "licence part edited by hand at line 3; left as it is") || !hasNote(res, "links part written") {
			t.Errorf("notes = %q", notes(res))
		}
	})

	refusals := []struct {
		name, content, note string
	}{
		{"duplicated", cases[4].content, "licence part duplicated at lines 3 and 9; nothing written"},
		{"unclosed opening", cases[5].content, "licence part's opening mark at line 3 has no closing mark; nothing written"},
		{"unclosed closing", cases[6].content, "licence part's closing mark at line 4 has no opening mark; nothing written"},
	}
	for _, r := range refusals {
		t.Run(r.name+" refuses the whole write", func(t *testing.T) {
			path := writeReadme(t, r.content)
			res, err := syncReadmeParts(path, append(want, readmePartWant{Name: "links", Body: "- [Changelog](CHANGELOG.md)"}))
			if err != nil || res.Wrote || !res.Refused {
				t.Fatalf("sync: wrote=%v refused=%v err=%v", res.Wrote, res.Refused, err)
			}
			if got := readReadme(t, path); got != r.content {
				t.Fatalf("a refused write changed the file:\n%q", got)
			}
			if !hasNote(res, r.note) {
				t.Errorf("notes = %q, want %q", notes(res), r.note)
			}
			rm, err := removeReadmeParts(path)
			if err != nil || rm.Wrote || !rm.Refused {
				t.Fatalf("remove: wrote=%v refused=%v err=%v", rm.Wrote, rm.Refused, err)
			}
			if got := readReadme(t, path); got != r.content {
				t.Fatalf("a refused removal changed the file:\n%q", got)
			}
		})
	}

	t.Run("removal leaves an owner's README with no part byte-identical", func(t *testing.T) {
		path := writeReadme(t, owner)
		old := backdate(t, path)
		res, err := removeReadmeParts(path)
		if err != nil || res.Wrote {
			t.Fatalf("remove: wrote=%v err=%v", res.Wrote, err)
		}
		if got := readReadme(t, path); got != owner {
			t.Fatalf("README changed:\n%q", got)
		}
		if mt := modTime(t, path); !mt.Equal(old) {
			t.Errorf("modification time changed")
		}
	})

	t.Run("removal takes the current part and names the edited one", func(t *testing.T) {
		content := "# Project\n\n" + readmePart("links", "- [Changelog](CHANGELOG.md)") + "\n\n" + edited + "\n\nThe owner's own words.\n"
		path := writeReadme(t, content)
		res, err := removeReadmeParts(path)
		if err != nil || !res.Wrote {
			t.Fatalf("remove: wrote=%v err=%v", res.Wrote, err)
		}
		wantText := "# Project\n\n" + edited + "\n\nThe owner's own words.\n"
		if got := readReadme(t, path); got != wantText {
			t.Fatalf("README after removal:\n%q\nwant\n%q", got, wantText)
		}
		if !hasNote(res, "links part removed") || !hasNote(res, "licence part edited by hand at line 7; left as it is") {
			t.Errorf("notes = %q", notes(res))
		}
	})

	t.Run("a part no longer chosen is removed when it matches its hash", func(t *testing.T) {
		path := writeReadme(t, "# Project\n\n"+readmePart("links", "- [Changelog](CHANGELOG.md)")+"\n\n"+readmePart("licence", licenceMIT)+"\n\nwords\n")
		res, err := syncReadmeParts(path, want)
		if err != nil || !res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		wantText := "# Project\n\n" + readmePart("licence", licenceMIT) + "\n\nwords\n"
		if got := readReadme(t, path); got != wantText {
			t.Fatalf("README:\n%q\nwant\n%q", got, wantText)
		}
	})
}

// TestReadmePartOfAnUnknownNameIsLeft: a well-formed part whose name is not
// one this abcd writes, a part a later abcd wrote, is left where it stands and
// named, as a part edited by hand is, by a write and by a removal alike, even
// when its body matches its hash: this abcd cannot tell whether the part is
// still wanted, so it does not take it out. The other parts are still
// written and removed.
func TestReadmePartOfAnUnknownNameIsLeft(t *testing.T) {
	later := readmePart("sponsors", "- [Sponsor](https://example.org/sponsor)")
	content := "# Project\n\n" + later + "\n\nThe owner's own words.\n"
	const note = "sponsors part at line 3 is not one this abcd writes; left as it is"

	path := writeReadme(t, content)
	res, err := syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: licenceMIT}})
	if err != nil || !res.Wrote {
		t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
	}
	got := readReadme(t, path)
	if !strings.Contains(got, later) {
		t.Fatalf("a write took out a part of an unknown name:\n%q", got)
	}
	if !strings.Contains(got, readmePart("licence", licenceMIT)) {
		t.Errorf("the chosen part was not written beside it:\n%q", got)
	}
	if !hasNote(res, note) {
		t.Errorf("notes = %q, want %q", notes(res), note)
	}

	path = writeReadme(t, content)
	old := backdate(t, path)
	rm, err := removeReadmeParts(path)
	if err != nil || rm.Wrote {
		t.Fatalf("remove: wrote=%v err=%v", rm.Wrote, err)
	}
	if got := readReadme(t, path); got != content {
		t.Fatalf("a removal changed a README holding only a part of an unknown name:\n%q", got)
	}
	if mt := modTime(t, path); !mt.Equal(old) {
		t.Errorf("modification time changed")
	}
	if !hasNote(rm, note) {
		t.Errorf("notes = %q, want %q", notes(rm), note)
	}
}

// TestReadmePartFenceRoundTrip holds the marked place's exact form: the fences
// name the part, the opening one carries SHA-256 over the body's lines joined
// with LF and no trailing newline, the parser reads back what the renderer
// wrote, and a write followed by a removal gives back the owner's bytes.
func TestReadmePartFenceRoundTrip(t *testing.T) {
	bodies := []string{licenceMIT, "- [Changelog](CHANGELOG.md)\n- [Contributing](CONTRIBUTING.md)", ""}
	for _, body := range bodies {
		sum := sha256.Sum256([]byte(body))
		hash := hex.EncodeToString(sum[:])
		if got := readmePartHash(body); got != hash {
			t.Errorf("readmePartHash(%q) = %s, want %s", body, got, hash)
		}
		for _, eol := range []string{"\n", "\r\n"} {
			rendered := string(renderReadmePart("links", body, []byte(eol)))
			lines := strings.Split(rendered, eol)
			if lines[0] != "<!-- BEGIN ABCD:links sha256="+hash+" -->" {
				t.Errorf("opening fence = %q", lines[0])
			}
			if lines[len(lines)-1] != "<!-- END ABCD:links -->" {
				t.Errorf("closing fence = %q", lines[len(lines)-1])
			}
			if eol == "\n" && strings.Contains(rendered, "\r") {
				t.Errorf("LF render holds a CR: %q", rendered)
			}
			doc := parseReadmeParts([]byte(rendered))
			if len(doc.faults) != 0 || len(doc.parts) != 1 {
				t.Fatalf("parse of %q: parts=%v faults=%v", rendered, doc.parts, doc.faults)
			}
			p := doc.parts[0]
			if p.Name != "links" || p.Hash != hash || p.Body != body || !p.hashMatches() {
				t.Errorf("parsed %+v, want links with hash %s and body %q", p, hash, body)
			}
		}
	}
	if readmePartHash("a\r\nb") != readmePartHash("a\nb") {
		t.Error("a CRLF body hashes differently from its LF form")
	}

	// Each original and what a write then a removal gives back. Removal is
	// composeMarkerRemoval's, the conventions block's: where a part stood
	// between two texts it leaves one blank line, so text the owner had set
	// directly under a heading or frontmatter comes back a blank line below it,
	// which renders the same.
	originals := []struct{ original, back string }{
		{"", ""},
		{"# Project\n", "# Project\n"},
		{"# Project", "# Project"},
		{"# Project\n\nprose\n", "# Project\n\nprose\n"},
		{"# Project\nprose right under\n", "# Project\n\nprose right under\n"},
		{"---\ntitle: x\n---\n\n# Project\n\nprose\n", "---\ntitle: x\n---\n\n# Project\n\nprose\n"},
		{"just prose, no heading\n\nmore\n", "just prose, no heading\n\nmore\n"},
		{"# Project\r\n\r\nprose\r\n", "# Project\r\n\r\nprose\r\n"},
	}
	want := []readmePartWant{
		{Name: "release-badge", Body: "[![Release](https://img.shields.io/github/v/release/o/r)](https://github.com/o/r/releases/latest)"},
		{Name: "licence", Body: licenceMIT},
		{Name: "links", Body: "- [Changelog](CHANGELOG.md)"},
	}
	for _, o := range originals {
		original := o.original
		path := writeReadme(t, original)
		if _, err := syncReadmeParts(path, want); err != nil {
			t.Fatal(err)
		}
		doc := parseReadmeParts([]byte(readReadme(t, path)))
		if len(doc.parts) != 3 || len(doc.faults) != 0 {
			t.Fatalf("after writing into %q: parts=%v faults=%v", original, doc.parts, doc.faults)
		}
		for _, w := range want {
			if s, _ := doc.classify(w.Name, w.Body); s != readmePartCurrent {
				t.Errorf("%s reads %q after the write into %q", w.Name, s, original)
			}
		}
		if _, err := removeReadmeParts(path); err != nil {
			t.Fatal(err)
		}
		if got, wantBack := readReadme(t, path), o.back; got != wantBack {
			t.Errorf("write then remove of %q gave back %q", original, got)
		}
	}
}

// TestReadmePartFencesInsideCodeAreIgnored: a README that shows the fences in a
// code example, whole or broken, holds no part and refuses nothing, and a part
// written into it lands outside the example.
func TestReadmePartFencesInsideCodeAreIgnored(t *testing.T) {
	example := "```markdown\n" + readmePart("licence", "an example") + "\n<!-- BEGIN ABCD:links sha256=00 -->\n```\n"
	tilde := "~~~\n<!-- END ABCD:contents -->\n~~~\n"
	content := "# Project\n\nHow a part looks:\n\n" + example + "\n" + tilde
	doc := parseReadmeParts([]byte(content))
	if len(doc.parts) != 0 || len(doc.faults) != 0 {
		t.Fatalf("fences inside code were read: parts=%v faults=%v", doc.parts, doc.faults)
	}
	for _, name := range []string{"licence", "links", "contents"} {
		if s, _ := doc.classify(name, "x"); s != readmePartAbsent {
			t.Errorf("%s reads %q, want absent", name, s)
		}
	}
	path := writeReadme(t, content)
	res, err := syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: licenceMIT}})
	if err != nil || !res.Wrote || res.Refused {
		t.Fatalf("sync: wrote=%v refused=%v err=%v", res.Wrote, res.Refused, err)
	}
	wantText := "# Project\n\n" + readmePart("licence", licenceMIT) + "\n\nHow a part looks:\n\n" + example + "\n" + tilde
	if got := readReadme(t, path); got != wantText {
		t.Fatalf("README:\n%q\nwant\n%q", got, wantText)
	}

	// A fenced `# ` line is no heading: the part goes after the live one.
	fencedH1 := "```bash\n# a shell comment\n```\n# Real Title\n\nbody\n"
	path = writeReadme(t, fencedH1)
	if _, err := syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: licenceMIT}}); err != nil {
		t.Fatal(err)
	}
	wantText = "```bash\n# a shell comment\n```\n# Real Title\n\n" + readmePart("licence", licenceMIT) + "\n\nbody\n"
	if got := readReadme(t, path); got != wantText {
		t.Fatalf("README:\n%q\nwant\n%q", got, wantText)
	}

	// A body that would break its own marks is refused, and nothing is written.
	path = writeReadme(t, "# Project\n")
	res, err = syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: "```\nan unclosed code block"}})
	if err != nil || res.Wrote {
		t.Fatalf("sync of a mark-breaking body: wrote=%v err=%v", res.Wrote, err)
	}
	if got := readReadme(t, path); got != "# Project\n" {
		t.Fatalf("README changed: %q", got)
	}
	if len(res.Parts) != 1 || res.Parts[0].Action != readmeActionRefused {
		t.Errorf("outcomes = %+v, want the licence part refused", res.Parts)
	}
}

// TestReadmePartPlacement holds where an absent part goes: beside the nearest
// present part in the fixed order, else after frontmatter, else after the first
// live level-one heading, else at the top, never appended at the end; each is
// set off by one blank line, and a part the owner moved is rewritten where it
// stands.
func TestReadmePartPlacement(t *testing.T) {
	lic := readmePart("licence", licenceMIT)
	licWant := []readmePartWant{{Name: "licence", Body: licenceMIT}}
	rb := readmePart("release-badge", "RB")
	ab := readmePart("abcd-badge", "AB")
	ln := readmePart("links", "LN")
	ct := readmePart("contents", "CT")
	cases := []struct {
		name, content string
		want          []readmePartWant
		result        string
	}{
		{"after frontmatter", "---\ntitle: x\n---\n# Project\n\ntext\n", licWant,
			"---\ntitle: x\n---\n\n" + lic + "\n\n# Project\n\ntext\n"},
		{"after the first heading", "# Project\nprose\n", licWant,
			"# Project\n\n" + lic + "\n\nprose\n"},
		{"at the top when the only heading has no newline", "# Project", licWant,
			lic + "\n\n# Project"},
		{"at the top with no heading", "prose\n", licWant,
			lic + "\n\nprose\n"},
		{"at the top, never appended inside an open code block", "text\n```\nnever closed\n", licWant,
			lic + "\n\ntext\n```\nnever closed\n"},
		{"at the top, never appended inside an open comment", "text\n<!-- never closed\n", licWant,
			lic + "\n\ntext\n<!-- never closed\n"},
		{"an empty file", "", licWant, lic + "\n"},
		{"beside the nearest part, before or after as the order says",
			"# Project\n\nintro\n\n" + ln + "\n\noutro\n",
			[]readmePartWant{{Name: "release-badge", Body: "RB"}, {Name: "licence", Body: licenceMIT}, {Name: "links", Body: "LN"}, {Name: "contents", Body: "CT"}},
			"# Project\n\nintro\n\n" + rb + "\n\n" + lic + "\n\n" + ln + "\n\n" + ct + "\n\noutro\n"},
		{"at equal distance, after the earlier part",
			ab + "\n\n# Project\n\nprose\n\n" + ln + "\n",
			[]readmePartWant{{Name: "abcd-badge", Body: "AB"}, {Name: "licence", Body: licenceMIT}, {Name: "links", Body: "LN"}},
			ab + "\n\n" + lic + "\n\n# Project\n\nprose\n\n" + ln + "\n"},
		{"a part the owner moved is rewritten where it stands",
			"# Project\n\nprose\n\n" + readmePart("licence", "old") + "\n",
			licWant,
			"# Project\n\nprose\n\n" + lic + "\n"},
		{"parts in a file with none, in the fixed order",
			"# Project\n\nprose\n",
			[]readmePartWant{{Name: "links", Body: "LN"}, {Name: "licence", Body: licenceMIT}},
			"# Project\n\n" + lic + "\n\n" + ln + "\n\nprose\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeReadme(t, c.content)
			if _, err := syncReadmeParts(path, c.want); err != nil {
				t.Fatal(err)
			}
			if got := readReadme(t, path); got != c.result {
				t.Fatalf("README:\n%q\nwant\n%q", got, c.result)
			}
		})
	}

	t.Run("a missing README is created holding only the parts", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "README.md")
		res, err := syncReadmeParts(path, []readmePartWant{{Name: "links", Body: "LN"}, {Name: "licence", Body: licenceMIT}})
		if err != nil || !res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		if got := readReadme(t, path); got != lic+"\n\n"+ln+"\n" {
			t.Fatalf("README:\n%q", got)
		}
	})

	t.Run("nothing chosen and no README creates nothing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "README.md")
		res, err := syncReadmeParts(path, nil)
		if err != nil || res.Wrote {
			t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("README created: %v", err)
		}
	})

	t.Run("a symlinked README is refused", func(t *testing.T) {
		dir := t.TempDir()
		real := filepath.Join(dir, "real.md")
		if err := os.WriteFile(real, []byte("# Real\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "README.md")
		if err := os.Symlink(real, path); err != nil {
			t.Fatal(err)
		}
		if _, err := syncReadmeParts(path, licWant); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("sync through a symlink: err=%v", err)
		}
		if got := readReadme(t, real); got != "# Real\n" {
			t.Fatalf("the symlink's target changed: %q", got)
		}
	})

	t.Run("a README over the size cap is refused", func(t *testing.T) {
		big := "# Big\n" + strings.Repeat("x", maxAhoyFileBytes)
		path := writeReadme(t, big)
		if _, err := syncReadmeParts(path, licWant); err == nil {
			t.Fatal("sync of an oversize README succeeded")
		}
		if got := readReadme(t, path); got != big {
			t.Fatal("an oversize README changed")
		}
	})

	t.Run("an unknown part name is a caller error", func(t *testing.T) {
		path := writeReadme(t, "# Project\n")
		if _, err := syncReadmeParts(path, []readmePartWant{{Name: "title", Body: "x"}}); err == nil {
			t.Fatal("an unknown part name was accepted")
		}
	})
}

// TestReadmePartKeepsLineEndings: a CRLF README keeps CRLF, the hash is the
// same as the LF form's, a second run writes nothing, and the owner's own line
// endings are never rewritten, even in a mixed file.
func TestReadmePartKeepsLineEndings(t *testing.T) {
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	body := "- [Changelog](CHANGELOG.md)\n- [Security](SECURITY.md)"
	want := []readmePartWant{{Name: "links", Body: body}}

	original := "# Project\r\n\r\nprose\r\n"
	path := writeReadme(t, original)
	if _, err := syncReadmeParts(path, want); err != nil {
		t.Fatal(err)
	}
	got := readReadme(t, path)
	wantText := "# Project\r\n\r\n" + crlf(readmePart("links", body)) + "\r\n\r\nprose\r\n"
	if got != wantText {
		t.Fatalf("CRLF README:\n%q\nwant\n%q", got, wantText)
	}
	if strings.Count(got, "\n") != strings.Count(got, "\r\n") {
		t.Errorf("a lone LF in a CRLF file: %q", got)
	}
	doc := parseReadmeParts([]byte(got))
	if len(doc.parts) != 1 || doc.parts[0].Hash != readmePartHash(body) || doc.parts[0].Body != body {
		t.Fatalf("parsed %+v", doc.parts)
	}
	old := backdate(t, path)
	res, err := syncReadmeParts(path, want)
	if err != nil || res.Wrote {
		t.Fatalf("second run: wrote=%v err=%v", res.Wrote, err)
	}
	if !modTime(t, path).Equal(old) {
		t.Error("second run changed the modification time")
	}
	// The same body written by hand in LF inside a CRLF file still matches.
	if s, _ := parseReadmeParts([]byte(crlf(readmePart("links", body)))).classify("links", body); s != readmePartCurrent {
		t.Errorf("CRLF part reads %q, want current", s)
	}
	if _, err := removeReadmeParts(path); err != nil {
		t.Fatal(err)
	}
	if got := readReadme(t, path); got != original {
		t.Errorf("remove gave back %q, want %q", got, original)
	}

	mixed := "# Project\r\nline a\nline b\r\n"
	path = writeReadme(t, mixed)
	if _, err := syncReadmeParts(path, want); err != nil {
		t.Fatal(err)
	}
	wantText = "# Project\r\n\r\n" + crlf(readmePart("links", body)) + "\r\n\r\nline a\nline b\r\n"
	if got := readReadme(t, path); got != wantText {
		t.Fatalf("mixed README:\n%q\nwant\n%q", got, wantText)
	}
}

// TestConventionsBlockIgnoresNamedFences: the conventions block's reading and
// the README parts' reading never see each other's fences. A file holding named
// parts has no conventions block; planting or stripping the block leaves the
// parts byte-identical; and writing or removing parts leaves the block alone.
func TestConventionsBlockIgnoresNamedFences(t *testing.T) {
	parts := readmePart("licence", licenceMIT) + "\n\n" + readmePart("links", "LN")
	content := "# Project\n\n" + parts + "\n\nprose\n"
	if doc := parseReadmeParts([]byte(content)); len(doc.parts) != 2 || len(doc.faults) != 0 {
		t.Fatalf("fixture holds parts=%v faults=%v, want two parts", doc.parts, doc.faults)
	}

	path := writeReadme(t, content)
	if s := classifyMarker(path); s != markerMissing {
		t.Errorf("classifyMarker = %q, want missing", s)
	}
	if markerFileHasBlock(path) {
		t.Error("markerFileHasBlock read a named fence as the conventions block")
	}
	if out, stripped := StripMarkerBlock([]byte(content)); stripped || string(out) != content {
		t.Errorf("StripMarkerBlock stripped named parts: %q", out)
	}
	if wrote, err := removeMarkerFile(path); err != nil || wrote {
		t.Errorf("removeMarkerFile: wrote=%v err=%v", wrote, err)
	}
	if wrote, err := installMarkerFile(path); err != nil || !wrote {
		t.Fatalf("installMarkerFile: wrote=%v err=%v", wrote, err)
	}
	withBlock := readReadme(t, path)
	if !strings.Contains(withBlock, parts) {
		t.Fatalf("planting the conventions block changed the parts:\n%s", withBlock)
	}
	if s := classifyMarker(path); s != markerCurrent {
		t.Errorf("classifyMarker after install = %q, want current", s)
	}
	doc := parseReadmeParts([]byte(withBlock))
	if len(doc.parts) != 2 || len(doc.faults) != 0 {
		t.Fatalf("the conventions block was read as a part: parts=%v faults=%v", doc.parts, doc.faults)
	}

	// Writing and removing parts leaves the conventions block byte-identical.
	block := string(synthesizeMarker(markerInner, []byte("\n")))
	res, err := syncReadmeParts(path, []readmePartWant{{Name: "licence", Body: licenceMIT}, {Name: "contents", Body: "CT"}})
	if err != nil || !res.Wrote {
		t.Fatalf("sync: wrote=%v err=%v", res.Wrote, err)
	}
	after := readReadme(t, path)
	if !strings.Contains(after, block) || classifyMarker(path) != markerCurrent {
		t.Fatalf("writing parts disturbed the conventions block:\n%s", after)
	}
	if _, err := removeReadmeParts(path); err != nil {
		t.Fatal(err)
	}
	after = readReadme(t, path)
	if !strings.Contains(after, block) || classifyMarker(path) != markerCurrent {
		t.Fatalf("removing parts disturbed the conventions block:\n%s", after)
	}
	if strings.Contains(after, "ABCD:") {
		t.Errorf("a part survived removal:\n%s", after)
	}
	if out, _ := StripMarkerBlock([]byte(after)); bytes.Contains(out, []byte("BEGIN ABCD")) {
		t.Errorf("the conventions block survived its strip:\n%s", out)
	}
}
