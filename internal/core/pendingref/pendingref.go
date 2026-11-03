// Package pendingref answers one question for record-lint's prose citations
// (iss-2611031731385043): which of the record ids a tree does not resolve name
// a record on an open pull request's branch on origin, a record minted and in
// review that has not reached the default branch yet.
//
// The ruling it serves: a record is cited by id, never in words, even when the
// record it cites is still in review. A decision written in one change often
// rests on a ruling recorded in another change not yet merged, and refusing the
// id pushed authors to name the ruling in a sentence instead, which drops the
// trace. So record-lint accepts such an id as a PENDING reference, reported and
// never refused, and the release cut is the gate: it refuses while any
// reference names a record the tree being cut does not hold.
//
// HOW IT READS. One forge listing of the open pull requests (the REST pulls
// endpoint through `gh api`, which every forge token reads), and
// for each one whose head is a branch of this repository (a fork's is not on
// origin), git's listing of the record stores in that head's tree, read from
// the objects this clone already holds. It never fetches: a lint that rewrote
// the clone's refs would be a gate with a side effect. A head the clone does
// not hold is named in Unread, and a clone that has fetched origin (CI's
// full-history checkout does) holds them all.
//
// HOW IT FAILS. Every way the question cannot be answered — no forge client, a
// forge that refuses or does not answer, a head not fetched, a listing at its
// limit — is said in Unread, and nothing is found that was not read. The
// caller's citation then stands refused, as it was before this package
// existed, and its message says why the pending check could not help. Failing
// toward the refusal is the safe direction: an unread forge never turns a
// citation of nothing into an accepted one.
package pendingref

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Deadline bounds the whole lookup, the forge listing and every tree read. It
// is far below a preflight's budget and well above a forge's ordinary answer.
const Deadline = 60 * time.Second

// pageSize and maxPages bound the listing: up to two pages of a hundred open
// pull requests. A listing whose last page is full may have left some out,
// and Unread says so.
const (
	pageSize = 100
	maxPages = 2
)

// maxTreeListing caps one head's listing of the record stores. The ledger is a
// few thousand files of short paths; a listing past this is not a record.
const maxTreeListing = 8 << 20

// Record is one record an open pull request's branch carries.
type Record struct {
	// ID is the canonical record id.
	ID string
	// Path is the record's repo-relative path on that branch.
	Path string
	// PR is the pull request's number.
	PR int
	// Branch is the pull request's head branch on origin.
	Branch string
}

// Answer is the lookup's result.
type Answer struct {
	// Found maps each asked id an open pull request's branch carries to that
	// record, the lowest-numbered pull request's when more than one does.
	Found map[string]Record
	// Unread says why some open pull requests' branches were not read, empty
	// when every listed one was. An id not in Found may be on one of them.
	Unread string
}

// Forge runs the forge client with args from dir, bound to ctx, and returns
// its stdout: landing.GHContext, or a test's fake.
type Forge func(ctx context.Context, dir string, args ...string) (string, error)

// Git runs one read-only git command under root and returns its stdout,
// failing past maxBytes rather than truncating: gitutil.RunCapped, or a
// test's seam.
type Git func(root string, maxBytes int, args ...string) (string, error)

// Lookup answers which of ids name a record on an open pull request's branch
// on origin, through the repository's own forge client and git.
func Lookup(repoRoot string, ids []string) Answer {
	ctx, cancel := context.WithTimeout(context.Background(), Deadline)
	defer cancel()
	return LookupWith(ctx, landing.GHContext, gitutil.RunCapped, repoRoot, ids)
}

// pullRequest is one open pull request as the REST listing sends it, the
// fields internal/ghcontract pins for the pull-requests resource.
type pullRequest struct {
	Number int    `json:"number"`
	Head   prSide `json:"head"`
	Base   prSide `json:"base"`
}

// prSide is a pull request's head or base: the branch, its commit, and the
// repository it is a branch of (null for a fork since deleted).
type prSide struct {
	Ref  string  `json:"ref"`
	SHA  string  `json:"sha"`
	Repo *prRepo `json:"repo"`
}

// prRepo names a repository.
type prRepo struct {
	FullName string `json:"full_name"`
}

// sameRepository reports whether the head is a branch of the repository the
// pull request is into: a fork's branch, or one whose fork is gone, is not on
// origin.
func (p pullRequest) sameRepository() bool {
	return p.Head.Repo != nil && p.Base.Repo != nil && p.Head.Repo.FullName != "" &&
		strings.EqualFold(p.Head.Repo.FullName, p.Base.Repo.FullName)
}

