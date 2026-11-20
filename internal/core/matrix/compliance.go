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
// The check reads the copy as data and never runs anything in it: the tests
// the copy was given are read from the frozen commit in the run's own
// checkout, and the copy's clone is read as plain files through an os.Root,
// so nothing the copy wrote into its git directory (a hook, a filter, a
// configuration) runs, and no link it made reaches outside the clone.

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

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
	// transcript shows.
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
	// cloneDir(Store, Run, Key) and the hidden tests are under it, at
	// hiddenDir.
	Store, Run string
	Key        TrialKey
	Frozen     string
	// Source is the copy's cell's source, which decides how its isolation
	// stands.
	Source Source
	// Transcript is the copy's transcript as its harness wrote it, nil when
	// none was read.
	Transcript []byte
}

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
	isolation := ComplianceCheck{Rule: RuleIsolation, State: StateKept,
		Detail: "abcd launches the copy's runner in the copy's own clone"}
	if in.Source == SourceHost {
		isolation = ComplianceCheck{Rule: RuleIsolation, State: StateAsserted,
			Detail: "the copy ran in the host's harness, which abcd neither launches in the clone nor confines to it, so its isolation is asserted, not proven"}
	}
	return Compliance{Checks: []ComplianceCheck{
		testsKept(in),
		ownCopyOnly(in),
		{Rule: RuleNetworkPolicy, State: StateAsserted,
			Detail: "no runner reports what a copy reached on the network, so the run's network policy is asserted, not examined"},
		isolation,
	}}, nil
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
// is returned as it is; one a fallback excluded stays its cell's failure.
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
		t.Status, t.Reason = StatusUnexamined, c.reason("the compliance check could not examine the copy", StateUnexamined)
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
// result says.
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
		v == StatusUnexamined && t.Status != StatusUnexamined && !fellBack,
		v == "" && (t.Status == StatusUnexamined || t.Status == StatusExcluded && !fellBack):
		return fmt.Errorf("the trial %s is %s, which its compliance result does not make it", t.Key, t.Status)
	}
	return nil
}

// frozenTest is one test file the frozen commit holds, as its tree names it.
type frozenTest struct {
	mode, oid, path string
	size            int64
}

// testsKept examines whether the copy left every test file the frozen commit
// holds as that commit holds it.
func testsKept(in complianceInputs) ComplianceCheck {
	unexamined := func(why string) ComplianceCheck {
		return ComplianceCheck{Rule: RuleTestsKept, State: StateUnexamined, Detail: why}
	}
	tests, newHash, err := frozenTests(in.RepoRoot, in.Frozen)
	if err != nil {
		return unexamined("the frozen commit's tests could not be read from the run's checkout, so the copy's edits to them cannot be examined")
	}
	clone := cloneDir(in.Store, in.Run, in.Key)
	if fi, err := os.Lstat(clone); err != nil || !fi.IsDir() {
		return unexamined("the copy's clone is gone or is no longer a directory, so its edits to the tests cannot be examined")
	}
	root, err := os.OpenRoot(clone)
	if err != nil {
		return unexamined("the copy's clone could not be opened, so its edits to the tests cannot be examined")
	}
	defer root.Close()
	var edited, unread []string
	for _, ft := range tests {
		how, err := ft.compare(root, newHash)
		switch {
		case err != nil:
			unread = append(unread, ft.path)
		case how != "":
			edited = append(edited, ft.path+" ("+how+")")
		}
	}
	if len(edited) > 0 {
		return ComplianceCheck{Rule: RuleTestsKept, State: StateBroken,
			Detail: clipTo(fmt.Sprintf("edited %d test file(s) the frozen commit holds: %s", len(edited), safeList(edited)), maxDetailBytes)}
	}
	if len(unread) > 0 {
		return unexamined(clipTo(fmt.Sprintf("%d test file(s) of the copy could not be read: %s", len(unread), safeList(unread)), maxDetailBytes))
	}
	return ComplianceCheck{Rule: RuleTestsKept, State: StateKept,
		Detail: fmt.Sprintf("edited none of the %d test file(s) the frozen commit holds", len(tests))}
}

// safeList is names as one line of plain text.
func safeList(names []string) string { return termsafe.Sanitize(strings.Join(names, ", ")) }

