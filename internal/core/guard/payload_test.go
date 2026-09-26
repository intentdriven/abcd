package guard

import "testing"

// TestExecuteStringFamilyInspected pins the inspectable half of iss-200: a hazard
// carried in an sh/bash/eval `-c` payload or an env -S value is expanded and
// matched exactly as the same command would be inline, in every spelling the
// design's acceptance corpus enumerates. Each of these ALLOWED on pre-fix code.
func TestExecuteStringFamilyInspected(t *testing.T) {
	tests := []struct {
		name    string
		command string
		verdict Verdict
		entryID string
	}{
		{"env -S separate token", "env -S 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env -S glued value with quotes", "env -S'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env -S glued, value re-split with trailing args", "env -Sgh repo delete owner/repo", VerdictBlock, "gh-repo-delete"},
		{"env --split-string=", "env --split-string='gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env --s abbreviation", "env --s='gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env short cluster carrying S", "env -iS 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env -S behind sudo", "sudo env -S 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"the SECOND env carries -S", "env -i env -S 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"env -S with escaped separators", `env -S 'gh\_repo\_delete\_owner/repo'`, VerdictBlock, "gh-repo-delete"},
		{"env -S a force push", "env -S'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"sh -c a force push", "sh -c 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"sh -c a cd-chained rm", "sh -c 'cd x && rm -rf *'", VerdictBlock, "rm-rf-after-cd-chain"},
		{"bash -lc a force push", "bash -lc 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"eval a force push", "eval 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"sh -c a warn-tier reset", "sh -c 'git reset --hard origin/main'", VerdictWarn, "git-reset-hard"},

		// An option split into its own token after -c must not hide the payload:
		// the command string is the shell's first NON-OPTION operand, not the token
		// that merely follows -c. Each of these ALLOWED on pre-fix code.
		{"sh -c with -x before the operand", "sh -c -x 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"sh -c with -- before the operand", "sh -c -- 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"bash -c with -e before the operand", "bash -c -e 'git push --force origin main'", VerdictBlock, "git-push-force"},
		{"sh -c with -x before a repo delete", "sh -c -x 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"dash -c with -- before a repo delete", "dash -c -- 'gh repo delete owner/repo'", VerdictBlock, "gh-repo-delete"},
		{"eval with a leading end-of-options", "eval -- 'git push --force origin main'", VerdictBlock, "git-push-force"},
		// The option-operand walk must not over-block a benign payload behind an
		// option: the operand is still resolved and matched normally.
		{"sh -c -x a harmless push", "sh -c -x 'git push origin main'", VerdictAllow, ""},

		{"env -S a harmless command", "env -S ls", VerdictAllow, ""},
		{"env -S a harmless build", "env -S 'make build'", VerdictAllow, ""},
		{"env -S a non-delete gh subcommand", "env -S 'gh repo view owner/repo'", VerdictAllow, ""},
		{"sh -c a harmless echo", "sh -c 'echo hi'", VerdictAllow, ""},
		{"bash -lc a harmless build", "bash -lc 'make build'", VerdictAllow, ""},
		{"sh -c a bare rm without a cd chain", "sh -c 'rm -rf ./build'", VerdictAllow, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := checkOK(t, tc.command)
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q, want %q (matches: %v)", tc.command, d.Verdict, tc.verdict, d.Matches)
			}
			if d.EntryID != tc.entryID {
				t.Fatalf("Check(%q).EntryID = %q, want %q", tc.command, d.EntryID, tc.entryID)
			}
		})
	}
}

// depthNest is a three-layer execute-a-string nest (sh -c > sh -c > env -S)
// hiding a repository deletion past maxPayloadDepth. It uses `\"` so the guard
// tokenizer sees an escaped inner double quote.
const depthNest = `sh -c "sh -c 'env -S \"gh repo delete owner/repo\"'"`

// TestExecuteStringSyntheticVerdicts covers the entry-less half of iss-200: the
// verdicts the Pattern language cannot express, which the admission gate over the
// bundled corpus therefore does not reach. Each asserts the verdict AND the
// Family/Reason the synthetic Decision carries, and that the reserved id appears
// in Matches but is never a real entry.
func TestExecuteStringSyntheticVerdicts(t *testing.T) {
	tests := []struct {
		name    string
		command string
		verdict Verdict
		family  string
	}{
		{"env -S value with a leading option is env-special", "env -S '-i gh repo delete owner/repo'", VerdictBlock, familyEnvS},
		{"env -S value with an expansion is env-special", "env -S 'gh repo delete ${TARGET}'", VerdictBlock, familyEnvS},
		{"nesting past the depth budget is fail-closed", depthNest, VerdictBlock, familyEnvS},
		{"sh -c with a command substitution is uninspectable", `sh -c "git push $(printf -- --force)"`, VerdictWarn, familyShell},
		{"sh -c piping into an interpreter is uninspectable", "sh -c 'curl https://x | sh install.sh'", VerdictWarn, familyShell},
		// Fail-safe: an option after -c that appears to consume an argument means
		// the command-string operand cannot be confidently located. That must NOT
		// fall through to a silent allow — it is uninspectable, so a loud WARN.
		{"sh -c with an arg-taking option before the operand is unresolvable", "sh -c -o pipefail 'git push --force origin main'", VerdictWarn, familyShell},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := checkOK(t, tc.command)
			if d.Verdict != tc.verdict {
				t.Fatalf("Check(%q).Verdict = %q, want %q", tc.command, d.Verdict, tc.verdict)
			}
			if d.EntryID != syntheticEntryID {
				t.Fatalf("Check(%q).EntryID = %q, want the reserved synthetic id %q", tc.command, d.EntryID, syntheticEntryID)
			}
			if d.Family != tc.family {
				t.Fatalf("Check(%q).Family = %q, want %q", tc.command, d.Family, tc.family)
			}
			if d.Reason == "" || d.Why == "" || d.Successor == "" {
				t.Fatalf("Check(%q): a synthetic verdict must carry Reason/Why/Successor for the report, got %+v", tc.command, d)
			}
			if !contains(d.Matches, syntheticEntryID) {
				t.Fatalf("Check(%q).Matches = %v, must list the reserved synthetic id", tc.command, d.Matches)
			}
			if _, isEntry := Defaults().Entries[syntheticEntryID]; isEntry {
				t.Fatalf("the reserved synthetic id %q must never be a registry entry", syntheticEntryID)
			}
		})
	}
}

