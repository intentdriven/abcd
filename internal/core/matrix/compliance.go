package matrix

// compliance.go is the check every copy passes before it is ranked
// (spc-2610161949559050 step 4, the technical criterion on compliance; the
// SOTA note's rule that rule-breaking runs are rejected before ranking). A
// copy that edited the tests it was given, or read another copy or the hidden
// tests, broke a rule of the run: it is excluded and named, counted apart and
// never as a pass or a failure (dcn-2610230216139961). A copy the check could
// not examine is marked unexamined and reported apart, never ranked as
// compliant. What the check cannot examine for any copy (the network policy,
// and a host copy's isolation) is recorded as asserted, not proven, so the
// report can say so.
//
// The check reads the copy as data and never runs anything the copy chose:
// the tests the copy was given are read from the frozen commit in the run's
// own checkout, and the copy's clone is read as plain files through an
// os.Root. Where a test file's bytes differ from the frozen commit's, git is
// asked whether it changed from a scratch repository of the check's own,
// which borrows the run's checkout's objects and takes the clone only as its
// work tree; the clone's line-ending settings are read from its configuration
// as values. So nothing the copy wrote into its git directory (a hook, a
// filter, a configuration) runs, and no link it made reaches outside the
// clone.
//
// The whole check runs under one overall deadline (complianceDeadline): every
// git call, the scan of the clone and the transcript scan together, past which
// the rules that read the copy are unexamined and the copy is reported apart,
// so one hung check never stalls the run.

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ComplianceRule is a rule of the run a copy is checked against.
type ComplianceRule string

const (
	// RuleTestsKept: the copy edited no test file the frozen commit holds (it
	// changed none, deleted none and replaced none with a link). Adding tests
	// of its own is no breach.
	RuleTestsKept ComplianceRule = "tests_kept"
	// RuleOwnCopyOnly: the copy read nothing in the run's clone store outside
	// its own clone, so no other copy and none of the hidden tests, as its
	// transcript and the links its clone holds show. The hidden tests and
	// the other clones sit beside the copy under one account, so their modes
	// do not bar it from them: the rule is a heuristic scan for the reads a
	// transcript records, not a barrier (ownCopyOnly).
	RuleOwnCopyOnly ComplianceRule = "own_copy_only"
	// RuleNetworkPolicy: the copy kept the run's network policy. No runner
	// reports what a copy reached on the network, so this is asserted for
	// every copy.
	RuleNetworkPolicy ComplianceRule = "network_policy"
	// RuleIsolation: the copy worked in its own clone. A runner's copy is
	// launched there by abcd; a host copy runs in the host's harness, which
	// abcd neither launches in the clone nor confines to it, so its isolation
	// is asserted, not proven.
	RuleIsolation ComplianceRule = "isolation"
)

// complianceRules is every rule, in the order a result lists them.
var complianceRules = []ComplianceRule{RuleTestsKept, RuleOwnCopyOnly, RuleNetworkPolicy, RuleIsolation}

// ComplianceState is how a rule came out for one copy.
type ComplianceState string

const (
	// StateKept: the check examined the rule and the copy kept it.
	StateKept ComplianceState = "kept"
	// StateBroken: the check examined the rule and the copy broke it, which
	// excludes the copy.
	StateBroken ComplianceState = "broken"
	// StateUnexamined: the check could not examine the rule for this copy,
	// which sets the copy apart as unexamined.
	StateUnexamined ComplianceState = "unexamined"
	// StateAsserted: nothing the check can read shows the rule either way;
	// the copy is ranked on the run's word, and the report lists it among
	// the run's limitations.
	StateAsserted ComplianceState = "asserted"
)

func (s ComplianceState) valid() bool {
	return s == StateKept || s == StateBroken || s == StateUnexamined || s == StateAsserted
}

// ComplianceCheck is one rule's outcome and abcd's account of it: one line of
// plain text naming paths in the copy's tree, never where anything sits on
// this machine.
type ComplianceCheck struct {
	Rule   ComplianceRule  `json:"rule"`
	State  ComplianceState `json:"state"`
	Detail string          `json:"detail"`
}

// Compliance is one copy's compliance result: every rule, in order.
type Compliance struct {
	Checks []ComplianceCheck `json:"checks"`
}

// maxTranscriptBytes bounds the transcript the check reads; a larger one is
// not read, and the rule it bears on is unexamined.
const maxTranscriptBytes = 64 << 20

// maxDetailBytes bounds one rule's detail, so four of them fit a trial's
// reason.
const maxDetailBytes = 240

// complianceInputs is what the check reads for one copy.
type complianceInputs struct {
	// RepoRoot is the run's own checkout, an absolute path, which holds the
	// frozen commit the copy's clone was made from.
	RepoRoot string
	// Store is the run's clone store, an absolute path: the copy's clone is
	// cloneDir(Store, Run, Alias) and the hidden tests are under it, at
	// hiddenDir.
	Store, Run string
	Key        TrialKey
	// Alias is the alias the copy's clone was made under (makeCopyClone).
	Alias  string
	Frozen string
	// Source is the copy's cell's source, which decides how its isolation
	// stands.
	Source Source
	// Transcript is the copy's transcript as its harness wrote it, nil when
	// none was read.
	Transcript []byte
	// State is the run's state, an absolute path, or empty: the repository's
	// lab store (labStore), where the record of which alias is which copy
	// sits beside every other record of the run. A copy whose transcript names
	// a path in it read what would un-blind it.
	State string
	// Within is the whole check's deadline; zero means complianceDeadline.
	Within time.Duration
}

// labStore is where the repository with root commit rootSHA keeps its labs,
// and so every matrix run's state, under the account's abcd home.
func labStore(home, rootSHA string) string { return abcdhome.Path(home, "lab", rootSHA) }

