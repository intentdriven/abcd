package claim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

// namespace is one family of branches on origin that the fetch reads as one
// refspec: where they sit there, where this checkout keeps what it last
// fetched of them, and the directory in the local tier that keeps when it did
// (the fetched-at stamp) and how far this checkout has read each branch (the
// history walk's marks). Each namespace keeps its own stamp, so one fetch of
// several namespaces stamps each, and a fetch of one never ages another.
type namespace struct {
	// remote is the branches' prefix on origin, a full ref under refs/heads/
	// ending in "/".
	remote string
	// tracking is the prefix this checkout keeps them under, a full ref under
	// refs/remotes/<Remote>/ ending in "/".
	tracking string
	// stateDir is the namespace's directory in the local tier, relative to the
	// checkout's root.
	stateDir string
	// noun names the branches in a refusal: "the reservations".
	noun string
	// also is every further refspec the namespace's branches are read with,
	// fetched in the same command: the reservations' release tags, which end
	// a release subject (release.go).
	also []string
	// only, when not nil, is the one set of branches the fetch names, each by
	// its own refspec, in place of the globbed one: the status reader's fetch,
	// which lists origin's branches first and fetches only those whose names
	// it reads (status.go). An empty set names no branch of the namespace.
	only []string
}

// reservations is the reservation branches' namespace.
var reservations = namespace{remote: RefPrefix, tracking: TrackingPrefix, stateDir: stateRelDir, noun: "the reservations",
	also: []string{releaseTagsRefspec}}

// refspec is the namespace's forced, globbed refspec: every branch under the
// prefix on origin to the same name under the tracking prefix.
func (ns namespace) refspec() string { return "+" + ns.remote + "*:" + ns.tracking + "*" }

// refspecs is every refspec a fetch of the namespace names: the globbed one,
// or with only set, one forced refspec per branch it names. A name in only is
// already validated by its family.
func (ns namespace) refspecs() []string {
	if ns.only == nil {
		return []string{ns.refspec()}
	}
	specs := make([]string, 0, len(ns.only))
	for _, name := range ns.only {
		specs = append(specs, "+"+ns.ref(name)+":"+ns.track(name))
	}
	return specs
}

// dropUnlisted deletes every tracking ref of the namespace this checkout
// holds whose name is not in keep, the names origin's listing named and the
// fetch reads, in one `git update-ref --stdin` whose every deletion names the
// value it removes and deletes the ref itself, never a ref a symbolic one
// names (no-deref). A fetch that names each branch by its own refspec (only)
// never prunes a branch origin no longer holds, so the listing that sets only
// drops the rest first: what is left under the namespace is what the fetch
// then refreshes. When git refuses the batch (a concurrent git holds a lock,
// or moved a ref since it was listed), each copy is deleted on its own, so
// one that cannot go keeps no other; what is left is listed and tried once
// more. A copy that fails again is not a failed fetch: dropUnlisted answers
// how many copies it left, never their names, and git's reason, for the
// caller to say, and the fetch goes on.
func (ns namespace) dropUnlisted(root string, keep []string) (left string) {
	var n int
	var why string
	for range 2 {
		n, why = ns.dropOnce(root, keep)
		if n == 0 {
			return ""
		}
	}
	switch {
	case n < 0:
		return "the copies of " + ns.noun + " to delete could not be listed here: " + why
	case n == 1:
		return "1 copy of a status branch origin no longer lists, or whose name is not read, could not be deleted here, so it is read as last fetched: " + why
	}
	return fmt.Sprintf("%d copies of status branches origin no longer lists, or whose names are not read, could not be deleted here, so they are read as last fetched: %s", n, why)
}

// maxDropEach is the most copies one try of dropUnlisted deletes one at a
// time, once git has refused them together. A test lowers it.
var maxDropEach = 200

// dropOnce is one try of dropUnlisted: how many deletions it could not make,
// and why; 0 when it made them all, and -1 when it could not list them.
func (ns namespace) dropOnce(root string, keep []string) (int, string) {
	var del strings.Builder
	var stale [][2]string
	cut := 0
	err := storedRefs(context.Background(), root, ns.tracking, func(ref, sha string, long bool) {
		name := strings.TrimPrefix(ref, ns.tracking)
		switch {
		case long:
			// Its name was not read whole, so it cannot be named to delete.
			cut++
		case name != "" && !slices.Contains(keep, name):
			fmt.Fprintf(&del, "option no-deref\ndelete %s %s\n", ref, sha)
			stale = append(stale, [2]string{ref, sha})
		}
	})
	if err != nil {
		return -1, redacted(err)
	}
	if len(stale) == 0 {
		if cut > 0 {
			return cut, fmt.Sprintf("a ref name longer than %d bytes is not read whole", maxListingLine)
		}
		return 0, ""
	}
	if _, err := storedGitInput(context.Background(), root, del.String(), 64<<10, "update-ref", "--stdin"); err == nil {
		if cut > 0 {
			return cut, fmt.Sprintf("a ref name longer than %d bytes is not read whole", maxListingLine)
		}
		return 0, ""
	}
	// git applies the batch all or none, so one copy it cannot delete keeps
	// every other: each is then deleted on its own, at most maxDropEach of
	// them, the rest left for a later fetch, and only those left are
	// counted, with the first one's reason. A lock another git holds fails
	// each at once rather than after git's wait for it, so a held lock costs
	// one command per copy, never a second each.
	n, why := cut, ""
	if cut > 0 {
		why = fmt.Sprintf("a ref name longer than %d bytes is not read whole", maxListingLine)
	}
	if len(stale) > maxDropEach {
		n += len(stale) - maxDropEach
		stale = stale[:maxDropEach]
	}
	for _, r := range stale {
		if _, err := storedGitCtx(context.Background(), root, 64<<10, "-c", "core.filesRefLockTimeout=0", "-c", "core.packedRefsTimeout=0",
			"update-ref", "--no-deref", "-d", r[0], r[1]); err != nil {
			if why == "" {
				why = redacted(err)
			}
			n++
		}
	}
	return n, why
}

// ref is the full ref of the branch name in the namespace on origin, and
// track this checkout's copy of it. name is already validated by its family.
func (ns namespace) ref(name string) string   { return ns.remote + name }
func (ns namespace) track(name string) string { return ns.tracking + name }

// stampRel is the namespace's fetched-at stamp, relative to the root.
func (ns namespace) stampRel() string { return ns.stateDir + "/fetched-at" }

// nouns names every namespace in a fetch, as a refusal reads them.
func nouns(spaces []namespace) string {
	names := make([]string, len(spaces))
	for i, ns := range spaces {
		names[i] = ns.noun
	}
	return strings.Join(names, " and ")
}

// mkStateDir makes dir, a directory under the local tier, and every directory
// between the tier and it, through r. The tier itself is never made here.
func mkStateDir(r *os.Root, dir string) error {
	rest, ok := strings.CutPrefix(dir, tierRelDir+"/")
	if !ok || rest == "" {
		return errors.New(dir + " is not under " + tierRelDir)
	}
	at := tierRelDir
	for _, seg := range strings.Split(rest, "/") {
		at += "/" + seg
		if err := r.Mkdir(at, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	return nil
}
