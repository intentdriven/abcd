package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec close names every link it repointed at the moved records, in the text
// render and in --json, so the operator sees what the close changed beyond the
// two records it moved (iss-2609091732329046).
func TestSpecCloseReportsRepointedLinks(t *testing.T) {
	setup := func(t *testing.T) string {
		repo := t.TempDir()
		gitInitAt(t, repo)
		t.Chdir(repo)
		writeRepoFile(t, repo, cliPlanned+"/itd-10-alpha.md",
			"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
		writeRepoFile(t, repo, cliSpecsOpen+"/spc-1-alpha.md",
			"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
		writeRepoFile(t, repo, ".abcd/development/plans/p.md",
			"# p\n\n[itd-10](../intents/planned/itd-10-alpha.md)\n")
		return repo
	}

	t.Run("text", func(t *testing.T) {
		setup(t)
		out := string(runCLI(t, "spec", "close", "spc-1"))
		if !strings.Contains(out, "repointed 1 link") ||
			!strings.Contains(out, ".abcd/development/plans/p.md:3  ../intents/planned/itd-10-alpha.md -> ../intents/shipped/itd-10-alpha.md") {
			t.Fatalf("close text must name the repointed link:\n%s", out)
		}
	})

	t.Run("json", func(t *testing.T) {
		setup(t)
		var got struct {
			Relinked []struct {
				File string `json:"file"`
				Line int    `json:"line"`
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"relinked"`
		}
		out := runCLI(t, "spec", "close", "spc-1", "--json")
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("not JSON: %v\n%s", err, out)
		}
		if len(got.Relinked) != 1 || got.Relinked[0].To != "../intents/shipped/itd-10-alpha.md" {
			t.Fatalf("relinked = %+v\n%s", got.Relinked, out)
		}
	})
}

// capture resolve names the links it repointed at the issue's new folder.
func TestCaptureResolveReportsRepointedLinks(t *testing.T) {
	repo := captureLedgerRepo(t)
	capOut := runCLI(t, "capture", "an issue a decision links to", "--json")
	var minted struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(capOut, &minted); err != nil || minted.ID == "" {
		t.Fatalf("capture envelope unreadable: %v\n%s", err, capOut)
	}
	adr := ".abcd/development/decisions/adrs/0001-x.md"
	if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(adr)), 0o755); err != nil {
		t.Fatal(err)
	}
	link := "../../../work/issues/open/" + filepath.Base(minted.Path)
	if err := os.WriteFile(filepath.Join(repo, adr), []byte("# x\n\n[it]("+link+")\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := string(runCLI(t, "capture", "resolve", minted.ID, "fixed", "--impact", "fix", "--grounds", cliGrounds))
	want := adr + ":3  " + link + " -> ../../../work/issues/resolved/" + filepath.Base(minted.Path)
	if !strings.Contains(out, "repointed 1 link") || !strings.Contains(out, want) {
		t.Fatalf("resolve text must name the repointed link %q:\n%s", want, out)
	}
}
