// Command precheck-audits runs, in every package `make precheck` does not
// already cover, the tests that audit the committed tree, so that a lane meets
// them in minutes rather than most of an hour into a preflight
// (iss-2611061656036069, iss-2610230048490215). There is no list of tests to
// keep: a new audit is chosen by what it does.
//
// A test is an audit here by testreuse.Audits: it reaches the repository tree
// from its package directory, in its own body, in a same-package helper it names
// or in a package-level value, and not only through a helper that starts a process.
//
// What it cannot see: a tree reached through a helper in another package, other
// than fsutil.ModuleRoot, and a climb assembled at run time. The command and
// site packages stay on the name pattern in the Makefile (PRECHECK_AUDIT_TESTS),
// where hundreds of tests reach the root and a whole selection would not be
// minutes.
//
// Usage, from the repository root:
//
//	go run ./cmd/precheck-audits ./internal/already/run ./internal/also/run
//
// Each argument is a package the recipe runs another way, left out here. It
// exits with go test's own failure, and 2 when it cannot read a test file, is
// given something that is not a package path, or finds nothing at all, which
// would mean the detector stopped working rather than that there are no audits.
//
// It is developer tooling, never part of a release.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/testreuse"
)

// timeout is the budget of precheck's other test lines.
const timeout = "20m"

// selection is what one run hands to go test.
type selection struct {
	names    []string
	packages []string
}

func main() {
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "./") {
			fmt.Fprintf(os.Stderr, "precheck-audits: %q is not a package path starting ./\n", arg)
			os.Exit(2)
		}
	}
	got, err := selectAudits(".", os.Args[1:])
	if err == nil && len(got.names) == 0 {
		err = errors.New("found no audit test at all, so the detector has stopped working")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "precheck-audits: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("precheck-audits: %d audit tests in %d packages\n", len(got.names), len(got.packages))
	args := []string{"test", "-timeout", timeout, "-run", "^(" + strings.Join(got.names, "|") + ")$"}
	cmd := exec.Command("go", append(args, got.packages...)...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "precheck-audits: %v\n", err)
		os.Exit(2)
	}
}

// selectAudits walks root's internal and cmd trees and returns the audit tests
// of every package outside skip, names and packages both sorted.
func selectAudits(root string, skip []string) (selection, error) {
	dirs := map[string][]string{}
	for _, top := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && d.Name() == "testdata":
				return filepath.SkipDir
			case !d.IsDir() && strings.HasSuffix(path, "_test.go"):
				dirs[filepath.Dir(path)] = append(dirs[filepath.Dir(path)], path)
			}
			return nil
		})
		if err != nil {
			return selection{}, err
		}
	}
	var got selection
	for dir, files := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return selection{}, err
		}
		pkg := "./" + filepath.ToSlash(rel)
		if slices.Contains(skip, pkg) {
			continue
		}
		names, err := testreuse.Audits(files)
		if err != nil {
			return selection{}, fmt.Errorf("%s: %w", pkg, err)
		}
		if len(names) > 0 {
			got.names = append(got.names, names...)
			got.packages = append(got.packages, pkg)
		}
	}
	slices.Sort(got.names)
	got.names = slices.Compact(got.names)
	slices.Sort(got.packages)
	return got, nil
}
