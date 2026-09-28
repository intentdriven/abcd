package gitutil

import (
	"strings"
	"testing"
)

// lastEnvValue returns the value git would resolve for key from an env slice:
// the LAST occurrence wins, matching how a child process reads its environment.
func lastEnvValue(env []string, key string) (string, bool) {
	val := ""
	found := false
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 && kv[:i] == key {
			val = kv[i+1:]
			found = true
		}
	}
	return val, found
}

// TestGitEnvPinsLocale asserts the isolated git environment pins LC_ALL=C and
// LANG=C so a translated git cannot alter the English chrome (e.g. the
// "N files changed" shortstat) the graveyard's rewrite signal parses — the
// cross-host determinism invariant. The pin must WIN over an ambient value.
func TestGitEnvPinsLocale(t *testing.T) {
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	t.Setenv("LANG", "fr_FR.UTF-8")

	env := gitEnv()

	if v, ok := lastEnvValue(env, "LC_ALL"); !ok || v != "C" {
		t.Errorf("resolved LC_ALL = %q (found=%v), want C", v, ok)
	}
	if v, ok := lastEnvValue(env, "LANG"); !ok || v != "C" {
		t.Errorf("resolved LANG = %q (found=%v), want C", v, ok)
	}
}

// TestScrubbedEnvStripsHijackKeepsGlobalConfig pins the ScrubbedEnv contract: the
// repo-selection and config-injection variables that could redirect the identity
// probe or forge a fake identity are removed, but the global/system config-file
// neutralisers IsolatedEnv appends are NOT — the identity probe must still read
// the caller's real ~/.gitconfig, or it would go blind and stop redacting.
func TestScrubbedEnvStripsHijackKeepsGlobalConfig(t *testing.T) {
	t.Setenv("GIT_DIR", "/somewhere/else/.git")
	t.Setenv("GIT_WORK_TREE", "/somewhere/else")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "user.email")
	t.Setenv("GIT_CONFIG_VALUE_0", "evil@example.com")
	t.Setenv("GIT_CONFIG_PARAMETERS", "'user.email'='evil@example.com'")

	env := ScrubbedEnv()

	for _, k := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_CONFIG_COUNT", "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_PARAMETERS"} {
		if v, ok := lastEnvValue(env, k); ok {
			t.Errorf("ScrubbedEnv leaked %s=%q; a hijack/injection var must be stripped", k, v)
		}
	}
	// Must NOT force the global-config neutralisers (that would blind the identity
	// probe). IsolatedEnv adds them; ScrubbedEnv must not.
	if v, ok := lastEnvValue(env, "GIT_CONFIG_GLOBAL"); ok {
		t.Errorf("ScrubbedEnv set GIT_CONFIG_GLOBAL=%q; it must keep global config in effect", v)
	}
}
