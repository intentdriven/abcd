package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestSiteSetupIsWiredAndReachesNoNetwork runs the verb through the CLI on a
// managed repository with no origin and no credential: the repository half is
// written, the forge is reported unreachable rather than guessed at, the host
// stage stops at the missing credential, and nothing leaves the machine.
func TestSiteSetupIsWiredAndReachesNoNetwork(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("AGENTS.md", "# Example\n\n<!-- BEGIN ABCD -->\nmanaged\n<!-- END ABCD -->\n")
	r.Write(".abcd/positioning.json", `{"schema_version": 1, "block": {"file": ".abcd/development/IDENTITY.md", "heading": "Identity (canonical)"}, "severity": "warn", "surfaces": []}`+"\n")
	r.Write(".abcd/development/IDENTITY.md", "# Identity\n\n## Identity (canonical)\n\n- **Title:** Example\n- **Tagline:** An example.\n")
	r.Write("docs/README.md", "# Example\n\nThe example's documentation.\n")
	r.Commit("the example")
	t.Chdir(r.Root())

	out, err := runCLIErr(t, "--json", "site", "setup", "--name", "example-site")
	if err != nil {
		t.Fatalf("site setup: %v\n%s", err, out)
	}
	var res struct {
		Status       string `json:"status"`
		Environments []struct {
			Status string `json:"status"`
		} `json:"environments"`
		Host struct {
			Status string `json:"status"`
		} `json:"host"`
		Remaining []string `json:"remaining"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.Status != "changed" || res.Host.Status != "no_credential" {
		t.Fatalf("status %q host %q:\n%s", res.Status, res.Host.Status, out)
	}
	for _, e := range res.Environments {
		if e.Status != "unreachable" {
			t.Fatalf("an environment is %q with no forge:\n%s", e.Status, out)
		}
	}
	for _, p := range []string{".abcd/site.json", ".github/workflows/site.yml", "wrangler.jsonc", "site-src/ui.json"} {
		if _, err := os.Stat(filepath.Join(r.Root(), p)); err != nil {
			t.Errorf("%s was not written: %v", p, err)
		}
	}
	if !strings.Contains(strings.Join(res.Remaining, "\n"), "hosting.cloudflare") {
		t.Fatalf("the remaining steps do not name the credential:\n%s", out)
	}

	// The text form of a second run says nothing changed.
	text, err := runCLIErr(t, "site", "setup")
	if err != nil {
		t.Fatalf("second run: %v\n%s", err, text)
	}
	if !strings.Contains(string(text), "abcd site setup — no_change") {
		t.Fatalf("the second run does not say it changed nothing:\n%s", text)
	}
}

func TestSiteSetupRefusesAnUnmanagedFolder(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("README.md", "# Unmanaged\n")
	r.Commit("unmanaged")
	t.Chdir(r.Root())
	out, err := runCLIErr(t, "site", "setup")
	if err == nil || !strings.Contains(string(out)+err.Error(), "not a repository abcd manages") {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if _, serr := os.Stat(filepath.Join(r.Root(), ".abcd", "site.json")); !os.IsNotExist(serr) {
		t.Fatal("a refused run wrote the composition")
	}
}

// TestSiteSetupTextAlignsEveryStatus renders a result whose statuses differ in
// length, `unrestricted` among them: every file and environment name starts in
// one column, and an environment's change lines sit under its name.
func TestSiteSetupTextAlignsEveryStatus(t *testing.T) {
	res := site.SetupResult{
		Status: site.StatusNoChange,
		Repo:   "example/site",
		Files: []scaffold.FileOutcome{
			{Path: ".abcd/site.json", Status: scaffold.StatusCurrent},
			{Path: ".github/workflows/site.yml", Status: scaffold.StatusKept},
		},
		Environments: []site.EnvironmentOutcome{
			{Name: "site-render", Status: site.RemoteCurrent},
			{Name: "site", Status: site.RemoteUnrestricted, Changes: []string{"admits every branch"}},
			{Name: "site-extra", Status: site.RemoteNotReached},
		},
		Host: site.HostOutcome{Provider: "cloudflare", Name: "example-site", Status: site.HostNoCredential},
	}
	var buf bytes.Buffer
	renderSiteSetup(&buf, res)
	names := []string{".abcd/site.json", ".github/workflows/site.yml",
		"site-render", "site", "site-extra", "admits every branch"}
	col, seen := -1, 0
	for _, line := range strings.Split(buf.String(), "\n") {
		for _, name := range names {
			if !strings.HasSuffix(line, " "+name) {
				continue
			}
			seen++
			at := len(line) - len(name)
			if col == -1 {
				col = at
			}
			if at != col {
				t.Fatalf("%q starts at column %d, not %d:\n%s", name, at, col, buf.String())
			}
			break
		}
	}
	if seen != len(names) {
		t.Fatalf("%d of %d names rendered:\n%s", seen, len(names), buf.String())
	}
}
