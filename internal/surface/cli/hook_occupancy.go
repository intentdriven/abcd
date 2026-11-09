package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// noteOccupancy is the session-start hook's record of the checkout this
// session starts in (iss-2610240859590333), so a record-writing verb can tell
// a main working tree another live session also works in. It never fails the
// hook: a session it cannot record is one the count cannot see, which is said
// on stderr whatever the reason.
func noteOccupancy(cmd *cobra.Command, in hookInput, cwd string) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	top, err := gitutil.Toplevel(cwd)
	if err != nil {
		return // not a checkout: there is no working tree to share
	}
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return
	}
	unseen := func(why string) {
		diagnosticLine(cmd.ErrOrStderr(), "abcd: could not record which checkout this session works in (%s), so a record verb's count of sessions sharing it cannot see this one.",
			termsafe.Sanitize(fsutil.RedactHome(why)))
	}
	tab, err := m.ReadTable()
	if err != nil {
		unseen(err.Error())
		return
	}
	host, err := sessionname.FindHost(tab, m.From)
	if err != nil {
		unseen(err.Error())
		return
	}
	if err := occupancy.Note(m.Home, in.SessionID, top, host, tab); err != nil {
		unseen(err.Error())
	}
}

// forgetOccupancy is the session-end hook's removal of that record: one file
// remove, so it fits the host's short budget for the hook. A session that ends
// without it stops counting when its host process does.
func forgetOccupancy(in hookInput) {
	if in.SessionID == "" || in.AgentID != "" {
		return
	}
	if m, ok := sessionNameMachine(); ok && m.Home != "" {
		_ = occupancy.Forget(m.Home, in.SessionID)
	}
}

// refuseSharedMainCheckout refuses a record write into root when root is a
// main working tree that two or more live sessions started in
// (iss-2610240859590333): a record written there lands under every one of
// them, in the tree, the index and the indexes a peer regenerates. A linked
// worktree is never refused, nor is a checkout one session or none works in,
// so a person working alone is not held up. Exit 3 is abcd's "back off",
// as a held reservation is. A count that cannot be made (no process table on
// this platform) refuses nothing and says so.
func refuseSharedMainCheckout(cmd *cobra.Command, verb, root string) error {
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return nil
	}
	err := occupancy.Check(m.Home, root, m.ReadTable)
	var shared *occupancy.SharedError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &shared):
		return &exitError{Code: 3, Msg: verb + ": " + shared.Error()}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: could not count the sessions working in this checkout (%s), so a shared main checkout is not refused this time\n",
		verb, termsafe.Sanitize(fsutil.RedactHome(err.Error())))
	return nil
}

// stageFilesFromEnv is f reading a line as this hook's walled stage runs it; f as
// it is for any other session.
func stageFilesFromEnv(f *guard.Files) *guard.Files {
	return f.WithStage(os.Getenv(runner.StageHomeEnv), filepath.SplitList(os.Getenv(runner.StageProtectEnv)),
		filepath.SplitList(os.Getenv(runner.StageDeclaredEnv))).WithStageHarness(os.Getenv(runner.StageHarnessEnv))
}

// stageActive reports whether this hook runs for a walled stage, where whatever
// the guard cannot judge is denied rather than let through with a warning.
func stageActive() bool { return os.Getenv(runner.StageWritesEnv) != "" }

// stageUncheckedDeny is that denial.
func stageUncheckedDeny() error {
	return denyCall("Blocked by the abcd guard (unchecked on a drive's stage): the guard could not judge this call, so it does not run. Run instead: the same call with a plain command, or a file named by its full path.")
}

// fileWriteTools are the host tool names that write a file. The hook
// manifest's PreToolUse matcher names exactly these beside the shell and
// question tools (TestGuardHookIsInstalledForBashCalls holds the two
// together), so the guard hook sees every write the host's own tools make.
var fileWriteTools = []string{"Write", "Edit", "MultiEdit", "NotebookEdit"}

// isFileWriteTool reports whether a hook payload's tool writes a file.
func isFileWriteTool(name string) bool {
	for _, f := range fileWriteTools {
		if strings.EqualFold(name, f) {
			return true
		}
	}
	return false
}