// LookupWith is Lookup through the given forge client and git runner, under
// ctx. Asked nothing, it reads nothing: the forge is consulted only for an id
// the tree could not resolve, so a tree whose every citation resolves never
// waits on the network.
func LookupWith(ctx context.Context, forge Forge, git Git, repoRoot string, ids []string) Answer {
	ans := Answer{Found: map[string]Record{}}
	want := map[string]bool{}
	for _, id := range ids {
		if c := recordid.CanonCitedID(id); c != "" {
			want[c] = true
		}
	}
	if len(want) == 0 {
		return ans
	}

	prs, unread := listOpen(ctx, forge, repoRoot)
	if unread != "" {
		ans.Unread = unread
		return ans
	}
	var notes []string
	if len(prs) >= pageSize*maxPages {
		notes = append(notes, fmt.Sprintf("the forge listed %d open pull requests, the most this lookup reads, so a record on one it left out is not seen", len(prs)))
	}

	var missing []string
	for _, pr := range prs {
		if len(ans.Found) == len(want) {
			break
		}
		if !pr.sameRepository() {
			// A fork's branch is not on origin, and its head is not this
			// repository's to vouch for.
			continue
		}
		label := "#" + strconv.Itoa(pr.Number) + " (" + termsafe.CleanProseLine(pr.Head.Ref, 120) + ")"
		if !gitutil.IsFullSHA(pr.Head.SHA) {
			missing = append(missing, label+": the forge named no commit for its head")
			continue
		}
		if ctx.Err() != nil {
			missing = append(missing, label+": the lookup ran out of time")
			continue
		}
		if _, err := git(repoRoot, 1<<10, "cat-file", "-e", pr.Head.SHA+"^{commit}"); err != nil {
			missing = append(missing, label+": its head "+pr.Head.SHA[:12]+" is not in this clone; fetch origin and run again")
			continue
		}
		resolver, err := headResolver(git, repoRoot, pr.Head.SHA)
		if err != nil {
			missing = append(missing, label+": "+termsafe.CleanProseLine(fsutil.RedactHome(err.Error()), 200))
			continue
		}
		for id := range want {
			if _, done := ans.Found[id]; done || !resolver.Has(id) {
				continue
			}
			p, _, err := resolver.Lookup(id)
			var amb *recordid.AmbiguousIDError
			if errors.As(err, &amb) {
				p = strings.Join(amb.Paths, ", ")
			}
			ans.Found[id] = Record{ID: id, Path: p, PR: pr.Number, Branch: pr.Head.Ref}
		}
	}
	if len(missing) > 0 && len(ans.Found) < len(want) {
		notes = append(notes, "open pull request "+strings.Join(missing, "; open pull request "))
	}
	ans.Unread = strings.Join(notes, "; ")
	return ans
}

// listOpen is the one forge read: the open pull requests, lowest number
// first, or why they could not be read. It pages the REST listing rather than
// asking `gh pr list`, whose GraphQL a forge proxy may refuse where REST is
// served.
func listOpen(ctx context.Context, forge Forge, dir string) ([]pullRequest, string) {
	var prs []pullRequest
	for page := 1; page <= maxPages; page++ {
		out, err := forge(ctx, dir, "api", "-X", "GET", "repos/{owner}/{repo}/pulls",
			"-f", "state=open", "-f", "per_page="+strconv.Itoa(pageSize), "-f", "page="+strconv.Itoa(page))
		switch {
		case err != nil && (ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded)):
			return nil, "the forge did not list the open pull requests within " + Deadline.String()
		case errors.Is(err, landing.ErrNoForgeClient):
			return nil, "the forge client `gh` is not installed, so the open pull requests were not listed"
		case err != nil:
			return nil, "the forge did not list the open pull requests: " +
				termsafe.CleanProseLine(gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())), 200)
		}
		var batch []pullRequest
		if err := json.Unmarshal([]byte(out), &batch); err != nil {
			return nil, "the forge's listing of the open pull requests does not parse"
		}
		prs = append(prs, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	sort.Slice(prs, func(i, j int) bool { return prs[i].Number < prs[j].Number })
	return prs, ""
}

// headResolver reads the record stores in commit's tree, regular files only,
// as the working-tree scan reads them: a link or a submodule names no record
// here as it names none there.
func headResolver(git Git, root, commit string) (*recordid.Resolver, error) {
	args := append([]string{"ls-tree", "-r", "-z", "--full-tree", commit, "--"}, recordid.FamilyRootDirs()...)
	out, err := git(root, maxTreeListing, args...)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range strings.Split(out, "\x00") {
		meta, file, ok := strings.Cut(entry, "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta) // <mode> <type> <object>
		if len(f) != 3 || f[1] != "blob" || f[0] != "100644" && f[0] != "100755" {
			continue
		}
		paths = append(paths, file)
	}
	return recordid.NewResolverFromPaths(paths)
}

// ForLint is Lookup in the lint package's vocabulary, the one adapter every
// front door registers (lint.SetPendingLookup): cmd/record-lint for the gate,
// and the CLI for `abcd lint` and every host-prose ingest that asks the gate
// before it writes, so the two can never answer the question differently.
func ForLint(repoRoot string, ids []string) (map[string]lint.PendingRecord, string) {
	ans := Lookup(repoRoot, ids)
	out := make(map[string]lint.PendingRecord, len(ans.Found))
	for id, rec := range ans.Found {
		out[id] = lint.PendingRecord(rec)
	}
	return out, ans.Unread
}
