package identity

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// isolateCommitter is isolate plus the committer's own environment overrides,
// cleared so the config-based cases stay hermetic whatever the test runner's
// shell exported.
func isolateCommitter(t *testing.T) {
	t.Helper()
	isolate(t)
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
}

// TestEffectiveCommitter_EnvFirst: git stamps the committer from
// GIT_COMMITTER_NAME/GIT_COMMITTER_EMAIL ahead of any config, exactly as it
// stamps the author from GIT_AUTHOR_*.
func TestEffectiveCommitter_EnvFirst(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	eff, err := effectiveCommitter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Name != "Test User" || eff.Email != "test@example.com" {
		t.Fatalf("committer did not follow GIT_COMMITTER_*: %+v", eff)
	}
}

// TestEffectiveCommitter_RoleConfigThenUser: committer.name/committer.email
// outrank user.name/user.email, each field on its own.
func TestEffectiveCommitter_RoleConfigThenUser(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	runGitT(t, dir, "config", "committer.email", "ci@example.com")
	eff, err := effectiveCommitter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Name != "Alex Reppel" || eff.Email != "ci@example.com" {
		t.Fatalf("committer resolution: got %+v, want Alex Reppel <ci@example.com>", eff)
	}
}

// TestCheck_CommitterEnvOverrideDiverges is the delta over the author-only
// check: the author matches the pin, so the author status stays OK exactly as
// before, and the committer divergence is reported beside it and blocks.
func TestCheck_CommitterEnvOverrideDiverges(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusOK {
		t.Fatalf("the author status must be unchanged by the committer: got %v — %s", res.Status, res.Reason)
	}
	if !res.CommitterDiverges {
		t.Fatalf("a GIT_COMMITTER_* override that differs from the pin was not detected: %+v", res)
	}
	if res.Committer.Name != "Test User" || !strings.Contains(res.CommitterReason, "Test User") {
		t.Fatalf("the committer divergence does not name the committer: %+v", res)
	}
	if !res.Blocks() {
		t.Fatal("a pinned repo whose committer differs from the pin must block")
	}
}

// TestCheck_CommitterConfigDiverges: a repo-local committer.name is the same
// divergence reached through config rather than the environment.
func TestCheck_CommitterConfigDiverges(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	runGitT(t, dir, "config", "committer.name", "Test User")
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusOK || !res.CommitterDiverges {
		t.Fatalf("want author OK and a committer divergence, got %v / %v — %s", res.Status, res.CommitterDiverges, res.CommitterReason)
	}
}

// TestCheck_AuthorNotCommitterUnpinned: with no pin the author status stays
// NoPin and nothing blocks (an un-pinned repo must never break commits), but an
// author that differs from the committer is still detected.
func TestCheck_AuthorNotCommitterUnpinned(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNoPin {
		t.Fatalf("want StatusNoPin, got %v", res.Status)
	}
	if !res.CommitterDiverges {
		t.Fatalf("author≠committer was not detected: %+v", res)
	}
	if res.Blocks() {
		t.Fatal("an un-pinned repo must never block")
	}
}

// TestCheck_AuthorMismatchIsNotAlsoACommitterDivergence: when the author and
// the committer are the same wrong identity, the author mismatch already says
// so and one repo-local fix mends both, so the committer is not reported twice.
func TestCheck_AuthorMismatchIsNotAlsoACommitterDivergence(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Test User", "test@example.com")
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusMismatch || res.CommitterDiverges {
		t.Fatalf("want a mismatch and no separate committer divergence, got %v / %v", res.Status, res.CommitterDiverges)
	}
}

// TestCheck_UnsetSurvivesTheCommitterPath is the regression guard the spec
// names: resolving the committer must not fabricate an identity (`git var`
// invents one from gecos and the hostname and exits 0), so a repo with no
// identity anywhere is still StatusUnset, with an empty effective committer.
func TestCheck_UnsetSurvivesTheCommitterPath(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "", "")
	writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusUnset {
		t.Fatalf("want StatusUnset, got %v — %s", res.Status, res.Reason)
	}
	if res.Committer != (Effective{}) {
		t.Fatalf("the committer was fabricated where git has none: %+v", res.Committer)
	}
	if !res.Blocks() {
		t.Fatal("an unset identity must still block")
	}
}

// TestCheck_FlagsAToolIdentity: the routine case — the harness's own default
// identity — is machine-visible in the result, in each role it occupies.
func TestCheck_FlagsAToolIdentity(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alex Reppel", "alex@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Claude")
	t.Setenv("GIT_COMMITTER_EMAIL", "noreply@anthropic.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuthorIsTool || !res.CommitterIsTool {
		t.Fatalf("want only the committer flagged as a tool identity, got author=%v committer=%v", res.AuthorIsTool, res.CommitterIsTool)
	}
}

