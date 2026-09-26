package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Answer is a caller's reply to the install question: yes or no, and in words
// how the reply was reached (typed at a terminal, named with a flag, no
// terminal to ask at), which the result repeats so a no is never silent.
type Answer struct {
	Yes bool
	Why string
}

// Confirm asks the person whether to run the install the explanation shows.
// The front door supplies it; a nil Confirm is a no.
type Confirm func(Explanation) Answer

// Result reports one Install: what it ran and whether it worked, or why it ran
// nothing and what the capability continues on (criterion 2).
type Result struct {
	Tool string `json:"tool"`
	// Ran reports that the install step was executed.
	Ran bool `json:"ran"`
	// Argv is the step as run (argv[0] resolved), when Ran.
	Argv []string `json:"argv,omitempty"`
	// Installed reports that the step exited zero.
	Installed bool `json:"installed"`
	// Verified reports that the verify command exited zero afterwards.
	Verified bool `json:"verified"`
	// Declined reports that nobody said yes (a no, no confirmation, CI).
	Declined bool `json:"declined"`
	// Why is the reason nothing ran, or what failed.
	Why string `json:"why,omitempty"`
	// Output is the bounded tail of the step's output on a failure, or the
	// verify command's first line on success.
	Output string `json:"output,omitempty"`
	// OnDecline is the capability's standing after a no or a failure.
	OnDecline string `json:"on_decline,omitempty"`
	step      string
	verify    string
}

// Summary is the one line a verb reports for the result. Every path that
// installed nothing ends with the capability's standing ("continuing on the
// native secret scanner"), so a no is loud rather than silent (spec scope 4).
func (r Result) Summary() string {
	tail := ""
	if r.OnDecline != "" {
		tail = "; " + r.OnDecline
	}
	switch {
	case !r.Ran:
		return r.Tool + " not installed (" + r.Why + ")" + tail
	case !r.Installed:
		s := r.Tool + ": ran " + r.step + " — it failed (" + r.Why + ")"
		if r.Output != "" {
			s += ": " + r.Output
		}
		return s + tail
	case !r.Verified:
		s := r.Tool + ": ran " + r.step + " — it succeeded, but the verify " + r.verify + " failed (" + r.Why + ")"
		if r.Output != "" {
			s += ": " + r.Output
		}
		return s + tail
	}
	s := r.Tool + ": ran " + r.step + " — installed, and verified with " + r.verify
	if r.Output != "" {
		s += " (" + r.Output + ")"
	}
	return s
}

// Installer runs confirmed install steps. Every process-touching seam is a
// field so the whole decision path is exercised in tests with no package
// manager; Default is the production wiring.
type Installer struct {
	// LookPath resolves a bare program name on the operator's PATH.
	LookPath func(string) (string, error)
	// Run executes an argv whose argv[0] is already resolved, without a shell.
	Run func(context.Context, []string) ([]byte, error)
	// Getenv reads the environment (CI detection).
	Getenv func(string) string
	// GOOS selects the platform's step.
	GOOS string
	// Guard is the directory a resolved program must lie outside: the
	// repository the verb was run from. A PATH entry pointing into it is
	// repository content, which is trusted to choose nothing that runs.
	Guard string
}

// Default is the production installer for a verb run from guard (its
// repository root, or its working directory outside one).
func Default(guard string) *Installer {
	return &Installer{LookPath: exec.LookPath, Run: runArgv, Getenv: os.Getenv, GOOS: runtime.GOOS, Guard: guard}
}

// Install explains name as capability uses it, asks confirm, and on a yes runs
// the registry's step for this platform and then its verify command. It is the
// package-level spelling of Default(guard).Install.
func Install(name string, capability Capability, confirm Confirm, guard string) Result {
	return Default(guard).Install(name, capability, confirm)
}

const (
	// installTimeout bounds a package manager run; a timed-out step's whole
	// process group is killed through its own handle.
	installTimeout = 15 * time.Minute
	// verifyTimeout bounds the verify command.
	verifyTimeout = 30 * time.Second
	// maxOutput bounds what is kept of a step's output.
	maxOutput = 64 * 1024
	// tailLines is how much of a failure's output the summary carries.
	tailLines = 6
)

