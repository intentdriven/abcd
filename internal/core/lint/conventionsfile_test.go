package lint_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
)

// toolConventionsFilesAtRoot are the files a tool reads in place of AGENTS.md
// that this repository once carried at its root, as committed links to
// AGENTS.md (itd-2610030814013772, criterion A7).
var toolConventionsFilesAtRoot = []string{"CLAUDE.md", "GEMINI.md"}

// TestRepositoryKeepsOneConventionsFile holds abcd's own tree to the rule that
// AGENTS.md is the one conventions file (adr-2610030814023326). A CLAUDE.md or
// GEMINI.md at the root, as a link or a file, is a second conventions file: a
// clone without symbolic-link support checks a link out as a text file holding
// "AGENTS.md", and a host that finds such a file reads it in place of AGENTS.md,
// so that clone loads no instructions at all. Both halves are asserted: git's
// own list of committed files, and the working tree, so a restored link fails
// here before it is committed.
func TestRepositoryKeepsOneConventionsFile(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	for _, name := range toolConventionsFilesAtRoot {
		if _, err := os.Lstat(filepath.Join(root, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s exists at the repository root (lstat err=%v); AGENTS.md is the one conventions "+
				"file this repository keeps, and a tool that finds %s reads it in place of AGENTS.md", name, err, name)
		}
	}

	args := append([]string{"-C", root, "ls-files", "-z", "--"}, toolConventionsFilesAtRoot...)
	cmd := exec.Command("git", args...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("git ls-files unavailable in %s: %v", root, err)
	}
	if listed := strings.TrimRight(string(out), "\x00"); listed != "" {
		t.Errorf("git commits a tool's own conventions file at the root: %q; AGENTS.md is the one "+
			"conventions file this repository keeps", strings.Split(listed, "\x00"))
	}
}

// canaryLineRe matches the conventions-file check line and captures its word.
var canaryLineRe = regexp.MustCompile(`(?m)^Conventions-file check word: ([a-z]+)\. `)

// agentsMDReadCap is the smallest instruction-file limit among the readers the
// research names (32 KiB, past which one host stops reading): a canary past it
// would measure the cap, not whether the file loads.
const agentsMDReadCap = 32768

// TestAgentsMDCanarySitsInTheFirstSection holds the canary word that the dated
// receipt of criterion A8 asks a fresh session for. The line sits in AGENTS.md's
// first section, between the title and abcd's managed block, so setup's drift
// refresh never rewrites it; inside the first 32,768 bytes, so the receipt
// measures whether the file loads and not how much of it a host reads; and its
// word appears in no other committed file, so a session can only name it by
// having loaded AGENTS.md.
func TestAgentsMDCanarySitsInTheFirstSection(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")

	loc := canaryLineRe.FindStringSubmatchIndex(body)
	if loc == nil {
		t.Fatalf("AGENTS.md carries no line matching %q; the canary receipt (itd-2610030814013772, A8) "+
			"asks a fresh session for that word", canaryLineRe)
	}
	word := body[loc[2]:loc[3]]

	if !strings.HasPrefix(body, "# ") {
		t.Fatalf("AGENTS.md does not open with its title; the canary's place is read from it")
	}
	begin := strings.Index(body, "<!-- BEGIN ABCD -->")
	if begin < 0 {
		t.Fatalf("AGENTS.md carries no BEGIN ABCD fence; the canary's place is read against it")
	}
	if loc[0] > begin {
		t.Errorf("the canary line starts at byte %d, after the managed block's BEGIN fence at byte %d; "+
			"it belongs between the title and the fence, where setup's refresh never rewrites it", loc[0], begin)
	}
	if section := strings.Index(body, "\n## "); section >= 0 && loc[0] > section {
		t.Errorf("the canary line starts at byte %d, past the first section heading at byte %d", loc[0], section)
	}
	end := loc[0] + strings.IndexByte(body[loc[0]:]+"\n", '\n')
	if end > agentsMDReadCap {
		t.Errorf("the canary line ends at byte %d, past the first %d bytes a host may read", end, agentsMDReadCap)
	}
	if strings.Count(body, word) != 1 {
		t.Errorf("AGENTS.md names the canary word %d times; it appears once, on its line", strings.Count(body, word))
	}

	cmd := exec.Command("git", "-C", root, "grep", "-l", "-i", "-F", word, "--", ".", ":(exclude)AGENTS.md")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	var exit *exec.ExitError
	switch {
	case err == nil:
		t.Errorf("the canary word is committed outside AGENTS.md, in %q; a session could then name it "+
			"without loading AGENTS.md", strings.Fields(string(out)))
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		// No other committed file names it.
	default:
		t.Skipf("git grep unavailable in %s: %v", root, err)
	}
}

