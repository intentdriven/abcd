package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// hooks_selfprovision_count_test.go — the brief's self-provisioning paragraphs
// are held to the SET the shipped manifest wires, not to a list of mentions
// (iss-2609091801075902). The first detector checked that each salvaging event
// was named somewhere in the paragraph, so a paragraph saying "three events
// self-provision" beside a manifest wiring four passed, because the fourth was
// mentioned for an unrelated reason. A count is the statement that goes stale
// between a change and its record, so every count here is derived from the
// manifest, and every event the paragraph names must be one the manifest wires.

// hookEventVocabulary is every hook event name a paragraph could plausibly
// name: the ones the manifest wires plus the host's other events, so a
// paragraph that names an event the manifest does not wire is caught as an
// invention rather than ignored as an unknown word.
var hookEventVocabulary = []string{
	"SessionStart", "SessionEnd", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"PostToolUseFailure", "PreCompact", "Stop", "SubagentStart", "SubagentStop",
	"Notification", "PermissionRequest",
}

// countWords are the number words a paragraph states a count with. "one" and
// "ten" are left out: the paragraphs use them for a line and a window, never
// for a count of events.
var countWords = map[string]int{
	"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9,
}

// hookSets is what the shipped manifest wires: every event, the events whose
// command reaches bootstrap.sh, and those of them whose salvage reads the
// ten-minute `.bootstrap.attempt` throttle (`-mmin`) rather than running on
// every firing.
type hookSets struct {
	wired, salvaging, throttled []string
}

// exceptions are the wired events that never reach bootstrap.sh.
func (s hookSets) exceptions() []string {
	var out []string
	for _, e := range s.wired {
		if !slices.Contains(s.salvaging, e) {
			out = append(out, e)
		}
	}
	return out
}

func shippedHookSets(t *testing.T) hookSets {
	t.Helper()
	data, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatalf("reading the committed hooks manifest: %v", err)
	}
	var doc sessionStartHooks
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}
	var s hookSets
	for event, entries := range doc.Hooks {
		s.wired = append(s.wired, event)
		for _, e := range entries {
			for _, h := range e.Hooks {
				if strings.Contains(h.Command, "bootstrap.sh") && !slices.Contains(s.salvaging, event) {
					s.salvaging = append(s.salvaging, event)
					if strings.Contains(h.Command, "-mmin") {
						s.throttled = append(s.throttled, event)
					}
				}
			}
		}
	}
	sort.Strings(s.wired)
	sort.Strings(s.salvaging)
	sort.Strings(s.throttled)
	return s
}

// countClaims are the count statements the paragraphs make, each with the set
// the manifest derives it from. A count word the passage uses outside every
// one of these is itself a finding: a new count in the prose fails until this
// test derives it, rather than passing unread.
var countClaims = []struct {
	re   *regexp.Regexp
	what string
	set  func(hookSets) []string
}{
	{regexp.MustCompile(`(?i)\b(\w+)\s+event\s+types\b`), "wired events", func(s hookSets) []string { return s.wired }},
	{regexp.MustCompile(`(?i)\b(\w+)\s+of\s+them\s+self-provision`), "self-provisioning events", func(s hookSets) []string { return s.salvaging }},
	{regexp.MustCompile(`(?i)\bthe\s+(\w+)\s+throttled\s+events\b`), "throttled events", func(s hookSets) []string { return s.throttled }},
	{regexp.MustCompile(`(?i)\bthe\s+last\s+(\w+)\s+self-provision`), "throttled events", func(s hookSets) []string { return s.throttled }},
	{regexp.MustCompile(`(?i)\bthe\s+(\w+)\s+exceptions\b`), "exceptions that download nothing", hookSets.exceptions},
}

var countWordRe = regexp.MustCompile(`(?i)\b(two|three|four|five|six|seven|eight|nine)\b`)

