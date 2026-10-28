package ahoy

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

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

// maxGoModBytes bounds the read of a checkout's go.mod.
const maxGoModBytes = 1 << 20

// goRunOfCheckout reports whether the running binary is a go run build of the
// module the checkout holding cwd declares, or the build scripts/abcd makes in
// its local tier (vintage.GoRunModule). In a source checkout of abcd,
// scripts/abcd is the sanctioned way to run abcd (iss-2610241240133009), and
// both builds are the tree as it stands, so the build is that checkout's own
// and never older than it (iss-2610240824047504). It reads git's top level and
// the go.mod there, and touches no network.
func goRunOfCheckout(cwd string) bool {
	mod, ok := goRunModule()
	if !ok || mod == "" {
		return false
	}
	return checkoutModule(cwd) == mod
}

// checkoutModule is the module path the go.mod at the top of the checkout
// holding cwd declares, or "" when there is none.
func checkoutModule(cwd string) string {
	top, err := gitutil.Toplevel(cwd)
	if err != nil || top == "" {
		return ""
	}
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
