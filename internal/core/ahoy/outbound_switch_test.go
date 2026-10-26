package ahoy

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The one outbound switch (iss-2610152024327989) and the abcd.binary it writes
// (iss-2610152024267547): the product thinker ruled on 2026-10-15 that one
// opt-in turns on both checks and that the installer writes abcd.binary itself.

// cloneAbcdBinary reads this clone's abcd.binary, "" when it is unset.
func cloneAbcdBinary(t *testing.T, repo *gittest.Repo) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo.Root(), "config", "--local", "--get", "abcd.binary")
	cmd.Env = repo.Env()
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return ""
		}
		t.Fatalf("git config --get abcd.binary: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// TestOutboundSwitchWritesAbcdBinaryToTheVerifiedCopy — iss-2610152024267547.
// The explicit opt-in tells the commit-message check where abcd is: this
// clone's abcd.binary names the absolute path of the owned PATH copy whose build
// provenance this run verified, and nothing else.
func TestOutboundSwitchWritesAbcdBinaryToTheVerifiedCopy(t *testing.T) {
	setupHermetic(t)
	target := binTarget()
	repo := gittest.NewRepo(t)
	res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertOwnedCopyAt(t, target)
	if got := cloneAbcdBinary(t, repo); got != target {
		t.Fatalf("abcd.binary = %q, want the verified copy %q (notes %v)", got, target, res.Notes)
	}
	var noted bool
	for _, k := range res.writeKinds {
		noted = noted || k == writeOutboundBinary
	}
	if !noted {
		t.Errorf("the abcd.binary write is not in the install's receipt: writes %v", res.Writes)
	}

	// A second opt-in has nothing left to do.
	res, err = Install(repo.Root(), outboundOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if settled(res) != "already_up_to_date" {
		t.Errorf("a second --outbound status = %q (writes %v, notes %v), want already_up_to_date", res.Status, res.Writes, res.Notes)
	}
}

// TestOutboundSwitchNeverPointsAbcdBinaryAtAnUnverifiedCopy — iss-2610152024267547.
// A copy whose build provenance did not verify, the dev shim, and a copy inside
// the working tree are each refused: nothing is written, and the install says so.
func TestOutboundSwitchNeverPointsAbcdBinaryAtAnUnverifiedCopy(t *testing.T) {
	assertRefused := func(t *testing.T, repo *gittest.Repo, res InstallResult, why string) {
		t.Helper()
		if got := cloneAbcdBinary(t, repo); got != "" {
			t.Errorf("abcd.binary = %q, want it left unset", got)
		}
		joined := notesJoined(res.Notes)
		if !strings.Contains(joined, "abcd.binary was not written") || !strings.Contains(joined, why) {
			t.Errorf("the install must say abcd.binary was not written and why (%q); notes = %v", why, res.Notes)
		}
	}

	t.Run("provenance did not verify", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		opts := outboundOpts()
		opts.VerifyProvenance = (&recordingProvenance{err: errors.New("no build-provenance attestation is published")}).check
		res, err := Install(repo.Root(), opts, RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		assertRefused(t, repo, res, "verified")
	})

	t.Run("the dev shim", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		opts := outboundOpts()
		opts.Dev = true
		res, err := Install(repo.Root(), opts, RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		assertRefused(t, repo, res, "verified")
	})

	t.Run("a copy inside the working tree", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		t.Setenv("ABCD_BIN_TARGET", filepath.Join(repo.Root(), "tools", "bin", "abcd"))
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		assertRefused(t, repo, res, "working tree")
	})

	t.Run("a value the person set is left as it is", func(t *testing.T) {
		setupHermetic(t)
		repo := gittest.NewRepo(t)
		own := filepath.Join(t.TempDir(), "my-abcd")
		repo.Git("config", "--local", "abcd.binary", own)
		res, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if got := cloneAbcdBinary(t, repo); got != own {
			t.Errorf("abcd.binary = %q, want the person's own value %q kept", got, own)
		}
		if joined := notesJoined(res.Notes); strings.Contains(joined, own) {
			t.Errorf("the note printed the person's own abcd.binary value: %v", res.Notes)
		}
	})
}

// TestOutboundSwitchTurnsOnBothChecks — iss-2610152024327989. One opt-in turns
// on the commit-message check and the forge-text check together.
func TestOutboundSwitchTurnsOnBothChecks(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t)
	if _, err := Install(repo.Root(), outboundOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	if !outboundOptedIn(repo.Root()) {
		t.Error("the switch did not record outbound.hook")
	}
	if !guard.ForgeTextOptedIn(repo.Root()) {
		t.Error("the switch did not record outbound.forge, so the forge-text check stays off")
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath))); err != nil {
		t.Errorf("the switch wrote no commit-msg hook: %v", err)
	}
}

