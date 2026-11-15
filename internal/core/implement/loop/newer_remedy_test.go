package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// withSourceBuild runs the test as a build of a source checkout's tree (built
// by scripts/abcd.sh, or by go run) when is holds, and as an installed release
// otherwise.
func withSourceBuild(t *testing.T, is bool) {
	t.Helper()
	was := sourceBuild
	sourceBuild = func() bool { return is }
	t.Cleanup(func() { sourceBuild = was })
}

// assertNewerRemedy holds the remedy of a newer abcd's state file to the build
// that refused it (iss-2611010041545467). A release is told to update, as
// before. A source checkout's build is never told to: `abcd update` swaps the
// installed binary and leaves the checkout's tree, and so the refusing build,
// as it was, so a person who followed it met the same refusal again. It is told
// to run the command from here with the scripts/abcd.sh of a newer checkout.
func assertNewerRemedy(t *testing.T, source bool, remedy string) {
	t.Helper()
	if !strings.Contains(remedy, "do not remove") {
		t.Errorf("the remedy keeps the newer abcd's file: %q", remedy)
	}
	if !source {
		if !strings.Contains(remedy, "`abcd update`") {
			t.Errorf("a release is told to update: %q", remedy)
		}
		return
	}
	if strings.Contains(remedy, "abcd update` brings") || !strings.Contains(remedy, "`abcd update` does not change it") {
		t.Errorf("a source build is never told that abcd update brings it up to date: %q", remedy)
	}
	for _, want := range []string{"scripts/abcd.sh", "from this directory", "git worktree add " + abcdhome.Display("worktrees") + "/", "origin/main"} {
		if !strings.Contains(remedy, want) {
			t.Errorf("a source build's remedy names %q: %q", want, remedy)
		}
	}
}

func TestANewerRunStatesRemedyFitsTheBuildThatRefusedIt(t *testing.T) {
	for name, source := range map[string]bool{"release": false, "source build": true} {
		t.Run(name, func(t *testing.T) {
			withSourceBuild(t, source)
			repo, id, _, _ := startAtValidate(t, "one.txt", "")
			editState(t, repo.Root(), id, func(doc map[string]any) {
				doc["schema_version"] = SchemaVersion + 1
			})
			_, err := ReadState(repo.Root(), id)
			r := mustRefusal(t, err)
			assertNewerRemedy(t, source, r.Remedy)
			if !strings.Contains(r.Remedy, "the run directory") {
				t.Errorf("the remedy keeps the run directory: %q", r.Remedy)
			}
		})
	}
}

func TestANewerDrainStatesRemedyFitsTheBuildThatRefusedIt(t *testing.T) {
	for name, source := range map[string]bool{"release": false, "source build": true} {
		t.Run(name, func(t *testing.T) {
			withSourceBuild(t, source)
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
				t.Fatal(err)
			}
			if err := writeDrain(root, writtenDrain()); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
			good, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(good, &doc); err != nil {
				t.Fatal(err)
			}
			doc["schema_version"] = DrainSchemaVersion + 1
			newer, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, newer, 0o600); err != nil {
				t.Fatal(err)
			}
			_, _, err = readDrain(root)
			r := mustRefusal(t, err)
			assertNewerRemedy(t, source, r.Remedy)
			if !strings.Contains(r.Remedy, "the drain state") {
				t.Errorf("the remedy keeps the drain state: %q", r.Remedy)
			}
		})
	}
}
