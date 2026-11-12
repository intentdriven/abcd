package landings

// forge.go is the reader's one door to the forge (spc-2610301950480609
// approach 6): every row that names a pull request is asked about at most
// once, in GraphQL documents of up to MaxBatch aliased pullRequest nodes, run
// as `gh api graphql` through the landing's own client (landing.GHAnswerContext,
// so each call is bounded by landing.ForgeTimeout and the caller's ctx, its
// environment scrubbed and its output bounded). Each node asks only what this
// step reads: who opened the pull request, its head, how it merged, and its
// reviews, with its commits only for a row that merged with one parent. The
// revert trace (revert.go) asks a squashed revert's body apart, in a document
// of its own (readBodies), after every row is classed, so no answer here
// depends on a body. What
// the forge said of a row's pull request is kept in the local tier's cache as
// each batch returns, keyed by the row's commit, so a later listing asks only
// about pull requests it has not read, and a listing whose time runs out keeps
// what it read of what cannot change: an answer the reader accepts for its
// row, one saying the pull request merged as another commit, and the forge's
// word that a number names no pull request. An unread answer, one whose
// reviews page is not whole, and one saying the pull request has not merged or
// naming no merge commit, is never kept.
//
// The forge's answer is judged a pull request at a time: an error that names
// one pull request leaves that one unread and the others answered, and a
// number the forge says is no pull request reads so for its row alone. A
// refusal that concerns the login or the repository rather than one pull
// request stops the asking: a missing client, the rate limit, credentials or
// scopes the forge does not take, an error for the repository itself, judged
// from the answer's typed fields where it carries them and from the client's
// text where it does not. An answer that passes the client's output bound is
// asked again in halves, down to one pull request, which is asked once more
// without its commits where it asked them and the listing's bound on asking
// again allows; an answer still past the bound is too large to read. One that
// carries errors naming no pull request is asked again along one chain of
// halves, and when both halves of a batch fail so, neither is asked further. A
// listing spends at most maxHalvingCalls calls asking again. Every row whose
// answer was not read, whatever the cause, reads as "forge unread", never
// guessed, no mode attends it (classify), and the reason is on the row and in
// the listing's note.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// MaxBatch is the most pull requests one forge document asks about.
const MaxBatch = 20

// maxPage is the page of reviews and of commits a pull request's node reads:
// the forge's maximum.
const maxPage = 100

// CacheRelDir is the forge cache, in the listing checkout's local tier: one
// file per pull request, pr-<N>.json. It holds the forge logins a row prints,
// so it lives in the gitignored tier and never in a committed file.
const CacheRelDir = landing.TierRelDir + "/landings"

// maxHalvingCalls bounds the calls one listing spends asking batches again in
// halves. A variable only so a test can hold a listing to a small one.
var maxHalvingCalls = 64

// maxCacheBytes bounds one cache file: an answer is bounded by the forge
// client's own output bound.
const maxCacheBytes = 2 * landing.MaxOutput

// prSelection is what the batch reads of each pull request, and
// commitsSelection what it adds for a row that merged with one parent. Every
// field is pinned in internal/ghcontract (graphql-fields.txt).
const (
	prSelection = `number author{login} headRefOid mergedAt mergeCommit{oid} ` +
		`reviews(last:100){totalCount nodes{author{__typename login} authorAssociation authorCanPushToRepository state commit{oid} submittedAt}}`
	commitsSelection = ` commits(first:100){totalCount nodes{commit{oid message}}}`
)

// prAsk is one row's question: its pull request, its commit, and whether the
// row needs the pull request's commits (a row with one parent does).
type prAsk struct {
	pr      int
	commit  string
	commits bool
}

// batchQuery is the GraphQL document asking about asks, each as the aliased
// node p<N>. The owner and name are its variables $o and $r.
//
// It stays unexported for this step: step 4 exports it when it wires the verb.
func batchQuery(asks []prAsk) string {
	var b strings.Builder
	b.WriteString(`query($o:String!,$r:String!){repository(owner:$o,name:$r){`)
	for _, a := range asks {
		sel := prSelection
		if a.commits {
			sel += commitsSelection
		}
		fmt.Fprintf(&b, "p%d:pullRequest(number:%d){%s} ", a.pr, a.pr, sel)
	}
	b.WriteString("}}")
	return b.String()
}

// batchArgs are the forge client's arguments for batchQuery on asks, for the
// repository the forge client resolves for the checkout ({owner} and {repo},
// which gh fills in).
func batchArgs(asks []prAsk) []string {
	return []string{"api", "graphql", "-F", "o={owner}", "-F", "r={repo}", "-f", "query=" + batchQuery(asks)}
}

