package identity

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

// toolIdentitiesFile is the one list of machine identities, embedded here and
// read at run time by scripts/check-attribution.sh, which names this file by its
// repository path. It is the list's only home: the CI gate's identity half and
// IsToolIdentity both read it, and neither carries a copy
// (TestToolIdentityListIsTheGatesOwn holds that).
const toolIdentitiesFile = "tool-identities.txt"

//go:embed tool-identities.txt
var toolIdentitiesRaw string

// toolPatternSource is the parsed list, pattern text keyed by its name.
var toolPatternSource = mustParseToolIdentities(toolIdentitiesRaw)

var (
	aiNameRe         = toolPattern("ai_name")
	aiMailRe         = toolPattern("ai_mail")
	machineNameRe    = toolPattern("machine_name")
	machineMailRe    = toolPattern("machine_mail")
	authorOnlyMailRe = toolPattern("author_only_mail")
)

// IsToolIdentity reports whether name <email> is a machine identity in the
// given role of a commit: an AI vendor's name or mail domain, the forge's own
// `[bot]` account, or a bot mailbox, in either role; and, in the AUTHOR role
// only, a mailbox named for not being read (`noreply@`). The asymmetry is the
// attribution gate's own — the forge stamps `GitHub <noreply@github.com>` as the
// committer of every web-UI merge made on a human's click, so it passes as a
// committer and nowhere else.
//
// It is how the routine case becomes machine-visible: an autonomous run that
// commits under the harness's default identity is flagged wherever the identity
// gate runs, before the attribution gate refuses the pull request in CI.
func IsToolIdentity(role Role, name, email string) bool {
	if aiNameRe.MatchString(name) || aiMailRe.MatchString(email) ||
		machineNameRe.MatchString(name) || machineMailRe.MatchString(email) {
		return true
	}
	return role == RoleAuthor && authorOnlyMailRe.MatchString(email)
}

// mustParseToolIdentities reads the embedded list's `key=pattern` lines. The
// list is compiled into the binary, so a malformed one is a build defect and
// panics at start-up rather than quietly matching nothing.
func mustParseToolIdentities(raw string) map[string]string {
	out := map[string]string{}
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, pat, ok := strings.Cut(line, "=")
		if !ok || key == "" || pat == "" {
			panic(fmt.Sprintf("identity: %s line %d is not key=pattern: %q", toolIdentitiesFile, i+1, line))
		}
		if _, dup := out[key]; dup {
			panic(fmt.Sprintf("identity: %s declares %s twice", toolIdentitiesFile, key))
		}
		out[key] = pat
	}
	return out
}

// toolPattern compiles one named pattern case-insensitively, as the gate's
// `grep -Ei` reads it.
func toolPattern(key string) *regexp.Regexp {
	pat, ok := toolPatternSource[key]
	if !ok {
		panic(fmt.Sprintf("identity: %s has no %s pattern", toolIdentitiesFile, key))
	}
	return regexp.MustCompile("(?i)" + pat)
}
