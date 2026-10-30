package vintage

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdrepo"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// GoRunModule reports the module path of the running binary when it is a
// `go run` build of that module's abcd command: go builds it from the working
// tree as it stands, reusing its build cache's copy when nothing changed, and
// runs it, so it is never older than its source. Such a
// build carries no vcs.revision (go run never stamps one), which is why it
// reads Unknown to Compare; GoRunModule is what lets a caller tell it apart
// from a real unstamped binary (iss-2610240824047504). ok is false for any
// other build: a binary on disk, a go test binary, an installed copy, and the
// build scripts/abcd.sh makes, which CheckoutBuild reports.
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
// when it is the build scripts/abcd.sh has just made of a checkout's tree
// (iss-2610241240133009): an unstamped development build of the module's
// cmd/abcd at abcdrepo.CheckoutBuildRel, run by the script, which names it in
// abcdrepo.CheckoutBuildEnv after it relinks it. Like a go run build it is the
// tree as it stood a moment ago, but it stays on disk: a copy left there and run
// directly, after the tree has moved on, carries no mark and reports ok false.
// Which checkout it is the build of is the caller's to check, against exe.
// The mark is read as the process started (checkoutMark), so a process this
// binary starts, which never sees it, cannot pass for the script's run.
func CheckoutBuild() (module, exe string, ok bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	exe, err := os.Executable()
	if err != nil {
		return "", "", false
	}
	module, ok = checkoutBuildOf(bi.Path, bi.Main.Path, bi.Main.Version, bi.Settings, exe, checkoutMark)
	return module, exe, ok
}

// checkoutMark is abcdrepo.CheckoutBuildEnv as this process started with it.
// The mark vouches only for the binary scripts/abcd.sh has just built and run, so
// it is taken out of the environment as the package initialises: a process
// abcd starts (a session a load waiter or a build runs, a hook) and anything
// that runs the local-tier binary by path from there never inherits it.
var checkoutMark = takeEnv(abcdrepo.CheckoutBuildEnv)

// CheckoutMark is the executable scripts/abcd.sh named when it started this
// process (abcdrepo.CheckoutBuildEnv), or "" when nothing did.
func CheckoutMark() string { return checkoutMark }

// takeEnv returns key's value and removes it from the process environment.
func takeEnv(key string) string {
	v := os.Getenv(key)
	os.Unsetenv(key)
	return v
}

// goRunExe is either place go run runs its executable from. On a build it is
// the link's output: <name> in the exe directory of an action directory (b001,
// b002, ...) of the build's work directory, which os.MkdirTemp names
// go-build<digits> under GOTMPDIR or the system temporary directory. On a
// build cache hit, the commonest case, it is the copy go run cached
// (cmd/go/internal/run sets CacheExecutable): <name> in the build cache's
// "<output id>-d" entry, under the directory named by the id's first byte,
// wherever GOCACHE puts the cache.
var goRunExe = regexp.MustCompile(`(/go-build[0-9]+/b[0-9]{3,}/exe|/[0-9a-f]{2}/[0-9a-f]{64}-d)/[^/]+$`)

// goRunModuleOf is GoRunModule's pure core: the command's package path, the
// main module's path and version, the build settings, and the executable's
// path. A go run build is a development build ("(devel)") of the module's
// cmd/abcd, with no vcs.revision, run from where go run runs it (goRunExe). Any
// directory merely named go-build... on the way, such as a checkout whose
// local tier holds a binary left there, is not that place.
func goRunModuleOf(pkgPath, mainPath, mainVersion string, settings []debug.BuildSetting, exe string) (string, bool) {
	if !unstampedCommand(pkgPath, mainPath, mainVersion, settings) || !goRunExe.MatchString(filepath.ToSlash(exe)) {
		return "", false
	}
	return mainPath, true
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
// scripts/abcd.sh make.
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
