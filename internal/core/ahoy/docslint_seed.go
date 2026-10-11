package ahoy

import (
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The em-dash-in-list-item token is house style rather than a currency rule, so
// the product thinker ruled it OFFERED AT INSTALL (ruling G1, 2026-09-23): the
// adopter chooses whether it blocks or warns, and the chosen severity is written
// into the seeded docs-lint config, which is where the choice is recorded. An
// unattended (--yes) install is not asked and seeds it as a warning.
const (
	// emDashTokenID is the house-style token the adopter chooses a severity for.
	emDashTokenID = "punctuation/em-dash-in-list-item"
	// emDashPromptKey keys the install question, so a scripted answer stream and
	// the transcript name the question they answer.
	emDashPromptKey = "docs_lint.em_dash_in_list_item"
	// emDashDefaultSeverity is what an install seeds when the adopter did not
	// choose: the unattended default, and the answer a bare Enter or end of input
	// takes. The embedded seed carries it, so the seed is a loadable config as it
	// stands.
	emDashDefaultSeverity = "warn"
)

// emDashChoices are the two answers the ruling names, as the question offers
// them. The prompt shows them; emDashSeverityFor maps an answer onto the config's
// own severity words.
var emDashChoices = []string{"blocking", "warning"}

// emDashSeverityFor maps an answer to a config severity. It accepts the words the
// question offers and the words the config itself uses, case-insensitively, and
// reports ok=false for anything else.
func emDashSeverityFor(answer string) (severity string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "blocking", "blocker", "block":
		return "blocker", true
	case "warning", "warn":
		return "warn", true
	}
	return "", false
}

// emDashSeverity settles the severity the seed is written at, and the note that
// says so when abcd chose rather than the adopter.
//
// Declining to choose (a bare Enter, or end of input) takes the default the
// question displays, exactly as every other install prompt does, and the seeded
// config records it; that is an answer, so it draws no note. An answer naming
// neither choice is NOT refused the way a mistyped config value is: withholding
// the seed would withhold every other rule with it, the empty-seed defect this
// config exists to end. It is not guessed into a blocker either, because the "y"
// a `yes |` pipe sends to every question is exactly such an answer, and a gate
// the adopter did not choose would fail their build. It takes the warning, and
// the note says what was heard and where the severity lives.
func (a *applyCtx) emDashSeverity() (severity, note string) {
	where := banlist.PublicConfigRelPath
	if a.autoYes {
		return emDashDefaultSeverity, "the " + emDashTokenID + " house-style rule was not offered (an unattended --yes install): it is seeded as a warning; set its severity to \"blocker\" in " + where + " to make it block"
	}
	answer := a.prompter.Prompt(emDashPromptKey, emDashChoices, "warning")
	if sev, ok := emDashSeverityFor(answer); ok {
		return sev, ""
	}
	if strings.TrimSpace(answer) == "" {
		return emDashDefaultSeverity, ""
	}
	return emDashDefaultSeverity, "the answer " + termsafe.Sanitize(strconv.Quote(answer)) + " to " + emDashPromptKey +
		" names neither choice (blocking or warning), so the " + emDashTokenID + " house-style rule is seeded as a warning; set its severity to \"blocker\" in " + where + " to make it block"
}

// seedRootsLine is the embedded seed's roots field, exactly as the file spells
// it: the one span seedRoots rewrites.
const seedRootsLine = "\"roots\": [\n    \"docs\",\n    \"README.md\"\n  ],"

// seedRootChoices are the roots the seed may list, in the order it lists
// them, each with the test a path must pass to be one the documentation check
// runs over rather than refuses: docs as a folder (a file named docs is not
// markdown, and the check refuses it), README.md as anything that exists.
var seedRootChoices = []struct {
	rel string
	ok  func(fs.FileInfo) bool
}{
	{"docs", fs.FileInfo.IsDir},
	{"README.md", func(fs.FileInfo) bool { return true }},
}

// seedRoots is the roots the seed lists for the project held open at root:
// those of seedRootChoices present at install time (iss-2610040758095861).
// Seeding a root that is not there armed a check that refused to run from the
// moment it was written, `abcd lint docs` exiting 2 over a root setup named
// itself. With neither present the list is empty, and the check runs, reads no
// document and says loudly that nothing was checked, at exit 0. Each look stays
// inside the project, through the root held open.
func seedRoots(root *os.Root) []string {
	out := []string{}
	for _, c := range seedRootChoices {
		if st, err := root.Stat(c.rel); err == nil && c.ok(st) {
			out = append(out, c.rel)
		}
	}
	return out
}

// docsLintSeed renders the docs-lint seed with the em-dash token at severity
// and roots as its roots. The embedded seed carries the default severity and
// both roots; any other value rewrites that one token's severity line, or the
// roots field, and nothing else. The render is textual rather than a decode
// and re-encode because the seed is a file a person reads and edits, and a
// round trip through a map would reorder it.
func docsLintSeed(severity string, roots []string) []byte {
	return []byte(renderSeedRoots(renderSeedSeverity(severity), roots))
}

// renderSeedRoots rewrites the seed's roots field to list roots, one per line
// as the embedded file spells them, or [] when there are none.
func renderSeedRoots(seed string, roots []string) string {
	field := "\"roots\": [],"
	if len(roots) > 0 {
		quoted := make([]string, len(roots))
		for i, r := range roots {
			quoted[i] = "    " + strconv.Quote(r)
		}
		field = "\"roots\": [\n" + strings.Join(quoted, ",\n") + "\n  ],"
	}
	return strings.Replace(seed, seedRootsLine, field, 1)
}

// renderSeedSeverity is the seed with the em-dash token at severity.
func renderSeedSeverity(severity string) string {
	if severity == emDashDefaultSeverity {
		return publicFamilySeed
	}
	const sevLine = `"severity": "` + emDashDefaultSeverity + `"`
	anchor := strings.Index(publicFamilySeed, `"id": "`+emDashTokenID+`"`)
	if anchor < 0 {
		return publicFamilySeed
	}
	rest := publicFamilySeed[anchor:]
	end := strings.Index(rest, "}")
	at := strings.Index(rest, sevLine)
	if at < 0 || (end >= 0 && at > end) {
		return publicFamilySeed
	}
	at += anchor
	return publicFamilySeed[:at] + `"severity": "` + severity + `"` + publicFamilySeed[at+len(sevLine):]
}
