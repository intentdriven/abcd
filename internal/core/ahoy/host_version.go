package ahoy

// The installed-version reading of an agent tool's command, and the one floor
// a vendor's version is compared with (itd-2610030814013772,
// spc-2610031156364295 open question 5, decided (a)). It is written to be
// shared: the harness-version check (itd-2610031026190632,
// spc-2610031342374947) widens this one reading to a table of harnesses in
// this file rather than restating it.
//
// The reading runs a program found on PATH, so it holds to the runner's
// discipline for a vendor binary (internal/core/runner/proc.go): an argv
// vector and never a shell line, the git-scrubbed parent environment
// (gitutil.ScrubbedEnv), the null device on stdin, a working directory that is
// not the project, output read up to a bound and the rest discarded, and a
// run past its time killed with the process group it leads, through the pid
// this handle holds. A command that resolves to a relative path or inside the
// project is repository content and is never run.

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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

// hostVersionMaxOutput is how much of the command's output is kept; a version
// line is short, and the rest is discarded unread.
const hostVersionMaxOutput = 4 << 10

// versionPattern is a major.minor.patch release number.
var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// Why no version was read. A caller that only warns below a floor treats every
// one of them as "raise nothing"; the harness check names them.
var (
	errHostAbsent     = errors.New("not on the search path")
	errHostRefused    = errors.New("resolves to a relative path or inside the project, so it is never run")
	errHostNoAnswer   = errors.New("gave no answer in time, or exited with an error")
	errHostNoVersion  = errors.New("printed no version")
	errHostNotStarted = errors.New("could not be started")
)

// admitHostCommand resolves command on PATH and refuses a result that is not
// absolute or that lies inside project, lexically or after its links are
// resolved: a program there is repository content and is never run. A
// variable so the harness check can put the runner's own admission here.
var admitHostCommand = func(command, project string) (string, error) {
	p, err := exec.LookPath(command)
	if err != nil {
		return "", errHostAbsent
	}
	if !filepath.IsAbs(p) {
		return "", errHostRefused
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", errHostAbsent
	}
	if project != "" {
		guards := []string{filepath.Clean(project)}
		if g, gerr := filepath.EvalSymlinks(project); gerr == nil {
			guards = append(guards, g)
		}
		fold := fsutil.CaseFoldingFS()
		for _, g := range guards {
			for _, c := range []string{filepath.Clean(p), resolved} {
				if fsutil.PathWithin(c, g, fold) {
					return "", errHostRefused
				}
			}
		}
	}
	return p, nil
}

// readHostVersion asks command, found on PATH, for its version with its
// --version flag, and returns the first major.minor.patch it prints. project
// is the folder the reading is made for; a command inside it is never run.
// The error says why no version was read.
func readHostVersion(command, project string) (hostVersion, error) {
	bin, err := admitHostCommand(command, project)
	if err != nil {
		return hostVersion{}, err
	}
	// #nosec G204 -- bin is a fixed command name resolved by admitHostCommand
	// (absolute, outside the project); the argument is a constant.
	cmd := exec.Command(bin, "--version")
	cmd.Dir = string(filepath.Separator)
	cmd.Env = gitutil.ScrubbedEnv()
	cmd.Stdin = nil
	out := &keepFirst{limit: hostVersionMaxOutput}
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return hostVersion{}, errHostNotStarted
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(hostVersionTimeout)
	defer timer.Stop()
	select {
	case err = <-done:
	case <-timer.C:
		// The group this child leads, through the pid this handle holds; the
		// leader is not yet reaped, so the group id is still its.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return hostVersion{}, errHostNoAnswer
	}
	if err != nil {
		return hostVersion{}, errHostNoAnswer
	}
	return parseHostVersion(out.buf.Bytes())
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

// keepFirst keeps the first limit bytes written to it and discards the rest,
// reporting every write as taken so the child is never stopped by a full pipe.
type keepFirst struct {
	buf   bytes.Buffer
	limit int
}

func (k *keepFirst) Write(p []byte) (int, error) {
	if room := k.limit - k.buf.Len(); room > 0 {
		if len(p) > room {
			k.buf.Write(p[:room])
		} else {
			k.buf.Write(p)
		}
	}
	return len(p), nil
}

// String is for a test's failure message.
func (v hostVersion) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

// readClaudeVersion is the reading the host warning makes. A variable so the
// package's tests run no harness of the machine's own; the version test puts
// the real reading back over a fake command.
var readClaudeVersion = func(project string) (hostVersion, error) {
	return readHostVersion(claudeCommand, project)
}
