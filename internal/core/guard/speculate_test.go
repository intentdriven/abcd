package guard

import (
	"strings"
	"testing"
)

// verdictOf runs one candidate through the bundled registry and returns the
// decision, failing the test on an evaluation fault (which is never a clearance).
func verdictOf(t *testing.T, command string) Decision {
	t.Helper()
	d, err := Defaults().Check(command)
	if err != nil {
		t.Fatalf("guard could not evaluate %q: %v", command, err)
	}
	return d
}

// unenumeratedLaunchers are the case Tier 2 exists for: a program that runs
// another one and is not, and could not be, in the wrappers table. The names
// iss-272 listed (nice, setsid, flock, …) are deliberately NOT here — part B
// enumerated them, so they resolve as precise Tier 1 blocks and are pinned in the
// adversarial corpus instead. What is left is the open-ended remainder, which is
// the whole point: the fail-safe has to hold for launchers nobody listed.
var unenumeratedLaunchers = []string{
	"myrunner git push --force origin main",
	"parallel git push --force ::: a b",
	"watch -n1 git push --force origin main",
	"./deploy.sh git push --force origin main",
	// git launches things, which is why the gate is not "argv[0] is unknown".
	"git bisect run git push --force origin main",
	"git submodule foreach git push --force origin main",
}

// tenVerifiedBypasses is the list iss-272 opened with: ten command lines that
// exited 0 with no output on the merged tree. Every one of them now produces a
// verdict, and the three routes are worth keeping distinct in one place —
// speculation alone reached six, the wrapper names took those to precise blocks,
// and the exec-string table reached the last three, which no amount of
// speculation could (their payload is one opaque token).
var tenVerifiedBypasses = []string{
	"nice git push --force origin main",
	"nice -n 5 git push --force origin main",
	"setsid git push --force origin main",
	"flock /tmp/l git push --force origin main",
	"stdbuf -o0 git push --force origin main",
	"chroot / git push --force origin main",
	`busybox sh -c "gh repo delete owner/repo"`,
	`su -c "gh repo delete owner/repo"`,
	`runuser -c "git push --force origin main"`,
	`script -c "git push --force origin main" /dev/null`,
}

// TestUnrecognisedLauncherWarnsInsteadOfAllowing is the whole point of the
// fail-safe: a hazard launched through a program the wrapper table does not name
// stops being a confident allow. It never becomes a block — an unknown command is
// not proof that it execs its arguments.
func TestUnrecognisedLauncherWarnsInsteadOfAllowing(t *testing.T) {
	for _, cmd := range unenumeratedLaunchers {
		t.Run(cmd, func(t *testing.T) {
			d := verdictOf(t, cmd)
			if d.Verdict != VerdictWarn {
				t.Fatalf("verdict = %q, want %q — this is the silent allow iss-272 recorded", d.Verdict, VerdictWarn)
			}
			if d.EntryID != speculativeEntryID {
				t.Errorf("EntryID = %q, want the Tier 2 id %q", d.EntryID, speculativeEntryID)
			}
		})
	}
}

// TestSpeculationCarriesTheHazardsLesson keeps the warn worth reading: the
// successor and the why come from the entry the speculative suffix matched, so a
// Tier 2 warn teaches the same safe form a Tier 1 block would.
func TestSpeculationCarriesTheHazardsLesson(t *testing.T) {
	d := verdictOf(t, "myrunner git push --force origin main")
	if d.Successor == "" {
		t.Error("a speculative warn carries no successor: the refusal teaches nothing")
	}
	if !strings.Contains(d.Message, "git-push-force") {
		t.Errorf("message does not name the entry that matched: %q", d.Message)
	}
}

// TestTier2IsDistinguishableFromTier1 is adr-42 decision 2's stated requirement.
// The rationale for warning rather than blocking is that an unknown command is not
// proof it execs its arguments — a reader who cannot tell a speculative warn from
// a literal one has not been given that.
func TestTier2IsDistinguishableFromTier1(t *testing.T) {
	spec := verdictOf(t, "myrunner git clean -fd")
	lit := verdictOf(t, "git clean -fd")
	if lit.Verdict != VerdictWarn || spec.Verdict != VerdictWarn {
		t.Fatalf("both should warn; literal=%q speculative=%q", lit.Verdict, spec.Verdict)
	}
	// Against the SYNTHETIC id, not the registry one. Comparing a synthetic id to
	// `git-clean` can never be equal, so that version of this assertion held
	// whatever the code did — it passed with the Tier 2 id removed entirely.
	if spec.EntryID != speculativeEntryID {
		t.Fatalf("EntryID = %q, want %q: a speculative verdict must not be reported as an ordinary payload one",
			spec.EntryID, speculativeEntryID)
	}
	if spec.EntryID == syntheticEntryID {
		t.Fatal("a speculative warn is reported under the generic execute-a-string id, so a reader cannot tell it is a guess")
	}
	if spec.EntryID == lit.EntryID {
		t.Fatal("a speculative warn is reported identically to a literal one")
	}
	if spec.Family == "" {
		t.Error("a speculative warn carries no family: nothing marks it as speculative")
	}
	if !strings.Contains(strings.ToLower(spec.Message), "does not recognise") {
		t.Errorf("the message does not say the verdict is speculative: %q", spec.Message)
	}
}

