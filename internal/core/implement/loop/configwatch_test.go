package loop

// configwatch_test.go is the run tripwire (iss-2610111600058575): a program
// the repository's shared git configuration names runs on abcd's git steps as
// it runs on the person's own, so the loop keeps those settings as they are and
// instead notices when they change during a run, stopping before its next git
// step until the person accepts the change.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// fingerprintPath is the run's tripwire file.
func fingerprintPath(repo *gittest.Repo, runID string) string {
	return filepath.Join(repo.Root(), filepath.FromSlash(runRel(runID)), ConfigFingerprintFileName)
}

// markerFilter plants, in the repository's shared config, a content filter
// named planted whose clean command leaves a marker file, and an attribute in
// worktree that routes every path through it. It returns the marker's path.
func markerFilter(t *testing.T, repo *gittest.Repo, worktree string) string {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "filter-ran")
	repo.Git("config", "filter.planted.clean", "touch '"+marker+"'; cat")
	if err := os.WriteFile(filepath.Join(worktree, ".gitattributes"), []byte("* filter=planted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return marker
}

// absent fails the test when path exists.
func absent(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s (%v)", what, err)
	}
}

// startedRun is a run started and stepped once, its first stage taken, so
// the tripwire has recorded its fingerprint.
func startedRun(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	return repo, start.RunID
}

// TestARunWithoutAFingerprintRecordsOneAndProceeds: a run with no fingerprint
// (every run started before the tripwire) is never refused for it, even with
// a program-naming setting already in place: its next step records one and
// takes its stage. Once recorded, an unchanged repository sees nothing: the
// steps after it move and the file is not rewritten.
func TestARunWithoutAFingerprintRecordsOneAndProceeds(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	fp := fingerprintPath(repo, start.RunID)
	_ = os.Remove(fp)
	repo.Git("config", "filter.lfs.clean", "git-lfs clean -- %f")
	repo.Git("config", "credential.helper", "store")

	res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("a run without a fingerprint is never refused for it: %v", err)
	}
	if res.PerformedStage != StageWorktree {
		t.Fatalf("the step took its stage: %+v", res)
	}
	first, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("the step recorded the run's fingerprint beside its state: %v", err)
	}
	info, err := os.Stat(fp)
	if err != nil || info.Mode().Perm() != filePerm {
		t.Fatalf("the fingerprint is the caller's own, mode %v: %v", info.Mode().Perm(), err)
	}

	for range 2 {
		res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{})
		if err != nil {
			t.Fatalf("an unchanged repository sees nothing: %v", err)
		}
		if strings.Contains(res.Next, "configuration") {
			t.Fatalf("an unchanged repository is not mentioned: %s", res.Next)
		}
	}
	if again, _ := os.ReadFile(fp); string(again) != string(first) {
		t.Fatalf("an unchanged check does not rewrite the fingerprint:\n%s\n---\n%s", first, again)
	}
}

// TestAFilterPlantedMidRunStopsTheRestartBeforeGit: a content filter planted
// in the shared config after the fingerprint was recorded stops a restart
// before its snapshot, whose add would have run it: the refusal names the key
// and the accepting command, the filter never ran, nothing is saved aside and
// the dead agent's edits are where it left them.
func TestAFilterPlantedMidRunStopsTheRestartBeforeGit(t *testing.T) {
	repo, runID, l, dir, _ := deadLane(t)
	marker := markerFilter(t, repo, l.Worktree)
	before := stateBytes(t, repo.Root(), runID)

	_, err := Restart(repo.Root(), runID, "lane-1", "", Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "filter.planted.clean") || !strings.Contains(r.Remedy, AcceptConfigFlag) {
		t.Fatalf("the restart is refused naming the planted key and the accepting flag: %+v", r)
	}
	absent(t, marker, "the planted filter never ran")
	noAside(t, dir)
	editsIntact(t, repo, l)
	if after := stateBytes(t, repo.Root(), runID); string(after) != string(before) {
		t.Fatal("a refused restart leaves the run's state as it was")
	}
}

