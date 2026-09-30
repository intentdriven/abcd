package lifeboat

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Operand paths — the lifeboat a verb reads or writes into, the embark target,
// the pack destination — were proved with a leaf-only IsRealDir, so a symlinked
// ANCESTOR was followed while the leaf check passed (iss-2609261232464351). A
// checkout is the one place an untrusted commit can plant a link, so every level
// from the checkout an operand sits in down to the operand is proved real, and a
// checkout reached through a link inside an enclosing checkout is proved the
// same way. Outside every checkout the path is the operator's own, taken as
// given — which is what keeps macOS's /var -> /private/var from refusing every
// temporary directory.

// markCheckout makes dir read as a checkout: the marker walk takes any .git
// entry, exactly as it does for a submodule's or a worktree's .git file.
func markCheckout(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// relocate moves the fixture directory src to dst, creating dst's parents.
func relocate(t *testing.T, src, dst string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

// symlinkedOperand lays out a checkout whose committed `boats` link points at a
// directory holding a real lifeboat, and returns the operand as the operator
// would spell it: <checkout>/boats/lb. When intoCheckout is true the link
// targets another checkout, the shape a marker walk that stopped at the nearest
// .git would miss.
func symlinkedOperand(t *testing.T, intoCheckout bool) string {
	t.Helper()
	repo, elsewhere := t.TempDir(), t.TempDir()
	markCheckout(t, repo)
	if intoCheckout {
		markCheckout(t, elsewhere)
	}
	relocate(t, stdFixture(t), filepath.Join(elsewhere, "lb"))
	if err := os.Symlink(elsewhere, filepath.Join(repo, "boats")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return filepath.Join(repo, "boats", "lb")
}

// nestedOperand is the control: a real lifeboat two real levels down inside a
// checkout.
func nestedOperand(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	markCheckout(t, repo)
	return relocate(t, stdFixture(t), filepath.Join(repo, "a", "b", "lb"))
}

// lifeboatOperandGates are the entry gates that take a lifeboat directory.
var lifeboatOperandGates = map[string]func(dir string) error{
	"synthesis (principles, press release, review)": func(dir string) error {
		_, _, err := gateSynthLifeboat(dir)
		return err
	},
	"graveyard lessons": func(dir string) error {
		_, err := IngestLessons(dir, []byte(`{"schema_version":1,"lessons":[]}`))
		return err
	},
	"manifest verification": verifyManifest,
	"embark (lifeboat operand)": func(dir string) error {
		_, err := runPlanner(dir, nestedTarget())
		return err
	},
}

// nestedTarget is an embark target that passes every gate: a real directory
// outside any checkout.
var nestedTarget = func() string { return os.TempDir() }

func TestLifeboatOperandRefusesASymlinkedAncestorInsideACheckout(t *testing.T) {
	for _, intoCheckout := range []bool{false, true} {
		for name, gate := range lifeboatOperandGates {
			t.Run(name, func(t *testing.T) {
				err := gate(symlinkedOperand(t, intoCheckout))
				if !errors.Is(err, fsutil.ErrNotRealDir) {
					t.Fatalf("%s through a symlinked ancestor inside a checkout (link into a checkout: %v) = %v; want ErrNotRealDir", name, intoCheckout, err)
				}
			})
		}
	}
}

func TestLifeboatOperandAcceptsAPlainNestedPath(t *testing.T) {
	for name, gate := range lifeboatOperandGates {
		t.Run(name, func(t *testing.T) {
			if err := gate(nestedOperand(t)); errors.Is(err, fsutil.ErrNotRealDir) {
				t.Fatalf("%s refused a plain nested operand: %v", name, err)
			}
		})
	}
	// The synthesis gate and the lessons ingest pass outright; the others go on
	// to fail the fixture's unsealed manifest, which is not what is under test.
	if _, _, err := gateSynthLifeboat(nestedOperand(t)); err != nil {
		t.Errorf("gateSynthLifeboat refused a plain nested operand: %v", err)
	}
	if _, err := IngestLessons(nestedOperand(t), []byte(`{"schema_version":1,"lessons":[]}`)); err != nil {
		t.Errorf("IngestLessons refused a plain nested operand: %v", err)
	}
}

// TestLifeboatOperandFollowsALinkOutsideEveryCheckout: an ancestor link outside
// every checkout is the operator's own and is followed.
func TestLifeboatOperandFollowsALinkOutsideEveryCheckout(t *testing.T) {
	plain, elsewhere := t.TempDir(), t.TempDir()
	relocate(t, stdFixture(t), filepath.Join(elsewhere, "lb"))
	if err := os.Symlink(elsewhere, filepath.Join(plain, "boats")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := gateSynthLifeboat(filepath.Join(plain, "boats", "lb")); err != nil {
		t.Fatalf("a link outside every checkout was refused: %v", err)
	}
}

func TestEmbarkTargetRefusesASymlinkedAncestorInsideACheckout(t *testing.T) {
	repo, elsewhere := t.TempDir(), t.TempDir()
	markCheckout(t, repo)
	if err := os.MkdirAll(filepath.Join(elsewhere, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(repo, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := runPlanner(nestedOperand(t), filepath.Join(repo, "link", "target"))
	if !errors.Is(err, fsutil.ErrNotRealDir) {
		t.Fatalf("embark target through a symlinked ancestor inside a checkout = %v; want ErrNotRealDir", err)
	}
}

func TestPackDestinationRefusesASymlinkedAncestorInsideACheckout(t *testing.T) {
	source := t.TempDir()
	repo, elsewhere := t.TempDir(), t.TempDir()
	markCheckout(t, repo)
	if err := os.Symlink(elsewhere, filepath.Join(repo, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := destinationGate(filepath.Join(repo, "link", "out", "lifeboat"), source)
	if !errors.Is(err, fsutil.ErrNotRealDir) {
		t.Fatalf("pack destination through a symlinked ancestor inside a checkout = %v; want ErrNotRealDir", err)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Errorf("the refused gate left %d entr(y|ies) at the link's target", len(entries))
	}
	// The control: an absent destination below real levels of a checkout passes.
	if err := destinationGate(filepath.Join(repo, "real", "lifeboat"), source); err != nil {
		t.Fatalf("pack refused a plain nested destination: %v", err)
	}
}
