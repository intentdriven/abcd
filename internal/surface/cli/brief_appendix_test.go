package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// TestSurfaceAppendicesMatchCommandTree is the drift gate for the brief's
// surface chapters (itd-147 ac-1): it regenerates every chapter's appendix from
// the live command tree and fails naming the chapter and each claim that
// differs. Regenerate with `go generate ./internal/surface/cli`.
func TestSurfaceAppendicesMatchCommandTree(t *testing.T) {
	chapters, err := SurfaceChapters(testRepoRoot())
	if err != nil {
		// A refused chapter is named; every other chapter is still checked below.
		t.Errorf("cannot regenerate every surface chapter: %v", err)
	}
	if len(chapters) == 0 {
		t.Fatal("no surface chapters found; the drift gate would pass vacuously")
	}
	for _, ch := range chapters {
		if d := ch.Drift(); d != "" {
			t.Error(d)
		}
	}
}

// TestSurfaceChapterProseStatesNoShape is the check that keeps the retired shape
// prose from coming back (itd-147 ac-5): the hand-written region above each
// chapter's opening marker states no flag and no sub-verb, because those live
// only in the generated appendix.
//
// It skips under -race: one goroutine matching every sub-verb path against
// every chapter took 12.8s instrumented against 0.6s uninstrumented, with
// nothing for the detector to watch (iss-2609261924541555). The uninstrumented
// lane asserts it on both CI legs.
func TestSurfaceChapterProseStatesNoShape(t *testing.T) {
	if raceEnabled {
		t.Skip("one goroutine over the committed chapters gains nothing under -race; the uninstrumented run asserts it")
	}
	chapters, err := SurfaceChapters(testRepoRoot())
	if err != nil {
		t.Errorf("cannot read every surface chapter: %v", err)
	}
	tree := commandSurface(NewRootCommand())
	for _, ch := range chapters {
		prose, err := surface.SplitChapter(ch.Committed)
		if err != nil {
			t.Errorf("%s: %v", ch.File, err)
			continue
		}
		for _, c := range surface.ProseShapeClaims(prose, ch.Commands, tree) {
			t.Errorf("%s/%s:%d: the prose above the appendix states %s; shape lives only in the generated appendix",
				surface.BriefSurfacesDir, ch.File, c.Line, c.Spelling)
		}
	}
}

// fixtureCobraTree is a two-level tree carrying metadata a generator must NOT
// read: an exit-code annotation, a JSON schema in the long help, and an example.
func fixtureCobraTree() *cobra.Command {
	root := &cobra.Command{Use: "abcd"}
	root.PersistentFlags().Bool("json", false, "")
	verb := &cobra.Command{
		Use:         "widget",
		Long:        "Exit codes: 0 ok, 3 refused. JSON output: {\"schema_version\":1,\"widgets\":[]}",
		Example:     "abcd widget --size 3",
		Annotations: map[string]string{"exit_codes": "0,3", "json_schema": "widgets.v1"},
		Run:         func(*cobra.Command, []string) {},
	}
	verb.Flags().Int("size", 0, "the widget size (exit 3 when negative)")
	sub := &cobra.Command{Use: "polish", Run: func(*cobra.Command, []string) {}}
	verb.AddCommand(sub)
	root.AddCommand(verb)
	return root
}

// ac-1 and ac-4 over a real cobra tree: the walk the generator uses carries the
// flags and sub-verbs and nothing else, and a flag added to the tree without a
// regeneration is a drift naming the chapter and the claim.
func TestSurfaceAppendixFromCobraTree(t *testing.T) {
	tree := commandSurface(fixtureCobraTree())
	appendix := surface.ComposeAppendix([]string{"abcd widget"}, tree)
	for _, w := range []string{"### `abcd widget`", "Sub-verbs: `abcd widget polish`.", "| `--size` | int |", "### `abcd widget polish`"} {
		if !strings.Contains(appendix, w) {
			t.Errorf("appendix lacks %q:\n%s", w, appendix)
		}
	}
	for _, n := range []string{"Exit codes:", "3 refused", "schema_version", "widgets.v1", "0,3", "--size 3", "size (exit"} {
		if strings.Contains(appendix, n) {
			t.Errorf("appendix carries %q, which is neither a flag nor a sub-verb:\n%s", n, appendix)
		}
	}

	committed, err := surface.RenderChapter("# Widget\n\n"+surface.AppendixBegin+"\n"+surface.AppendixEnd+"\n", appendix)
	if err != nil {
		t.Fatal(err)
	}
	grown := fixtureCobraTree()
	for _, c := range grown.Commands() {
		if c.Name() == "widget" {
			c.Flags().String("colour", "", "")
		}
	}
	want, err := surface.RenderChapter(committed, surface.ComposeAppendix([]string{"abcd widget"}, commandSurface(grown)))
	if err != nil {
		t.Fatal(err)
	}
	d := surface.RegeneratedChapter{Chapter: surface.Chapter{File: "99-widget.md"}, Committed: committed, Want: want}.Drift()
	for _, w := range []string{"99-widget.md", "missing: | `--colour` | string |"} {
		if !strings.Contains(d, w) {
			t.Errorf("drift message lacks %q:\n%s", w, d)
		}
	}
}

// TestShippedChapterNeverClaimsNoShippedSurface: a chapter whose register row
// reads shipped never states there is no shipped surface (iss-2609231931006041).
// The host-delegated commands ship as command pages with no Go verb, and the
// committed appendix of each once said the opposite of its own register row.
func TestShippedChapterNeverClaimsNoShippedSurface(t *testing.T) {
	dir := filepath.Join(testRepoRoot(), filepath.FromSlash(surface.BriefSurfacesDir))
	register, err := os.ReadFile(filepath.Join(dir, surface.RegisterFile))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, row := range surface.ParseRegister(string(register)) {
		if row.Status != "shipped" || row.Chapter == "" {
			continue
		}
		text, err := os.ReadFile(filepath.Join(dir, row.Chapter))
		if err != nil {
			t.Fatal(err)
		}
		checked++
		if strings.Contains(string(text), "There is no shipped surface") {
			t.Errorf("%s/%s: register row %d reads shipped, but the chapter says there is no shipped surface",
				surface.BriefSurfacesDir, row.Chapter, row.Line)
		}
	}
	if checked == 0 {
		t.Fatal("no shipped register rows found; the check would pass vacuously")
	}
}
