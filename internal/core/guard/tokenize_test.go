package guard

import (
	"errors"
	"strings"
	"testing"
)

// render flattens segments to a comparable form: chain index, then the tokens
// joined with "|" so an empty or embedded-space token is still visible.
func render(segs []segment) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		out[i] = string(rune('0'+s.chain)) + ":" + strings.Join(s.tokens, "|")
	}
	return out
}

func TestTokenizeSegments(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "single command",
			line: "rm -rf *",
			want: []string{"0:rm|-rf|*"},
		},
		{
			name: "cd chain across &&",
			line: "cd scratch && rm -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			name: "quoted hazard is one argument token",
			line: `abcd capture "agent ran cd scratch && rm -rf * — one failed cd from disaster"`,
			want: []string{"0:abcd|capture|agent ran cd scratch && rm -rf * — one failed cd from disaster"},
		},
		{
			name: "single quotes suppress operators",
			line: `echo 'a && b'`,
			want: []string{"0:echo|a && b"},
		},
		{
			name: "backslash escapes a space",
			line: `echo a\ b`,
			want: []string{`0:echo|a b`},
		},
		{
			name: "backslash escapes a quote inside double quotes",
			line: `echo "a\"b"`,
			want: []string{`0:echo|a"b`},
		},
		{
			name: "backslash escapes a separator",
			line: `echo a\&\&b`,
			want: []string{`0:echo|a&&b`},
		},
		{
			name: "all compound separators split command position",
			line: "a; b | c || d && e",
			want: []string{"0:a", "0:b", "0:c", "0:d", "0:e"},
		},
		{
			name: "background operator splits",
			line: "sleep 1 & rm -rf *",
			want: []string{"0:sleep|1", "0:rm|-rf|*"},
		},
		{
			name: "subshell parentheses split",
			line: "(cd x && rm -rf *)",
			want: []string{"0:cd|x", "0:rm|-rf|*"},
		},
		{
			// Backtick command substitution runs its inner command exactly as
			// `$( … )` does, so the tokenizer must split it into command position
			// the same way — otherwise the hazard is swallowed into a token and
			// never matched (gh-312).
			name: "backtick command substitution splits into command position",
			line: "echo `gh repo delete owner/repo`",
			want: []string{"0:echo", "0:gh|repo|delete|owner/repo"},
		},
		{
			name: "a bare backtick substitution is a command-position segment",
			line: "`git push --force origin main`",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "an assignment carrying a backtick substitution splits it out",
			line: "x=`git push --force origin main`",
			want: []string{"0:x=", "0:git|push|--force|origin|main"},
		},
		{
			name: "a backtick inside single quotes stays literal",
			line: "echo '`gh repo delete owner/repo`'",
			want: []string{"0:echo|`gh repo delete owner/repo`"},
		},
		{
			name: "newline starts a new chain",
			line: "cd x\nrm -rf *",
			want: []string{"0:cd|x", "1:rm|-rf|*"},
		},
		{
			name: "comment at word start is stripped",
			line: "echo hi # rm -rf *",
			want: []string{"0:echo|hi"},
		},
		{
			name: "hash inside a word is not a comment",
			line: "curl https://example.test/#frag",
			want: []string{"0:curl|https://example.test/#frag"},
		},
		{
			name: "quoted hash is not a comment",
			line: `git commit -m "fix #123"`,
			want: []string{"0:git|commit|-m|fix #123"},
		},
		{
			name: "a newline after a list operator continues the same chain",
			line: "cd scratch &&\nrm -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			name: "a newline after a pipe continues the same chain",
			line: "cd scratch |\nrm -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			// The `> doc.md` redirection is dropped (a target is never in
			// command position), and the heredoc body stays data — the
			// `git push --force` inside it must never reach command position.
			name: "heredoc body is data, not commands",
			line: "cat > doc.md <<'EOF'\ngit push --force\nEOF",
			want: []string{"0:cat"},
		},
		{
			name: "heredoc body ends at its delimiter line",
			line: "cat <<EOF\nrm -rf *\nEOF\nls -la",
			want: []string{"0:cat", "1:ls|-la"},
		},
		{
			name: "dash heredoc delimiter may be indented",
			line: "cat <<-EOF\n\trm -rf *\n\tEOF\nls",
			want: []string{"0:cat", "1:ls"},
		},
		{
			name: "the rest of the heredoc's own line is still command text",
			line: "cat <<EOF && ls\nrm -rf *\nEOF",
			want: []string{"0:cat", "0:ls"},
		},
		{
			// The body starts on the next line even when the redirection line
			// ends in a list operator: bash collects here-document bodies at the
			// end of the PHYSICAL line, so `grep x` and `rm -rf *` here are
			// document text and the pipeline's second command is whatever
			// follows `EOF`. Probed on bash 3.2: this input alone is a syntax
			// error (the pipeline has no second member) and adding a line after
			// `EOF` makes THAT the member, with neither body line run. The
			// earlier reading — body deferred until the list completed — put
			// document text in command position, where an apostrophe in a
			// document became ErrUnparsableCommand and the hook failed open.
			name: "a heredoc body starts despite a trailing list operator",
			line: "cat <<EOF |\ngrep x\nrm -rf *\nEOF",
			want: []string{"0:cat"},
		},
		{
			name: "a blank line does not break a list continuation",
			line: "cd scratch &&\n\nrm -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			name: "a comment line does not break a list continuation",
			line: "cd scratch &&\n# note\nrm -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			name: "a quoted heredoc delimiter may be exotic",
			line: "cat <<'---'\nrm -rf *\n---\nls",
			want: []string{"0:cat", "1:ls"},
		},
		{
			name: "an arithmetic shift is not a heredoc",
			line: "echo $((1<<20))\ncd scratch",
			want: []string{"0:echo|$", "0:1<<20", "1:cd|scratch"},
		},
		{
			name: "a herestring is an argument, not a heredoc",
			line: `grep foo <<< "rm -rf *"`,
			want: []string{"0:grep|foo|<<<|rm -rf *"},
		},
		{
			name: "empty line yields no segments",
			line: "   \t  ",
			want: nil,
		},
		{
			name: "adjacent quoting concatenates into one token",
			line: `git push '--force' origin`,
			want: []string{"0:git|push|--force|origin"},
		},
		{
			name: "empty quoted token is preserved",
			line: `grep "" file`,
			want: []string{"0:grep||file"},
		},
		{
			name: "a glued redirection terminates the word and drops the target",
			line: "git push --force>/dev/null",
			want: []string{"0:git|push|--force"},
		},
		{
			name: "a glued redirection with a spaced target keeps later words",
			line: "git push --force>out.txt origin main",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "a leading redirection does not displace the command",
			line: ">/dev/null git push --force origin main",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "an append redirection drops only its target",
			line: "echo a >> log b",
			want: []string{"0:echo|a|b"},
		},
		{
			name: "an fd-prefixed dup redirection is dropped whole",
			line: "go test ./... 2>&1",
			want: []string{"0:go|test|./..."},
		},
		{
			name: "process substitution keeps its prior handling",
			line: "cat <(echo hi)",
			want: []string{"0:cat|<", "0:echo|hi"},
		},
		{
			name: "an ampersand redirection does not split the command",
			line: "git push &>/dev/null --force origin main",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "a glued ampersand redirection terminates the word only",
			line: "git push&>/dev/null --force origin main",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "an ampersand-append redirection drops only its target",
			line: "git commit &>>log --no-verify -m x",
			want: []string{"0:git|commit|--no-verify|-m|x"},
		},
		{
			name: "an ampersand redirection mid cd chain keeps both segments",
			line: "cd scratch && rm &>/dev/null -rf *",
			want: []string{"0:cd|scratch", "0:rm|-rf|*"},
		},
		{
			name: "a digit before an ampersand redirection stays a real argument",
			line: "timeout 30&>/dev/null rm -rf *",
			want: []string{"0:timeout|30|rm|-rf|*"},
		},
		{
			name: "a logical and is not an ampersand redirection",
			line: "git push && git push --force origin main",
			want: []string{"0:git|push", "0:git|push|--force|origin|main"},
		},
		{
			// ANSI-C quoting must not carry the `$` into the word: bash hands
			// git the same `--force` argv either way, so the token must be
			// `--force`, never `$--force`.
			name: "ansi-c quoting yields the bare flag",
			line: "git push $'--force' origin main",
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "ansi-c quoting decodes hex escapes",
			line: `git push $'\x2d\x2dforce' origin main`,
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "ansi-c quoting decodes short unicode escapes",
			line: `git push $'\u002d\u002dforce' origin main`,
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "ansi-c quoting decodes long unicode escapes",
			line: `git push $'\U0000002d\U0000002dforce' origin main`,
			want: []string{"0:git|push|--force|origin|main"},
		},
		{
			name: "ansi-c quoting glued to a word joins it",
			line: "gh repo$'' delete owner/repo",
			want: []string{"0:gh|repo|delete|owner/repo"},
		},
		{
			name: "locale quoting yields the bare flag",
			line: `git push $"--force" origin main`,
			want: []string{"0:git|push|--force|origin|main"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			segs, err := tokenize(tc.line)
			if err != nil {
				t.Fatalf("tokenize(%q): unexpected error: %v", tc.line, err)
			}
			got := render(segs)
			if len(got) != len(tc.want) {
				t.Fatalf("tokenize(%q) = %q, want %q", tc.line, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("tokenize(%q) segment %d = %q, want %q", tc.line, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestTokenizeRejectsUnterminatedQuote pins the error class that STAYS an error:
// an unterminated quote is refused by every shell (bash, zsh, dash all exit
// without running anything), so the hook's loud fail-open on it is harmless. A
// trailing backslash is deliberately NOT in this list any more — bash runs that
// line (GHSA-5wx3-2c86-fjpx), so it gets a verdict instead; see
// TestBashRecoverableTokenizerStatesAreVerdicts.
func TestTokenizeRejectsUnterminatedQuote(t *testing.T) {
	for _, line := range []string{`echo "unterminated`, `echo 'unterminated`, `echo $'unterminated`} {
		if _, err := tokenize(line); !errors.Is(err, ErrUnparsableCommand) {
			t.Fatalf("tokenize(%q) error = %v, want ErrUnparsableCommand", line, err)
		}
	}
}

// TestTokenizeFlagsUnterminatedHeredoc: a here-document whose delimiter line
// never appears is NOT an error any more — an error reaches the pre-tool-use
// hook as fail-OPEN, and bash runs the line (it recovers silently, taking the
// rest of the input as the body). It is a segment flag instead, which Check
// folds into a fail-closed block, the brace-group precedent: what the tokenizer
// cannot read it refuses, on both front doors.
func TestTokenizeFlagsUnterminatedHeredoc(t *testing.T) {
	const line = "cat <<EOF\nrm -rf *"
	segs, err := tokenize(line)
	if err != nil {
		t.Fatalf("tokenize(%q): unexpected error: %v — an unterminated body must be a flag, not an error", line, err)
	}
	if len(segs) == 0 || !segs[len(segs)-1].heredocUnterminated {
		t.Fatalf("tokenize(%q) = %q: the command that opened the heredoc must carry heredocUnterminated", line, render(segs))
	}
	d, err := Defaults().Check(line)
	if err != nil {
		t.Fatalf("Check(%q): unexpected error: %v", line, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != heredocEntryID {
		t.Fatalf("Check(%q) = %q via %q, want %q via %q", line, d.Verdict, d.EntryID, VerdictBlock, heredocEntryID)
	}
}

// TestBashRecoverableTokenizerStatesAreVerdicts — GHSA-5wx3-2c86-fjpx. Two
// tokenizer error states are inputs every shell here EXECUTES: a backslash before
// EOF (bash 3.2 and zsh drop it and run the line) and a here-document body with
// no delimiter line (bash recovers silently). The hook maps a tokenizer error to
// fail-open, so both were a deterministic bypass of every blocker: one appended
// byte, or one `<<EOF` and a newline. Each now gets a VERDICT from core — a
// trailing backslash is parsed as bash 3.2 does (dropped), an unterminated body
// is a fail-closed block — and the error route is kept for what no shell runs.
//
// The third row is the prerequisite: spaced arithmetic `$(( x << y ))` was
// classified as a heredoc because the paren test looked only at the byte right
// after the delimiter word, so the hazard on the next line was swallowed as
// body. Without that fix the fail-closed arm would turn this fail-open into a
// false block.
func TestBashRecoverableTokenizerStatesAreVerdicts(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		verdict Verdict
		entry   string
	}{
		{"trailing backslash on a hazard", "git push --force origin main \\", VerdictBlock, "git-push-force"},
		{"trailing backslash on a benign line", "echo hello \\", VerdictAllow, ""},
		{"empty unterminated heredoc body on a hazard", "git push --force origin main <<EOF\n", VerdictBlock, "git-push-force"},
		{"unterminated heredoc body swallowing a hazard", "cat <<EOF\ngit push --force origin main\n", VerdictBlock, heredocEntryID},
		{"spaced arithmetic then a hazard", "echo $(( x << y ))\ngit push --force origin main", VerdictBlock, "git-push-force"},
		{"spaced arithmetic alone", "echo $(( x << y ))", VerdictAllow, ""},
		{"spaced arithmetic in a group", "(( x << y ))\ngit push --force origin main", VerdictBlock, "git-push-force"},
		// The sibling: shellInspect's inner tokenize mapped the same errors to a
		// warn, so the payload forms were loud-warn allows.
		{"trailing backslash inside a shell payload", "sh -c 'git push --force origin main \\'", VerdictBlock, "git-push-force"},
		{"unterminated heredoc inside a shell payload", "bash -c 'git push --force origin main <<EOF\n'", VerdictBlock, "git-push-force"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v — bash runs this line, so the guard must answer, not error", tc.line, err)
			}
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q via %q, want %q", tc.line, d.Verdict, d.EntryID, tc.verdict)
			}
			if tc.entry != "" && d.EntryID != tc.entry {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.line, d.EntryID, tc.entry)
			}
		})
	}
	// The unterminated heredoc's own synthetic id is reported alongside the
	// entry that wins the block pool, so the lesson (terminate the document) is
	// not lost behind the hazard it happened to carry.
	d := guardVerdict(t, "git push --force origin main <<EOF\n")
	if !containsString(d.Matches, heredocEntryID) {
		t.Errorf("Matches = %v, want %q listed beside the entry", d.Matches, heredocEntryID)
	}
}

// TestArithmeticShiftByIdentifierIsNotAHeredoc pins the identifier-operand
// form of the same boundary the literal-digit case above already covers: an
// unquoted `<<` whose "delimiter" word is immediately followed by a bare paren
// — e.g. `$((1<<shift))`, where readHeredocDelim reads "shift" and stops at
// the arithmetic expression's own closing paren — is never a real heredoc. A
// genuine delimiter word is always followed by its body and terminator line,
// never by `(` or `)` with no separator. Classifying this correctly at parse
// time (rather than merely erroring when no terminator line happens to exist)
// matters: an unquoted `<<` misread this way must still let every later line
// reach command position and be matched normally, not merely fail loud.
func TestArithmeticShiftByIdentifierIsNotAHeredoc(t *testing.T) {
	const line = "shift=8\necho $((1<<shift))\ngit push --force origin main"
	segs, err := tokenize(line)
	if err != nil {
		t.Fatalf("tokenize(%q): unexpected error: %v", line, err)
	}
	got := render(segs)
	swallowed := true
	for _, s := range got {
		if strings.Contains(s, "push") {
			swallowed = false
		}
	}
	if swallowed {
		t.Errorf("tokenize(%q) = %q: the `git push --force` line never reached command position", line, got)
	}

	d, err := Defaults().Check(line)
	if err != nil {
		t.Fatalf("Check(%q): unexpected error: %v", line, err)
	}
	if d.Verdict != VerdictBlock {
		t.Fatalf("Check(%q).Verdict = %q (entry %q), want %q via git-push-force", line, d.Verdict, d.EntryID, VerdictBlock)
	}

	// Control: the literal-operand form must still parse and block normally.
	const lit = "echo $((1<<20))\ngit push --force origin main"
	dl, err := Defaults().Check(lit)
	if err != nil {
		t.Fatalf("Check(%q): unexpected error: %v", lit, err)
	}
	if dl.Verdict != VerdictBlock {
		t.Fatalf("control: Check(%q).Verdict = %q, want %q", lit, dl.Verdict, VerdictBlock)
	}
}

// TestArithmeticShiftCoincidentalDelimiterStillBlocks is the adversarial case
// that TestArithmeticShiftByIdentifierIsNotAHeredoc alone would miss: an
// attacker who knows the tokenizer once looked for a line matching the
// misread "delimiter" could supply exactly that line (here, a bare `shift`
// with no arguments — a harmless no-op POSIX builtin), so the earlier,
// narrower fix (erroring only when no such line exists) would still let this
// swallow the guarded command with no error and no signal. Classifying the
// `<<` correctly up front — never treating it as a heredoc in the first place
// — closes this regardless of what later lines happen to contain.
func TestArithmeticShiftCoincidentalDelimiterStillBlocks(t *testing.T) {
	for _, line := range []string{
		"echo $((1<<shift))\ngit push --force origin main\nshift",
		"echo $((1<<k))\ngit push --force origin main\nk",
		"echo $(( $((1<<a)) ))\ngit push --force origin main\na",
	} {
		d, err := Defaults().Check(line)
		if err != nil {
			t.Fatalf("Check(%q): unexpected error: %v", line, err)
		}
		if d.Verdict != VerdictBlock {
			t.Fatalf("Check(%q).Verdict = %q, want %q — the guarded line must not be silently swallowed", line, d.Verdict, VerdictBlock)
		}
	}
}