type forgeLogin struct {
	Login string `json:"login"`
}

// forgeActor is a review's author: its concrete type (User, Bot, ...) and
// login.
type forgeActor struct {
	Typename string `json:"__typename"`
	Login    string `json:"login"`
}

type forgeOid struct {
	Oid string `json:"oid"`
}

type forgeReview struct {
	Author            *forgeActor `json:"author"`
	AuthorAssociation string      `json:"authorAssociation"`
	AuthorCanPush     bool        `json:"authorCanPushToRepository"`
	State             string      `json:"state"`
	Commit            *forgeOid   `json:"commit"`
	SubmittedAt       string      `json:"submittedAt"`
}

type forgeCommit struct {
	Commit struct {
		Oid     string `json:"oid"`
		Message string `json:"message"`
	} `json:"commit"`
}

// prAnswer is the forge's answer for one pull request, as asked and as cached.
type prAnswer struct {
	Number      int         `json:"number"`
	Author      *forgeLogin `json:"author"`
	HeadRefOid  string      `json:"headRefOid"`
	MergedAt    *string     `json:"mergedAt"`
	MergeCommit *forgeOid   `json:"mergeCommit"`
	Reviews     struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []forgeReview `json:"nodes"`
	} `json:"reviews"`
	Commits struct {
		TotalCount int           `json:"totalCount"`
		Nodes      []forgeCommit `json:"nodes"`
	} `json:"commits"`
	// commitsUnread marks an answer asked without the commits its row wanted
	// (asker.alone): its commits are not read, and it is not kept.
	commitsUnread bool
}

