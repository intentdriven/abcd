package ahoy

import (
	"regexp"
	"strings"
	"testing"
)

// machineLocalRefs returns the 1-based line numbers of every machine-local path
// reference in data: a home-relative `~/` path, or the private templates directory
// the adopt phase used to reach into.
//
// It is the detector iss-87 asked for, and it is deliberately blunt. A path under
// `~` is unavailable to every reader but the one who wrote it, so an adopt step
// that reaches for one degrades to nothing on a fresh clone — silently, because the
// step was written "if present". Blunt is the point: there is no legitimate `~`
// reference in an onboarding record, so there is no exception to reason about.
func machineLocalRefs(data []byte) []int {
	var hits []int
	for i, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.Contains(line, "~/"),
			strings.Contains(line, "$HOME/"),
			strings.Contains(line, ".agents/templates"):
			hits = append(hits, i+1)
		}
	}
	return hits
}

// TestOnboardingIsSelfContained is itd-162's gate. The adopt phase reached for a
// pre-commit config and a prepare-commit-msg hook under a maintainer-local
// templates directory, both "if present" — so on any machine but that one the step
// did nothing at all, and the adoption silently degraded against loud-staging.
// Every asset it applies now resolves from this record or from the binary.
func TestOnboardingIsSelfContained(t *testing.T) {
	// The preparation workflow on the ahoy page is what the adopt phase reads and
	// applies from. Every asset it names must resolve from inside this record or
	// from the binary; a path outside both is a template only the machine that
	// wrote it has. The page's later account of the binary install names the
	// binary's own user-scope paths, which are not assets the workflow applies.
	if hits := machineLocalRefs([]byte(preparationWorkflow(t))); len(hits) > 0 {
		t.Errorf("%s's preparation workflow references a machine-local path on line(s) %v (counted from the install heading); "+
			"every adopt-phase asset must resolve from this record or from the binary, never from a path only one machine has", ahoyPage, hits)
	}
}

// TestMachineLocalRefsCatchesAReintroduction is the detector's own must-fail half.
// A gate nobody has watched fire is an assertion, not a check: without this,
// deleting the scan's body would leave the gate above green for ever.
func TestMachineLocalRefsCatchesAReintroduction(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"tilde template path", "1. **Commit gates.** template at\n   `~/ABCDevelopment/.agents/templates/pre-commit-config.yaml`, if present\n"},
		{"HOME-relative path", "install the hook from `$HOME/.agents/templates/` if present\n"},
		{"the templates directory alone", "copy it out of the .agents/templates directory\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if hits := machineLocalRefs([]byte(tc.body)); len(hits) == 0 {
				t.Errorf("the self-containment scan missed a machine-local reference in:\n%s", tc.body)
			}
		})
	}
	// And the must-pass half: the shapes the rewritten record actually uses must not
	// trip it, or the gate would push the record back towards the `~` path it fixes.
	clean := "run `\"${CLAUDE_PLUGIN_ROOT}/abcd\" ahoy install --attribution`, then\n" +
		"point git at the committed hooks: `git config core.hooksPath .githooks`\n"
	if hits := machineLocalRefs([]byte(clean)); len(hits) > 0 {
		t.Errorf("the scan flags the binary-resolved form on line(s) %v; it must only catch machine-local paths", hits)
	}
}

// creatingVerb matches the words an instruction uses to make a file: create,
// scaffold, symlink, link or copy it, link one to another, keep a copy of one
// as another, mirror, write, make, add, put, place, generate, produce, touch,
// set up, save, duplicate, and the commands `ln` and `cp`. It is blunt on
// purpose, as machineLocalRefs is: the page names a tool's own file only to
// say what setup finds, so no sentence on it needs to pair one with a making
// verb, and the page is worded to keep it so. "Place" counts only with an
// object after it, so the idiom "in place of" is not read as one. The class
// is a tripwire for the ordinary ways of saying "make this file", proven
// against the rewordings below; it is not a reading of the page's meaning.
var creatingVerb = regexp.MustCompile(`(?i)\b(?:(creat(e|es|ed|ing)|scaffold(s|ed|ing)?|symlink(s|ed|ing)?|link(s|ed|ing)?\s+(it|them)|as\s+an?\s+(sym)?link|link(s|ed|ing)?\s+\S+\s+to|cop(y|ies|ied|ying)\s+(it|them|AGENTS|of\s+\S+\s+as)|mirror(s|ed|ing)?|writ(e|es|ten|ing)|mak(e|es|ing)|made|add(s|ed|ing)?|put(s|ting)?|plac(e|es|ed|ing)\s+(a|an|the|it|them)|generat(e|es|ed|ing)|produc(e|es|ed|ing)|touch(es|ed|ing)?|set(s|ting)?\s+up|sav(e|es|ed|ing)|duplicat(e|es|ed|ing))\b|(ln|cp)\s)`)

