package vintage

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdrepo"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// GoRunModule reports the module path of the running binary when it is a
// `go run` build of that module's abcd command: go compiles it from the
// working tree into a temporary go-build directory and runs it once, so it is
// built from the tree as it stands and is never older than its source. Such a
// build carries no vcs.revision (go run never stamps one), which is why it
// reads Unknown to Compare; GoRunModule is what lets a caller tell it apart
// from a real unstamped binary (iss-2610240824047504). ok is false for any
// other build: a binary on disk, a go test binary, an installed copy, and the
// build scripts/abcd makes, which CheckoutBuild reports.
func GoRunModule() (module string, ok bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	return goRunModuleOf(bi.Path, bi.Main.Path, bi.Main.Version, bi.Settings, exe)
}

// CheckoutBuild reports the module path and executable of the running binary
// when it is the build scripts/abcd has just made of a checkout's tree
// (iss-2610241240133009): an unstamped development build of the module's
// cmd/abcd at abcdrepo.CheckoutBuildRel, run by the script, which names it in
// abcdrepo.CheckoutBuildEnv after it relinks it. Like a go run build it is the
// tree as it stood a moment ago, but it stays on disk: a copy left there and run
// directly, after the tree has moved on, carries no mark and reports ok false.
// Which checkout it is the build of is the caller's to check, against exe.
func CheckoutBuild() (module, exe string, ok bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	exe, err := os.Executable()
	if err != nil {
		return "", "", false
	}
	module, ok = checkoutBuildOf(bi.Path, bi.Main.Path, bi.Main.Version, bi.Settings, exe, os.Getenv(abcdrepo.CheckoutBuildEnv))
	return module, exe, ok
}

// goRunModuleOf is GoRunModule's pure core: the command's package path, the
// main module's path and version, the build settings, and the executable's
// path. A go run build is a development build ("(devel)") of the module's
// cmd/abcd, with no vcs.revision, run from a go-build work directory.
func goRunModuleOf(pkgPath, mainPath, mainVersion string, settings []debug.BuildSetting, exe string) (string, bool) {
	if !unstampedCommand(pkgPath, mainPath, mainVersion, settings) {
		return "", false
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(exe)), "/") {
		if strings.HasPrefix(part, "go-build") {
			return mainPath, true
		}
	}
	return "", false
}

// checkoutBuildOf is CheckoutBuild's core: the build facts goRunModuleOf reads,
// the executable's path, and the value of abcdrepo.CheckoutBuildEnv. The
// executable must sit at a checkout's abcdrepo.CheckoutBuildRel and be the same
// file the mark names; it reads the file system for that, and nothing else.
func checkoutBuildOf(pkgPath, mainPath, mainVersion string, settings []debug.BuildSetting, exe, marked string) (string, bool) {
	if !unstampedCommand(pkgPath, mainPath, mainVersion, settings) || marked == "" {
		return "", false
	}
	if !strings.HasSuffix(filepath.ToSlash(exe), "/"+abcdrepo.CheckoutBuildRel) {
		return "", false
	}
	if !fsutil.SameFile(exe, marked) {
		return "", false
	}
	return mainPath, true
}

// unstampedCommand reports a development build ("(devel)") of the main
// module's cmd/abcd with no vcs.revision stamped, the build both go run and
// scripts/abcd make.
func unstampedCommand(pkgPath, mainPath, mainVersion string, settings []debug.BuildSetting) bool {
	if mainPath == "" || mainVersion != "(devel)" || pkgPath != mainPath+"/cmd/abcd" {
		return false
	}
	for _, s := range settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			return false
		}
	}
	return true
}
