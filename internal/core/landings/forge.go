package landings

// forge.go is the reader's one door to the forge (spc-2610301950480609
// approach 6): every row that names a pull request is asked about at most
// once, in GraphQL documents of up to MaxBatch aliased pullRequest nodes, run
// as `gh api graphql` through the landing's own client (landing.GHContext, so
// each call is bounded by landing.ForgeTimeout and the caller's ctx, its
// environment scrubbed and its output bounded). An answer is kept in the local
// tier's cache, keyed by the row's commit, so a later listing asks only about
// pull requests it has not read; an unread answer is never kept.
//
// A forge that cannot be asked, or whose answer does not parse, carries errors,
// or leaves a pull request out, answers for none of the rows concerned: they
// read as "forge unread", never guessed, and the reason is the listing's note.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// prSelection is what the batch reads of each pull request. Every field is
// pinned in internal/ghcontract (graphql-fields.txt).
const prSelection = `number author{login} headRefOid mergedAt body mergeCommit{oid} ` +
	`reviews(last:100){totalCount nodes{author{login} state commit{oid} submittedAt}} ` +
	`commits(first:100){totalCount nodes{commit{oid message}}}`

// BatchQuery is the GraphQL document asking about the pull requests prs, each
// as the aliased node p<N>. The owner and name are its variables $o and $r.
func BatchQuery(prs []int) string {
	var b strings.Builder
	b.WriteString(`query($o:String!,$r:String!){repository(owner:$o,name:$r){`)
	for _, n := range prs {
		fmt.Fprintf(&b, "p%d:pullRequest(number:%d){%s} ", n, n, prSelection)
	}
	b.WriteString("}}")
	return b.String()
}

// batchArgs are the forge client's arguments for BatchQuery on prs, for the
// repository the forge client resolves for the checkout ({owner} and {repo},
// which gh fills in).
func batchArgs(prs []int) []string {
	return []string{"api", "graphql", "-F", "o={owner}", "-F", "r={repo}", "-f", "query=" + BatchQuery(prs)}
}

type forgeLogin struct {
	Login string `json:"login"`
}

type forgeOid struct {
	Oid string `json:"oid"`
}