// TestAHookPlantedMidRunStopsTheNextStep: a hook file added to the
// repository's hooks directory, and a core.hooksPath set, after the
// fingerprint was recorded each stop the next step before its stage: the
// refusal names the file or the key, and the state is as it was.
func TestAHookPlantedMidRunStopsTheNextStep(t *testing.T) {
	for _, tc := range []struct {
		name  string
		plant func(t *testing.T, repo *gittest.Repo) string
	}{
		{"a new hook file", func(t *testing.T, repo *gittest.Repo) string {
			hook := filepath.Join(repo.Root(), ".git", "hooks", "post-checkout")
			if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch hook-ran\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			return "post-checkout"
		}},
		{"core.hooksPath", func(t *testing.T, repo *gittest.Repo) string {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "pre-commit"), []byte("#!/bin/sh\ntouch hook-ran\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			repo.Git("config", "core.hooksPath", dir)
			return "core.hookspath"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, runID := startedRun(t)
			want := tc.plant(t, repo)
			before := stateBytes(t, repo.Root(), runID)

			_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageConfig || !strings.Contains(r.Reason, want) || !strings.Contains(r.Remedy, AcceptConfigFlag) {
				t.Fatalf("the step is refused naming %q and the accepting flag: %+v", want, r)
			}
			if after := stateBytes(t, repo.Root(), runID); string(after) != string(before) {
				t.Fatal("the refused step took no stage")
			}
		})
	}
}

// TestAcceptingTheChangeResumesTheRun: once the person has looked, accepting
// re-records the fingerprint, names what it accepted in the run record, and
// the run moves again; a second accept with nothing changed accepts nothing.
func TestAcceptingTheChangeResumesTheRun(t *testing.T) {
	repo, runID := startedRun(t)
	repo.Git("config", "filter.lfs.clean", "git-lfs clean -- %f")
	if _, err := advance(repo.Root(), runID, DefaultStages(), Options{}); err == nil {
		t.Fatal("the changed configuration stops the step")
	}

	accepted, err := AcceptConfigChange(repo.Root(), runID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 1 || !strings.Contains(accepted[0], "filter.lfs.clean") {
		t.Fatalf("the accept names what it accepted: %q", accepted)
	}
	notes := entries(func() State { st, _ := ReadState(repo.Root(), runID); return st }(), StageConfig)[""]
	if len(notes) != 1 || !strings.Contains(notes[0], "filter.lfs.clean") {
		t.Fatalf("the run record names the accepted change: %q", notes)
	}
	if _, err := advance(repo.Root(), runID, DefaultStages(), Options{}); err != nil {
		t.Fatalf("the run moves once the change is accepted: %v", err)
	}
	if again, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil || len(again) != 0 {
		t.Fatalf("nothing changed, nothing is accepted: %q %v", again, err)
	}
}

// TestACredentialHelperValueIsNeverShown: a credential helper's value may
// carry a secret, so neither the stored fingerprint, the refusal (text and
// JSON), the accept's answer nor the run record ever holds it; they name the
// key alone.
func TestACredentialHelperValueIsNeverShown(t *testing.T) {
	const one, two = "s3cr3t-token-one", "s3cr3t-token-two"
	repo := briefRepo(t, agentsMarked)
	repo.Git("config", "credential.helper", "!echo password="+one)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "credential.helper", "!echo password="+two)

	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "credential.helper") {
		t.Fatalf("the refusal names the key: %+v", r)
	}
	doc, _ := json.Marshal(r)
	accepted, err := AcceptConfigChange(repo.Root(), start.RunID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := os.ReadFile(fingerprintPath(repo, start.RunID))
	for what, text := range map[string]string{
		"the refusal": r.Error(), "the refusal's JSON": string(doc), "the stored fingerprint": string(stored),
		"the accept's answer": strings.Join(accepted, " "), "the run's state": string(stateBytes(t, repo.Root(), start.RunID)),
	} {
		if strings.Contains(text, one) || strings.Contains(text, two) || strings.Contains(text, "password=") {
			t.Fatalf("%s holds the helper's value:\n%s", what, text)
		}
	}
}