// checkCompliance runs every rule for one copy. It refuses only inputs that
// name no copy; whatever it cannot read of the copy is that rule's
// unexamined state, never an error, so a copy is never ranked for want of an
// answer.
func checkCompliance(in complianceInputs) (Compliance, error) {
	if !runIDRe.MatchString(in.Run) {
		return Compliance{}, fmt.Errorf("%q is not a run id", clip(in.Run))
	}
	if err := in.Key.validate(); err != nil {
		return Compliance{}, err
	}
	if !gitutil.IsFullSHA(in.Frozen) {
		return Compliance{}, fmt.Errorf("%q is not a full commit id, so it is no frozen commit", clip(in.Frozen))
	}
	if !filepath.IsAbs(in.RepoRoot) || !filepath.IsAbs(in.Store) {
		return Compliance{}, errors.New("the compliance check reads the run's checkout and clone store by absolute path")
	}
	if !in.Source.valid() {
		return Compliance{}, fmt.Errorf("%q is not a cell's source", clip(string(in.Source)))
	}
	if !aliasRe.MatchString(in.Alias) {
		return Compliance{}, fmt.Errorf("%q is not a copy's alias", clip(in.Alias))
	}
	if in.State != "" && !filepath.IsAbs(in.State) {
		return Compliance{}, errors.New("the compliance check reads the run's state by absolute path")
	}
	read := complianceRead(in)
	isolation := ComplianceCheck{Rule: RuleIsolation, State: StateKept,
		Detail: "abcd launches the copy's runner in the copy's own clone"}
	if in.Source == SourceHost {
		isolation = ComplianceCheck{Rule: RuleIsolation, State: StateAsserted,
			Detail: "the copy ran in the host's harness, which abcd neither launches in the clone nor confines to it, so its isolation is asserted, not proven"}
	}
	return Compliance{Checks: []ComplianceCheck{
		read[0],
		read[1],
		{Rule: RuleNetworkPolicy, State: StateAsserted,
			Detail: "no runner reports what a copy reached on the network, so the run's network policy is asserted, not examined"},
		isolation,
	}}, nil
}

// complianceDeadline is the whole compliance check's deadline for one copy:
// every git call it makes and the transcript scan together, so one check that
// hangs cannot stall the run. Each git call keeps its own bound
// (checkGitBound, reviewGitTimeout) inside it.
const complianceDeadline = 10 * time.Minute

// complianceRead runs the two rules that read the copy, tests_kept and
// own_copy_only, under the check's one overall deadline (in.Within, or
// complianceDeadline): reading the frozen tree, scanning the clone, every git
// call and the transcript scan. Past the deadline each git is killed, and
// both rules are unexamined, naming the deadline, even where the reading
// finished as it passed, since a reading cut short can answer wrongly; the
// copy is then reported apart, never ranked.
func complianceRead(in complianceInputs) [2]ComplianceCheck {
	within := in.Within
	if within <= 0 {
		within = complianceDeadline
	}
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()
	done := make(chan [2]ComplianceCheck, 1)
	go func() {
		tree, treeErr := readFrozenTree(ctx, in.RepoRoot, in.Frozen)
		scan, scanErr := scanClone(ctx, cloneDir(in.Store, in.Run, in.Alias))
		done <- [2]ComplianceCheck{testsKept(ctx, in, tree, treeErr), ownCopyOnly(ctx, in, tree, treeErr, scan, scanErr)}
	}()
	var got [2]ComplianceCheck
	select {
	case got = <-done:
	case <-ctx.Done():
	}
	if ctx.Err() == nil {
		return got
	}
	late := func(rule ComplianceRule, what string) ComplianceCheck {
		return ComplianceCheck{Rule: rule, State: StateUnexamined,
			Detail: fmt.Sprintf("the compliance check did not finish within %s, so %s was not examined", within, what)}
	}
	return [2]ComplianceCheck{late(RuleTestsKept, "whether the copy edited its tests"), late(RuleOwnCopyOnly, "what the copy read")}
}

// validate holds a result to every rule once, in order, each in a known
// state (isolation kept or asserted, never examined otherwise), with one
// bounded line of detail.
func (c Compliance) validate() error {
	if len(c.Checks) != len(complianceRules) {
		return fmt.Errorf("a compliance result lists %d rules; it lists every one of %d", len(c.Checks), len(complianceRules))
	}
	for i, ch := range c.Checks {
		if ch.Rule != complianceRules[i] {
			return fmt.Errorf("compliance rule %d is %q; want %s", i, clip(string(ch.Rule)), complianceRules[i])
		}
		if !ch.State.valid() || ch.Rule == RuleIsolation && ch.State != StateKept && ch.State != StateAsserted {
			return fmt.Errorf("compliance rule %s has state %q", ch.Rule, clip(string(ch.State)))
		}
		if ch.Detail == "" {
			return fmt.Errorf("compliance rule %s says nothing of why it is %s", ch.Rule, ch.State)
		}
		if err := line("compliance."+string(ch.Rule)+".detail", ch.Detail, maxDetailBytes); err != nil {
			return err
		}
	}
	return nil
}

// verdict is what the result makes of the copy: StatusExcluded when it broke
// a rule, StatusUnexamined when it broke none but one could not be examined,
// and "" when it is ranked as compliant.
func (c Compliance) verdict() TrialStatus {
	var unexamined bool
	for _, ch := range c.Checks {
		switch ch.State {
		case StateBroken:
			return StatusExcluded
		case StateUnexamined:
			unexamined = true
		}
	}
	if unexamined {
		return StatusUnexamined
	}
	return ""
}

// asserted reports whether rule stands on the run's word alone for this copy.
func (c Compliance) asserted(rule ComplianceRule) bool {
	for _, ch := range c.Checks {
		if ch.Rule == rule {
			return ch.State == StateAsserted
		}
	}
	return false
}