// TestAgentsMDFitsTheSmallestReadersCap holds the whole of AGENTS.md inside the
// smallest instruction-file limit among its native readers (iss-2610031012543135).
// The one-conventions-file rule (adr-2610030814023326) counts files, not bytes:
// a reader that stops at 32,768 bytes works without every convention past the
// cut, so for that reader a longer file is not one file. Detail that would push
// the file past the cap moves under the on-demand rules loader
// (.abcd/rules.json), which injects it when a prompt recalls it.
func TestAgentsMDFitsTheSmallestReadersCap(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")
	if len(body) >= agentsMDReadCap {
		t.Errorf("AGENTS.md is %d bytes, at or past the %d bytes a host may read; move detail under the "+
			"rules loader (.abcd/rules.json) rather than let a reader stop before the end",
			len(body), agentsMDReadCap)
	}
}

// TestAgentsMDDetailMovedUnderTheLoaderIsRecalled holds the other half of the
// move (iss-2610031012543135): each section of AGENTS.md that keeps only the
// short form of its rules names the repository domain carrying the whole of
// them, and that domain exists, is recalled by a prompt about its subject, and
// still carries a fact that left AGENTS.md. Without it the cap above is met as
// well by deleting the detail as by moving it.
func TestAgentsMDDetailMovedUnderTheLoaderIsRecalled(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")
	var repo rules.RuleSet
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/rules.json")), &repo); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	if err := rules.Validate(repo); err != nil {
		t.Fatalf(".abcd/rules.json does not validate: %v", err)
	}
	for _, tc := range []struct {
		section, domain, prompt, fact string
	}{
		{"## Concurrent sessions\n", "CONCURRENCY",
			"start a second session in its own worktree", "TestPayloadTreeImplementationsResolveIdentically"},
		{"## Attribution and acknowledgements\n", "ATTRIBUTION",
			"revert that commit with the right trailer", "DEPENDENCY_REAUTHOR_APP_ID"},
	} {
		t.Run(tc.domain, func(t *testing.T) {
			start := strings.Index(body, tc.section)
			if start < 0 {
				t.Fatalf("AGENTS.md has no %q section", strings.TrimSpace(tc.section))
			}
			section := body[start+len(tc.section):]
			if next := strings.Index(section, "\n## "); next >= 0 {
				section = section[:next]
			}
			if !strings.Contains(section, "`"+tc.domain+"`") {
				t.Errorf("AGENTS.md's %q section does not name the %s domain that carries its detail",
					strings.TrimSpace(tc.section), tc.domain)
			}
			d, ok := repo.Domains[tc.domain]
			if !ok {
				t.Fatalf(".abcd/rules.json declares no %s domain", tc.domain)
			}
			if !strings.Contains(strings.Join(d.Rules, "\n"), tc.fact) {
				t.Errorf("the %s domain no longer carries %q, detail that left AGENTS.md for it", tc.domain, tc.fact)
			}
			recalled := false
			for _, m := range repo.Match(tc.prompt) {
				recalled = recalled || m.Name == tc.domain
			}
			if !recalled {
				t.Errorf("the prompt %q does not recall the %s domain", tc.prompt, tc.domain)
			}
		})
	}
}

