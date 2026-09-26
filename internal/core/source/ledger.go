package source

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxLedgerBytes caps a ledger read (trust boundary).
const maxLedgerBytes = 64 << 20

// Record is one ledger line. The first eight fields are the spec's; used_in traces
// the influence to the consuming documents; corrects and flips name, by 1-based line
// number, the earlier line a correction or a citation flip refers to. A line is
// never edited: a correction and a flip are both new lines.
type Record struct {
	TS            string   `json:"ts"`
	Repo          string   `json:"repo"`
	DecisionRef   string   `json:"decision_ref"`
	Claim         string   `json:"claim"`
	SourceKey     string   `json:"source_key"`
	Locator       string   `json:"locator"`
	Influence     string   `json:"influence"`
	UsedIn        []string `json:"used_in,omitempty"`
	CitedPublicly bool     `json:"cited_publicly"`
	Corrects      int      `json:"corrects,omitempty"`
	Flips         int      `json:"flips,omitempty"`
}

// AppendRequest records one influence.
type AppendRequest struct {
	Corpus      string
	Repo        string
	DecisionRef string
	Claim       string
	SourceKey   string
	Locator     string
	Influence   string
	UsedIn      []string
	// Corrects is the 1-based line this record corrects, or 0.
	Corrects int
	Now      time.Time
}

// AppendResult is a line written (or read), with its number.
type AppendResult struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Record Record `json:"record"`
}

// ledgerRel is the corpus-relative ledger path for repo.
func ledgerRel(repo string) string { return path.Join(LedgerDir, repo+".jsonl") }

// readLedger reads repo's ledger; an absent file is an empty ledger.
func readLedger(dir, repo string) ([]Record, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(dir, filepath.FromSlash(ledgerRel(repo))), maxLedgerBytes)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("%w: %s cannot be read (oversize, or not a regular file)", ErrInvalidLedger, ledgerRel(repo))
	}
	var out []Record
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), maxLedgerBytes)
	n := 0
	for sc.Scan() {
		n++
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%w: line %d of %s is not a JSON record", ErrInvalidLedger, n, ledgerRel(repo))
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidLedger, ledgerRel(repo), err)
	}
	return out, nil
}

// List returns repo's ledger, numbered from 1. An absent ledger is empty.
func List(corpus, repo string) ([]AppendResult, error) {
	if _, err := Load(corpus); err != nil {
		return nil, err
	}
	if !repoRe.MatchString(repo) {
		return nil, fmt.Errorf("%w: repository handle %q is not a plain name", ErrInvalidLedger, repo)
	}
	recs, err := readLedger(corpus, repo)
	if err != nil {
		return nil, err
	}
	out := make([]AppendResult, len(recs))
	for i, r := range recs {
		out[i] = AppendResult{Path: ledgerRel(repo), Line: i + 1, Record: r}
	}
	return out, nil
}

// stamp is a ledger timestamp: UTC, RFC 3339, to the second.
func stamp(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Format(time.RFC3339)
}

// Append adds one influence record to repo's ledger and commits it. cited_publicly
// is always false here: exercising the right to cite is Flip's, and only Flip's.
func Append(req AppendRequest) (AppendResult, error) {
	c, err := Load(req.Corpus)
	if err != nil {
		return AppendResult{}, err
	}
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidLedger}, a...)...)
	}
	switch {
	case !repoRe.MatchString(req.Repo):
		return AppendResult{}, bad("repository handle %q is not a plain name", req.Repo)
	case !oneLine(req.DecisionRef):
		return AppendResult{}, bad("the decision reference is empty or spans lines")
	case strings.TrimSpace(req.Claim) == "":
		return AppendResult{}, bad("the claim is empty")
	case !slices.Contains(Influences, req.Influence):
		return AppendResult{}, bad("influence %q is not one of %s", req.Influence, strings.Join(Influences, ", "))
	case req.Locator != "" && strings.ContainsAny(req.Locator, "\r\n"):
		return AppendResult{}, bad("the locator spans lines")
	case req.Corrects < 0:
		return AppendResult{}, bad("corrects must name a line number")
	}
	for i, u := range req.UsedIn {
		if !oneLine(u) {
			return AppendResult{}, bad("used-in path %d is empty or spans lines", i+1)
		}
	}
	if _, ok := c.Lookup(req.SourceKey); !ok {
		return AppendResult{}, fmt.Errorf("%w: %q", ErrUnknownSource, req.SourceKey)
	}
	rec := Record{TS: stamp(req.Now), Repo: req.Repo, DecisionRef: strings.TrimSpace(req.DecisionRef),
		Claim: strings.TrimSpace(req.Claim), SourceKey: req.SourceKey, Locator: req.Locator,
		Influence: req.Influence, UsedIn: req.UsedIn, Corrects: req.Corrects}
	return appendRecord(c.Dir, req.Repo, func(existing []Record) (Record, error) {
		if req.Corrects > len(existing) {
			return Record{}, bad("corrects names line %d, and the ledger has %d", req.Corrects, len(existing))
		}
		return rec, nil
	})
}

