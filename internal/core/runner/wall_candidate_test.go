package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// candidateWorld is a drive holding its settings and a candidate standing beside
// it, on the stand-in harness.
func candidateWorld(t *testing.T, mode string) (Profile, string, []byte) {
	t.Helper()
	newFake(t, mode, Claude)
	p := wallProfile(t)
	if _, err := WriteWallSettings(p, SelfBinary()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
	if err != nil {
		t.Fatal(err)
	}
	return p, filepath.Join(t.TempDir(), ".abcd.new"), before
}

// TestACandidateIsProvedAsTheApproverAndTheHooksThenTheSettingsAreRestored: the
// launch reads settings naming the candidate and an approver naming it, by the one
// proof a drive makes of itself, and after it the settings are what the check
// compares against.
func TestACandidateIsProvedAsTheApproverAndTheHooksThenTheSettingsAreRestored(t *testing.T) {
	p, candidate, before := candidateWorld(t, "wall-ok")
	if err := CheckWallSettings(p, SelfBinary()); err != nil {
		t.Fatal(err)
	}
	var during []byte
	var approver string
	prev := proveCandidateWall
	proveCandidateWall = func(ctx context.Context, p Profile, bin string) error {
		during, _ = os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
		approver = bin
		return prev(ctx, p, bin)
	}
	t.Cleanup(func() { proveCandidateWall = prev })
	if err := ProveCandidate(context.Background(), p, candidate); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(during, wallSettings(candidate)) || approver != candidate {
		t.Fatalf("the launch's approver is %q, and its settings name the candidate:\n%s", approver, during)
	}
	after, err := os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("the settings are restored byte for byte:\n%s", after)
	}
	if err := CheckWallSettings(p, SelfBinary()); err != nil {
		t.Fatalf("the running drive's check still holds: %v", err)
	}
}

// TestAWallTheCandidateDoesNotHoldLeavesTheSettingsAsTheyWere is the core of the
// "proof fails" row.
func TestAWallTheCandidateDoesNotHoldLeavesTheSettingsAsTheyWere(t *testing.T) {
	p, candidate, before := candidateWorld(t, "wall-tools")
	err := ProveCandidate(context.Background(), p, candidate)
	var we *WallError
	if !errors.As(err, &we) || we.Wall != WallTools {
		t.Fatalf("got %v, want the tool list named", err)
	}
	after, rerr := os.ReadFile(filepath.Join(p.Dir, wallSettingsName))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("the settings are restored byte for byte:\n%s", after)
	}
	if strings.Contains(string(after), candidate) {
		t.Fatalf("no trace of the candidate:\n%s", after)
	}
	if err := CheckWallSettings(p, SelfBinary()); err != nil {
		t.Fatalf("the running drive's check still holds: %v", err)
	}
}

// TestRenderedByAbcdKnowsThePreviousRendering is the "rendering compatibility"
// row: each fixture is a settings file a release wrote, kept as bytes, so a new
// binary re-wiring it says nothing about an edit. A release that changes the
// rendering adds its predecessor's bytes here.
func TestRenderedByAbcdKnowsThePreviousRendering(t *testing.T) {
	names, err := filepath.Glob(filepath.Join("testdata", "wall-settings-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no rendering is kept as a fixture")
	}
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if !renderedByAbcd(raw) {
			t.Errorf("%s is a release's rendering, and it reads as edited", name)
		}
	}
	if renderedByAbcd([]byte(`{"hooks":{},"model":"mine"}`)) {
		t.Error("a person's file reads as abcd's")
	}
}

// TestAStageWritingTheBinaryOrTheSettingsEndsTheRunAndTheMarkDoesNot is the core of
// the "stage cannot decide" row: what a stage can write beside the proof's bytes
// is compared before every stage, and what is written after a swap is never read.
func TestAStageWritingTheBinaryOrTheSettingsEndsTheRunAndTheMarkDoesNot(t *testing.T) {
	newFake(t, "wall-ok", Claude)
	p := wallProfile(t)
	installed := filepath.Join(t.TempDir(), "abcd")
	if err := os.WriteFile(installed, []byte("the release"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWallSettings(p, installed); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, installed); err != nil {
		t.Fatal(err)
	}
	mark := abcdhome.Path(p.PersonHome, "release-mark")
	if err := os.MkdirAll(filepath.Dir(mark), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mark, []byte("release_tag=v9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, installed); err != nil {
		t.Fatalf("the release mark decides nothing: %v", err)
	}
	if err := os.WriteFile(installed, []byte("a stage's"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, installed); err == nil {
		t.Fatal("a binary a stage wrote ends the run")
	}
	if err := os.WriteFile(installed, []byte("the release"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Dir, wallSettingsName), wallSettings(filepath.Join(t.TempDir(), "elsewhere")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckWallSettings(p, installed); err == nil {
		t.Fatal("settings a stage wrote end the run")
	}
}
