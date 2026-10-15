package guard

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The forge-text check (iss-2610142210511831). A repository that opted in with
// `ahoy install --outbound` (outbound.forge in .abcd/config.json) has the
// text a posting gh command hands the forge judged against the outbound policy
// before it runs: a pull request's or an issue's title and body, a comment's
// body, and a body file. A live agent-session URL or a tool attribution footer
// blocks under forge-text-outbound; a body file the guard cannot read warns
// under forge-text-unread; a repository that did not opt in sees nothing.

// forgeSessionID and forgeSessionURL are assembled at run time, so no
// committed file carries a live-shaped session URL (the payload scan and
// `abcd lint` refuse a literal one).
const forgeSessionID = "qNs22jeg43nekhIpcwrcSr"

func forgeSessionURL() string {
	return "https://agent-host.dev/code/" + "session_" + forgeSessionID
}

// forgeFooter is a tool attribution footer line, assembled at run time for the
// same reason.
func forgeFooter() string {
	return "🤖 Generated " + "with [Some Tool](https://sometool.dev)"
}

// forgeRepo lays a repository root whose config.json holds config, and returns
// it with a registry loaded for it that reads files from that root.
func forgeRepo(t *testing.T, config string) (string, Registry) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return root, reg.ReadingFrom(NewFiles(root))
}

const forgeOptedIn = `{"outbound": {"forge": true}}`

func forgeCheck(t *testing.T, reg Registry, cmd string) Decision {
	t.Helper()
	d, err := reg.Check(cmd)
	if err != nil {
		t.Fatalf("Check(%q): %v", cmd, err)
	}
	return d
}

func TestForgeTextBlocksAPostingCommandCarryingALeak(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	url, footer := forgeSessionURL(), forgeFooter()
	cases := []string{
		`gh pr create --title "fix the walk" --body "Summary` + "\n\n" + footer + `"`,
		`gh pr create --title "see ` + url + `" --body "clean"`,
		`gh pr create -t "fix" -b "Summary` + "\n\n" + footer + `"`,
		`gh pr create --body="Summary` + "\n\n" + footer + `"`,
		`gh pr create -b"Summary` + "\n\n" + footer + `"`,
		`gh pr create -b="Session: ` + url + `"`,
		`gh pr new --body "` + url + `"`,
		`gh pr edit 12 --body "` + url + `"`,
		`gh pr comment 12 --body "` + url + `"`,
		`gh issue create --title t --body "` + url + `"`,
		`gh issue new --body "` + url + `"`,
		`gh issue edit 7 --title "` + url + `"`,
		`gh issue comment 7 -b "` + url + `"`,
		"gh pr create --title t --body \"$(cat <<'EOF'\nSummary\n\n" + footer + "\nEOF\n)\"",
		"cd /tmp && gh pr create --body \"" + url + "\"",
		"sh -c 'gh pr comment 3 --body \"" + url + "\"'",
		"command gh pr comment 3 --body \"" + url + "\"",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if d.Verdict != VerdictBlock || d.EntryID != forgeTextEntryID {
				t.Fatalf("Check = %q via %q (%s), want block via %q", d.Verdict, d.EntryID, d.Message, forgeTextEntryID)
			}
			if strings.Contains(d.Message, forgeSessionID) || strings.Contains(d.Message, "Some Tool") ||
				strings.Contains(d.Message, "sometool.dev") {
				t.Errorf("the refusal republished the text it refused: %s", d.Message)
			}
			if !strings.Contains(d.Message, "Assisted-by") {
				t.Errorf("the refusal does not teach the safe form: %s", d.Message)
			}
		})
	}
}

