package site

// The TG1 ruling (the product thinker, 2026-09-30): on the next `site setup` or
// `site build`, abcd adds only the missing required labels to ui.json, with
// their default words, and never changes the project's own wording. These tests
// hold the adding to exactly that: absent declared keys gain the bundled text,
// every byte already in the file stays, and nothing else widens.

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// bundledUI is the seed ui.json abcd ships, the source of every default word.
func bundledUI(t *testing.T) string {
	t.Helper()
	data, err := setupSources.ReadFile("setupsrc/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// writeUI puts body at site-src/ui.json under a fresh root.
func writeUI(t *testing.T, body string) (root string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site-src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site-src", "ui.json"), []byte(body), 0o640); err != nil {
		t.Fatal(err)
	}
	return root
}

func readUIFile(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "site-src", "ui.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// cut removes the first occurrence of s from body, failing when it is absent.
func cut(t *testing.T, body, s string) string {
	t.Helper()
	if !strings.Contains(body, s) {
		t.Fatalf("the fixture does not carry %q", s)
	}
	return strings.Replace(body, s, "", 1)
}

// The reviewer's probe: a ui.json written before status.target existed gains
// that one line, in the file's own indentation, and nothing else moves.
func TestAMissingLabelIsAddedWithItsDefaultWords(t *testing.T) {
	full := bundledUI(t)
	// The project's own wording, and an escape the re-encoding would spell
	// differently, both of which must survive byte for byte.
	own := strings.Replace(full, `"now": "Now"`, `"now": "Right now é"`, 1)
	own = strings.Replace(own, `"nav_story": "Story"`, `"nav_story": "Our <story>"`, 1)
	old := cut(t, own, "    \"target\": \"target\",\n")
	root := writeUI(t, old)

	added, err := addMissingLabels(root, "site-src/ui.json")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !reflect.DeepEqual(added, []string{"status.target"}) {
		t.Fatalf("added %v, want [status.target]", added)
	}
	got := readUIFile(t, root)
	want := strings.Replace(old, "\"order_record_id\": \"READY intents read oldest id first\"\n",
		"\"order_record_id\": \"READY intents read oldest id first\",\n    \"target\": \"target\"\n", 1)
	if got != want {
		t.Fatalf("the file is not the old one with one line added:\n--- got\n%s\n--- want\n%s", got, want)
	}
	fi, err := os.Stat(filepath.Join(root, "site-src", "ui.json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Errorf("the file's mode is %v, want the project's 0640 kept", fi.Mode().Perm())
	}
	ui, err := LoadUI(root, "site-src/ui.json")
	if err != nil {
		t.Fatalf("the completed file does not load: %v", err)
	}
	if ui.Status.Now != "Right now é" || ui.NavStory != "Our <story>" {
		t.Errorf("the project's wording changed: now=%q nav_story=%q", ui.Status.Now, ui.NavStory)
	}

	// A second pass has nothing to add and writes nothing.
	before, _ := os.Stat(filepath.Join(root, "site-src", "ui.json"))
	again, err := addMissingLabels(root, "site-src/ui.json")
	if err != nil || len(again) != 0 {
		t.Fatalf("second pass added %v, err %v", again, err)
	}
	after, _ := os.Stat(filepath.Join(root, "site-src", "ui.json"))
	if !os.SameFile(before, after) {
		t.Error("a pass with nothing to add replaced the file")
	}
}

// A whole block missing is added whole, each of its labels named, and the file
// loads with nothing missing. A compact, one-line block keeps its shape.
func TestAMissingBlockIsAddedWholeAndACompactFileStaysCompact(t *testing.T) {
	full := bundledUI(t)
	start := strings.Index(full, ",\n  \"status\": {")
	if start < 0 {
		t.Fatal("the bundled file has no status block")
	}
	noStatus := full[:start] + "\n}\n"
	root := writeUI(t, noStatus)
	added, err := addMissingLabels(root, "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"status.now", "status.target", "status.order_record_id"} {
		if !contains(added, want) {
			t.Errorf("added %v does not name %s", added, want)
		}
	}
	if _, err := LoadUI(root, "site-src/ui.json"); err != nil {
		t.Fatalf("the completed file does not load: %v", err)
	}
	if got := readUIFile(t, root); !strings.HasPrefix(got, full[:start]) {
		t.Error("the bytes before the added block changed")
	}

	compact := `{"relations": {"blocked_by": "blocks", "supersedes": "superseded by"}, "status": {}}`
	root = writeUI(t, compact)
	added, err = addMissingLabels(root, "site-src/ui.json")
	if err != nil {
		t.Fatal(err)
	}
	got := readUIFile(t, root)
	if strings.Contains(got, "\n") {
		t.Errorf("a one-line file gained line breaks:\n%s", got)
	}
	if !strings.HasPrefix(got, `{"relations": {"blocked_by": "blocks", "supersedes": "superseded by", "implements": `) {
		t.Errorf("the compact block's own members moved:\n%s", got)
	}
	if !contains(added, "relations.implements") || !contains(added, "status.target") || !contains(added, "nav_story") {
		t.Errorf("added %v", added)
	}
	if _, err := LoadUI(root, "site-src/ui.json"); err != nil {
		t.Fatalf("the completed compact file does not load: %v", err)
	}
}

// A label the project declared, even blank, is its own: it is never rewritten,
// and the blank one is still refused by name.
func TestADeclaredLabelIsNeverRewritten(t *testing.T) {
	blank := strings.Replace(bundledUI(t), `"target": "target"`, `"target": " "`, 1)
	root := writeUI(t, blank)
	added, err := addMissingLabels(root, "site-src/ui.json")
	if err != nil || len(added) != 0 {
		t.Fatalf("added %v, err %v; want nothing", added, err)
	}
	if got := readUIFile(t, root); got != blank {
		t.Error("a declared label was rewritten")
	}
	if _, err := LoadUI(root, "site-src/ui.json"); err == nil || !strings.Contains(err.Error(), "no text for status.target") {
		t.Fatalf("a blank declared label loads: %v", err)
	}
}

// Adding applies to declared keys only: a file carrying an unknown key is left
// exactly as it is, and the closed allowlist still refuses it.
func TestAnUnknownKeyIsStillRefusedAndNothingIsAdded(t *testing.T) {
	body := cut(t, bundledUI(t), "    \"target\": \"target\",\n")
	body = strings.Replace(body, `"nav_story": "Story",`, `"nav_story": "Story", "nav_blog": "Blog",`, 1)
	root := writeUI(t, body)
	added, err := addMissingLabels(root, "site-src/ui.json")
	if err != nil || len(added) != 0 {
		t.Fatalf("added %v, err %v; want nothing", added, err)
	}
	if got := readUIFile(t, root); got != body {
		t.Error("a file carrying an unknown key was written")
	}
	if _, err := LoadUI(root, "site-src/ui.json"); err == nil || !strings.Contains(err.Error(), "nav_blog") {
		t.Fatalf("the unknown key is not refused: %v", err)
	}
}

// The adder reads the file the way the site's other reads do: a symlinked or
// non-regular ui.json is refused and nothing is written through it.
func TestASymlinkedOrNonRegularUIIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site-src"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "ui.json")
	body := cut(t, bundledUI(t), "    \"target\": \"target\",\n")
	if err := os.WriteFile(outside, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "site-src", "ui.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := addMissingLabels(root, "site-src/ui.json"); !errors.Is(err, fsutil.ErrNotRegular) {
		t.Fatalf("a symlinked ui.json: err %v, want ErrNotRegular", err)
	}
	if got, _ := os.ReadFile(outside); string(got) != body {
		t.Error("the symlink's target was written")
	}

	dirRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dirRoot, "site-src", "ui.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := addMissingLabels(dirRoot, "site-src/ui.json"); !errors.Is(err, fsutil.ErrNotRegular) {
		t.Fatalf("a directory at ui.json: err %v, want ErrNotRegular", err)
	}
}

// The bundled file is the source of every default word, so it must declare
// every label: a default that is blank would add a refusal, not a word.
func TestTheBundledUIDeclaresEveryLabel(t *testing.T) {
	root := writeUI(t, bundledUI(t))
	if _, err := LoadUI(root, "site-src/ui.json"); err != nil {
		t.Fatal(err)
	}
}

// `site build` completes an older file before it loads it: the reviewer's
// probe builds, and the build says which label it added.
func TestBuildAddsAMissingLabelAndSaysSo(t *testing.T) {
	f := newFixture(t)
	body, err := os.ReadFile(filepath.Join(f.Root(), "site-src", "ui.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.write("site-src/ui.json", cut(t, string(body), `"target": "target", `))
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
	if err != nil {
		t.Fatalf("a ui.json without status.target does not build: %v", err)
	}
	if !reflect.DeepEqual(res.AddedLabels, []string{"status.target"}) {
		t.Fatalf("build added %v, want [status.target]", res.AddedLabels)
	}
	if _, err := LoadUI(f.Root(), "site-src/ui.json"); err != nil {
		t.Fatalf("the file build completed does not load: %v", err)
	}
}

// A build that fails after it completed the file carries the labels it added
// in its error, which unwraps to the cause.
func TestAFailedBuildNamesTheLabelsItAdded(t *testing.T) {
	f := newFixture(t)
	body, err := os.ReadFile(filepath.Join(f.Root(), "site-src", "ui.json"))
	if err != nil {
		t.Fatal(err)
	}
	cutBody := cut(t, string(body), `"target": "target", `)
	blanked := regexp.MustCompile(`"now": "[^"]*"`).ReplaceAllString(cutBody, `"now": ""`)
	if blanked == cutBody {
		t.Fatal("fixture: ui.json carries no now label")
	}
	f.write("site-src/ui.json", blanked)
	_, err = Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
	var added *LabelsAddedError
	if !errors.As(err, &added) || !reflect.DeepEqual(added.Labels, []string{"status.target"}) || added.File != "site-src/ui.json" {
		t.Fatalf("the failed build names the label it added: %v", err)
	}
	if errors.Unwrap(err) == nil || err.Error() != errors.Unwrap(err).Error() {
		t.Fatalf("the error unwraps to its cause and reads as it: %v", err)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// `site setup` keeps a ui.json the repository already has, and completes it:
// the missing label is added, the file is reported written with the label
// named, and the commit step names it.
func TestSetupAddsAMissingLabelToAKeptUI(t *testing.T) {
	h := newHarness(t)
	h.run(t)
	path := filepath.Join(h.repo.Root(), "site-src", "ui.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	own := strings.Replace(cut(t, string(body), "    \"target\": \"target\",\n"), `"now": "Now"`, `"now": "Today"`, 1)
	if err := os.WriteFile(path, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}

	res := h.run(t)
	if !reflect.DeepEqual(res.AddedLabels, []string{"status.target"}) {
		t.Fatalf("setup added %v, want [status.target]", res.AddedLabels)
	}
	if got := fileStatuses(res)["site-src/ui.json"]; got != "written" {
		t.Errorf("site-src/ui.json is %q, want written", got)
	}
	if !strings.Contains(strings.Join(res.Remaining, "\n"), "site-src/ui.json") {
		t.Errorf("the commit step does not name the completed file: %v", res.Remaining)
	}
	ui, err := LoadUI(h.repo.Root(), "site-src/ui.json")
	if err != nil {
		t.Fatalf("the completed file does not load: %v", err)
	}
	if ui.Status.Now != "Today" || ui.Status.Target != "target" {
		t.Errorf("now=%q target=%q", ui.Status.Now, ui.Status.Target)
	}
}

// The gate over a rendered site writes only inside its output directory, the
// render it makes when the directory is empty included: it never completes the
// repository's ui.json, and an older file is refused by name there, as before.
func TestTheGatesRenderLeavesTheUIAlone(t *testing.T) {
	f := newFixture(t)
	body, err := os.ReadFile(filepath.Join(f.Root(), "site-src", "ui.json"))
	if err != nil {
		t.Fatal(err)
	}
	old := cut(t, string(body), `"target": "target", `)
	f.write("site-src/ui.json", old)
	_, err = Check(CheckRequest{RepoRoot: f.Root(), OutDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no text for status.target") {
		t.Fatalf("the gate's render: err %v, want the missing label named", err)
	}
	if got := readUIFile(t, f.Root()); got != old {
		t.Error("the gate's render wrote the repository's ui.json")
	}
}