// TestVerifierCopyGoesOutsideEveryWorkingTree holds the verifier-copy rule, in
// both its short form (AGENTS.md, § Concurrent sessions) and its long form (the
// CONCURRENCY domain of .abcd/rules.json), to naming where the copy goes
// (iss-2610090642392144). A copy extracted under the local tier's scratch/ sits
// inside the live worktree; it is not a checkout, so every verb run from it that
// reads git or the record store resolves upward to the worktree around it and
// verifies that tree, not the copy. Both forms must say the copy goes outside
// every working tree and name the two homes that are: the session's scratchpad
// and a directory under the machine-scoped abcd home (abcdhome.Display). The
// itd-193 discipline and the users-directory principle, which both forms lean
// on, are held to the same three phrases.
func TestVerifierCopyGoesOutsideEveryWorkingTree(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	body := readRepoFile(t, root, "AGENTS.md")
	var repo rules.RuleSet
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/rules.json")), &repo); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}

	const shortHead = "- **A verifier works on a copy**"
	start := strings.Index(body, shortHead)
	if start < 0 {
		t.Fatalf("AGENTS.md has no %q rule", shortHead)
	}
	short := body[start+len(shortHead):]
	if next := strings.Index(short, "\n- "); next >= 0 {
		short = short[:next]
	}

	const longHead = "**A verifier works on a copy.**"
	long := ""
	for _, r := range repo.Domains["CONCURRENCY"].Rules {
		if strings.HasPrefix(r, longHead) {
			long = r
		}
	}
	if long == "" {
		t.Fatalf("the CONCURRENCY domain of .abcd/rules.json has no rule opening %q", longHead)
	}

	// The two records both forms lean on say it too: the discipline they cite
	// (itd-193) and the principle the CONCURRENCY domain links as the stance.
	// Each is held by the paragraph its anchor falls in.
	paragraph := func(rel, anchor string) string {
		for _, p := range strings.Split(readRepoFile(t, root, rel), "\n\n") {
			if flat := strings.Join(strings.Fields(p), " "); strings.Contains(flat, anchor) {
				return flat
			}
		}
		t.Fatalf("%s has no paragraph saying %q", rel, anchor)
		return ""
	}
	itd193 := paragraph(".abcd/development/intents/disciplines/itd-193-a-verifier-works-on-a-copy-no-agent.md",
		"Before reporting, the verifier proves the original is untouched")
	principle := paragraph(".abcd/development/principles/the-users-directory-is-theirs.md", "a verifier's copy goes")

	for name, text := range map[string]string{
		"AGENTS.md": short, "the CONCURRENCY domain": long, "itd-193": itd193, "the users-directory principle": principle,
	} {
		flat := strings.Join(strings.Fields(text), " ")
		for _, want := range []string{"outside every working tree", "scratchpad", abcdhome.Display() + "/"} {
			if !strings.Contains(flat, want) {
				t.Errorf("%s's verifier-copy rule does not say %q; a copy made inside a worktree is "+
					"read as that worktree by every verb run from it", name, want)
			}
		}
	}
}

// worktreeStoreIntent is the intent that carries the worktree store's own verbs
// (add, list, reclaim), which the CONCURRENCY rule names as the reason the
// store has none yet.
const worktreeStoreIntent = "itd-2609091014076309"

