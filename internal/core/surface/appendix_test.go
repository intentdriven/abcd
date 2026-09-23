package surface

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureTree is a small command tree in snapshot form: a root with persistent
// flags, a verb with sub-verbs (one nested), a verb with neither flags nor
// sub-verbs, and a verb whose only surface is its own flags.
func fixtureTree() []Command {
	return []Command{
		{Path: "abcd", Flags: []Flag{{Name: "json", Type: "bool"}, {Name: "no-color", Type: "bool"}}},
		{Path: "abcd capture", Flags: []Flag{{Name: "severity", Type: "string"}}},
		{Path: "abcd capture list", Flags: []Flag{{Name: "all", Type: "bool"}}},
		{Path: "abcd capture resolve", Flags: []Flag{{Name: "commit", Shorthand: "c", Type: "string", Required: true}}},
		{Path: "abcd docs"},
		{Path: "abcd docs cite"},
		{Path: "abcd docs cite refresh", Flags: []Flag{{Name: "dry-run", Type: "bool", Hidden: true}}},
		{Path: "abcd version", Flags: []Flag{{Name: "check", Type: "bool"}}},
		{Path: "abcd empty"},
	}
}

func TestComposeAppendixListsFlagsAndSubVerbs(t *testing.T) {
	cases := []struct {
		name     string
		paths    []string
		want     []string
		wantNone []string
	}{
		{
			name:  "flags and sub-verbs",
			paths: []string{"abcd capture"},
			want: []string{
				"### `abcd capture`",
				"Sub-verbs: `abcd capture list`, `abcd capture resolve`.",
				"| `--severity` | string |",
				"### `abcd capture list`",
				"| `--all` | bool |",
				"### `abcd capture resolve`",
				"| `--commit`, `-c` (required) | string |",
			},
			wantNone: []string{"abcd version", "--json"},
		},
		{
			name:  "sub-verbs only, nested",
			paths: []string{"abcd docs"},
			want: []string{
				"### `abcd docs`",
				"Sub-verbs: `abcd docs cite`.",
				"### `abcd docs cite`",
				"Sub-verbs: `abcd docs cite refresh`.",
				"### `abcd docs cite refresh`",
				"| `--dry-run` (hidden) | bool |",
			},
		},
		{
			name:  "flags only",
			paths: []string{"abcd version"},
			want:  []string{"### `abcd version`", "Sub-verbs: none.", "| `--check` | bool |"},
		},
		{
			name:  "neither",
			paths: []string{"abcd empty"},
			want:  []string{"### `abcd empty`", "Sub-verbs: none.", "Flags: none."},
		},
		{
			name:     "the bare root lists its own flags and no verbs",
			paths:    []string{"abcd"},
			want:     []string{"### `abcd`", "| `--json` | bool |", "| `--no-color` | bool |"},
			wantNone: []string{"Sub-verbs", "abcd capture"},
		},
		{
			name:  "two commands in one chapter, in register order",
			paths: []string{"abcd version", "abcd empty"},
			want:  []string{"### `abcd version`", "### `abcd empty`"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComposeAppendix(tc.paths, fixtureTree())
			for _, w := range tc.want {
				if !strings.Contains(got, w+"\n") {
					t.Errorf("appendix lacks line %q:\n%s", w, got)
				}
			}
			for _, n := range tc.wantNone {
				if strings.Contains(got, n) {
					t.Errorf("appendix carries %q, which is not this chapter's surface:\n%s", n, got)
				}
			}
		})
	}
	// Register order is kept, not sorted: the chapter's own command comes first.
	two := ComposeAppendix([]string{"abcd version", "abcd empty"}, fixtureTree())
	if strings.Index(two, "`abcd version`") > strings.Index(two, "`abcd empty`") {
		t.Errorf("register order not kept:\n%s", two)
	}
}