// selfProvisionFindings is the detector: every way passage misstates the sets.
// It asserts the set rather than the mentions — the events named are exactly
// the events wired, no fewer and none invented — and it derives every count.
func selfProvisionFindings(passage string, s hookSets) []string {
	var out []string
	named := map[string]bool{}
	for _, event := range hookEventVocabulary {
		if regexp.MustCompile(`\b` + event + `\b`).MatchString(passage) {
			named[event] = true
		}
	}
	for _, event := range s.wired {
		if !named[event] {
			out = append(out, "does not name "+event+", which the manifest wires")
		}
	}
	for event := range named {
		if !slices.Contains(s.wired, event) {
			out = append(out, "names "+event+", which the manifest does not wire")
		}
	}
	consumed := map[int]bool{}
	for _, c := range countClaims {
		for _, m := range c.re.FindAllStringSubmatchIndex(passage, -1) {
			word := strings.ToLower(passage[m[2]:m[3]])
			n, isCount := countWords[word]
			if !isCount {
				continue
			}
			consumed[m[2]] = true
			if want := len(c.set(s)); n != want {
				out = append(out, fmt.Sprintf("says %q (%d %s) where the manifest wires %d: %v",
					passage[m[0]:m[1]], n, c.what, want, c.set(s)))
			}
		}
	}
	for _, m := range countWordRe.FindAllStringIndex(passage, -1) {
		if !consumed[m[0]] {
			out = append(out, fmt.Sprintf("states a count, %q, that no claim here derives from the manifest", passage[m[0]:m[1]]))
		}
	}
	sort.Strings(out)
	return out
}

func TestSelfProvisionParagraphsMatchTheShippedSets(t *testing.T) {
	sets := shippedHookSets(t)
	for _, rel := range []string{
		".abcd/development/brief/04-surfaces/01-ahoy.md",
		".abcd/development/brief/05-internals/03-configuration.md",
	} {
		passage := bootstrapPassage(t, rel, briefChapter(t, rel))
		for _, f := range selfProvisionFindings(passage, sets) {
			t.Errorf("%s: %s", rel, f)
		}
	}
}

// The detector proves itself: a paragraph that disagrees with the manifest in
// either direction — a wrong count, an invented event — is a finding.
func TestSelfProvisionDetectorSeesAWrongCountAndAnInventedEvent(t *testing.T) {
	sets := hookSets{
		wired:     []string{"PreCompact", "PreToolUse", "SessionEnd", "SessionStart", "SubagentStop", "UserPromptSubmit"},
		salvaging: []string{"PreCompact", "PreToolUse", "SessionStart", "UserPromptSubmit"},
		throttled: []string{"PreCompact", "PreToolUse", "UserPromptSubmit"},
	}
	good := "The manifest wires six event types. Four of them self-provision: `SessionStart`, " +
		"`UserPromptSubmit`, `PreToolUse` and `PreCompact`; the three throttled events keep a " +
		"`.bootstrap.attempt` marker. `SessionEnd` and `SubagentStop` are the two exceptions."
	if f := selfProvisionFindings(good, sets); len(f) != 0 {
		t.Fatalf("a paragraph that matches the manifest drew findings: %v", f)
	}
	for name, bad := range map[string]string{
		"wrong salvage count": strings.Replace(good, "Four of them", "Three of them", 1),
		"wrong wired count":   strings.Replace(good, "six event types", "five event types", 1),
		"wrong throttled":     strings.Replace(good, "the three throttled", "the two throttled", 1),
		"wrong exceptions":    strings.Replace(good, "the two exceptions", "the three exceptions", 1),
		"invented event":      good + " `PostToolUse` provisions too.",
		"underived count":     good + " Seven shims share the rung.",
	} {
		if f := selfProvisionFindings(bad, sets); len(f) == 0 {
			t.Errorf("%s: the detector passed a paragraph that disagrees with the manifest:\n%s", name, bad)
		}
	}
}
