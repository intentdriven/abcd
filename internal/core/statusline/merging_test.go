package statusline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The merging element (itd-2610150602547271, spc-2610161609350616 step 7).
// While the repository's facilitator.mode is auto, the row says merging is
// autonomous; under human, or where the mode cannot be read, it says nothing.
// The mode is read as the merge gate reads it: the default branch's committed
// .abcd/config.json and the machine layer, never the working tree's copy.

// mergingRepo is a checkout whose default branch, main, commits config as its
// .abcd/config.json ("" commits none), with HOME sandboxed so the machine
// layer is the test's own.
func mergingRepo(t *testing.T, config string) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := composeRepo(t)
	if config != "" {
		writeRepoConfig(t, root, config)
		git(t, root, "add", ".abcd/config.json")
	}
	git(t, root, "commit", "--allow-empty", "-m", "root")
	return root
}

func writeRepoConfig(t *testing.T, root, config string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMachineConfig(t *testing.T, config string) {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

const (
	autoConfig  = `{"facilitator": {"mode": "auto"}}`
	humanConfig = `{"facilitator": {"mode": "human"}}`
)

func TestTheRowSaysMergingIsAutonomousUnderAuto(t *testing.T) {
	root := mergingRepo(t, autoConfig)
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if got := plainOf(t, res.Row, KeyMerging); got != "autonomous merging" {
		t.Errorf("merging element = %q, want %q", got, "autonomous merging")
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none", res.Notes)
	}
	branch, merging := -1, -1
	for i, el := range res.Row.Elements {
		switch el.Key {
		case KeyBranch:
			branch = i
		case KeyMerging:
			merging = i
		}
	}
	if branch < 0 || merging != branch+1 {
		t.Errorf("the merging element does not follow the branch: %q", res.Row.Plain())
	}
}

func TestTheRowSaysNothingOfMergingOtherwise(t *testing.T) {
	cases := map[string]func(t *testing.T) string{
		"human committed":  func(t *testing.T) string { return mergingRepo(t, humanConfig) },
		"no mode anywhere": func(t *testing.T) string { return mergingRepo(t, "") },
		"auto only in the working tree": func(t *testing.T) string {
			root := mergingRepo(t, humanConfig)
			writeRepoConfig(t, root, autoConfig)
			return root
		},
		"human in the machine layer": func(t *testing.T) string {
			root := mergingRepo(t, autoConfig)
			writeMachineConfig(t, humanConfig)
			return root
		},
		"no default branch yet": func(t *testing.T) string {
			t.Setenv("HOME", t.TempDir())
			root := composeRepo(t)
			writeRepoConfig(t, root, autoConfig)
			return root
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := setup(t)
			res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
			if err != nil {
				t.Fatal(err)
			}
			if el, ok := res.Row.Element(KeyMerging); ok {
				t.Errorf("the row says %q: %q", el.Plain, res.Row.Plain())
			}
			if len(res.Notes) != 0 {
				t.Errorf("notes = %q, want none", res.Notes)
			}
		})
	}
}

func TestTheMergingElementIsSwitchable(t *testing.T) {
	// A mode that does not read would be a note: switched off, the mode is
	// not read at all, so there is none.
	root := mergingRepo(t, `{"facilitator": {"mode": "sometimes"}}`)
	set := Defaults()
	if !set.Enabled(KeyMerging) {
		t.Fatal("the merging element is off by default")
	}
	set.Elements[KeyMerging] = false
	res, err := ComposeContext(context.Background(), root, Payload{}, set)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("a switched-off element rendered: %q", res.Row.Plain())
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none: a switched-off element reads nothing", res.Notes)
	}

	root = mergingRepo(t, autoConfig)
	res, err = ComposeContext(context.Background(), root, Payload{}, set)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("a switched-off element rendered under auto: %q", res.Row.Plain())
	}
}

func TestAnUnreadableModeDropsTheMergingElementAndSaysSo(t *testing.T) {
	cases := map[string]string{
		"a value that is not a mode": `{"facilitator": {"mode": "sometimes"}}`,
		"a hostile key":              `{"facilitator": {"mode": "auto", "mo\u001b[31mde": "auto"}}`,
	}
	for name, config := range cases {
		t.Run(name, func(t *testing.T) {
			root := mergingRepo(t, config)
			res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
			if err != nil {
				t.Fatal(err)
			}
			if el, ok := res.Row.Element(KeyMerging); ok {
				t.Errorf("the row says %q over a mode that does not read", el.Plain)
			}
			if _, ok := res.Row.Element(KeyPresence); !ok {
				t.Errorf("the badge dropped: %q", res.Row.Plain())
			}
			if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "DROPPED the merging element") ||
				!strings.Contains(res.Notes[0], "facilitator.mode") {
				t.Fatalf("notes = %q, want one note dropping the merging element over facilitator.mode", res.Notes)
			}
			if strings.ContainsRune(res.Notes[0], 0x1b) {
				t.Errorf("the note carries a raw escape byte: %q", res.Notes[0])
			}
			if home := os.Getenv("HOME"); strings.Contains(res.Notes[0], home) || strings.Contains(res.Notes[0], root) {
				t.Errorf("the note carries a local path: %q", res.Notes[0])
			}
		})
	}
}

// TestTheMergingElementDropsWhenGitRunsOut: the mode is read with git's
// questions bounded by the row's own deadline; with that time spent the
// element drops and says why, and the row still renders.
func TestTheMergingElementDropsWhenGitRunsOut(t *testing.T) {
	root := mergingRepo(t, autoConfig)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := ComposeContext(ctx, root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("the merging element rendered with git's time spent: %q", res.Row.Plain())
	}
	if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "DROPPED the merging element") {
		t.Errorf("notes = %q, want the drop named", res.Notes)
	}
}