// TestSpeculationIsGatedPerSegment closes the suppression bypass adr-42 decision
// 4 records: Check merges matches across every segment, so a gate that asked
// "did anything match the command line" would let one warn-tier command disarm
// the fail-safe for everything after it.
func TestSpeculationIsGatedPerSegment(t *testing.T) {
	d := verdictOf(t, "git clean -fd ; myrunner git push --force origin main")
	if d.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q", d.Verdict, VerdictWarn)
	}
	var sawLiteral, sawSpeculative bool
	for _, id := range d.Matches {
		switch id {
		case "git-clean":
			sawLiteral = true
		case speculativeEntryID:
			sawSpeculative = true
		}
	}
	if !sawLiteral {
		t.Errorf("the literal git-clean match is missing from %v", d.Matches)
	}
	if !sawSpeculative {
		t.Errorf("prefixing a warn-tier command disarmed the fail-safe: %v\n"+
			"the gate must be per segment, not per command line", d.Matches)
	}
}

// TestSpeculationSurvivesAnUnrelatedPayloadWarn keeps the fail-safe's fire in the
// report. The merge keeps only the FIRST synthetic signal per severity pool for
// the message, so a Tier 2 hit that arrives after an unrelated payload warn used
// to vanish from Matches entirely — and Matches is what the warn-rate gate counts,
// so the blind spot hid itself.
func TestSpeculationSurvivesAnUnrelatedPayloadWarn(t *testing.T) {
	d := verdictOf(t, `sh -c "$(echo hi)" ; myrunner git push --force origin main`)
	if d.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q", d.Verdict, VerdictWarn)
	}
	var sawPayloadWarn, sawSpeculative bool
	for _, id := range d.Matches {
		switch id {
		case syntheticEntryID:
			sawPayloadWarn = true
		case speculativeEntryID:
			sawSpeculative = true
		}
	}
	if !sawPayloadWarn {
		t.Errorf("the payload warn is missing from %v", d.Matches)
	}
	if !sawSpeculative {
		t.Errorf("the fail-safe fired and is absent from %v:\n"+
			"an unrelated warn took its slot, so the user is never told about the force-push", d.Matches)
	}
}

// TestSpeculationNeverBlocks pins the demotion. expandPayloads raises blocker
// signals of its own, and a speculative suffix reaches them; honouring one at its
// own tier would block on two layers of uncertainty at once — an unknown command
// that may not exec its arguments, carrying a payload the guard could not read.
func TestSpeculationNeverBlocks(t *testing.T) {
	// Verified reachable: the same payload directly under env -S blocks today.
	if d := verdictOf(t, `env -S 'git push --force $X'`); d.Verdict != VerdictBlock {
		t.Fatalf("precondition failed: %q = %q, want a synthetic block", `env -S`, d.Verdict)
	}
	d := verdictOf(t, `myrunner env -S 'git push --force $X'`)
	if d.Verdict == VerdictBlock {
		t.Fatalf("a speculative suffix produced a BLOCK: %q\nadr-42 decision 2: Tier 2 never blocks", d.Message)
	}
	if d.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q — the signal was dropped, not demoted", d.Verdict, VerdictWarn)
	}
}

// TestNoneOfTheTenBypassesIsSilent is the issue's own acceptance test. iss-272
// listed ten command lines that exited 0 with no output; not one of them may do
// that again, and this is the assertion that says so in the issue's own terms
// rather than in the implementation's.
func TestNoneOfTheTenBypassesIsSilent(t *testing.T) {
	for _, cmd := range tenVerifiedBypasses {
		t.Run(cmd, func(t *testing.T) {
			d := verdictOf(t, cmd)
			if d.Verdict == VerdictAllow {
				t.Fatalf("SILENT ALLOW, exactly as iss-272 recorded it: %s", cmd)
			}
			if d.Message == "" {
				t.Errorf("a non-allow verdict with no message tells the user nothing")
			}
		})
	}
}