// TestTheProbeStopsWhenTransportSettingsChange: a remote's upload-pack
// program planted mid-run stops the shared network probe a step finds due,
// before it reaches the remote: the probe is never run.
func TestTheProbeStopsWhenTransportSettingsChange(t *testing.T) {
	f := newParFixture(t, "", Options{Session: "host-a"})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	run := f.shared(t)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-1", "git fetch"); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	f.repo.Git("config", "remote.origin.uploadpack", "touch probe-ran; git-upload-pack")

	calls := 0
	o := f.opts()
	o.Session = "host-a"
	o.NetworkProbe = func() (bool, string) { calls++; return true, "origin answered" }
	_, err := outageFor(f.repo.Root(), f.runID, o)
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "remote.origin.uploadpack") || calls != 0 {
		t.Fatalf("the due probe is stopped before it runs (calls %d): %+v", calls, r)
	}
	if cur := outageNow(t, run); cur == nil || len(cur.Probes) != 0 {
		t.Fatalf("no probe is recorded: %+v", cur)
	}
}

// TestARateLimitCheckpointKeepsTheLaneWhenTheConfigurationChanged: the
// rate-limit checkpoint saves a lane aside and resets it with git; a filter
// planted mid-run keeps the lane as it is instead, the checkpoint naming why,
// and the filter never runs.
func TestARateLimitCheckpointKeepsTheLaneWhenTheConfigurationChanged(t *testing.T) {
	f := newParFixture(t, "", Options{})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	l1 := f.lane(t, "lane-1")
	laneCommit(t, f.repo, l1, "one.txt")
	if err := os.WriteFile(filepath.Join(l1.Worktree, "wip.txt"), []byte("at work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	marker := markerFilter(t, f.repo, l1.Worktree)

	res, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", l1.Awaits[0],
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "rate limited"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	if res.RateLimit == nil || len(res.RateLimit.Checkpoints) != 1 || !strings.Contains(res.RateLimit.Checkpoints[0].Kept, "filter.planted.clean") {
		t.Fatalf("the checkpoint keeps the lane, naming the change: %+v", res.RateLimit)
	}
	absent(t, marker, "the planted filter never ran")
	if _, err := os.Stat(filepath.Join(l1.Worktree, "wip.txt")); err != nil {
		t.Fatalf("the lane's uncommitted work is left where it is: %v", err)
	}
	if l := f.lane(t, "lane-1"); len(l.Awaits) != 1 {
		t.Fatalf("the lane keeps its await: %+v", l.Awaits)
	}
}

// TestACoreWorktreePlantedMidRunStopsTheNextStep: core.worktree set after the
// fingerprint was recorded stops the next step before its stage, whether it
// is set in the shared config or, with extensions.worktreeConfig on, in the
// lane worktree's own config.worktree, which git reads only there; turning
// extensions.worktreeConfig on mid-run is a change in its own right.
func TestACoreWorktreePlantedMidRunStopsTheNextStep(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		before     func(repo *gittest.Repo)
		plant      func(t *testing.T, repo *gittest.Repo, lane string)
	}{
		{"in the shared config", "core.worktree", nil, func(t *testing.T, repo *gittest.Repo, _ string) {
			repo.Git("config", "core.worktree", t.TempDir())
		}},
		{"in the lane's per-worktree config", "core.worktree in the worktree",
			func(repo *gittest.Repo) { repo.Git("config", "extensions.worktreeConfig", "true") },
			func(t *testing.T, repo *gittest.Repo, lane string) {
				repo.Git("-C", lane, "config", "--worktree", "core.worktree", t.TempDir())
			}},
		{"extensions.worktreeConfig turned on", "extensions.worktreeconfig", nil, func(t *testing.T, repo *gittest.Repo, _ string) {
			repo.Git("config", "extensions.worktreeConfig", "true")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := briefRepo(t, agentsMarked)
			if tc.before != nil {
				tc.before(repo)
			}
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
				t.Fatal(err)
			}
			st, err := ReadState(repo.Root(), start.RunID)
			if err != nil || len(st.Lanes) != 1 || st.Lanes[0].Worktree == "" {
				t.Fatalf("the first step made the lane's worktree: %+v %v", st.Lanes, err)
			}
			tc.plant(t, repo, st.Lanes[0].Worktree)
			before := stateBytes(t, repo.Root(), start.RunID)

			_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageConfig || !strings.Contains(r.Reason, tc.want) {
				t.Fatalf("the step is refused naming %q: %+v", tc.want, r)
			}
			if after := stateBytes(t, repo.Root(), start.RunID); string(after) != string(before) {
				t.Fatal("the refused step took no stage")
			}
		})
	}
}

