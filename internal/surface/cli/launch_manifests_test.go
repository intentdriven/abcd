package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lockstepTree lays a version-location contract, a primary plugin.json and a
// marketplace.json under a fresh root; an empty version omits the key.
func lockstepTree(t *testing.T, primary, market string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".abcd/config/version-location.json", `{"manifest_path": ".claude-plugin/plugin.json", "json_pointer": "/version"}`)
	pj := `{"name": "abcd"`
	if primary != "" {
		pj += `, "version": "` + primary + `"`
	}
	write(".claude-plugin/plugin.json", pj+"}")
	mk := `{"plugins": [{"name": "abcd", "source": "./"`
	if market != "" {
		mk += `, "version": "` + market + `", "changelog": {"version": "` + market + `"}`
	}
	write(".claude-plugin/marketplace.json", mk+"}]}")
	return root
}

// TestLaunchManifestsRunsTheChosenTreeOverANamedRoot: itd-69's first two
// criteria promise a checker run with `--tree public` and `--tree dev`, and a
// public checkout (a marketplace install, a release source archive) must be
// checkable from the CLI, not only by the payload render (iss-2609261423222935).
// Exit 0 consistent, 1 drift with per-field lines, 2 unreadable; no bypass flag.
func TestLaunchManifestsRunsTheChosenTreeOverANamedRoot(t *testing.T) {
	cases := []struct {
		name, tree, primary, market string
		code                        int
		drift                       bool
	}{
		{"public agreement", "public", "1.2.3", "1.2.3", 0, false},
		{"public disagreement", "public", "1.2.3", "1.2.4", 1, true},
		{"dev with a version key", "dev", "1.2.3", "", 1, true},
		{"dev with the keys absent", "dev", "", "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := lockstepTree(t, c.primary, c.market)
			out, err := runCLIErr(t, "launch", "manifests", "--tree", c.tree, "--root", root, "--json")
			if got := exitCodeOf(err); got != c.code {
				t.Fatalf("exit = %d (%v); want %d\n%s", got, err, c.code, out)
			}
			var res struct {
				Tree     string   `json:"tree"`
				OK       bool     `json:"ok"`
				Drifts   []string `json:"drifts"`
				ExitCode int      `json:"exit_code"`
			}
			if err := json.Unmarshal(out, &res); err != nil {
				t.Fatalf("output is not the result: %v\n%s", err, out)
			}
			if res.Tree != c.tree || res.ExitCode != c.code || (len(res.Drifts) > 0) != c.drift {
				t.Fatalf("result = %+v", res)
			}
		})
	}

	t.Run("unreadable contract", func(t *testing.T) {
		root := t.TempDir()
		out, err := runCLIErr(t, "launch", "manifests", "--tree", "public", "--root", root)
		if got := exitCodeOf(err); got != 2 {
			t.Fatalf("exit = %d (%v); want 2\n%s", got, err, out)
		}
	})

	t.Run("no tree is refused", func(t *testing.T) {
		_, err := runCLIErr(t, "launch", "manifests", "--root", t.TempDir())
		if got := exitCodeOf(err); got != 2 {
			t.Fatalf("exit = %d (%v); want 2", got, err)
		}
	})

	t.Run("a tree it does not know is refused", func(t *testing.T) {
		_, err := runCLIErr(t, "launch", "manifests", "--tree", "staging", "--root", t.TempDir())
		if got := exitCodeOf(err); got != 2 {
			t.Fatalf("exit = %d (%v); want 2", got, err)
		}
	})

	t.Run("no bypass flag", func(t *testing.T) {
		cmd, _, err := NewRootCommand().Find([]string{"launch", "manifests"})
		if err != nil || cmd.Name() != "manifests" {
			t.Fatalf("no manifests command: %v", err)
		}
		for _, name := range []string{"allow-dirty", "skip", "force", "dirty", "no-verify"} {
			if cmd.Flags().Lookup(name) != nil {
				t.Errorf("manifests exposes --%s, a bypass the checker must not have", name)
			}
		}
		if !strings.Contains(cmd.Flags().Lookup("tree").Usage, "public") {
			t.Error("--tree does not name its polarities")
		}
	})
}
