package loop

// drain_origin.go is where a drain move reads its ledger (spc-2610161044461448
// step 1; iss-2610160957404827, iss-2610161015448977; dcn-2610161040350256).
// Every account draining one repository shares origin and nothing else, and a
// checkout falls behind origin within hours, so a move never plans from the
// checkout's own ledger when it has a remote named origin: under the drain's
// lock, before any judgement or reservation, it fetches the reservations and
// origin's default branch and reads one snapshot of the ledger out of the
// commit it fetched. The plan, the lane's start check, the judgement's remedy
// digest and the reservation's footprint all read that snapshot.
//
// A move that cannot read origin plans nothing: a fetch that fails, a default
// branch the fetch could not read, or no default branch of origin this checkout
// can name. It records the failure as an origin outage probe in the drain's
// state and waits as the outage gate waits, on implement.DefaultSchedule: a
// move before the next probe is due looks at nothing, and one at or after it
// fetches again. It never plans from what the checkout last saw.
//
// The dry run fetches nothing: it plans from origin's default branch as last
// fetched and says how old that view is, how many commits the checkout is
// behind it, and which issues open here origin has already finished.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Why a drain move could not read origin: the kinds an origin wait records.
const (
	// DrainOriginFetch: the fetch from origin failed.
	DrainOriginFetch = "fetch"
	// DrainOriginDefaultBranch: the fetch could not read origin's default
	// branch (renamed since this checkout last knew it), or its commit could
	// not be resolved.
	DrainOriginDefaultBranch = "default-branch"
	// DrainOriginHead: this checkout names no default branch of origin (no
	// origin/HEAD, and no origin/main, master, trunk or develop).
	DrainOriginHead = "origin-head"
)

// drainOriginKinds are the kinds a drain's state may record.
var drainOriginKinds = []string{DrainOriginFetch, DrainOriginDefaultBranch, DrainOriginHead}

// DrainOrigin is the view of origin's default branch a plan was read from.
type DrainOrigin struct {
	// Ref is the branch as a person names it (origin/main), and Commit its
	// commit as fetched.
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
	// FetchedAt is when abcd last fetched here, nil when nothing here stamped
	// a fetch; AgeMinutes is the view's age then, at the plan's time.
	FetchedAt  *time.Time `json:"fetched_at"`
	AgeMinutes int        `json:"age_minutes"`
	// Behind counts the commits of Ref this checkout's HEAD lacks.
	Behind int `json:"behind"`
	// Finished are the issues open in this checkout's ledger that Ref holds
	// resolved or wontfix: the work a stale checkout would repeat.
	Finished []string `json:"finished"`
}

// DrainOriginWait is a drain waiting on origin: since when, why (one of the
// DrainOrigin* kinds), the probes that failed, and when the next is due.
type DrainOriginWait struct {
	Since       time.Time `json:"since"`
	Kind        string    `json:"kind"`
	Probes      int       `json:"probes"`
	NextProbeAt time.Time `json:"next_probe_at"`
}

// written reports whether w is one the drain writes: a known kind and at least
// one probe.
func (w *DrainOriginWait) written() bool {
	return w == nil || slices.Contains(drainOriginKinds, w.Kind) && w.Probes > 0
}

// DrainPreview is `abcd drain --dry-run`: the plan, and the view of origin it
// was read from. Origin is nil in a checkout with no remote named origin, and
// when this checkout names no default branch of origin; OriginUnread then says
// why, and the plan is the checkout's own ledger.
type DrainPreview struct {
	capture.DrainPlan
	Origin       *DrainOrigin `json:"origin"`
	OriginUnread string       `json:"origin_unread,omitempty"`
}

