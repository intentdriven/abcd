package ahoy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// Tools' own conventions files in an adopted project (itd-2610030814013772,
// spc-2610031156364295 step 4, criteria A3 and A4): a file holding the owner's
// words is never touched and is named in a warning; a file that only repeats
// AGENTS.md is offered for retirement, at a terminal, and removed only on the
// person's own answer.

const agentsFixture = "# Conventions\n\nRun make check before a commit.\n"

// conventionsRoot is a project root holding AGENTS.md and nothing else.
func conventionsRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "AGENTS.md", agentsFixture, 0o644)
	return root
}

func writeFixture(t *testing.T, root, rel, body string, mode os.FileMode) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
}

func linkFixture(t *testing.T, root, rel, target string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, p); err != nil {
		t.Fatal(err)
	}
}

// TestToolFileClassification holds each class: the five ways a file repeats
// AGENTS.md, and every way a file is the owner's words, reading nothing
// through a link and nothing it cannot read guarded.
func TestToolFileClassification(t *testing.T) {
	type tc struct {
		name  string
		rel   string
		build func(t *testing.T, root string)
		class toolFileClass
		how   string
	}
	cases := []tc{
		{"a link to AGENTS.md", "CLAUDE.md", func(t *testing.T, root string) {
			linkFixture(t, root, "CLAUDE.md", "AGENTS.md")
		}, toolFileRepeats, repeatsLink},
		{"a link in a folder to AGENTS.md", ".claude/CLAUDE.md", func(t *testing.T, root string) {
			linkFixture(t, root, ".claude/CLAUDE.md", "../AGENTS.md")
		}, toolFileRepeats, repeatsLink},
		{"an absolute link to AGENTS.md", "GEMINI.md", func(t *testing.T, root string) {
			linkFixture(t, root, "GEMINI.md", filepath.Join(root, "AGENTS.md"))
		}, toolFileRepeats, repeatsLink},
		{"a link checked out as text", "GEMINI.md", func(t *testing.T, root string) {
			writeFixture(t, root, "GEMINI.md", "AGENTS.md", 0o644)
		}, toolFileRepeats, repeatsLinkText},
		{"a byte-for-byte copy", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "CLAUDE.md", agentsFixture, 0o644)
		}, toolFileRepeats, repeatsCopy},
		{"a lone import line", ".rules", func(t *testing.T, root string) {
			writeFixture(t, root, ".rules", "\n@AGENTS.md\n\n", 0o644)
		}, toolFileRepeats, repeatsImport},
		{"blank once abcd's block is stripped", "CLAUDE.md", func(t *testing.T, root string) {
			if _, err := installMarkerFile(filepath.Join(root, "CLAUDE.md")); err != nil {
				t.Fatal(err)
			}
		}, toolFileRepeats, repeatsBlank},
		{"an empty file", ".cursorrules", func(t *testing.T, root string) {
			writeFixture(t, root, ".cursorrules", "", 0o644)
		}, toolFileRepeats, repeatsBlank},

		{"the owner's words", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "CLAUDE.md", "Always run make check first\n", 0o644)
		}, toolFileOwners, ""},
		{"an import beside the owner's words", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "CLAUDE.md", "@AGENTS.md\nAlso run the tests.\n", 0o644)
		}, toolFileOwners, ""},
		{"the owner's words beside abcd's block", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "CLAUDE.md", "Always run make check first\n", 0o644)
			if _, err := installMarkerFile(filepath.Join(root, "CLAUDE.md")); err != nil {
				t.Fatal(err)
			}
		}, toolFileOwners, ""},
		{"a link to another file", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "README.md", agentsFixture, 0o644)
			linkFixture(t, root, "CLAUDE.md", "README.md")
		}, toolFileOwners, ""},
		{"text naming another file", "GEMINI.md", func(t *testing.T, root string) {
			writeFixture(t, root, "GEMINI.md", "README.md", 0o644)
		}, toolFileOwners, ""},
		{"too large to read", ".github/copilot-instructions.md", func(t *testing.T, root string) {
			writeFixture(t, root, ".github/copilot-instructions.md", strings.Repeat("\n", maxAhoyFileBytes+1), 0o644)
		}, toolFileOwners, ""},
		{"unreadable", "CLAUDE.md", func(t *testing.T, root string) {
			writeFixture(t, root, "CLAUDE.md", "AGENTS.md", 0o000)
		}, toolFileOwners, ""},
		{"not a regular file", "CLAUDE.md", func(t *testing.T, root string) {
			if err := os.Mkdir(filepath.Join(root, "CLAUDE.md"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, toolFileOwners, ""},
		{"in a linked folder", ".claude/CLAUDE.md", func(t *testing.T, root string) {
			elsewhere := t.TempDir()
			writeFixture(t, elsewhere, "CLAUDE.md", "AGENTS.md", 0o644)
			linkFixture(t, root, ".claude", elsewhere)
		}, toolFileOwners, ""},

		{"absent", "CLAUDE.md", func(*testing.T, string) {}, toolFileAbsent, ""},
	}
	if os.Geteuid() == 0 {
		t.Log("running as root: the unreadable case reads all the same and is skipped")
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.name == "unreadable" && os.Geteuid() == 0 {
				t.Skip("root reads a mode-000 file")
			}
			root := conventionsRoot(t)
			c.build(t, root)
			class, how := classifyToolFile(root, c.rel)
			if class != c.class || how != c.how {
				t.Errorf("classifyToolFile(%s) = (%v, %q), want (%v, %q)", c.rel, class, how, c.class, c.how)
			}
			if c.name == "unreadable" {
				_ = os.Chmod(filepath.Join(root, "CLAUDE.md"), 0o644)
			}
		})
	}
}