type forgeReview struct {
	Author      *forgeLogin `json:"author"`
	State       string      `json:"state"`
	Commit      *forgeOid   `json:"commit"`
	SubmittedAt string      `json:"submittedAt"`
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
	Body        string      `json:"body"`
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

// loginRe is a forge login's shape; any other is no reviewer and no author.
var loginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})(?:\[bot\])?$`)

// check says why a is not an answer for pull request n, "" when it is: the
// number asked, object names in their shape, and pages no longer than asked.
func (a *prAnswer) check(n int) string {
	switch {
	case a.Number != n:
		return fmt.Sprintf("the forge's answer for pull request #%d names another", n)
	case a.HeadRefOid != "" && !gitutil.IsFullSHA(a.HeadRefOid),
		a.MergeCommit != nil && !gitutil.IsFullSHA(a.MergeCommit.Oid):
		return fmt.Sprintf("the forge's answer for pull request #%d names a commit by no full object name", n)
	case len(a.Reviews.Nodes) > maxPage || len(a.Commits.Nodes) > maxPage ||
		a.Reviews.TotalCount < len(a.Reviews.Nodes) || a.Commits.TotalCount < len(a.Commits.Nodes):
		return fmt.Sprintf("the forge's answer for pull request #%d holds pages it was not asked for", n)
	}
	return ""
}

// landedAs says why a does not answer for the row commit, "" when it does: the
// forge must say pull request n merged, as commit. A subject is text whoever
// made the commit typed, so a commit pushed with another merge's subject reads
// none of that pull request's reviews or commits.
func (a *prAnswer) landedAs(n int, commit string) string {
	switch {
	case a.MergedAt == nil:
		return fmt.Sprintf("the forge says pull request #%d has not merged, so it does not answer for %s", n, commit[:12])
	case a.MergeCommit == nil || a.MergeCommit.Oid != commit:
		return fmt.Sprintf("the forge says pull request #%d did not merge as %s, so it does not answer for it", n, commit[:12])
	}
	return ""
}

// approval is the review of head that counts in a, by countedApproval's
// reading of a page without the permission read: the latest submitted review
// of each reviewer, not pending and not dismissed, that approves head, by
// someone other than the pull request's author. A reviews page the forge says
// is not whole counts none, and says why.
func (a *prAnswer) approval(head string) (Approval, string) {
	if a.Reviews.TotalCount > len(a.Reviews.Nodes) {
		return Approval{}, fmt.Sprintf("pull request #%d has more than %d reviews, a later one may sit beyond the page read, so none counts", a.Number, maxPage)
	}
	if head == "" {
		return Approval{}, ""
	}
	author := ""
	if a.Author != nil {
		author = a.Author.Login
	}
	latest := map[string]forgeReview{}
	var order []string
	for _, r := range a.Reviews.Nodes {
		if r.Author == nil || !loginRe.MatchString(r.Author.Login) || r.State == "PENDING" || r.State == "DISMISSED" {
			continue
		}
		k := strings.ToLower(r.Author.Login)
		if _, ok := latest[k]; !ok {
			order = append(order, k)
		}
		latest[k] = r
	}
	for _, k := range order {
		r := latest[k]
		if r.State != "APPROVED" || strings.EqualFold(r.Author.Login, author) || r.Commit == nil || r.Commit.Oid != head {
			continue
		}
		at, _ := time.Parse(time.RFC3339, r.SubmittedAt)
		return Approval{Seen: true, By: r.Author.Login, At: at.UTC(), Source: SourceForge}, ""
	}
	return Approval{}, ""
}

// prAsk is one row's question: its pull request and its commit.
type prAsk struct {
	pr     int
	commit string
}

// forgeAnswers are readForge's answers by pull request, why each one it has
// none for was not read, and why an answer could not be cached.
type forgeAnswers struct {
	got      map[int]*prAnswer
	why      map[int]string
	cacheWhy string
}

// readForge answers asks from the cache where it holds the row's commit, and
// asks the forge about the rest, at most once each, MaxBatch at a time. A
// missing forge client or an ended ctx stops the asking, and every pull
// request not yet answered reads as unread with the reason.
func readForge(ctx context.Context, repoRoot string, forge landing.ForgeCall, asks []prAsk) forgeAnswers {
	fa := forgeAnswers{got: map[int]*prAnswer{}, why: map[int]string{}}
	commitOf := map[int]string{}
	var want []int
	for _, a := range asks {
		if _, ok := commitOf[a.pr]; ok {
			continue
		}
		commitOf[a.pr] = a.commit
		want = append(want, a.pr)
	}
	root, rootErr := os.OpenRoot(repoRoot)
	if rootErr == nil {
		defer root.Close()
	}
	var ask []int
	cached, tracked := readCache(ctx, root, repoRoot, want, commitOf)
	if tracked != "" {
		fa.cacheWhy = tracked
	}
	for _, n := range want {
		if a := cached[n]; a != nil {
			fa.got[n] = a
		} else {
			ask = append(ask, n)
		}
	}
	var stop string
	for i := 0; i < len(ask); i += MaxBatch {
		batch := ask[i:min(i+MaxBatch, len(ask))]
		if stop == "" && ctx.Err() != nil {
			stop = "the listing's time ran out before the forge was asked: " + whyText(ctx.Err())
		}
		if stop != "" {
			for _, n := range batch {
				fa.why[n] = stop
			}
			continue
		}
		got, why, err := askBatch(forge, batch)
		if errors.Is(err, landing.ErrNoForgeClient) {
			stop = why
		}
		for _, n := range batch {
			if a := got[n]; a != nil {
				fa.got[n] = a
				if tracked != "" {
					continue
				}
				if w := writeCache(root, rootErr, repoRoot, n, commitOf[n], a); w != "" && fa.cacheWhy == "" {
					fa.cacheWhy = w
				}
			} else if why != "" {
				fa.why[n] = why
			} else {
				fa.why[n] = fmt.Sprintf("the forge's answer leaves out pull request #%d", n)
			}
		}
	}
	return fa
}

// batchAnswer is the forge's answer to one BatchQuery. An answer may carry
// errors beside its data, and gh exits 0 on some of them, so Errors is read.
type batchAnswer struct {
	Data *struct {
		Repository map[string]json.RawMessage `json:"repository"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// askBatch asks the forge about batch in one document and returns the answers