// apply records the result on t and gives t the status it makes: a copy that
// broke a rule is excluded and one the check could not examine is
// unexamined, each with a reason naming the rules; a compliant copy keeps the
// status its judging gave. A copy that never started has nothing to check and
// is returned as it is; one a fallback excluded stays its cell's failure, and
// so does a failed copy the check could not examine (a runner that crashed
// leaves no transcript): a failure is never set apart from the score for
// want of an examination, and only a broken rule excludes it.
func (c Compliance) apply(t TrialRecord) TrialRecord {
	if t.Status == StatusNotStarted {
		return t
	}
	c.Checks = slices.Clone(c.Checks)
	t.Compliance = &c
	if t.Status == StatusExcluded && len(t.Fallbacks) > 0 {
		return t
	}
	switch c.verdict() {
	case StatusExcluded:
		t.Status, t.Reason = StatusExcluded, c.reason("the compliance check excluded the copy", StateBroken)
	case StatusUnexamined:
		if t.Status != StatusFailed {
			t.Status, t.Reason = StatusUnexamined, c.reason("the compliance check could not examine the copy", StateUnexamined)
		}
	}
	return t
}

// reason is lead and every rule in state, with its detail, as one line.
func (c Compliance) reason(lead string, state ComplianceState) string {
	var parts []string
	for _, ch := range c.Checks {
		if ch.State == state {
			parts = append(parts, string(ch.Rule)+": "+ch.Detail)
		}
	}
	return clipTo(lead+"; "+strings.Join(parts, "; "), maxReasonBytes)
}

// compliance holds t's status to its compliance result (validate): a ranked copy
// carries one, and a copy is excluded or unexamined by the check only as the
// result says; a failed copy the result could not examine stays failed.
func (t TrialRecord) compliance() error {
	fellBack := t.Status == StatusExcluded && len(t.Fallbacks) > 0
	c := t.Compliance
	if c == nil {
		if t.Status == StatusPassed || t.Status == StatusUnexamined || t.Status == StatusExcluded && !fellBack ||
			t.Status == StatusFailed && t.Score != nil {
			return fmt.Errorf("the trial %s is %s, which only a compliance check that ran first can make it, and carries none", t.Key, t.Status)
		}
		return nil
	}
	if t.Status == StatusNotStarted {
		return fmt.Errorf("the trial %s never started, so it carries no compliance result", t.Key)
	}
	if err := c.validate(); err != nil {
		return fmt.Errorf("the trial %s: %w", t.Key, err)
	}
	switch v := c.verdict(); {
	case v == StatusExcluded && t.Status != StatusExcluded,
		v == StatusUnexamined && t.Status != StatusUnexamined && t.Status != StatusFailed && !fellBack,
		v == "" && (t.Status == StatusUnexamined || t.Status == StatusExcluded && !fellBack):
		return fmt.Errorf("the trial %s is %s, which its compliance result does not make it", t.Key, t.Status)
	}
	return nil
}

// frozenTest is one file the frozen commit holds, as its tree names it.
type frozenTest struct {
	mode, oid, path string
	size            int64
}

// frozenTree is what the package reads of the frozen commit's tree: its test
// files, its attribute files (.gitattributes), the links it holds by their
// blob ids, and the hash the checkout names its objects with, which the
// compliance check reads; and every file it holds by path, which the review's
// assembly reads.
type frozenTree struct {
	tests, attrs []frozenTest
	links        map[string]string
	files        map[string]treeEntry
	newHash      func() hash.Hash
}

// testsKept examines whether the copy left every test file the frozen commit
// holds as that commit holds it, as git reads the clone: a file whose bytes
// match the frozen blob is kept, a missing one is deleted, and one whose
// bytes differ is asked of git (gitChanged), so a checkout that rewrote its
// line endings (an eol attribute, core.autocrlf) is no edit. A copy that also
// changed the attribute files git reads the tests by is held to the bytes.
func testsKept(ctx context.Context, in complianceInputs, tree frozenTree, treeErr error) ComplianceCheck {
	unexamined := func(why string) ComplianceCheck {
		return ComplianceCheck{Rule: RuleTestsKept, State: StateUnexamined, Detail: why}
	}
	if treeErr != nil {
		return unexamined("the frozen commit's tests could not be read from the run's checkout, so the copy's edits to them cannot be examined")
	}
	clone := cloneDir(in.Store, in.Run, in.Alias)
	if fi, err := os.Lstat(clone); err != nil || !fi.IsDir() {
		return unexamined("the copy's clone is gone or is no longer a directory, so its edits to the tests cannot be examined")
	}
	root, err := os.OpenRoot(clone)
	if err != nil {
		return unexamined("the copy's clone could not be opened, so its edits to the tests cannot be examined")
	}
	defer root.Close()
	var edited, unread []string
	var differ []frozenTest
	for _, ft := range tree.tests {
		how, err := ft.compare(root, tree.newHash)
		switch {
		case err != nil:
			unread = append(unread, ft.path)
		case how == "deleted":
			edited = append(edited, ft.path+" ("+how+")")
		case how != "":
			differ = append(differ, ft)
		}
	}
	var viewErr error
	if len(differ) > 0 {
		var viewed bool
		viewed, viewErr = attributesKept(ctx, root, clone, tree)
		var changed map[string]bool
		if viewed && viewErr == nil {
			changed, viewErr = gitChanged(ctx, in.RepoRoot, root, clone, in.Frozen, differ)
		}
		for _, ft := range differ {
			switch {
			case viewErr != nil:
				unread = append(unread, ft.path)
			case !viewed:
				edited = append(edited, ft.path+" (changed)")
			case changed[ft.path]:
				edited = append(edited, ft.path+" (changed)")
			}
		}
	}
	if len(edited) > 0 {
		slices.Sort(edited)
		return ComplianceCheck{Rule: RuleTestsKept, State: StateBroken,
			Detail: clipTo(fmt.Sprintf("edited %d test file(s) the frozen commit holds: %s", len(edited), safeList(edited)), maxDetailBytes)}
	}
	if errors.Is(viewErr, errConfigUnread) {
		return unexamined(fmt.Sprintf("the copy's git configuration is not a regular file of at most %d bytes, so git cannot be asked whether its %d changed test file(s) were edited",
			maxConfigBytes, len(differ)))
	}
	if len(unread) > 0 {
		slices.Sort(unread)
		return unexamined(clipTo(fmt.Sprintf("%d test file(s) of the copy could not be read: %s", len(unread), safeList(unread)), maxDetailBytes))
	}
	return ComplianceCheck{Rule: RuleTestsKept, State: StateKept,
		Detail: fmt.Sprintf("edited none of the %d test file(s) the frozen commit holds", len(tree.tests))}
}