// fileWriteFailOpen is the guard's fail-open-loud contract on a file tool's
// call: the write runs, and exit 1 keeps the warning in front of a human.
func fileWriteFailOpen(w io.Writer, format string, a ...any) error {
	if stageActive() {
		return stageUncheckedDeny()
	}
	diagnosticLine(w, "abcd guard: NOT CHECKED — "+format+". This file write runs UNGUARDED.", a...)
	return &exitError{Code: 1}
}

// storeWriteRefusal is the host's deny for a file tool writing the stop or the
// checklist by any of the readings of its path, or nil. The stop and the setting
// are the person's to write, wherever the file tool stands (itd-2611041549471712).
// The checklist is the verbs' to write (spc-2610230805429429 step 5): a mistake
// filter only, since a shell line still reaches the file and the reader holds.
func storeWriteRefusal(tool string, readings ...string) error {
	for _, p := range readings {
		if guard.StopStorePath(p) {
			return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + " into the stop): " + guard.StopStoreRefusal())
		}
	}
	for _, p := range readings {
		if guard.ChecklistStorePath(p) {
			return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + " into the checklist): " + guard.ChecklistStoreRefusal())
		}
	}
	return nil
}

// maxJudgedPath is the longest path, in bytes, the stores are judged by: Linux's
// PATH_MAX, above macOS's 1024. The kernel refuses a longer path as written, so
// the guard refuses it rather than read it. A reading of a path of n parts
// makes up to n passes of up to n parts each, and a chain of links multiplies
// that by the targets it reads; maxReadingSteps and readingDeadline, not this
// limit, bound the total. A var only so a test can lower it: a link on macOS
// holds no target longer than 1023 bytes, so no real link reaches the limit
// there.
var maxJudgedPath = 4096

// maxLinkHops is how many links deep storeReadings follows a chain of dangling
// links, the kernel's own bound on a chain of links (40 on Linux, 32 on macOS).
const maxLinkHops = 40

// maxReadingSteps is how many path parts the readings of one call may hand to
// filepath.EvalSymlinks, in all. It counts path parts, not lookups:
// EvalSymlinks also looks up the parts of each link it meets on the way, which
// it follows under its own bound of 255 links and which are not counted here,
// so a crafted set of links can make few parts cost many lookups. The budget
// refuses a path whose own spelling is too costly to read; readingDeadline is
// what bounds the time. An ordinary path, a few tens of parts ending in a part
// or two that do not exist yet, costs under a few hundred steps across all its
// readings, where a long path missing its tail, read through a chain of links,
// would cost millions. A var only so a test can lower it.
var maxReadingSteps = 1 << 16

// readingDeadline is how long the readings of one call may take, in all. It is
// checked before every EvalSymlinks, Lstat and Readlink the readings make, and
// a call past it is refused, so no lookup starts after it. The one call in
// flight when it passes can run well past it: one EvalSymlinks follows up to
// 255 links and looks up every part of each target, which a crafted set of
// links makes seconds on macOS and tens of seconds on Linux, by estimate. The
// host waits ten minutes for the hook (hooks_timeout_test.go). An ordinary path
// takes microseconds to read, so the deadline is set well past what a slow,
// busy machine needs, and its refusal says to try again. A var only so a test
// can lower it.
var readingDeadline = 20 * time.Second

// readingClock is the time the deadline is counted and checked by. A var only so
// a test can move it.
var readingClock = time.Now

// readingBudget is what the readings of one call may still spend: path parts
// handed to EvalSymlinks, and time.
type readingBudget struct {
	steps    int
	deadline time.Time
}

// newReadingBudget is the budget of one call, its deadline counted from now.
func newReadingBudget() *readingBudget {
	return &readingBudget{steps: maxReadingSteps, deadline: readingClock().Add(readingDeadline)}
}

// late is the refusal when the deadline has passed, or "". It is asked before
// every lookup.
func (b *readingBudget) late() string {
	if readingClock().Before(b.deadline) {
		return ""
	}
	return fmt.Sprintf("the machine is too slow or busy to finish reading the path: it takes longer than %s", readingDeadline) +
		storeReadingsTail + "Wait a moment, then try the call again."
}

// unreadablePath is the refusal for a reading longer than maxJudgedPath, or "".
// It is asked before a reading is handed to any lookup, so the system's own
// length limit is never what decides it.
func unreadablePath(p string) string {
	if len(p) <= maxJudgedPath {
		return ""
	}
	return fmt.Sprintf("the guard cannot read the path: a reading of it through its links is longer than %d bytes", maxJudgedPath) +
		storeReadingsTail + "Name the file by a shorter path, through fewer links."
}