// TestSpeculationLeavesOrdinaryWorkAlone is the false-positive floor in unit
// form. Quoting silences the fail-safe, and a command whose own entry matched is
// never speculated on.
func TestSpeculationLeavesOrdinaryWorkAlone(t *testing.T) {
	quiet := []string{
		`echo 'do not run git push --force'`,
		`rg 'git push --force' docs/`,
		`git commit -m 'ban git push --force'`,
		"make preflight",
		"go test ./...",
		"npm run build",
		"docker ps -a",
		"ls -la",
	}
	for _, cmd := range quiet {
		t.Run(cmd, func(t *testing.T) {
			if d := verdictOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("verdict = %q, want %q (entry %q): %s", d.Verdict, VerdictAllow, d.EntryID, d.Message)
			}
		})
	}
}

// TestSpeculationCapWarnsRatherThanSkipping holds the fail-loud half of the DoS
// bound. Past the start cap the guard has stopped looking, and stopping quietly
// would be the silent allow this whole change exists to remove.
func TestSpeculationCapWarnsRatherThanSkipping(t *testing.T) {
	long := "myrunner " + strings.Repeat("alpha ", maxSpeculativeStarts+8) + "beta"
	d := verdictOf(t, long)
	if d.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q: a command line too long to inspect was waved through", d.Verdict, VerdictWarn)
	}
	if !strings.Contains(strings.ToLower(d.Message), "too long") {
		t.Errorf("the cap warn does not say why it fired: %q", d.Message)
	}
}

// TestSpeculationTruncationWarnsRatherThanSkipping pins the OTHER fail-loud
// path, which the cap test does not reach. A single start whose suffix is longer
// than the window is truncated without the start cap ever being hit: the guard
// stopped looking, and stopping quietly is the silent allow this file removes.
func TestSpeculationTruncationWarnsRatherThanSkipping(t *testing.T) {
	// One eligible start (`alpha`), then a long tail of flags — flags are skipped
	// losslessly as starts, so the start cap is never reached, but the window
	// truncates the suffix.
	long := "myrunner alpha " + strings.Repeat("-x ", maxSpeculativeWindow+64)
	d := verdictOf(t, long)
	if d.Verdict != VerdictWarn {
		t.Fatalf("verdict = %q, want %q: a suffix the guard stopped reading was waved through", d.Verdict, VerdictWarn)
	}
	if !strings.Contains(strings.ToLower(d.Message), "too long") {
		t.Errorf("the truncation warn does not say why it fired: %q", d.Message)
	}
}

// TestSpeculationRespectsAfterCD keeps the fail-safe from inventing a hazard that
// the registry only names in context. A recursive delete of a named directory
// is a blocker only after a `cd`, because that is the shape that eats a working
// tree when the cd fails; without the guard clause a speculative match would
// fire on every `rm -rf build`.
func TestSpeculationRespectsAfterCD(t *testing.T) {
	if d := verdictOf(t, "myrunner rm -rf build"); d.Verdict != VerdictAllow {
		t.Errorf("verdict = %q, want %q: the after_cd condition was dropped, so an ordinary rm now warns",
			d.Verdict, VerdictAllow)
	}
	d := verdictOf(t, "cd scratch && myrunner rm -rf *")
	if d.Verdict == VerdictAllow {
		t.Fatalf("the cd-then-rm shape behind an unrecognised launcher is silent")
	}
	if !firedTier2(d) {
		t.Errorf("the fail-safe did not fire on %v; after_cd context is lost across the speculative boundary", d.Matches)
	}
}

// TestReservedEntryIDsAreRefused stops a registry from claiming an id the
// synthetic verdicts use. A colliding entry would be indexed out of r.Entries by
// a synthetic winner and render a blank message — and, worse, would let a repo
// dress a real entry up as the guard's own voice.
func TestReservedEntryIDsAreRefused(t *testing.T) {
	for _, id := range []string{syntheticEntryID, speculativeEntryID} {
		t.Run(id, func(t *testing.T) {
			r := Defaults()
			r.Entries[id] = Entry{
				ID: id, Tier: TierWarn, Why: "x", Successor: "y",
				Pattern: Pattern{Command: "definitelynotacommand"},
			}
			if err := Validate(r); err == nil {
				t.Fatalf("Validate accepted an entry claiming the reserved id %q", id)
			}
		})
	}
}