// DrainDryRun plans the drain as its next move would, from origin's default
// branch as last fetched, and fetches and writes nothing.
func DrainDryRun(repoRoot string, now time.Time) (DrainPreview, error) {
	if _, err := drainrule.Load(repoRoot); err != nil {
		return DrainPreview{}, err
	}
	has, err := claim.HasOrigin(repoRoot)
	if err != nil {
		return DrainPreview{}, err
	}
	var pv DrainPreview
	if has {
		v, _, why := lastFetchedOrigin(repoRoot)
		if why == "" {
			plan, origin, err := planFromOrigin(repoRoot, v, now)
			if err != nil {
				return DrainPreview{}, err
			}
			pv.DrainPlan, pv.Origin = plan, origin
			return pv, nil
		}
		pv.OriginUnread = why + "; this plan reads the checkout's own ledger, and a drain move plans nothing until origin's default branch can be read"
	}
	pv.DrainPlan, err = capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
	return pv, err
}

// originView is origin's default branch as this checkout last fetched it.
type originView struct {
	ref, commit string
}

// lastFetchedOrigin is origin's default branch as last fetched (the ref the
// lane's worktree is cut from, gitutil.DefaultRef), or the kind of wait and
// why there is none.
func lastFetchedOrigin(repoRoot string) (v originView, kind, why string) {
	ref := gitutil.DefaultRef(repoRoot)
	if !strings.HasPrefix(ref, "refs/remotes/"+claim.Remote+"/") {
		return originView{}, DrainOriginHead, "this checkout names no default branch of " + claim.Remote + " (no origin/HEAD, and no origin/main, master, trunk or develop)"
	}
	sha, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	if err != nil || !gitutil.IsFullSHA(sha) {
		return originView{}, DrainOriginDefaultBranch, "could not resolve " + gitutil.ShortRef(ref) + "'s commit"
	}
	return originView{ref: ref, commit: sha}, "", ""
}

// planFromOrigin plans the drain from the ledger v's commit holds, and
// describes the view at now.
func planFromOrigin(repoRoot string, v originView, now time.Time) (capture.DrainPlan, *DrainOrigin, error) {
	ledger, err := treeLedger(repoRoot, v)
	if err != nil {
		return capture.DrainPlan{}, nil, err
	}
	plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot, Ledger: ledger})
	if err != nil {
		return capture.DrainPlan{}, nil, err
	}
	o := &DrainOrigin{Ref: gitutil.ShortRef(v.ref), Commit: v.commit, Finished: []string{}}
	if at := claim.LastFetched(repoRoot); !at.IsZero() {
		o.FetchedAt = &at
		if age := now.Sub(at); age > 0 {
			o.AgeMinutes = int(age / time.Minute)
		}
	}
	if out, err := gitutil.Run(repoRoot, "rev-list", "--count", "HEAD.."+v.commit, "--"); err == nil {
		o.Behind, _ = strconv.Atoi(strings.TrimSpace(out))
	}
	terminal := ledger.Terminal()
	for _, id := range openHere(repoRoot) {
		if _, found := slices.BinarySearch(terminal, id); found {
			o.Finished = append(o.Finished, id)
		}
	}
	return plan, o, nil
}

// treeLedger reads the issue ledger's status folders out of v's commit: every
// file's name, and the content of each record in open/.
func treeLedger(repoRoot string, v originView) (*capture.TreeLedger, error) {
	at := baseTree{root: repoRoot, sha: v.commit}
	source := gitutil.ShortRef(v.ref) + " at " + shortSHA(v.commit)
	entries, err := at.list(recordid.IssuesRelDir+"/", true)
	if err != nil {
		return nil, fmt.Errorf("reading the ledger at %s: %w", source, err)
	}
	l := &capture.TreeLedger{Source: source}
	for _, e := range entries {
		rest, _ := strings.CutPrefix(e.path, recordid.IssuesRelDir+"/")
		folder, name, ok := strings.Cut(rest, "/")
		if !ok || strings.Contains(name, "/") {
			continue
		}
		f := capture.LedgerFile{Folder: folder, Path: e.path}
		if folder == string(capture.StateOpen) && strings.HasPrefix(name, "iss-") {
			b, err := at.blob(e, issueschema.RecordReadLimit)
			if err != nil {
				f.Unread = err.Error()
			} else {
				f.Text = string(b)
			}
		}
		l.Files = append(l.Files, f)
	}
	return l, nil
}

