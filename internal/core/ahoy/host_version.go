package ahoy

// The installed-version reading of an agent tool's command, and the one floor
// a vendor's version is compared with (itd-2610030814013772,
// spc-2610031156364295 open question 5, decided (a)). It is written to be
// shared: the harness-version check (itd-2610031026190632,
// spc-2610031342374947) widens this one reading to a table of harnesses in
// this file rather than restating it.
//
// The reading runs a program found on PATH through the runner's one launch
// primitive for a vendor binary (runner.Exec, internal/core/runner/proc.go),
// so its admission, its bounds and its group kill are the harness's own and
// never a second copy: an argv vector and never a shell line, the binary
// refused when it is not absolute, when group or other can write it or a
// folder it is reached through, or when it resolves inside the project; the
// git-scrubbed parent environment, the null device on stdin, a working
// directory that is not the project, each stream bounded, and a run past its
// time killed with the process group it leads, through the pid that handle
// holds.

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// claudeCommand is the command Claude Code installs on PATH.
const claudeCommand = "claude"

// claudeCodeAgentsFloor is the first Claude Code release from which every
// session type reads AGENTS.md as project instructions on its own. Native
// reading began at v2.1.277; until v2.1.281 a remote flag that defaulted off
// without telemetry made sessions with telemetry switched off, and some cloud
// providers, skip it silently. Source: the 2026-10-03 re-check,
// .abcd/development/research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md
// (findings 1 and 5, from the release pages v2.1.277 to v2.1.282). This is the
// one place a vendor's version lives in the code.
var claudeCodeAgentsFloor = hostVersion{Major: 2, Minor: 1, Patch: 281}

// hostVersion is a release number. Versions compare part by part as numbers,
// never as strings, so 1.10.0 sits above 1.9.2.
type hostVersion struct {
	Major, Minor, Patch int
}

// less reports whether v is an earlier release than w.
func (v hostVersion) less(w hostVersion) bool {
	if v.Major != w.Major {
		return v.Major < w.Major
	}
	if v.Minor != w.Minor {
		return v.Minor < w.Minor
	}
	return v.Patch < w.Patch
}

// hostVersionTimeout bounds one reading: a version flag answers at once, and a
// command that does not is read as giving no version. A variable so a test
// can shorten it.
var hostVersionTimeout = 3 * time.Second

// hostVersionMaxOutput bounds each of the command's streams; a version line
// is short, and a command writing past the bound gives no version.
const hostVersionMaxOutput = 4 << 10

// versionPattern is a major.minor.patch release number.
var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Why no version was read. A caller that only warns below a floor treats every
// one of them as "raise nothing"; the harness check names them.
var (
	errHostAbsent     = errors.New("not on the search path")
	errHostRefused    = errors.New("resolves to a relative path, inside the project or through a folder others can write, so it is never run")
	errHostNoAnswer   = errors.New("gave no answer in time, or exited with an error")
	errHostNoVersion  = errors.New("printed no version")
	errHostNotStarted = errors.New("could not be started")
)

// readHostVersion asks command, found on PATH, for its version with its
// --version flag, and returns the first major.minor.patch it prints on
// stdout. project is the folder the reading is made for; a command inside it
// is never run. The error says why no version was read. It is the one reading
// of an agent tool's installed version: the harness-version check calls it per
// harness.
func readHostVersion(command, project string) (hostVersion, error) {
	var guards []string
	if project != "" {
		guards = []string{project}
	}
	out, err := runner.Exec(context.Background(), runner.Command{
		Name: command, Args: []string{"--version"}, Dir: "/", Guards: guards,
		Timeout: hostVersionTimeout, MaxStdout: hostVersionMaxOutput, MaxStderr: hostVersionMaxOutput,
	})
	switch {
	case errors.Is(err, runner.ErrNotOnPath):
		return hostVersion{}, errHostAbsent
	case errors.Is(err, runner.ErrRefused):
		return hostVersion{}, errHostRefused
	case errors.Is(err, runner.ErrNotStarted):
		return hostVersion{}, errHostNotStarted
	case err != nil:
		return hostVersion{}, errHostNoAnswer
	}
	// stdout alone: a diagnostic on stderr can carry a dotted number of its own.
	return parseHostVersion(out.Stdout)
}

// parseHostVersion is the first major.minor.patch in out.
func parseHostVersion(out []byte) (hostVersion, error) {
	m := versionPattern.FindSubmatch(out)
	if m == nil {
		return hostVersion{}, errHostNoVersion
	}
	var parts [3]int
	for i := range parts {
		n, err := strconv.Atoi(string(m[i+1]))
		if err != nil {
			return hostVersion{}, errHostNoVersion
		}
		parts[i] = n
	}
	return hostVersion{Major: parts[0], Minor: parts[1], Patch: parts[2]}, nil
}

// String is for a test's failure message.
func (v hostVersion) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// readClaudeVersion is the reading the host warning makes. A variable so no
// unit test runs a harness of the machine's own: the package's tests and the
// front-door tests swap it through the seams below, and the version test puts
// the real reading back over a fake command.
var readClaudeVersion = func(project string) (hostVersion, error) {
	return readHostVersion(claudeCommand, project)
}

// noHostVersion reads no version, as when no agent tool is on PATH.
func noHostVersion(string) (hostVersion, error) { return hostVersion{}, errHostAbsent }

// NoHostVersionForTest makes the version reading find no agent tool, as on a
// machine with none on PATH, and returns a restore func. It is a test-only
// seam (in the manner of SetCurrentVintageForTest): a front-door test package
// runs installs in-process and calls it from its TestMain, so no unit test
// runs the machine's own agent tool and no test's output depends on its
// release. Production never calls it.
func NoHostVersionForTest() (restore func()) {
	return swapHostVersion(noHostVersion)
}

// OldHostVersionForTest makes the version reading report an agent tool one
// release below the floor, without running anything, and returns a restore
// func: the stub a front-door test proves the warning renders with.
// Production never calls it.
func OldHostVersionForTest() (restore func()) {
	old := claudeCodeAgentsFloor
	if old.Patch > 0 {
		old.Patch--
	} else {
		old.Minor--
	}
	return swapHostVersion(func(string) (hostVersion, error) { return old, nil })
}

func swapHostVersion(f func(string) (hostVersion, error)) (restore func()) {
	prev := readClaudeVersion
	readClaudeVersion = f
	return func() { readClaudeVersion = prev }
}
