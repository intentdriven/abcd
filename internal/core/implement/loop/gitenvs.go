package loop

// gitenvs.go names every environment a drive's git runs in
// (iss-2611131308266006, fix round 2, ruling 2). driveGitCommands is one
// table of every constructor of a git command a drive can run, each with the
// environment it gives git and whether that command runs the repository's
// hooks; TestEveryGitCommandADriveRunsIsInTheTable fails when a constructor a
// drive's entry reaches is not a row, or a row is one it no longer reaches.
// The hold (remoteid.go) iterates the distinct environments of that table
// (driveEnvs): in each it lists the settings, holds the files they include
// and the attributes files git reads there, and, where a command of that
// environment runs hooks, the hooks directory git names there. Where two
// environments name the same file or directory it is held once.

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// gitEnv is one environment a drive's git runs in: its name in words, for a
// refusal and the page; how a read-only git command runs in it (ask), its
// output past limit bytes an error and the error of a git that ran and
// failed carrying git's exit status (exitStatus); and the variables its git
// is given (environ), for what git reads from them.
type gitEnv struct {
	name    string
	ask     func(ctx context.Context, dir string, limit int, args ...string) (string, error)
	environ func() []string
}

// hooks reports whether any command of the table runs the repository's
// hooks in e.
func (e *gitEnv) hooks() bool {
	for _, c := range driveGitCommands {
		if c.env == e && c.hooks {
			return true
		}
	}
	return false
}

// The environments a drive's git runs in.
var (
	// personEnv is the person's own: their home, their global and system
	// settings and their credentials (gitutil.ScrubbedEnv). The landing
	// push and the reservation's pushes and fetches run in it with the
	// repository's hooks on; the ask of origin and the other reads in it
	// run with the hooks pinned off.
	personEnv = &gitEnv{
		name: "the person's own environment (their home and global settings: the landing push, the reservation's pushes and fetches, the ask of origin)",
		ask:  gitutil.NetAskContext,
		environ: func() []string {
			return gitutil.NetEnviron()
		},
	}
	// stageEnv is the environment of the lane's add, commit, merge and
	// reset (gitutil.WorkingTreeCommitCommand, gitutil.LocalGit): in a drive,
	// the stage's home and global settings file (gitutil.SetHookedEnv),
	// with the repository's hooks on.
	stageEnv = &gitEnv{
		name: "the lane's add and commit's environment (in a drive, the stage's home and global settings file, HOME as the drive sets it for them)",
		ask:  workingTreeAsk,
		environ: func() []string {
			return gitutil.WorkingTreeCommitCommand(context.Background(), ".", "version").Env
		},
	}
	// isolatedEnv is the environment of abcd's own reads of refs and
	// objects (gitutil.Run and its kin): global and system settings off,
	// every program-starting setting pinned off, the hooks among them.
	isolatedEnv = &gitEnv{
		name: "abcd's isolated reads (no global or system settings, hooks pinned off)",
		ask:  isolatedAsk,
		environ: func() []string {
			return gitutil.IsolatedEnv()
		},
	}
)

// driveGitCommand is one row of driveGitCommands: a constructor of a git
// command a drive can run (its package-qualified name), the environment it
// gives git, and whether a command it builds runs the repository's hooks.
type driveGitCommand struct {
	name  string
	env   *gitEnv
	hooks bool
}