func TestForgeTextAllowsCleanAndNonPostingCommands(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	url := forgeSessionURL()
	cases := []string{
		`gh pr create --title "fix the walk" --body "Closes #3. Assisted-by: Claude:claude-opus-5-5"`,
		"gh pr create --title t --body \"$(cat <<'EOF'\nSummary\n\nAssisted-by: Claude:claude-opus-5-5\nEOF\n)\"",
		`gh pr view 12 --json body`,
		`gh issue list --search "` + url + `"`,
		`gh pr checks 12`,
		`gh api repos/o/r/pulls/12`,
		// Prose quoting the footer mid-sentence is documentation, not a footer.
		`gh pr comment 3 --body "the guard refuses a footer like 🤖 Generated with [x](y) on its own line"`,
		// A gh posting command named inside an argument is not in command position.
		`echo "gh pr create --body ` + url + `"`,
		// A body the line does not fix is not seen (commands/guard.md names it).
		`gh pr create --body "$BODY"`,
		`gh pr create --body-file -`,
		`gh pr create --fill`,
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			d := forgeCheck(t, reg, cmd)
			if contains(d.Matches, forgeTextEntryID) || contains(d.Matches, forgeUnreadEntryID) {
				t.Fatalf("Check = %q via %q, want no forge-text verdict: %s", d.Verdict, d.EntryID, d.Message)
			}
		})
	}
}

func TestForgeTextIsSilentWhereTheRepositoryDidNotOptIn(t *testing.T) {
	cmd := `gh pr create --body "` + forgeSessionURL() + `"`
	for name, config := range map[string]string{
		"no config":           "",
		"no outbound key":     `{"repo": {"visibility": "public"}}`,
		"forge false":         `{"outbound": {"forge": false}}`,
		"forge not a boolean": `{"outbound": {"forge": "true"}}`,
		"only the hook":       `{"outbound": {"hook": true}}`,
		"unparsable":          `{"outbound": `,
	} {
		t.Run(name, func(t *testing.T) {
			_, reg := forgeRepo(t, config)
			if d := forgeCheck(t, reg, cmd); d.Verdict != VerdictAllow {
				t.Fatalf("Check = %q via %q, want allow: %s", d.Verdict, d.EntryID, d.Message)
			}
		})
	}
	if d := forgeCheck(t, Defaults(), cmd); d.Verdict != VerdictAllow {
		t.Fatalf("the bundled registry with no repository = %q via %q, want allow", d.Verdict, d.EntryID)
	}
}

func TestForgeTextReadsTheBodyFile(t *testing.T) {
	root, reg := forgeRepo(t, forgeOptedIn)
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("leak.md", "Summary\n\n"+forgeFooter()+"\n")
	write("clean.md", "Summary\n\nAssisted-by: Claude:claude-opus-5-5\n")
	if err := os.Symlink(filepath.Join(root, "leak.md"), filepath.Join(root, "link.md")); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{
		"gh pr create --title t --body-file leak.md",
		"gh pr create -F leak.md",
		"gh pr create --body-file=leak.md",
		"gh issue comment 4 -F ./leak.md",
		"gh pr create -F " + filepath.Join(root, "leak.md"),
		"gh pr create -F link.md",
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := forgeCheck(t, reg, cmd); d.Verdict != VerdictBlock || d.EntryID != forgeTextEntryID {
				t.Fatalf("Check = %q via %q, want block via %q", d.Verdict, d.EntryID, forgeTextEntryID)
			}
		})
	}
	if d := forgeCheck(t, reg, "gh pr create --title t --body-file clean.md"); d.Verdict != VerdictAllow {
		t.Fatalf("a clean body file = %q via %q, want allow: %s", d.Verdict, d.EntryID, d.Message)
	}
}