// laneRun is a run started and stepped once with before applied to the
// repository first: the fingerprint is recorded and the lane's worktree made.
// It returns the lane's worktree.
func laneRun(t *testing.T, before func(repo *gittest.Repo)) (*gittest.Repo, string, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	if before != nil {
		before(repo)
	}
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil || len(st.Lanes) != 1 || st.Lanes[0].Worktree == "" {
		t.Fatalf("the first step made the lane's worktree: %+v %v", st.Lanes, err)
	}
	return repo, start.RunID, st.Lanes[0].Worktree
}

// notConfigRefused fails the test when err is the tripwire's refusal.
func notConfigRefused(t *testing.T, err error, what string) {
	t.Helper()
	if r, ok := AsRefusal(err); ok && r.Stage == StageConfig {
		t.Fatalf("%s: %+v", what, r)
	}
}

// TestARelativeHooksPathIsWatchedInEachLane: git resolves a relative
// core.hooksPath against the working tree it runs in, so in a lane it runs the
// lane's own copy of the hooks. A hook edited in the lane's copy mid-run stops
// the next step, naming the file; the copy as the lane was made is no change.
func TestARelativeHooksPathIsWatchedInEachLane(t *testing.T) {
	repo, runID, lane := laneRun(t, func(repo *gittest.Repo) {
		repo.Write(".githooks/pre-commit", "#!/bin/sh\nexit 0\n")
		repo.Commit("hooks")
		repo.Git("config", "core.hooksPath", ".githooks")
	})
	_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	notConfigRefused(t, err, "the lane's hooks as it was made are no change")

	hook := filepath.Join(lane, ".githooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch hook-ran\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), runID)
	_, err = advance(repo.Root(), runID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "pre-commit") {
		t.Fatalf("the step is refused naming the lane's hook: %+v", r)
	}
	if after := stateBytes(t, repo.Root(), runID); string(after) != string(before) {
		t.Fatal("the refused step took no stage")
	}
}

// TestALaneOnlyIncludeIsWatched: a conditional include that applies only in
// the lanes adds settings git reads there and not in the checkout; a watched
// key it gains mid-run stops the next step, naming the key in the worktree.
func TestALaneOnlyIncludeIsWatched(t *testing.T) {
	inc := filepath.Join(t.TempDir(), "lane.inc")
	if err := os.WriteFile(inc, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	repo, runID, lane := laneRun(t, func(repo *gittest.Repo) {
		repo.Git("config", "includeIf.gitdir:**/.git/worktrees/**.path", inc)
	})
	if err := os.WriteFile(inc, []byte("[core]\n\tsshCommand = touch planted; ssh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := repo.Git("-C", lane, "config", "--get", "core.sshCommand"); got == "" {
		t.Fatal("the include applies in the lane")
	}
	_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "core.sshcommand in the worktree") {
		t.Fatalf("the step is refused naming the key the lane gained: %+v", r)
	}
}

// TestMoreProgramNamingKeysAreWatched: an askpass program, a remote's URL
// and an ssh signing key command each stop the next step when set mid-run.
func TestMoreProgramNamingKeysAreWatched(t *testing.T) {
	for _, tc := range []struct{ key, value, want string }{
		{"core.askPass", "touch planted", "core.askpass"},
		{"remote.origin.url", "ext::touch planted", "remote.origin.url"},
		{"gpg.ssh.defaultKeyCommand", "touch planted", "gpg.ssh.defaultkeycommand"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			repo, runID := startedRun(t)
			repo.Git("config", tc.key, tc.value)
			_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageConfig || !strings.Contains(r.Reason, tc.want) {
				t.Fatalf("the step is refused naming %q: %+v", tc.want, r)
			}
		})
	}
}

// TestAKeyNameNeverCarriesItsSecret: a URL rewrite or a credential context
// names a URL in the key itself, which may carry a token. The change is still
// refused and named, by a placeholder and a short id that tells the two keys
// apart, and the token is in neither the stored fingerprint, the refusal (text
// and JSON), the accept's answer nor the run's state.
func TestAKeyNameNeverCarriesItsSecret(t *testing.T) {
	token, pw := testsecret.Synthetic(61, 32), testsecret.Synthetic(62, 24)
	repo, runID := startedRun(t)
	repo.Git("config", "url.https://x-access-token:"+token+"@github.com/.insteadOf", "https://github.com/")
	repo.Git("config", "credential.https://user:"+pw+"@example.invalid.helper", "store")

	_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "url.<hidden") || !strings.Contains(r.Reason, ".insteadof (added)") ||
		!strings.Contains(r.Reason, "credential.<hidden") || !strings.Contains(r.Reason, ".helper (added)") {
		t.Fatalf("the refusal names both keys, hidden: %+v", r)
	}
	doc, _ := json.Marshal(r)
	accepted, err := AcceptConfigChange(repo.Root(), runID, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 2 || accepted[0] == accepted[1] {
		t.Fatalf("the accept names both keys apart: %q", accepted)
	}
	stored, _ := os.ReadFile(fingerprintPath(repo, runID))
	for what, text := range map[string]string{
		"the refusal": r.Error(), "the refusal's JSON": string(doc), "the stored fingerprint": string(stored),
		"the accept's answer": strings.Join(accepted, " "), "the run's state": string(stateBytes(t, repo.Root(), runID)),
	} {
		if strings.Contains(text, token) || strings.Contains(text, pw) || strings.Contains(text, "x-access-token") {
			t.Fatalf("%s holds the key's secret:\n%s", what, text)
		}
	}
}

// TestATokenInAnyChangedNameIsSealedWhenNamed: a token in a key name the
// URL placeholder does not cover (a filter driver's name) is still sealed by
// the canonical redactor wherever the change is named: the refusal and the
// accept's answer.
func TestATokenInAnyChangedNameIsSealedWhenNamed(t *testing.T) {
	pat := "ghp_" + testsecret.Synthetic(63, 36)
	repo, runID := startedRun(t)
	repo.Git("config", "filter."+pat+".clean", "cat")

	_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, ".clean (added)") {
		t.Fatalf("the change is refused and named: %+v", r)
	}
	accepted, err := AcceptConfigChange(repo.Root(), runID, Options{})
	if err != nil || len(accepted) != 1 {
		t.Fatalf("the accept names the change: %q %v", accepted, err)
	}
	for what, text := range map[string]string{"the refusal": r.Error(), "the accept's answer": accepted[0],
		"the run's state": string(stateBytes(t, repo.Root(), runID))} {
		if strings.Contains(text, pat) {
			t.Fatalf("%s names the token:\n%s", what, text)
		}
	}
}