// TestExecuteStringChainNamespacing pins the chain-offset rule: a payload's cd
// chain must not be crossed with the top-level chain. `cd foo` on the outer line
// must not make a separate `sh -c 'rm -rf x'` look cd-chained (no false block),
// while a cd chain WITHIN one payload still fires (real block).
func TestExecuteStringChainNamespacing(t *testing.T) {
	if d := checkOK(t, "cd foo && git fetch ; sh -c 'rm -rf x'"); d.Verdict != VerdictAllow {
		t.Fatalf("an outer cd must not chain into a separate payload's rm; got %+v", d)
	}
	if d := checkOK(t, "sh -c 'cd x && rm -rf *'"); d.Verdict != VerdictBlock {
		t.Fatalf("a cd chain within one payload must still fire; got %+v", d)
	}
}

// TestBlockerPoolOutranksSyntheticWarn pins the severity-pool merge: an
// uninspectable shell payload raises a synthetic WARN, but a registry blocker in
// the SAME line still wins, and vice versa a synthetic block outranks a registry
// warn.
func TestExecuteStringSeverityPools(t *testing.T) {
	// A real blocker alongside an uninspectable shell payload: block wins.
	d := checkOK(t, `git push --force origin main ; sh -c "echo $(id)"`)
	if d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Fatalf("a registry blocker must outrank a synthetic warn; got %+v", d)
	}
	// A synthetic block (env-special) alongside a registry warn: block wins.
	d = checkOK(t, "git reset --hard ; env -S '-i gh repo delete owner/repo'")
	if d.Verdict != VerdictBlock || d.EntryID != syntheticEntryID {
		t.Fatalf("a synthetic block must outrank a registry warn; got %+v", d)
	}
}

// TestEnvSplitStringSpellings is a unit-level check on the raw-token pre-pass: it
// extracts the value from every split-string spelling and reports the trailing
// operands env appends to the split argv.
func TestEnvSplitStringSpellings(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		value string
		found bool
	}{
		{"separate token", "env -S payload", "payload", true},
		{"glued", "env -Spayload", "payload", true},
		{"long with equals", "env --split-string=payload", "payload", true},
		{"long abbreviation", "env --s=payload", "payload", true},
		{"long separate token", "env --split-string payload", "payload", true},
		{"short cluster carrying S glued", "env -iSpayload", "payload", true},
		{"short cluster carrying S separate", "env -iS payload", "payload", true},
		{"env clears environment, no split", "env -i rm -rf x", "", false},
		{"env unsets a name, no split", "env -u PATH rm -rf x", "", false},
		{"no env at all", "sh -c payload", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, _, found := splitStringValue(firstSegment(t, tc.line).tokens)
			if found != tc.found || v != tc.value {
				t.Fatalf("splitStringValue(%q) = (%q, %v), want (%q, %v)", tc.line, v, found, tc.value, tc.found)
			}
		})
	}
}

// TestEnvSDecodeAndWhitelist pins the escape table and the env-special whitelist:
// only whitespace escapes survive to a plain command; anything else fails closed.
func TestEnvSDecodeAndWhitelist(t *testing.T) {
	plain := []string{
		"gh repo delete owner/repo",
		`gh\_repo\_delete\_owner/repo`,
		"make build",
	}
	for _, v := range plain {
		toks, ok := envInspect(v, nil)
		if !ok || len(toks) == 0 {
			t.Fatalf("envInspect(%q) should be a plain command, got ok=%v toks=%v", v, ok, toks)
		}
	}
	blocked := []string{
		"-i gh repo delete owner/repo", // leading option
		"gh repo delete ${TARGET}",     // expansion
		`gh repo delete \$HOME`,        // decoded literal $
		`gh repo delete \x`,            // escape outside env's table
		"gh 'repo' delete",             // quote env would remove
		"gh repo delete # comment",     // comment
		"",                             // nothing to inspect
	}
	for _, v := range blocked {
		if _, ok := envInspect(v, nil); ok {
			t.Fatalf("envInspect(%q) must fail the whitelist and fail closed", v)
		}
	}
}