// openHere is the ids this checkout's ledger holds in open/, by file name.
func openHere(repoRoot string) []string {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(recordid.IssuesRelDir), string(capture.StateOpen)))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if id, _, ok := recordid.SplitRecordFilename("iss", e.Name()); ok {
			out = append(out, recordid.CanonCitedID(id))
		}
	}
	slices.Sort(out)
	return out
}

// movePlan is a move's plan. In a checkout with no remote named origin it is
// the checkout's own ledger, as it has always been. Otherwise it is origin's
// default branch, fetched now; a move that cannot read it returns the rule
// alone and wait, the words of the wait, having recorded the probe in st (a
// move before the next probe is due fetches nothing). probed reports whether
// st changed. A move that reads origin clears the wait, and recovered says so.
func movePlan(repoRoot string, st *DrainState, rule drainrule.Rule, now time.Time) (plan capture.DrainPlan, origin *DrainOrigin, wait, recovered string, probed bool, err error) {
	has, err := claim.HasOrigin(repoRoot)
	if err != nil {
		return plan, nil, "", "", false, err
	}
	if !has {
		plan, err = capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
		return plan, nil, "", "", false, err
	}
	if w := st.OriginWait; w != nil && now.Before(w.NextProbeAt) {
		at := w.NextProbeAt.UTC().Format(time.RFC3339)
		return capture.RuleOnlyPlan(rule), nil, fmt.Sprintf("%s could not be read at the last move (%s), so the drain plans nothing before %s, when it looks again: it never plans from what this checkout last saw; run `abcd drain` again at or after %s",
			claim.Remote, originKindWords(w.Kind), at, at), "", false, nil
	}
	kind, why := "", ""
	gap, ferr := claim.Fetch(repoRoot, claim.Options{Now: func() time.Time { return now }})
	v, unreadKind, unread := lastFetchedOrigin(repoRoot)
	switch {
	case ferr != nil:
		kind, why = DrainOriginFetch, "could not fetch "+claim.Remote+": "+termsafe.Sanitize(ferr.Error())
	case gap != "":
		kind, why = DrainOriginDefaultBranch, termsafe.Sanitize(gap)+"; if its default branch was renamed, `git remote set-head origin --auto` names the new one"
	case unreadKind == DrainOriginHead:
		kind, why = DrainOriginHead, unread+"; run `git remote set-head origin --auto` and `git fetch origin`"
	case unreadKind != "":
		kind, why = unreadKind, "could not read "+claim.Remote+"'s default branch: "+unread
	}
	if kind != "" {
		w := &DrainOriginWait{Since: now, Kind: kind, Probes: 1}
		if prev := st.OriginWait; prev != nil {
			w.Since, w.Probes = prev.Since, prev.Probes+1
		}
		w.NextProbeAt = implement.DefaultSchedule.Next(now, w.Probes-1)
		st.OriginWait = w
		at := w.NextProbeAt.UTC().Format(time.RFC3339)
		return capture.RuleOnlyPlan(rule), nil, fmt.Sprintf("%s, so the move plans nothing: it never plans from what this checkout last saw; it waits on %s as for an outage (probe %d since %s) and looks again at or after %s; run `abcd drain` again then",
			why, claim.Remote, w.Probes, w.Since.UTC().Format(time.RFC3339), at), "", true, nil
	}
	if plan, origin, err = planFromOrigin(repoRoot, v, now); err != nil {
		return plan, nil, "", "", false, err
	}
	if w := st.OriginWait; w != nil {
		recovered = fmt.Sprintf("%s answered again after %d failed probe(s) since %s", claim.Remote, w.Probes, w.Since.UTC().Format(time.RFC3339))
		st.OriginWait, probed = nil, true
	}
	return plan, origin, "", recovered, probed, nil
}

// originKindWords is a wait's kind in words.
func originKindWords(kind string) string {
	switch kind {
	case DrainOriginFetch:
		return "the fetch failed"
	case DrainOriginDefaultBranch:
		return "its default branch could not be read"
	}
	return "this checkout names no default branch of " + claim.Remote + "; run `git remote set-head origin --auto` and `git fetch origin`"
}