// fileState is what a test compares to prove a file untouched.
type fileState struct {
	data []byte
	mode os.FileMode
	link string
}

func stateOf(t *testing.T, p string) fileState {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatalf("%s: %v", p, err)
	}
	s := fileState{mode: fi.Mode()}
	if fi.Mode()&os.ModeSymlink != 0 {
		s.link, _ = os.Readlink(p)
		return s
	}
	s.data, _ = os.ReadFile(p)
	return s
}

// conventionsPrompter approves every category question at a terminal,
// declines every other confirm, and answers each retirement question as told.
func conventionsPrompter(answer string) terminalScripted {
	sp := &scriptedPrompter{
		confirm: func(q string) bool { return strings.HasPrefix(q, "Apply ") },
		answers: map[string]string{},
	}
	for _, f := range toolConventionsFiles {
		for _, how := range repeatKinds {
			sp.answers[retirePromptKey(f.Rel, how)] = answer
		}
	}
	return terminalScripted{sp}
}

func retireAsked(asked []string) []string {
	var out []string
	for _, q := range asked {
		if strings.HasPrefix(q, conventionsRetirePromptPrefix) {
			out = append(out, q)
		}
	}
	return out
}

// TestOwnersToolFileIsUntouchedAndNamed (A3): a CLAUDE.md holding the owner's
// words survives an install under --yes and one at a terminal byte for byte,
// mode included, is never offered for retirement, and is named in exactly one
// warning that says AGENTS.md stays hidden and how to end it.
func TestOwnersToolFileIsUntouchedAndNamed(t *testing.T) {
	for _, mode := range []string{"--yes", "terminal"} {
		t.Run(mode, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, repo, "CLAUDE.md", "Always run make check first\n", 0o640)
			before := stateOf(t, filepath.Join(repo, "CLAUDE.md"))

			opts := installOpts()
			var p Prompter = RefusingPrompter{}
			var tp terminalScripted
			if mode == "terminal" {
				opts.Yes = false
				tp = conventionsPrompter("retire")
				p = tp
			}
			res, err := Install(repo, opts, p)
			if err != nil {
				t.Fatal(err)
			}
			after := stateOf(t, filepath.Join(repo, "CLAUDE.md"))
			if !bytes.Equal(before.data, after.data) || before.mode != after.mode {
				t.Errorf("the owner's CLAUDE.md changed: %q %v -> %q %v", before.data, before.mode, after.data, after.mode)
			}
			if tp.scriptedPrompter != nil {
				if got := retireAsked(tp.asked); len(got) != 0 {
					t.Errorf("the owner's file was offered for retirement: %v", got)
				}
			}
			if len(res.Warnings) != 1 {
				t.Fatalf("warnings = %q, want exactly one", res.Warnings)
			}
			w := res.Warnings[0]
			for _, want := range []string{
				"CLAUDE.md holds your own words, so Claude Code reads it and not AGENTS.md",
				"hidden from Claude Code",
				"move those words into AGENTS.md and remove CLAUDE.md",
				"abcd never edits or removes this file",
			} {
				if !strings.Contains(w, want) {
					t.Errorf("the warning does not say %q:\n%s", want, w)
				}
			}
			if strings.Contains(w, "\n") {
				t.Errorf("the warning is not one line:\n%s", w)
			}
			raw, err := json.Marshal(res)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(raw, []byte(`"warnings":[`)) {
				t.Errorf("the JSON carries no warnings list: %s", raw)
			}
		})
	}
}

// TestOwnersFileWarningNamesEachTool: the warning names the file and the tool
// that loses AGENTS.md, for every file of the registry.
func TestOwnersFileWarningNamesEachTool(t *testing.T) {
	for _, f := range toolConventionsFiles {
		w := ownerFileWarning(f)
		if !strings.HasPrefix(w, f.Rel+" holds your own words, so "+f.Tool) ||
			!strings.Contains(w, "hidden from "+f.Tool) || !strings.Contains(w, "remove "+f.Rel+".") {
			t.Errorf("%s: the warning does not name the file and %s:\n%s", f.Rel, f.Tool, w)
		}
	}
}