// TestForgeTextWarnsOnABodyFileItCannotRead: a body file the guard cannot read
// is never a silent pass and never a block. The command runs, with a warning
// saying the body was not checked; a named pipe is refused at the open
// (O_NONBLOCK, regular files only), so the check never waits on a writer.
func TestForgeTextWarnsOnABodyFileItCannotRead(t *testing.T) {
	root, reg := forgeRepo(t, forgeOptedIn)
	fifo := filepath.Join(root, "pipe.md")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	big := filepath.Join(root, "big.md")
	if err := os.WriteFile(big, []byte(strings.Repeat("a line of body text\n", (maxScriptBytes/20)+10)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []string{
		"gh pr create -F missing.md",
		"gh pr create -F pipe.md",
		"gh pr create -F big.md",
		"gh pr create -F dir.md",
		// Written earlier on the same line: the file the guard reads now is
		// not the one gh will read.
		"printf 'x' > fresh.md && gh pr create -F fresh.md",
	}
	if os.Geteuid() != 0 {
		locked := filepath.Join(root, "locked.md")
		if err := os.WriteFile(locked, []byte("x"), 0o000); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, "gh pr create -F locked.md")
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			done := make(chan Decision, 1)
			go func() { done <- forgeCheck(t, reg, cmd) }()
			select {
			case d := <-done:
				if d.Verdict != VerdictWarn || !contains(d.Matches, forgeUnreadEntryID) {
					t.Fatalf("Check = %q via %q (matches %v), want warn via %q", d.Verdict, d.EntryID, d.Matches, forgeUnreadEntryID)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("the check hung reading the body file")
			}
		})
	}
}

// TestForgeTextWarnsWhenTheScannerConfigIsBroken: a pii.json that does not
// load would judge with a weakened detector set, so the check says it could
// not judge (a warn), rather than either passing in silence or blocking.
func TestForgeTextWarnsWhenTheScannerConfigIsBroken(t *testing.T) {
	root, reg := forgeRepo(t, forgeOptedIn)
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "pii.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := forgeCheck(t, reg, `gh pr create --body "`+forgeSessionURL()+`"`)
	if d.Verdict != VerdictWarn || !contains(d.Matches, forgeUnreadEntryID) {
		t.Fatalf("Check = %q via %q (matches %v), want warn via %q", d.Verdict, d.EntryID, d.Matches, forgeUnreadEntryID)
	}
	if strings.Contains(d.Message, forgeSessionID) {
		t.Errorf("the warning republished the text: %s", d.Message)
	}
}

func TestForgeTextIdsAreReserved(t *testing.T) {
	for _, id := range []string{forgeTextEntryID, forgeUnreadEntryID} {
		if !contains(reservedEntryIDs, id) {
			t.Errorf("reservedEntryIDs = %v, want %q listed so no repo entry can claim the guard's own voice", reservedEntryIDs, id)
		}
	}
}

func TestForgeTextOptedIn(t *testing.T) {
	root, _ := forgeRepo(t, forgeOptedIn)
	if !ForgeTextOptedIn(root) {
		t.Error("ForgeTextOptedIn = false for outbound.forge: true")
	}
	other, _ := forgeRepo(t, `{"outbound": {"hook": true}}`)
	if ForgeTextOptedIn(other) {
		t.Error("ForgeTextOptedIn = true for a repository that opted into the commit-msg hook only")
	}
}

// TestForgeTextJudgesABoundedNumberOfTexts: each text judged builds a scanner
// from the repository's configuration, so a line of thousands of posting
// commands must not cost thousands of them. Past maxForgeTexts the check warns
// that it stopped, and never passes the rest in silence.
func TestForgeTextJudgesABoundedNumberOfTexts(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	line := strings.Repeat(`gh pr comment 1 --body "fine"; `, maxForgeTexts) + `gh pr comment 1 --body "` + forgeSessionURL() + `"`
	d := forgeCheck(t, reg, line)
	if d.Verdict != VerdictWarn || !contains(d.Matches, forgeUnreadEntryID) {
		t.Fatalf("Check = %q via %q (matches %v), want warn via %q past the bound", d.Verdict, d.EntryID, d.Matches, forgeUnreadEntryID)
	}
}