// safeList is names as one line of plain text.
func safeList(names []string) string { return termsafe.Sanitize(strings.Join(names, ", ")) }

// readFrozenTree reads the frozen commit's tree from the run's own checkout,
// once, for every reader of it in the package: the compliance check (its test
// files, attribute files and links) and the review's assembly (every file).
// Each git call runs through reviewGit: under the caller's deadline and
// reviewGitTimeout, the listing within maxFrozenTreeBytes, and a refusal that
// names the subcommand and never git's own words, which can name a path on
// this machine. A gitlink holds no file and is left out; an entry whose object
// is not a full id is refused with the listing.
func readFrozenTree(ctx context.Context, repoRoot, frozen string) (frozenTree, error) {
	out, err := reviewGit(ctx, repoRoot, maxObjectFormatBytes, "rev-parse", "--show-object-format")
	if err != nil {
		return frozenTree{}, fmt.Errorf("the frozen commit's tree cannot be read: %w", err)
	}
	tree := frozenTree{links: map[string]string{}, files: map[string]treeEntry{}}
	switch format := strings.TrimSpace(string(out)); format {
	case "sha1":
		tree.newHash = sha1.New
	case "sha256":
		tree.newHash = sha256.New
	default:
		return frozenTree{}, fmt.Errorf("the checkout names its objects by %q", clip(format))
	}
	out, err = reviewGit(ctx, repoRoot, maxFrozenTreeBytes, "ls-tree", "-r", "-z", "-l", "--full-tree", frozen+"^{tree}")
	if err != nil {
		return frozenTree{}, fmt.Errorf("the frozen commit's tree cannot be read: %w", err)
	}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		meta, raw, ok := bytes.Cut(rec, []byte{'\t'})
		f := strings.Fields(string(meta))
		if !ok || len(f) != 4 {
			return frozenTree{}, errors.New("the frozen commit's tree listing is not in the shape git documents")
		}
		if f[1] != "blob" {
			continue
		}
		size, err := strconv.ParseInt(f[3], 10, 64)
		if err != nil || size < 0 || !gitutil.IsFullSHA(f[2]) {
			return frozenTree{}, errors.New("the frozen commit's tree listing is not in the shape git documents")
		}
		p := string(raw)
		ft := frozenTest{mode: f[0], oid: f[2], path: p, size: size}
		tree.files[p] = treeEntry{mode: ft.mode, oid: ft.oid, size: size}
		if ft.mode == "120000" {
			tree.links[p] = ft.oid
		}
		if path.Base(p) == ".gitattributes" {
			tree.attrs = append(tree.attrs, ft)
		}
		if isTestPath(p) {
			tree.tests = append(tree.tests, ft)
		}
	}
	return tree, nil
}

// compare says how the copy's file at ft's path differs from the frozen
// commit's: "deleted", "changed", or "" for a file left as it was. An error
// is a file the check could not read, which leaves the rule unexamined.
func (ft frozenTest) compare(root *os.Root, newHash func() hash.Hash) (string, error) {
	p := filepath.FromSlash(ft.path)
	fi, err := root.Lstat(p)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return "deleted", nil
	}
	if err != nil {
		return "", err
	}
	if ft.mode == "120000" {
		if fi.Mode()&fs.ModeSymlink == 0 {
			return "changed", nil
		}
		target, err := root.Readlink(p)
		if err != nil {
			return "", err
		}
		if blobID(newHash, []byte(target)) != ft.oid {
			return "changed", nil
		}
		return "", nil
	}
	if !fi.Mode().IsRegular() || fi.Size() != ft.size {
		return "changed", nil
	}
	// Opened without blocking and judged again on the descriptor: a FIFO put
	// in the file's place after the Lstat is never waited on.
	f, err := root.OpenFile(p, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if st, err := f.Stat(); err != nil {
		return "", err
	} else if !st.Mode().IsRegular() || !os.SameFile(fi, st) {
		return "changed", nil
	}
	h := newHash()
	fmt.Fprintf(h, "blob %d\x00", ft.size)
	n, err := io.Copy(h, io.LimitReader(f, ft.size+1))
	if err != nil {
		return "", err
	}
	if n != ft.size || fmt.Sprintf("%x", h.Sum(nil)) != ft.oid {
		return "changed", nil
	}
	return "", nil
}

// attributesKept reports whether the clone's attribute files are the frozen
// commit's, line endings aside (an eol attribute rewrites a .gitattributes
// file's own), and the clone holds no attribute file the frozen commit lacks,
// at any path outside its git directory, in any letter case: only then does
// git read the clone's tests by the frozen commit's attributes. Each file is
// read as a regular file within twice its frozen size (fsutil.ReadGuardedInRoot),
// and the walk reads at most maxScanEntries entries: a clone past that is an
// error, so its tests go unexamined.
func attributesKept(ctx context.Context, root *os.Root, clone string, tree frozenTree) (bool, error) {
	frozen := map[string]bool{}
	for _, a := range tree.attrs {
		frozen[a.path] = true
		data, err := fsutil.ReadGuardedInRoot(root, a.path, 2*a.size+1)
		if err != nil || blobID(tree.newHash, data) != a.oid &&
			blobID(tree.newHash, bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))) != a.oid {
			return false, nil
		}
	}
	kept, n := true, 0
	err := filepath.WalkDir(clone, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if n++; n > maxScanEntries {
			return errScanBound
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" && filepath.Dir(p) == clone {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), ".gitattributes") {
			rel, err := filepath.Rel(clone, p)
			if err != nil || !frozen[filepath.ToSlash(rel)] {
				kept = false
				return filepath.SkipAll
			}
		}
		return nil
	})
	if errors.Is(err, errScanBound) {
		return false, err
	}
	return err == nil && kept, nil
}