// Install is the trust boundary. In order, and each refusal runs nothing:
//
//  1. An unknown tool is refused: nothing is composed from a name.
//  2. A platform without a step is refused.
//  3. CI never installs, and is not asked.
//  4. The caller's Confirm must return yes; nil is a no.
//  5. The step's program must resolve on PATH to an absolute path outside the
//     guarded tree, lexically and after symlinks.
//
// The step then runs as an argv (no shell), with stdin closed, in its own
// process group, bounded in time; the verify command follows under the same
// rules.
func (in *Installer) Install(name string, capability Capability, confirm Confirm) Result {
	e := explainFor(name, capability, in.GOOS)
	r := Result{Tool: name, OnDecline: e.OnDecline}
	if !e.Known {
		r.Declined = true
		r.Why = "abcd's tool registry has no entry for " + name + ", so it holds no install step it trusts; " +
			"it knows " + strings.Join(Names(), ", ")
		return r
	}
	if len(e.Step) == 0 {
		r.Why = "abcd's registry holds no install step for " + in.GOOS + "; see " + e.Homepage
		return r
	}
	r.step = strings.Join(e.Step, " ")
	r.verify = strings.Join(e.Verify, " ")
	if ci := in.Getenv("CI"); ci != "" {
		r.Declined = true
		r.Why = "CI is set: abcd never installs a tool in CI"
		return r
	}
	if confirm == nil {
		r.Declined = true
		r.Why = "no one was asked"
		return r
	}
	ans := confirm(e)
	if !ans.Yes {
		r.Declined = true
		r.Why = ans.Why
		if r.Why == "" {
			r.Why = "the answer was no"
		}
		return r
	}
	prog, err := in.admit(e.Step[0])
	if err != nil {
		if errors.Is(err, errNotFound) {
			r.Why = e.StepManager + " (" + e.Step[0] + ") is not on PATH, so the step cannot run; install " +
				e.StepManager + " first, or install " + name + " by the route " + e.Homepage + " gives"
		} else {
			r.Why = err.Error()
		}
		return r
	}
	argv := append([]string{prog}, e.Step[1:]...)
	r.Ran = true
	r.Argv = argv
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	out, err := in.Run(ctx, argv)
	cancel()
	if err != nil {
		r.Why = err.Error()
		r.Output = tail(out)
		return r
	}
	r.Installed = true
	vprog, err := in.admit(e.Verify[0])
	if err != nil {
		r.Why = err.Error()
		return r
	}
	ctx, cancel = context.WithTimeout(context.Background(), verifyTimeout)
	vout, err := in.Run(ctx, append([]string{vprog}, e.Verify[1:]...))
	cancel()
	if err != nil {
		r.Why = err.Error()
		r.Output = tail(vout)
		return r
	}
	r.Verified = true
	r.OnDecline = ""
	r.Output = firstLine(vout)
	return r
}

var errNotFound = errors.New("not found on PATH")

// admit resolves a bare program name on PATH and refuses a result inside the
// guarded tree. PATH is the operator's environment, which abcd trusts to locate
// git and gh everywhere else; what a checkout CAN reach is a PATH entry that
// points into it, so the result is judged as the gitleaks adapter judges a
// binary: absolute, and outside the guarded tree both lexically and after
// symlink resolution. An empty guard is refused rather than trusted.
func (in *Installer) admit(name string) (string, error) {
	p, err := in.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, errNotFound)
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("%s resolves to %q, which is not an absolute path; nothing was run", name, p)
	}
	if in.Guard == "" || !filepath.IsAbs(in.Guard) {
		return "", fmt.Errorf("%s cannot be judged without an absolute directory to hold it outside; nothing was run", name)
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", fmt.Errorf("%s resolves to %q, which does not resolve; nothing was run", name, p)
	}
	guards := []string{filepath.Clean(in.Guard)}
	if g, err := filepath.EvalSymlinks(in.Guard); err == nil {
		guards = append(guards, g)
	}
	fold := fsutil.CaseFoldingFS()
	for _, g := range guards {
		for _, c := range []string{filepath.Clean(p), resolved} {
			if fsutil.PathWithin(c, g, fold) {
				return "", fmt.Errorf("%s resolves to %q, inside the repository this verb ran from; "+
					"a program there is repository content and is never run as an install step", name, p)
			}
		}
	}
	return resolved, nil
}

// runArgv is the production runner: the argv is executed directly (no shell),
// stdin is the null device so a package manager cannot stop to ask, the working
// directory is the temporary directory rather than the repository, and the
// child leads its own process group so a timeout kills everything it started
// through this handle and nothing else.
func runArgv(ctx context.Context, argv []string) ([]byte, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = os.TempDir()
	cmd.Stdin = nil
	var buf bytes.Buffer
	w := &capped{w: &buf, remaining: maxOutput}
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return buf.Bytes(), err
	case <-ctx.Done():
		// The group is the one this child leads (Setpgid), addressed through
		// the pid this handle holds: never a pattern, never another process.
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return buf.Bytes(), fmt.Errorf("did not finish within the time allowed (%w)", ctx.Err())
	}
}

// capped keeps at most remaining bytes and drops the rest without failing the
// writer, so a chatty package manager is never killed for its verbosity.
type capped struct {
	w         *bytes.Buffer
	remaining int
}

func (c *capped) Write(p []byte) (int, error) {
	n := len(p)
	if c.remaining > 0 {
		k := n
		if k > c.remaining {
			k = c.remaining
		}
		c.w.Write(p[:k])
		c.remaining -= k
	}
	return n, nil
}

// tail is the last few non-blank lines of out, joined on " | ".
func tail(out []byte) string {
	var keep []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			keep = append(keep, l)
		}
	}
	if len(keep) > tailLines {
		keep = keep[len(keep)-tailLines:]
	}
	return strings.Join(keep, " | ")
}

func firstLine(out []byte) string {
	s := strings.TrimSpace(string(out))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}