// TestHalfOptedRepositoryIsCompleted — iss-2610152024327989. A repository that
// recorded one half under the two retired flags is not left half on: detection
// names it, and the next plain install completes it.
func TestHalfOptedRepositoryIsCompleted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		hook   bool
	}{
		{"only the commit-message check", `{"outbound": {"hook": true}}`, true},
		{"only the forge-text check", `{"outbound": {"forge": true}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t)
			if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			setOutboundKeys(t, repo.Root(), tc.config)
			if tc.hook {
				if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath)), outboundHookTemplate, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			det, err := Detect(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			if !gapIDSet(det.Gaps)[outboundHalfGapID] {
				t.Fatalf("a half-on repository raises no %s gap: %v", outboundHalfGapID, gapIDs(det.Gaps))
			}

			res, err := Install(repo.Root(), installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if settled(res) != "clean" {
				t.Errorf("completing install status = %q (remaining %v), want clean", res.Status, res.Remaining)
			}
			if !outboundOptedIn(repo.Root()) || !guard.ForgeTextOptedIn(repo.Root()) {
				t.Errorf("the half-on repository was not completed: hook=%v forge=%v", outboundOptedIn(repo.Root()), guard.ForgeTextOptedIn(repo.Root()))
			}
			if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath))); err != nil {
				t.Errorf("the completed repository has no commit-msg hook: %v", err)
			}
			if got := cloneAbcdBinary(t, repo); got != "" {
				t.Errorf("a plain install wrote abcd.binary = %q; only the explicit opt-in writes it", got)
			}
		})
	}
}

// setOutboundKeys replaces the outbound section of the repository's
// config.json with the one in body, a JSON object such as
// `{"outbound": {"hook": true, "forge": false}}`, keeping every other key.
func setOutboundKeys(t *testing.T, repo, body string) {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("setOutboundKeys body %q: %v", body, err)
	}
	delete(cfg, "outbound")
	if sec, ok := parsed["outbound"]; ok {
		cfg["outbound"] = sec
	}
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
}

// outboundSection reads the outbound section of the repository's config.json
// back, as written.
func outboundSection(t *testing.T, repo string) map[string]any {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	return subMap(cfg, "outbound")
}

// TestAnExplicitFalseHalfIsThePersonsChoice — a key present and false is the
// person's choice, never a half the next install completes. Only a key that is
// ABSENT, which the retired flags left behind, raises outbound.half_on; a
// `false` is left as it is under --yes, and detection says at most that the
// switch is half on by choice, in a note nothing resolves.
func TestAnExplicitFalseHalfIsThePersonsChoice(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		hook   bool
	}{
		{"commit-message check on, forge-text check set false", `{"outbound": {"hook": true, "forge": false}}`, true},
		{"forge-text check on, commit-message check set false", `{"outbound": {"hook": false, "forge": true}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t)
			if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			setOutboundKeys(t, repo.Root(), tc.config)
			hookPath := filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath))
			if tc.hook {
				if err := os.WriteFile(hookPath, outboundHookTemplate, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			before := outboundSection(t, repo.Root())

			det, err := Detect(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			for _, g := range det.Gaps {
				if g.ID == outboundHalfGapID {
					t.Errorf("an explicit false raised %s, which the next install would resolve: %+v", outboundHalfGapID, g)
				}
				if strings.HasPrefix(g.ID, "outbound.") && (g.Required || g.Resolvable) {
					t.Errorf("an explicit false raised a gap install would act on: %+v", g)
				}
			}

			res, err := Install(repo.Root(), installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			after := outboundSection(t, repo.Root())
			if len(after) != len(before) || after["hook"] != before["hook"] || after["forge"] != before["forge"] {
				t.Errorf("--yes changed the person's outbound choice: before %v, after %v (writes %v)", before, after, res.Writes)
			}
			if !tc.hook {
				if _, err := os.Lstat(hookPath); err == nil {
					t.Errorf("--yes wrote the commit-msg hook against outbound.hook: false (writes %v)", res.Writes)
				}
			}
		})
	}
}