// loginRe is a person's forge login's shape, any other no reviewer: letters,
// digits and hyphens, and the underscore an enterprise-managed account's login
// carries, at most 39 long. A bot's or an app's login is refused by the type
// the review node names, not by this shape. The merge gate's own name check
// (landing's ownerName) still takes no underscore, so the gate reads such a
// reviewer's permission as unverified where this reader counts them.
var loginRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,38}$`)

// writeAssociations are the reviewer associations the forge gives an account
// that holds the repository or was let into it: an owner, a member of the
// owning organisation, and a collaborator.
var writeAssociations = map[string]bool{"OWNER": true, "MEMBER": true, "COLLABORATOR": true}

// check says why a is not an answer for pull request n, "" when it is: the
// number asked, a head in an object name's shape, and pages no longer than
// asked.
func (a *prAnswer) check(n int) string {
	switch {
	case a.Number != n:
		return fmt.Sprintf("the forge's answer for pull request #%d names another", n)
	case a.HeadRefOid != "" && !gitutil.IsFullSHA(a.HeadRefOid):
		// The merge commit needs no shape check here: landedAs compares it
		// with the row's full object name, so no other shape answers a row.
		return fmt.Sprintf("the forge's answer for pull request #%d names a head by no full object name", n)
	case len(a.Reviews.Nodes) > maxPage || len(a.Commits.Nodes) > maxPage ||
		a.Reviews.TotalCount < len(a.Reviews.Nodes) || a.Commits.TotalCount < len(a.Commits.Nodes):
		return fmt.Sprintf("the forge's answer for pull request #%d holds pages it was not asked for", n)
	}
	return ""
}

// landedAs says why a does not answer for the row commit, "" when it does: the
// forge must say pull request n merged, as commit. A subject is text whoever
// made the commit typed, so the forge's answer is read for a row only when the
// forge says the pull request merged as that row's commit.
func (a *prAnswer) landedAs(n int, commit string) string {
	switch {
	case a.MergedAt == nil:
		return fmt.Sprintf("the forge says pull request #%d has not merged, so it does not answer for %s", n, commit[:12])
	case a.MergeCommit == nil || a.MergeCommit.Oid != commit:
		return fmt.Sprintf("the forge says pull request #%d did not merge as %s, so it does not answer for it", n, commit[:12])
	}
	return ""
}

// lasting says whether a, the forge's answer for the row commit, is one a
// later listing may read again without asking: one saying the pull request
// merged as another commit, or one the row accepts whose reviews page is
// whole and whose commits were asked where the row wanted them. An answer
// saying the pull request has not merged, or naming no merge commit, can
// change once the forge marks the merge, so it is asked again.
func (a *prAnswer) lasting(commit string) bool {
	switch {
	case a.MergedAt == nil || a.MergeCommit == nil:
		return false
	case a.MergeCommit.Oid != commit:
		return true
	}
	return a.whole() == "" && !a.commitsUnread
}

// whole says why a's reviews cannot be judged, "" when the page holds them
// all.
func (a *prAnswer) whole() string {
	if a.Reviews.TotalCount > len(a.Reviews.Nodes) {
		return fmt.Sprintf("pull request #%d has more than %d reviews, a later one may sit beyond the page read, so none counts", a.Number, maxPage)
	}
	return ""
}

// approval is the review of head that counts in a, by the merge gate's reading
// of a page (landing.LatestReviews, then Review.Approves): each reviewer's
// latest submitted review, which must approve head and be by someone other than
// the pull request's author. Where the merge gate asks the forge for the
// reviewer's permission, the reader takes what the same review node says, so
// the forge is asked once per pull request: the reviewer must be a person (a
// User, never a bot or an app), associated with the repository as its owner, a
// member or a collaborator, and able to push to it. A review submitted after
// the pull request merged counts for nothing, and so does one whose times do
// not read. A reviews page the forge says is not whole counts none, and says
// why.
//
// Two limits stand. The association and the push access are the forge's
// answer as of the read, not as of the review: a reviewer who has since lost
// access no longer counts on a fresh read, one who has since gained it does,
// and a cached answer keeps what was true when it was read. And the forge
// gives a machine account of type User the same answer as a person, so one
// with write access counts as a person.
func (a *prAnswer) approval(head string) (Approval, string) {
	if why := a.whole(); why != "" {
		return Approval{}, why
	}
	if a.MergedAt == nil {
		return Approval{}, ""
	}
	merged, err := time.Parse(time.RFC3339, *a.MergedAt)
	if err != nil {
		return Approval{}, ""
	}
	author := ""
	if a.Author != nil {
		author = a.Author.Login
	}
	page := make([]landing.Review, len(a.Reviews.Nodes))
	for i, r := range a.Reviews.Nodes {
		page[i] = landing.Review{State: r.State}
		if r.Author != nil {
			page[i].Login = r.Author.Login
		}
		if r.Commit != nil {
			page[i].Commit = r.Commit.Oid
		}
	}
	for _, lr := range landing.LatestReviews(page) {
		if _, ok := lr.Approves(head, author); !ok {
			continue
		}
		r := a.Reviews.Nodes[lr.Index]
		at, err := time.Parse(time.RFC3339, r.SubmittedAt)
		if err != nil || at.After(merged) || !loginRe.MatchString(lr.Login) || r.Author.Typename != "User" ||
			!writeAssociations[r.AuthorAssociation] || !r.AuthorCanPush {
			continue
		}
		return Approval{Seen: true, By: lr.Login, At: at.UTC(), Source: SourceForge}, ""
	}
	return Approval{}, ""
}

// forgeAnswers are readForge's answers by pull request, why each one it has
// none for was not read, which numbers the forge says name no pull request,
// and why an answer could not be cached.
type forgeAnswers struct {
	got      map[int]*prAnswer
	why      map[int]string
	notPR    map[int]bool
	cacheWhy string
	// left is what the listing's bound on asking again (maxHalvingCalls)
	// holds once the answers are read, for the body query (readBodies).
	left int
}

// readForge answers asks from the cache where it holds the row's commit, and
// asks the forge about the rest, at most once each, MaxBatch at a time (a
// batch answered for none of its pull requests is asked again in halves). A
// missing forge client, a refusal that would refuse every batch, or an ended
// ctx stops the asking, and every pull request not yet answered reads as
// unread with the reason.
func readForge(ctx context.Context, repoRoot string, forge landing.ForgeCall, asks []prAsk) forgeAnswers {
	fa := forgeAnswers{got: map[int]*prAnswer{}, why: map[int]string{}, notPR: map[int]bool{}}
	byPR := map[int]*prAsk{}
	var want []*prAsk
	for _, a := range asks {
		if b := byPR[a.pr]; b != nil {
			b.commits = b.commits || a.commits
			continue
		}
		byPR[a.pr] = &a
		want = append(want, &a)
	}
	root, rootErr := os.OpenRoot(repoRoot)
	if rootErr == nil {
		defer root.Close()
	}
	cached, cachedNotPR, cacheWhy := readCache(ctx, root, repoRoot, want)
	fa.cacheWhy = cacheWhy
	var ask []prAsk
	for _, a := range want {
		switch {
		case cached[a.pr] != nil:
			fa.got[a.pr] = cached[a.pr]
		case cachedNotPR[a.pr]:
			fa.notPR[a.pr], fa.why[a.pr] = true, notPRWhy(a.pr)
		default:
			ask = append(ask, *a)
		}
	}
	q := &asker{ctx: ctx, forge: forge, fa: &fa, halvings: maxHalvingCalls}
	if len(ask) > 0 && cacheWhy == "" {
		// Judged once, before the first batch, so each batch's answers are
		// kept as it returns.
		if ok, why := cacheWritable(ctx, root, rootErr, repoRoot); ok {
			q.cache = root
		} else {
			fa.cacheWhy = why
		}
	}
	for i := 0; i < len(ask); i += MaxBatch {
		q.ask(ask[i:min(i+MaxBatch, len(ask))])
	}
	fa.left = q.halvings
	return fa
}

// asker asks the forge about batches for readForge, asking again in halves a
// batch the forge answered for none of, within halvings, stopping on a refusal
// that would refuse every batch, and keeping each batch's answers in cache (nil
// for no cache) as it returns.
type asker struct {
	ctx      context.Context
	forge    landing.ForgeCall
	fa       *forgeAnswers
	stop     string
	halvings int
	cache    *os.Root
}

// call asks the forge about batch unless the asking has stopped, and reports
// whether the answer is there to read; a batch it does not ask about, or whose
// answer stops the asking, reads as unread.
func (q *asker) call(batch []prAsk) (batchResult, bool) {
	if q.stop == "" && q.ctx.Err() != nil {
		q.stop = "the listing's time ran out before the forge was asked: " + whyText(q.ctx.Err())
	}
	if q.stop != "" {
		q.unread(batch, q.stop)
		return batchResult{}, false
	}
	r := askBatch(q.forge, batch)
	if r.stop {
		q.stop = r.whole
		q.unread(batch, r.whole)
		return r, false
	}
	return r, true
}

func (q *asker) ask(batch []prAsk) {
	r, ok := q.call(batch)
	switch {
	case !ok:
	case r.whole != "" && r.split:
		q.halve(batch, r.whole, r.size)
	case r.whole != "":
		q.unread(batch, r.whole)
	default:
		q.take(batch, r)
	}
}

// halve asks batch, which the forge answered for none of (why; size when for
// the output bound), again in two halves. A half that passed the output bound
// is halved again in turn, down to one pull request, which alone settles; one
// that again carries errors naming no pull request is followed alone, as one
// chain, and when both halves fail other than for their size the failure is
// not one pull request's, so neither is asked further for it, while a half
// past the bound is still halved, so the others in it are read. Each call
// spends one of the listing's halvings, and with none left the batch reads as
// unread.
func (q *asker) halve(batch []prAsk, why string, size bool) {
	switch {
	case len(batch) < 2 && size:
		q.alone(batch, why)
		return
	case len(batch) < 2:
		q.unread(batch, why)
		return
	case q.halvings < 2:
		q.unread(batch, why+"; it is not asked again in halves, the listing's bound on asking again being spent")
		return
	}
	q.halvings -= 2
	type failed struct {
		batch []prAsk
		why   string
		size  bool
	}
	var fails []failed
	for _, h := range [][]prAsk{batch[:len(batch)/2], batch[len(batch)/2:]} {
		r, ok := q.call(h)
		switch {
		case !ok:
		case r.whole != "" && r.split:
			fails = append(fails, failed{h, r.whole, r.size})
		case r.whole != "":
			q.unread(h, r.whole)
		default:
			q.take(h, r)
		}
	}
	if len(fails) == 2 && !(fails[0].size && fails[1].size) {
		for _, f := range fails {
			if f.size {
				q.halve(f.batch, f.why, true)
				continue
			}
			q.unread(f.batch, f.why+"; neither half of the batch answered, so it is not asked again")
		}
		return
	}
	for _, f := range fails {
		q.halve(f.batch, f.why, f.size)
	}
}

// alone settles batch, one pull request whose answer passed the output bound
// asked alone (why). Where it was asked with its commits and a halving is
// left, it is asked once more without them, spending that halving, and that
// answer is read with its commits unread. Otherwise, or past the bound still,
// it is too large to read and reads as unread, so it is not asked again in the
// listing.
func (q *asker) alone(batch []prAsk, why string) {
	if a := batch[0]; len(batch) == 1 && a.commits && q.halvings >= 1 {
		q.halvings--
		lean := a
		lean.commits = false
		r, ok := q.call([]prAsk{lean})
		switch {
		case !ok:
			return
		case r.whole == "":
			if got := r.got[a.pr]; got != nil {
				got.commitsUnread = true
			}
			q.take(batch, r)
			return
		case !r.size:
			q.unread(batch, r.whole)
			return
		}
		why = r.whole
	}
	q.unread(batch, why+", asked alone, so it is too large to read")
}

// take reads one answered batch's answers into the listing's, and keeps what
// the forge said of each pull request in the cache.
func (q *asker) take(batch []prAsk, r batchResult) {
	for _, a := range batch {
		switch {
		case r.got[a.pr] != nil:
			q.fa.got[a.pr] = r.got[a.pr]
		case r.notPR[a.pr]:
			q.fa.notPR[a.pr] = true
			q.fa.why[a.pr] = r.why[a.pr]
		case r.why[a.pr] != "":
			q.fa.why[a.pr] = r.why[a.pr]
		default:
			q.fa.why[a.pr] = fmt.Sprintf("the forge's answer leaves out pull request #%d", a.pr)
		}
		q.keep(a)
	}
}

// keep writes what the forge said of a's pull request to the cache where it
// cannot change (prAnswer.lasting): an answer the row accepts, one saying it
// merged as another commit, or the forge's word that the number names no pull
// request; never an unread answer, nor one a later listing asks again. The
// first write that fails stops the keeping and says why.
func (q *asker) keep(a prAsk) {
	if q.cache == nil {
		return
	}
	e := cacheEntry{Commit: a.commit}
	switch got := q.fa.got[a.pr]; {
	case q.fa.notPR[a.pr]:
		e.NotPR = true
	case got == nil, !got.lasting(a.commit):
		return
	default:
		e.PR = *got
	}
	if why := writeCache(q.cache, a.pr, e); why != "" {
		q.fa.cacheWhy, q.cache = why, nil
	}
}

func (q *asker) unread(batch []prAsk, why string) {
	for _, a := range batch {
		q.fa.why[a.pr] = why
	}
}

// batchAnswer is the forge's answer to one batchQuery. An answer may carry
// errors beside its data, and gh exits non-zero on them while writing the
// answer whole, so the answer is read beside the call's error.
type batchAnswer struct {
	Data *struct {
		Repository map[string]json.RawMessage `json:"repository"`
	} `json:"data"`
	Errors []forgeError `json:"errors"`
	// Status is the HTTP status the forge writes in the body of a refusal
	// that never reached the query (a string or a number).
	Status json.RawMessage `json:"status"`
}

// forgeError is one GraphQL error: its type, the path into the answer it
// names (["repository", "p12"] for one pull request's node), and nothing
// else, since its message is the forge's prose.
type forgeError struct {
	Type string `json:"type"`
	Path []any  `json:"path"`
}

// batchResult is one batch's answer: the pull requests answered, why each
// other was not, which the forge says are no pull request; or, for the whole
// batch, why none was answered (whole), whether asking about fewer at a time
// may answer (split, and size when it was the output bound), and whether to
// stop asking (stop).
type batchResult struct {
	got   map[int]*prAnswer
	why   map[int]string
	notPR map[int]bool
	whole string
	split bool
	size  bool
	stop  bool
}

// stopsAsking says whether a refusal's text is one every later batch would
// meet too: the forge's rate limit, or credentials it does not take. It reads
// the forge client's own wording, for a refusal whose answer carries no typed
// field (refusal).
func stopsAsking(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "rate limit") || strings.Contains(t, "http 401") || strings.Contains(t, "http 403") ||
		strings.Contains(t, "bad credentials")
}

// accessTypes are the GraphQL error types that, on an error naming no pull
// request, refuse the login or the repository: every batch would meet them.
var accessTypes = map[string]bool{"FORBIDDEN": true, "INSUFFICIENT_SCOPES": true, "NOT_FOUND": true, "UNAUTHORIZED": true}

// refusal says, from the answer's typed fields, why a would refuse every
// batch, "" when nothing in it does: an HTTP status of 401 or 403 in the body,
// the rate limit, an error for the repository itself (its path is the
// repository alone), an error naming no pull request whose type refuses the
// login or the repository, or data whose repository is null.
func refusal(a batchAnswer) string {
	if s := strings.Trim(string(a.Status), `"`); s == "401" || s == "403" {
		return "the forge answered HTTP " + s
	}
	for _, e := range a.Errors {
		switch {
		case e.Type == "RATE_LIMITED":
			return "for its rate limit"
		case len(e.Path) == 1 && e.Path[0] == "repository":
			return "for the repository"
		case len(e.Path) == 0 && accessTypes[e.Type]:
			return "with an error of type " + e.Type + " that names no pull request"
		}
	}
	if a.Data != nil && a.Data.Repository == nil {
		return "it names no repository"
	}
	return ""
}