// ac-3: a chapter whose surface the tree does not register gets the declared
// sentence and nothing else between the markers.
func TestComposeAppendixUnbuiltChapterStatesNoShippedSurface(t *testing.T) {
	got := ComposeAppendix([]string{"abcd reflect"}, fixtureTree())
	want := "\n" + UnbuiltSentence("abcd reflect") + "\n\n"
	if got != want {
		t.Fatalf("unbuilt appendix = %q, want exactly the declared sentence %q", got, want)
	}
	if !strings.Contains(UnbuiltSentence("abcd reflect"), "no shipped surface") {
		t.Fatalf("the unbuilt sentence does not say there is no shipped surface: %q", UnbuiltSentence("abcd reflect"))
	}
	text := "# Reflect\n\nWhy it exists.\n\n" + AppendixBegin + "\n" + AppendixEnd + "\n"
	out, err := RenderChapter(text, got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, AppendixBegin+"\n\n"+UnbuiltSentence("abcd reflect")+"\n\n"+AppendixEnd+"\n") {
		t.Fatalf("rendered unbuilt chapter lacks the markers around the sentence:\n%s", out)
	}
}

// ac-4: the composer's whole input is the command path, its flags and its
// sub-verbs; the flag rows carry the spelling and the value type only.
func TestComposeAppendixCarriesOnlyFlagsAndSubVerbs(t *testing.T) {
	got := ComposeAppendix([]string{"abcd capture"}, fixtureTree())
	for _, line := range strings.Split(got, "\n") {
		switch {
		case line == "", strings.HasPrefix(line, "### `abcd"), strings.HasPrefix(line, "Sub-verbs: "),
			line == "Flags: none.", line == "| Flag | Type |", line == "|---|---|",
			strings.HasPrefix(line, "| `--"), strings.HasPrefix(line, "## "), strings.HasPrefix(line, "_Generated"):
		default:
			t.Errorf("appendix line %q is neither a flag nor a sub-verb", line)
		}
	}
}

func chapterText(prose string) string {
	return prose + AppendixBegin + "\n" + AppendixEnd + "\n"
}

// ac-2 and idempotence: regeneration rewrites only the bytes between the
// markers, and a second regeneration changes nothing.
func TestRenderChapterPreservesProseAndIsIdempotent(t *testing.T) {
	prose := "# Capture\n\nThe ledger exists because *memory is lossy*.  \nTrailing spaces kept.\n\n"
	first, err := RenderChapter(chapterText(prose), ComposeAppendix([]string{"abcd capture"}, fixtureTree()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, prose+AppendixBegin+"\n") {
		t.Fatalf("prose above the marker changed:\n%s", first)
	}
	second, err := RenderChapter(first, ComposeAppendix([]string{"abcd capture"}, fixtureTree()))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("regeneration is not idempotent:\n--- first\n%s\n--- second\n%s", first, second)
	}
	p, err := SplitChapter(second)
	if err != nil || p != prose {
		t.Fatalf("SplitChapter = %q, %v; want the prose back", p, err)
	}
}

func TestSplitChapterRefusesMalformedMarkers(t *testing.T) {
	cases := []struct {
		name, text string
		want       error
	}{
		{"absent", "# Chapter\n\nProse.\n", ErrMarkerAbsent},
		{"begin only", "# C\n" + AppendixBegin + "\n", ErrMarkerAbsent},
		{"end only", "# C\n" + AppendixEnd + "\n", ErrMarkerAbsent},
		{"crossed", "# C\n" + AppendixEnd + "\n" + AppendixBegin + "\n", ErrMarkerCrossed},
		{"two begins", "# C\n" + AppendixBegin + "\n" + AppendixBegin + "\n" + AppendixEnd + "\n", ErrMarkerDuplicated},
		{"two ends", "# C\n" + AppendixBegin + "\n" + AppendixEnd + "\n" + AppendixEnd + "\n", ErrMarkerDuplicated},
		{"inside a fence", "# C\n```\n" + AppendixBegin + "\n```\n" + AppendixBegin + "\n" + AppendixEnd + "\n", ErrMarkerInFence},
		{"misspelt", "# C\n<!-- surface-appendix:start -->\n" + AppendixBegin + "\n" + AppendixEnd + "\n", ErrMarkerMalformed},
		{"prose after the end", "# C\n" + AppendixBegin + "\n" + AppendixEnd + "\n\nMore prose.\n", ErrMarkerNotAtEnd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SplitChapter(tc.text)
			if !errors.Is(err, tc.want) {
				t.Fatalf("SplitChapter err = %v, want %v", err, tc.want)
			}
			if _, rerr := RenderChapter(tc.text, "\nx\n"); !errors.Is(rerr, tc.want) {
				t.Fatalf("RenderChapter err = %v, want %v (it must refuse, not repair)", rerr, tc.want)
			}
		})
	}
}