// Flip exercises the right to cite for one ledger line (adr-41 gate 2), and only
// after gate 1 grants it: the line's source must sit in public/ and carry
// permission_status citable. A refusal names the failing gate and appends nothing.
// A successful flip is itself a new line — the original with cited_publicly true
// and flips naming it — so the ledger keeps who cited what, and when.
//
// A person flips a line; an agent never does. The binary cannot tell the two apart,
// so the command pages carry that rule.
func Flip(corpus, repo string, line int, now time.Time) (AppendResult, error) {
	c, err := Load(corpus)
	if err != nil {
		return AppendResult{}, err
	}
	if !repoRe.MatchString(repo) {
		return AppendResult{}, fmt.Errorf("%w: repository handle %q is not a plain name", ErrInvalidLedger, repo)
	}
	return appendRecord(c.Dir, repo, func(existing []Record) (Record, error) {
		if line < 1 || line > len(existing) {
			return Record{}, fmt.Errorf("%w: the ledger has no line %d", ErrInvalidLedger, line)
		}
		orig := existing[line-1]
		if orig.Flips != 0 {
			return Record{}, fmt.Errorf("%w: line %d is itself a flip of line %d", ErrCitationRefused, line, orig.Flips)
		}
		for i, r := range existing {
			if r.Flips == line {
				return Record{}, fmt.Errorf("%w: line %d was already flipped, at line %d", ErrCitationRefused, line, i+1)
			}
		}
		e, ok := c.Lookup(orig.SourceKey)
		if !ok {
			return Record{}, fmt.Errorf("%w: gate 1 — source %q is no longer in the corpus", ErrCitationRefused, orig.SourceKey)
		}
		if e.Custom.PermissionStatus != PermissionCitable {
			return Record{}, fmt.Errorf("%w: gate 1 — source %q has permission_status %q, and only %q grants the right to cite",
				ErrCitationRefused, orig.SourceKey, e.Custom.PermissionStatus, PermissionCitable)
		}
		if class := c.Class(orig.SourceKey); class != ClassPublic || e.Custom.Confidential {
			return Record{}, fmt.Errorf("%w: gate 1 — source %q is confidential (its folder is not under public/); declassify it when it is published",
				ErrCitationRefused, orig.SourceKey)
		}
		flip := orig
		flip.TS = stamp(now)
		flip.CitedPublicly = true
		flip.Corrects = 0
		flip.Flips = line
		return flip, nil
	})
}

// appendRecord appends the record build returns — build sees the ledger as it
// stands, under the corpus lock, and may refuse — then commits it.
func appendRecord(dir, repo string, build func(existing []Record) (Record, error)) (AppendResult, error) {
	var res AppendResult
	err := withLock(dir, func() error {
		existing, err := readLedger(dir, repo)
		if err != nil {
			return err
		}
		rec, err := build(existing)
		if err != nil {
			return err
		}
		b, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		root, err := os.OpenRoot(dir)
		if err != nil {
			return err
		}
		defer root.Close()
		if err := root.MkdirAll(LedgerDir, 0o700); err != nil {
			return fmt.Errorf("%w: cannot create %s/: %v", ErrCorpusInvalid, LedgerDir, err)
		}
		rel := ledgerRel(repo)
		if err := fsutil.AppendLineIn(root, rel, b, 0o600); err != nil {
			return fmt.Errorf("%w: appending to %s failed: %v", ErrInvalidLedger, rel, err)
		}
		line := len(existing) + 1
		msg := fmt.Sprintf("ledger(%s): line %d — %s", repo, line, rec.SourceKey)
		if rec.Flips != 0 {
			msg = fmt.Sprintf("ledger(%s): line %d flips line %d to cited_publicly — %s", repo, line, rec.Flips, rec.SourceKey)
		}
		if err := commit(dir, msg, rel); err != nil {
			return err
		}
		res = AppendResult{Path: rel, Line: line, Record: rec}
		return nil
	})
	return res, err
}
