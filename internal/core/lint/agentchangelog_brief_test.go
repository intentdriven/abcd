package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// silentAgentChangelogEntries returns, as "<agent> <version>", every entry of
// the prompt-version log whose own text and whose dated section's preamble both
// leave out the word "unmeasured": the entries that never record the
// calibration delta as unmeasured. A "## " line opens a dated section and an
// entry heading ("### <agent> <semver>") opens an entry; the preamble is what
// sits between the section heading and its first entry.
func silentAgentChangelogEntries(text string) []string {
	var silent []string
	var preamble, body strings.Builder
	entry, inEntry := "", false
	flush := func() {
		if inEntry && !strings.Contains(strings.ToLower(preamble.String()+body.String()), "unmeasured") {
			silent = append(silent, entry)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			flush()
			inEntry = false
			preamble.Reset()
		case strings.HasPrefix(line, "### "):
			flush()
			m := agentChangelogHeadingRe.FindStringSubmatch(line)
			if m == nil {
				inEntry = false
				continue
			}
			entry, inEntry = m[1]+" "+m[2], true
			body.Reset()
		case inEntry:
			body.WriteString(line + "\n")
		default:
			preamble.WriteString(line + "\n")
		}
	}
	flush()
	return silent
}

// The prompt-quality chapter's calibration-band bullet lists the changelog
// entries that record the delta as unmeasured nowhere. It once named the five
// 2026-08-19 entries and "one cold-reading bump" while the log held three such
// cold-reading entries and two renames besides (iss-2611040555021637), so the
// bullet is held to the log: it names every silent entry by its agent and its
// version, and its count is theirs.
func TestBriefCalibrationBandNamesEveryEntryThatStatesNoDelta(t *testing.T) {
	const briefRel = ".abcd/development/brief/05-internals/05-prompt-quality.md"
	root := filepath.Join("..", "..", "..")
	brief, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(briefRel)))
	if err != nil {
		t.Fatalf("read %s: %v", briefRel, err)
	}
	log, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(defaultAgentChangelog)))
	if err != nil {
		t.Fatalf("read %s: %v", defaultAgentChangelog, err)
	}

	const opener = "- **The `0.x` calibration band (ships).**"
	text := string(brief)
	start := strings.Index(text, opener)
	if start < 0 {
		t.Fatalf("%s has no bullet opening %q", briefRel, opener)
	}
	bullet := text[start:]
	if end := strings.Index(bullet[len(opener):], "\n- "); end >= 0 {
		bullet = bullet[:len(opener)+end]
	}
	flat := strings.Join(strings.Fields(bullet), " ")

	silent := silentAgentChangelogEntries(string(log))
	if len(silent) == 0 {
		t.Fatalf("%s holds no entry that leaves the delta unstated; the bullet's list should go", defaultAgentChangelog)
	}
	for _, e := range silent {
		agent, version, _ := strings.Cut(e, " ")
		if !strings.Contains(flat, "`"+agent+"`") || !strings.Contains(flat, "`"+version+"`") {
			t.Errorf("%s states no delta, but the calibration-band bullet of %s does not name `%s` and `%s`", e, briefRel, agent, version)
		}
	}

	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
		"eleven", "twelve", "thirteen", "fourteen", "fifteen"}
	m := regexp.MustCompile(`\b(\w+) (?:entries )?say it nowhere`).FindStringSubmatch(flat)
	if m == nil {
		t.Fatalf("the calibration-band bullet of %s no longer says how many entries say it nowhere:\n%s", briefRel, flat)
	}
	if len(silent) >= len(words) || !strings.EqualFold(m[1], words[len(silent)]) {
		t.Errorf("the calibration-band bullet says %q entries say it nowhere; %s holds %d: %v", m[1], defaultAgentChangelog, len(silent), silent)
	}
}
