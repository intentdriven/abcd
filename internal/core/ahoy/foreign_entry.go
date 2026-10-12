package ahoy

import (
	"bytes"
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// abcdMainPackage is the package path every abcd build records as its main
// package, release or source build alike.
const abcdMainPackage = "github.com/intentdriven/abcd/cmd/abcd"

// maxBuildValueRunes bounds each value describeBuild prints from the file, so a
// binary stamped with an enormous version or module path cannot flood the gap's
// detail or the JSON.
const maxBuildValueRunes = 64

// describeForeignFile says what a regular file standing at the PATH entry is,
// so the person deciding whether to clear it does not have to inspect it by
// hand (iss-2609120447482255): its size, when it was last modified, and whether
// it identifies as an abcd build and which. Only a regular file is read — a
// FIFO or a device would block or have side effects on open — and only its
// build metadata, never executed. The caller's lstat is not trusted to still
// hold at the read: the open follows no symlink and never blocks, and the
// descriptor itself must be a regular file, so a FIFO swapped in after the
// lstat is described by the size and time alone, like any non-regular entry.
func describeForeignFile(path string, fi os.FileInfo) string {
	desc := fmt.Sprintf("%d bytes, last modified %s", fi.Size(), fi.ModTime().UTC().Format("2006-01-02 15:04 UTC"))
	if !fi.Mode().IsRegular() {
		return desc
	}
	f, _, err := fsutil.OpenRegular(path)
	if err != nil {
		return desc
	}
	defer f.Close()
	bi, err := buildinfo.Read(f)
	if err != nil {
		return desc + "; not an abcd build (it carries no Go build metadata)"
	}
	return desc + "; " + describeBuild(bi)
}

// foreignOccupant names what kind of thing an lstat found at the PATH entry, so
// a directory or a named pipe is not called a file. regular is the caller's
// own wording for the ordinary case.
func foreignOccupant(fi os.FileInfo, regular string) string {
	m := fi.Mode()
	switch {
	case m.IsRegular():
		return regular
	case m.IsDir():
		return "a directory"
	case m&os.ModeNamedPipe != 0:
		return "a named pipe"
	case m&os.ModeSocket != 0:
		return "a socket"
	case m&os.ModeDevice != 0:
		return "a device"
	}
	return "something other than a regular file"
}

// describeBuild identifies a Go binary from its embedded build metadata: an abcd
// build by its version and revision, or another program by its package path.
// Every value comes from the file, so it is sanitised for the terminal and cut
// to a bounded length; the revision is named once, by its first twelve runes.
func describeBuild(bi *debug.BuildInfo) string {
	if bi.Path != abcdMainPackage {
		return "not an abcd build: a Go program built from " + buildValue(bi.Path, maxBuildValueRunes, "…")
	}
	v := bi.Main.Version
	if v == "" || v == "(devel)" {
		return "an abcd build of no stated version"
	}
	out := "an abcd build, version " + buildValue(v, maxBuildValueRunes, "…")
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			out += " (revision " + buildValue(s.Value, 12, "") + ")"
			break
		}
	}
	return out
}

// buildValue sanitises one value read from a binary's build metadata and cuts it
// to at most n runes, appending mark when it cut. The cut counts runes, so it
// never splits one.
func buildValue(s string, n int, mark string) string {
	s = termsafe.Sanitize(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + mark
}

// pseudoVersionTail is the commit stamp a Go pseudo-version ends in
// (yyyymmddhhmmss-abcdefabcdef): the version Go gives a build from an
// untagged commit, which names no release.
var pseudoVersionTail = regexp.MustCompile(`[0-9]{14}-[0-9a-f]{12}$`)

// copyRelease names the release an owned PATH copy claims to be, read from
// its bytes' build metadata, never by running it. A seam: the package's
// fixtures are shell scripts with no build metadata, so TestMain stands in a
// fixed release and the tests of the reader itself put the real one back.
var copyRelease = releaseTagOfBinary

// releaseTagOfBinary reads the release tag a binary's Go build metadata
// claims (releaseOfBuild), or why it claims none.
func releaseTagOfBinary(data []byte) (string, error) {
	bi, err := buildinfo.Read(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("it carries no Go build metadata, so it names no release")
	}
	return releaseOfBuild(bi)
}

// releaseOfBuild is the release an abcd build claims: the main module version
// Go stamps from the release tag the build was made at. A release build is
// made at its tag from a clean tree, so anything else (a source build's
// (devel), a pseudo-version from an untagged commit, a +dirty tree, or
// another program) names no release.
func releaseOfBuild(bi *debug.BuildInfo) (string, error) {
	if bi.Path != abcdMainPackage {
		return "", errors.New("it is not an abcd build: a Go program built from " + buildValue(bi.Path, maxBuildValueRunes, "…"))
	}
	v := bi.Main.Version
	if _, ok := parseReleaseTag(v); !ok || strings.Contains(v, "+") || pseudoVersionTail.MatchString(v) {
		if v == "" {
			v = "none"
		}
		return "", errors.New("its build metadata names no release tag (version " + buildValue(v, maxBuildValueRunes, "…") + "), so it is not a release build")
	}
	return v, nil
}
