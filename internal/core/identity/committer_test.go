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
	dir := gitRepo(t, "Alice", "alice@example.com")
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
	dir := gitRepo(t, "Alice", "alice@example.com")
	runGitT(t, dir, "config", "committer.email", "ci@example.com")
	eff, err := effectiveCommitter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Name != "Alice" || eff.Email != "ci@example.com" {
		t.Fatalf("committer resolution: got %+v, want Alice <ci@example.com>", eff)
	}
}

// TestCheck_CommitterEnvOverrideDiverges is the delta over the author-only
// check: the author matches the pin, so the author status stays OK exactly as
// before, and the committer divergence is reported beside it and blocks.
func TestCheck_CommitterEnvOverrideDiverges(t *testing.T) {
	isolateCommitter(t)
	dir := gitRepo(t, "Alice", "alice@example.com")
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
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
	dir := gitRepo(t, "Alice", "alice@example.com")
	runGitT(t, dir, "config", "committer.name", "Test User")
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
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
	dir := gitRepo(t, "Alice", "alice@example.com")
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusNoRecord {
		t.Fatalf("want StatusNoRecord, got %v", res.Status)
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
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
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
	writePin(t, dir, `{"name":"Alice","email":"alice@example.com"}`)
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
	dir := gitRepo(t, "Alice", "alice@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Claude")
	t.Setenv("GIT_COMMITTER_EMAIL", "noreply@anthropic.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuthorIsTool || !res.CommitterIsTool {
		t.Fatalf("want only the committer flagged as a tool identity, got author=%v committer=%v", res.AuthorIsTool, res.CommitterIsTool)
	}
	// Criterion 5: with no record on a fresh machine, a commit is never made
	// as the tool.
	if !res.Blocks() {
		t.Fatal("a machine identity in the committer role must block, record or not")
	}
}

// noIdentityEnv is isolateCommitter with the identity variables unset rather than
// blank: git refuses a blank GIT_AUTHOR_NAME where the resolver reads it as unset,
// so a blank one would stop `git var` before it stamped anything. EMAIL is pinned
// because git falls back to it for an unset email.
func noIdentityEnv(t *testing.T) {
	t.Helper()
	isolateCommitter(t)
	t.Setenv("EMAIL", "")
	for _, key := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCheck_JudgesTheIdentityGitWouldStamp: git removes some bytes when it stamps
// an identity, and the check judges the stamped form, as the attribution gate
// reads the commit.
func TestCheck_JudgesTheIdentityGitWouldStamp(t *testing.T) {
	for _, tc := range []struct {
		name          string
		user, mail    string
		env           map[string]string
		wantAuthor    bool
		wantCommitter bool
	}{
		{name: "an angle bracket inside the name and the email", user: "Cl<aude", mail: "no<reply@anthro>pic.com", wantAuthor: true, wantCommitter: true},
		{name: "an angle bracket inside the name alone", user: "Cl>aude", mail: "alice@example.com", wantAuthor: true, wantCommitter: true},
		{name: "an angle bracket inside the email alone", user: "Alice", mail: "alice@anthro<pic.com", wantAuthor: true, wantCommitter: true},
		{name: "a trailing comma after a name shape", user: "ci-bot,", mail: "alice@example.com", wantAuthor: true, wantCommitter: true},
		{name: "a mailbox noreply only once stamped, in the author role", user: "Alice", mail: "<noreply@example.com", wantAuthor: true},
		{name: "the author's environment", user: "Alice", mail: "alice@example.com",
			env: map[string]string{"GIT_AUTHOR_NAME": "Cl>aude"}, wantAuthor: true},
		{name: "the committer's environment", user: "Alice", mail: "alice@example.com",
			env: map[string]string{"GIT_COMMITTER_EMAIL": "ci@anthro>pic.com"}, wantCommitter: true},
		{name: "a command-line entry, as a hook inherits it", user: "Alice", mail: "alice@example.com",
			env: map[string]string{"GIT_CONFIG_COUNT": "1", "GIT_CONFIG_KEY_0": "committer.name", "GIT_CONFIG_VALUE_0": "ci-bot,"}, wantCommitter: true},
		{name: "a person whose name carries the same bytes", user: "Carol Talbot,", mail: "carol.talbot@example.com"},
		{name: "a person's name holding a machine's only in part", user: "Claudette<", mail: "claudette@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			noIdentityEnv(t)
			dir := gitRepo(t, tc.user, tc.mail)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			res, err := Check(dir)
			if err != nil {
				t.Fatal(err)
			}
			if res.AuthorIsTool != tc.wantAuthor || res.CommitterIsTool != tc.wantCommitter {
				t.Fatalf("author=%v committer=%v, want author=%v committer=%v", res.AuthorIsTool, res.CommitterIsTool, tc.wantAuthor, tc.wantCommitter)
			}
			if tc.wantAuthor || tc.wantCommitter {
				if !res.Blocks() {
					t.Fatal("a machine identity as stamped must block")
				}
			}
		})
	}
}

// TestCheck_StampedIdentityUnderUseConfigOnly: user.useConfigOnly stops git
// guessing a field, which fails `git var` only when a field is unset. A complete
// identity is still read as stamped, and an incomplete one stays what it is, an
// unset identity, with no error from the failed read.
func TestCheck_StampedIdentityUnderUseConfigOnly(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		noIdentityEnv(t)
		dir := gitRepo(t, "ci-bot,", "alice@example.com")
		runGitT(t, dir, "config", "user.useConfigOnly", "true")
		res, err := Check(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !res.AuthorIsTool || !res.CommitterIsTool {
			t.Fatalf("author=%v committer=%v, want both flagged", res.AuthorIsTool, res.CommitterIsTool)
		}
	})
	t.Run("no email", func(t *testing.T) {
		noIdentityEnv(t)
		dir := gitRepo(t, "ci-bot,", "")
		runGitT(t, dir, "config", "user.useConfigOnly", "true")
		res, err := Check(dir)
		if err != nil {
			t.Fatalf("a failed read of the stamped form is not a failed check: %v", err)
		}
		if res.Status != StatusUnset || !res.Blocks() {
			t.Fatalf("status %v blocks %v, want an unset identity that blocks", res.Status, res.Blocks())
		}
		if res.AuthorIsTool || res.CommitterIsTool {
			t.Fatalf("author=%v committer=%v, want neither: the set value is not a machine's and git stamps nothing", res.AuthorIsTool, res.CommitterIsTool)
		}
	})
}