// driveGitCommands is every constructor of a git command a drive can run,
// with the environment each uses.
var driveGitCommands = []driveGitCommand{
	// The landing push and the reservation's pushes and fetches, with the
	// repository's hooks on (pre-push, reference-transaction).
	{"gitutil.Net", personEnv, true},
	{"gitutil.NetEach", personEnv, true},
	{"gitutil.NetInput", personEnv, true},
	// The ask of origin and other commands in the person's own environment,
	// the hooks pinned off (core.hooksPath=/dev/null or every exec pin).
	{"gitutil.NetContext", personEnv, false},
	{"gitutil.NetAskContext", personEnv, false},
	{"gitutil.RemoteHeadTip", personEnv, false},
	{"gitutil.WorkingTreeCommand", personEnv, false},
	{"gitutil.WorkingTreeCommandContext", personEnv, false},
	{"gitutil.GitAtLeast", personEnv, false},
	{"claim.personalGitContext", personEnv, false},
	{"scanner.configListing", personEnv, false},
	// The lane's add, commit, merge and reset, and the other commands that
	// run the repository's hooks on a working tree abcd makes.
	{"gitutil.WorkingTreeCommitCommand", stageEnv, true},
	{"gitutil.LocalGit", stageEnv, true},
	{"gitutil.RunCommitCommandBounded", stageEnv, true},
	{"gitutil.RunCommitCommandBoundedAsStored", stageEnv, true},
	{"gitutil.BuiltinMergeDriver", stageEnv, false},
	// abcd's own reads of refs, objects and the repository's settings.
	{"gitutil.CatFileBatch", isolatedEnv, false},
	{"gitutil.CheckoutRoot", isolatedEnv, false},
	{"gitutil.CommonDir", isolatedEnv, false},
	{"gitutil.CommonDirContext", isolatedEnv, false},
	{"gitutil.DefaultRef", isolatedEnv, false},
	{"gitutil.DefaultRefContext", isolatedEnv, false},
	{"gitutil.EachAsStoredContext", isolatedEnv, false},
	{"gitutil.EachWorktree", isolatedEnv, false},
	{"gitutil.EmptyTree", isolatedEnv, false},
	{"gitutil.EmptyTreeName", isolatedEnv, false},
	{"gitutil.EmptyTreeNameContext", isolatedEnv, false},
	{"gitutil.IsAncestor", isolatedEnv, false},
	{"gitutil.ListWorktrees", isolatedEnv, false},
	{"gitutil.ListWorktreesContext", isolatedEnv, false},
	{"gitutil.MergeDriverOverrides", isolatedEnv, false},
	{"gitutil.RepoConfig", isolatedEnv, false},
	{"gitutil.ResolveCommit", isolatedEnv, false},
	{"gitutil.RevParseAbsPath", isolatedEnv, false},
	{"gitutil.RootCommit", isolatedEnv, false},
	{"gitutil.RootCommitContext", isolatedEnv, false},
	{"gitutil.Run", isolatedEnv, false},
	{"gitutil.RunCapped", isolatedEnv, false},
	{"gitutil.RunCappedAsStored", isolatedEnv, false},
	{"gitutil.RunCappedBytes", isolatedEnv, false},
	{"gitutil.RunCappedBytesAsStoredContext", isolatedEnv, false},
	{"gitutil.RunCappedBytesContext", isolatedEnv, false},
	{"gitutil.RunCappedContext", isolatedEnv, false},
	{"gitutil.RunCappedExitOne", isolatedEnv, false},
	{"gitutil.RunCappedExitOneAsStored", isolatedEnv, false},
	{"gitutil.RunCappedExitOneContext", isolatedEnv, false},
	{"gitutil.RunLimitedNetContext", isolatedEnv, false},
	{"gitutil.RunWithIndex", isolatedEnv, false},
	{"gitutil.Status", isolatedEnv, false},
	{"gitutil.StreamWithIndex", isolatedEnv, false},
	{"gitutil.Toplevel", isolatedEnv, false},
	{"gitutil.TrackedDefaultBranch", isolatedEnv, false},
	{"gitutil.TrackedDefaultBranchContext", isolatedEnv, false},
	{"gitutil.TrackedFiles", isolatedEnv, false},
	{"gitutil.TrackedInTreeContext", isolatedEnv, false},
	{"gitutil.TreePathTypes", isolatedEnv, false},
}

// driveEnvs are the distinct environments of driveGitCommands, in the
// table's order.
func driveEnvs() []*gitEnv {
	var out []*gitEnv
	seen := map[*gitEnv]bool{}
	for _, c := range driveGitCommands {
		if !seen[c.env] {
			seen[c.env] = true
			out = append(out, c.env)
		}
	}
	return out
}

// heldReadTimeout bounds each read the hold makes, its second ask included
// (gitEnv.read): the first ask has half of what is left after the wait
// between the two, the second the rest.
var heldReadTimeout = originAskTimeout

// heldReadRetryWait is how long a read that did not finish waits before it
// is asked once more (gitEnv.read).
var heldReadRetryWait = 2 * time.Second

// heldCompareTimeout bounds one comparison of what a drive holds with what
// its git would read now, as a whole, every environment's reads included
// (boundedCompare); each read inside it keeps its own bound and its one
// retry, and none runs past it.
//
// It is far below a reservation's renewal margin (a renewal falls due at half
// its lease, an hour into the default two), and above the drain lock's wait
// (lockTimeout, three seconds). A drive's let-go compares before it takes
// that lock, so a difference is met with the lock never taken; but each push
// of its release, inside the lock, asks a comparison of its own, so one runs
// under the lock, usually in about one read's time and at most this bound,
// and another session waiting on the lock for its three seconds can be told
// another drain is moving. The renewal pass holds its own lock (claim's
// renew.lock, waited on for two minutes) across its pushes, one comparison
// per due reservation: two due at once can take two of these bounds, past
// another pass's wait, which then renews nothing that time and leaves the
// renewal to the next pass, well inside the hour's margin.
var heldCompareTimeout = time.Minute

