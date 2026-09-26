package guard

import "testing"

// TestSubstitutionKeepsTheEnclosingCommandWhole — iss-148. A command or process
// substitution runs its inner command AND leaves the enclosing command's argv
// intact on both sides of it: `rm $(true) -rf *` is `rm -rf *`, because an
// unquoted substitution that expands to nothing leaves no word behind. The
// tokenizer used to end the enclosing segment at the substitution, so the flags
// written after it became a separate "command" called `-rf`, and a blocker the
// registry names answered allow.
//
// Each case also pins the three regressions an earlier frame design shipped
// with: the enclosing command's chain surviving a newline inside the
// substitution (after_cd entries), a substitution in LEADING command position
// never becoming argv[0], and a nested bare `(` never closing the substitution
// early.
func TestSubstitutionKeepsTheEnclosingCommandWhole(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Verdict
	}{
		{"dollar-paren before trailing flags", "cd s && rm $(true) -rf *", VerdictBlock},
		{"backtick before trailing flags", "cd s && rm `true` -rf *", VerdictBlock},
		{"dollar-paren between subcommand and flag", "git push $(true) --force origin main", VerdictBlock},
		{"backtick between subcommand and flag", "git push `true` --force origin main", VerdictBlock},
		{"substitution between command and subcommand", "git $(true) push --force origin main", VerdictBlock},
		{"newline inside the substitution keeps the chain", "cd s && rm $(true\ntrue) -rf *", VerdictBlock},
		{"leading-position substitution never becomes argv[0]", "$(true) gh repo delete owner/repo", VerdictBlock},
		{"leading backtick never becomes argv[0]", "`true` gh repo delete owner/repo", VerdictBlock},
		{"nested bare paren does not close the substitution", "git push $( (true) ) --force origin main", VerdictBlock},
		{"nested substitution", "git push $(echo $(true)) --force origin main", VerdictBlock},
		{"arithmetic expansion before trailing flags", "git push $((1+2)) --force origin main", VerdictBlock},
		{"unterminated substitution still checks the enclosing command", "git push --force origin main $(true", VerdictBlock},
		{"unterminated backtick still checks the enclosing command", "git push --force origin main `true", VerdictBlock},
		// The inner command is still its own command-position segment.
		{"inner hazard still blocks", "echo $(git push --force origin main) done", VerdictBlock},
		{"inner hazard behind a cd still blocks", "cd s && echo $(rm -rf *)", VerdictBlock},
		// Benign lines stay allowed: the stitch must not invent a hazard.
		{"benign substitution", "git commit -m \"$(date)\" --allow-empty", VerdictAllow},
		{"benign flags after a substitution", "ls $(pwd) -la", VerdictAllow},
		{"substitution mid-word", "echo build-$(date +%s).log", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardVerdict(t, tc.command).Verdict; got != tc.want {
				t.Errorf("Check(%q).Verdict = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

// TestProcessSubstitutionIsAnOperand — iss-2608221126066631. `>(cmd)` and
// `<(cmd)` run cmd and hand the enclosing command ONE operand, a /dev/fd path,
// so the argv after them is still the enclosing command's. The tokenizer read
// the substitution as the end of the command, and a blocker-tier flag glued
// behind one escaped: `git push >(cat) --force origin main` answered allow
// while its plain-redirection spelling blocked.
func TestProcessSubstitutionIsAnOperand(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    Verdict
	}{
		{"output process substitution before a force flag", "git push >(cat) --force origin main", VerdictBlock},
		{"input process substitution before a force flag", "git push <(cat) --force origin main", VerdictBlock},
		{"command substitution inside a process substitution", "git push >$(echo x) --force origin main", VerdictBlock},
		{"hazard inside a process substitution", "diff <(git push --force origin main) b", VerdictBlock},
		{"cd-chained rm behind a process substitution", "cd s && rm <(true) -rf *", VerdictBlock},
		{"benign diff of two listings", "diff <(ls a) <(ls b)", VerdictAllow},
		{"benign tee into a process substitution", "make 2>&1 | tee >(grep error) build.log", VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardVerdict(t, tc.command).Verdict; got != tc.want {
				t.Errorf("Check(%q).Verdict = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

// TestSubstitutionTokenShape pins the tokenizer's reading directly: the inner
// command is emitted first (it runs first), the enclosing command keeps every
// token around the substitution and holds unknownMark where its output goes
// (unknown.go), a process substitution leaves one /dev/fd operand, and a
// newline inside a substitution never renumbers the enclosing command's chain.
func TestSubstitutionTokenShape(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{"rm $(true) -rf *", []string{"0:true", "0:rm|\x00|-rf|*"}},
		{"$(true) gh repo delete", []string{"0:true", "0:\x00|gh|repo|delete"}},
		{"echo a$(x)b c", []string{"0:x", "0:echo|a\x00b|c"}},
		{"echo --$(x) -r`y`", []string{"0:x", "0:y", "0:echo|--\x00|-r\x00"}},
		{"git push >(cat) --force", []string{"0:cat", "0:git|push|/dev/fd/63|--force"}},
		{"cd s && rm $(a\nb) -rf *", []string{"0:cd|s", "0:a", "1:b", "0:rm|\x00|-rf|*"}},
		{"echo $(a)\nls", []string{"0:a", "0:echo|\x00", "1:ls"}},
	}
	for _, tc := range cases {
		segs, err := tokenize(tc.line)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", tc.line, err)
		}
		got := render(segs)
		if len(got) != len(tc.want) {
			t.Errorf("tokenize(%q) = %q, want %q", tc.line, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("tokenize(%q) = %q, want %q", tc.line, got, tc.want)
				break
			}
		}
	}
}
