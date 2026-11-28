package matrix

// hiddenrun.go runs a code copy's hidden tests itself, so a code copy's score
// is abcd's own run rather than the host's word (decision 9,
// spc-2610161949559050 step 10). The tests were placed at the run's start
// outside every clone, under a name made from a digest (hidden.go); here they
// are read back by the content the run record keeps, and run with the test
// command the declaration names against an export of the copy with the tests
// added. The export is a fresh directory in the clone store, outside every
// clone and every working tree, made by reading the copy's files as plain data
// (no git runs in the clone, a link is never followed) and removed once the
// run ends; the copy's clone is never the run's directory and never receives a
// test.
//
// Running the tests executes code a model wrote on the person's machine, so the
// command runs bounded: one process group of its own, led by the command and
// killed whole through the handle the run holds when its deadline passes (or
// when the run is stopped), never by a pattern, and what is left in the group
// once its leader exits is killed too. A process that leaves the group (by
// setsid) is out of the kill's reach. Its environment is an allowlist that
// carries no credential, its standard input is empty, and what it prints is
// read within a bound, its places and credentials withheld, before any of it
// is kept. The bound is the environment, the working directory and the
// process lifetime only, and is no sandbox: the code runs as this account,
// with HOME set and the network reachable, as the builder's own run of it
// already could (dcn-2611251934117439). A run that does not end by its deadline is unexamined, never failed: a
// copy is not scored on a run abcd did not see end. A stop is the run's error,
// never the copy's score.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// AbcdScoreDetail is a code copy's score detail when abcd placed and ran its
// hidden tests, beside hostScoreDetail for a score the host reported. The
// score is the test command's exit status alone, a known limit
// (dcn-2611251934117439): code whose init or TestMain exits 0 before any test
// runs passes.
const AbcdScoreDetail = "abcd placed the hidden tests outside the copy's clone and ran them with the declared test command " +
	"in an export of the copy; the score is that run's exit status"

const (
	// maxHiddenDeadline bounds the deadline a test run is given.
	maxHiddenDeadline = 6 * time.Hour
	// maxTestArgs and maxTestArgBytes bound the test command.
	maxTestArgs     = 64
	maxTestArgBytes = 4 << 10
	// hiddenGrace bounds the wait for a killed group's output to close.
	hiddenGrace = 5 * time.Second
	// maxHiddenOutputRead is how much of the command's output is read; past
	// it the rest is drained unread. maxHiddenOutputKept is how much of the
	// screened output is kept.
	maxHiddenOutputRead = 256 << 10
	maxHiddenOutputKept = 16 << 10
	// The export's bounds: entries listed, bytes copied, and one file's size.
	maxExportEntries   = maxReviewEntries
	maxExportBytes     = 1 << 30
	maxExportFileBytes = 256 << 20
	// exportsDir is the clone store's directory of exports. No run id names
	// it (a run id is mx-...), so it is never inside any copy's clone,
	// cloneDir(store, run, alias), nor inside the hidden tests' directory.
	exportsDir = "exports"
)

// hiddenRunInputs is one run of a code copy's hidden tests.
type hiddenRunInputs struct {
	// RepoRoot is the run's own checkout, whose scanner configuration the
	// output is screened through.
	RepoRoot string
	// Store is the run's clone store; Run and Task name the hidden tests'
	// directory in it (hiddenDir), and Tests are the tests as the run record
	// keeps them.
	Store string
	Run   string
	Task  string
	Tests []ContentRef
	// Clone is the copy's clone, read only to export the copy, and
	// TreeDigest the digest of the copy's tree (copyTreeDigest) recorded when
	// the copy was made: an export whose tree has another digest is not the
	// copy that was made, and is not scored.
	Clone      string
	TreeDigest string
	// Command is the declaration's test command, run with no shell between:
	// its first word is the program, found on the allowlisted PATH.
	Command []string
	// Deadline bounds the export and the run together.
	Deadline time.Duration
}

// hiddenOutcome is abcd's account of one run of a copy's hidden tests.
type hiddenOutcome struct {
	// Status is passed or failed by the command's exit status, or unexamined
	// when the run did not end by its deadline or the copy could not be
	// exported whole.
	Status TrialStatus
	// Reason is one line of plain text naming no place on this machine.
	Reason string
	// Score is nil for an unexamined run: one test command, passed or not.
	Score *Score
	// Output is what the command printed, read within its bound, with every
	// place and credential withheld, made inert for a terminal; empty when
	// the screen could not run clean.
	Output string
}