// TestForgeTextWorkIsLinear: the forge-text reading adds a constant per word
// to a posting command's check, for a long body and for many commands alike.
func TestForgeTextWorkIsLinear(t *testing.T) {
	_, reg := forgeRepo(t, forgeOptedIn)
	t.Run("a long body", func(t *testing.T) {
		build := func(n int) string {
			return `gh pr create --title t --body "` + strings.Repeat("a line of the body\n", n) + `"`
		}
		assertWorkGrowthOn(t, reg, build, 1<<9, "a body is read once, as one word")
	})
	t.Run("many posting commands", func(t *testing.T) {
		build := func(n int) string {
			return strings.Repeat(`gh pr comment 1 -b "fine"; `, n)
		}
		assertWorkGrowthOn(t, reg, build, 1<<7, "each posting command is read once, and the texts judged are bounded")
	})
}

// forgeBlockedScript is a script the guard blocks (git-push-force), spelled in
// pieces so no tool reading this file's source mistakes it for a command.
func forgeBlockedScript() string { return "git push " + "--force origin main\n" }

// TestForgeTextBodyFilesNeverSpendTheScriptBudget: body files are read on a
// budget of their own. Drawn from the scripts' one (16 files, 1 MiB), sixteen
// posting commands' body files before a script the guard would block turned
// that block into a script-unread warn.
func TestForgeTextBodyFilesNeverSpendTheScriptBudget(t *testing.T) {
	cases := []struct {
		name  string
		n     int
		bytes int
	}{
		{"sixteen body files", maxScriptFiles, len("Summary\n")},
		{"four 256 KiB body files", 4, maxScriptBytes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, reg := forgeRepo(t, forgeOptedIn)
			var line strings.Builder
			for i := 0; i < tc.n; i++ {
				name := "body" + string(rune('a'+i)) + ".md"
				body := strings.Repeat("a", tc.bytes-1) + "\n"
				if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
				line.WriteString("gh pr comment 1 -F " + name + "; ")
			}
			if err := os.WriteFile(filepath.Join(root, "s.sh"), []byte(forgeBlockedScript()), 0o644); err != nil {
				t.Fatal(err)
			}
			if d := forgeCheck(t, reg, "bash s.sh"); d.Verdict != VerdictBlock || d.EntryID != scriptHazardEntryID {
				t.Fatalf("the script alone = %q via %q, want the block via %q", d.Verdict, d.EntryID, scriptHazardEntryID)
			}
			line.WriteString("bash s.sh")
			d := forgeCheck(t, reg, line.String())
			if d.Verdict != VerdictBlock || d.EntryID != scriptHazardEntryID {
				t.Fatalf("after %d body files the script = %q via %q (matches %v), want the block via %q",
					tc.n, d.Verdict, d.EntryID, d.Matches, scriptHazardEntryID)
			}
			if contains(d.Matches, forgeUnreadEntryID) {
				t.Errorf("the body files themselves went unread: matches %v", d.Matches)
			}
		})
	}
}

// TestForgeTextBodyFilesHaveABoundOfTheirOwn: past maxForgeReadBytes of body
// files the next one warns unread rather than being read without bound.
func TestForgeTextBodyFilesHaveABoundOfTheirOwn(t *testing.T) {
	root, reg := forgeRepo(t, forgeOptedIn)
	var line strings.Builder
	// Four files of the most the guard reads of one fill the 1 MiB budget; the
	// fifth is past it.
	n := 5
	for i := 0; i < n; i++ {
		name := "big" + string(rune('a'+i)) + ".md"
		if err := os.WriteFile(filepath.Join(root, name), []byte(strings.Repeat("a", maxScriptBytes-1)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		line.WriteString("gh pr comment 1 -F " + name + "; ")
	}
	d := forgeCheck(t, reg, line.String())
	if d.Verdict != VerdictWarn || !contains(d.Matches, forgeUnreadEntryID) {
		t.Fatalf("%d body files of %d KiB = %q via %q (matches %v), want warn via %q", n, maxScriptBytes>>10, d.Verdict, d.EntryID, d.Matches, forgeUnreadEntryID)
	}
}
