package vintage

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// GoRunModule reports the module path of the running binary when it is a
// `go run` build of that module's abcd command: go compiles it from the
// working tree into a temporary go-build directory and runs it once, so it is
// built from the tree as it stands and is never older than its source. Such a
// build carries no vcs.revision (go run never stamps one), which is why it
// reads Unknown to Compare; GoRunModule is what lets a caller tell it apart
// from a real unstamped binary (iss-2610240824047504). ok is false for any
// other build: a binary on disk, a go test binary, an installed copy.
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

// goRunModuleOf is GoRunModule's pure core: the command's package path, the
// main module's path and version, the build settings, and the executable's
// path. A go run build is a development build ("(devel)") of the module's
// cmd/abcd, with no vcs.revision, run from a go-build work directory.
func goRunModuleOf(pkgPath, mainPath, mainVersion string, settings []debug.BuildSetting, exe string) (string, bool) {
	if mainPath == "" || mainVersion != "(devel)" || pkgPath != mainPath+"/cmd/abcd" {
		return "", false
	}
	for _, s := range settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			return "", false
		}
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(exe)), "/") {
		if strings.HasPrefix(part, "go-build") {
			return mainPath, true
		}
	}
	return "", false
}
