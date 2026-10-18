package embargo

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// releases.go is what counts as a published release for a plan. A release
// tag in this checkout counts only when origin holds the tag at the same
// object, and GitHub reports a release for it that is neither a draft nor a
// prerelease: the release workflow drafts the release, checks its assets,
// then publishes it, so a tag stands on origin for that whole window, and a
// run that failed leaves a tag no release was published from. A tag that is
// not a published release gives no patched version and is never the plan's
// release.

// originRemote is the remote whose tags count.
const originRemote = "origin"

// maxRefLine bounds one line of origin's tag listing: an object name, a tab
// and a ref name a release tag's shape keeps short.
const maxRefLine = 1024

// releaseTags is what one plan reads of the release tags: the checkout's,
// at once; origin's, once, on the first question that needs them; and
// GitHub's answer for each tag asked about, once.
//
// Everything origin and GitHub send is untrusted input. Only a line naming
// a full object name and a tag of the release form (launch.ParseReleaseTag)
// is read from origin's listing, and a name it lists twice at two objects
// counts at neither. A tag reaches GitHub's API path only after it is held
// to the release form, and GitHub's answer must name that tag and carry
// both flags as booleans. Nothing either sends is repeated in a reason or a
// refusal.
type releaseTags struct {
	checkout, repo string
	local          map[string]string // release tag -> object, as this checkout holds it
	origin         map[string]string // as origin holds it; nil until read
	onGitHub       map[string]tagWhy // GitHub's answer for each tag asked about
}

// tagWhy is why a tag is not a published release, and the remedy; both
// empty when it is one.
type tagWhy struct{ why, remedy string }

// readLocalReleaseTags reads the checkout's release tags.
func readLocalReleaseTags(checkout, repo string) (*releaseTags, error) {
	out, err := gitutil.RunCappedBytes(checkout, maxTagListing, "for-each-ref", "--format=%(objectname) %(refname)", "refs/tags/")
	if err != nil {
		return nil, refuse("git could not list the release tags", "settle what git reports, then run it again")
	}
	rt := &releaseTags{checkout: checkout, repo: repo, local: map[string]string{}, onGitHub: map[string]tagWhy{}}
	for _, line := range strings.Split(string(out), "\n") {
		obj, ref, ok := strings.Cut(line, " ")
		name, isTag := strings.CutPrefix(ref, "refs/tags/")
		if ok && isTag && gitutil.IsFullSHA(obj) && isReleaseTag(name) {
			rt.local[name] = obj
		}
	}
	return rt, nil
}

// sorted is the checkout's release tags, lowest version first.
func (rt *releaseTags) sorted() []string {
	names := make([]string, 0, len(rt.local))
	for n := range rt.local {
		names = append(names, n)
	}
	return sortReleaseTags(names)
}

// readOrigin lists origin's tags once (`git ls-remote --refs --tags
// origin`, through the networked runner, under the person's own git
// configuration and credentials), read line by line. A listing that fails
// refuses the plan whole: whether a tag is published cannot be told, and the
// checkout's own tags are never taken in its place.
func (rt *releaseTags) readOrigin() error {
	if rt.origin != nil {
		return nil
	}
	var o originListing
	err := gitutil.NetEach(rt.checkout, maxRefLine, o.add, "ls-remote", "--refs", "--tags", originRemote)
	if err != nil {
		return refuse("origin's tags could not be listed, so whether a release tag is published cannot be told; nothing was recorded",
			"settle what `git ls-remote --tags origin` reports (the network, origin's URL, your credentials), then run it again")
	}
	rt.origin = o.tags()
	return nil
}

// originListing reads origin's tag listing line by line: only a line of a
// full object name, a tab and a release tag's ref is read, and a tag named
// twice at two objects is held at neither.
type originListing struct {
	seen  map[string]string
	twice map[string]bool
}

func (o *originListing) add(line string) {
	obj, ref, ok := strings.Cut(strings.TrimSpace(line), "\t")
	name, isTag := strings.CutPrefix(ref, "refs/tags/")
	if !ok || !isTag || !gitutil.IsFullSHA(obj) || !isReleaseTag(name) {
		return
	}
	if o.seen == nil {
		o.seen, o.twice = map[string]string{}, map[string]bool{}
	}
	if prev, dup := o.seen[name]; dup && prev != obj {
		o.twice[name] = true
	}
	o.seen[name] = obj
}

// tags is each release tag origin holds, at its object.
func (o *originListing) tags() map[string]string {
	out := map[string]string{}
	for name, obj := range o.seen {
		if !o.twice[name] {
			out[name] = obj
		}
	}
	return out
}

// originRepoOf is how a plan reads the repository origin names: originRepo,
// unless a test has swapped it (SwapOriginRepoForTest).
var originRepoOf = originRepo

