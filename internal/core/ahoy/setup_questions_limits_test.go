package ahoy

import (
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// heldQuestion is the value question h as setup builds it (valueQuestion, the
// builder every front door uses), offering every choice the help names. A
// question a flag answers is held with no default, as the install asks it;
// any other with its last choice standing in for the default, so the
// decide-later meaning that names a default is held too.
func heldQuestion(n int, h PromptHelp) question.Question {
	choices := make([]string, len(h.Choices))
	for i, c := range h.Choices {
		choices[i] = c.Value
	}
	def := ""
	if h.Flag == "" && len(choices) > 0 {
		def = choices[len(choices)-1]
	}
	return valueQuestion(n, h, choices, def)
}

// confirmQuestion is the approval whose text ends on its question, tail, as
// setup builds it (SetupConfirmQuestion): the text before the tail is the
// material (an indented line an item of a list), the tail the question, its
// id the one given, and the answers yes, no and decide later.
func confirmQuestion(t *testing.T, n int, id, text, tail string) question.Question {
	t.Helper()
	q := SetupConfirmQuestion(n, text)
	if q.Ask != tail {
		t.Fatalf("%s: the question is %q, not its tail %q", id, q.Ask, tail)
	}
	if q.ID != id {
		t.Fatalf("%s: setup ids the question %q", id, q.ID)
	}
	return q
}

// setupLimitsOwed is every limit a setup question breaks today, as
// "<question id> <rule>". It is empty: iss-2610031236155833's questions all fit
// since itd-2610030814013772 retired claude_md and both (docs_target) and the
// 2026-10-03 fit (artefact_kind, visibility, the machine routing offer).
// oracle_backend is not asked while one answer has an adapter (the 2026-10-03
// ruling), so it owes nothing while it stays unasked. The list may only shrink: a
// question that newly breaks a limit fails, and so does a line here that no
// longer breaks, so the fix deletes its line.
var setupLimitsOwed = map[string]bool{}

// TestEverySetupQuestionPassesTheLimits holds every fixed question the install
// builds, the value questions (every PromptHelp, the status line's elements
// included) and the routing confirmation's two offers, to question.CheckLimits
// through their field view (spc-2610030944505997 step 4: "every fixed question
// abcd builds (setup, routing) passing CheckLimits"). The words are core's, so
// a change to one that breaks a limit fails here, not in front of a person.
// What breaks a limit today is owed by name in setupLimitsOwed. A question the
// install does not ask is not held to them.
func TestEverySetupQuestionPassesTheLimits(t *testing.T) {
	var helps []PromptHelp
	for _, h := range promptHelp {
		if h.Key == "oracle_backend" && !oracleBackendAsked() {
			continue // not asked while one answer has an adapter; it returns with a second
		}
		helps = append(helps, h)
	}
	for k := range statusLineElementAbout {
		h, ok := helpFor(elementPromptPrefix + string(k))
		if !ok {
			t.Fatalf("no help for the status-line element %s", k)
		}
		helps = append(helps, h)
	}
	sort.Slice(helps, func(i, j int) bool { return helps[i].Key < helps[j].Key })
	// The visibility question is held twice: as most repositories see it, and
	// as a repository whose .abcd/ holds tracked records sees it, with public's
	// caveat, its tallest form, labelled so a finding names which form broke.
	tracked := visibilityHelp(true)
	tracked.Key = "visibility (tracked)"
	helps = append(helps, tracked)

	var qs []question.Question
	for i, h := range helps {
		qs = append(qs, heldQuestion(i+1, h))
	}
	qs = append(qs,
		confirmQuestion(t, len(qs)+1, OracleRoutingMachineGapID, machineRoutingQuestion(), oracleRoutingMachineQuestionTail),
		confirmQuestion(t, len(qs)+2, OracleRoutingRepoGapID, repoRoutingQuestion(), oracleRoutingRepoQuestionTail))

	var asks []question.Ask
	for _, q := range qs {
		asks = append(asks, question.Ask{Questions: []question.Question{q}})
	}
	// The README pair (spc-2610031704338417 step 4): Q1 alone, and every
	// part's tab in the calls of at most four the page makes, each built as
	// the run builds it, with its current setting, its change-later line and
	// its example, in the projects that make them tallest.
	asks = append(asks, readmeLimitAsks(t)...)

	broken := map[string]bool{}
	for _, a := range asks {
		for _, f := range question.Check(a) {
			t.Errorf("%s: structural: %s", askLabel(a, f.Tab), f)
		}
		for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
			key := askLabel(a, f.Tab) + " " + string(f.Rule)
			broken[key] = true
			if !setupLimitsOwed[key] {
				t.Errorf("%s: %s", askLabel(a, f.Tab), f)
			}
		}
	}
	for key := range setupLimitsOwed {
		if !broken[key] {
			t.Errorf("%q is owed in setupLimitsOwed but the question no longer breaks it; delete its line", key)
		}
	}
}

