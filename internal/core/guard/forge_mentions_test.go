package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The mention half of the forge-text check (iss-2610261152125261). In a
// repository that also opted into it (outbound.declare_mentions in
// .abcd/config.json), a `gh pr create`, `gh pr new` or `gh pr edit` whose title
// and body name an issue id without a `Resolves:` or `Refs:` line declaring it
// is blocked under forge-text-mention: RS004 over a pull request, judged before
// the forge keeps it rather than in hosted CI after the push.

const forgeMentionsOptedIn = `{"outbound": {"forge": true, "declare_mentions": true}}`

// disclosed is a body tail carrying the disclosure line, so a case below is
// judged on its mentions alone.
const disclosed = "\n\nAssisted-by: Claude:claude-opus-5-5"

func TestForgeMentionBlocksAnUndeclaredIssueId(t *testing.T) {
	_, reg := forgeRepo(t, forgeMentionsOptedIn)
	cases := []string{
		`gh pr create --title "fix the walk (iss-12)" --body "Summary` + disclosed + `"`,
		`gh pr create --title "fix the walk" --body "Fixes iss-12.` + disclosed + `"`,
		`gh pr new -t fix -b "Ref: iss-12` + disclosed + `"`,
		`gh pr edit 3 --body "Touches iss-12 and iss-13.\n\nRefs: iss-12` + disclosed + `"`,
		`gh pr create --title fix --body "iss-9 and itd-146\n\nRefs: itd-146, iss-9` + disclosed + `"`,
	}
	for _, cmd := range cases {
		// A case writes a newline in the body as \n; the line gh is handed
		// carries a real one.
		cmd = strings.ReplaceAll(cmd, `\n`, "\n")
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if d.Verdict != VerdictBlock || !contains(d.Matches, forgeMentionEntryID) {
				t.Fatalf("Check = %q via %q (matches %v), want block via %q: %s", d.Verdict, d.EntryID, d.Matches, forgeMentionEntryID, d.Message)
			}
			if !strings.Contains(d.Message, "Refs: iss-") || !strings.Contains(d.Message, "Resolves: iss-") {
				t.Errorf("the refusal does not teach both declarations: %s", d.Message)
			}
		})
	}
}

func TestForgeMentionAllowsADeclaredOrUnseenMention(t *testing.T) {
	_, reg := forgeRepo(t, forgeMentionsOptedIn)
	cases := []string{
		`gh pr create --title "fix the walk (iss-12)" --body "Summary\n\nResolves: iss-12` + disclosed + `"`,
		`gh pr create --title fix --body "iss-1 and iss-2\n\nRefs: iss-1, iss-2` + disclosed + `"`,
		`gh pr create --title "Refs: iss-12" --body "See iss-12.` + disclosed + `"`,
		`gh pr create --title fix --body "xiss-12 is not a mention` + disclosed + `"`,
		// Only a pull request is held to it: a comment and an issue are not.
		`gh pr comment 3 --body "This touches iss-12."`,
		`gh issue create --title "iss-12 follow-up" --body "Split from iss-12."`,
		// A title or body the line does not fix may hold the declaration.
		`gh pr edit 3 --title "fix iss-12"`,
		`gh pr create --title "fix iss-12" --body "$BODY"`,
		`gh pr create --title "$TITLE" --body "Fixes iss-12.` + disclosed + `"`,
		`gh pr create --title "fix iss-12" --body-file -`,
		`gh pr create --title "fix iss-12" --fill`,
	}
	for _, cmd := range cases {
		// A case writes a newline in the body as \n; the line gh is handed
		// carries a real one.
		cmd = strings.ReplaceAll(cmd, `\n`, "\n")
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if contains(d.Matches, forgeMentionEntryID) {
				t.Fatalf("Check = %q via %q, want no mention verdict: %s", d.Verdict, d.EntryID, d.Message)
			}
		})
	}
}

func TestForgeMentionReadsTheBodyFile(t *testing.T) {
	root, reg := forgeRepo(t, forgeMentionsOptedIn)
	for name, body := range map[string]string{
		"bare.md":     "Fixes iss-12." + disclosed + "\n",
		"declared.md": "Fixes iss-12.\n\nResolves: iss-12" + disclosed + "\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if d := forgeCheck(t, reg, "cd "+root+" && gh pr create --title fix --body-file bare.md"); !contains(d.Matches, forgeMentionEntryID) {
		t.Fatalf("a body file naming an undeclared id: Check = %q via %q, want block via %q", d.Verdict, d.EntryID, forgeMentionEntryID)
	}
	if d := forgeCheck(t, reg, "cd "+root+" && gh pr create --title \"fix iss-12\" -F declared.md"); contains(d.Matches, forgeMentionEntryID) {
		t.Fatalf("a body file declaring the title's id: Check = %q via %q, want no mention verdict: %s", d.Verdict, d.EntryID, d.Message)
	}
}

// TestForgeMentionNeedsItsOwnOptIn: the rule is this repository's (RS004), not
// a convention every managed repository keeps, so outbound.forge alone does not
// turn it on.
func TestForgeMentionNeedsItsOwnOptIn(t *testing.T) {
	cmd := `gh pr create --title fix --body "Fixes iss-12.` + disclosed + `"`
	for name, config := range map[string]string{
		"forge only":                 forgeOptedIn,
		"mentions without forge":     `{"outbound": {"declare_mentions": true}}`,
		"mentions not a boolean":     `{"outbound": {"forge": true, "declare_mentions": "true"}}`,
		"mentions explicitly turned": `{"outbound": {"forge": true, "declare_mentions": false}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, reg := forgeRepo(t, config)
			if d := forgeCheck(t, reg, cmd); contains(d.Matches, forgeMentionEntryID) {
				t.Fatalf("Check = %q via %q, want no mention verdict under %s", d.Verdict, d.EntryID, config)
			}
		})
	}
}