// TestCheck_JudgesNoFieldGitWouldMakeUp: where a field is unset, `git var` fills
// it from the environment or the account and exits 0. That field is left out of
// the judgment, so an unset identity is reported as unset and never as a machine's.
func TestCheck_JudgesNoFieldGitWouldMakeUp(t *testing.T) {
	noIdentityEnv(t)
	dir := gitRepo(t, "Alice", "")
	t.Setenv("EMAIL", "noreply@anthropic.com")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuthorIsTool || res.CommitterIsTool {
		t.Fatalf("author=%v committer=%v, want neither: the email is not configured", res.AuthorIsTool, res.CommitterIsTool)
	}
	if res.Status != StatusUnset || !res.Blocks() {
		t.Fatalf("status %v blocks %v, want an unset identity that blocks", res.Status, res.Blocks())
	}
}

// TestStampedIdentityReadsTheRepositoryAtRoot: the read runs under
// commitConfigEnv, so an inherited GIT_DIR does not point it at another repository.
func TestStampedIdentityReadsTheRepositoryAtRoot(t *testing.T) {
	noIdentityEnv(t)
	dir := gitRepo(t, "Alice", "alice@example.com")
	other := gitRepo(t, "Bob", "bob@example.com")
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	for _, role := range []Role{RoleAuthor, RoleCommitter} {
		got, ok := stampedIdentity(dir, role)
		if !ok || got != (Effective{Name: "Alice", Email: "alice@example.com"}) {
			t.Errorf("%s: read %+v, %v, want the identity of the repository at root", role, got, ok)
		}
	}
}

// TestParseIdent reads the line `git var` prints.
func TestParseIdent(t *testing.T) {
	for line, want := range map[string]Effective{
		"Alice Example <alice@example.com> 1793945264 +0000\n": {Name: "Alice Example", Email: "alice@example.com"},
		"Alice <> 1793945264 +0000":                            {Name: "Alice"},
	} {
		got, ok := parseIdent(line)
		if !ok || got != want {
			t.Errorf("parseIdent(%q) = %+v, %v, want %+v", line, got, ok, want)
		}
	}
	for _, line := range []string{"", "no brackets 1793945264 +0000", "Alice >a@example.com< 1 +0000"} {
		if got, ok := parseIdent(line); ok {
			t.Errorf("parseIdent(%q) = %+v, want a refusal", line, got)
		}
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
		{RoleAuthor, "Alice", "bot@openai.com", true},
		{RoleAuthor, "  copilot ", "someone@example.com", true},
		{RoleAuthor, "Claudette", "claudette@example.com", false},
		{RoleAuthor, "dependabot[bot]", "49699333+dependabot[bot]@users.noreply.github.com", true},
		{RoleCommitter, "renovate[bot]", "renovate[bot]@users.noreply.github.com", true},
		{RoleAuthor, "Alice", "1234+alice@users.noreply.github.com", false},
		{RoleAuthor, "GitHub", "noreply@github.com", true},
		{RoleCommitter, "GitHub", "noreply@github.com", false},
		{RoleAuthor, "Some Tool", "do-not-reply@example.com", true},
		{RoleAuthor, "Alice", "alice@example.com", false},
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