// askLabel names the question at tab (counted from one) of a, or the call as
// a whole for tab zero.
func askLabel(a question.Ask, tab int) string {
	if tab < 1 || tab > len(a.Questions) {
		ids := make([]string, len(a.Questions))
		for i, q := range a.Questions {
			ids[i] = q.ID
		}
		return "(" + strings.Join(ids, ", ") + ")"
	}
	return a.Questions[tab-1].ID
}

// readmeLimitAsks is every call the README pair makes, in the projects whose
// words are tallest: a public project with long names, every link, a long
// table of contents and a build workflow, so every part has an example to
// show; a project not declared public, whose Q1 says so; a public project
// where no part can be written, so every tab says why instead; and a public
// project whose eight long-named workflows hold no ci.yml, so the build badge's
// reason would name every one of them.
func readmeLimitAsks(t *testing.T) []question.Ask {
	t.Helper()
	long := "a-rather-long-owner-name-of-39-chars-xx"
	var headings strings.Builder
	headings.WriteString("# Project\n")
	for _, h := range []string{"Installing it on every platform it supports", "Configuring the tool for a team",
		"Running the test suite", "Contributing changes", "Security reports", "Licence and credits"} {
		headings.WriteString("\n## " + h + "\n")
	}
	full := readmeRepo(t, "https://github.com/"+long+"/"+long+"-with-a-longer-repository-name.git", map[string]string{
		"LICENSE": "MIT License\n", "CHANGELOG.md": "x\n", "CONTRIBUTING.md": "x\n", "SECURITY.md": "x\n",
		"CODE_OF_CONDUCT.md": "x\n", ".github/workflows/continuous-integration.yml": "on: push\n",
		"README.md": headings.String(), ".abcd/config.json": `{"repo": {"visibility": "public"}}`,
	})
	private := readmeRepo(t, "", map[string]string{".abcd/config.json": `{"repo": {"visibility": "private"}}`})
	empty := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{
		"README.md": "# Project\n", ".github/workflows/build-and-test.yml": "x\n",
		".github/workflows/release-candidates.yml": "x\n", ".github/workflows/nightly-checks.yml": "x\n",
		".abcd/config.json": `{"repo": {"visibility": "public"}}`,
	})
	workflows := map[string]string{"README.md": "# Project\n", ".abcd/config.json": `{"repo": {"visibility": "public"}}`}
	for _, n := range []string{"build-and-test-on-every-supported-platform", "codeql-analysis-of-the-go-sources",
		"dependency-review-of-every-pull-request", "release-candidates-for-the-next-version",
		"nightly-checks-against-the-latest-toolchain", "documentation-site-render-and-publish",
		"scheduled-secret-scanning-of-the-history", "merge-queue-sweep-of-dropped-entries"} {
		workflows[".github/workflows/"+n+".yml"] = "on: push\n"
	}
	many := readmeRepo(t, "https://github.com/owner/repo.git", workflows)
	var out []question.Ask
	for _, repo := range []string{full, private, empty, many} {
		v, ok := readmeView(repo)
		if !ok {
			t.Fatalf("no README view in %s", repo)
		}
		out = append(out, v.Asks...)
	}
	return out
}