// partMissing reports whether a lookup's error says only that a part of the
// path does not exist (ENOENT) or is not a folder (ENOTDIR): the two answers
// that leave the rest of the path for the write to create, or to fail on.
func partMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// evalSymlinksLinkBound is filepath.EvalSymlinks' own error past 255 links,
// the one error it makes that is not the system's.
const evalSymlinksLinkBound = "EvalSymlinks: too many links"

// unreadableError is the refusal for a lookup that failed for any other reason:
// a name too long for the system (ENAMETOOLONG), a loop the system meets
// (ELOOP), a folder it may not search (EACCES), a failed read (EIO), or an
// error the guard does not know. None says where the write lands.
func unreadableError(err error) string {
	cause := "an error the guard does not know"
	var errno syscall.Errno
	if errors.As(err, &errno) {
		cause = errno.Error()
	}
	return "the guard cannot read the path (" + cause + ")" + storeReadingsTail +
		"Name the file by a path the guard can read: shorter, through fewer links, in folders it may search."
}

// spend charges parts path parts about to be handed to EvalSymlinks, and
// answers the refusal when the deadline has passed or the parts run out, or "".
func (b *readingBudget) spend(parts int) string {
	if why := b.late(); why != "" {
		return why
	}
	if b.steps -= parts; b.steps < 0 {
		return fmt.Sprintf("reading the path through its links takes lookups of more than %d path parts in all", maxReadingSteps) +
			storeReadingsTail + "Name the file by a shorter path, through fewer links."
	}
	return ""
}

// storeReadingsTail ends every refusal storeReadings answers.
const storeReadingsTail = ", so the guard cannot tell whether the write reaches the stop or the checklist. "

// storeReadings is every reading of the absolute path p the stop and the
// checklist are judged by, or the reason the guard cannot read them all. Each
// path is read three ways: as written (the store checks clean it); cleaned
// first, then through the links of its leading parts that exist, as a host
// that cleans a path before it opens it reaches; and as the volume resolves it
// as written (volumeReading). When a reading's last part is itself a link,
// which neither resolves when its target does not exist yet, its target,
// joined to the link's own folder when relative, is read the same three ways:
// a write that creates the file follows the link. A target is read once, keyed
// by where the volume resolves it, so a loop of links ends where it closes, as
// the kernel's ELOOP leaves nothing written, and two readings of one link that
// name its target two ways cost one. Each target carries how many links deep
// it is, and a target past maxLinkHops, a path or target longer than
// maxJudgedPath, readings handing EvalSymlinks more than the budget's path
// parts, readings outlasting its deadline, and a lookup that fails for any
// reason but a missing part each refuse the write rather than leave a part of
// the chain unjudged: the store check refuses anything it cannot finish
// reading.
//
// Every exit of the store check, and what it answers:
//
//	storeReadings  path over maxJudgedPath            refuse  too long to read
//	storeReadings  link target over maxJudgedPath     refuse  too long to read
//	storeReadings  chain past maxLinkHops             refuse  rest of the chain unread
//	storeReadings  queue empty                        allow   every reading read to its end
//	volumeReading  reading over maxJudgedPath         refuse  before any lookup
//	volumeReading  budget's parts spent, or deadline  refuse  reading unfinished
//	volumeReading  EvalSymlinks resolves              allow   the reading
//	volumeReading  ENOENT, ENOTDIR                    allow   walk up: the write creates or fails on the rest
//	volumeReading  EvalSymlinks' own 255-link bound   allow   walk up: storeReadings follows the chain itself, bounded
//	volumeReading  any other error                    refuse  ENAMETOOLONG, ELOOP, EACCES, EIO, unknown
//	volumeReading  no leading part resolves           allow   the path as written, every error a missing part
//	linkTarget     reading over maxJudgedPath         refuse  before the Lstat
//	linkTarget     deadline, before Lstat or Readlink refuse  lookup would start late
//	linkTarget     Lstat ENOENT, ENOTDIR              allow   not a link: nothing there to follow
//	linkTarget     Lstat or Readlink, any other error refuse  cannot tell whether it is a link
//	linkTarget     not a link                         allow   nothing to follow
//	linkTarget     Readlink ENOENT, or empty target   allow   gone, or a link the system cannot follow
//	linkTarget     a target                           follow  read three ways by storeReadings
//	gate           no file named                      allow   loud: there is no path to read
//	gate           relative, no session directory     allow   loud, after its spelling is judged (exit 1;
//	                                                          a drive's stage denies); pinned by
//	                                                          TestGuardHookFailsOpenLoudOnAnUnreadableFileToolPayload
//	gate           any reading names a store          refuse  the stop or the checklist
func storeReadings(p string, b *readingBudget) ([]string, string) {
	if len(p) > maxJudgedPath {
		return nil, fmt.Sprintf("the path is longer than %d bytes (a relative path counts the session directory before it)", maxJudgedPath) +
			storeReadingsTail + "Name the file by a shorter path."
	}
	vol, why := volumeReading(p, b)
	if why != "" {
		return nil, why
	}
	type link struct {
		path, vol string
		depth     int // links followed to reach path
	}
	queue := []link{{p, vol, 0}}
	seen, queued := map[string]bool{}, map[string]bool{vol: true}
	var out []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		cleaned, why := volumeReading(filepath.Clean(cur.path), b)
		if why != "" {
			return nil, why
		}
		for _, r := range []string{cur.path, cleaned, cur.vol} {
			if seen[r] {
				continue
			}
			seen[r] = true
			out = append(out, r)
			t, ok, why := linkTarget(r, b)
			if why != "" {
				return nil, why
			}
			if !ok {
				continue
			}
			named := termsafe.Sanitize(fsutil.RedactHome(r))
			if len(t) > maxJudgedPath {
				return nil, fmt.Sprintf("the target of the link %s is longer than %d bytes", named, maxJudgedPath) +
					storeReadingsTail + "Name the file the link leads to by a shorter path."
			}
			tv, why := volumeReading(t, b)
			if why != "" {
				return nil, why
			}
			if queued[tv] {
				continue
			}
			if cur.depth >= maxLinkHops {
				return nil, fmt.Sprintf("the path ends in a chain of more than %d links, the last read %s", maxLinkHops, named) +
					storeReadingsTail + "Name the file the chain leads to."
			}
			queued[tv] = true
			queue = append(queue, link{t, tv, cur.depth + 1})
		}
	}
	return out, ""
}

