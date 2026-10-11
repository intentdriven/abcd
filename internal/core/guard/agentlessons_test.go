package guard

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The agent definitions' guard lessons (iss-2610091942156774, the ruling's
// option 2). The rules loader injects the SHELL domain into the session, and
// a sub-agent the session starts is handed none of it: it reads its own
// definition under agents/ and nothing else, so an rm the guard refuses
// reaches it only as a refusal after the fact. Every definition that can run
// a shell — one that declares Bash among its tools, or declares no tools and
// so is given all of them — carries a generated block restating the lessons
// named in agentLessonEntries, rendered from the registry exactly as the
// SHELL domain renders them (Entry.Lesson), so the two planes teach one text.
// An entry's words are edited in defaults/guard.json; this test then fails
// until the block is regenerated with -update.

// updateAgentLessons rewrites the agent definitions' blocks from the registry
// instead of failing on a stale one.
var updateAgentLessons = flag.Bool("update", false, "rewrite the agent definitions' guard-lessons blocks from the registry")

// agentLessonEntries are the registry entries every shell-capable agent
// definition restates: the hazards a sub-agent is likely to write and the
// host otherwise asks the person to approve blind. It is the one list; add an
// entry id here and run -update to teach it to every definition.
var agentLessonEntries = []string{"rm-unguarded-variable-path"}

// agentLessonsExempt are the shell-capable definitions that cannot carry the
// block, each with the reason. A definition is exempt only where another
// contract over its text forbids what the block holds; the guard still
// refuses the command it would have taught.
var agentLessonsExempt = map[string]string{
	"scribe.md": "the scribe's definition is held to ledger content alone (internal/core/lint's " +
		"scribecontract_test): it may name no path outside .abcd/work/issues/ and no code point " +
		"outside its own marks, and the lesson names `/*` and the registry's path",
}

const (
	agentLessonsBegin = "<!-- generated: guard-lessons -->"
	agentLessonsEnd   = "<!-- /generated -->"
)

// renderAgentLessons is the block a shell-capable definition carries, from
// its begin marker through its end marker and the newline after it.
func renderAgentLessons(t *testing.T, r Registry) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(agentLessonsBegin + "\n")
	b.WriteString("<!-- Written by `go test ./internal/core/guard -run TestAgentDefinitionsCarryTheGuardLessons -update` from the guard registry (internal/core/guard/defaults/guard.json); edit the entry there, never this block. -->\n\n")
	b.WriteString("## Shell commands the guard refuses\n\n")
	b.WriteString("A sub-agent is not handed the shell rules the session is taught, so the rules for the commands you are most likely to write are restated here. The guard refuses a command that breaks one before it runs; write it the way the rule says from the start.\n\n")
	for _, id := range agentLessonEntries {
		e, ok := r.Entries[id]
		if !ok {
			t.Fatalf("agentLessonEntries names %s, which the bundled registry does not carry", id)
		}
		e.ID = id
		b.WriteString("- " + e.Lesson() + "\n")
	}
	b.WriteString(agentLessonsEnd + "\n")
	return b.String()
}

// runsShell reports whether an agent definition can run a shell: its
// frontmatter declares no tools line, which gives it every tool, or declares
// Bash among them.
func runsShell(def string) bool {
	if !strings.HasPrefix(def, "---\n") {
		return true
	}
	front, _, ok := strings.Cut(def[len("---\n"):], "\n---\n")
	if !ok {
		return true
	}
	for _, line := range strings.Split(front, "\n") {
		if v, found := strings.CutPrefix(line, "tools:"); found {
			for _, tool := range strings.Split(v, ",") {
				if strings.TrimSpace(tool) == "Bash" {
					return true
				}
			}
			return false
		}
	}
	return true
}

// spliceAgentLessons is def with its guard-lessons block replaced by block,
// or with block appended after a blank line where it carries none. ok is
// false for a definition carrying the begin marker more than once, or one
// it never closes: the test never guesses where the block belongs.
func spliceAgentLessons(def, block string) (string, bool) {
	switch strings.Count(def, agentLessonsBegin) {
	case 0:
		return strings.TrimRight(def, "\n") + "\n\n" + block, true
	case 1:
	default:
		return "", false
	}
	start := strings.Index(def, agentLessonsBegin)
	stop := strings.Index(def[start:], agentLessonsEnd)
	if stop < 0 {
		return "", false
	}
	end := start + stop + len(agentLessonsEnd)
	if end < len(def) && def[end] == '\n' {
		end++
	}
	return def[:start] + block + def[end:], true
}

// TestAgentDefinitionsCarryTheGuardLessons holds every agent definition to
// the block its tools call for: a shell-capable one carries it exactly as the
// registry renders it, and one that cannot run a shell carries none.
func TestAgentDefinitionsCarryTheGuardLessons(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	defs, _ := filepath.Glob(filepath.Join(root, "agents", "*.md"))
	sort.Strings(defs)
	if len(defs) < 10 {
		t.Fatalf("found %d agent definitions under %s; the tree has gone missing", len(defs), filepath.Join(root, "agents"))
	}
	block := renderAgentLessons(t, Defaults())
	carrying := 0
	for _, path := range defs {
		rel, _ := filepath.Rel(root, path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		def := string(data)
		if _, exempt := agentLessonsExempt[filepath.Base(path)]; exempt || !runsShell(def) {
			if strings.Contains(def, agentLessonsBegin) {
				t.Errorf("%s is exempt or cannot run a shell, and carries the guard-lessons block; remove it", rel)
			}
			continue
		}
		carrying++
		want, ok := spliceAgentLessons(def, block)
		if !ok {
			t.Errorf("%s carries %s more than once, or never closes it with %s", rel, agentLessonsBegin, agentLessonsEnd)
			continue
		}
		if want == def {
			continue
		}
		if *updateAgentLessons {
			if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("rewrote the guard-lessons block of %s", rel)
			continue
		}
		t.Errorf("%s does not carry the guard-lessons block the registry renders; run "+
			"`go test ./internal/core/guard -run TestAgentDefinitionsCarryTheGuardLessons -update`, "+
			"then bump the definition's prompt_version and add its entry to .abcd/development/agents/CHANGELOG.md", rel)
	}
	if carrying == 0 {
		t.Fatal("no agent definition can run a shell; the tools reading is broken")
	}
}

// TestRunsShellReadsTheToolsLine pins the frontmatter reading the test above
// rests on.
func TestRunsShellReadsTheToolsLine(t *testing.T) {
	for _, tc := range []struct {
		def  string
		want bool
	}{
		{"---\nname: a\n---\nbody\n", true},
		{"---\nname: a\ntools: Read, Grep, Glob, Bash\n---\nbody\n", true},
		{"---\nname: a\ntools: WebSearch, Read, Grep\n---\nbody\n", false},
		{"---\nname: a\ntools: Bashful\n---\nbody\n", false},
		{"no frontmatter\n", true},
	} {
		if got := runsShell(tc.def); got != tc.want {
			t.Errorf("runsShell(%q) = %v, want %v", tc.def, got, tc.want)
		}
	}
}
