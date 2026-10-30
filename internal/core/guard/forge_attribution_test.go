package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The disclosure half of the forge-text check (iss-2609061525437969). In a
// repository that opted in, a pull request's body discloses assistance as its
// commits do: a `gh pr create`, `gh pr new` or `gh pr edit` whose body or body
// file carries no Assisted-by line in one of the three accepted forms, judged as
// the attribution gate's body arm judges it, is blocked under
// forge-text-disclosure before the forge keeps it.

func TestForgeDisclosureBlocksAPullRequestBodyWithoutTheLine(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	cases := []string{
		`gh pr create --title "fix the walk" --body "Summary"`,
		`gh pr create --title t --body ""`,
		`gh pr new -b "Summary"`,
		`gh pr edit 12 --body "Summary"`,
		`gh pr create --body "Closes #3. Assisted-by: Claude:claude-opus-5-5"`,
		"gh pr create --title t --body \"$(cat <<'EOF'\nSummary\n\n```\nAssisted-by: Claude:x\n```\nEOF\n)\"",
		"gh pr create --body \"Summary\n\nCo-authored-by: A <a@example.invalid>\nAssisted-by: Claude:x\"",
		"gh pr create --body \"Summary\n\nAssisted-by: abcd:latest\"",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if d.Verdict != VerdictBlock || d.EntryID != forgeDisclosureEntryID {
				t.Fatalf("Check = %q via %q (%s), want block via %q", d.Verdict, d.EntryID, d.Message, forgeDisclosureEntryID)
			}
			if !strings.Contains(d.Message, "Assisted-by") {
				t.Errorf("the refusal does not teach the accepted forms: %s", d.Message)
			}
		})
	}
}

func TestForgeDisclosureAllowsEachAcceptedForm(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	cases := []string{
		"gh pr create --title t --body \"Summary\n\nAssisted-by: Claude:claude-opus-5-5\"",
		"gh pr create --title t --body \"Summary\n\nAssisted-by: None\"",
		"gh pr create --title t --body \"Composed.\n\nAssisted-by: abcd:dev\"",
		"gh pr create --title t --body \"Summary\r\n\r\nAssisted-by: Claude:claude-opus-5[1m]\r\n\"",
		"gh pr edit 12 --body \"Summary\n\nAssisted-by: Claude:a\nAssisted-by: Vendor:b\"",
		"gh pr create --title t --body \"$(cat <<'EOF'\nSummary\n\n```\nCo-authored-by: an example\n```\n\nAssisted-by: Claude:claude-opus-5-5\nEOF\n)\"",
		// Only a pull request's body is held to it: a title, a comment and an
		// issue are not.
		`gh pr edit 12 --title "fix the walk"`,
		`gh pr comment 12 --body "Thanks, merged."`,
		`gh issue create --title t --body "Summary"`,
		`gh issue edit 7 --body "Summary"`,
		// A body the line does not fix is not seen.
		`gh pr create --body "$BODY"`,
		"gh pr create --body \"$(cat <<EOF\nSummary\n\n$TRAILER\nEOF\n)\"",
		`gh pr create --body-file -`,
		`gh pr create --fill`,
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if contains(d.Matches, forgeDisclosureEntryID) {
				t.Fatalf("Check = %q via %q, want no disclosure verdict: %s", d.Verdict, d.EntryID, d.Message)
			}
		})
	}
}

func TestForgeDisclosureReadsTheBodyFile(t *testing.T) {
	root, reg := forgeRepo(t, forgeOptedIn)
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("bare.md", "Summary\n")
	write("disclosed.md", "Summary\n\nAssisted-by: Claude:claude-opus-5-5\n")
	if d := forgeCheck(t, reg, "cd "+root+" && gh pr create --title t --body-file bare.md"); d.Verdict != VerdictBlock || d.EntryID != forgeDisclosureEntryID {
		t.Fatalf("a body file without the line: Check = %q via %q, want block via %q", d.Verdict, d.EntryID, forgeDisclosureEntryID)
	}
	if d := forgeCheck(t, reg, "cd "+root+" && gh pr create --title t -F disclosed.md"); contains(d.Matches, forgeDisclosureEntryID) {
		t.Fatalf("a body file with the line: Check = %q via %q, want no disclosure verdict: %s", d.Verdict, d.EntryID, d.Message)
	}
}

// TestForgeDisclosureDefersToALeak: a body that carries a session URL or a
// footer is refused for that, once; the disclosure refusal is not stacked on
// the same body.
func TestForgeDisclosureDefersToALeak(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	d := forgeCheck(t, reg, `gh pr create --body "`+forgeSessionURL()+`"`)
	if d.EntryID != forgeTextEntryID || contains(d.Matches, forgeDisclosureEntryID) {
		t.Fatalf("Check = %q via %q (matches %v), want the leak alone", d.Verdict, d.EntryID, d.Matches)
	}
}

func TestForgeDisclosureIsSilentWhereTheRepositoryDidNotOptIn(t *testing.T) {
	_, reg := forgeRepo(t, `{"outbound": {"hook": true}}`)
	if d := forgeCheck(t, reg, `gh pr create --title t --body "Summary"`); contains(d.Matches, forgeDisclosureEntryID) {
		t.Fatalf("Check = %q via %q, want nothing where outbound.forge is off", d.Verdict, d.EntryID)
	}
}
