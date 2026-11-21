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
	machineNameWord  = toolPattern("machine_name_word")
	machineLocalWord = toolPattern("machine_local_word")
	authorOnlyMailRe = toolPattern("author_only_mail")
)

// IsToolIdentity reports whether name <email> is a machine identity in the
// given role of a commit: an AI vendor's name or mail domain, the forge's own
// `[bot]` account, a bot mailbox, or a configured automation's name shape (a
// trailing `bot`, `robot` or `automation` word ending the name or the mailbox's
// local part), in either role; and, in the AUTHOR role only, a mailbox named for
// not being read (`noreply@`). The asymmetry is the
// attribution gate's own — the forge stamps `GitHub <noreply@github.com>` as the
// committer of every web-UI merge made on a human's click, so it passes as a
// committer and nowhere else.
//
// It is how the routine case becomes machine-visible: an autonomous run that
// commits under the harness's default identity is flagged wherever the identity
// gate runs, before the attribution gate refuses the pull request in CI.
//
// It judges the values it is given. Check gives it the identity as git would stamp
// it too (isToolAsStamped), since a value as set can differ from its stamp.
func IsToolIdentity(role Role, name, email string) bool {
	return aiNameRe.MatchString(name) || aiMailRe.MatchString(email) ||
		IsMachineName(name) || IsMachineAddress(role, email) ||
		machineNameWord.MatchString(name) || machineLocalWord.MatchString(email)
}

// AINameSpans is where s holds a name on the list the attribution gate refuses
// as an author (`claude`, `codex` and the rest), case-insensitively and
// wherever it stands, inside a longer word too (`xclaude`, `Codex2`): each
// span's start and end, leftmost first, none overlapping. A reader that must
// withhold such a name from text, such as the model matrix keeping a check
// copy from signing its findings, reads the list through it, so the list
// keeps its one home.
func AINameSpans(s string) [][]int { return aiNameInRe.FindAllStringIndex(s, -1) }

// aiNameInRe is the ai_name pattern without the anchors that make it match a
// whole name: its list of names alone, matched anywhere.
var aiNameInRe = unanchored("ai_name")

// unanchored is key's pattern with its whole-name anchors cut off. A pattern
// not written between them is a build defect, as a malformed list is.
func unanchored(key string) *regexp.Regexp {
	const open, closing = "^[[:space:]]*", "[[:space:]]*$"
	pat := toolPatternSource[key]
	core, ok := strings.CutPrefix(pat, open)
	core, ok2 := strings.CutSuffix(core, closing)
	if !ok || !ok2 || core == "" {
		panic(fmt.Sprintf("identity: %s's %s pattern is not one list between %q and %q", toolIdentitiesFile, key, open, closing))
	}
	return regexp.MustCompile("(?i)" + core)
}

// IsMachineName reports the STRUCTURAL name signal alone: the `[bot]` suffix the
// forge stamps on an app's account name. It names no vendor, so a reader that
// must not decide on a vendor token (the contributors page) can use it.
func IsMachineName(name string) bool { return machineNameRe.MatchString(name) }

// IsMachineAddress reports the STRUCTURAL address signals alone, in role: a bot
// mailbox in either role, and a mailbox named for not being read as the author.
// The mailbox is the discriminator, never the host: a person's forge privacy
// address (`1234+name@users.noreply.github.com`) is not a machine's.
func IsMachineAddress(role Role, email string) bool {
	return machineMailRe.MatchString(email) ||
		(role == RoleAuthor && authorOnlyMailRe.MatchString(email))
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