// linkTarget is the target of p when p's last part is a link, joined as written
// to the folder holding the link when it is relative, so the readings of the
// result resolve it where the kernel would; false when p is not a link. The
// length and the deadline are asked before each lookup, and a lookup failing
// for any reason but a missing part is the refusal (the table at storeReadings).
func linkTarget(p string, b *readingBudget) (string, bool, string) {
	if why := unreadablePath(p); why != "" {
		return "", false, why
	}
	if why := b.late(); why != "" {
		return "", false, why
	}
	fi, err := os.Lstat(p)
	if err != nil {
		if partMissing(err) {
			return "", false, ""
		}
		return "", false, unreadableError(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return "", false, ""
	}
	if why := b.late(); why != "" {
		return "", false, why
	}
	t, err := os.Readlink(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, ""
		}
		return "", false, unreadableError(err)
	}
	if t == "" {
		return "", false, ""
	}
	if filepath.IsAbs(t) {
		return t, true, ""
	}
	sep := string(filepath.Separator)
	dir := sep
	if i := strings.LastIndex(p, sep); i > 0 {
		dir = p[:i]
	}
	return strings.TrimSuffix(dir, sep) + sep + t, true, ""
}

// volumeReading is an absolute path as the volume resolves it on a write: the
// longest leading part of it, as written and not cleaned, that resolves is read
// through its links (filepath.EvalSymlinks applies a ".." to what it has
// resolved so far), so a ".." after a link climbs out of the link's target,
// where filepath.Abs and path.Clean climb out of the folder holding the link.
// A symlinked parent resolves the same way. The rest is joined on lexically: a
// ".." in it follows a name that does not exist, so the write would fail
// there, and a last part that is a link whose target does not exist is left as
// the link's own name, for storeReadings to follow. A path no leading part of
// which resolves is returned as it stands. Each pass re-reads the leading part
// and pays its parts out of b first; the refusal, with nothing read past that
// pass, when the parts run out or the deadline has passed. Only a missing part,
// or EvalSymlinks' own bound on links, walks up a part; any other error is the
// refusal (the table at storeReadings). Given a cleaned absolute path it
// answers what fsutil.RealExistingPath does, with the cost counted and every
// error it cannot read past refused.
func volumeReading(p string, b *readingBudget) (string, string) {
	if why := unreadablePath(p); why != "" {
		return "", why
	}
	sep := string(filepath.Separator)
	cur, rest := p, ""
	for {
		if why := b.spend(strings.Count(cur, sep)); why != "" {
			return "", why
		}
		real, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(real, rest), ""
		}
		if !partMissing(err) && err.Error() != evalSymlinksLinkBound {
			return "", unreadableError(err)
		}
		i := strings.LastIndex(cur, sep)
		if i <= 0 {
			return p, ""
		}
		rest = filepath.Join(cur[i+1:], rest)
		cur = cur[:i]
	}
}