// TestHalfOnCompletionNamesAForeignHook — a repository that recorded only the
// forge-text half, with a commit-msg hook of its own at the committed path,
// is told the truth: the completion records outbound.hook and says the hook was
// not written because it is not abcd's, and nothing reports a hook write that
// did not happen. abcd's own repository is this case.
func TestHalfOnCompletionNamesAForeignHook(t *testing.T) {
	for _, tc := range []struct {
		name, hook, says string
	}{
		{"a hook that does not run the check", "#!/bin/sh\n# the repository's own hook\nexit 0\n", "chain"},
		{"a hook that already runs the check", "#!/bin/sh\n\"$bin\" lint outbound --label \"commit message\" \"$1\"\n", "already runs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t)
			if _, err := Install(repo.Root(), installOpts(), RefusingPrompter{}); err != nil {
				t.Fatal(err)
			}
			setOutboundKeys(t, repo.Root(), `{"outbound": {"forge": true}}`)
			hookPath := filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath))
			if err := os.WriteFile(hookPath, []byte(tc.hook), 0o755); err != nil {
				t.Fatal(err)
			}

			det, err := Detect(repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			var half *Gap
			for i := range det.Gaps {
				if det.Gaps[i].ID == outboundHalfGapID {
					half = &det.Gaps[i]
				}
			}
			if half == nil {
				t.Fatalf("no %s gap: %v", outboundHalfGapID, gapIDs(det.Gaps))
			}
			if strings.Contains(half.FixHint, "writes the commit-msg hook") {
				t.Errorf("the gap promises a hook write that cannot happen: %q", half.FixHint)
			}
			if !strings.Contains(half.FixHint, "not abcd's") || !strings.Contains(half.FixHint, tc.says) {
				t.Errorf("the gap does not say the hook is not abcd's and what follows (%q): %q", tc.says, half.FixHint)
			}

			res, err := Install(repo.Root(), installOpts(), RefusingPrompter{})
			if err != nil {
				t.Fatal(err)
			}
			if got, _ := os.ReadFile(hookPath); string(got) != tc.hook {
				t.Error("the completion replaced a commit-msg hook abcd did not write")
			}
			if !outboundOptedIn(repo.Root()) {
				t.Error("the completion did not record outbound.hook")
			}
			for _, w := range res.Writes {
				if w == OutboundHookRelPath {
					t.Errorf("the receipt reports a hook write that did not happen: %v", res.Writes)
				}
			}
			for _, k := range res.writeKinds {
				if k == writeOutboundHook {
					t.Errorf("the summary would say a commit-message check was added: kinds %v", res.writeKinds)
				}
			}
			joined := notesJoined(res.Notes)
			if !strings.Contains(joined, OutboundHookRelPath) || !strings.Contains(joined, "not abcd's") || !strings.Contains(joined, tc.says) {
				t.Errorf("the notes do not say the hook was left because it is not abcd's (%q): %v", tc.says, res.Notes)
			}
		})
	}
}

// TestAForeignHookThatRunsTheCheckIsNotReported — a commit-msg hook abcd did
// not write but that runs `abcd lint outbound` itself (abcd's own repository
// carries one) is not reported as leaving the check off: the advisory says
// the check does not run on this path, which would be false. A comment naming
// the verb is not running it.
func TestAForeignHookThatRunsTheCheckIsNotReported(t *testing.T) {
	for _, tc := range []struct {
		name, hook string
		reported   bool
	}{
		{"runs it", "#!/bin/sh\n\"$work/abcd\" lint outbound --label \"commit message\" \"$1\" || exit 1\n", false},
		{"only names it in a comment", "#!/bin/sh\n# we should run abcd lint outbound here\nexit 0\n", true},
		{"does not mention it", "#!/bin/sh\nexit 0\n", true},
		{"only echoes it", "#!/bin/sh\necho lint outbound\nexit 0\n", true},
		{"names it in an echoed sentence", "#!/bin/sh\necho \"and runs \\`abcd lint outbound\\`, and\" >&2\nexit 0\n", true},
		{"runs it from PATH", "#!/bin/sh\nset -e\nabcd lint outbound --label \"commit message\" \"$1\"\n", false},
		{"runs it as an if condition", "#!/bin/sh\nif ! \"$bin\" lint outbound \"$1\"; then exit 1; fi\n", false},
		{"chains abcd's copy of the check", "#!/bin/sh\n" + outboundShimChainLine + "\nexit 0\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t)
			setOutboundKeys(t, repo.Root(), `{"outbound": {"hook": true, "forge": true}}`)
			hookPath := filepath.Join(repo.Root(), filepath.FromSlash(OutboundHookRelPath))
			if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(hookPath, []byte(tc.hook), 0o755); err != nil {
				t.Fatal(err)
			}
			var got bool
			for _, g := range detectOutboundHook(repo.Root(), HooksPathStateUnarmed) {
				got = got || g.ID == "outbound.hook_foreign"
			}
			if got != tc.reported {
				t.Errorf("outbound.hook_foreign reported = %v, want %v", got, tc.reported)
			}
		})
	}
}