// askBatch asks the forge about batch in one document and reads the answer a
// pull request at a time.
func askBatch(forge landing.ForgeCall, batch []prAsk) batchResult {
	r, nodes := askNodes(forge, batch, batchArgs(batch), 0)
	for _, b := range batch {
		raw, ok := nodes[b.pr]
		if !ok {
			continue
		}
		var pr prAnswer
		if json.Unmarshal(raw, &pr) != nil {
			r.why[b.pr] = fmt.Sprintf("the forge's answer for pull request #%d does not parse", b.pr)
			continue
		}
		if why := pr.check(b.pr); why != "" {
			r.why[b.pr] = why
			continue
		}
		r.got[b.pr] = &pr
	}
	return r
}

// askNodes asks the forge args, one document about batch, and judges the
// answer as askBatch reads it: whole, for a refusal, a failure or an answer
// past the forge client's output bound (or past limit, where it is above 0),
// and otherwise a pull request at a time, giving each node no error names and
// the forge did not answer null for, unparsed.
func askNodes(forge landing.ForgeCall, batch []prAsk, args []string, limit int) (batchResult, map[int]json.RawMessage) {
	r := batchResult{got: map[int]*prAnswer{}, why: map[int]string{}, notPR: map[int]bool{}}
	names := make([]string, len(batch))
	inBatch := map[string]int{}
	for i, a := range batch {
		names[i] = "#" + strconv.Itoa(a.pr)
		inBatch["p"+strconv.Itoa(a.pr)] = a.pr
	}
	about := "pull requests " + strings.Join(names, ", ")
	out, err := forge(args...)
	switch {
	case errors.Is(err, landing.ErrNoForgeClient):
		r.whole, r.stop = "the forge could not be asked about "+about+": "+whyText(err), true
		return r, nil
	case errors.Is(err, landing.ErrOutputCap), limit > 0 && len(out) > limit:
		// A failed exit whose output also passes the bound is halved before
		// its answer is read as a refusal; that order follows from the bound
		// being judged first, and is not a rule.
		r.whole, r.split, r.size = "the forge's answer about "+about+" passes the forge client's output bound", true, true
		return r, nil
	}
	var a batchAnswer
	decoded := json.Unmarshal([]byte(out), &a) == nil
	if decoded {
		if why := refusal(a); why != "" {
			r.whole, r.stop = "the forge refused "+about+" "+why, true
			return r, nil
		}
	}
	if err != nil && stopsAsking(err.Error()) {
		r.whole, r.stop = "the forge refused "+about+": "+whyText(err), true
		return r, nil
	}
	switch {
	case !decoded && err != nil, decoded && len(a.Errors) == 0 && err != nil:
		r.whole = "the forge could not be asked about " + about + ": " + whyText(err)
		return r, nil
	case !decoded:
		r.whole = "the forge's answer about " + about + " does not parse"
		return r, nil
	case a.Data == nil:
		// Errors with no data that refuse nothing (refusal) may come of the
		// document's cost or of one pull request; asked about fewer at a
		// time, the others may answer.
		r.whole, r.split = "the forge's answer about "+about+" carries no data", len(a.Errors) > 0
		return r, nil
	}
	for _, e := range a.Errors {
		alias, _ := pathAlias(e.Path)
		n, ok := inBatch[alias]
		if !ok {
			r.whole, r.split = fmt.Sprintf("the forge's answer about %s carries errors that name no one pull request", about), true
			return r, nil
		}
		switch {
		case e.Type == "NOT_FOUND" && len(e.Path) == 2:
			r.notPR[n] = true
			r.why[n] = notPRWhy(n)
		case r.why[n] == "":
			r.why[n] = fmt.Sprintf("the forge's answer for pull request #%d carries an error", n)
		}
	}
	nodes := map[int]json.RawMessage{}
	for _, b := range batch {
		if r.why[b.pr] != "" {
			continue
		}
		raw, ok := a.Data.Repository["p"+strconv.Itoa(b.pr)]
		if !ok || string(raw) == "null" {
			continue
		}
		nodes[b.pr] = raw
	}
	return r, nodes
}

