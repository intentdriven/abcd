package ahoy

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// ghFake installs a stand-in `gh` at the FRONT of PATH and returns the path of the
// log it appends one record to per invocation. There is no injected seam in the
// production API: the verb shells out to `gh`, so the test substitutes the binary
// rather than the call, and what is under test is the real argv and the real
// ordering rather than a mock's idea of them.
//
// getBody is what the GET returns; every PATCH exits 0 unless GH_FAKE_PATCH_FAIL
// names a substring of its stdin.
func ghFake(t *testing.T, getBody string) (logPath string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "gh.log")
	body := filepath.Join(dir, "get.json")
	if err := os.WriteFile(body, []byte(getBody), 0o644); err != nil {
		t.Fatal(err)
	}
	script := `#!/usr/bin/env bash
set -euo pipefail
stdin=$(cat)
{
  printf 'ARGV %s\n' "$*"
  printf 'STDIN %s\n' "$stdin"
} >>"` + logPath + `"
for arg in "$@"; do
  if [ "$arg" = "PATCH" ]; then
    if [ -n "${GH_FAKE_PATCH_FAIL:-}" ] && [ "${stdin#*$GH_FAKE_PATCH_FAIL}" != "$stdin" ]; then
      echo "gh: HTTP 403" >&2
      exit 1
    fi
    exit 0
  fi
done
if [ -n "${GH_FAKE_GET_FAIL:-}" ]; then
  echo "gh: HTTP 404" >&2
  exit 1
fi
cat "` + body + `"
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// ghCalls returns one string per fake-gh invocation, "ARGV … | STDIN …".
func ghCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var calls []string
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for i := 0; i+1 < len(lines); i += 2 {
		calls = append(calls, lines[i]+" | "+lines[i+1])
	}
	return calls
}

const (
	mergeHygiene = `,"delete_branch_on_merge":true,"allow_merge_commit":true,` +
		`"allow_squash_merge":true,"allow_rebase_merge":false`
	bothDisabled = `{"security_and_analysis":{"secret_scanning":{"status":"disabled"},` +
		`"secret_scanning_push_protection":{"status":"disabled"}}` + mergeHygiene + `}`
	bothEnabled = `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},` +
		`"secret_scanning_push_protection":{"status":"enabled"}}` + mergeHygiene + `}`
)

// managedRepoWithOrigin stands up an abcd-managed repo whose origin points at
// GitHub — the only shape the verb will act on.
func managedRepoWithOrigin(t *testing.T, origin string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	env := gittest.Env(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("config", "user.name", "Alice Example")
	git("config", "user.email", "alice@example.com")
	if origin != "" {
		git("remote", "add", "origin", origin)
	}
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// mirror reads the committed desired-state mirror, or "" when it is absent.
func mirror(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(RepoSettingsMirrorRelPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(data)
}

// TestRemoteApplyRequiresConfirmationNotJustInvocation pins invariant 10: a remote
// write happens only through a dedicated verb the user invokes AND confirms. The
// default prompter DECLINES, so a non-interactive caller that never answers changes
// nothing — the safe direction for the only abcd operation that reaches outside the
// machine.
func TestRemoteApplyRequiresConfirmationNotJustInvocation(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")

	res, err := RemoteApply(repo, RefusingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "aborted" {
		t.Errorf("status = %q, want aborted", res.Status)
	}
	for _, c := range ghCalls(t, logPath) {
		if strings.Contains(c, "PATCH") {
			t.Errorf("an unconfirmed apply wrote to the remote:\n%s", c)
		}
	}
	if m := mirror(t, repo); m != "" {
		t.Errorf("an unconfirmed apply mirrored a state it never applied:\n%s", m)
	}
	if len(res.Notes) == 0 {
		t.Error("the decline is silent")
	}
	// A nil prompter is the same refusal: a caller that forgot the seam must not
	// get an unconfirmed write by omission.
	nilRes, err := RemoteApply(repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if nilRes.Status != "aborted" {
		t.Errorf("a nil prompter produced status %q, want aborted", nilRes.Status)
	}
}

// TestRemoteApplyDoesNotAskWhenNothingWouldChange: a confirmation over a write
// that is not going to happen trains the answerer to say yes without reading.
func TestRemoteApplyDoesNotAskWhenNothingWouldChange(t *testing.T) {
	setupHermetic(t)
	ghFake(t, bothEnabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")

	res, err := RemoteApply(repo, RefusingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "aborted" {
		t.Errorf("a repo already in the desired state was aborted over a confirmation it never needed: %v", res.Notes)
	}
}

// TestRemoteApplyEnablesSecretScanningBeforePushProtection is itd-153's first
// acceptance criterion, and the ordering half of it is load-bearing rather than
// stylistic: GitHub refuses push protection on a repo whose secret scanning is off,
// so the reverse order enables nothing and reports a failure the operator cannot
// act on.
func TestRemoteApplyEnablesSecretScanningBeforePushProtection(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo.git")

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("status = %q (notes=%v), want clean", res.Status, res.Notes)
	}
	calls := ghCalls(t, logPath)
	if len(calls) != 3 {
		t.Fatalf("want one GET and two PATCHes, got %d calls:\n%s", len(calls), strings.Join(calls, "\n"))
	}
	if strings.Contains(calls[0], "PATCH") {
		t.Errorf("the first call writes; the state must be READ before anything is changed:\n%s", calls[0])
	}
	if !strings.Contains(calls[1], `"secret_scanning"`) {
		t.Errorf("the first write is not secret scanning:\n%s", calls[1])
	}
	if !strings.Contains(calls[2], `"secret_scanning_push_protection"`) {
		t.Errorf("the second write is not push protection:\n%s", calls[2])
	}
	// Every call names the repo the verb resolved, and no other.
	for _, c := range calls {
		if !strings.Contains(c, "repos/example-org/example-repo") {
			t.Errorf("a call does not name the resolved repo:\n%s", c)
		}
	}
}

// TestRemoteApplyMirrorsTheDesiredState is AC4: a later verify reads the same
// intended state this apply wrote, from the tree rather than from a web console.
func TestRemoteApplyMirrorsTheDesiredState(t *testing.T) {
	setupHermetic(t)
	ghFake(t, bothDisabled)
	repo := managedRepoWithOrigin(t, "git@github.com:example-org/example-repo.git")

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Writes) == 0 {
		t.Error("the apply reports no write, so nothing says where the desired state landed")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(mirror(t, repo)), &got); err != nil {
		t.Fatalf("the mirror is not readable JSON: %v", err)
	}
	managed, _ := got["managed"].(map[string]any)
	sec, _ := managed["security_and_analysis"].(map[string]any)
	for _, key := range []string{"secret_scanning", "secret_scanning_push_protection"} {
		entry, _ := sec[key].(map[string]any)
		if entry["status"] != "enabled" {
			t.Errorf("the mirror does not record %s enabled: %v", key, sec)
		}
	}
	if got["repo"] != "example-org/example-repo" {
		t.Errorf("the mirror does not name the repo it describes: %v", got["repo"])
	}
	// The merge-hygiene settings had no source of truth in the tree at all
	// (iss-2608270512210664): the ruleset mirror beside this one covers rulesets,
	// and these live behind the repository-object API. They are RECORDED, in a
	// section separate from what abcd drives — including the false, so a reader can
	// tell a setting that is off from one the API never reported.
	observed, ok := got["observed"].(map[string]any)
	if !ok {
		t.Fatalf("the mirror records no merge-hygiene snapshot: %v", got)
	}
	for key, want := range map[string]bool{
		"delete_branch_on_merge": true,
		"allow_merge_commit":     true,
		"allow_squash_merge":     true,
		"allow_rebase_merge":     false,
	} {
		if observed[key] != want {
			t.Errorf("mirror observed[%s] = %v, want %v", key, observed[key], want)
		}
	}
	if _, ok := got["observed"].(map[string]any)["secret_scanning"]; ok {
		t.Error("the observed section carries a setting abcd drives; the two must stay legibly apart")
	}
}

// TestRemoteApplyIsIdempotent is AC3. A repo already in the desired state must
// take no write that alters it — not on the remote, and not in the tree: a verb
// that rewrote its own mirror on every run would make "nothing changed" and "the
// change was reapplied" produce the same diff.
func TestRemoteApplyIsIdempotent(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothEnabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")

	// The FIRST run still writes the mirror — the tree does not yet record the
	// intent — but it changes no toggle, and it must take no remote write at all.
	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Errorf("first status = %q, want clean (the mirror landed)", res.Status)
	}
	if len(res.Changes) != 0 {
		t.Errorf("a repo already in the desired state reported changes: %v", res.Changes)
	}
	for _, c := range ghCalls(t, logPath) {
		if strings.Contains(c, "PATCH") {
			t.Errorf("a repo already in the desired state took a write:\n%s", c)
		}
	}
	before := treeHash(t, repo)
	res2, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != "already_up_to_date" {
		t.Errorf("second status = %q, want already_up_to_date", res2.Status)
	}
	if len(res2.Writes) != 0 {
		t.Errorf("a re-run reported writes: %v", res2.Writes)
	}
	if msg, ok := sameTree(before, treeHash(t, repo)); !ok {
		t.Errorf("a re-run changed the tree: %s", msg)
	}
}

// TestRemoteApplyHonoursTheOptOut is AC2, and it is stricter than the criterion:
// an opted-out repo takes no remote call AT ALL, not merely no write. A verb that
// still asked GitHub about a repo the maintainer told it to leave alone would be
// reaching across the boundary the opt-out draws.
func TestRemoteApplyHonoursTheOptOut(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")
	writeNativeScanningOptOut(t, repo)

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "opted_out" {
		t.Errorf("status = %q, want opted_out", res.Status)
	}
	if calls := ghCalls(t, logPath); len(calls) != 0 {
		t.Errorf("an opted-out repo was still contacted:\n%s", strings.Join(calls, "\n"))
	}
	if m := mirror(t, repo); m != "" {
		t.Errorf("a committed mirror was written for a repo that opted out:\n%s", m)
	}
	if len(res.Notes) == 0 {
		t.Error("the opt-out is silent; a thing abcd deliberately did not do must say so")
	}
}

// writeNativeScanningOptOut records the explicit opt-out in the repo's config.
func writeNativeScanningOptOut(t *testing.T, repo string) {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	setSub(cfg, "scan", "native_secret_scanning", false)
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
}

// TestRemoteApplyRefusesRatherThanGuess pins the fail-closed direction on every
// input the verb cannot trust. A remote write is the one place where "carry on and
// hope" mutates something outside this machine.
func TestRemoteApplyRefusesRatherThanGuess(t *testing.T) {
	t.Run("no origin remote", func(t *testing.T) {
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		repo := managedRepoWithOrigin(t, "")
		res, err := RemoteApply(repo, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "refused" {
			t.Errorf("status = %q, want refused", res.Status)
		}
		if calls := ghCalls(t, logPath); len(calls) != 0 {
			t.Errorf("a repo with no resolvable identity was still contacted:\n%s", strings.Join(calls, "\n"))
		}
	})

	t.Run("a remote that is not GitHub", func(t *testing.T) {
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		repo := managedRepoWithOrigin(t, "https://gitlab.example.com/example-org/example-repo.git")
		res, err := RemoteApply(repo, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "refused" {
			t.Errorf("status = %q, want refused", res.Status)
		}
		if calls := ghCalls(t, logPath); len(calls) != 0 {
			t.Errorf("a non-GitHub remote was sent to the GitHub API:\n%s", strings.Join(calls, "\n"))
		}
	})

	t.Run("an origin that would redirect the API path", func(t *testing.T) {
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		// A path segment that traverses would turn `repos/OWNER/REPO` into some other
		// endpoint entirely — on a verb whose whole job is a privileged write.
		repo := managedRepoWithOrigin(t, "https://github.com/../../user/repos.git")
		res, err := RemoteApply(repo, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "refused" {
			t.Errorf("status = %q, want refused", res.Status)
		}
		if calls := ghCalls(t, logPath); len(calls) != 0 {
			t.Errorf("a traversing owner/repo reached the API:\n%s", strings.Join(calls, "\n"))
		}
	})

	t.Run("run from a subdirectory, the opt-out still binds", func(t *testing.T) {
		// The repository identity is resolved by git's UPWARD search, so a run from a
		// subdirectory still names the enclosing repo — while a consent check read from
		// the cwd would find no config there and read ABSENCE AS CONSENT, sending two
		// PATCHes to a repository whose maintainer recorded the opt-out. The verb
		// anchors every question at the working-tree root, so all of them name the same
		// repository.
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")
		writeNativeScanningOptOut(t, repo)
		sub := filepath.Join(repo, "internal", "deep")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := RemoteApply(sub, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "opted_out" {
			t.Errorf("status from a subdirectory = %q, want opted_out", res.Status)
		}
		if calls := ghCalls(t, logPath); len(calls) != 0 {
			t.Errorf("a run from a subdirectory contacted a repo that opted out:\n%s", strings.Join(calls, "\n"))
		}
		if _, err := os.Stat(filepath.Join(sub, ".abcd")); err == nil {
			t.Error("a stray .abcd tree was created in the subdirectory")
		}
	})

	t.Run("run from a subdirectory, the mirror lands at the repo root", func(t *testing.T) {
		setupHermetic(t)
		ghFake(t, bothDisabled)
		repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")
		sub := filepath.Join(repo, "internal", "deep")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := RemoteApply(sub, confirmingPrompter{}, nil); err != nil {
			t.Fatal(err)
		}
		if mirror(t, repo) == "" {
			t.Error("the mirror did not land at the repo root, so the committed record was not refreshed")
		}
		if _, err := os.Stat(filepath.Join(sub, ".abcd")); err == nil {
			t.Error("the mirror was written into the subdirectory, outside the committed tier")
		}
	})

	t.Run("a repo abcd does not manage", func(t *testing.T) {
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		repo := t.TempDir()
		res, err := RemoteApply(repo, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "refused" {
			t.Errorf("status = %q, want refused", res.Status)
		}
		if calls := ghCalls(t, logPath); len(calls) != 0 {
			t.Errorf("a folder abcd does not manage was acted on:\n%s", strings.Join(calls, "\n"))
		}
	})

	t.Run("the read fails", func(t *testing.T) {
		setupHermetic(t)
		logPath := ghFake(t, bothDisabled)
		t.Setenv("GH_FAKE_GET_FAIL", "1")
		repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")
		res, err := RemoteApply(repo, confirmingPrompter{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "refused" {
			t.Errorf("status = %q, want refused", res.Status)
		}
		for _, c := range ghCalls(t, logPath) {
			if strings.Contains(c, "PATCH") {
				t.Errorf("a write followed a read that failed; the current state is unknown:\n%s", c)
			}
		}
		if m := mirror(t, repo); m != "" {
			t.Errorf("a mirror was written for a state nobody could read:\n%s", m)
		}
	})
}

// TestRemoteApplyStopsAtTheFirstFailedWrite: push protection requires secret
// scanning, so a failed first step must not be followed by a second that cannot
// succeed — and a partial apply must never be mirrored as if it were complete.
func TestRemoteApplyStopsAtTheFirstFailedWrite(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	t.Setenv("GH_FAKE_PATCH_FAIL", `"secret_scanning":`)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")

	res, err := RemoteApply(repo, confirmingPrompter{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" {
		t.Errorf("status = %q, want refused", res.Status)
	}
	for _, c := range ghCalls(t, logPath) {
		if strings.Contains(c, "push_protection") {
			t.Errorf("push protection was attempted after secret scanning failed; GitHub refuses it:\n%s", c)
		}
	}
	if m := mirror(t, repo); m != "" {
		t.Errorf("a partial apply was mirrored as the desired state:\n%s", m)
	}
	if len(res.Notes) == 0 {
		t.Error("the refusal carries no reason")
	}
}

// TestRemoteReadIsReadOnly pins the bare-invocation convention on a verb that can
// write: the read reports what is there and what would change, and touches nothing.
func TestRemoteReadIsReadOnly(t *testing.T) {
	setupHermetic(t)
	logPath := ghFake(t, bothDisabled)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")

	before := treeHash(t, repo)
	res, err := RemoteRead(repo)
	if err != nil {
		t.Fatal(err)
	}
	if res.Observed.SecretScanning != "disabled" {
		t.Errorf("observed secret scanning = %q, want disabled", res.Observed.SecretScanning)
	}
	if len(res.Changes) != 2 {
		t.Errorf("the read does not name both changes an apply would make: %v", res.Changes)
	}
	for _, c := range ghCalls(t, logPath) {
		if strings.Contains(c, "PATCH") {
			t.Errorf("the read wrote to the remote:\n%s", c)
		}
	}
	if msg, ok := sameTree(before, treeHash(t, repo)); !ok {
		t.Errorf("the read changed the tree: %s", msg)
	}
}
