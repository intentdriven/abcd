package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
)

// readPreflight reads the pre-flight report a run left at dir (repo-relative).
func readPreflight(t *testing.T, root, dir string) launch.PreflightReport {
	t.Helper()
	if dir == "" {
		t.Fatal("no pre-flight report path was reported")
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dir), "preflight.json"))
	if err != nil {
		t.Fatalf("the pre-flight report was not written: %v", err)
	}
	var rep launch.PreflightReport
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatalf("the pre-flight report is not JSON: %v", err)
	}
	return rep
}

// TestLaunchDryRunWritesThePreflightReport is the wiring detector for the
// preview's half of spec piece 5: every preview leaves its report in the local
// logs tier and says where, and the documentation audit the front door measures
// reaches the gate row.
func TestLaunchDryRunWritesThePreflightReport(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write(".abcd/docs-lint.json", `{"roots": ["docs"], "banned_tokens": [], "rules": {}}`+"\n")
	r.Commit("arm docs lint")

	out, err := shipIn(t, r, "launch", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	var rep launch.DryRunReport
	if err := json.Unmarshal(out, &rep); err != nil {
		t.Fatalf("dry-run JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(rep.ReportPath, ".abcd/.work.local/logs/launch/") {
		t.Fatalf("report_path = %q, want a directory under the local logs tier (report_error %q)", rep.ReportPath, rep.ReportError)
	}
	written := readPreflight(t, r.Root(), rep.ReportPath)
	if written.Mode != launch.ModePreview || len(written.Gates) != len(rep.Gates) {
		t.Errorf("the written report does not carry the preview: %+v", written)
	}
	for _, g := range rep.Gates {
		if g.Name == "documentation-auditor" && g.Status != "ran" {
			t.Errorf("the documentation audit was not measured by the front door: %+v", g)
		}
	}

	plain, err := shipIn(t, r, "launch", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, plain)
	}
	if !strings.Contains(string(plain), "report:") || !strings.Contains(string(plain), ".abcd/.work.local/logs/launch/") {
		t.Errorf("the plain preview does not say where its report is:\n%s", plain)
	}
}

// TestLaunchShipRefusesADirtyTreeUnlessAllowed is itd-65 AC4 at the shipped
// verb: an uncommitted file refuses the cut before anything is written, naming
// the file; --allow-dirty lets it proceed, and the cut's pre-flight report
// records the override and what it carried.
func TestLaunchShipRefusesADirtyTreeUnlessAllowed(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write("notes.txt", "scratch\n")
	before := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md"))

	dest := filepath.Join(t.TempDir(), "payload")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--payload-dir", dest)
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit = %d, want 2\n%s\n%v", code, out, err)
	}
	for _, want := range []string{"uncommitted", "notes.txt", "--allow-dirty", "pre-flight report"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if after := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")); after != before {
		t.Error("a dirty-tree refusal wrote the release record anyway")
	}

	dest = filepath.Join(t.TempDir(), "payload")
	out, err = shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--payload-dir", dest, "--allow-dirty", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0 with --allow-dirty\n%s\n%v", code, out, err)
	}
	var res struct {
		Preflight    string   `json:"preflight_report"`
		AllowedDirty []string `json:"allowed_dirty"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("ship JSON: %v\n%s", err, out)
	}
	if strings.Join(res.AllowedDirty, ",") != "notes.txt" {
		t.Errorf("allowed_dirty = %v, want the carried notes.txt", res.AllowedDirty)
	}
	written := readPreflight(t, r.Root(), res.Preflight)
	if written.Mode != launch.ModeCut || !written.AllowDirty || strings.Join(written.Dirty, ",") != "notes.txt" {
		t.Errorf("the cut's report does not record the override: %+v", written)
	}
}

// TestLaunchShipAllowDirtyNeedsTheRenderPath keeps the flag honest: the
// dirty-tree gate runs on the render path, so asking for its waiver where no
// render runs is an operand error, not a flag that silently does nothing.
func TestLaunchShipAllowDirtyNeedsTheRenderPath(t *testing.T) {
	r := shipRenderableRepo(t)
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	for name, args := range map[string][]string{
		"the emit step":            {"launch", "ship", "--allow-dirty"},
		"an ingest with no render": {"launch", "ship", "--changelog-json", payload, "--allow-dirty"},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := shipIn(t, r, args...)
			if code := exitCodeOf(err); code != 2 {
				t.Fatalf("exit = %d, want 2\n%s", code, out)
			}
			if !strings.Contains(err.Error(), "--allow-dirty") || !strings.Contains(err.Error(), "renders no payload") {
				t.Errorf("the refusal should name the flag and why it has nothing to waive, got %v", err)
			}
		})
	}
}