// maxPathspecs bounds the paths one git command is handed.
const maxPathspecs = 256

// checkGitBound bounds one git command the compliance check runs: each reads
// the run's checkout or a copy's clone, which the copy could have made large
// or made to wait, so a git past it is stopped and what it was asking is
// unexamined, never answered.
const checkGitBound = 2 * time.Minute

// checkGit is git bounded by checkGitBound and by ctx, its output trimmed.
func checkGit(ctx context.Context, args ...string) (string, error) {
	out, err := gitRawCtx(ctx, checkGitBound, args...)
	return strings.TrimSpace(string(out)), err
}

// gitChanged asks git which of tests changed in the clone from the frozen
// commit. git runs from a scratch repository of the check's own, removed
// after, whose objects are borrowed from the run's checkout and whose work
// tree is the clone: the clone's git directory, which the copy could have
// written, is never git's, so no hook, filter or configuration of the copy's
// runs, and nothing of the run's checkout is written. The scratch repository
// has a fresh configuration, as the clone had when makeClone checked it out
// (isolated from the person's), with the clone's own line-ending settings
// (core.autocrlf, core.eol) read as values from a copy of its configuration
// (lineEndings). Every git command is bounded by checkGitBound.
func gitChanged(ctx context.Context, repoRoot string, root *os.Root, clone, frozen string, tests []frozenTest) (map[string]bool, error) {
	common, err := checkGit(ctx, "-C", repoRoot, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(repoRoot, common)
	}
	tmp, err := os.MkdirTemp("", "abcd-matrix-view-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	endings, err := lineEndings(ctx, root, tmp)
	if err != nil {
		return nil, err
	}
	view := filepath.Join(tmp, "view")
	if _, err := checkGit(ctx, "init", "-q", "--template=", view); err != nil {
		return nil, err
	}
	gitDir := filepath.Join(view, ".git")
	info := filepath.Join(gitDir, "objects", "info")
	if err := os.MkdirAll(info, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(info, "alternates"), []byte(filepath.Join(common, "objects")+"\n"), 0o600); err != nil {
		return nil, err
	}
	base := append(endings, "--literal-pathspecs", "--git-dir="+gitDir, "--work-tree="+clone)
	if _, err := checkGit(ctx, append(slices.Clone(base), "read-tree", frozen)...); err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	for chunk := range slices.Chunk(tests, maxPathspecs) {
		args := append(slices.Clone(base), "diff", "--name-only", "-z", "--no-renames", "--")
		for _, ft := range chunk {
			args = append(args, ft.path)
		}
		out, err := gitRawCtx(ctx, checkGitBound, args...)
		if err != nil {
			return nil, err
		}
		for _, name := range strings.Split(string(out), "\x00") {
			if name != "" {
				changed[name] = true
			}
		}
	}
	return changed, nil
}

// maxConfigBytes bounds the clone's configuration the check reads.
const maxConfigBytes = 64 << 10

// errConfigUnread is a clone whose configuration the check would not read: a
// git directory that is not a directory, or a configuration that is not a
// regular file within maxConfigBytes (a FIFO, a link, an endless device).
var errConfigUnread = errors.New("the copy's git configuration is not a regular file within its bound")

// lineEndings is the clone's own line-ending settings as git options, each
// passed only when it is a value git knows. The copy controls the clone's git
// directory, so git never reads its configuration where it sits: the file is
// read once, through root, as a regular file within maxConfigBytes
// (fsutil.ReadGuardedInRoot), and git is asked its settings from a copy in
// tmp, the check's own scratch directory. A clone with no git directory or no
// configuration has no settings; one whose configuration the check would not
// read is errConfigUnread.
func lineEndings(ctx context.Context, root *os.Root, tmp string) ([]string, error) {
	fi, err := root.Lstat(".git")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !fi.IsDir() {
		return nil, errConfigUnread
	}
	data, err := fsutil.ReadGuardedInRoot(root, ".git/config", maxConfigBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errConfigUnread
	}
	config := filepath.Join(tmp, "clone-config")
	if err := os.WriteFile(config, data, 0o600); err != nil {
		return nil, err
	}
	var opts []string
	for _, setting := range []struct {
		key    string
		values map[string]string
	}{
		{"core.autocrlf", map[string]string{"true": "true", "yes": "true", "on": "true", "1": "true",
			"false": "false", "no": "false", "off": "false", "0": "false", "input": "input"}},
		{"core.eol", map[string]string{"lf": "lf", "crlf": "crlf", "native": "native"}},
	} {
		v, err := checkGit(ctx, "config", "--file", config, "--get", setting.key)
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			// Not set, or a configuration git cannot parse: no setting.
			continue
		}
		if err != nil {
			return nil, err
		}
		if norm, ok := setting.values[strings.ToLower(v)]; ok {
			opts = append(opts, "-c", setting.key+"="+norm)
		}
	}
	return opts, nil
}

