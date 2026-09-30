package docfidelity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const falseLine = "Capture prints YAML."

func holdWithDraft() fixedReviewer {
	return fixedReviewer{Status: ReviewHold, Commit: "c1", Findings: []Sentence{{
		Doc: DocBrief, Chapter: "06-capture.md", Sentence: falseLine,
		Replacement: "Capture prints JSON.", Evidence: "cli.go:1 prints JSON",
	}}}
}

func lagging() Inputs {
	in := fixture()
	in.Chapters["06-capture.md"] += falseLine + "\n"
	return in
}

// The judge proposes the reviewer's drafted edit and never applies it.
func TestJudgeProposesTheDraftedEditAndStillRefuses(t *testing.T) {
	v := Judge(lagging(), holdWithDraft(), false)
	if !v.Refuse {
		t.Fatal("a confirmed false sentence with a draft did not refuse before it was applied")
	}
	if len(v.Proposed) != 1 || v.Proposed[0].Replacement != "Capture prints JSON." {
		t.Fatalf("the drafted edit is not proposed: %+v", v.Proposed)
	}
}

// ac-5: applied and flagged, the change proceeds and says what was applied.
func TestAnAppliedAndFlaggedEditLetsTheChangeProceed(t *testing.T) {
	in := lagging()
	in.Chapters["06-capture.md"] = strings.Replace(in.Chapters["06-capture.md"], falseLine, "Capture prints JSON.", 1)
	in.Flags = []Flag{{Chapter: "06-capture.md", Sentence: falseLine, Replacement: "Capture prints JSON."}}
	v := Judge(in, holdWithDraft(), false)
	if v.Refuse {
		t.Fatalf("an applied and flagged edit still refused: %v", v.Reasons)
	}
	if len(v.Applied) != 1 {
		t.Fatalf("the applied edit is not reported for review: %+v", v)
	}
}

// ac-5's other half: no path completes with the brief lagging and no flag.
func TestAnEditWithoutAFlagStillRefuses(t *testing.T) {
	in := lagging()
	in.Chapters["06-capture.md"] = strings.Replace(in.Chapters["06-capture.md"], falseLine, "Capture prints JSON.", 1)
	if v := Judge(in, holdWithDraft(), false); !v.Refuse {
		t.Fatal("an unflagged edit let the change proceed")
	}
	flagged := lagging() // flagged, but the chapter still carries the sentence
	flagged.Flags = []Flag{{Chapter: "06-capture.md", Sentence: falseLine, Replacement: "Capture prints JSON."}}
	if v := Judge(flagged, holdWithDraft(), false); !v.Refuse {
		t.Fatal("a flag with the sentence still in the chapter let the change proceed")
	}
}

func TestApplyWritesTheChapterAndTheFlag(t *testing.T) {
	root := armedRepo(t)
	write(t, root, ChaptersDir+"/06-capture.md", "### `abcd capture`\n\n"+falseLine+" It lists.\n")
	edits := []Edit{{Chapter: "06-capture.md", Sentence: falseLine, Replacement: "Capture prints JSON.", Evidence: "cli.go:1"}}
	flags, err := Apply(root, edits, "c1", at)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(root, ChaptersDir, "06-capture.md"))
	if strings.Contains(string(data), falseLine) || !strings.Contains(string(data), "Capture prints JSON. It lists.") {
		t.Fatalf("the chapter was not edited:\n%s", data)
	}
	if len(flags) != 1 {
		t.Fatalf("flags = %+v", flags)
	}
	in, err := ReadInputs(root, tree, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Flags) != 1 || in.Flags[0].Sentence != falseLine || in.Flags[0].Commit != "c1" {
		t.Fatalf("the flag was not recorded: %+v", in.Flags)
	}
}

func TestApplyRefusesAnAmbiguousOrAbsentSentenceAndWritesNothing(t *testing.T) {
	for name, body := range map[string]string{
		"absent":    "### `abcd capture`\n",
		"ambiguous": falseLine + "\n" + falseLine + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := armedRepo(t)
			write(t, root, ChaptersDir+"/06-capture.md", body)
			edits := []Edit{{Chapter: "06-capture.md", Sentence: falseLine, Replacement: "x."}}
			if _, err := Apply(root, edits, "c1", at); err == nil {
				t.Fatal("applied")
			}
			if _, err := os.Stat(filepath.Join(root, FlagsPath)); !os.IsNotExist(err) {
				t.Fatal("a refused apply recorded a flag")
			}
		})
	}
	t.Run("sentence spanning lines", func(t *testing.T) {
		root := armedRepo(t)
		body := "# Capture\n\n### `abcd capture`\n\nBody line one.\nBody line two.\n"
		write(t, root, ChaptersDir+"/06-capture.md", body)
		edits := []Edit{{Chapter: "06-capture.md", Sentence: "### `abcd capture`\n\nBody line one.\nBody line two.\n", Replacement: "gone"}}
		if _, err := Apply(root, edits, "c1", at); err == nil {
			t.Fatal("applied a sentence spanning lines")
		}
		if got, _ := os.ReadFile(filepath.Join(root, ChaptersDir, "06-capture.md")); string(got) != body {
			t.Fatalf("the chapter changed: %q", got)
		}
		if _, err := os.Stat(filepath.Join(root, FlagsPath)); !os.IsNotExist(err) {
			t.Fatal("a refused apply recorded a flag")
		}
	})
	t.Run("chapter outside the surfaces directory", func(t *testing.T) {
		root := armedRepo(t)
		if _, err := Apply(root, []Edit{{Chapter: "../../../README.md", Sentence: "a", Replacement: "b"}}, "c1", at); err == nil {
			t.Fatal("applied outside the chapters")
		}
	})
}