// repeatingRepo is an installed repository (AGENTS.md carrying abcd's block)
// holding a GEMINI.md link to AGENTS.md and a CLAUDE.md copy of it.
func repeatingRepo(t *testing.T) string {
	t.Helper()
	setupHermetic(t)
	repo := installedRepo(t)
	block, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	// The owner's words beside abcd's block, so a copy is not also blank once
	// the block is stripped.
	agents := agentsFixture + "\n" + string(block)
	writeFixture(t, repo, "AGENTS.md", agents, 0o644)
	linkFixture(t, repo, "GEMINI.md", "AGENTS.md")
	writeFixture(t, repo, "CLAUDE.md", string(agents), 0o644)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, g := range det.Gaps {
		if g.ID == ConventionsRetireGapID {
			n++
			if g.Required || !g.Resolvable || g.Category != ConventionsFile {
				t.Errorf("the offer is not an optional, resolvable %s gap: %+v", ConventionsFile, g)
			}
		}
	}
	if n != 2 {
		t.Fatalf("want one retirement offer per file, got %d: %+v", n, det.Gaps)
	}
	return repo
}

// TestRepeatingToolFileIsOfferedForRetirement (A4): each answer to each file;
// --yes and a piped run do not ask, and name the offer in optional_skipped.
func TestRepeatingToolFileIsOfferedForRetirement(t *testing.T) {
	for _, answer := range []string{"retire", "keep", "later"} {
		t.Run(answer, func(t *testing.T) {
			repo := repeatingRepo(t)
			gemini, claude := filepath.Join(repo, "GEMINI.md"), filepath.Join(repo, "CLAUDE.md")
			beforeG, beforeC := stateOf(t, gemini), stateOf(t, claude)
			p := conventionsPrompter(answer)
			opts := installOpts()
			opts.Yes = false
			res, err := Install(repo, opts, p)
			if err != nil {
				t.Fatal(err)
			}
			asked := retireAsked(p.asked)
			want := []string{retirePromptKey("CLAUDE.md", repeatsCopy), retirePromptKey("GEMINI.md", repeatsLink)}
			if strings.Join(asked, "|") != strings.Join(want, "|") {
				t.Fatalf("asked %q, want one question per file %q", asked, want)
			}
			switch answer {
			case "retire":
				for _, p := range []string{gemini, claude} {
					if _, err := os.Lstat(p); !os.IsNotExist(err) {
						t.Errorf("%s was not removed on retire: %v", p, err)
					}
				}
				for _, rel := range []string{"CLAUDE.md", "GEMINI.md"} {
					if !containsString(res.Writes, rel) {
						t.Errorf("the removal of %s is not reported: %v", rel, res.Writes)
					}
				}
				det, _ := Detect(repo)
				if hasGap(det.Gaps, ConventionsRetireGapID) {
					t.Error("a retired file is offered again")
				}
			default:
				if g := stateOf(t, gemini); g.link != "AGENTS.md" || g.link != beforeG.link || g.mode != beforeG.mode {
					t.Errorf("GEMINI.md changed on %s: %+v", answer, g)
				}
				if c := stateOf(t, claude); !bytes.Equal(c.data, beforeC.data) || c.mode != beforeC.mode {
					t.Errorf("CLAUDE.md changed on %s", answer)
				}
				det, _ := Detect(repo)
				if !hasGap(det.Gaps, ConventionsRetireGapID) {
					t.Errorf("%s was recorded; the next install must offer again", answer)
				}
			}
		})
	}

	t.Run("--yes", func(t *testing.T) {
		repo := repeatingRepo(t)
		p := conventionsPrompter("retire")
		res, err := Install(repo, installOpts(), p)
		if err != nil {
			t.Fatal(err)
		}
		if got := retireAsked(p.asked); len(got) != 0 {
			t.Errorf("--yes asked %v", got)
		}
		assertKeptAndSkipped(t, repo, res)
	})

	t.Run("piped", func(t *testing.T) {
		repo := repeatingRepo(t)
		p := &pipedPrompter{answers: []bool{true, true, true, true, true, true, true, true}}
		opts := installOpts()
		opts.Yes = false
		res, err := Install(repo, opts, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, q := range p.asked {
			if strings.Contains(q, string(ConventionsFile)) {
				t.Errorf("a piped run was asked %q", q)
			}
		}
		assertKeptAndSkipped(t, repo, res)
	})
}

func assertKeptAndSkipped(t *testing.T, repo string, res InstallResult) {
	t.Helper()
	for _, rel := range []string{"GEMINI.md", "CLAUDE.md"} {
		if _, err := os.Lstat(filepath.Join(repo, rel)); err != nil {
			t.Errorf("%s was removed without being asked: %v", rel, err)
		}
	}
	if !containsString(res.OptionalSkipped, ConventionsRetireGapID) {
		t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, ConventionsRetireGapID)
	}
}

