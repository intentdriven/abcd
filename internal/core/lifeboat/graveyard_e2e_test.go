package lifeboat

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGraveyardPackIngestRoundTrip is the integrated M4 contract: a pack carries
// real layer-1 and layer-2 findings, a lesson citing a live finding id is
// written into the packed lifeboat, and the ingest does not perturb the pinned
// pack-time manifest (the lessons files are deliberately outside
// manifest_sha256).
func TestGraveyardPackIngestRoundTrip(t *testing.T) {
	repo := packFixture(t)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(repo, "engine.go"), []byte("package engine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-q", "-m", "add engine")
	run("revert", "--no-edit", "HEAD")

	before, err := Plan(repo)
	if err != nil {
		t.Fatalf("Plan before: %v", err)
	}
	hashBefore := ManifestSHA256(before.Files)

	dest, _ := packInto(t, repo, okScan)

	var arch Archaeology
	raw, err := os.ReadFile(filepath.Join(dest, "graveyard", "archaeology.json"))
	if err != nil {
		t.Fatalf("read packed archaeology.json: %v", err)
	}
	if err := json.Unmarshal(raw, &arch); err != nil {
		t.Fatalf("unmarshal archaeology.json: %v", err)
	}
	revID := ""
	for _, f := range arch.Findings {
		if f.Signal == SignalRevert {
			revID = f.ID
		}
	}
	if revID == "" {
		t.Fatalf("packed archaeology carries no revert finding: %+v", arch.Findings)
	}

	var aband Abandoned
	raw, err = os.ReadFile(filepath.Join(dest, "graveyard", "abandoned.json"))
	if err != nil {
		t.Fatalf("read packed abandoned.json: %v", err)
	}
	if err := json.Unmarshal(raw, &aband); err != nil {
		t.Fatalf("unmarshal abandoned.json: %v", err)
	}
	altSeen := false
	for _, f := range aband.Findings {
		if f.Signal == SignalAlternativesConsidered {
			altSeen = true
		}
	}
	if !altSeen {
		t.Fatalf("packed abandoned carries no alternatives-considered finding: %+v", aband.Findings)
	}

	payload := fmt.Sprintf(`{"schema_version":1,"lessons":[{"id":"les-engine-reverted","lesson":"The engine was added and reverted in the same breath.","confidence":"high","evidence":[%q,"rev-000000000000"]}]}`, revID)
	res, err := IngestLessons(dest, []byte(payload), okScan)
	if err != nil {
		t.Fatalf("IngestLessons: %v", err)
	}
	if res.Written != 1 || res.Dropped != 0 {
		t.Fatalf("ingest result = %+v, want 1 written 0 dropped", res)
	}
	lessons, err := os.ReadFile(filepath.Join(dest, "graveyard", "lessons.json"))
	if err != nil {
		t.Fatalf("read lessons.json: %v", err)
	}
	if !strings.Contains(string(lessons), revID) {
		t.Fatalf("lessons.json does not cite the live ref %q: %s", revID, lessons)
	}
	if strings.Contains(string(lessons), "rev-000000000000") {
		t.Fatalf("lessons.json kept a dead ref: %s", lessons)
	}

	after, err := Plan(repo)
	if err != nil {
		t.Fatalf("Plan after: %v", err)
	}
	if got := ManifestSHA256(after.Files); got != hashBefore {
		t.Fatalf("manifest hash changed across pack+ingest: %s != %s", got, hashBefore)
	}
}
