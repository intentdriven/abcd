package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallDevShimNotesEveryFailure is iss-227: each of the three writes the
// dev-shim install makes (removing the pinned entry it replaces, creating the
// directory, writing the shim) used to fail with a bare return, so a run that
// installed nothing reported no reason. Each failure must leave a note naming
// what was not done, and no write on the receipt.
func TestInstallDevShimNotesEveryFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so the forced failures cannot occur")
	}
	_, pluginRoot := setupHermetic(t)

	readOnlyDir := func(t *testing.T) string {
		t.Helper()
		d := t.TempDir()
		t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
		return d
	}

	cases := []struct {
		name  string
		setup func(t *testing.T) (target string, kind binTargetKind)
		want  string
	}{
		{
			name: "remove",
			setup: func(t *testing.T) (string, binTargetKind) {
				d := readOnlyDir(t)
				target := filepath.Join(d, "abcd")
				if err := os.Symlink(pluginBinaryPath(pluginRoot), target); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(d, 0o555); err != nil {
					t.Fatal(err)
				}
				return target, binTargetOwnedSymlink
			},
			want: "could not replace the existing PATH entry",
		},
		{
			name: "mkdir",
			setup: func(t *testing.T) (string, binTargetKind) {
				f := filepath.Join(t.TempDir(), "a-file")
				if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(f, "bin", "abcd"), binTargetAbsent
			},
			want: "could not create the install directory",
		},
		{
			name: "write",
			setup: func(t *testing.T) (string, binTargetKind) {
				d := readOnlyDir(t)
				if err := os.Chmod(d, 0o555); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(d, "abcd"), binTargetAbsent
			},
			want: "could not write the dev PATH entry",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target, kind := tc.setup(t)
			a := &applyCtx{cwd: t.TempDir(), det: DetectionResult{pluginRoot: pluginRoot}}
			a.installDevShim(target, kind)
			if len(a.writes) != 0 {
				t.Errorf("a failed shim install reported writes %v", a.writes)
			}
			joined := strings.Join(a.notes, "\n")
			if !strings.Contains(joined, tc.want) {
				t.Errorf("no note saying %q; notes = %q", tc.want, a.notes)
			}
		})
	}
}

// TestInstallPinnedSymlinkNotesAFailedShimRemoval is the twin of the dev-shim
// case in the opposite direction: switching a dev shim back to the pinned
// entry removes the shim first, and that removal returned bare on failure.
func TestInstallPinnedSymlinkNotesAFailedShimRemoval(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so the forced failure cannot occur")
	}
	_, pluginRoot := setupHermetic(t)
	d := t.TempDir()
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	target := filepath.Join(d, "abcd")
	if err := os.WriteFile(target, []byte(renderDevShim(pluginRoot, pluginBinaryPath(pluginRoot))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	a := &applyCtx{cwd: t.TempDir(), det: DetectionResult{pluginRoot: pluginRoot}}
	a.installPinnedSymlink(target, binTargetDevShim)
	if len(a.writes) != 0 {
		t.Errorf("a failed switch reported writes %v", a.writes)
	}
	if !strings.Contains(strings.Join(a.notes, "\n"), "could not replace the dev PATH entry") {
		t.Errorf("no note for the failed shim removal; notes = %q", a.notes)
	}
}

// TestRepoWriteStepsNoteAFailedWrite covers the same silence in the steps that
// write the repository's own files: the starter config, the rules file and the
// setup stamp each dropped a failed write without a word.
func TestRepoWriteStepsNoteAFailedWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions, so the forced failure cannot occur")
	}
	setupHermetic(t)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	abcd := filepath.Join(repo, ".abcd")
	if err := os.MkdirAll(abcd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(abcd, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(abcd, 0o755) })

	for _, tc := range []struct {
		name, gap, want string
		step            func(a *applyCtx)
	}{
		{"skeleton", "skeleton.config_missing", "could not write the starter settings", (*applyCtx).stepSkeleton},
		{"rules", "rules.missing", "could not write .abcd/rules.json", (*applyCtx).stepRules},
		{"stamp", "install_meta.missing", "could not record the setup version", (*applyCtx).stepVersionStamp},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &applyCtx{
				cwd:        repo,
				approved:   map[GapCategory]bool{SafeAutocreate: true},
				gapPresent: map[string]bool{tc.gap: true},
			}
			tc.step(a)
			if len(a.writes) != 0 {
				t.Errorf("a failed write reported writes %v", a.writes)
			}
			if !strings.Contains(strings.Join(a.notes, "\n"), tc.want) {
				t.Errorf("no note saying %q; notes = %q", tc.want, a.notes)
			}
		})
	}
}
