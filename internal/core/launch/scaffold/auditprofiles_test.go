package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestWriteAuditProfilesWritesEveryProfile holds the renderer CI's workflow
// audit reads (iss-2609251939472371): every profile the in-repo audit
// renders is written, both workflows of each, byte for byte as Render
// produces them, under <dir>/<profile>/.github/workflows/.
func TestWriteAuditProfilesWritesEveryProfile(t *testing.T) {
	dir := t.TempDir()
	written, err := WriteAuditProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	profiles := AuditProfiles()
	if len(profiles) < 2 || len(written) != 2*len(profiles) {
		t.Fatalf("wrote %d file(s) for %d profile(s); want two per profile", len(written), len(profiles))
	}
	for name, subs := range profiles {
		r, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		for file, want := range map[string][]byte{"release.yml": r.ReleaseYML, "auto-release.yml": r.AutoReleaseYML} {
			got, err := os.ReadFile(filepath.Join(dir, name, ".github", "workflows", file))
			if err != nil {
				t.Fatalf("%s/%s was not written: %v", name, file, err)
			}
			if string(got) != string(want) {
				t.Errorf("%s/%s differs from what Render produces", name, file)
			}
		}
	}
}

// TestCIAuditsEveryScaffoldedProfile is the other half: the in-repo audit
// (TestScaffoldedWorkflowsPassTheWorkflowAudit) covers injection and duplicate
// keys only, and CI's zizmor job audited the committed workflows alone, which
// are the abcd profile. So the job renders every profile with
// cmd/scaffold-render and runs the same pinned zizmor over the result, which
// brings action pinning, permissions and credential handling of the profiles a
// managed repository receives under the audit.
func TestCIAuditsEveryScaffoldedProfile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var job []string
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "  zizmor:" {
			in = true
			continue
		}
		if in && regexp.MustCompile(`^  ([#]|[a-z][a-z0-9_-]*:\s*$)`).MatchString(line) {
			break
		}
		if in {
			job = append(job, line)
		}
	}
	body := strings.Join(job, "\n")
	if body == "" {
		t.Fatal("ci.yml has no zizmor job")
	}
	render := regexp.MustCompile(`go run \./cmd/scaffold-render "\$RUNNER_TEMP/scaffold-profiles"`)
	if !render.MatchString(body) {
		t.Errorf("the zizmor job does not render the scaffolded profiles with cmd/scaffold-render:\n%s", body)
	}
	if n := strings.Count(body, "ghcr.io/zizmorcore/zizmor@sha256:"); n != 2 {
		t.Errorf("the zizmor job runs the pinned zizmor image %d time(s); want two, the committed workflows and the rendered profiles", n)
	}
	if !strings.Contains(body, `"$RUNNER_TEMP/scaffold-profiles:/profiles:ro"`) {
		t.Errorf("the zizmor job does not audit the rendered profiles' directory:\n%s", body)
	}
}
