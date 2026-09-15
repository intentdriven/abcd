package guard

import "testing"

// TestGitInProcessAliasRewritesTheSubcommand — GHSA-m2r8-fx7r-rq34. git rewrites
// its own subcommand from configuration handed to it IN THE COMMAND LINE: `-c
// alias.p=<body>`, `--config-env alias.p=VAR`, the `GIT_CONFIG_COUNT`/`KEY_n`/
// `VALUE_n` triple, and `GIT_CONFIG_PARAMETERS`. The matcher stepped over every
// one of those values without reading it — `-c` and `--config-env` are in every
// git entry's value_flags (gh-299) and an assignment prefix is stepped by
// commandOf — so the attacker-chosen alias NAME reached operand 0 and the
// subcommand compare missed. Each spelling below runs the hazard on a real git
// (verified on 2.52) and was a silent allow.
func TestGitInProcessAliasRewritesTheSubcommand(t *testing.T) {
	force := "--force"
	// want is the entry's own tier — the rewrite hands the segment to the
	// existing entries and changes nothing about what they say, so a warn-tier
	// entry reached through an alias is still a warn.
	blocked := []struct {
		name  string
		line  string
		entry string
		want  Verdict
	}{
		{"-c separate token", "git -c alias.p='push " + force + "' p origin main", "git-push-force", VerdictBlock},
		{"--config-env separate token", "FORCE_ALIAS='push " + force + "' git --config-env alias.p=FORCE_ALIAS p origin main", "git-push-force", VerdictBlock},
		{"--config-env attached", "FORCE_ALIAS='push " + force + "' git --config-env=alias.p=FORCE_ALIAS p origin main", "git-push-force", VerdictBlock},
		{"GIT_CONFIG_COUNT triple", "GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=alias.p GIT_CONFIG_VALUE_0='push " + force + "' git p origin main", "git-push-force", VerdictBlock},
		{"GIT_CONFIG_PARAMETERS", `GIT_CONFIG_PARAMETERS="'alias.p=push ` + force + `'" git p origin main`, "git-push-force", VerdictBlock},
		{"GIT_CONFIG_PARAMETERS split form", `GIT_CONFIG_PARAMETERS="'alias.p'='push ` + force + `'" git p origin main`, "git-push-force", VerdictBlock},
		// The alias body carries the whole hazard including its own subcommand,
		// and the arguments after the alias name are appended by git.
		{"body carries flags only", "git -c alias.p=push p " + force + " origin main", "git-push-force", VerdictBlock},
		// A second hop: an alias whose body names another alias.
		{"nested alias", "git -c alias.a=b -c alias.b='push " + force + "' a origin main", "git-push-force", VerdictBlock},
		// A different entry, to show the rewrite is not push-shaped.
		{"reset --hard through an alias", "git -c alias.nuke='reset --hard' nuke", "git-reset-hard", VerdictWarn},
		// Case: git config keys are case-insensitive in section and name.
		{"case-varied key", "git -c ALIAS.P='push " + force + "' p origin main", "git-push-force", VerdictBlock},
		// Inside a shell payload the expansion still has to happen.
		{"inside sh -c", "sh -c \"git -c alias.p='push " + force + "' p origin main\"", "git-push-force", VerdictBlock},
		// The hiding alias, and the one ACCEPTED over-block: git ignores an alias
		// that names a builtin, so this line is a plain `git push origin main`
		// and blocking it is a false positive. It is taken deliberately — the
		// alternative is a 146-name builtin table probe-synced against the
		// installed git, and over-blocking is the fail-safe direction — and
		// pinned here so the cost is visible rather than discovered.
		{"alias naming a builtin (accepted over-block)", "git -c alias.push='push " + force + "' push origin main", "git-push-force", VerdictBlock},
	}
	for _, tc := range blocked {
		t.Run(tc.name, func(t *testing.T) {
			d := checkGuard(t, tc.line)
			if d.Verdict != tc.want || d.EntryID != tc.entry {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q: git runs the alias body, not the name", tc.line, d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}

	allowed := []string{
		// The legitimate idiom: the expansion is benign, so it stays allowed.
		"git -c alias.st=status st",
		// Declared but not invoked.
		"git -c alias.p='push " + force + "' status",
		// A config value that merely LOOKS like a subcommand (the gh-299 shape).
		"git -c user.name=push status",
		// An alias body that is not a hazard, invoked.
		"git -c alias.lg='log --oneline' lg",
	}
	for _, line := range allowed {
		t.Run(line, func(t *testing.T) {
			if d := checkGuard(t, line); d.Verdict != VerdictAllow {
				t.Fatalf("Check(%q) = %q via %q, want %q: the expansion is benign", line, d.Verdict, d.EntryID, VerdictAllow)
			}
		})
	}
}

// TestGitBangAliasIsInspectedAsAShellPayload — a `!`-prefixed alias body is not
// a git subcommand at all: git hands it to the shell. It is therefore read the
// way every other execute-a-string payload is, so a hazard inside it is a
// precise block rather than a token nothing opens.
func TestGitBangAliasIsInspectedAsAShellPayload(t *testing.T) {
	force := "--force"
	d := checkGuard(t, "git -c alias.p='!git push "+force+" origin main' p")
	if d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Fatalf("Check(bang alias carrying a force push) = %q via %q, want %q via %q", d.Verdict, d.EntryID, VerdictBlock, "git-push-force")
	}
	// A bang body the guard cannot read is the shell family's loud warn, never a
	// silent allow.
	d = checkGuard(t, `git -c alias.p='!$(printf cm)' p`)
	if d.Verdict != VerdictWarn || d.EntryID != syntheticEntryID {
		t.Fatalf("Check(bang alias carrying a substitution) = %q via %q, want %q via %q", d.Verdict, d.EntryID, VerdictWarn, syntheticEntryID)
	}
}

// TestGitFileDeliveredConfigIsALoudWarn — the residual adr-42 calls permanently
// invisible: `GIT_CONFIG_GLOBAL`, `GIT_CONFIG_SYSTEM`, `-c include.path=` and
// `--config-env` naming a variable the command line does not set all deliver the
// alias body from somewhere the guard cannot read. The DIRECTIVE is visible, so
// the verdict is a loud warn under the guard's own reserved id — not a block
// (nothing here is proof of a hazard) and not silence.
func TestGitFileDeliveredConfigIsALoudWarn(t *testing.T) {
	lines := []string{
		"GIT_CONFIG_GLOBAL=/tmp/other.gitconfig git p origin main",
		"GIT_CONFIG_SYSTEM=/tmp/other.gitconfig git p origin main",
		"git -c include.path=/tmp/other.gitconfig p origin main",
		"git --config-env alias.p=NOT_SET_HERE p origin main",
	}
	for _, line := range lines {
		t.Run(line, func(t *testing.T) {
			d := checkGuard(t, line)
			if d.Verdict != VerdictWarn || d.EntryID != gitConfigEntryID {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q", line, d.Verdict, d.EntryID, VerdictWarn, gitConfigEntryID)
			}
			if !contains(d.Matches, gitConfigEntryID) {
				t.Errorf("Matches = %v, want %q listed", d.Matches, gitConfigEntryID)
			}
		})
	}
	if _, isEntry := Defaults().Entries[gitConfigEntryID]; isEntry {
		t.Errorf("the reserved id %q must never be a registry entry", gitConfigEntryID)
	}
	if !contains(reservedEntryIDs, gitConfigEntryID) {
		t.Errorf("reservedEntryIDs = %v, want %q listed so no repo entry can claim the guard's own voice", reservedEntryIDs, gitConfigEntryID)
	}
}

// TestGitOrdinaryWorkIsUnaffectedByTheAliasPrePass keeps the pre-pass off every
// git command line that declares no alias — the shape the repo corpus is full
// of, and the one the warn-rate ceiling is measured on.
func TestGitOrdinaryWorkIsUnaffectedByTheAliasPrePass(t *testing.T) {
	for _, line := range []string{
		"git status --porcelain",
		"git -C /repo log --oneline -5",
		"git -c core.pager=cat log",
		"git commit -m 'alias.p=push is only a message'",
		"git config --get alias.p",
	} {
		t.Run(line, func(t *testing.T) {
			if d := checkGuard(t, line); d.Verdict != VerdictAllow {
				t.Fatalf("Check(%q) = %q via %q, want %q", line, d.Verdict, d.EntryID, VerdictAllow)
			}
		})
	}
}

// TestEachBangAliasBodyGetsItsOwnChainRange pins the chain boundary between two
// `!`-alias bodies on one command line. A bang body is a separate command string
// — git hands it to a fresh shell — so a `cd` in one body does not precede an
// `rm` in another, exactly as it does not across two `sh -c` payloads.
//
// Every body was offset into the SAME range once, after the collection loop, and
// each body's own tokenize numbers its chains from 0: body A's chain 0 and body
// B's chain 0 landed on the same number, so the `after_cd` entry read across the
// boundary. Over-block only — but a guard that blocks what it cannot justify
// teaches the session to route around it.
func TestEachBangAliasBodyGetsItsOwnChainRange(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		verdict Verdict
		entry   string
	}{
		{
			"a cd in one bang body does not reach an rm in another",
			`git -c alias.a='!cd /tmp' a && git -c alias.b='!rm -rf .' b`,
			VerdictAllow, "",
		},
		{
			"the sh -c twin, which already read it that way",
			`sh -c 'cd /tmp' && sh -c 'rm -rf .'`,
			VerdictAllow, "",
		},
		{
			"a cd and an rm in the SAME bang body still chain",
			`git -c alias.a='!cd /tmp && rm -rf .' a`,
			VerdictBlock, "rm-rf-after-cd-chain",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Defaults().Check(tc.line)
			if err != nil {
				t.Fatalf("Check(%q): %v", tc.line, err)
			}
			if d.Verdict != tc.verdict || d.EntryID != tc.entry {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q", tc.line, d.Verdict, d.EntryID, tc.verdict, tc.entry)
			}
		})
	}
}
