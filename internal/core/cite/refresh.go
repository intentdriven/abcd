// Package cite is the live half of the citation gate: the on-demand refresh
// that fetches every cited URL and writes the committed baseline the
// zero-network docs lint then enforces offline.
//
// The split is spc-17's whole design bet. Fetching is non-deterministic and
// needs the network; deciding whether a commit is allowed must be neither. So
// this package is the ONLY place abcd dials out on behalf of documentation, it
// runs when a maintainer asks it to, and everything it learns arrives at the
// gate as a reviewable committed record rather than as whatever the network
// happened to answer during CI.
//
// It writes nothing it did not establish. A source that refuses automated
// fetchers produces no entry at all — not a guess, not a "probably fine" —
// because the one thing a committed record must never contain is a receipt
// nobody earned. Those URLs go to a human instead, and the confirm verb records
// that a human cleared them and when, never how.
package cite

import (
	"errors"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// RefreshError is a refused refresh — a distinct type so a caller can tell a
// refusal to write from an I/O fault without string matching.
type RefreshError struct{ msg string }

func (e *RefreshError) Error() string { return e.msg }

// dateLayout matches the baseline's date precision. The staleness policy is
// measured in days, so an instant would add churn to a committed file for no
// gain in the only arithmetic that reads it.
const dateLayout = "2006-01-02"

// defaultParallel bounds how many citations are in flight at once. It is small
// on purpose: a refresh is a courtesy visit to other people's servers, and the
// run's cost is already bounded by one attempt per URL.
const defaultParallel = 6

// RefreshRequest is the input to one refresh run.
type RefreshRequest struct {
	// RepoRoot is the repository to refresh.
	RepoRoot string
	// Config supplies the roots to walk and the citation policy (baseline path
	// and staleness thresholds) — the same config the gate reads, so the verb
	// and the gate can never disagree about either.
	Config lint.Config
	// Checker is the refresh seam. Nil means the native bounded fetcher.
	Checker Checker
	// Now is the clock. Supplied by the caller so a run's record is a function
	// of its inputs.
	Now time.Time
	// Parallel bounds concurrent checks; zero means defaultParallel.
	Parallel int
}

// Outcome is one cited URL's disposition in a run.
type Outcome struct {
	URL      string `json:"url"`
	Status   Status `json:"status"`
	FinalURL string `json:"final_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// StatusPreserved is the fourth disposition a REFRESH can report, distinct from
// the three a checker can return: the entry was not checked at all, because a
// human's receipt for it is still current.
const StatusPreserved Status = "preserved"

// QueueItem is one line of the manual checklist (spc-17's rung 1): a URL an
// automated fetcher may not read, and where the docs cite it.
type QueueItem struct {
	URL    string              `json:"url"`
	Detail string              `json:"detail"`
	Sites  []lint.CitationSite `json:"sites"`
}

// RefreshResult is the record of one run, for the operator's transcript and for
// `--json`.
type RefreshResult struct {
	// BaselinePath is repo-relative — never absolute, so machine output carries
	// no developer-identity path.
	BaselinePath string `json:"baseline_path"`
	Cited        int    `json:"cited"`
	// Fetched counts the URLs actually requested this run; Preserved counts the
	// current manual receipts left untouched. They sum to Cited.
	Fetched   int       `json:"fetched"`
	Preserved int       `json:"preserved"`
	Outcomes  []Outcome `json:"outcomes"`
	// Queue is the manual checklist: everything a human must clear.
	Queue []QueueItem `json:"queue"`
	// Dropped lists entries removed because the docs no longer cite them.
	Dropped []string `json:"dropped,omitempty"`
}

// Counts summarises the outcomes by status, for a one-line report.
func (r RefreshResult) Counts() (ok, broken, blocked, preserved int) {
	for _, o := range r.Outcomes {
		switch o.Status {
		case StatusOK:
			ok++
		case StatusBroken:
			broken++
		case StatusBlocked:
			blocked++
		case StatusPreserved:
			preserved++
		}
	}
	return ok, broken, blocked, preserved
}

// Refresh checks every cited URL and rewrites the baseline.
//
// The merge is where the honesty rules live, and there are three:
//
//   - A current MANUAL receipt is preserved verbatim and its URL is not even
//     requested. Re-fetching it could only produce a worse answer (the source
//     blocks robots, which is why a human looked in the first place), and
//     overwriting a human's confirmation with a robot's failure is the exact
//     downgrade the record must never suffer.
//   - A STALE manual receipt is re-checked, because AC 3 puts human and machine
//     verifications on one clock: once it ages out it stops being authoritative.
//     If the re-check succeeds the entry becomes a fresh automatic one; if the
//     source still blocks, the stale receipt is KEPT (so the gate keeps warning
//     truthfully) and the URL re-enters the queue.
//   - A BLOCKED URL with no prior entry writes nothing. The gate then reports it
//     as unreceipted, which is precisely the truth, and the queue names it.
//
// Entries whose URLs the docs no longer cite are dropped: the baseline answers
// for what is published today.
func Refresh(req RefreshRequest) (RefreshResult, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stamp := now.Format(dateLayout)

	checker := req.Checker
	if checker == nil {
		checker = newShippedHTTPChecker()
	}
	parallel := req.Parallel
	if parallel <= 0 {
		parallel = defaultParallel
	}

	policy, err := lint.CitationPolicy(req.Config, req.RepoRoot)
	if err != nil {
		return RefreshResult{}, err
	}
	relPath, absPath, warnDays := policy.BaselineRel, policy.BaselinePath, policy.WarnDays

	cited, err := lint.CollectCitedURLs(req.Config, req.RepoRoot)
	if err != nil {
		return RefreshResult{}, err
	}

	// A missing baseline is the ordinary first run. A malformed one is a
	// refusal: silently replacing a record we cannot read would discard whatever
	// manual receipts it held, which is the one loss no later run can repair.
	previous := map[string]lint.BaselineEntry{}
	old, err := lint.LoadBaseline(absPath)
	switch {
	case err == nil:
		previous = old.Entries
	case isNotExist(err):
	default:
		return RefreshResult{}, err
	}

	// Decide per URL whether it is checked at all, then run the checks bounded.
	type job struct {
		ref  lint.CitedURL
		prev lint.BaselineEntry
		had  bool
	}
	var jobs []job
	var toCheck []string
	res := RefreshResult{BaselinePath: relPath, Cited: len(cited)}
	next := map[string]lint.BaselineEntry{}

	for _, ref := range cited {
		prev, had := previous[ref.URL]
		if had && prev.Verification == lint.VerificationManual && !stale(prev, now, warnDays) {
			next[ref.URL] = prev
			res.Preserved++
			res.Outcomes = append(res.Outcomes, Outcome{
				URL: ref.URL, Status: StatusPreserved, FinalURL: prev.FinalURL,
				Detail: "human-verified " + prev.VerifiedOn + "; not re-checked until it goes stale",
			})
			continue
		}
		jobs = append(jobs, job{ref: ref, prev: prev, had: had})
		toCheck = append(toCheck, ref.URL)
	}
	res.Fetched = len(jobs)

	checked := runChecks(checker, toCheck, parallel)
	// answered counts the checks a host replied to at all — any status line,
	// including a 404 or a 403. It is the signal the wholesale-failure guard
	// below reads, and it is a fact the CHECKER establishes rather than one this
	// function infers: StatusBroken alone cannot tell a dead link from a dead
	// network, and every attempt to recover that bit from a proxy was wrong.
	answered := 0

	for _, j := range jobs {
		out := checked[j.ref.URL]
		res.Outcomes = append(res.Outcomes, Outcome{
			URL: j.ref.URL, Status: out.Status, FinalURL: out.FinalURL, Detail: out.Detail,
		})
		// StatusOK and StatusBlocked ENTAIL an answer — a 2xx cannot exist
		// without one, and "blocked" is defined by a status code. Only
		// StatusBroken is genuinely ambiguous, so only there does the checker's
		// own bit decide. This is not inference from a proxy; it is what those
		// two statuses mean, and it keeps a third-party adapter that omits the
		// field from refusing a run in which every check succeeded.
		if out.Answered || out.Status == StatusOK || out.Status == StatusBlocked {
			answered++
		}
		switch out.Status {
		case StatusOK:
			final := out.FinalURL
			if final == "" {
				final = j.ref.URL
			}
			next[j.ref.URL] = lint.BaselineEntry{
				URL: j.ref.URL, FinalURL: final, LastChecked: stamp,
				Outcome: lint.OutcomeAlive, Verification: lint.VerificationAutomatic, VerifiedOn: stamp,
			}
		case StatusBroken:
			// A broken entry still needs a resolved address for the schema; the
			// cited one is the honest answer when nothing else resolved.
			final := out.FinalURL
			if final == "" {
				final = j.ref.URL
			}
			next[j.ref.URL] = lint.BaselineEntry{
				URL: j.ref.URL, FinalURL: final, LastChecked: stamp,
				Outcome: lint.OutcomeBroken, Verification: lint.VerificationAutomatic, VerifiedOn: stamp,
			}
		case StatusBlocked:
			if j.had {
				next[j.ref.URL] = j.prev
			}
			res.Queue = append(res.Queue, QueueItem{URL: j.ref.URL, Detail: out.Detail, Sites: j.ref.Sites})
		default:
			// The seam is exported, so a later adapter is a producer this package
			// does not control. An unrecognised status must stop the run: falling
			// through would drop the URL from BOTH the baseline and the queue,
			// which is the one outcome nothing downstream could detect.
			return RefreshResult{}, &RefreshError{"checker returned an unrecognised status " +
				strconv.Quote(string(out.Status)) + " for " + j.ref.URL}
		}
	}

	// A run in which NO HOST ANSWERED is a broken network, not a corpus that died
	// overnight. Committing it would rewrite every entry as broken — stale human
	// receipts included, since those are re-checked and so never reach the
	// blocked branch that protects the current ones — and the gate would then
	// block every commit until someone reverted the file by hand.
	//
	// The condition tests ANSWERED, not "succeeded", and deliberately does not
	// consult Preserved. A preserved receipt was never fetched, so it is no
	// evidence the network worked; letting it veto the guard meant any repo that
	// had ever used `confirm` — the designed steady state for robot-refusing
	// sources — silently lost the protection. And because a genuine 404 IS an
	// answer, a repo whose only citation is really dead still records it rather
	// than being told forever to check its connectivity.
	//
	// It binds only when there was a record to protect: a first run has nothing
	// to lose and is allowed to write one.
	if len(previous) > 0 && res.Fetched > 0 && answered == 0 {
		return RefreshResult{}, &RefreshError{"no host answered any of the " + strconv.Itoa(res.Fetched) +
			" citation(s) checked, which is a broken network far more often than a broken corpus; " +
			"the existing baseline is left untouched. Check connectivity and re-run"}
	}

	citedSet := map[string]bool{}
	for _, ref := range cited {
		citedSet[ref.URL] = true
	}
	for u := range previous {
		if !citedSet[u] {
			res.Dropped = append(res.Dropped, u)
		}
	}

	sort.Slice(res.Outcomes, func(i, j int) bool { return res.Outcomes[i].URL < res.Outcomes[j].URL })
	sort.Slice(res.Queue, func(i, j int) bool { return res.Queue[i].URL < res.Queue[j].URL })
	sort.Strings(res.Dropped)

	if err := lint.SaveBaseline(absPath, lint.Baseline{SchemaVersion: lint.BaselineSchemaVersion, Entries: next}); err != nil {
		return RefreshResult{}, err
	}
	return res, nil
}

// stale reports whether an entry has passed the warn threshold — the point at
// which AC 3 stops treating a verification, human or machine, as authoritative.
func stale(e lint.BaselineEntry, now time.Time, warnDays int) bool {
	checked, err := e.Checked()
	if err != nil {
		return true // unparsable is not current; re-check it
	}
	// The gate's own arithmetic, not a second copy: the verb must call an entry
	// stale on exactly the day the lint will.
	return lint.DaysBetween(checked, now) >= warnDays
}

// runChecks fans the checks out over a bounded worker pool and collects them
// into a map, so the caller's iteration order — and therefore the record — stays
// a function of the sorted cited set rather than of scheduling.
func runChecks(checker Checker, urls []string, parallel int) map[string]CheckOutcome {
	out := make(map[string]CheckOutcome, len(urls))
	if len(urls) == 0 {
		return out
	}
	if parallel > len(urls) {
		parallel = len(urls)
	}
	if parallel <= 1 {
		for _, u := range urls {
			out[u] = checker.Check(u)
		}
		return out
	}

	results := make([]CheckOutcome, len(urls))
	next := make(chan int)
	done := make(chan struct{})
	for w := 0; w < parallel; w++ {
		go func() {
			for i := range next {
				results[i] = checker.Check(urls[i])
			}
			done <- struct{}{}
		}()
	}
	for i := range urls {
		next <- i
	}
	close(next)
	for w := 0; w < parallel; w++ {
		<-done
	}
	for i, u := range urls {
		out[u] = results[i]
	}
	return out
}

// isNotExist reports the "no baseline yet" case. LoadBaseline returns the
// underlying os.ErrNotExist unwrapped for exactly this distinction.
func isNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }
