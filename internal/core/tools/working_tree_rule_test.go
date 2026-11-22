//go:build unix

package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkoutWithSubdir lays out a working tree at r.repo (a .git directory at
// its root), a subdirectory of it, and a program named tool in the tree's own
// bin folder, and returns the subdirectory and the program's path.
func checkoutWithSubdir(t *testing.T, r *rig, tool string) (sub, prog string) {
	t.Helper()
	sub = filepath.Join(r.repo, "pkg", "deep")
	bin := filepath.Join(r.repo, "bin")
	for _, d := range []string{filepath.Join(r.repo, ".git"), sub, bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	prog = filepath.Join(bin, tool)
	if err := os.WriteFile(prog, []byte("#!/bin/sh\necho 8.30.1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return sub, prog
}

// TestAProgramAtTheCheckoutRootIsRefusedFromASubdirectory pins that the
// installer holds a program to the whole working tree its guard lies in, not
// to the guard directory alone: from a subdirectory, a program PATH finds in
// the checkout's own bin folder is refused and never run, as it is from the
// checkout's root. A tree that is the account's own home holds nothing.
func TestAProgramAtTheCheckoutRootIsRefusedFromASubdirectory(t *testing.T) {
	r := newRig(t)
	sub, prog := checkoutWithSubdir(t, r, "gitleaks")
	r.env["PATH"] = filepath.Dir(prog) + string(os.PathListSeparator) + r.env["PATH"]
	accountHomeIs(t, r.home)

	in := r.installer()
	in.Guard = sub
	if _, err := in.admit("gitleaks"); err == nil || !strings.Contains(err.Error(), "inside the repository") {
		t.Fatalf("admit from a subdirectory = %v, want the in-checkout refusal", err)
	}
	if !WithinTree(prog, prog, sub) {
		t.Error("WithinTree does not hold a program at the checkout's root to a guard in its subdirectory")
	}
	plan := in.Plan("gitleaks")
	if plan.Action == UseExisting || plan.Existing != nil {
		t.Errorf("the plan counts the program inside the checkout: %+v", plan)
	}
	for _, c := range r.fx.calls {
		if c[0] == prog {
			t.Fatalf("the program inside the checkout was run: %q", c)
		}
	}

	accountHomeIs(t, r.repo)
	if _, err := in.admit("gitleaks"); err != nil {
		t.Errorf("a program in the account home's own tree is refused from a subdirectory of it: %v", err)
	}
}