// sentenceBreak splits a page into sentences and list items: a full stop,
// question or exclamation mark before whitespace, a blank line, or the start
// of a bullet or numbered item. A file name's own dot is never followed by
// whitespace, so `CLAUDE.md` stays whole.
var sentenceBreak = regexp.MustCompile(`[.!?]\s+|\n\s*\n|\n\s*([-*]|\d+\.)\s`)

// toolFileInstructions returns every sentence of page that names a file in
// toolConventionsFiles together with a verb that makes a file, whitespace
// collapsed.
func toolFileInstructions(page string) []string {
	var hits []string
	for _, sentence := range sentenceBreak.Split(page, -1) {
		sentence = strings.Join(strings.Fields(sentence), " ")
		if !creatingVerb.MatchString(sentence) {
			continue
		}
		for _, f := range toolConventionsFiles {
			name := regexp.MustCompile(`(^|[^A-Za-z0-9_./-])` + regexp.QuoteMeta(f.Rel) + `($|[^A-Za-z0-9_.-]|\.($|[^A-Za-z0-9]))`)
			if name.MatchString(sentence) {
				hits = append(hits, sentence)
				break
			}
		}
	}
	return hits
}

// TestAhoyPageScaffoldsNoToolConventionsFile is the page half of
// itd-2610030814013772's A1 (spc-2610031156364295, step 6): the preparation
// workflow, which folded into the ahoy page's install section
// (spc-2610100613109045, step 4), creates AGENTS.md and no tool's own conventions file, link or copy, since
// AGENTS.md is the one conventions file abcd writes
// (adr-2610030814023326). The detector is watched fire first on the shapes the
// page once carried, so an emptied scan cannot leave the gate green.
func TestAhoyPageScaffoldsNoToolConventionsFile(t *testing.T) {
	for _, bad := range []string{
		"3. **AGENTS.md.** Merge into the repo's `AGENTS.md`\n   (create it if absent, with `CLAUDE.md` as a symlink to it):\n",
		"Then run `ln -s AGENTS.md GEMINI.md` so Gemini CLI reads it.\n",
		"Copy it to `.github/copilot-instructions.md` as well.\n",
		"Scaffold `.claude/CLAUDE.md` beside it.\n",
		"Write a `.cursorrules` that names AGENTS.md.\n",
		// The rewordings review-agentsStep6 fed the first, narrower verb class,
		// which caught none of them.
		"Add a `CLAUDE.md` that points at it.\n",
		"Make `GEMINI.md` a hard link: `ln AGENTS.md GEMINI.md`\n",
		"Put a `.cursorrules` beside it.\n",
		"Generate `.github/copilot-instructions.md` from it.\n",
		"Keep a copy of AGENTS.md as `CLAUDE.md`.\n",
		"Link `CLAUDE.md` to AGENTS.md.\n",
		"Run `cp AGENTS.md CLAUDE.md` once.\n",
		"Place a `.rules` file at the root.\n",
		"Touch `.claude/CLAUDE.md` so Claude Code finds it.\n",
		"Set up `GEMINI.md` for Gemini CLI.\n",
		"Save it as `CLAUDE.md` too.\n",
		"Duplicate it as `.cursorrules`.\n",
	} {
		if len(toolFileInstructions(bad)) == 0 {
			t.Errorf("the scan missed an instruction making a tool's own file in:\n%s", bad)
		}
	}
	for _, good := range []string{
		"Create it if absent. No other tool's conventions file is made, as a link or a copy.\n",
		"A `CLAUDE.md` that only repeats AGENTS.md (a link to it, or a copy of it) is named; setup offers to retire it.\n",
		"Read `.abcd/rules.json` and `CLAUDE.local.md`; create `.abcd/work/`.\n",
		"A file an agent tool reads in place of `AGENTS.md` (`CLAUDE.md`, `GEMINI.md`) is named as setup names it.\n",
	} {
		if hits := toolFileInstructions(good); len(hits) > 0 {
			t.Errorf("the scan flags a sentence that makes no tool's file: %q", hits)
		}
	}

	for _, hit := range toolFileInstructions(readAhoyPage(t)) {
		t.Errorf("%s instructs making a tool's own conventions file; it creates AGENTS.md and no other, "+
			"link or copy (adr-2610030814023326):\n  %s", ahoyPage, hit)
	}
}