// frozenTests lists the test files the frozen commit's tree holds, read from
// the run's own checkout, and the hash the checkout names its objects with.
func frozenTests(repoRoot, frozen string) ([]frozenTest, func() hash.Hash, error) {
	format, err := git("-C", repoRoot, "rev-parse", "--show-object-format")
	if err != nil {
		return nil, nil, err
	}
	var newHash func() hash.Hash
	switch format {
	case "sha1":
		newHash = sha1.New
	case "sha256":
		newHash = sha256.New
	default:
		return nil, nil, fmt.Errorf("the checkout names its objects by %q", clip(format))
	}
	out, err := git("-C", repoRoot, "ls-tree", "-r", "-z", "-l", "--full-tree", frozen+"^{tree}")
	if err != nil {
		return nil, nil, err
	}
	var tests []frozenTest
	for _, entry := range strings.Split(out, "\x00") {
		if entry == "" {
			continue
		}
		meta, p, ok := strings.Cut(entry, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 4 {
			return nil, nil, fmt.Errorf("the frozen tree lists an entry git does not write")
		}
		if f[1] != "blob" || !isTestPath(p) {
			continue
		}
		size, err := strconv.ParseInt(f[3], 10, 64)
		if err != nil || size < 0 {
			return nil, nil, fmt.Errorf("the frozen tree lists a blob with no size")
		}
		tests = append(tests, frozenTest{mode: f[0], oid: f[2], path: p, size: size})
	}
	return tests, newHash, nil
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
	f, err := root.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
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

// ownCopyOnly examines the copy's transcript for any path in the run's clone
// store outside its own clone: another copy's clone, the hidden tests, or the
// store itself. The store is looked for as given and as its links resolve,
// with its slashes as written and as JSON may escape them, and a path inside
// the copy's own clone that climbs out of it (a ".." segment) is a read
// outside it. A read through a relative path that names no part of the store
// is not seen; the report lists that among the run's limitations.
func ownCopyOnly(in complianceInputs) ComplianceCheck {
	unexamined := func(why string) ComplianceCheck {
		return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateUnexamined, Detail: why}
	}
	switch {
	case in.Transcript == nil:
		return unexamined("no transcript of the copy was read, so what it read cannot be examined")
	case len(in.Transcript) == 0:
		return unexamined("the copy's transcript is empty, so what it read cannot be examined")
	case len(in.Transcript) > maxTranscriptBytes:
		return unexamined(fmt.Sprintf("the copy's transcript is past %d bytes, so what it read was not examined", maxTranscriptBytes))
	}
	roots := []string{filepath.Clean(in.Store)}
	if real, err := filepath.EvalSymlinks(in.Store); err == nil && real != roots[0] {
		roots = append(roots, real)
	}
	own := []string{in.Run, in.Key.Task, in.Key.Cell, strconv.Itoa(in.Key.Repeat)}
	for _, root := range roots {
		for _, sep := range []string{"/", `\/`} {
			needle := strings.ReplaceAll(filepath.ToSlash(root), "/", sep)
			if seen, ok := outsideOwn(in.Transcript, needle, sep, sep+strings.Join(own, sep)); ok {
				return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateBroken,
					Detail: clipTo("the copy's transcript names the run's clone store outside its own clone: <clone store>"+
						termsafe.Sanitize(seen), maxDetailBytes)}
			}
		}
	}
	return ComplianceCheck{Rule: RuleOwnCopyOnly, State: StateKept,
		Detail: "the copy's transcript names nothing in the run's clone store outside its own clone"}
}

// outsideOwn finds the first place transcript names the store (needle)
// anywhere but inside own, the copy's clone below it, and returns what
// followed the store there.
func outsideOwn(transcript []byte, needle, sep, own string) (string, bool) {
	at := 0
	for {
		i := bytes.Index(transcript[at:], []byte(needle))
		if i < 0 {
			return "", false
		}
		at += i + len(needle)
		after := transcript[at:]
		if len(after) > 0 && nameByte(after[0]) {
			// A longer name that begins like the store's: not the store.
			continue
		}
		tok := pathToken(after)
		if strings.HasPrefix(tok, own) && (len(tok) == len(own) || !nameByte(tok[len(own)])) &&
			!slices.Contains(strings.Split(tok[len(own):], sep), "..") {
			continue
		}
		return tok, true
	}
}

// nameByte reports whether b continues a file name.
func nameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '.' || b == '_' || b == '-'
}

// pathToken is the path at the start of b: up to a space, a quote or any
// other byte a shell or JSON would end a path at, and at most 160 bytes.
func pathToken(b []byte) string {
	n := 0
	for n < len(b) && n < 160 {
		c := b[n]
		if c <= ' ' || c == 0x7f || strings.IndexByte("\"'`,;|&<>()[]{}", c) >= 0 {
			break
		}
		n++
	}
	return string(b[:n])
}