// bodySelection is what the body query reads of a pull request: its number
// and merge commit, so the answer is read only for the row its own answer
// confirmed, its body, for the forge's revert line, and the name of the
// repository it merged into, which that line's owner/repo form is checked
// against. Every field is pinned in internal/ghcontract (graphql-fields.txt).
const bodySelection = `number mergeCommit{oid} body baseRepository{nameWithOwner}`

// maxBodyAnswerBytes is the body query's own output bound: one document's
// answer past it is asked again in halves, and one pull request's alone past
// it is not read. It holds a body at the forge's bound of 65,536 characters
// with each written as a six-byte escape, and the rest of its node.
const maxBodyAnswerBytes = 512 << 10

// bodyQuery is the GraphQL document asking the body of each of asks, as the
// aliased node p<N>, in the repository batchQuery asks about.
func bodyQuery(asks []prAsk) string {
	var b strings.Builder
	b.WriteString(`query($o:String!,$r:String!){repository(owner:$o,name:$r){`)
	for _, a := range asks {
		fmt.Fprintf(&b, "p%d:pullRequest(number:%d){%s} ", a.pr, a.pr, bodySelection)
	}
	b.WriteString("}}")
	return b.String()
}

func bodyArgs(asks []prAsk) []string {
	return []string{"api", "graphql", "-F", "o={owner}", "-F", "r={repo}", "-f", "query=" + bodyQuery(asks)}
}