// fileWriteGate is the guard hook's judgement of a file the host's own tools
// are about to write (iss-2610280051498465): the record verbs refuse a main
// checkout two or more live sessions share, and a file written there through
// Write or Edit lands under all of them just the same. The file's working
// tree is found by walking up to its .git, with no git subprocess, and judged
// by occupancy.Check, so a linked worktree, a checkout one session or none
// works in, and a path in no checkout are allowed silently. A shared one is
// the host's deny with the verbs' own refusal, naming the worktree store. A
// call naming no file, or a count that could not be made, fails open loud. A
// call naming a file by both file_path and notebook_path is judged on each: a
// deny on either is the answer, and otherwise the first that fails open. The
// readings of both are paid out of one budget, with one deadline for the call.
func fileWriteGate(cmd *cobra.Command, tool, cwd, filePath, notebookPath string) error {
	var paths []string
	for _, p := range []string{filePath, notebookPath} {
		if strings.TrimSpace(p) != "" && (len(paths) == 0 || paths[0] != p) {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return fileWriteFailOpen(cmd.ErrOrStderr(), "the %s call names no file to check", termsafe.Sanitize(tool))
	}
	var first error
	budget := newReadingBudget()
	for _, p := range paths {
		err := fileWritePathGate(cmd, tool, cwd, p, budget)
		var deny *hookDeny
		if errors.As(err, &deny) {
			return err
		}
		if first == nil {
			first = err
		}
	}
	return first
}

// fileWritePathGate is fileWriteGate's judgement of one named file, its
// readings paid out of the call's one budget.
func fileWritePathGate(cmd *cobra.Command, tool, cwd, path string, budget *readingBudget) error {
	if stageActive() {
		if why := guard.StageWriteRefusal(filepath.SplitList(os.Getenv(runner.StageWritesEnv)), filepath.SplitList(os.Getenv(runner.StageDeclaredEnv)), path); why != "" {
			return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + "): " + why)
		}
	}
	// The stores are judged by the path as written first, so a relative one is
	// refused before the fail-open branches below look for a directory to join it to.
	if err := storeWriteRefusal(tool, path); err != nil {
		return err
	}
	if !filepath.IsAbs(path) {
		base := cwd
		if base == "" {
			wd, err := os.Getwd()
			if err != nil {
				return fileWriteFailOpen(cmd.ErrOrStderr(), "the %s call names a relative path and the session directory is unknown", termsafe.Sanitize(tool))
			}
			base = wd
		}
		// Joined as written, not cleaned, so the raw reading of a ".." after
		// a link is still there for CheckPath to judge.
		path = base + string(filepath.Separator) + path
		if !filepath.IsAbs(path) {
			return fileWriteFailOpen(cmd.ErrOrStderr(), "the %s call names a relative path and a session directory that is not absolute", termsafe.Sanitize(tool))
		}
	}
	readings, why := storeReadings(path, budget)
	if why != "" {
		return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + "): " + why)
	}
	if err := storeWriteRefusal(tool, readings...); err != nil {
		return err
	}
	m, ok := sessionNameMachine()
	if !ok || m.Home == "" {
		return nil
	}
	err := occupancy.CheckPath(m.Home, path, m.ReadTable)
	var shared *occupancy.SharedError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &shared):
		return denyCall("Blocked by the abcd guard (" + termsafe.Sanitize(tool) + " into a shared main checkout): " + shared.Error())
	}
	return fileWriteFailOpen(cmd.ErrOrStderr(), "could not count the sessions working in the checkout this file is in (%s)",
		termsafe.Sanitize(fsutil.RedactHome(err.Error())))
}