// errCompareUnfinished is a comparison of what a drive holds that did not
// finish within heldCompareTimeout: neither a difference nor none.
var errCompareUnfinished = errors.New("the comparison did not finish")

// boundedCompare runs compare, given its deadline, and returns what it
// returns, or errCompareUnfinished once heldCompareTimeout has passed,
// whatever compare is doing then: its reads stop at the deadline, and a walk
// of a hooks directory that is still going finishes on its own with no one
// waiting. A read compare makes that the deadline cut is errCompareUnfinished
// too.
func boundedCompare[T any](compare func(by time.Time) (T, error)) (T, error) {
	by := time.Now().Add(heldCompareTimeout)
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := compare(by)
		done <- result{v, err}
	}()
	timer := time.NewTimer(heldCompareTimeout)
	defer timer.Stop()
	var zero T
	select {
	case r := <-done:
		if errors.Is(r.err, errCompareUnfinished) {
			return zero, errCompareUnfinished
		}
		return r.v, r.err
	case <-timer.C:
		return zero, errCompareUnfinished
	}
}

// unfinishedReason is what a step stopped by errCompareUnfinished is told.
func unfinishedReason() string {
	return "the comparison of what the drive holds with what its git would read now did not finish within " + heldCompareTimeout.String() +
		" (its reads run side by side, each with a bound of its own inside that one), so what its git would run is taken neither as changed nor as unchanged"
}

// unfinishedRemedy is what the person does about errCompareUnfinished.
const unfinishedRemedy = "look at what makes git slow here (a loaded machine, a home or a hooks directory on a slow disk), then run the step again; nothing was made"

// readFailure is a read the hold depends on that gave no answer the hold
// can take: git failed, did not finish in time, or answered in a form abcd
// does not take. It stops the step (or the drive's start) at the config
// stage, naming the read and why: it is never read as nothing there, nor as
// nothing changed.
type readFailure struct{ what, why string }

func (e *readFailure) Error() string { return e.what + ": " + e.why }

// exitStatus is the exit status of the git a read ran, -1 when there is none.
func exitStatus(err error) int {
	var ec interface{ ExitCode() int }
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return -1
}

// answer is what one git read returned: what it wrote to its output, its
// exit status (0 on success), and, for a status other than 0, what git said
// of it (its error and standard error, the home and any URL credential
// taken out). stderr is git's own standard error as it wrote it, for an
// environment whose ask gives it apart (gitStderr), never shown: a caller
// reads git's first words from it (noRepository).
type answer struct {
	out    string
	code   int
	said   string
	stderr string
}

// gitStderr is the error of an ask that gives git's standard error apart
// from the text it shows (workingTreeAsk): err is what is shown, stderr git's
// own words as written.
type gitStderr struct {
	err    error
	stderr string
}

func (e *gitStderr) Error() string { return e.err.Error() }
func (e *gitStderr) Unwrap() error { return e.err }

// read runs one read-only git command at dir in e and returns its answer.
// A drive makes many reads a lane, and a loaded machine can make one slow,
// so a git that did not finish is asked once more after heldReadRetryWait;
// both asks together stay inside heldReadTimeout, the first given half of
// what is left after the wait, the second the rest. A git that did not
// finish either time, could not start, or answered past limit bytes is a
// readFailure naming what, and the bound each ask had; a git that ran and
// exited non-zero is not, its status returned with what it wrote and said,
// for the caller to judge.
func (e *gitEnv) read(dir, what string, limit int, args ...string) (answer, error) {
	return e.readBy(time.Time{}, dir, what, limit, args...)
}

