package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

// TestSiteVerbsReadTheCheckoutFromASubdirectory: `abcd site`, `site build` and
// `lint site` read the repository the working directory sits in, not the
// working directory itself, so run from a subdirectory they report and build
// the same site as from the root (iss-2609251750202525).
func TestSiteVerbsReadTheCheckoutFromASubdirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("AGENTS.md", "# Example\n\n<!-- BEGIN ABCD -->\nmanaged\n<!-- END ABCD -->\n")
	r.Write(".abcd/positioning.json", `{"schema_version": 1, "block": {"file": ".abcd/development/IDENTITY.md", "heading": "Identity (canonical)"}, "severity": "warn", "surfaces": []}`+"\n")
	r.Write(".abcd/development/IDENTITY.md", "# Identity\n\n## Identity (canonical)\n\n- **Title:** Example\n- **Tagline:** An example.\n")
	r.Write("docs/README.md", "# Example\n\nThe example's documentation.\n")
	r.Commit("the example")
	t.Chdir(r.Root())
	if out, err := runCLIErr(t, "--json", "site", "setup", "--name", "example-site"); err != nil {
		t.Fatalf("site setup: %v\n%s", err, out)
	}
	atRoot, err := runCLIErr(t, "--json", "site")
	if err != nil {
		t.Fatalf("site at the root: %v\n%s", err, atRoot)
	}
	t.Chdir(filepath.Join(r.Root(), "docs"))
	inSub, err := runCLIErr(t, "--json", "site")
	if err != nil {
		t.Fatalf("site in a subdirectory: %v\n%s", err, inSub)
	}
	if string(inSub) != string(atRoot) {
		t.Fatalf("the board differs by working directory:\nroot: %s\nsub:  %s", atRoot, inSub)
	}

	// The default output directory is the checkout's, wherever the build runs.
	if out, err := runCLIErr(t, "site", "build"); err != nil {
		t.Fatalf("site build in a subdirectory: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(r.Root(), "site", "index.html")); err != nil {
		t.Fatalf("the build did not land in the checkout's site directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.Root(), "docs", "site")); !os.IsNotExist(err) {
		t.Fatalf("the build wrote beside the working directory: %v", err)
	}
	if out, err := runCLIErr(t, "lint", "site"); exitCodeOf(err) == 2 {
		t.Fatalf("lint site in a subdirectory refused: %v\n%s", err, out)
	}
}

// TestSiteVerbsSayWhichLabelsTheyAdded is the TG1 ruling at the CLI: a ui.json
// lacking a declared label is completed by `site setup` and by `site build`,
// and each says so on stderr, one line per label, leaving stdout to the result.
func TestSiteVerbsSayWhichLabelsTheyAdded(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := gittest.NewRepo(t)
	r.Write("AGENTS.md", "# Example\n\n<!-- BEGIN ABCD -->\nmanaged\n<!-- END ABCD -->\n")
	r.Write(".abcd/positioning.json", `{"schema_version": 1, "block": {"file": ".abcd/development/IDENTITY.md", "heading": "Identity (canonical)"}, "severity": "warn", "surfaces": []}`+"\n")
	r.Write(".abcd/development/IDENTITY.md", "# Identity\n\n## Identity (canonical)\n\n- **Title:** Example\n- **Tagline:** An example.\n")
	r.Write("docs/README.md", "# Example\n\nThe example's documentation.\n")
	r.Commit("the example")
	t.Chdir(r.Root())
	if out, err := runCLIErr(t, "site", "setup", "--name", "example-site"); err != nil {
		t.Fatalf("site setup: %v\n%s", err, out)
	}
	uiPath := filepath.Join(r.Root(), "site-src", "ui.json")
	drop := func(label string) {
		t.Helper()
		body, err := os.ReadFile(uiPath)
		if err != nil {
			t.Fatal(err)
		}
		// The seeded line, or the last member a completion put back.
		line := "    \"" + label + "\": \"" + label + "\",\n"
		if !strings.Contains(string(body), line) {
			line = ",\n    \"" + label + "\": \"" + label + "\""
		}
		if !strings.Contains(string(body), line) {
			t.Fatalf("ui.json carries no %s label:\n%s", label, body)
		}
		if err := os.WriteFile(uiPath, []byte(strings.Replace(string(body), line, "", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want := "added the missing label status.target to site-src/ui.json with its default words\n"

	drop("target")
	out, errOut, err := runCLISplit(t, "site", "setup")
	if err != nil {
		t.Fatalf("site setup: %v\n%s%s", err, out, errOut)
	}
	if errOut != "abcd site setup: "+want {
		t.Fatalf("setup stderr = %q", errOut)
	}

	drop("target")
	out, errOut, err = runCLISplit(t, "site", "build", "--out", t.TempDir())
	if err != nil {
		t.Fatalf("site build: %v\n%s%s", err, out, errOut)
	}
	if errOut != "abcd site build: "+want {
		t.Fatalf("build stderr = %q", errOut)
	}
	if strings.Contains(out, "added the missing label") {
		t.Errorf("the added-label lines reached stdout:\n%s", out)
	}

	// A build that fails after it completed the file still says what it
	// added, before the error: the write stands, so it is never silent.
	drop("target")
	body, err := os.ReadFile(uiPath)
	if err != nil {
		t.Fatal(err)
	}
	blanked := regexp.MustCompile(`"now": "[^"]*"`).ReplaceAllString(string(body), `"now": ""`)
	if blanked == string(body) {
		t.Fatalf("ui.json carries no now label:\n%s", body)
	}
	if err := os.WriteFile(uiPath, []byte(blanked), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, err = runCLISplit(t, "site", "build", "--out", t.TempDir())
	if err == nil {
		t.Fatalf("site build with a blank label must fail:\n%s%s", out, errOut)
	}
	if !strings.HasPrefix(errOut, "abcd site build: "+want) {
		t.Fatalf("a failed build names the label it added before the error: stderr = %q, err = %v", errOut, err)
	}
}