// forgeRepo is the repository a pull request merged into, by its owner/name.
type forgeRepo struct {
	NameWithOwner string `json:"nameWithOwner"`
}

// prBody is the forge's answer to the body query for one pull request. It is
// never kept: the cache holds what batchQuery reads and nothing else.
type prBody struct {
	Number         int        `json:"number"`
	MergeCommit    *forgeOid  `json:"mergeCommit"`
	Body           string     `json:"body"`
	BaseRepository *forgeRepo `json:"baseRepository"`
}

// bodyAnswers are readBodies' answers by pull request, and why each one it
// has none for was not read.
type bodyAnswers struct {
	got map[int]*prBody
	why map[int]string
}

// readBodies asks the forge the body of each of asks (pull requests whose
// answer the listing read and accepted for their row), MaxBatch to a
// document, under ctx, within calls more calls asking again (what the
// listing's bound on asking again holds once its answers are read). A
// document past the output bound is asked again in halves; one that fails
// otherwise, or a refusal, leaves its pull requests' bodies unread, and a
// refusal or an ended ctx stops the asking. An answer is read only where it
// names the pull request asked and says it merged as the row's commit. It
// writes nothing.
func readBodies(ctx context.Context, forge landing.ForgeCall, asks []prAsk, calls int) bodyAnswers {
	b := &bodyAsker{ctx: ctx, forge: forge, calls: calls, out: bodyAnswers{got: map[int]*prBody{}, why: map[int]string{}}}
	for i := 0; i < len(asks); i += MaxBatch {
		b.ask(asks[i:min(i+MaxBatch, len(asks))])
	}
	return b.out
}

