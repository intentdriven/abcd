package landings

// forge.go is the reader's one door to the forge (spc-2610301950480609
// approach 6): every row that names a pull request is asked about at most
// once, in GraphQL documents of up to MaxBatch aliased pullRequest nodes, run
// as `gh api graphql` through the landing's own client (landing.GHAnswerContext,
// so each call is bounded by landing.ForgeTimeout and the caller's ctx, its
// environment scrubbed and its output bounded). Each node asks only what this
// step reads: who opened the pull request, its head, how it merged, and its
// reviews, with its commits only for a row that merged with one parent. An
// answer the reader accepts for its row is kept in the local tier's cache,
// keyed by the row's commit, so a later listing asks only about pull requests
// it has not read; an answer it does not accept, and an unread one, is never
// kept.
//
// The forge's answer is judged a pull request at a time: an error that names
// one pull request leaves that one unread and the others answered, and a
// number the forge says is no pull request reads so for its row alone. An
// answer that does not parse, or carries errors that name no pull request, or
// passes the client's output bound, answers for none of the batch, which is
// asked again in halves, down to one pull request, while the listing's time
// lasts. A missing client and a refusal for the rate limit or the credentials
// stop the asking. Every row left unanswered reads as "forge unread", never
// guessed, and the reason is the listing's note.

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
}

// loginRe is a person's forge login's shape; any other is no reviewer.
var loginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)

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
	cached, cacheWhy := readCache(ctx, root, repoRoot, want)
	fa.cacheWhy = cacheWhy
	var ask []prAsk
	for _, a := range want {
		if c := cached[a.pr]; c != nil {
			fa.got[a.pr] = c
		} else {
			ask = append(ask, *a)
		}
	}
	q := &asker{ctx: ctx, forge: forge, fa: &fa}
	for i := 0; i < len(ask); i += MaxBatch {
		q.ask(ask[i:min(i+MaxBatch, len(ask))])
	}
	var keep []prAsk
	for _, a := range ask {
		if got := fa.got[a.pr]; got != nil && got.landedAs(a.pr, a.commit) == "" && got.whole() == "" {
			keep = append(keep, a)
		}
	}
	if cacheWhy != "" || len(keep) == 0 {
		return fa
	}
	if ok, why := cacheWritable(ctx, root, rootErr, repoRoot); !ok {
		fa.cacheWhy = why
		return fa
	}
	for _, a := range keep {
		if why := writeCache(root, a.pr, a.commit, fa.got[a.pr]); why != "" {
			fa.cacheWhy = why
			break
		}
	}
	return fa
}

// asker asks the forge about batches for readForge, splitting a batch the
// forge answered for none of, and stopping on a refusal that would refuse
// every batch.
type asker struct {
	ctx   context.Context
	forge landing.ForgeCall
	fa    *forgeAnswers
	stop  string
}