// TestIsToolIdentity pins the predicate to the attribution gate's reading,
// including its role asymmetry: a `noreply@` mailbox is a machine as the
// AUTHOR, but the forge stamps `GitHub <noreply@github.com>` as the committer
// of every web-UI merge on a human's click, so it passes in that role alone.
func TestIsToolIdentity(t *testing.T) {
	cases := []struct {
		role        Role
		name, email string
		want        bool
	}{
		{RoleAuthor, "Claude", "noreply@anthropic.com", true},
		{RoleCommitter, "Claude", "noreply@anthropic.com", true},
		{RoleAuthor, "Alex Reppel", "bot@openai.com", true},
		{RoleAuthor, "  copilot ", "someone@example.com", true},
		{RoleAuthor, "Claudette", "claudette@example.com", false},
		{RoleAuthor, "dependabot[bot]", "49699333+dependabot[bot]@users.noreply.github.com", true},
		{RoleCommitter, "renovate[bot]", "renovate[bot]@users.noreply.github.com", true},
		{RoleAuthor, "Alex Reppel", "1234+alex@users.noreply.github.com", false},
		{RoleAuthor, "GitHub", "noreply@github.com", true},
		{RoleCommitter, "GitHub", "noreply@github.com", false},
		{RoleAuthor, "Some Tool", "do-not-reply@example.com", true},
		{RoleAuthor, "Alex Reppel", "alex@example.com", false},
		// A configured automation's name shape: the machine_name_word and
		// machine_local_word keys the attribution gate reads are refused here
		// too, since the list has two readers (iss-2609090951276167).
		{RoleAuthor, "semantic-release-bot", "12345+semantic-release-bot@users.noreply.github.com", true},
		{RoleCommitter, "ci_bot", "carol@example.com", true},
		{RoleAuthor, "Renovate Bot", "bot@renovateapp.com", true},
		{RoleAuthor, "release-automation", "carol@example.com", true},
		{RoleAuthor, "Jan Bot", "jan@example.com", false},
		{RoleAuthor, "Carol Talbot", "carol.talbot@example.com", false},
		{RoleAuthor, "Jean", "jean.bot@example.com", false},
	}
	for _, c := range cases {
		if got := IsToolIdentity(c.role, c.name, c.email); got != c.want {
			t.Errorf("IsToolIdentity(%s, %q, %q) = %v, want %v", c.role, c.name, c.email, got, c.want)
		}
	}
}

// TestStructuralSignalsLeaveTheNameShapeOut: the contributors page reads the
// structural signals alone, so the name-shape keys never reach it — refusing a
// configured name is the gates' job, not the published history's.
func TestStructuralSignalsLeaveTheNameShapeOut(t *testing.T) {
	if IsMachineName("semantic-release-bot") {
		t.Error("IsMachineName reads the name-shape word; it is the structural `[bot]` suffix alone")
	}
	if IsMachineAddress(RoleAuthor, "ci-bot@example.com") {
		t.Error("IsMachineAddress reads the local-part word; it is the structural bot mailbox alone")
	}
}

// TestToolIdentityListIsTheGatesOwn holds the one-list property: the CI
// attribution gate reads its identity patterns from the same file this package
// embeds, and defines none of its own. A literal pattern reappearing in the
// script is the second copy this test exists to refuse.
func TestToolIdentityListIsTheGatesOwn(t *testing.T) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Env = gittest.Env(t)
	top, err := cmd.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	script, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(top)), "scripts", "check-attribution.sh"))
	if err != nil {
		t.Skipf("gate script not found: %v", err)
	}
	if !strings.Contains(string(script), "internal/core/identity/"+toolIdentitiesFile) {
		t.Fatalf("scripts/check-attribution.sh does not read internal/core/identity/%s", toolIdentitiesFile)
	}
	for key, pat := range toolPatternSource {
		if strings.Contains(string(script), pat) {
			t.Errorf("scripts/check-attribution.sh carries its own copy of the %s pattern %q", key, pat)
		}
	}
}

// TestToolIdentityPatternsSpeakBothDialects: every pattern is read by POSIX ERE
// (grep -Ei, in the gate) and by RE2 (here), so each must parse in both.
func TestToolIdentityPatternsSpeakBothDialects(t *testing.T) {
	want := []string{"ai_name", "ai_mail", "machine_name", "machine_mail", "machine_name_word", "machine_local_word", "author_only_mail"}
	if len(toolPatternSource) != len(want) {
		t.Fatalf("the list carries %d keys, want exactly %v", len(toolPatternSource), want)
	}
	grep, grepErr := exec.LookPath("grep")
	for _, key := range want {
		pat, ok := toolPatternSource[key]
		if !ok || pat == "" {
			t.Fatalf("key %s missing or empty", key)
		}
		if _, err := regexp.Compile("(?i)" + pat); err != nil {
			t.Errorf("%s does not compile as RE2: %v", key, err)
		}
		if grepErr != nil {
			continue
		}
		// grep exits 1 on no match and 2 on a malformed pattern.
		cmd := exec.Command(grep, "-Eiq", pat)
		cmd.Stdin = strings.NewReader("")
		var ee *exec.ExitError
		if err := cmd.Run(); errors.As(err, &ee) && ee.ExitCode() == 2 {
			t.Errorf("%s is not a valid POSIX ERE: %q", key, pat)
		}
	}
}
