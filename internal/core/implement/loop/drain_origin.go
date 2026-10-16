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
// fetches again. A probe failing HourlyFor into the schedule's hourly stage
// gives up, as the outage gate does, and the drain ends with stopped "origin".
// It never plans from what the checkout last saw.
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

// DrainStoppedOrigin is why a drain ended that waited on origin through the
// outage schedule's hourly stage and gave up.
const DrainStoppedOrigin = "origin"

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
	// HourlySince is the probe that began the hourly stage, nil before it.
	HourlySince *time.Time `json:"hourly_since,omitempty"`
}

// written reports whether w is one the drain writes: a known kind and at least
// one probe.
func (w *DrainOriginWait) written() bool {
	return w == nil || slices.Contains(drainOriginKinds, w.Kind) && w.Probes > 0
}

// DrainPreview is `abcd drain --dry-run`: the plan, and the view of origin it
// was read from. Origin is nil in two cases, and the plan is then the
// checkout's own ledger: in a checkout with no remote named origin, where
// OriginUnread is empty, since there is no origin to read; and when origin's
// default branch as last fetched cannot be read (this checkout names none, or
// its commit does not resolve), where OriginUnread says why.
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
			// The order is this copy's as its next move draws it, the key
			// read and never minted: the dry run writes nothing.
			plan, origin, err := planFromOrigin(repoRoot, v, now, drainSpread(repoRoot, claim.OwnCopyKey(repoRoot)))
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

// drainSpread is the key a drain on origin draws its own order within each
// group from (capture.DrainPlanRequest.Spread): the holder this copy resolves
// to with no network act, and the copy's key, so two accounts, and one
// person's two copies, take a group in different orders, each the same at
// every move. "" when neither can be told, which takes each group oldest first.
func drainSpread(repoRoot, copyKey string) string {
	holder := strings.ToLower(claim.ReaderHolder(repoRoot, claim.Options{}))
	if holder == "" && copyKey == "" {
		return ""
	}
	return holder + "\x00" + copyKey
}

// treeFiles reports which of the paths named v's commit holds as a file (a
// blob: a regular file or a symbolic link), through one `git cat-file
// --batch-check` asking each path on its own line. A directory is not a file,
// nor is a path the commit does not hold, so an issue naming either sorts
// with those naming a directory or no path; and since each path is resolved
// on its own, a directory asked beside a path beneath it is still a
// directory, and the answer is one line per path, whatever lies beneath.
func treeFiles(repoRoot string, v originView) func([]string) (map[string]bool, error) {
	return func(paths []string) (map[string]bool, error) {
		out := map[string]bool{}
		types, err := gitutil.TreePathTypes(repoRoot, v.commit, paths)
		if err != nil {
			return nil, err
		}
		for i, kind := range types {
			if kind == "blob" {
				out[paths[i]] = true
			}
		}
		return out, nil
	}
}

// planFromOrigin plans the drain from the ledger v's commit holds, each group
// in spread's order, and describes the view at now.
func planFromOrigin(repoRoot string, v originView, now time.Time, spread string) (capture.DrainPlan, *DrainOrigin, error) {
	ledger, err := treeLedger(repoRoot, v)
	if err != nil {
		return capture.DrainPlan{}, nil, err
	}
	plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot, Ledger: ledger, Spread: spread, Files: treeFiles(repoRoot, v)})
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

// maxLedgerBatch caps what one move's read of the open records may answer:
// each distinct record once, headers included. A ledger past it is refused,
// not buffered.
const maxLedgerBatch = 64 << 20