func (q *asker) ask(batch []prAsk) {
	if q.stop == "" && q.ctx.Err() != nil {
		q.stop = "the listing's time ran out before the forge was asked: " + whyText(q.ctx.Err())
	}
	if q.stop != "" {
		q.unread(batch, q.stop)
		return
	}
	r := askBatch(q.forge, batch)
	switch {
	case r.stop:
		q.stop = r.whole
		q.unread(batch, r.whole)
		return
	case r.whole != "" && r.split && len(batch) > 1:
		q.ask(batch[:len(batch)/2])
		q.ask(batch[len(batch)/2:])
		return
	case r.whole != "":
		q.unread(batch, r.whole)
		return
	}
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
// may answer (split), and whether to stop asking (stop).
type batchResult struct {
	got   map[int]*prAnswer
	why   map[int]string
	notPR map[int]bool
	whole string
	split bool
	stop  bool
}

// stopsAsking says whether a refusal's text is one every later batch would
// meet too: the forge's rate limit, or credentials it does not take.
func stopsAsking(text string) bool {
	t := strings.ToLower(text)
	return strings.Contains(t, "rate limit") || strings.Contains(t, "http 401") || strings.Contains(t, "bad credentials")
}

// askBatch asks the forge about batch in one document and reads the answer a
// pull request at a time.
func askBatch(forge landing.ForgeCall, batch []prAsk) batchResult {
	r := batchResult{got: map[int]*prAnswer{}, why: map[int]string{}, notPR: map[int]bool{}}
	names := make([]string, len(batch))
	inBatch := map[string]int{}
	for i, a := range batch {
		names[i] = "#" + strconv.Itoa(a.pr)
		inBatch["p"+strconv.Itoa(a.pr)] = a.pr
	}
	about := "pull requests " + strings.Join(names, ", ")
	out, err := forge(batchArgs(batch)...)
	switch {
	case errors.Is(err, landing.ErrNoForgeClient):
		r.whole, r.stop = "the forge could not be asked about "+about+": "+whyText(err), true
		return r
	case errors.Is(err, landing.ErrOutputCap):
		r.whole, r.split = "the forge's answer about "+about+" passes the forge client's output bound", true
		return r
	}
	if err != nil && stopsAsking(err.Error()) {
		r.whole, r.stop = "the forge refused "+about+": "+whyText(err), true
		return r
	}
	var a batchAnswer
	decoded := json.Unmarshal([]byte(out), &a) == nil
	for _, e := range a.Errors {
		if e.Type == "RATE_LIMITED" {
			r.whole, r.stop = "the forge refused "+about+" for its rate limit", true
			return r
		}
	}
	switch {
	case !decoded && err != nil, decoded && len(a.Errors) == 0 && err != nil:
		r.whole = "the forge could not be asked about " + about + ": " + whyText(err)
		return r
	case !decoded:
		r.whole = "the forge's answer about " + about + " does not parse"
		return r
	case a.Data == nil || a.Data.Repository == nil:
		// Errors with no data under the repository may come of one pull
		// request; asked about fewer at a time, the others may answer.
		r.whole, r.split = "the forge's answer about "+about+" names no repository", len(a.Errors) > 0
		return r
	}
	for _, e := range a.Errors {
		alias, _ := pathAlias(e.Path)
		n, ok := inBatch[alias]
		if !ok {
			r.whole, r.split = fmt.Sprintf("the forge's answer about %s carries errors that name no one pull request", about), true
			return r
		}
		switch {
		case e.Type == "NOT_FOUND" && len(e.Path) == 2:
			r.notPR[n] = true
			r.why[n] = fmt.Sprintf("the forge says #%d names no pull request", n)
		case r.why[n] == "":
			r.why[n] = fmt.Sprintf("the forge's answer for pull request #%d carries an error", n)
		}
	}
	for _, b := range batch {
		if r.why[b.pr] != "" {
			continue
		}
		raw, ok := a.Data.Repository["p"+strconv.Itoa(b.pr)]
		if !ok || string(raw) == "null" {
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

// pathAlias is the pull request node an error's path names: its second
// element under "repository".
func pathAlias(path []any) (string, bool) {
	if len(path) < 2 || path[0] != "repository" {
		return "", false
	}
	s, ok := path[1].(string)
	return s, ok
}

// cacheEntry is one cache file: the answer, and the row commit it was asked
// for.
type cacheEntry struct {
	Commit string   `json:"commit"`
	PR     prAnswer `json:"pr"`
}

func cacheRel(n int) string { return CacheRelDir + "/pr-" + strconv.Itoa(n) + ".json" }

// readCache reads the cached answers for want whose entry names the row's
// commit. A cache file is something any process of the account could have
// written, so it is read as the forge's answer is: a regular file, never a
// link, within its bound (fsutil.ReadGuardedInRoot), decoded strictly and
// checked. A cache git tracks came with a commit, so none of it counts, and
// the reason says so, so the listing writes none over it.
func readCache(ctx context.Context, root *os.Root, repoRoot string, want []*prAsk) (got map[int]*prAnswer, why string) {
	got = map[int]*prAnswer{}
	if root == nil {
		return got, ""
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
		if dec.Decode(&e) != nil || dec.More() || e.Commit != a.commit || e.PR.check(a.pr) != "" {
			continue
		}
		got[a.pr] = &e.PR
	}
	if len(rels) > 0 {
		if err := localtier.UntrackedEach(ctx, repoRoot, rels...); err != nil {
			return map[int]*prAnswer{}, "the forge cache is neither read nor written: " + whyText(err)
		}
	}
	return got, ""
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
		return false, "the forge's answers are not kept: git does not say it ignores " + CacheRelDir + "/ in this checkout"
	}
	return true, ""
}

// writeCache keeps a, the answer for pull request n asked for commit, in the
// local tier cacheWritable allowed, and says why it could not, "" when it did.
// The file is one a later listing makes again from the forge, so it is
// written without the flushes only a power cut needs
// (fsutil.WriteRebuildableInRoot).
func writeCache(root *os.Root, n int, commit string, a *prAnswer) string {
	data, err := json.Marshal(cacheEntry{Commit: commit, PR: *a})
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