// readBy is read inside a comparison whose deadline is by (boundedCompare):
// no ask runs past it, and a read it cuts short is errCompareUnfinished,
// never a readFailure of its own. A zero by is read's own bound alone.
func (e *gitEnv) readBy(by time.Time, dir, what string, limit int, args ...string) (answer, error) {
	deadline := time.Now().Add(heldReadTimeout)
	first := (heldReadTimeout - heldReadRetryWait) / 2
	if first <= 0 {
		first = heldReadTimeout
	}
	cutBy := func() bool { return !by.IsZero() && !time.Now().Before(by) }
	// byLimits: the comparison's deadline, not the read's own bound, is what
	// ends this read's asks, so a retry it leaves no room for, or an ask it
	// cuts, is the comparison not finishing.
	byLimits := !by.IsZero() && !by.After(deadline)
	if byLimits {
		deadline = by
	}
	var had []time.Duration
	for try := 1; ; try++ {
		bound := time.Until(deadline)
		if try == 1 && first < bound {
			bound = first
		}
		if bound <= 0 {
			if cutBy() {
				return answer{code: -1}, errCompareUnfinished
			}
			bound = time.Millisecond
		}
		ctx, cancel := context.WithTimeout(context.Background(), bound)
		out, err := e.ask(ctx, dir, limit, args...)
		late := err != nil && ctx.Err() != nil
		cancel()
		if late {
			had = append(had, bound)
			if try == 1 && time.Until(deadline) > heldReadRetryWait {
				time.Sleep(heldReadRetryWait)
				continue
			}
			if cutBy() || byLimits {
				return answer{code: -1}, errCompareUnfinished
			}
			why := "git did not finish in the " + shownBound(had[0]) + " its ask had, so the read is not taken"
			if len(had) == 2 {
				why = "git did not finish in the " + shownBound(had[0]) + " its first ask had, nor in the " + shownBound(had[1]) +
					" its second had, " + heldReadRetryWait.String() + " later, so the read is not taken"
			}
			return answer{code: -1}, &readFailure{what: what + ", in " + e.name, why: why}
		}
		if err == nil {
			return answer{out: out}, nil
		}
		said := fsutil.RedactHome(gitutil.ScrubURLCredentials(err.Error()))
		if code := exitStatus(err); code > 0 {
			a := answer{out: out, code: code, said: said}
			var gs *gitStderr
			if errors.As(err, &gs) {
				a.stderr = gs.stderr
			}
			return a, nil
		}
		return answer{code: -1}, &readFailure{what: what + ", in " + e.name, why: said}
	}
}

// shownBound is d as a refusal names it: to the second from ten seconds,
// to the tenth below.
func shownBound(d time.Duration) string {
	if d >= 10*time.Second {
		return d.Round(time.Second).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

// failed is the readFailure of a read in e that git answered in a form abcd
// does not take (why).
func (e *gitEnv) failed(what, why string) error {
	return &readFailure{what: what + ", in " + e.name, why: why}
}

// exited is the readFailure of a read in e that git answered with a status
// the caller does not take, naming the status and what git said.
func (e *gitEnv) exited(what string, a answer) error {
	why := "git exited " + strconv.Itoa(a.code)
	if a.said != "" {
		why += ": " + a.said
	}
	return e.failed(what, why)
}

// onePath is git's answer out read as one path: a path holding a newline or
// a NUL, or none, is not taken.
func onePath(out string) (string, bool) {
	p := strings.TrimSuffix(out, "\n")
	return p, p != "" && !strings.ContainsAny(p, "\n\x00")
}

// exitOne is a git that ran and exited 1.
type exitOne struct{}

func (exitOne) Error() string { return "exit status 1" }
func (exitOne) ExitCode() int { return 1 }

// isolatedAsk is isolatedEnv's ask (gitutil.RunCappedExitOneContext).
func isolatedAsk(ctx context.Context, dir string, limit int, args ...string) (string, error) {
	out, one, err := gitutil.RunCappedExitOneContext(ctx, dir, limit, args...)
	if err != nil {
		return "", err
	}
	if one {
		return string(out), exitOne{}
	}
	return string(out), nil
}

// lastEnv is the value env gives key, as exec gives it (its last), and
// whether it is set.
func lastEnv(env []string, key string) (string, bool) {
	v, ok := "", false
	for _, kv := range env {
		if k, val, found := strings.Cut(kv, "="); found && k == key {
			v, ok = val, true
		}
	}
	return v, ok
}

// gitBool reads an environment value as git reads a boolean one: ok is
// false for a value git refuses.
func gitBool(v string) (on, ok bool) {
	switch strings.ToLower(v) {
	case "", "false", "no", "off":
		return false, true
	case "true", "yes", "on":
		return true, true
	}
	n, err := strconv.ParseInt(v, 0, 64)
	if err != nil {
		return false, false
	}
	return n != 0, true
}

// attributesFiles are the global and system attributes files git reads at
// dir in e. Each is asked of git (`git var GIT_ATTR_GLOBAL`,
// `GIT_ATTR_SYSTEM`); a git that says there is none (status 1, nothing
// written) holds none. Where git does not know the names (git older than
// 2.42 answers its usage, status 129), the global file is the
// core.attributesFile git expands in e, else git's documented default
// under e's XDG_CONFIG_HOME or home; the system file is none where
// GIT_ATTR_NOSYSTEM is set true in e, else the gitattributes beside the
// system settings file the listing names (system), else none: it is then
// not held. Any other answer is a readFailure.
func attributesFiles(e *gitEnv, dir, system string) ([]string, error) {
	return attributesFrom(e, dir, system, askAttributes(e, dir, time.Time{}), time.Time{})
}

// attributeWhich are the two attributes files git names, global then system.
var attributeWhich = [2]string{"GLOBAL", "SYSTEM"}

// attributeAnswer is one `git var GIT_ATTR_<which>` read: its answer, or
// why there is none.
type attributeAnswer struct {
	a   answer
	err error
}

// askAttributes asks git in e at dir for both attributes files, side by
// side, inside the comparison's deadline by (zero for none).
func askAttributes(e *gitEnv, dir string, by time.Time) [2]attributeAnswer {
	var out [2]attributeAnswer
	var wg sync.WaitGroup
	for i, which := range attributeWhich {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i].a, out[i].err = e.readBy(by, dir, attributeWhat(which), 4096, "var", "GIT_ATTR_"+which)
		}()
	}
	wg.Wait()
	return out
}