// ac-1: a flag the tree gained and the chapter did not is a drift naming the
// missing claim; a flag the tree lost is a stale one.
func TestAppendixDriftNamesTheMissingClaim(t *testing.T) {
	tree := fixtureTree()
	committed, err := RenderChapter(chapterText("# Capture\n\n"), ComposeAppendix([]string{"abcd capture"}, tree))
	if err != nil {
		t.Fatal(err)
	}
	grown := fixtureTree()
	grown[2].Flags = append(grown[2].Flags, Flag{Name: "since", Type: "string"})
	grown = append(grown, Command{Path: "abcd capture wontfix"})
	want, err := RenderChapter(committed, ComposeAppendix([]string{"abcd capture"}, grown))
	if err != nil {
		t.Fatal(err)
	}
	missing, stale := DriftLines(committed, want)
	joined := strings.Join(missing, "\n")
	for _, claim := range []string{"| `--since` | string |", "### `abcd capture wontfix`"} {
		if !strings.Contains(joined, claim) {
			t.Errorf("drift does not name the missing claim %q; missing = %q", claim, missing)
		}
	}
	if len(stale) != 1 || !strings.Contains(stale[0], "Sub-verbs: `abcd capture list`, `abcd capture resolve`.") {
		t.Errorf("stale = %q, want the superseded sub-verb line", stale)
	}
	if m, s := DriftLines(committed, committed); len(m)+len(s) != 0 {
		t.Errorf("identical chapters drift: %q %q", m, s)
	}
}

// ac-5: the prose above the marker states no flag and no sub-verb.
func TestProseShapeClaims(t *testing.T) {
	tree := fixtureTree()
	clean := "# Capture\n\nThe ledger exists because memory is lossy. Listing is\n" +
		"cheap, resolving is deliberate — an em-dash is not a flag.\n\n" +
		"| a | b |\n|---|---|\n<!-- index: x -->\n\n" +
		"## Sub-verbs\n\n> _The `list` bucket note._\n\n| Verb | Bucket | Status |\n|---|---|---|\n| `list` | — | shipped |\n\n" +
		"## Next\n\nThe `abcd capture` verb and `/abcd:version` are verbs, not sub-verbs.\n"
	if got := ProseShapeClaims(clean, []string{"abcd capture"}, tree); len(got) != 0 {
		t.Fatalf("clean prose reported shape claims: %+v", got)
	}
	dirty := "# Capture\n\n" +
		"Pass --all to widen it.\n" + // 3: a flag spelling
		"Run `abcd capture resolve` later.\n" + // 4: a sub-verb as a command path
		"The `list` sub-verb reads.\n" + // 5: a bare sub-verb name of this chapter's verb
		"See /abcd:docs cite refresh too.\n" + // 6: another verb's sub-verb path
		"```\nabcd version --check\n```\n" + // 8: a fenced flag is still shape
		"`--json` is global.\n" + // 10
		"## Sub-verbs\n\n> _The `list` note is exempt._\n\n| `list` | — | shipped |\n\n" + // 11-16: exempt
		"The `list` row above is a gate.\n" + // 17: section prose is still checked
		"## Wrapped\n\nRun `/abcd:capture\n  resolve` or `abcd docs cite\nrefresh` later.\n" // 20-22: a path across a soft wrap
	got := ProseShapeClaims(dirty, []string{"abcd capture"}, tree)
	want := map[int]string{3: "--all", 4: "capture resolve", 5: "`list`", 6: "docs cite refresh", 8: "--check", 10: "--json", 17: "`list`", 20: "capture resolve", 21: "docs cite refresh"}
	seen := map[int]bool{}
	for _, c := range got {
		if w, ok := want[c.Line]; ok && c.Spelling == w {
			seen[c.Line] = true
		}
	}
	for line, w := range want {
		if !seen[line] {
			t.Errorf("line %d: shape claim %q not reported; got %+v", line, w, got)
		}
	}
	for _, c := range got {
		if c.Line >= 11 && c.Line <= 16 {
			t.Errorf("line %d: the Sub-verbs table and note are exempt, got %+v", c.Line, c)
		}
	}
}