// treeLedger reads the issue ledger's status folders out of v's commit: every
// file's name, and the content of each record in open/. The records are read
// through one `git cat-file --batch`, so a move starts the same few processes
// however long the ledger is; each is first judged as baseTree.blob judges
// one (a regular file, within issueschema.RecordReadLimit), and only those it
// would read are asked for, each distinct object once: records that name one
// blob share its one read and its one text. What the batch answers is bounded
// by maxLedgerBatch, and a ledger past it is refused before anything is read.
func treeLedger(repoRoot string, v originView) (*capture.TreeLedger, error) {
	at := baseTree{root: repoRoot, sha: v.commit}
	source := gitutil.ShortRef(v.ref) + " at " + shortSHA(v.commit)
	entries, err := at.list(recordid.IssuesRelDir+"/", true)
	if err != nil {
		return nil, fmt.Errorf("reading the ledger of %s: %w", source, err)
	}
	l := &capture.TreeLedger{Source: source}
	// want maps each file read to its object's place in the batch.
	want := map[int]int{}
	asked := map[string]int{}
	var objects []string
	budget := 0
	for _, e := range entries {
		rest, _ := strings.CutPrefix(e.path, recordid.IssuesRelDir+"/")
		folder, name, ok := strings.Cut(rest, "/")
		if !ok || strings.Contains(name, "/") {
			continue
		}
		f := capture.LedgerFile{Folder: folder, Path: e.path}
		if folder == string(capture.StateOpen) && strings.HasPrefix(name, "iss-") {
			if err := at.readable(e, issueschema.RecordReadLimit); err != nil {
				f.Unread = err.Error()
			} else {
				k, seen := asked[e.object]
				if !seen {
					k = len(objects)
					asked[e.object], objects = k, append(objects, e.object)
					// The answer is the header and the content, as the
					// listing sized it, and the newline after.
					budget += len(e.object) + 32 + int(e.size)
				}
				want[len(l.Files)] = k
			}
		}
		l.Files = append(l.Files, f)
	}
	if budget > maxLedgerBatch {
		return nil, fmt.Errorf("reading the ledger of %s: its open records come to %d bytes, past the %d MiB one move reads; nothing was read", source, budget, maxLedgerBatch>>20)
	}
	got, err := gitutil.CatFileBatch(repoRoot, objects, budget)
	if err != nil {
		return nil, fmt.Errorf("reading the ledger of %s: %w", source, err)
	}
	// One text per object, which every file naming it shares.
	texts := make([]string, len(got))
	for k, o := range got {
		if !o.Missing && o.Type == "blob" {
			// Trimmed at the end as a single read's answer is (gitutil.RunCapped).
			texts[k] = strings.TrimRight(string(o.Content), " \t\r\n")
		}
	}
	for i, k := range want {
		switch o := got[k]; {
		case o.Missing:
			l.Files[i].Unread = "the base holds no object " + o.Name
		case o.Type != "blob":
			l.Files[i].Unread = fmt.Sprintf("the base holds it as a %s, not a regular file", o.Type)
		default:
			l.Files[i].Text = texts[k]
		}
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
			w.Since, w.Probes, w.HourlySince = prev.Since, prev.Probes+1, prev.HourlySince
		}
		// The schedule as the outage gate keeps it: the first failed move
		// opens the wait, the next three wait one minute, five and ten, the
		// probe after them begins the hourly stage, and a probe failing
		// HourlyFor into that stage gives up.
		s := implement.DefaultSchedule
		failures := w.Probes - 1
		if w.HourlySince == nil && failures >= len(s.Waits) {
			hourly := now
			w.HourlySince = &hourly
		}
		st.OriginWait = w
		if w.HourlySince != nil && s.GivesUp(*w.HourlySince, now) {
			st.Stopped, st.EndedAt = DrainStoppedOrigin, &now
			return capture.RuleOnlyPlan(rule), nil, fmt.Sprintf("%s, so the move plans nothing, and the drain gives up on %s as the outage gate gives up: %s has not been read since %s, %d probe(s) failed, hourly for %d hours since %s; the drain stops here with %d lane(s) opened; the next `abcd drain` begins a new drain, waiting again if %s still cannot be read",
				why, claim.Remote, claim.Remote, w.Since.UTC().Format(time.RFC3339), w.Probes, int(s.HourlyFor/time.Hour), w.HourlySince.UTC().Format(time.RFC3339), len(st.Lanes), claim.Remote), "", true, nil
		}
		w.NextProbeAt = s.Next(now, failures)
		at := w.NextProbeAt.UTC().Format(time.RFC3339)
		return capture.RuleOnlyPlan(rule), nil, fmt.Sprintf("%s, so the move plans nothing: it never plans from what this checkout last saw; it waits on %s as for an outage (probe %d since %s) and looks again at or after %s; run `abcd drain` again then",
			why, claim.Remote, w.Probes, w.Since.UTC().Format(time.RFC3339), at), "", true, nil
	}
	// The copy's key is minted before the first plan draws the order from
	// it, so the order a move asks a judgement in is the one the next move
	// keeps (the first reservation would otherwise mint it in between).
	key, err := claim.CopyKey(repoRoot)
	if err != nil {
		return plan, nil, "", "", false, err
	}
	if plan, origin, err = planFromOrigin(repoRoot, v, now, drainSpread(repoRoot, key)); err != nil {
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