// attributeWhat names the read of one attributes file.
func attributeWhat(which string) string {
	return "asking git for its " + strings.ToLower(which) + " attributes file (git var GIT_ATTR_" + which + ")"
}

// attributesFrom is attributesFiles from the answers askAttributes gave.
func attributesFrom(e *gitEnv, dir, system string, asked [2]attributeAnswer, by time.Time) ([]string, error) {
	var out []string
	for i, which := range attributeWhich {
		what := attributeWhat(which)
		a, err := asked[i].a, asked[i].err
		if err != nil {
			return nil, err
		}
		switch {
		case a.code == 0:
			p, ok := onePath(a.out)
			if !ok {
				return nil, e.failed(what, "git did not answer with one path, so the answer is not taken")
			}
			out = append(out, p)
			continue
		case a.code == 1 && a.out == "":
			continue
		case a.code != 129:
			return nil, e.exited(what, a)
		}
		var p string
		if which == "GLOBAL" {
			p, err = globalAttributes(e, dir, by)
		} else {
			p, err = systemAttributes(e, system)
		}
		if err != nil {
			return nil, err
		}
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// globalAttributes is the global attributes file git older than 2.42 reads
// in e, as git finds it: core.attributesFile as a path, else
// $XDG_CONFIG_HOME/git/attributes, else $HOME/.config/git/attributes, else
// none.
func globalAttributes(e *gitEnv, dir string, by time.Time) (string, error) {
	what := "asking git for core.attributesFile as a path (git config --type=path --get core.attributesFile)"
	a, err := e.readBy(by, dir, what, 4096, "config", "--type=path", "--get", "core.attributesFile")
	if err != nil {
		return "", err
	}
	switch {
	case a.code == 0:
		p, ok := onePath(a.out)
		if !ok {
			return "", e.failed(what, "git did not answer with one path, so the answer is not taken")
		}
		return p, nil
	case a.code != 1:
		return "", e.exited(what, a)
	}
	env := e.environ()
	if xdg, _ := lastEnv(env, "XDG_CONFIG_HOME"); xdg != "" {
		return xdg + "/git/attributes", nil
	}
	if home, _ := lastEnv(env, "HOME"); home != "" {
		return home + "/.config/git/attributes", nil
	}
	return "", nil
}

// systemAttributes is the system attributes file git older than 2.42 reads
// in e: none where GIT_ATTR_NOSYSTEM is set true there, else the
// gitattributes beside system, the system settings file the listing names,
// else none.
func systemAttributes(e *gitEnv, system string) (string, error) {
	if v, set := lastEnv(e.environ(), "GIT_ATTR_NOSYSTEM"); set {
		on, ok := gitBool(v)
		if !ok {
			return "", e.failed("reading GIT_ATTR_NOSYSTEM", "it holds a value git does not read as true or false")
		}
		if on {
			return "", nil
		}
	}
	if system == "" {
		return "", nil
	}
	return system[:strings.LastIndexByte(system, '/')+1] + "gitattributes", nil
}