// ac-5, the other side: a check that fires on things that are not abcd's shape
// forces true prose out of the record. Another program's flag, a flag abcd does
// not register, and a plain-English phrase that happens to spell a sub-verb path
// are prose; a registered flag (long or its single-dash shorthand) and a
// sub-verb path that is backticked or prefixed as an invocation are shape.
func TestProseShapeClaimsFiresOnlyOnAbcdShape(t *testing.T) {
	tree := fixtureTree()
	prose := "# Guard\n\n" +
		"Ask git with `git merge-base --is-ancestor <sha> origin/main` first.\n" + // 3: git's flag, unregistered
		"A force push spelled `git pus? --force` blocks, and --no-edit is git's too.\n" + // 4: git's flags
		"The capture list and the docs cite refresh are plain words here.\n" + // 5: plain-English paths
		"A `-C` for git and a `-x` for anything are not abcd's.\n" + // 6: unregistered shorthands
		"Pass `--commit` with the sha.\n" + // 7: a registered flag
		"Or its shorthand `-c`, or bare -c.\n" + // 8: a registered shorthand, twice
		"Then run `capture list` to read it.\n" + // 9: a backticked path
		"Or abcd capture resolve, or /abcd:docs cite refresh.\n" // 10: prefixed paths
	got := ProseShapeClaims(prose, []string{"abcd guard"}, tree)
	type key struct {
		line     int
		spelling string
	}
	want := map[key]bool{
		{7, "--commit"}: true, {8, "-c"}: true, {9, "capture list"}: true,
		{10, "capture resolve"}: true, {10, "docs cite refresh"}: true,
	}
	counts := map[key]int{}
	for _, c := range got {
		k := key{c.Line, c.Spelling}
		if !want[k] {
			t.Errorf("line %d: %q reported as abcd shape; it is prose", c.Line, c.Spelling)
		}
		counts[k]++
	}
	for k := range want {
		if counts[k] == 0 {
			t.Errorf("line %d: %q not reported; got %+v", k.line, k.spelling, got)
		}
	}
	if counts[key{8, "-c"}] != 2 {
		t.Errorf("line 8: want both spellings of the shorthand reported, got %d", counts[key{8, "-c"}])
	}
}