// blobID is git's object id of a blob holding data.
func blobID(newHash func() hash.Hash, data []byte) string {
	h := newHash()
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// testDirs are the directory names whose files are tests or the tests' data,
// in the languages a task is commonly written in.
var testDirs = map[string]bool{"test": true, "tests": true, "testdata": true, "__tests__": true,
	"spec": true, "__snapshots__": true, "fixtures": true}

// isTestPath reports whether the slash-separated path p names a test file or
// a test's data by the conventions of the common test runners: a file under a
// test directory, or a file named as Go, Python, JavaScript and TypeScript,
// Ruby, Rust, Java and Kotlin name their tests.
func isTestPath(p string) bool {
	segs := strings.Split(p, "/")
	for _, s := range segs[:len(segs)-1] {
		if testDirs[s] {
			return true
		}
	}
	base := segs[len(segs)-1]
	stem := strings.TrimSuffix(base, path.Ext(base))
	switch {
	case strings.HasSuffix(base, "_test.go"),
		strings.HasSuffix(base, ".py") && (strings.HasPrefix(base, "test_") || strings.HasSuffix(stem, "_test") || base == "conftest.py"),
		strings.Contains(base, ".test.") || strings.Contains(base, ".spec."),
		strings.HasSuffix(base, ".rb") && (strings.HasSuffix(stem, "_spec") || strings.HasSuffix(stem, "_test")),
		strings.HasSuffix(base, ".rs") && strings.HasSuffix(stem, "_test"),
		(strings.HasSuffix(base, ".java") || strings.HasSuffix(base, ".kt")) && (strings.HasSuffix(stem, "Test") || strings.HasSuffix(stem, "Tests")):
		return true
	}
	return false
}

// ownCopyOnly examines whether the copy read anything in the run's clone
// store outside its own clone: another copy's clone, the hidden tests, or the
// store itself. It is a heuristic and not a barrier: the hidden tests and the
// other clones sit beside the copy under the same account, so their modes do
// not keep the copy from them, and the check sees only the reads its
// transcript records and the links its clone holds.
//
// It reads:
//   - every link the clone holds, its git directory included, that leads out
//     of it into the store or above it (one the frozen commit holds as it is
//     is the commit's, not the copy's);
//   - every absolute path the transcript names, spelled as given, through a
//     ~, $HOME or ${HOME} (quoted before the path goes on, "$HOME"/x, or
//     after an option's prefix, -I$HOME/x), with slashes plain or
//     JSON-escaped, and cleaned of
//     doubled slashes, "." segments and ".." climbs, matched in any letter
//     case, against the store as given and as its links resolve;
//   - every relative path with a ".." climb, from $PWD or bare, resolved
//     against the copy's working folder: from its clone's root and from each
//     folder depth inside it, a climb that reaches an existing path in the
//     store outside the clone, and that no folder of the clone would keep
//     inside it, is a read outside.
//
// What it does not see, the report lists among the run's limitations: a read
// through a relative path with no climb after a cd out of the clone, a path a
// program built at run time, and any read the transcript does not record.
func ownCopyOnly(ctx context.Context, in complianceInputs, tree frozenTree, treeErr error, scan cloneScan, scanErr error) ComplianceCheck {
	unexamined := func(why string) ComplianceCheck {
		return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateUnexamined, Detail: why}
	}
	broken := func(why string) ComplianceCheck {
		return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateBroken, Detail: clipTo(why, maxDetailBytes)}
	}
	clone := cloneDir(in.Store, in.Run, in.Alias)
	forms := storeForms(in)
	if scanErr == nil {
		if link, ok := linkOut(clone, forms, scan, tree, treeErr); ok {
			return broken("the copy's clone holds a link leading out of it into the run's clone store: " + termsafe.Sanitize(link))
		}
	}
	switch {
	case in.Transcript == nil:
		return unexamined("no transcript of the copy was read, so what it read cannot be examined")
	case len(in.Transcript) == 0:
		return unexamined("the copy's transcript is empty, so what it read cannot be examined")
	case len(in.Transcript) > maxTranscriptBytes:
		return unexamined(fmt.Sprintf("the copy's transcript is past %d bytes, so what it read was not examined", maxTranscriptBytes))
	}
	sc := &transcriptScan{ctx: ctx, forms: forms, clone: filepath.ToSlash(filepath.Clean(clone)), scan: scan,
		scanned: scanErr == nil, homes: homeForms(), climbs: map[string]string{}}
	if seen, ok := sc.find(in.Transcript); ok {
		return broken(seen)
	}
	if scanErr != nil {
		return unexamined("the copy's clone could not be read, so a climb or a link out of it cannot be examined")
	}
	return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateKept,
		Detail: "the copy's transcript names nothing in the run's clone store outside its own clone, and its clone holds no link out into it; a heuristic scan, not a barrier"}
}

// storeForm is one spelling of a place no copy reads, and of the copy's own
// clone within it ("" where the place holds none): slash-separated, clean and
// lower-cased (lowerASCII). state marks the run's state rather than its clone
// store.
type storeForm struct {
	store, clone string
	state        bool
}

// storeForms is the store, and the run's state where it is given, each as
// given and as its links resolve.
func storeForms(in complianceInputs) []storeForm {
	own := lowerASCII(path.Join(in.Run, in.Alias))
	var forms []storeForm
	add := func(dir string, state bool) {
		s := lowerASCII(path.Clean(filepath.ToSlash(dir)))
		for _, f := range forms {
			if f.store == s {
				return
			}
		}
		f := storeForm{store: s, state: state}
		if !state {
			f.clone = path.Join(s, own)
		}
		forms = append(forms, f)
	}
	for _, place := range []struct {
		dir   string
		state bool
	}{{in.Store, false}, {in.State, true}} {
		if place.dir == "" {
			continue
		}
		add(place.dir, place.state)
		if real, err := filepath.EvalSymlinks(place.dir); err == nil {
			add(real, place.state)
		}
	}
	return forms
}

// outsideOwn reports whether the clean, lower-cased path lp is in f's place
// and outside the copy's own clone.
func (f storeForm) outsideOwn(lp string) bool {
	return under(lp, f.store) && (f.clone == "" || !under(lp, f.clone))
}