func (in hiddenRunInputs) validate() error {
	if !filepath.IsAbs(in.RepoRoot) {
		return errors.New("the run's checkout is not named by an absolute path")
	}
	if !filepath.IsAbs(in.Store) {
		return errors.New("the clone store is not named by an absolute path")
	}
	if !filepath.IsAbs(in.Clone) {
		return errors.New("the copy's clone is not named by an absolute path")
	}
	if !sha256Re.MatchString(in.TreeDigest) {
		return errors.New("the copy's tree digest is not a sha256 in hex")
	}
	if len(in.Command) == 0 || len(in.Command) > maxTestArgs || in.Command[0] == "" {
		return fmt.Errorf("a test command is from 1 to %d words, its first the program it runs", maxTestArgs)
	}
	for _, a := range in.Command {
		if len(a) > maxTestArgBytes || strings.ContainsRune(a, 0) {
			return fmt.Errorf("a word of the test command holds a NUL or more than %d bytes", maxTestArgBytes)
		}
	}
	if in.Deadline <= 0 || in.Deadline > maxHiddenDeadline {
		return fmt.Errorf("a test run's deadline is more than nothing and at most %s", maxHiddenDeadline)
	}
	return nil
}

// runHiddenTests runs a code copy's hidden tests (hiddenRunInputs) and says how
// the copy came out. It refuses, with nothing run, inputs that do not
// validate, hidden tests that are not the content the run recorded
// (readHiddenTests), hidden tests that sit inside the copy's clone or the
// clone inside them, and an exports directory inside a working tree. It
// returns an error, never an outcome, when the run is stopped (ctx ends before
// the deadline), when the command cannot be started, and when the export
// cannot be made or removed: none of those says anything of the copy. The
// export is always removed before it returns.
func runHiddenTests(ctx context.Context, in hiddenRunInputs) (out hiddenOutcome, err error) {
	if err := in.validate(); err != nil {
		return hiddenOutcome{}, err
	}
	// An interrupt, a termination or a hangup abcd receives while the run is
	// at work is a stop: the run's group is killed whole, through the handle
	// it holds, before abcd returns, so the copy's code never outlives abcd's
	// own stop. A stop abcd cannot catch is the backstop's (superviseScript).
	ctx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stopSignals()
	tests, err := readHiddenTests(in.Store, in.Run, in.Task, in.Tests)
	if err != nil {
		return hiddenOutcome{}, err
	}
	if err := hiddenApart(in.Clone, hiddenDir(in.Store, in.Run, in.Task)); err != nil {
		return hiddenOutcome{}, err
	}
	work, err := makeExport(in.Store, in.Run)
	if err != nil {
		return hiddenOutcome{}, err
	}
	defer func() {
		if rerr := removeExport(work); rerr != nil && err == nil {
			// The outcome stands on an export abcd could not clear away, which
			// still holds the hidden tests: the copy is not scored on it, and
			// the reason names what is left, under the clone store.
			kept := out.Output
			out = unexamined(fmt.Sprintf("the copy's export could not be removed after its hidden tests ran, and %s is left in the clone store; "+
				"the copy is not scored until it is removed", labelled(work, in.Store, "<clone store>")))
			out.Output = kept
		}
	}()
	tree, tmp := filepath.Join(work, "tree"), filepath.Join(work, "tmp")
	for _, d := range []string{tree, tmp} {
		if err := os.Mkdir(d, 0o700); err != nil {
			return hiddenOutcome{}, errors.New("the copy's export could not be made in the clone store")
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, in.Deadline)
	defer cancel()

	digest, why, err := exportCopy(runCtx, in.Clone, tree)
	if err == nil && why == "" && runCtx.Err() == nil && digest != in.TreeDigest {
		why = fmt.Sprintf("the copy's tree read for its export has digest %s, not the %s recorded when the copy was made, "+
			"so it is not the copy that was made and is not scored", digest[:16], in.TreeDigest[:16])
	}
	if err == nil && why == "" {
		why, err = addHiddenTests(tree, tests)
	}
	switch {
	case ctx.Err() != nil:
		return hiddenOutcome{}, errors.New("the run was stopped before the copy's hidden tests ran; the copy is not scored")
	case runCtx.Err() != nil:
		return unexamined(fmt.Sprintf("the copy was not exported within its test run's deadline of %s", in.Deadline)), nil
	case err != nil:
		return hiddenOutcome{}, err
	case why != "":
		return unexamined(why), nil
	}

	beforeTestStart()
	run, startErr := runTestCommand(runCtx, in.Command, tree, tmp, in.Deadline+hiddenBackstopMargin)
	// Whether the run was cut, and by what, is read before the output is
	// screened, so a screen that takes long never makes a run that ended in
	// time read as killed.
	stopped := run.cut && ctx.Err() != nil
	if startErr != nil && run.cut && !stopped {
		// The deadline passed between the export and the start: the command
		// never ran, so the copy is unexamined, as any run abcd did not see
		// end is.
		return unexamined(fmt.Sprintf("the copy's hidden tests did not start within their deadline of %s", in.Deadline)), nil
	}
	if startErr != nil && !stopped && errors.Is(startErr, errNoBackstop) {
		return hiddenOutcome{}, fmt.Errorf("%w, so no copy is scored by it", startErr)
	}
	if startErr != nil && !stopped {
		return hiddenOutcome{}, fmt.Errorf("the test command %q could not be run, so no copy is scored by it", clip(termsafe.SanitizeLine(filepath.Base(in.Command[0]))))
	}
	if stopped {
		return hiddenOutcome{}, errors.New("the run was stopped while the copy's hidden tests ran; the copy is not scored")
	}
	screenedOut := screenTestOutput(in, work, run.output)
	if run.cut {
		o := unexamined(fmt.Sprintf("the hidden tests did not end within their deadline of %s, and their process group was killed", in.Deadline))
		o.Output = screenedOut
		return o, nil
	}
	runErr := run.err
	var exit *exec.ExitError
	switch {
	case runErr == nil:
		return hiddenOutcome{Status: StatusPassed, Reason: "its hidden tests passed: the test command exited 0",
			Score: &Score{Passed: 1, Total: 1, Detail: AbcdScoreDetail}, Output: screenedOut}, nil
	case errors.As(runErr, &exit) && exit.ExitCode() < 0:
		// The supervisor, which leads the group, ended on a signal: the group
		// was ended, by its backstop or by a process in it, before abcd saw
		// the command end, so the run did not see the copy through.
		o := unexamined("the hidden tests' process group was ended before their command's end was seen, by its backstop or by a process in it")
		o.Output = screenedOut
		return o, nil
	case errors.As(runErr, &exit):
		reason := fmt.Sprintf("its hidden tests failed: the test command exited %d", exit.ExitCode())
		if exit.ExitCode() > 128 {
			reason += ", the status a command a signal ends also gives"
		}
		return hiddenOutcome{Status: StatusFailed, Reason: reason,
			Score: &Score{Passed: 0, Total: 1, Detail: AbcdScoreDetail}, Output: screenedOut}, nil
	}
	return hiddenOutcome{}, errors.New("the test command's end could not be read, so no copy is scored by it")
}

// beforeTestStart is called once the copy is exported and its hidden tests
// added, just before the test command starts: a test stands a delay in it.
var beforeTestStart = func() {}

// unexamined is the outcome of a run abcd could not see through: no score.
func unexamined(why string) hiddenOutcome {
	return hiddenOutcome{Status: StatusUnexamined, Reason: clipTo(termsafe.SanitizeLine(why), maxReasonBytes)}
}

// hiddenApart refuses hidden tests inside the copy's clone or the clone inside
// them, each read through its links.
func hiddenApart(clone, dir string) error {
	rc, err := filepath.EvalSymlinks(clone)
	if err != nil {
		return errors.New("the copy's clone cannot be read")
	}
	rd, err := resolved(dir)
	if err != nil {
		return errors.New("the hidden tests' directory cannot be resolved")
	}
	if within(rd, rc) || within(rc, rd) {
		return errors.New("the hidden tests sit inside the copy's clone, or the clone inside them; they are kept beside the copies, never in one")
	}
	return nil
}

// makeExport makes a fresh directory, private to this account, for one export
// under the clone store's exports directory, refusing one inside a working
// tree or a repository, where a git the tests start would find it.
func makeExport(store, run string) (string, error) {
	base := filepath.Join(store, exportsDir, run)
	if err := outsideWorkingTrees(base); err != nil {
		return "", errors.New("a copy's export is never made inside a working tree or a repository, and the clone store is inside one or could not be told; it lives outside every checkout")
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", errors.New("the clone store's exports directory could not be made")
	}
	if fi, err := os.Lstat(base); err != nil || !fi.IsDir() {
		return "", errors.New("the clone store's exports directory is not a directory")
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errors.New("no name could be drawn for the copy's export")
	}
	dir := filepath.Join(base, "x_"+hex.EncodeToString(b[:]))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", errors.New("the copy's export could not be made in the clone store")
	}
	return dir, nil
}

// exportCopy copies the copy's tree at clone into dst, an empty directory, as
// plain data: every regular file with its executable bit, every directory, and
// every link whose target is relative and climbs nowhere, read through an
// os.Root with every entry judged by Lstat and no link followed. A directory or
// file named .git, at any depth and in any letter case, is not copied. It
// returns why the copy could not be exported whole (a link that is absolute or
// climbs, a FIFO, device or socket, an entry or byte bound passed), which makes
// the run unexamined, and an error for a failure on the export's side and for
// a ctx that ends before the tree is read whole, so no digest is ever given
// for part of a tree.
func exportCopy(ctx context.Context, clone, dst string) (string, string, error) {
	if fi, err := os.Lstat(clone); err != nil || !fi.IsDir() {
		return "", "", errors.New("the copy's clone is not a directory that can be read")
	}
	src, err := os.OpenRoot(clone)
	if err != nil {
		return "", "", errors.New("the copy's clone is not a directory that can be read")
	}
	defer src.Close()
	e := &export{ctx: ctx, src: src}
	if dst != "" {
		out, err := os.OpenRoot(dst)
		if err != nil {
			return "", "", errors.New("the copy's export cannot be written")
		}
		defer out.Close()
		e.dst = out
	}
	dirs := []string{"."}
	for len(dirs) > 0 {
		d := dirs[len(dirs)-1]
		dirs = dirs[:len(dirs)-1]
		sub, why, err := e.dir(d)
		if err != nil || why != "" {
			return "", why, err
		}
		if err := ctx.Err(); err != nil {
			return "", "", fmt.Errorf("the copy's tree was not read whole before the read was stopped or its deadline passed: %w", err)
		}
		dirs = append(dirs, sub...)
	}
	return e.digest(), "", nil
}

