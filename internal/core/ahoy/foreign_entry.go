package ahoy

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// abcdMainPackage is the package path every abcd build records as its main
// package, release or source build alike.
const abcdMainPackage = "github.com/intentdriven/abcd/cmd/abcd"

// describeForeignFile says what a regular file standing at the PATH entry is,
// so the person deciding whether to clear it does not have to inspect it by
// hand (iss-2609120447482255): its size, when it was last modified, and whether
// it identifies as an abcd build and which. Only a regular file is read — a
// FIFO or a device would block or have side effects on open — and only its
// build metadata, never executed.
func describeForeignFile(path string, fi os.FileInfo) string {
	desc := fmt.Sprintf("%d bytes, last modified %s", fi.Size(), fi.ModTime().UTC().Format("2006-01-02 15:04 UTC"))
	if !fi.Mode().IsRegular() {
		return desc
	}
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return desc + "; not an abcd build (it carries no Go build metadata)"
	}
	return desc + "; " + describeBuild(bi)
}

// describeBuild identifies a Go binary from its embedded build metadata: an abcd
// build by its version and revision, or another program by its package path.
// Every value comes from the file, so it is sanitised for the terminal.
func describeBuild(bi *debug.BuildInfo) string {
	if bi.Path != abcdMainPackage {
		return "not an abcd build: a Go program built from " + termsafe.Sanitize(bi.Path)
	}
	v := bi.Main.Version
	if v == "" || v == "(devel)" {
		return "an abcd build of no stated version"
	}
	out := "an abcd build, version " + termsafe.Sanitize(v)
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			out += " (revision " + shortRev(termsafe.Sanitize(s.Value)) + ")"
		}
	}
	return out
}