// TestALiveLaneReplacedByASymlinkIsAChange: the loop's lane steps run git at
// a live lane's worktree path whatever stands there now. A lane directory
// moved aside and replaced by a symlink to the moved copy, whose hooks were
// edited, stops the next step at the configuration stage, naming the lane's
// path; the state is as it was.
func TestALiveLaneReplacedByASymlinkIsAChange(t *testing.T) {
	repo, runID, lane := laneRun(t, func(repo *gittest.Repo) {
		repo.Write(".githooks/pre-commit", "#!/bin/sh\nexit 0\n")
		repo.Commit("hooks")
		repo.Git("config", "core.hooksPath", ".githooks")
	})
	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(lane, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moved, ".githooks", "pre-commit"), []byte("#!/bin/sh\ntouch hook-ran\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, lane); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), runID)

	_, err := advance(repo.Root(), runID, DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "the worktree "+fsutil.RedactHome(lane)+" (changed)") {
		t.Fatalf("the step is refused naming the lane's path: %+v", r)
	}
	if after := stateBytes(t, repo.Root(), runID); string(after) != string(before) {
		t.Fatal("the refused step took no stage")
	}
}

// TestARemovedLaneIsAChangeOnlyWhileItIsLive: a live lane's worktree that is
// no longer there is named gone, since the loop still runs git at its path; a
// lane that landed or was discarded runs nothing, so its worktree's removal
// is no change.
func TestARemovedLaneIsAChangeOnlyWhileItIsLive(t *testing.T) {
	repo, runID, lane := laneRun(t, nil)
	// An accept with nothing changed records the fingerprint as it stands,
	// the lane's worktree with it.
	if accepted, err := AcceptConfigChange(repo.Root(), runID, Options{}); err != nil || len(accepted) != 0 {
		t.Fatalf("nothing changed, nothing is accepted: %q %v", accepted, err)
	}
	root, err := os.OpenRoot(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	was, ok, err := readFingerprint(root, runID)
	if err != nil || !ok {
		t.Fatalf("the run recorded its fingerprint: %v", err)
	}
	if _, recorded := was.Worktrees[filepath.Clean(lane)]; !recorded {
		t.Fatalf("the fingerprint holds the lane: %+v", was.Worktrees)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(lane); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		stage Stage
		want  []string
	}{
		{st.Lanes[0].Stage, []string{"the worktree " + fsutil.RedactHome(filepath.Clean(lane)) + " (gone)"}},
		{StageDone, nil},
		{StageDiscarded, nil},
	} {
		s := st
		s.Lanes = slices.Clone(st.Lanes)
		s.Lanes[0].Stage = tc.stage
		cur, err := takeFingerprint(repo.Root(), laneWorktrees(repo.Root(), s), was.Salt, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if got := was.changes(repo.Root(), cur); !slices.Equal(got, tc.want) {
			t.Fatalf("a lane at %s removed: %q, want %q", tc.stage, got, tc.want)
		}
	}
}

// TestGitsErrorTextIsSealedInTheRefusal: git echoes a value it cannot parse
// in its error, and a value may carry a secret; an accept refused because the
// configuration cannot be read names git's error through the canonical
// redactor, so the token is in neither the refusal's text nor its JSON.
func TestGitsErrorTextIsSealedInTheRefusal(t *testing.T) {
	pat := "ghp_" + testsecret.Synthetic(64, 36)
	repo, runID := startedRun(t)
	repo.Git("config", "extensions.worktreeConfig", pat)

	_, err := AcceptConfigChange(repo.Root(), runID, Options{})
	r := mustRefusal(t, err)
	if r.Stage != StageConfig || !strings.Contains(r.Reason, "cannot be read") {
		t.Fatalf("the accept is refused, the configuration unreadable: %+v", r)
	}
	doc, _ := json.Marshal(r)
	for what, text := range map[string]string{"the refusal": r.Error(), "the refusal's JSON": string(doc)} {
		if strings.Contains(text, pat) {
			t.Fatalf("%s names the token:\n%s", what, text)
		}
	}
}
