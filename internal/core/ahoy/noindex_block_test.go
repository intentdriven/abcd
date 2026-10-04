package ahoy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// TestSetupRefreshesTheBlocksHomePaths is D4 of itd-2610030720038073: a
// project set up before the home was renamed keeps its block, naming
// ~/.abcd/rules.json and ~/.abcd/trusted-roots, until setup runs there again.
// Detection reads the file and reports the block outdated without touching a
// byte; install then rewrites the fenced block alone, which names the renamed
// home, and every line outside it is byte-identical. The fixture is the block
// as it was planted before the change, with the owner's own text around it.
func TestSetupRefreshesTheBlocksHomePaths(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if res, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil || res.Status != "clean" {
		t.Fatalf("first install: status %q, err %v", res.Status, err)
	}
	planted, err := os.ReadFile(filepath.Join("testdata", "agents-md-planted-before-noindex.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(planted, []byte("~/.abcd/rules.json")) || !bytes.Contains(planted, []byte("~/.abcd/trusted-roots")) {
		t.Fatal("precondition: the fixture is not the block as planted before the rename")
	}
	agents := filepath.Join(repo, "AGENTS.md")
	if err := os.WriteFile(agents, planted, 0o644); err != nil {
		t.Fatal(err)
	}

	// Before setup runs again: the file is untouched, the block reported outdated.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "marker.outdated") {
		t.Fatalf("detection does not report the planted block outdated: %+v", det.Gaps)
	}
	if got, _ := os.ReadFile(agents); !bytes.Equal(got, planted) {
		t.Fatalf("detection changed AGENTS.md:\n%s", got)
	}

	// Setup runs again: the block names the renamed home, nothing else moves.
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	gotBlock := markerBlockRe.Find(got)
	if gotBlock == nil {
		t.Fatalf("no block after install:\n%s", got)
	}
	for _, want := range []string{"~/.abcd.noindex/rules.json", "~/.abcd.noindex/trusted-roots"} {
		if !bytes.Contains(gotBlock, []byte(want)) {
			t.Errorf("the refreshed block does not name %s", want)
		}
	}
	if bytes.Contains(gotBlock, []byte("~/.abcd/")) || bytes.Contains(gotBlock, []byte("~/.abcd ")) {
		t.Errorf("the refreshed block still names the old home:\n%s", gotBlock)
	}
	plantedLoc := markerBlockRe.FindIndex(planted)
	gotLoc := markerBlockRe.FindIndex(got)
	if !bytes.Equal(got[:gotLoc[0]], planted[:plantedLoc[0]]) || !bytes.Equal(got[gotLoc[1]:], planted[plantedLoc[1]:]) {
		t.Errorf("install changed a line outside the block:\nbefore:\n%s\nafter:\n%s", planted, got)
	}
}

// TestEmbeddedDefaultsNameTheNewHome: neither text the binary embeds and
// plants (the managed block, the bundled rules) names the old home, so a
// project set up after the change carries only the renamed one.
func TestEmbeddedDefaultsNameTheNewHome(t *testing.T) {
	bundled, err := json.Marshal(rules.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string][]byte{
		"the managed block": markerInner,
		"the bundled rules": bundled,
	} {
		for _, old := range []string{"~/.abcd/", "~/.abcd ", "~/.abcd`", "~/.abcd,", "~/.abcd."} {
			for i := 0; ; {
				j := bytes.Index(text[i:], []byte(old))
				if j < 0 {
					break
				}
				at := i + j
				if !bytes.HasPrefix(text[at:], []byte("~/.abcd.noindex")) {
					t.Errorf("%s names the old home at byte %d: %q", name, at, text[at:min(len(text), at+40)])
				}
				i = at + 1
			}
		}
	}
	if !bytes.Contains(markerInner, []byte("~/.abcd.noindex/rules.json")) {
		t.Error("the managed block does not name ~/.abcd.noindex/rules.json")
	}
}
