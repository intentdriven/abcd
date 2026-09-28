package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// itd-2609150819432059 decision 3: a managed repository with no artefact
// declaration is an ahoy gap that `ahoy install` prompts for and writes; a
// plugin repository adopts kind plugin without being asked.

// recordingStub answers like stubPrompter and records every prompt key asked.
type recordingStub struct {
	stubPrompter
	asked *[]string
}

func (r recordingStub) Prompt(key string, choices []string, def string) string {
	*r.asked = append(*r.asked, key)
	return r.stubPrompter.Prompt(key, choices, def)
}

func artefactRepo(t *testing.T) string {
	t.Helper()
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func artefactGap(gaps []Gap, id string) (Gap, bool) {
	for _, g := range gaps {
		if g.ID == id {
			return g, true
		}
	}
	return Gap{}, false
}

func TestDetectRaisesArtefactMissingUntilTheKindIsDeclared(t *testing.T) {
	repo := artefactRepo(t)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := artefactGap(det.Gaps, ArtefactMissingGapID)
	if !ok || !g.Required || !g.Resolvable || g.Category != ConfigChange || g.Scope != "repo" {
		t.Fatalf("gap %+v (present %v), want a required, resolvable repo config-change gap", g, ok)
	}
	artefactWrite(t, repo, launch.ArtefactRelPath, `{"kind": "binary"}`)
	det, err = Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := artefactGap(det.Gaps, ArtefactMissingGapID); ok {
		t.Error("a declared kind still raises the missing gap")
	}
}

// A declaration that is present and wrong is the user's data: a diagnostic gap
// naming the fault, which install does not overwrite.
func TestDetectReportsAnInvalidDeclarationWithoutClaimingToFixIt(t *testing.T) {
	repo := artefactRepo(t)
	artefactWrite(t, repo, launch.ArtefactRelPath, `{"kind": "wheel"}`)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := artefactGap(det.Gaps, ArtefactInvalidGapID)
	if !ok || g.Resolvable || !g.Required {
		t.Fatalf("gap %+v (present %v), want a required, non-resolvable diagnostic", g, ok)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if got := artefactRead(t, repo, launch.ArtefactRelPath); got != `{"kind": "wheel"}` {
		t.Errorf("install overwrote the user's declaration: %s", got)
	}
}

func TestInstallAdoptsKindPluginSilentlyForAPluginRepository(t *testing.T) {
	repo := artefactRepo(t)
	artefactWrite(t, repo, ".claude-plugin/plugin.json", `{"name": "fixture"}`)
	var asked []string
	p := recordingStub{stubPrompter{confirm: true}, &asked}
	if _, err := Install(repo, installOpts(), p); err != nil {
		t.Fatal(err)
	}
	art, err := launch.LoadArtefact(repo)
	if err != nil || art.Kind != launch.KindPlugin {
		t.Fatalf("declaration %+v, err %v; want kind plugin", art, err)
	}
	for _, k := range asked {
		if k == artefactKindKey {
			t.Error("a plugin repository was asked its kind")
		}
	}
}

func TestInstallPromptsForTheKindAndWritesIt(t *testing.T) {
	repo := artefactRepo(t)
	var asked []string
	p := recordingStub{stubPrompter{confirm: true, answers: map[string]string{artefactKindKey: "binary"}}, &asked}
	opts := installOpts()
	opts.Yes = false
	res, err := Install(repo, opts, p)
	if err != nil {
		t.Fatal(err)
	}
	art, err := launch.LoadArtefact(repo)
	if err != nil || art.Kind != launch.KindBinary {
		t.Fatalf("declaration %+v, err %v; want kind binary (result %+v)", art, err, res)
	}
	seen := false
	for _, k := range asked {
		seen = seen || k == artefactKindKey
	}
	if !seen {
		t.Errorf("the kind was never asked: %v", asked)
	}
	// The written file reads back through the one reader and re-running is a
	// no-op for it.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := artefactGap(det.Gaps, ArtefactMissingGapID); ok {
		t.Error("the gap survived the install that answered it")
	}
}

// An answer naming no kind — the "y" a `yes |` pipe sends — declares the
// default and says what was heard, as the house-style question does, rather
// than leaving every launch verb refusing the repository.
func TestInstallDeclaresTheDefaultOnAnAnswerOutsideTheSet(t *testing.T) {
	repo := artefactRepo(t)
	var asked []string
	p := recordingStub{stubPrompter{confirm: true, answers: map[string]string{artefactKindKey: "wheel"}}, &asked}
	opts := installOpts()
	opts.Yes = false
	res, err := Install(repo, opts, p)
	if err != nil {
		t.Fatal(err)
	}
	art, err := launch.LoadArtefact(repo)
	if err != nil || art.Kind != launch.KindApplication {
		t.Fatalf("declaration %+v, err %v; want the default, application", art, err)
	}
	noted := false
	for _, n := range res.Notes {
		noted = noted || (strings.Contains(n, `"wheel"`) && strings.Contains(n, "plugin, binary, application"))
	}
	if !noted {
		t.Errorf("the note does not say what was heard: %v", res.Notes)
	}
}

// An unattended --yes install is not asked: it declares the default and says so.
func TestInstallYesDeclaresTheDefaultWithoutAsking(t *testing.T) {
	repo := artefactRepo(t)
	var asked []string
	res, err := Install(repo, installOpts(), recordingStub{stubPrompter{confirm: true}, &asked})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range asked {
		if k == artefactKindKey {
			t.Error("--yes asked the artefact kind")
		}
	}
	art, err := launch.LoadArtefact(repo)
	if err != nil || art.Kind != launch.KindApplication {
		t.Fatalf("declaration %+v, err %v; want the default, application", art, err)
	}
	noted := false
	for _, n := range res.Notes {
		noted = noted || strings.Contains(n, "--yes")
	}
	if !noted {
		t.Errorf("the default was declared without a note: %v", res.Notes)
	}
}

func artefactWrite(t *testing.T, repo, rel, content string) {
	t.Helper()
	abs := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func artefactRead(t *testing.T, repo, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