// homeForms is the home directory as given and as its links resolve, for a
// path a transcript names through ~ or $HOME.
func homeForms() []string {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return nil
	}
	homes := []string{filepath.ToSlash(filepath.Clean(home))}
	if real, err := filepath.EvalSymlinks(home); err == nil && filepath.ToSlash(real) != homes[0] {
		homes = append(homes, filepath.ToSlash(real))
	}
	return homes
}

// lowerASCII is s with its ASCII letters lower-cased, so it keeps its length
// and an index into it is an index into s.
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// under reports whether the clean slash-separated path p is base or beneath
// it.
func under(p, base string) bool {
	if !strings.HasPrefix(p, base) {
		return false
	}
	return len(p) == len(base) || strings.HasSuffix(base, "/") || p[len(base)] == '/'
}

// maxScanEntries bounds the entries the check reads of a clone; a larger
// clone is not scanned, and the rule it bears on is unexamined.
const maxScanEntries = 1 << 20

// cloneScan is what the check reads of the copy's clone as plain files, links
// not followed.
type cloneScan struct {
	// heights is every folder outside the clone's git directory, by its
	// slash-separated path in the clone ("" its root), and how many folders
	// deep below it the clone goes.
	heights map[string]int
	// parents is every entry's name, lower-cased, and the folders holding one.
	parents map[string][]string
	// links is every link the clone holds, its git directory included.
	links []string
}

var errScanBound = errors.New("the clone holds more entries than the check reads")

