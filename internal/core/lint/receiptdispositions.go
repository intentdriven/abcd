package lint

// receiptdispositions.go — the release gate's closed disposition vocabulary
// (the deferred-finding discipline in .abcd/development/intents/disciplines/).
//
// From release-gate manifest schemaVersion 2, every finding a release receipt
// records is dispositioned with exactly one of three words, compared byte for
// byte: fixed, invalid or deferred. A deferred finding names the open issue
// record it is captured as, `"record": "iss-<id>"`, and no two deferred
// findings across the release's receipts name the same record: a deferral is
// its own capture, never a line in a catch-all. fixed and invalid name no
// record. The record is resolved through the ledger listing record-lint's
// ledger rules read, must be the one file that id names and sit in open/, is
// read guarded on the gate's own os.Root, and must be a record the ledger
// reader (capture.ReadRefusal, registered by every front door) reads.
//
// The gate cannot tell whether the record describes the finding: that is a
// judgement, and the discipline leaves it to the reviewer who disposes it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// closedDispositionsSchemaVersion is the release-gate manifest schemaVersion
// from which a release receipt's findings use the closed vocabulary.
const closedDispositionsSchemaVersion = 2

// The three dispositions a release receipt's finding may carry.
const (
	dispositionFixed    = "fixed"
	dispositionInvalid  = "invalid"
	dispositionDeferred = "deferred"
)

// deferredRecordRe is the shape a deferred finding's record takes: an issue id
// in its canonical spelling, so the id a receipt names is the id the ledger
// keys its file by.
var deferredRecordRe = regexp.MustCompile(`^iss-[1-9][0-9]*$`)

// schemaVersionRe is a manifest schemaVersion written as a positive integer.
var schemaVersionRe = regexp.MustCompile(`^[1-9][0-9]*$`)