func TestParseRegisterAndChapters(t *testing.T) {
	register := "# Surfaces\n\n| # | Command | Status | Purpose | File |\n|---|---|---|---|---|\n" +
		"| 1 | `/abcd:capture` | shipped | x | [`06-capture.md`](06-capture.md) |\n" +
		"| 2 | `/abcd` | shipped | x | [`08-abcd.md`](08-abcd.md) |\n" +
		"| 3 | `/abcd:reflect` | staged | x | [`09-reflect.md`](09-reflect.md) |\n" +
		"| 4 | `/abcd:worktree` | staged | x | [`elsewhere`](../05-internals/03-configuration.md#the-worktree-store) |\n" +
		"| 5 | `/abcd:mode` | shipped | x | [`08-abcd.md`](08-abcd.md) |\n"
	rows := ParseRegister(register)
	if len(rows) != 5 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[1].Command != "abcd" || rows[0].Command != "abcd capture" || rows[3].Chapter != "" {
		t.Fatalf("rows = %+v", rows)
	}
	chs, err := Chapters(rows, []string{"06-capture.md", "08-abcd.md", "09-reflect.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(chs) != 3 || chs[1].File != "08-abcd.md" || strings.Join(chs[1].Commands, ",") != "abcd,abcd mode" {
		t.Fatalf("chapters = %+v", chs)
	}
	if _, err := Chapters(rows, []string{"06-capture.md", "08-abcd.md", "09-reflect.md", "30-peers.md"}); !errors.Is(err, ErrChapterWithoutRow) || !strings.Contains(err.Error(), "30-peers.md") {
		t.Fatalf("a chapter with no register row: err = %v", err)
	}
	if _, err := Chapters(rows, []string{"06-capture.md", "08-abcd.md"}); !errors.Is(err, ErrRowWithoutChapter) || !strings.Contains(err.Error(), "09-reflect.md") {
		t.Fatalf("a row naming a missing chapter: err = %v", err)
	}
}

func TestRegeneratedChapterDriftNamesChapterAndClaim(t *testing.T) {
	committed, _ := RenderChapter(chapterText("# Capture\n\n"), ComposeAppendix([]string{"abcd capture"}, fixtureTree()))
	grown := fixtureTree()
	grown[1].Flags = append(grown[1].Flags, Flag{Name: "recurs", Type: "string"})
	want, _ := RenderChapter(committed, ComposeAppendix([]string{"abcd capture"}, grown))
	rc := RegeneratedChapter{Chapter: Chapter{File: "06-capture.md", Commands: []string{"abcd capture"}}, Committed: committed, Want: want}
	msg := rc.Drift()
	for _, w := range []string{"04-surfaces/06-capture.md", "missing: | `--recurs` | string |"} {
		if !strings.Contains(msg, w) {
			t.Errorf("drift message lacks %q:\n%s", w, msg)
		}
	}
	rc.Want = committed
	if rc.Drift() != "" {
		t.Errorf("an agreeing chapter reports drift: %q", rc.Drift())
	}
}

// Two lanes landing surfaces in sequence each add a register row and a chapter.
// A chapter that cannot be regenerated — its markers absent, or no row naming it
// — is refused by name, and every other chapter is still regenerated, so one
// unfinished chapter never hides the drift of the rest.
func TestRegenerateChaptersSkipsAndReportsARefusedChapter(t *testing.T) {
	dir := t.TempDir()
	register := "# Surfaces\n\n| # | Command | Status | Purpose | File |\n|---|---|---|---|---|\n" +
		"| 1 | `/abcd:capture` | shipped | x | [`06-capture.md`](06-capture.md) |\n" +
		"| 2 | `/abcd:version` | shipped | x | [`12-version.md`](12-version.md) |\n" +
		"| 3 | `/abcd:docs` | shipped | x | [`10-docs.md`](10-docs.md) |\n"
	files := map[string]string{
		RegisterFile:    register,
		"06-capture.md": chapterText("# Capture\n\n"),
		"12-version.md": "# Version\n\nNo markers yet.\n",
		"10-docs.md":    chapterText("# Docs\n\n"),
		"30-peers.md":   chapterText("# Peers\n\n"),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := RegenerateChapters(dir, fixtureTree())
	if !errors.Is(err, ErrMarkerAbsent) || !strings.Contains(err.Error(), "12-version.md") {
		t.Errorf("err = %v; want the unmarked chapter refused by name", err)
	}
	if !errors.Is(err, ErrChapterWithoutRow) || !strings.Contains(err.Error(), "30-peers.md") {
		t.Errorf("err = %v; want the chapter with no row refused by name", err)
	}
	var names []string
	for _, c := range got {
		names = append(names, c.File)
		if !strings.Contains(c.Want, AppendixBegin) {
			t.Errorf("%s: not regenerated: %q", c.File, c.Want)
		}
	}
	if strings.Join(names, ",") != "06-capture.md,10-docs.md" {
		t.Errorf("regenerated %v; want every chapter but the refused ones", names)
	}
}