// TestWorktreeStoreRuleNamesTheIntentsActualState holds every present-state
// sentence that says why the worktree store has no verbs to the lifecycle folder
// its intent actually sits in (iss-2610090744073409). The CONCURRENCY rule of
// .abcd/rules.json said the intent was in drafts/ while its own link pointed at
// planned/, so the prose contradicted the link beside it; the build surface page
// and the lane's worktree code repeated the claim as "a draft". The rule must
// name the folder the record is in and say the store has no verbs because that
// intent has not shipped; no site may call a planned intent a draft. The
// planning records that wait on the store (itd-148, its spec spc-42, and
// iss-213, which itd-148 carries) went on calling it "the store draft" after it
// was planned (iss-2610111926302792), so they are sites too, found by id in
// whichever lifecycle folder holds them. A site is the list item, prose
// sentence or front-matter line that names the intent or says "store draft"
// (worktreeStoreSites), never the whole paragraph, so a draft word said of
// another intent in the same list is not blamed on this one
// (iss-2610111926302443). Dated records (ADRs, the decision log, research
// notes) and the dated "## Deferral" sections inside a record say what was true
// when they were written and are not held here.
func TestWorktreeStoreRuleNamesTheIntentsActualState(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	matches, err := filepath.Glob(filepath.Join(root, ".abcd", "development", "intents", "*", worktreeStoreIntent+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want exactly one record of %s in the intent store, got %v (err=%v)", worktreeStoreIntent, matches, err)
	}
	bucket := filepath.Base(filepath.Dir(matches[0]))
	if bucket == "shipped" {
		t.Fatalf("%s has shipped: the store has verbs now, so the CONCURRENCY rule's \"no verbs yet\" is stale "+
			"and this test's premise with it", worktreeStoreIntent)
	}

	var repo rules.RuleSet
	if err := json.Unmarshal([]byte(readRepoFile(t, root, ".abcd/rules.json")), &repo); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	rule := ""
	for _, r := range repo.Domains["CONCURRENCY"].Rules {
		if strings.Contains(r, worktreeStoreIntent) {
			rule = strings.Join(strings.Fields(r), " ")
		}
	}
	if rule == "" {
		t.Fatalf("the CONCURRENCY domain of .abcd/rules.json has no rule naming %s", worktreeStoreIntent)
	}
	for _, want := range []string{"/intents/" + bucket + "/" + worktreeStoreIntent + "-", bucket, "has not shipped"} {
		if !strings.Contains(rule, want) {
			t.Errorf("the CONCURRENCY rule naming %s does not say %q; the intent is in %s/ and has not shipped, "+
				"which is why the store has no verbs yet", worktreeStoreIntent, want, bucket)
		}
	}

	// Every present-state passage naming the intent.
	sites := map[string][]string{"the CONCURRENCY rule": {rule}}
	rels := []string{
		".abcd/development/brief/04-surfaces/34-build.md",
		"internal/core/implement/loop/lane.go",
	}
	// The planning records waiting on the store, wherever their lifecycle
	// folder puts them.
	for _, pattern := range []string{
		".abcd/development/intents/*/itd-148-*.md",
		".abcd/development/specs/*/spc-42-*.md",
		".abcd/work/issues/*/iss-213-*.md",
	} {
		found, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil || len(found) != 1 {
			t.Fatalf("want exactly one record matching %s, got %v (err=%v)", pattern, found, err)
		}
		rel, err := filepath.Rel(root, found[0])
		if err != nil {
			t.Fatalf("relative path of %s: %v", found[0], err)
		}
		rels = append(rels, filepath.ToSlash(rel))
	}
	for _, rel := range rels {
		sites[rel] = append(sites[rel], worktreeStoreSites(readRepoFile(t, root, rel))...)
		if len(sites[rel]) == 0 {
			t.Errorf("%s no longer names %s; update this test's list of sites", rel, worktreeStoreIntent)
		}
	}
	if bucket == "drafts" {
		return
	}
	for name, paras := range sites {
		for _, p := range paras {
			for _, stale := range staleDraftWords(p) {
				t.Errorf("%s calls %s %q, but the intent is in %s/:\n%s", name, worktreeStoreIntent, stale, bucket, p)
			}
		}
	}
}

// withoutDeferralSections drops each dated "## Deferral <date>" section of a
// record, from its heading to the next level-two heading or the end: such a
// section repeats the deferral reason as it read on that date, so it is a dated
// note, not present-state prose.
func withoutDeferralSections(text string) string {
	var kept []string
	skipping := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			skipping = strings.HasPrefix(line, "## Deferral ")
		}
		if !skipping {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// worktreeStoreSites returns the passages of one file that
// TestWorktreeStoreRuleNamesTheIntentsActualState judges, each flattened to
// single spaces (iss-2610111926302443). A passage is a list item, a sentence of
// prose, or a line of front matter, so a draft word said of another intent in
// the same list or paragraph is not blamed on the store intent; a wrapped line
// is never the unit, because the stale shapes this check was written against
// ("(a\n  draft)", "a draft\n// (itd-...") broke across lines. A passage is a
// site when it names the store intent or says "store draft". Dated
// "## Deferral" sections are skipped.
func worktreeStoreSites(text string) []string {
	var sites []string
	for _, u := range siteUnits(withoutDeferralSections(text)) {
		if strings.Contains(u, worktreeStoreIntent) || strings.Contains(u, "store draft") {
			sites = append(sites, u)
		}
	}
	return sites
}

// listItemStart matches the first line of a Markdown list item, nested or not.
var listItemStart = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s`)

// sentenceBreak matches the end of a sentence in flattened prose: its stop, any
// closing marks after it (submatch 1 ends there), a space, and the capital or
// emphasis that opens the next sentence.
var sentenceBreak = regexp.MustCompile("([.!?][)\"'*_`\\]]*) [A-Z*_]")

// siteUnits splits a file into the passages worktreeStoreSites judges: each
// front-matter line, each list item with its continuation lines, and each
// sentence of a prose paragraph. A Go comment's "//" leader is dropped first,
// so a bare "//" line separates paragraphs as a blank line does.
func siteUnits(text string) []string {
	lines := strings.Split(text, "\n")
	var units []string
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				for _, l := range lines[1:i] {
					if flat := flatten(l); flat != "" {
						units = append(units, flat)
					}
				}
				lines = lines[i+1:]
				break
			}
		}
	}
	var block []string
	item := false
	flush := func() {
		flat := flatten(strings.Join(block, " "))
		block = nil
		if flat == "" {
			return
		}
		if item {
			units = append(units, flat)
			return
		}
		start := 0
		for _, m := range sentenceBreak.FindAllStringSubmatchIndex(flat, -1) {
			units = append(units, flat[start:m[3]])
			start = m[3] + 1
		}
		units = append(units, flat[start:])
	}
	for _, l := range lines {
		l = strings.TrimLeft(l, " \t")
		if strings.HasPrefix(l, "//") {
			l = strings.TrimPrefix(l, "//")
		}
		switch {
		case strings.TrimSpace(l) == "":
			flush()
		case listItemStart.MatchString(l):
			flush()
			item = true
			block = append(block, l)
		default:
			if len(block) == 0 {
				item = false
			}
			block = append(block, l)
		}
	}
	flush()
	return units
}

