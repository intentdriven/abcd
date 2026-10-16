package claim

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
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
// commit it removes. A fetch that names each branch by its own refspec
// (only) never prunes a branch origin no longer holds, so the listing that
// sets only drops the rest first: what is left under the namespace is what
// the fetch then refreshes.
func (ns namespace) dropUnlisted(root string, keep []string) error {
	out, err := storedGitCtx(context.Background(), root, 16<<20, "for-each-ref", "--format=%(refname) %(objectname)", ns.tracking)
	if err != nil {
		return fmt.Errorf("listing %s held here: %s", ns.noun, redacted(err))
	}
	var del strings.Builder
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) != 2 || !gitutil.IsFullSHA(f[1]) {
			continue
		}
		name, ok := strings.CutPrefix(f[0], ns.tracking)
		if !ok || slices.Contains(keep, name) {
			continue
		}
		fmt.Fprintf(&del, "delete %s %s\n", f[0], f[1])
	}
	if del.Len() == 0 {
		return nil
	}
	if _, err := storedGitInput(context.Background(), root, del.String(), 64<<10, "update-ref", "--stdin"); err != nil {
		return fmt.Errorf("dropping %s origin no longer lists: %s", ns.noun, redacted(err))
	}
	return nil
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