// copyTreeDigest is the digest of the copy's tree at clone as an export
// reads it (exportCopy), written nowhere: what the run records when the copy
// is made, so the export its hidden tests later run in is proved to be that
// copy. It returns why the tree cannot be exported whole, as exportCopy does.
func copyTreeDigest(ctx context.Context, clone string) (string, string, error) {
	return exportCopy(ctx, clone, "")
}

// copyTreeTag heads what a copy's tree digest is taken over.
const copyTreeTag = "abcd matrix copy tree 1"

// digest is the sha256 of every entry the export read, sorted: a directory
// by its path, a file by its path, executable bit and content's sha256, a
// link by its path and target, each field ended by a NUL, which no path
// holds.
func (e *export) digest() string {
	slices.Sort(e.entries)
	h := sha256.New()
	h.Write([]byte(copyTreeTag + "\x00"))
	for _, ent := range e.entries {
		h.Write([]byte(ent))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// export is one copy of a copy's tree into its export, or, with no dst, one
// reading of it for its digest alone.
type export struct {
	ctx     context.Context
	src     *os.Root
	dst     *os.Root
	listed  int
	copied  int64
	entries []string
}

// dir copies the entries of the directory d, listed in batches, and returns
// its subdirectories to copy.
func (e *export) dir(d string) ([]string, string, error) {
	f, err := openDirInRoot(e.src, d)
	if err != nil {
		return nil, fmt.Sprintf("the copy's directory %s could not be listed", exportShown(d)), nil
	}
	defer f.Close()
	var sub []string
	for {
		if e.ctx.Err() != nil {
			return nil, "", nil
		}
		ents, err := f.ReadDir(reviewDirBatch)
		for _, de := range ents {
			if e.listed++; e.listed > maxExportEntries {
				return nil, fmt.Sprintf("the copy's tree holds more than %d entries, more than an export copies", maxExportEntries), nil
			}
			p := path.Join(d, de.Name())
			isDir, why, err := e.entry(p, de)
			if err != nil || why != "" {
				return nil, why, err
			}
			if isDir {
				sub = append(sub, p)
			}
		}
		switch {
		case errors.Is(err, io.EOF), err == nil && len(ents) == 0:
			return sub, "", nil
		case err != nil:
			return nil, fmt.Sprintf("the copy's directory %s could not be listed whole", exportShown(d)), nil
		}
	}
}

// entry copies one listed entry at p, and reports whether it is a directory
// to copy.
func (e *export) entry(p string, de fs.DirEntry) (bool, string, error) {
	if strings.EqualFold(path.Base(p), ".git") {
		return false, "", nil
	}
	name := filepath.FromSlash(p)
	t := de.Type()
	switch {
	case t.IsDir():
		e.entries = append(e.entries, "d\x00"+p+"\x00")
		if e.dst != nil {
			if err := e.dst.Mkdir(name, 0o755); err != nil {
				return false, "", errors.New("the copy's export could not be written")
			}
		}
		return true, "", nil
	case t&fs.ModeSymlink != 0:
		target, err := e.src.Readlink(name)
		if err != nil {
			return false, fmt.Sprintf("the copy's link %s could not be read", exportShown(p)), nil
		}
		if !inertLink(target) {
			return false, fmt.Sprintf("the copy's link %s is absolute or climbs, and an export carries only a link that leads down its own tree", exportShown(p)), nil
		}
		e.entries = append(e.entries, "l\x00"+p+"\x00"+target+"\x00")
		if e.dst != nil {
			if err := e.dst.Symlink(target, name); err != nil {
				return false, "", errors.New("the copy's export could not be written")
			}
		}
		return false, "", nil
	case !t.IsRegular():
		return false, fmt.Sprintf("the copy's entry %s is neither a file, a directory nor a link", exportShown(p)), nil
	}
	data, err := fsutil.ReadGuardedInRoot(e.src, name, maxExportFileBytes)
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return false, fmt.Sprintf("the copy's file %s holds more than the %d bytes an export copies of one file", exportShown(p), maxExportFileBytes), nil
	case err != nil:
		return false, fmt.Sprintf("the copy's file %s could not be read as a regular file", exportShown(p)), nil
	}
	if e.copied += int64(len(data)); e.copied > maxExportBytes {
		return false, fmt.Sprintf("the copy's tree holds more than the %d bytes an export copies", maxExportBytes), nil
	}
	mode := os.FileMode(0o644)
	if fi, err := e.src.Lstat(name); err == nil && fi.Mode().Perm()&0o111 != 0 {
		mode = 0o755
	}
	e.entries = append(e.entries, fmt.Sprintf("f\x00%s\x00%o\x00%s\x00", p, mode, sha256Hex(data)))
	if e.dst != nil {
		if err := writeNew(e.dst, name, data, mode); err != nil {
			return false, "", errors.New("the copy's export could not be written")
		}
	}
	return false, "", nil
}

// inertLink reports whether a link's target leads only down the tree it sits
// in: relative, with no .. among its parts. Composed with other such links it
// still resolves beneath the link's own directory, so no chain of them leaves
// the export; one with a .. can, through a link that resolves higher than it
// reads.
func inertLink(target string) bool {
	if target == "" || filepath.IsAbs(target) || strings.HasPrefix(target, "/") || strings.ContainsRune(target, '\\') {
		return false
	}
	return !slices.Contains(strings.Split(target, "/"), "..")
}

// exportShown is a copy's path as a reason shows it: one inert line, bounded.
func exportShown(p string) string { return clip(termsafe.SanitizeLine(p)) }

// writeNew writes data at name in root, creating it and refusing to replace
// anything.
func writeNew(root *os.Root, name string, data []byte, mode os.FileMode) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// addHiddenTests writes the hidden tests into the export at tree, each at its
// own path, over whatever the copy left there: a copy's file at a hidden
// test's path is replaced, so a copy cannot stand its own test in for the
// hidden one. It returns why the copy could not be tested (a hidden test's
// path, or a directory above it, that the copy made a directory or a link),
// which makes the run unexamined, and an error for a failure on the export's
// side.
func addHiddenTests(tree string, tests []hiddenTest) (string, error) {
	root, err := os.OpenRoot(tree)
	if err != nil {
		return "", errors.New("the copy's export cannot be written")
	}
	defer root.Close()
	for _, h := range tests {
		parts := strings.Split(h.Path, "/")
		at := ""
		for _, part := range parts[:len(parts)-1] {
			at = path.Join(at, part)
			fi, err := root.Lstat(filepath.FromSlash(at))
			switch {
			case errors.Is(err, fs.ErrNotExist):
				if err := root.Mkdir(filepath.FromSlash(at), 0o755); err != nil {
					return "", errors.New("the copy's export could not be written")
				}
			case err != nil:
				return "", errors.New("the copy's export could not be read")
			case !fi.IsDir():
				return fmt.Sprintf("the copy made %s, above the hidden test %s, something other than a directory", exportShown(at), h.Path), nil
			}
		}
		name := filepath.FromSlash(h.Path)
		fi, err := root.Lstat(name)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return "", errors.New("the copy's export could not be read")
		case !fi.Mode().IsRegular():
			return fmt.Sprintf("the copy made the hidden test's path %s something other than a file", h.Path), nil
		default:
			if err := root.Remove(name); err != nil {
				return "", errors.New("the copy's export could not be written")
			}
		}
		if err := writeNew(root, name, h.Data, 0o644); err != nil {
			return "", errors.New("the copy's export could not be written")
		}
	}
	return "", nil
}

