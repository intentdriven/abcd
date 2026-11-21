//go:build unix

package tools

import (
	"errors"
	"testing"
)

// TestLookPathRefusesAProgramInsideTheWorkingTree pins tools.LookPath, the
// PATH lookup every program abcd runs by name goes through: a result inside
// the working tree of the process's working directory, or of the directory
// the caller names, is refused with ErrInsideWorkingTree, while one outside
// both is returned as PATH gives it.
func TestLookPathRefusesAProgramInsideTheWorkingTree(t *testing.T) {
	r := newRig(t)
	sub, prog := checkoutWithSubdir(t, r, "gh")
	accountHomeIs(t, r.home)
	withHome(t, r.home, func(string) (string, error) { return prog, nil })
	outside := t.TempDir()

	wd := workingDir
	t.Cleanup(func() { workingDir = wd })
	workingDir = func() (string, error) { return sub, nil }
	if _, err := LookPath("gh", ""); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath from a subdirectory of the checkout = %v, want ErrInsideWorkingTree", err)
	}
	if _, err := Resolve("gh"); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("Resolve from a subdirectory of the checkout = %v, want ErrInsideWorkingTree", err)
	}

	workingDir = func() (string, error) { return outside, nil }
	if _, err := LookPath("gh", sub); !errors.Is(err, ErrInsideWorkingTree) {
		t.Errorf("LookPath for a directory in the checkout = %v, want ErrInsideWorkingTree", err)
	}
	if got, err := LookPath("gh", ""); err != nil || got != prog {
		t.Errorf("LookPath from outside the checkout = %q, %v; want %q", got, err, prog)
	}
}