// SwapOriginRepoForTest substitutes how a plan reads the repository origin
// names, and returns the restore: a test binary's git reaches no network, so
// a fixture stands origin in with a local repository, which names none. It
// is exported because the surface's tests live in another package. Tests
// only; never called in production code, and never safe to call from a
// parallel test.
func SwapOriginRepoForTest(fn func(top string) (string, error)) (restore func()) {
	prev := originRepoOf
	originRepoOf = fn
	return func() { originRepoOf = prev }
}

// published says why tag, a release tag this checkout holds, is not a
// published release ("" when it is one), with the remedy. An error is a
// question origin or GitHub could not answer, which refuses the plan whole.
func (rt *releaseTags) published(tag string) (tagWhy, error) {
	if !isReleaseTag(tag) {
		return tagWhy{"it is not a release tag", "name a release tag of the form vMAJOR.MINOR.PATCH"}, nil
	}
	if err := rt.readOrigin(); err != nil {
		return tagWhy{}, err
	}
	obj, ok := rt.origin[tag]
	switch {
	case !ok:
		return tagWhy{"origin does not hold the tag", "push it with `git push origin " + tag + "`, then run it again"}, nil
	case obj != rt.local[tag]:
		return tagWhy{"origin holds the tag at another object than this checkout does",
			"settle which " + tag + " is the release, so origin and this checkout hold the same one, then run it again"}, nil
	}
	if w, asked := rt.onGitHub[tag]; asked {
		return w, nil
	}
	w, err := releaseOnGitHub(rt.checkout, rt.repo, tag)
	if err != nil {
		return tagWhy{}, err
	}
	rt.onGitHub[tag] = w
	return w, nil
}

// remoteRelease is the part of a GitHub release this package reads; a flag
// GitHub leaves out, or sends as anything but a boolean, is refused.
type remoteRelease struct {
	TagName    *string `json:"tag_name"`
	Draft      *bool   `json:"draft"`
	Prerelease *bool   `json:"prerelease"`
}

// releaseOnGitHub reads GitHub's release of tag (already held to the
// release form, so it is safe in the API path) and says why it is not a
// published release. GitHub's 404 is no release for the tag; any other
// failure, or an answer that is not of the release's shape or names
// another tag, refuses.
func releaseOnGitHub(checkout, repo, tag string) (tagWhy, error) {
	const remedy = "publish its release on GitHub, neither a draft nor a prerelease, then run it again"
	out, err := github.API(checkout, nil, "repos/"+repo+"/releases/tags/"+tag)
	if err != nil {
		rerr := remoteError(err, false)
		var hidden *notShownRefusal
		switch {
		case refusesWhole(rerr):
			return tagWhy{}, rerr
		case errors.As(rerr, &hidden) && strings.Contains(err.Error(), "404"):
			return tagWhy{"GitHub reports no release for it", remedy}, nil
		}
		return tagWhy{}, refuse("GitHub could not be read for the release of "+tag+", so whether it is published cannot be told; nothing was recorded",
			"settle what gh reports, then run it again")
	}
	var r remoteRelease
	if err := json.Unmarshal(out, &r); err != nil || r.TagName == nil || r.Draft == nil || r.Prerelease == nil {
		return tagWhy{}, refuse("GitHub's answer for the release of "+tag+" is not the JSON this abcd reads; nothing was recorded",
			"run it again; if it persists, report it")
	}
	switch {
	case *r.TagName != tag:
		return tagWhy{}, refuse("GitHub answered for another release than the one of "+tag+"; nothing was recorded",
			"run it again; if it persists, report it")
	case *r.Draft:
		return tagWhy{"its release on GitHub is a draft", remedy}, nil
	case *r.Prerelease:
		return tagWhy{"its release on GitHub is a prerelease", remedy}, nil
	}
	return tagWhy{}, nil
}

// pickRelease is the release named, which must be a release tag this
// checkout holds that is a published release, or, when none is named, the
// highest such tag.
func pickRelease(rt *releaseTags, release string) (string, error) {
	tags := rt.sorted()
	if release == "" {
		if len(tags) == 0 {
			return "", refuse("this repository has no release tag (vMAJOR.MINOR.PATCH) to publish from", "cut the release that carries the fixes first")
		}
		for _, t := range slices.Backward(tags) {
			w, err := rt.published(t)
			if err != nil {
				return "", err
			}
			if w.why == "" {
				return t, nil
			}
		}
		return "", refuse("no release tag this checkout holds is a published release: one origin holds at the same object, "+
			"whose release GitHub reports neither a draft nor a prerelease",
			"push the release tag to origin and publish its release on GitHub, then run it again")
	}
	if !slices.Contains(tags, release) {
		return "", refuse("there is no tag "+release+" in this repository", "fetch the tags, or name a release tag this checkout holds")
	}
	w, err := rt.published(release)
	if err != nil {
		return "", err
	}
	if w.why != "" {
		return "", refuse(release+" is not a published release: "+w.why, w.remedy)
	}
	return release, nil
}