// it gave, each checked; why is the reason for the whole batch when the forge
// could not be asked or its answer cannot be read, with the call's error.
func askBatch(forge landing.ForgeCall, batch []int) (map[int]*prAnswer, string, error) {
	names := make([]string, len(batch))
	for i, n := range batch {
		names[i] = "#" + strconv.Itoa(n)
	}
	about := "pull requests " + strings.Join(names, ", ")
	out, err := forge(batchArgs(batch)...)
	if err != nil {
		return nil, "the forge could not be asked about " + about + ": " + whyText(err), err
	}
	var a batchAnswer
	if json.Unmarshal([]byte(out), &a) != nil {
		return nil, "the forge's answer about " + about + " does not parse", nil
	}
	if len(a.Errors) > 0 {
		return nil, fmt.Sprintf("the forge's answer about %s carries %d errors", about, len(a.Errors)), nil
	}
	if a.Data == nil || a.Data.Repository == nil {
		return nil, "the forge's answer about " + about + " names no repository", nil
	}
	got := map[int]*prAnswer{}
	for _, n := range batch {
		raw, ok := a.Data.Repository["p"+strconv.Itoa(n)]
		if !ok || string(raw) == "null" {
			continue
		}
		var pr prAnswer
		if json.Unmarshal(raw, &pr) != nil || pr.check(n) != "" {
			continue
		}
		got[n] = &pr
	}
	return got, "", nil
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
// written, so it is read as the forge's answer is: a regular file within its
// bound, decoded strictly and checked. A cache git tracks came with a commit,
// so none of it counts, and tracked says why, so the listing writes none
// over it.
func readCache(ctx context.Context, root *os.Root, repoRoot string, want []int, commitOf map[int]string) (got map[int]*prAnswer, tracked string) {
	got = map[int]*prAnswer{}
	if root == nil {
		return got, ""
	}
	var rels []string
	for _, n := range want {
		rel := cacheRel(n)
		fi, err := root.Lstat(rel)
		if err != nil {
			continue
		}
		rels = append(rels, rel)
		if !fi.Mode().IsRegular() || fi.Size() > maxCacheBytes {
			continue
		}
		f, err := root.Open(rel)
		if err != nil {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(f, maxCacheBytes+1))
		f.Close()
		if err != nil || len(raw) > maxCacheBytes {
			continue
		}
		var e cacheEntry
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if dec.Decode(&e) != nil || dec.More() || e.Commit != commitOf[n] || e.PR.check(n) != "" {
			continue
		}
		got[n] = &e.PR
	}
	if len(rels) > 0 {
		if err := localtier.UntrackedEach(ctx, repoRoot, rels...); err != nil {
			return map[int]*prAnswer{}, "the forge cache is neither read nor written: " + whyText(err)
		}
	}
	return got, ""
}

// writeCache keeps a, the answer for pull request n asked for commit, in the
// local tier, and says why it could not, "" when it did or when the checkout
// has no local tier to keep it in.
func writeCache(root *os.Root, rootErr error, repoRoot string, n int, commit string, a *prAnswer) string {
	if rootErr != nil {
		return "the forge's answers cannot be kept: " + whyText(rootErr)
	}
	if fi, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(landing.TierRelDir))); err != nil || !fi.IsDir() {
		return ""
	}
	data, err := json.Marshal(cacheEntry{Commit: commit, PR: *a})
	if err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	if err := root.MkdirAll(CacheRelDir, 0o700); err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	if err := fsutil.WriteFileAtomicInRoot(root, cacheRel(n), append(data, '\n'), 0o600); err != nil {
		return "the forge's answers cannot be kept: " + whyText(err)
	}
	return ""
}