// flatten collapses every run of white space in s to one space.
func flatten(s string) string { return strings.Join(strings.Fields(s), " ") }

// staleDraftWords returns the words in one site that call the store intent a
// draft, which a planned intent is not.
func staleDraftWords(site string) []string {
	var found []string
	for _, stale := range []string{"drafts/", "a draft", "drafted", "in drafts", "store draft", "that draft"} {
		if strings.Contains(site, stale) {
			found = append(found, stale)
		}
	}
	return found
}

// TestWorktreeStoreSitesNarrowToTheItemNamingTheIntent pins the unit the
// store-intent check judges (iss-2610111926302443). The references list of the
// build surface page is one blank-line paragraph naming six intents, so judging
// the whole paragraph would blame the store intent for "a draft" written beside
// another intent in that list. A site is the list item that names the store
// intent, or the sentence that does in prose, or the front-matter line; the
// stale shapes this check was written against still fail, wrapped across lines
// as they were.
func TestWorktreeStoreSitesNarrowToTheItemNamingTheIntent(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		stale      []string
	}{
		{
			name: "a draft beside another intent in the same list passes",
			text: "## Where this sits\n\n" +
				"- The pace and the window clock: itd-2609201925079472 (a draft) and its\n" +
				"  design record, spc-2609202134341288, which drafted the clock.\n" +
				"- The worktree store the lane's checkout lives in: itd-2609091014076309\n" +
				"  (planned, not yet shipped), the rule it enacts, adr-2609091248200336.\n" +
				"- The pick: itd-2609211116005482, still in drafts/.\n",
		},
		{
			name: "a draft beside another intent in the same prose paragraph passes",
			text: "The clock is a draft (itd-2609201925079472). The store's own verbs are\n" +
				"planned and not yet shipped (itd-2609091014076309), so the lane is a plain\n" +
				"`git worktree add` into the store's path.\n",
		},
		{
			name: "a draft wrapped inside the list item naming the store intent fails",
			text: "- The pace: itd-2609201925079472.\n" +
				"- The worktree store the lane's checkout lives in: itd-2609091014076309 (a\n" +
				"  draft), the rule it enacts, adr-2609091248200336.\n",
			stale: []string{"a draft"},
		},
		{
			name: "a draft wrapped in a Go comment sentence fails",
			text: "// cut from the default branch. The store's own verbs are a draft\n" +
				"// (itd-2609091014076309), so the lane is a plain `git worktree add` into the\n" +
				"// store's path until they ship.\n",
			stale: []string{"a draft"},
		},
		{
			name: "the store draft and that draft in the sentence naming the intent fail",
			text: "Where the worktree lives is settled elsewhere: a per-change\n" +
				"worktree goes in the machine-scoped store the store draft\n" +
				"(itd-2609091014076309) describes, and\n" +
				"that draft owns the verbs that add, list and clear away worktrees.\n",
			stale: []string{"store draft", "that draft"},
		},
		{
			name: "a list item saying store draft without the id fails",
			text: "1. **The store draft is planned first**: nothing from the store\n" +
				"   draft is built before its planning interview.\n" +
				"2. **A coordinator is exempt.** It is not a draft.\n",
			stale: []string{"store draft"},
		},
		{
			name: "a front-matter line naming the store intent fails, the next line does not",
			text: "---\nid: \"iss-213\"\n" +
				"deferral_reason: \"Blocked on the store draft itd-2609091014076309.\"\n" +
				"note: \"itd-2609201925079472 is a draft.\"\n---\n\nBody.\n",
			stale: []string{"store draft"},
		},
		{
			name: "a dated Deferral section is skipped",
			text: "Body naming itd-2609091014076309.\n\n## Deferral 2026-09-29\n\n" +
				"Deferred: blocked on the store draft itd-2609091014076309.\n\n## Notes\n\nNone.\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sites := worktreeStoreSites(tc.text)
			if len(sites) == 0 {
				t.Fatalf("no site found in:\n%s", tc.text)
			}
			got := map[string]bool{}
			for _, s := range sites {
				for _, w := range staleDraftWords(s) {
					got[w] = true
				}
			}
			want := map[string]bool{}
			for _, w := range tc.stale {
				want[w] = true
			}
			for w := range want {
				if !got[w] {
					t.Errorf("stale %q not flagged; sites:\n%s", w, strings.Join(sites, "\n"))
				}
			}
			for w := range got {
				if !want[w] {
					t.Errorf("%q flagged against %s, but it is not said of the store intent; sites:\n%s",
						w, worktreeStoreIntent, strings.Join(sites, "\n"))
				}
			}
		})
	}
}