// manifestSchemaVersion reads the release-gate manifest's schemaVersion. A
// repeated key is refused by name rather than read last-wins, because a
// manifest that says 2 to a reader and 1 to the gate would disarm the rule
// unseen; an absent, null or non-integer value is refused too.
func manifestSchemaVersion(data []byte) (int, error) {
	if err := jsonstrict.NoDuplicateKeys(data); err != nil {
		return 0, err
	}
	var m struct {
		SchemaVersion *json.RawMessage `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, err
	}
	if m.SchemaVersion == nil {
		return 0, errors.New("the manifest carries no schemaVersion")
	}
	raw := string(*m.SchemaVersion)
	if !schemaVersionRe.MatchString(raw) {
		return 0, fmt.Errorf("schemaVersion %s is not a positive integer", raw)
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("schemaVersion %s is not a positive integer", raw)
	}
	return v, nil
}

// deferralSite is one deferred finding: the gate whose receipt carries it, the
// receipt's path, and the finding's index in its failing list.
type deferralSite struct {
	gate  string
	rel   string
	index int
}

// deferralCheck judges each finding of the release's receipts against the
// closed vocabulary and the ledger, and remembers every deferred finding so a
// record named twice is refused once, with its count. The ledger is listed on
// the first deferred finding and each record is judged once.
type deferralCheck struct {
	repoRoot  string
	root      *os.Root
	issuesDir string
	listed    bool
	listErr   string
	files     map[string][]issueFile
	verdicts  map[string]string
	sites     map[string][]deferralSite
}

func newDeferralCheck(repoRoot string, root *os.Root, issuesDir string) *deferralCheck {
	return &deferralCheck{
		repoRoot: repoRoot, root: root, issuesDir: issuesDir,
		verdicts: map[string]string{}, sites: map[string][]deferralSite{},
	}
}

// judge returns what is wrong with finding i of gate's receipt at rel, or ""
// when it is a well-formed disposition. The message completes a sentence that
// begins with the receipt and the finding's index.
func (d *deferralCheck) judge(gate, rel string, i int, f ReceiptFinding) string {
	switch f.Disposition {
	case dispositionFixed, dispositionInvalid:
		if f.Record != "" {
			return "is " + f.Disposition + " but names record " + strconv.Quote(f.Record) + "; only a deferred finding names a record"
		}
		return ""
	case dispositionDeferred:
	default:
		return "disposition " + strconv.Quote(f.Disposition) + " is not fixed, invalid or deferred; a release receipt's finding carries exactly one of those three words"
	}
	if f.Record == "" {
		return `is deferred but names no record; a deferred finding names its own open issue record as "record": "iss-<id>"`
	}
	if !deferredRecordRe.MatchString(f.Record) {
		return "record " + strconv.Quote(f.Record) + " is not an issue id: a deferred finding names its record as iss-<digits>, with no leading zero"
	}
	d.sites[f.Record] = append(d.sites[f.Record], deferralSite{gate: gate, rel: rel, index: i})
	if msg := d.record(f.Record); msg != "" {
		return "names record " + f.Record + ", " + msg
	}
	return ""
}

// record judges one deferred record id against the ledger, once.
func (d *deferralCheck) record(id string) string {
	if v, ok := d.verdicts[id]; ok {
		return v
	}
	v := d.recordVerdict(id)
	d.verdicts[id] = v
	return v
}

func (d *deferralCheck) recordVerdict(id string) string {
	if !d.listed {
		d.listed = true
		d.list()
	}
	if d.listErr != "" {
		return d.listErr
	}
	files := d.files[id]
	switch {
	case len(files) == 0:
		return "which no issue record in " + filepath.ToSlash(d.issuesDir) + " carries; a deferred finding is captured as its own record before the release"
	case len(files) > 1:
		rels := make([]string, 0, len(files))
		for _, f := range files {
			rels = append(rels, filepath.ToSlash(repoRel(d.repoRoot, f.abs)))
		}
		return "which " + strconv.Itoa(len(files)) + " files claim (" + strings.Join(rels, ", ") + "); a record id names one record"
	}
	file := files[0]
	if file.status != "open" {
		return "which is in " + file.status + "/, not open/; a deferred finding's record stays open until the finding is fixed"
	}
	rel := repoRel(d.repoRoot, file.abs)
	if reason := symlinkedAncestor(d.root, rel); reason != "" {
		return "whose file cannot be read safely (" + reason + ")"
	}
	content, err := fsutil.ReadGuardedInRoot(d.root, rel, maxRepoFileBytes)
	if err != nil {
		return "whose file cannot be read safely (" + guardedReason(err) + ")"
	}
	if issueReadRefusal == nil {
		return "but no ledger reader is registered to read it (lint.SetIssueReader), so the release gate fails closed"
	}
	if err := issueReadRefusal(string(content), file.status, rel); err != nil {
		return "which the ledger reader refuses: " + err.Error()
	}
	return ""
}

// list reads the ledger listing once. Every failure is a refusal of each
// deferred finding rather than an aborted gate: an armed gate with nothing
// valid to check is never a pass.
func (d *deferralCheck) list() {
	if err := containedRepoPath(d.issuesDir); err != nil {
		d.listErr = "but the issues directory " + quote(d.issuesDir) + " " + err.Error() + ", so the release gate fails closed"
		return
	}
	files, err := listIssueLedger(d.repoRoot, d.issuesDir)
	if err != nil {
		d.listErr = "but the issue ledger cannot be listed (" + bareCause(err) + "), so the release gate fails closed"
		return
	}
	d.files = map[string][]issueFile{}
	for _, f := range files {
		d.files[f.id] = append(d.files[f.id], f)
	}
}

// symlinkedAncestor names the first directory on rel's path inside root that
// is a symlink, or "" when there is none. os.Root admits a link that stays
// inside the root, but no directory of the issue ledger is ever legitimately a
// link, so one is refused rather than followed.
func symlinkedAncestor(root *os.Root, rel string) string {
	dir := filepath.Dir(rel)
	var parts []string
	for dir != "." && dir != string(filepath.Separator) && dir != "" {
		parts = append([]string{dir}, parts...)
		dir = filepath.Dir(dir)
	}
	for _, p := range parts {
		fi, err := root.Lstat(p)
		if err != nil {
			return guardedReason(err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "the directory " + filepath.ToSlash(p) + " is a symlink"
		}
	}
	return ""
}

// sharedRecord is one refusal of a record named by several deferred findings.
type sharedRecord struct{ rel, msg string }

// shared returns one refusal per record that more than one deferred finding
// names, in id order, each naming every finding that names it.
func (d *deferralCheck) shared() []sharedRecord {
	ids := make([]string, 0, len(d.sites))
	for id, sites := range d.sites {
		if len(sites) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]sharedRecord, 0, len(ids))
	for _, id := range ids {
		sites := d.sites[id]
		where := make([]string, 0, len(sites))
		for _, s := range sites {
			where = append(where, "'"+s.gate+"' failing["+strconv.Itoa(s.index)+"]")
		}
		out = append(out, sharedRecord{
			rel: sites[0].rel,
			msg: "deferred record " + id + " is named by " + strconv.Itoa(len(sites)) + " deferred findings (" +
				strings.Join(where, ", ") + "); a deferred finding is its own captured record, so each names a record of its own",
		})
	}
	return out
}