// testRun is how one run of the test command ended.
type testRun struct {
	// output is at most maxHiddenOutputRead bytes of what it printed on
	// either stream.
	output []byte
	// err is the leader's own end, as Wait reports it.
	err error
	// cut is whether ctx had ended when the leader's end was read: the group
	// was killed for the deadline or a stop, not judged on its exit.
	cut bool
}

// runTestCommand runs argv in dir as one process group of its own, under ctx,
// with testEnv(tmp) for its environment and nothing on its standard input.
// Both streams go to one pipe abcd made, handed to the command as a file, so
// the leader's end is read the moment it exits, whatever still holds the
// pipe: a child left in the group holding its output never keeps the run to
// its deadline. ctx's end kills the whole group through the handle cmd holds;
// once the leader is reaped, what it left in its group is killed, and only
// then is the pipe drained, within hiddenGrace, since a process that left the
// group could hold it open. The group is led by superviseScript, which runs
// argv and ends the group itself, with no word from abcd, once backstop
// passes or once abcd is gone. It returns an error only when the command could
// not be started, or its program not found.
func runTestCommand(ctx context.Context, argv []string, dir, tmp string, backstop time.Duration) (testRun, error) {
	// A bare name is found on abcd's PATH, as exec.Command finds it; a name
	// holding a slash is read against the export, the command's working
	// directory, as exec reads it against Dir. Either is checked here, so a
	// program that is not there is the run's error, never the copy's failure.
	prog := argv[0]
	if strings.Contains(prog, "/") && !filepath.IsAbs(prog) {
		prog = filepath.Join(dir, prog)
	}
	prog, err := exec.LookPath(prog)
	if err != nil {
		return testRun{cut: ctx.Err() != nil}, err
	}
	// The supervisor's arguments are settled before any pipe is made, so a
	// system with no backstop refuses the run with nothing to close.
	sargs, err := superviseArgs(prog, argv[1:], backstop)
	if err != nil {
		return testRun{}, err
	}
	if ctx.Err() != nil {
		return testRun{cut: true}, ctx.Err()
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return testRun{}, err
	}
	// abcd holds alive's write end for as long as it lives and never writes
	// to it: the supervisor reads its end, and its end of file is abcd gone.
	lr, alive, err := os.Pipe()
	if err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return testRun{}, err
	}
	defer alive.Close()
	cmd := exec.CommandContext(ctx, superviseShell, sargs...)
	cmd.Dir = dir
	cmd.Env = testEnv(os.Environ(), tmp)
	cmd.Stdout, cmd.Stderr = pw, pw
	cmd.ExtraFiles = []*os.File{lr}
	procgroup.Bound(cmd, hiddenGrace)
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		_ = lr.Close()
		return testRun{cut: ctx.Err() != nil}, err
	}
	_ = pw.Close()
	_ = lr.Close()
	out := &boundedBuffer{limit: maxHiddenOutputRead}
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		_, _ = io.Copy(out, pr)
	}()
	werr := cmd.Wait()
	cut := ctx.Err() != nil
	procgroup.KillLeft(cmd)
	select {
	case <-drained:
	case <-time.After(hiddenGrace):
		// A process outside the group holds the pipe: what it has not
		// written by now is not read.
	}
	_ = pr.Close()
	<-drained
	return testRun{output: out.buf.Bytes(), err: werr, cut: cut}, nil
}

