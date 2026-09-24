package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The drift kinds IssueDrift reports. Each names one way the promote join
// (itd-4 AC3) can stop reading the same from both ends.
const (
	// DriftOneSided: an intent names a ledger record in related_issues that does
	// not name the intent back in related_intents — or a reading item names an
	// intent that does not name it back. An ISSUE naming an intent that does not
	// name it back is not drift: an issue may be captured already related to an
	// intent it was never promoted into, and the reading item carries no such
	// loose relation, which is why the two ends are judged differently.
	DriftOneSided = "one_sided"
	// DriftDangling: either end names a record this tree does not hold.
	DriftDangling = "dangling"
	// DriftShippedUnresolved: a shipped intent names an issue that is not in
	// resolved/ — the intent delivered, and the observation it graduated from
	// still reads as unaddressed (or as declined, in wontfix/).
	DriftShippedUnresolved = "shipped_unresolved"
	// DriftRetiredField: a record still carries a retired back-link key, which no
	// reader reads; `abcd capture migrate --apply` rewrites it.
	DriftRetiredField = "retired_field"
)

// IssueDriftRequest is the input to IssueDrift. Now stamps the receipt; the
// zero value means the current time.
type IssueDriftRequest struct {
	RepoRoot   string
	IssuesRoot string
	Now        time.Time
}

// DriftFinding is one broken join. Record is the record carrying the edge the
// finding is about, Other the record that edge names; Path is Record's path,
// repo-relative.
type DriftFinding struct {
	Kind    string `json:"kind"`
	Record  string `json:"record"`
	Other   string `json:"other"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// IssueDriftResult is the outcome of one drift walk.
type IssueDriftResult struct {
	Scanned  int            `json:"scanned"`
	Findings []DriftFinding `json:"findings"`
	// ReceiptPath is the repo-relative report this run left in the local tier.
	ReceiptPath string `json:"receipt_path"`
}

// driftReceiptRelDir is where each run's report lands: the gitignored local
// tier's logs, one directory per run (the spc-23 receipt shape).
var driftReceiptRelDir = filepath.Join(".abcd", ".work.local", "logs", "audit")

// IssueDrift walks the intent store and the issue ledger (readings included)
// and reports every place the promote join does not read the same from both
// ends: the bidirectional cross-reference check itd-4 AC3 names, in the shape
// the predecessor store's spc-23 gave it. It writes nothing to either store; it
// leaves one report under .abcd/.work.local/logs/audit/issue-drift-<ts>/, and
// the front door decides what the findings mean for its exit code.
func IssueDrift(req IssueDriftRequest) (IssueDriftResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return IssueDriftResult{}, err
	}
	records, err := migrateScan(repoRoot, issuesRoot)
	if err != nil {
		return IssueDriftResult{}, err
	}
	ledgerByID := map[string]*migrateRecord{}
	intentByID := map[string]*migrateRecord{}
	for _, r := range records {
		if r.intentHalf {
			intentByID[r.id] = r
		} else {
			ledgerByID[r.id] = r
		}
	}
	findings := []DriftFinding{}
	add := func(kind string, r *migrateRecord, other, msg string) {
		findings = append(findings, DriftFinding{Kind: kind, Record: r.id, Other: other, Path: r.rel, Message: msg})
	}
	for _, r := range records {
		if r.retired != "" {
			add(DriftRetiredField, r, r.retiredValue, fmt.Sprintf(
				"%s carries the retired `%s: %s`, which no reader reads (renamed to `%s`); %s",
				r.id, r.retired, r.retiredValue, r.key, issueschema.MigrateHint))
		}
	}
	for _, it := range records {
		if !it.intentHalf {
			continue
		}
		bucket := filepath.Base(filepath.Dir(it.abs))
		for _, src := range it.list {
			rec, ok := ledgerByID[src]
			if !ok {
				add(DriftDangling, it, src, fmt.Sprintf(
					"%s names %s in `related_issues`, and no ledger record carries that id", it.id, src))
				continue
			}
			if !containsString(rec.list, it.id) {
				add(DriftOneSided, it, src, fmt.Sprintf(
					"%s names %s in `related_issues`, but %s does not name %s in `related_intents`; the join reads from one end only (`abcd capture promote %s --intent %s` writes the missing half)",
					it.id, src, src, it.id, src, it.id))
			}
			if bucket == intent.BucketShipped && strings.HasPrefix(src, "iss-") {
				if status := filepath.Base(filepath.Dir(rec.abs)); status != string(StateResolved) {
					add(DriftShippedUnresolved, it, src, fmt.Sprintf(
						"%s is shipped, but %s, which it names in `related_issues`, is in %s/ rather than resolved/",
						it.id, src, status))
				}
			}
		}
	}
	for _, rec := range records {
		if rec.intentHalf {
			continue
		}
		for _, itd := range rec.list {
			it, ok := intentByID[itd]
			if !ok {
				add(DriftDangling, rec, itd, fmt.Sprintf(
					"%s names %s in `related_intents`, and no intent bucket holds it", rec.id, itd))
				continue
			}
			if strings.HasPrefix(rec.id, issueschema.ReadingItemFamily+"-") && !containsString(it.list, rec.id) {
				add(DriftOneSided, rec, itd, fmt.Sprintf(
					"%s names %s in `related_intents`, but %s does not name %s in `related_issues`; a reading item's forward stamp is written only by promote, so the join reads from one end only",
					rec.id, itd, itd, rec.id))
			}
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind+findings[i].Other < findings[j].Kind+findings[j].Other
	})

	res := IssueDriftResult{Scanned: len(records), Findings: findings}
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	rel, err := writeDriftReceipt(repoRoot, now, &res)
	if err != nil {
		return IssueDriftResult{}, err
	}
	res.ReceiptPath = rel
	return res, nil
}

// writeDriftReceipt allocates this run's receipt directory and writes the
// report into it, returning its repo-relative path.
func writeDriftReceipt(repoRoot string, now time.Time, res *IssueDriftResult) (string, error) {
	base := filepath.Join(repoRoot, driftReceiptRelDir)
	stamp := "issue-drift-" + now.UTC().Format("20060102T150405Z")
	dir := filepath.Join(base, stamp)
	for n := 1; ; n++ {
		if _, err := os.Lstat(dir); os.IsNotExist(err) {
			break
		}
		if n >= 1000 {
			return "", fmt.Errorf("issue drift: could not allocate a unique receipt directory for %s", stamp)
		}
		dir = filepath.Join(base, fmt.Sprintf("%s-%03d", stamp, n))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("issue drift: %w", err)
	}
	path := filepath.Join(dir, "report.json")
	res.ReceiptPath = fsutil.RepoRel(repoRoot, path)
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	if err := fsutil.WriteFileAtomicPreserveMode(path, append(data, '\n')); err != nil {
		return "", fmt.Errorf("issue drift: %w", err)
	}
	return res.ReceiptPath, nil
}