// scanClone reads the clone's folders, entries and links.
func scanClone(ctx context.Context, clone string) (cloneScan, error) {
	s := cloneScan{heights: map[string]int{"": 0}, parents: map[string][]string{}}
	if fi, err := os.Lstat(clone); err != nil || !fi.IsDir() {
		return cloneScan{}, fmt.Errorf("the copy's clone is not a directory")
	}
	n := 0
	err := filepath.WalkDir(clone, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if n++; n > maxScanEntries {
			return errScanBound
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(clone, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			s.links = append(s.links, rel)
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git/") {
			return nil
		}
		dir, name := path.Split(rel)
		dir = strings.TrimSuffix(dir, "/")
		s.parents[lowerASCII(name)] = append(s.parents[lowerASCII(name)], dir)
		if d.IsDir() {
			s.heights[rel] = 0
			depth := strings.Count(rel, "/") + 1
			for a := dir; ; a = parentOf(a) {
				if h := depth - folderDepth(a); h > s.heights[a] {
					s.heights[a] = h
				}
				if a == "" {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return cloneScan{}, err
	}
	return s, nil
}

// folderDepth is how many folders deep the clone-relative folder rel is.
func folderDepth(rel string) int {
	if rel == "" {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

// parentOf is the clone-relative folder holding rel.
func parentOf(rel string) string {
	if i := strings.LastIndexByte(rel, '/'); i >= 0 {
		return rel[:i]
	}
	return ""
}

// linkOut finds a link the clone holds that leads out of it into the store,
// or to the store or above it, and returns its path in the clone. A link the
// frozen commit holds with the same target is the commit's, not the copy's.
func linkOut(clone string, forms []storeForm, scan cloneScan, tree frozenTree, treeErr error) (string, bool) {
	for _, rel := range scan.links {
		full := filepath.Join(clone, filepath.FromSlash(rel))
		target, err := os.Readlink(full)
		if err != nil {
			continue
		}
		if oid, ok := tree.links[rel]; treeErr == nil && ok && blobID(tree.newHash, []byte(target)) == oid {
			continue
		}
		t := filepath.ToSlash(target)
		if !path.IsAbs(t) {
			t = path.Join(filepath.ToSlash(filepath.Dir(full)), t)
		}
		cands := []string{path.Clean(t)}
		if real, err := filepath.EvalSymlinks(full); err == nil {
			cands = append(cands, filepath.ToSlash(real))
		}
		for _, c := range cands {
			lc := lowerASCII(c)
			for _, f := range forms {
				if f.outsideOwn(lc) || under(f.store, lc) {
					return rel, true
				}
			}
		}
	}
	return "", false
}

// transcriptScan reads a transcript's paths against the store.
type transcriptScan struct {
	// ctx is the check's deadline: the scan stops reading once it ends.
	ctx   context.Context
	forms []storeForm
	// clone is the copy's clone as given, slash-separated and clean.
	clone   string
	scan    cloneScan
	scanned bool
	homes   []string
	// climbs is each relative climb already judged, cleaned, and what it
	// was found to read ("" for nothing outside).
	climbs map[string]string
}

// pathEnds are the bytes a shell, JSON or a tool's option ends a path at.
const pathEnds = "\"'`,;|&<>()[]{}=:"

// find reads every path the transcript names and returns the first that
// reads the store outside the copy's clone, as that rule's detail. A JSON
// escape ends a path, except an escaped slash, which is a slash.
func (sc *transcriptScan) find(b []byte) (string, bool) {
	var tok []byte
	flush := func() (string, bool) {
		if len(tok) == 0 {
			return "", false
		}
		t := string(tok)
		tok = tok[:0]
		return sc.token(t)
	}
	for i := 0; i < len(b); i++ {
		if i&0xffff == 0 && sc.ctx != nil && sc.ctx.Err() != nil {
			return "", false
		}
		c := b[i]
		switch {
		case (c == '"' || c == '\'') && homeGoesOn(tok, b[i+1:]):
			// A quote closing "$HOME" before the path goes on ("$HOME"/x):
			// the shell joins the two into one word.
		case c == '\\' && i+1 < len(b) && b[i+1] == '"' && homeGoesOn(tok, b[i+2:]):
			// The same, JSON-escaped (\"$HOME\"\/x).
			i++
		case c == '\\' && i+1 < len(b) && b[i+1] == '/':
			tok = append(tok, '/')
			i++
		case c == '\\':
			if seen, ok := flush(); ok {
				return seen, true
			}
			if i+1 < len(b) && b[i+1] == 'u' {
				i += 5
			} else {
				i++
			}
		case c == '$' && i+1 < len(b) && b[i+1] == '{':
			// ${NAME} is one word of the path, braces and all.
			end := bytes.IndexByte(b[i:min(len(b), i+64)], '}')
			if end < 0 {
				tok = append(tok, c)
				continue
			}
			tok = append(tok, b[i:i+end+1]...)
			i += end
		case c <= ' ' || c == 0x7f || strings.IndexByte(pathEnds, c) >= 0:
			if seen, ok := flush(); ok {
				return seen, true
			}
		default:
			tok = append(tok, c)
		}
	}
	return flush()
}

// homeGoesOn reports whether tok ends with $HOME or ${HOME} and next, the
// bytes after a quote that closes it, goes on with a path below it, its slash
// plain or JSON-escaped.
func homeGoesOn(tok, next []byte) bool {
	if !bytes.HasSuffix(tok, []byte("$HOME")) && !bytes.HasSuffix(tok, []byte("${HOME}")) {
		return false
	}
	return bytes.HasPrefix(next, []byte("/")) || bytes.HasPrefix(next, []byte(`\/`))
}

// token judges one path-like token of the transcript.
func (sc *transcriptScan) token(t string) (string, bool) {
	// A sentence's full stop is not part of the path it ends.
	if last := t[strings.LastIndexByte(t, '/')+1:]; strings.HasSuffix(last, ".") && strings.Trim(last, ".") != "" {
		t = strings.TrimRight(t, ".")
	}
	if t == "~" || strings.HasPrefix(t, "~/") {
		return sc.home(t[1:])
	}
	if rest, ok := variable(t, "HOME"); ok {
		return sc.home(rest)
	}
	if rest, ok := variable(t, "PWD"); ok {
		return sc.climb("." + rest)
	}
	if strings.HasPrefix(t, "/") {
		return sc.absolute(t)
	}
	if seen, ok := sc.climb(t); ok {
		return seen, true
	}
	if i := strings.IndexByte(t, '$'); i > 0 {
		// $HOME after an option's prefix (-I$HOME/x).
		if rest, ok := variable(t[i:], "HOME"); ok {
			return sc.home(rest)
		}
	}
	if i := strings.IndexByte(t, '/'); i > 0 {
		// A path after an option's or a scheme's prefix (-I/x, @/x).
		return sc.absolute(t[i:])
	}
	return "", false
}

// variable is the rest of t after a leading $name or ${name}, when that rest
// is empty or a path below it.
func variable(t, name string) (string, bool) {
	for _, v := range []string{"$" + name, "${" + name + "}"} {
		if rest, ok := strings.CutPrefix(t, v); ok && (rest == "" || rest[0] == '/') {
			return rest, true
		}
	}
	return "", false
}

// home judges a path below the home directory.
func (sc *transcriptScan) home(rest string) (string, bool) {
	for _, h := range sc.homes {
		if seen, ok := sc.absolute(h + "/" + rest); ok {
			return seen, true
		}
	}
	return "", false
}

// absolute judges an absolute path: in the store, outside the copy's clone,
// in any letter case.
func (sc *transcriptScan) absolute(p string) (string, bool) {
	p = path.Clean(p)
	lp := lowerASCII(p)
	for _, f := range sc.forms {
		if f.outsideOwn(lp) {
			if f.state {
				return "the copy's transcript names the run's state, where which copy is which is recorded: <run state>" +
					termsafe.Sanitize(p[len(f.store):]), true
			}
			return "the copy's transcript names the run's clone store outside its own clone: <clone store>" +
				termsafe.Sanitize(p[len(f.store):]), true
		}
	}
	return "", false
}

// climb judges a relative path that climbs with "..": a read outside when,
// from the clone's root or a folder inside it as deep as the climb needs, it
// reaches an existing path in the store outside the clone, and from no folder
// of the clone does it reach an existing path inside it (a copy that cd'd
// into widget/ and read ../README.md stayed in its clone).
func (sc *transcriptScan) climb(t string) (string, bool) {
	if !sc.scanned || !strings.Contains(t, "..") {
		return "", false
	}
	c := path.Clean(t)
	if c != ".." && !strings.HasPrefix(c, "../") {
		return "", false
	}
	if seen, ok := sc.climbs[c]; ok {
		return seen, seen != ""
	}
	k, rest := 0, c
	for rest == ".." || strings.HasPrefix(rest, "../") {
		k++
		rest = strings.TrimPrefix(strings.TrimPrefix(rest, ".."), "/")
	}
	seen := ""
	if sc.outside(k, rest) && !sc.inside(k, rest) {
		seen = "the copy's transcript climbs out of its own clone into the run's clone store: " + termsafe.Sanitize(t)
	}
	sc.climbs[c] = seen
	return seen, seen != ""
}

// outside reports whether k climbs then rest, from the clone's root or a
// folder of it fewer than k deep, reach an existing path in the store
// outside the clone.
func (sc *transcriptScan) outside(k int, rest string) bool {
	base := sc.clone
	for j := 1; j <= k; j++ {
		base = path.Dir(base)
		if k-j > sc.scan.heights[""] {
			continue
		}
		p := path.Join(base, rest)
		lp := lowerASCII(p)
		for _, f := range sc.forms {
			if f.outsideOwn(lp) {
				if _, err := os.Lstat(filepath.FromSlash(p)); err == nil {
					return true
				}
			}
		}
	}
	return false
}

// inside reports whether k climbs then rest, from some folder of the clone at
// least k deep, reach an existing path inside the clone.
func (sc *transcriptScan) inside(k int, rest string) bool {
	if rest == "" {
		return sc.scan.heights[""] >= k
	}
	first, _, _ := strings.Cut(rest, "/")
	for _, a := range sc.scan.parents[lowerASCII(first)] {
		if sc.scan.heights[a] < k {
			continue
		}
		if _, err := os.Lstat(filepath.FromSlash(path.Join(sc.clone, a, rest))); err == nil {
			return true
		}
	}
	return false
}
