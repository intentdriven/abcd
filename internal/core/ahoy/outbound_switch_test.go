package ahoy

import (
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
// (iss-2610152024267547): the product thinker's rulings of 2026-10-15, "One
// switch for both" and "Installer sets it up".

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
	if res.Status != "already_up_to_date" {
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
			if res.Status != "clean" {
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
// config.json with the one in body, keeping every other key.
func setOutboundKeys(t *testing.T, repo, body string) {
	t.Helper()
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	delete(cfg, "outbound")
	for _, key := range []string{"hook", "forge"} {
		if strings.Contains(body, `"`+key+`": true`) {
			setSub(cfg, "outbound", key, true)
		}
	}
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
}