// superviseShell runs superviseScript; it is the system's own shell on every
// system abcd targets.
const superviseShell = "/bin/sh"

// hiddenBackstopMargin is how long past a run's deadline its backstop waits
// before it ends the group itself: long enough that abcd's own kill at the
// deadline, which says the copy is unexamined, always comes first while abcd
// lives.
const hiddenBackstopMargin = 30 * time.Second

// superviseScript leads a test run's process group and is the run's backstop
// for a stop abcd cannot catch (SIGKILL, a crash, a power cut of abcd alone).
// It starts two watchers in the group, in the background, that end the group
// whole with kill -9 0, which names the group the watcher is in and no other,
// so no number that could be reused is ever signalled: one once the
// backstop's seconds pass, timed by the system's own sleep named by its path,
// the other once abcd's end of the pipe on fd 3 closes, which happens when
// abcd is gone. Neither holds the output pipe or the command's input. It then
// runs the test command in its own foreground, never as a job: a shell
// without job control starts a job with interrupts and quits ignored, and the
// copy's code keeps their default actions. It exits with the command's status
// as the shell reports it and raises no signal on itself, so a status the
// command chooses, a stop signal's among them, is never taken for one; a
// command a signal ended reads as 128 and its number, a failure either way.
// What is left in the group, the watchers among them, abcd then kills
// (procgroup.KillLeft). The supervisor itself ending on a signal means the
// group was ended before abcd saw the command end. A process that leaves the
// group by setsid is out of every kill's reach (dcn-2611251934117439). Why a
// shell supervisor rather than a CPU limit is dcn-2611280309537858.
const superviseScript = `s=$1; z=$2; shift 2
("$z" "$s"; kill -9 0) 3<&- </dev/null >/dev/null 2>&1 &
(read _ <&3; kill -9 0) </dev/null >/dev/null 2>&1 &
exec 3<&-
"$@" </dev/null
exit "$?"`

// backstopSleeps are where the backstop's timer is looked for, in order: the
// system's own sleep, never one found through the PATH the run carries.
var backstopSleeps = []string{"/bin/sleep", "/usr/bin/sleep"}

// errNoBackstop is a run refused because its backstop cannot be set: the
// run's error, naming what is missing, never the test command's.
var errNoBackstop = errors.New("the hidden tests' run has no backstop")

// superviseArgs is superviseShell's arguments for running prog with args
// under superviseScript, its backstop at backstop rounded up to a second. It
// refuses a system with no sleep at a path it knows, since the run would have
// no backstop.
func superviseArgs(prog string, args []string, backstop time.Duration) ([]string, error) {
	sleep := ""
	for _, p := range backstopSleeps {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0 {
			sleep = p
			break
		}
	}
	if sleep == "" {
		return nil, fmt.Errorf("%w: there is no sleep program at %s", errNoBackstop, strings.Join(backstopSleeps, " or "))
	}
	secs := int64((backstop + time.Second - 1) / time.Second)
	return append([]string{"-c", superviseScript, "abcd-hidden-tests", strconv.FormatInt(max(secs, 1), 10), sleep, prog}, args...), nil
}

