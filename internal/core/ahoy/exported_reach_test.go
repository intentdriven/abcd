package ahoy

import (
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/reachaudit"
)

// TestEveryExportedAhoyFunctionHasAFrontDoor is the caller audit iss-33 asked
// for, held at zero for this package: an exported function exists to be reached
// from outside it, so one that no production code outside the package names is
// silent scaffolding — it compiles, it can rot, and nothing would notice. The
// loud way to stage a function is to leave it unexported until a front door
// calls it (loud-staging.md); an exported one nothing reaches is refused here.
//
// The matcher is reachaudit's, the one every package's audit shares: it parses
// the module's production Go (tests, testdata/, nested modules and files no
// release target compiles excluded) and counts only a selector on the imported
// ahoy package, so a comment, a string literal or a local variable shadowing the
// import's name is no caller. Tests do not count: a function reached only by its
// own test is exactly the scaffolding the rule is about. The ForTest suffix
// exempts a declared cross-package test seam by its name alone. The other core
// packages are held by reachaudit's baseline ratchet; this one has no baseline.
func TestEveryExportedAhoyFunctionHasAFrontDoor(t *testing.T) {
	reach, err := reachaudit.Scan(filepath.Join("..", "..", ".."), "internal/core/ahoy")
	if err != nil {
		t.Fatal(err)
	}
	if len(reach) == 0 {
		t.Fatal("found no exported functions; the audit is reading the wrong directory")
	}
	if unreached := reach.Unreached(); len(unreached) > 0 {
		t.Fatalf("exported ahoy functions no production code outside the package calls — wire a front door or unexport/delete them: %v", unreached)
	}
}