// TestRetireRechecksTheFile: a file that changes while its question is open is
// classified again at the answer, left as it now is, and named.
func TestRetireRechecksTheFile(t *testing.T) {
	repo := repeatingRepo(t)
	claude := filepath.Join(repo, "CLAUDE.md")
	p := conventionsPrompter("retire")
	const owners = "Always run make check first\n"
	p.onPrompt = func(key string) {
		if key == retirePromptKey("CLAUDE.md", repeatsCopy) {
			if err := os.WriteFile(claude, []byte(owners), 0o644); err != nil {
				t.Error(err)
			}
		}
	}
	opts := installOpts()
	opts.Yes = false
	res, err := Install(repo, opts, p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(claude)
	if err != nil {
		t.Fatalf("CLAUDE.md, changed while the question was open, was removed: %v", err)
	}
	if string(got) != owners {
		t.Errorf("CLAUDE.md = %q, want the words written while the question was open", got)
	}
	named := false
	for _, n := range res.Notes {
		if strings.Contains(n, "CLAUDE.md") && strings.Contains(n, "changed while the question was open") {
			named = true
		}
	}
	if !named {
		t.Errorf("the changed file is not named: notes %q", res.Notes)
	}
	if containsString(res.Writes, "CLAUDE.md") {
		t.Errorf("a removal that did not happen is reported: %v", res.Writes)
	}
	if _, err := os.Lstat(filepath.Join(repo, "GEMINI.md")); !os.IsNotExist(err) {
		t.Errorf("GEMINI.md, unchanged and answered retire, was not removed: %v", err)
	}
}

// TestConventionsRetireQuestionPassesTheLimits holds the retirement question,
// for every file and every way it repeats AGENTS.md, to the asking rules the
// other fixed setup questions keep (spc-2610030944505997,
// spc-2610030911534855): the file and what it repeats first, the question
// last, the decide-later answer last.
func TestConventionsRetireQuestionPassesTheLimits(t *testing.T) {
	n := 0
	for _, f := range toolConventionsFiles {
		for _, how := range repeatKinds {
			n++
			key := retirePromptKey(f.Rel, how)
			h, ok := HelpFor(key)
			if !ok {
				t.Fatalf("no help for %s", key)
			}
			if !strings.HasPrefix(h.About, f.Rel+" ") || !strings.Contains(h.About, "AGENTS.md") {
				t.Errorf("%s: the material does not open on the file and what it repeats:\n%s", key, h.About)
			}
			ask := retireAsk(f.Rel)
			material, found := strings.CutSuffix(h.About, " "+ask)
			if !found {
				t.Fatalf("%s: the question does not end on %q:\n%s", key, ask, h.About)
			}
			var vals []string
			for _, c := range h.Choices {
				vals = append(vals, c.Value)
			}
			if strings.Join(vals, ",") != strings.Join(retireChoices, ",") || retireChoices[len(retireChoices)-1] != retireLater {
				t.Errorf("%s: choices %v, want %v with decide-later last", key, vals, retireChoices)
			}
			q := question.Question{
				ID:       key,
				Chip:     "Setup Q1",
				Material: []question.Block{{Kind: question.KindParagraph, Text: material}},
				Ask:      ask,
				Options: []question.Option{
					{Value: h.Choices[0].Value, Label: h.Choices[0].Value, Meaning: h.Choices[0].Meaning},
					{Value: h.Choices[1].Value, Label: h.Choices[1].Value, Meaning: h.Choices[1].Meaning},
				},
				Later: question.Option{
					Value:   h.Choices[2].Value,
					Label:   question.Default.LaterLabels[0],
					Meaning: h.Choices[2].Meaning,
				},
				Now:         question.Default.NotApplicable,
				ChangeLater: question.Default.NotApplicable,
			}
			a := question.Ask{Questions: []question.Question{q}}
			for _, finding := range question.Check(a) {
				t.Errorf("%s: structural: %s", key, finding)
			}
			for _, finding := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
				t.Errorf("%s: %s", key, finding)
			}
		}
	}
	if _, ok := HelpFor(conventionsRetirePromptPrefix + "README.md:" + repeatsCopy); ok {
		t.Error("help is given for a file the registry does not name")
	}
	if _, ok := HelpFor(retirePromptKey("CLAUDE.md", "no-such-kind")); ok {
		t.Error("help is given for a way of repeating AGENTS.md that does not exist")
	}
	if n == 0 {
		t.Fatal("the registry is empty")
	}
}
