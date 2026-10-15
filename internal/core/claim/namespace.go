package claim

import (
	"errors"
	"os"
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
}

// reservations is the reservation branches' namespace.
var reservations = namespace{remote: RefPrefix, tracking: TrackingPrefix, stateDir: stateRelDir, noun: "the reservations"}

// refspec is the namespace's forced, globbed refspec: every branch under the
// prefix on origin to the same name under the tracking prefix.
func (ns namespace) refspec() string { return "+" + ns.remote + "*:" + ns.tracking + "*" }

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