// testEnvNames are what a test run carries of abcd's environment: what finds
// programs and the person's toolchain and caches, and the locale. Nothing
// else passes, so no credential, token or socket abcd was started with
// reaches the copy's code through its environment. That is the whole of the
// bound on what the code reads: it runs as this account, HOME and the network
// as reachable as to the builder's own run (dcn-2611251934117439).
var testEnvNames = []string{
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TZ", "LANG",
	"LC_ALL", "LC_COLLATE", "LC_CTYPE", "LC_MESSAGES", "LC_MONETARY", "LC_NUMERIC", "LC_TIME",
	"SSL_CERT_FILE", "SSL_CERT_DIR",
	"GOROOT", "GOPATH", "GOCACHE", "GOMODCACHE", "GOFLAGS", "GOTOOLCHAIN", "GOMAXPROCS",
	"GOPRIVATE", "GONOSUMDB", "GONOPROXY", "GOSUMDB",
}

// testProxyNames are carried with any credentials in their URLs taken out, and
// left out when they hold some in a shape that cannot be taken apart.
var testProxyNames = []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "GOPROXY"}

// testEnv is what a test run's environment holds of env, with TMPDIR set to
// tmp, the export's own, so what the tests leave there goes with the export.
func testEnv(env []string, tmp string) []string {
	var out []string
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch {
		case slices.Contains(testProxyNames, k):
			if clean, ok := proxyListClean(v); ok {
				out = append(out, k+"="+clean)
			}
		case slices.Contains(testEnvNames, k):
			out = append(out, kv)
		}
	}
	return append(out, "TMPDIR="+tmp)
}

// proxyListClean is a proxy value, or a list of them a comma or a bar
// separates, with each URL's credentials taken out; ok is false for a value
// holding some in a shape that cannot be taken apart.
func proxyListClean(v string) (string, bool) {
	var b strings.Builder
	start := 0
	for i := 0; i <= len(v); i++ {
		if i < len(v) && v[i] != ',' && v[i] != '|' {
			continue
		}
		entry := v[start:i]
		if strings.Contains(entry, "@") {
			u, err := url.Parse(entry)
			if err != nil || u.User == nil || u.Host == "" || strings.Contains(u.Host, "@") {
				return "", false
			}
			u.User = nil
			entry = u.String()
		}
		b.WriteString(entry)
		if i < len(v) {
			b.WriteByte(v[i])
		}
		start = i + 1
	}
	return b.String(), true
}

// screenTestOutput is what the test command printed made fit to keep: every
// mention of where the copy ran (its export, its clone, the clone store, the
// run's checkout, the account's home, the run's id) withheld, then every
// credential and identity the repository's secret scanner finds sealed, then
// made inert for a terminal and bounded. A scanner that cannot run, runs
// degraded, or leaves a credential standing keeps nothing of the output.
func screenTestOutput(in hiddenRunInputs, work string, output []byte) string {
	if len(output) == 0 {
		return ""
	}
	red := newRedactor(reviewInputs{Dir: in.Clone, Store: in.Store, RepoRoot: in.RepoRoot, Run: in.Run})
	text, _ := red.redact(strings.ToValidUTF8(string(output), "?"))
	text, _ = newRedactor(reviewInputs{Dir: work, Store: in.Store, RepoRoot: in.RepoRoot, Run: in.Run}).redact(text)
	sc, err := scanner.New(in.RepoRoot)
	if err != nil {
		return ""
	}
	if gap := sc.AugmenterGap(); gap != "" {
		return ""
	}
	text, _ = scanner.Redact(text, sc.ScanText(text, "hidden-tests-output"))
	if degraded, _ := sc.Unavailable(); degraded {
		return ""
	}
	for _, fd := range sc.ScanText(text, "hidden-tests-output") {
		if refuses(fd) {
			return ""
		}
	}
	text = scanner.SweepCallerHome(text, scanner.CallerHome())
	return clipTo(termsafe.SanitizeBlock(text), maxHiddenOutputKept)
}

// clearFlags clears the file flags of everything under dir, never following
// a link, on a system with them (chflags -R -P 0), so a file or directory a
// test made immutable (chflags uchg) does not keep its export. The system's
// own chflags runs, bounded as one process group with a fixed environment;
// where there is none, or it fails, the removal that follows says what is
// left.
func clearFlags(dir string) {
	var bin string
	switch runtime.GOOS {
	case "darwin":
		bin = "/usr/bin/chflags"
	case "freebsd", "netbsd", "openbsd", "dragonfly":
		bin = "/bin/chflags"
	default:
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), clearFlagsBound)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-R", "-P", "0", dir)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LC_ALL=C"}
	procgroup.Bound(cmd, time.Second)
	_ = cmd.Run()
}