type bodyAsker struct {
	ctx   context.Context
	forge landing.ForgeCall
	calls int
	stop  string
	out   bodyAnswers
}

func (b *bodyAsker) unread(batch []prAsk, why string) {
	for _, a := range batch {
		b.out.why[a.pr] = why
	}
}

func (b *bodyAsker) ask(batch []prAsk) {
	if b.stop == "" && b.ctx.Err() != nil {
		b.stop = "the listing's time ran out before the forge was asked: " + whyText(b.ctx.Err())
	}
	if b.stop != "" {
		b.unread(batch, b.stop)
		return
	}
	r, nodes := askNodes(b.forge, batch, bodyArgs(batch), maxBodyAnswerBytes)
	switch {
	case r.stop:
		b.stop = r.whole
		b.unread(batch, r.whole)
		return
	case r.size && len(batch) == 1:
		b.unread(batch, r.whole+", asked alone, so it is too large to read")
		return
	case r.size && b.calls >= 2:
		b.calls -= 2
		b.ask(batch[:len(batch)/2])
		b.ask(batch[len(batch)/2:])
		return
	case r.size:
		b.unread(batch, r.whole+"; it is not asked again in halves, the listing's bound on asking again being spent")
		return
	case r.whole != "":
		b.unread(batch, r.whole)
		return
	}
	for _, a := range batch {
		raw, ok := nodes[a.pr]
		var p prBody
		switch {
		case r.why[a.pr] != "":
			b.out.why[a.pr] = r.why[a.pr]
		case !ok:
			b.out.why[a.pr] = fmt.Sprintf("the forge's answer leaves out pull request #%d", a.pr)
		case json.Unmarshal(raw, &p) != nil:
			b.out.why[a.pr] = fmt.Sprintf("the forge's answer for pull request #%d does not parse", a.pr)
		case p.Number != a.pr:
			b.out.why[a.pr] = fmt.Sprintf("the forge's answer for pull request #%d names another", a.pr)
		case p.MergeCommit == nil || p.MergeCommit.Oid != a.commit:
			b.out.why[a.pr] = fmt.Sprintf("the forge's body answer for pull request #%d does not say it merged as %s", a.pr, a.commit[:12])
		default:
			b.out.got[a.pr] = &p
		}
	}
}

