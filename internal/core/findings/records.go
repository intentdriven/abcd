package findings

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// maxListing bounds one family's listing on the default branch; past it the
// answer is an error, never a part read as the whole.
const maxListing = 8 << 20

// The places an answer is found, as a refusal names them.
const (
	whereDefault  = "the default branch"
	whereCheckout = "the checkout"
	whereStart    = "the session's start copy"
)

// Records answers a captured or settled claim on a record (criterion 12): the
// default branch decides, read with one `git ls-tree` of the family's
// directory the first time an id of that family is asked; a record it lacks is
// looked for with recordid.LookupOne in the checkout and then the session's
// start copy, and nowhere else. With no default branch the checkouts answer
// alone. Every git question is bounded by the context it was made with.
type Records struct {
	ctx       context.Context
	checkouts []string
	ref       string
	refRead   bool
	listed    map[string]*recordid.Resolver
}

// NewRecords is the resolver for a payload's checkout and, after it, the
// session's start copy (either may be ""). git is asked in the first.
func NewRecords(ctx context.Context, checkouts ...string) *Records {
	return &Records{ctx: ctx, checkouts: checkouts, listed: map[string]*recordid.Resolver{}}
}

// familyDirs are the families a claim can be answered by.
var familyDirs = map[string]string{
	"iss": recordid.IssuesRelDir,
	"adr": recordid.ADRsRelDir,
	"dcn": recordid.DecisionsRelDir,
}

// Answer is what id says to act (ActCaptured or ActSettled).
func (r *Records) Answer(act Act, id string) (Answer, error) {
	if !recordid.CitedIDRe.MatchString(id) {
		return Answer{State: "not a record id"}, nil
	}
	prefix, _, _ := strings.Cut(id, "-")
	switch prefix {
	case "itd":
		return Answer{State: "an intent"}, nil
	case "spc":
		return Answer{State: "a spec"}, nil
	}
	onDefault, err := r.defaultListing(prefix)
	if err != nil {
		return Answer{}, err
	}
	if onDefault != nil {
		p, ok, err := onDefault.Lookup(id)
		if err != nil {
			return Answer{}, err
		}
		if ok {
			return judge(act, prefix, p, whereDefault), nil
		}
	}
	for i, co := range r.checkouts {
		if co == "" {
			continue
		}
		p, ok, err := recordid.LookupOne(co, id)
		if err != nil {
			return Answer{}, err
		}
		if ok {
			where := whereCheckout
			if i > 0 {
				where = whereStart
			}
			return judge(act, prefix, p, where), nil
		}
	}
	return Answer{State: "absent"}, nil
}

// judge is the claim's answer for a record found at p. A captured claim holds
// for an open issue, and for a resolved one only where the default branch
// lacks it (filed and fixed in this change); a settled claim for an open
// issue, a wontfix, an ADR or a decision.
func judge(act Act, prefix, p, where string) Answer {
	state := ""
	switch prefix {
	case "adr":
		state = "an ADR"
	case "dcn":
		state = "a decision"
	default:
		state = path.Base(path.Dir(p))
	}
	holds := false
	switch act {
	case ActCaptured:
		holds = state == "open" || state == "resolved" && where != whereDefault
	case ActSettled:
		holds = state == "open" || state == "wontfix" || prefix == "adr" || prefix == "dcn"
	}
	return Answer{Holds: holds, State: state, Where: where}
}

// defaultListing is the family's records on the default branch, or nil when
// there is no default branch to read.
func (r *Records) defaultListing(prefix string) (*recordid.Resolver, error) {
	if res, ok := r.listed[prefix]; ok {
		return res, nil
	}
	if len(r.checkouts) == 0 || r.checkouts[0] == "" {
		return nil, nil
	}
	root := r.checkouts[0]
	if !r.refRead {
		r.ref, r.refRead = gitutil.DefaultRefContext(r.ctx, root), true
		if r.ref == "" && r.ctx.Err() != nil {
			r.refRead = false
			return nil, fmt.Errorf("findings: the default branch was not read in time: %w", r.ctx.Err())
		}
	}
	if r.ref == "" {
		r.listed[prefix] = nil
		return nil, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(r.ctx, root, "", maxListing,
		"ls-tree", "-r", "-z", "--full-tree", r.ref, "--", familyDirs[prefix])
	if err != nil {
		return nil, fmt.Errorf("findings: listing %s on the default branch: %w", familyDirs[prefix], err)
	}
	var paths []string
	for _, ent := range strings.Split(string(out), "\x00") {
		meta, file, ok := strings.Cut(ent, "\t")
		if !ok {
			continue
		}
		if f := strings.Fields(meta); len(f) == 3 && f[1] == "blob" && (f[0] == "100644" || f[0] == "100755") {
			paths = append(paths, file)
		}
	}
	res, err := recordid.NewResolverFromPaths(paths)
	if err != nil {
		return nil, err
	}
	r.listed[prefix] = res
	return res, nil
}
