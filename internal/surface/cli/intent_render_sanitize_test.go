package cli

import (
	"strings"
	"testing"
)

// intent_render_sanitize_test.go — iss-259. The intent plan/link/create human
// renders interpolate `.Path`, which is the on-disk filename tail. intentFileRe
// is `^itd-[0-9]+.*\.md$`: the `.*` lets a hostile clone name a file
// `itd-10-alpha<ESC>x.md` while its frontmatter slug stays clean kebab-case, so
// the control bytes ride the Path into the terminal render unless it is masked
// through termsafe.Sanitize — the mask the sibling promote render (cli.go:219)
// already applies. Attack runes are built numerically so this source file carries
// none of the invisible characters it defends against.

// intentAttackRunes are the terminal-display attack runes a raw path render would
// replay: ESC (opens an ANSI escape), the 8-bit CSI (C1), and the RLO bidi
// override (Trojan-Source reordering).
var intentAttackRunes = map[string]rune{
	"ESC":          0x1b,
	"C1 CSI":       0x9b,
	"RLO override": 0x202e,
}

func assertNoAttackRunes(t *testing.T, where, out string) {
	t.Helper()
	for name, r := range intentAttackRunes {
		if strings.ContainsRune(out, r) {
			t.Errorf("iss-259: %s render carries a raw %s (U+%04X) attack rune from the path tail; it must be masked like the sibling promote render (cli.go:219)\noutput: %q", where, name, r, out)
		}
	}
}

// poisonTail is a filename tail carrying every attack rune, kept between a clean
// prefix and the `.md` suffix so intentFileRe still matches.
func poisonTail() string {
	var b strings.Builder
	b.WriteString("x")
	for _, r := range intentAttackRunes {
		b.WriteRune(r)
	}
	b.WriteString("y")
	return b.String()
}

func TestIntentPlanRenderMasksPathTail(t *testing.T) {
	repo := intentTestRepo(t)
	// Clean kebab-case slug in the frontmatter; the ATTACK is in the filename tail.
	name := "itd-10-alpha" + poisonTail() + ".md"
	writeRepoFile(t, repo, cliDrafts+"/"+name, cliDraftWithAC("itd-10", "alpha"))

	out := string(runCLI(t, "intent", "plan", "itd-10"))
	if !strings.Contains(out, "abcd intent plan") {
		t.Fatalf("plan render missing header:\n%s", out)
	}
	assertNoAttackRunes(t, "intent plan", out)
}

func TestIntentLinkRenderMasksPathTail(t *testing.T) {
	repo := intentTestRepo(t)
	name := "itd-10-alpha" + poisonTail() + ".md"
	writeRepoFile(t, repo, cliPlanned+"/"+name,
		"---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: standalone\n---\n# alpha\n")
	writeRepoFile(t, repo, cliSpecsOpen+"/spc-3-alpha.md",
		"---\nid: spc-3\nslug: alpha\nintent: itd-10\n---\n# alpha\n")

	out := string(runCLI(t, "intent", "link", "itd-10", "spc-3"))
	if !strings.Contains(out, "abcd intent link") {
		t.Fatalf("link render missing header:\n%s", out)
	}
	assertNoAttackRunes(t, "intent link", out)
}