// notPRWhy is the reason a row reads for number n the forge says names no
// pull request.
func notPRWhy(n int) string { return fmt.Sprintf("the forge says #%d names no pull request", n) }

// pathAlias is the pull request node an error's path names: its second
// element under "repository".
func pathAlias(path []any) (string, bool) {
	if len(path) < 2 || path[0] != "repository" {
		return "", false
	}
	s, ok := path[1].(string)
	return s, ok
}

// cacheEntry is one cache file: the answer, or the forge's word that the
// number names no pull request (NotPR, with no answer), and the row commit it
// was asked for.
type cacheEntry struct {
	Commit string   `json:"commit"`
	PR     prAnswer `json:"pr"`
	NotPR  bool     `json:"not_pr,omitempty"`
}

func cacheRel(n int) string { return CacheRelDir + "/pr-" + strconv.Itoa(n) + ".json" }

// readCache reads the cached answers for want whose entry names the row's
// commit. A cache file is something any process of the account could have
// written, so it is read as the forge's answer is: a regular file, never a
// link, within its bound (fsutil.ReadGuardedInRoot), decoded strictly and
// checked, and an answer keep would not write (prAnswer.lasting) is asked
// again. A cache git tracks came with a commit, so none of it counts, and
// the reason says so, so the listing writes none over it. notPR holds the
// numbers an entry says name no pull request.
func readCache(ctx context.Context, root *os.Root, repoRoot string, want []*prAsk) (got map[int]*prAnswer, notPR map[int]bool, why string) {
	got, notPR = map[int]*prAnswer{}, map[int]bool{}
	if root == nil {
		return got, notPR, ""
	}
	var rels []string
	for _, a := range want {
		rel := cacheRel(a.pr)
		raw, err := fsutil.ReadGuardedInRoot(root, rel, maxCacheBytes)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		rels = append(rels, rel)
		if err != nil {
			continue
		}
		var e cacheEntry
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		switch {
		case dec.Decode(&e) != nil || dec.More() || e.Commit != a.commit:
		case e.NotPR:
			if e.PR.Number == 0 {
				notPR[a.pr] = true
			}
		case e.PR.check(a.pr) == "" && e.PR.lasting(a.commit):
			got[a.pr] = &e.PR
		}
	}
	if len(rels) > 0 {
		if err := localtier.UntrackedEach(ctx, repoRoot, rels...); err != nil {
			return map[int]*prAnswer{}, map[int]bool{}, "the forge cache is neither read nor written: " + whyText(err)
		}
	}
	return got, notPR, ""
}

// cacheWritable says whether the cache may be written in the checkout at
// repoRoot, and why not when there is something to say: a checkout with no
// local tier keeps nothing and says nothing, and a tier git does not say it
// ignores keeps nothing, so a listing never leaves the checkout with files to
// commit.
func cacheWritable(ctx context.Context, root *os.Root, rootErr error, repoRoot string) (bool, string) {
	if rootErr != nil {
		return false, "the forge's answers cannot be kept: " + whyText(rootErr)
	}
	if fi, err := root.Lstat(landing.TierRelDir); err != nil || !fi.IsDir() {
		return false, ""
	}
	if err := gitutil.WorkingTreeCommandContext(ctx, repoRoot, "check-ignore", "-q", "--", CacheRelDir+"/").Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, "the forge's answers are not kept: the listing's time ran out before git said whether it ignores " + CacheRelDir + "/: " + whyText(ctxErr)
		}
		return false, "the forge's answers are not kept: git does not say it ignores " + CacheRelDir + "/ in this checkout"
	}
	return true, ""
}

// writeCache keeps e, what the forge said of pull request n, in the local
// tier cacheWritable allowed, and says why it could not, "" when it did. The
// file is one a later listing makes again from the forge, so it is written
// without the flushes only a power cut needs (fsutil.WriteRebuildableInRoot),
// and none of it waits on the listing's ctx, so an answer read is kept even
// when the listing's time runs out after it.
func writeCache(root *os.Root, n int, e cacheEntry) string {
	data, err := json.Marshal(e)
	if err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	if err := root.MkdirAll(CacheRelDir, 0o700); err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	if err := fsutil.WriteRebuildableInRoot(root, cacheRel(n), append(data, '\n'), 0o600); err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	return ""
}
