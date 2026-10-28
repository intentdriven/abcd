package ahoy

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdrepo"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// goRunModule is the seam for reading whether the running binary is a go run
// build, and of which module (vintage.GoRunModule); a test swaps it.
var goRunModule = vintage.GoRunModule

// swapGoRunModule replaces the seam and returns a restore func.
func swapGoRunModule(f func() (string, bool)) (restore func()) {
	prev := goRunModule
	goRunModule = f
	return func() { goRunModule = prev }
}

// checkoutBuild is the seam for reading whether the running binary is the
// build scripts/abcd has just made, of which module, and its executable
// (vintage.CheckoutBuild); a test swaps it.
var checkoutBuild = vintage.CheckoutBuild

// swapCheckoutBuild replaces the seam and returns a restore func.
func swapCheckoutBuild(f func() (string, string, bool)) (restore func()) {
	prev := checkoutBuild
	checkoutBuild = f
	return func() { checkoutBuild = prev }
}

// maxGoModBytes bounds the read of a checkout's go.mod.
const maxGoModBytes = 1 << 20

// goRunOfCheckout reports whether the running binary is a build of the
// checkout holding cwd as that tree stands: a go run build of the module the
// checkout declares (vintage.GoRunModule, iss-2610240824047504), or the build
// scripts/abcd has just made in that checkout's own local tier
// (vintage.CheckoutBuild, iss-2610241240133009). Either is the tree as it
// stands, so the build is that checkout's own and never older than it. The
// local-tier build must be the same file as <top>/abcdrepo.CheckoutBuildRel,
// where top is the checkout holding cwd: another checkout's build of the same
// module, run by its script's path, is that tree's build and not this one's. It
// reads git's top level and the go.mod there, and touches no network.
func goRunOfCheckout(cwd string) bool {
	top, err := gitutil.Toplevel(cwd)
	if err != nil || top == "" {
		return false
	}
	if mod, ok := goRunModule(); ok && mod != "" {
		return checkoutModuleAt(top) == mod
	}
	mod, exe, ok := checkoutBuild()
	if !ok || mod == "" || checkoutModuleAt(top) != mod {
		return false
	}
	return fsutil.SameFile(exe, filepath.Join(top, filepath.FromSlash(abcdrepo.CheckoutBuildRel)))
}

// checkoutModule is the module path the go.mod at the top of the checkout
// holding cwd declares, or "" when there is none.
func checkoutModule(cwd string) string {
	top, err := gitutil.Toplevel(cwd)
	if err != nil || top == "" {
		return ""
	}
	return checkoutModuleAt(top)
}

// checkoutModuleAt is the module path the go.mod at a checkout's top level
// declares, or "" when there is none.
func checkoutModuleAt(top string) string {
	data, err := fsutil.ReadGuarded(filepath.Join(top, "go.mod"), maxGoModBytes)
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`)
		}
	}
	return ""
}