// clearFlagsBound bounds clearing an export's file flags.
const clearFlagsBound = 30 * time.Second

// removeExport removes a copy's export; a test stands in a removal that fails.
var removeExport = removeTree

// removeTree removes dir, which a test run may have left read-only or
// immutable in part, never following a link: the file flags of everything
// under it are cleared where the system has them (clearFlags), then every
// directory made writable. The flags are cleared a second time once every
// directory is opened, since the first clearing cannot list a directory a
// test closed to everyone (mode 000), nor reach an immutable file under it;
// the directories are opened again after it, for one that was itself
// immutable, whose mode the first opening could not change.
func removeTree(dir string) error {
	open := func() {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				_ = os.Chmod(p, 0o700)
			}
			return nil
		})
	}
	clearFlags(dir)
	open()
	clearFlags(dir)
	open()
	return os.RemoveAll(dir)
}

// hiddenRunDeadline bounds one run of a copy's hidden tests, its export
// included; the declaration names none yet (dcn-2611280309536132).
const hiddenRunDeadline = 30 * time.Minute

// hiddenOutputRel is where what copy key's hidden tests printed is kept in
// run's state, screened (screenTestOutput, then the secret screen again).
func hiddenOutputRel(run string, key TrialKey) string {
	return run + "/hidden-output/" + key.Task + "/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".txt"
}

// digestForHiddenTests is code copy t as abcd reads its tree before its compliance
// check, when abcd runs its hidden tests: t says so (HiddenTestsByAbcd) and
// names its tree's digest (copyTreeDigest), so the tree the check judges and
// the export the tests run in are proved to be one. A tree an export cannot
// read whole leaves the copy unexamined, saying why; a tree that cannot be
// read at all is an error, and the copy is not taken back.
func digestForHiddenTests(clone string, t TrialRecord) (TrialRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), complianceDeadline)
	defer cancel()
	t.HiddenTests = HiddenTestsByAbcd
	digest, why, err := copyTreeDigest(ctx, clone)
	switch {
	case err != nil:
		return t, fmt.Errorf("copy %s's tree could not be read for its hidden tests: %w", t.Key, err)
	case why != "":
		t.Status, t.Reason = StatusUnexamined, clipTo(termsafe.SanitizeLine("its hidden tests cannot run on it: "+why), maxReasonBytes)
		return t, nil
	}
	t.TreeDigest = digest
	return t, nil
}

// scoreByAbcd runs code copy t's hidden tests, placed for task as refs, in an
// export of the copy's clone whose tree is the one t names (digestForHiddenTests), and
// writes t as its trial with abcd's outcome: passed or failed by the test
// command's exit status, or unexamined where abcd did not see the run end.
// What the tests printed is kept beside the trial once it is written. It
// refuses a copy not awaiting its score, and a run that returns an error (a
// stop, a command that could not start) writes nothing: the copy keeps
// waiting, to be handed back again.
func scoreByAbcd(ctx context.Context, repoRoot, labID, store string, task Task, refs []ContentRef, clone string, t TrialRecord) (TrialRecord, error) {
	if t.Status != "" || t.TreeDigest == "" || !t.AbcdRanHiddenTests() {
		return TrialRecord{}, fmt.Errorf("copy %s is not one whose hidden tests abcd runs now", t.Key)
	}
	out, err := runHiddenTests(ctx, hiddenRunInputs{RepoRoot: repoRoot, Store: store, Run: t.Run, Task: task.ID, Tests: refs,
		Clone: clone, TreeDigest: t.TreeDigest, Command: task.TestCommand, Deadline: hiddenRunDeadline})
	if err != nil {
		return TrialRecord{}, fmt.Errorf("copy %s's hidden tests did not run to an outcome, so it is not scored: %w", t.Key, err)
	}
	t.Status, t.Reason, t.Score = out.Status, out.Reason, out.Score
	if err := writeTrial(repoRoot, labID, t); err != nil {
		return TrialRecord{}, err
	}
	if out.Output != "" {
		if err := keepHiddenOutput(repoRoot, labID, t.Run, t.Key, out.Output); err != nil {
			return t, afterWrite(fmt.Errorf("copy %s's trial is written, and what its hidden tests printed is not kept: %w", t.Key, err))
		}
	}
	return t, nil
}

// keepHiddenOutput keeps what copy key's hidden tests printed, passed through
// the secret screen once more, since it is the copy's code's words.
func keepHiddenOutput(repoRoot, labID, run string, key TrialKey, output string) error {
	rel := hiddenOutputRel(run, key)
	data := []byte(output)
	if err := screen(repoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, data}}); err != nil {
		return err
	}
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	return proc.WriteState(rel, data)
}
