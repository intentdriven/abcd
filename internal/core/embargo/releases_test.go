package embargo

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// releases_test.go: a release tag counts for a plan only when it is a
// published release: origin holds the tag at the object this checkout
// holds, and GitHub reports a release for it that is neither a draft nor a
// prerelease. Origin is a bare repository standing in for the remote; GitHub
// is the fake gh.

// releaseFixture holds advisory A (ghsaOpen) judged a vulnerability, its fix
// committed and resolved in the public ledger before both tags are made:
// v0.8.0 first, then v0.9.0, each holding the fix.
func releaseFixture(t *testing.T) planFixture {
	t.Helper()
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.resolve(issA, fix)
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	return pf
}

func (pf planFixture) planErr(release string) error {
	_, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: release,
		Now: func() time.Time { return fixedNow.Add(time.Hour) }})
	return err
}

// TestUnpublishedFirstTagIsSkipped: a tag holding the fix that is not a
// published release does not give the patched version: the next published
// tag does. Each way a tag fails to be one is held: origin lacks it, origin
// holds it at another object, GitHub reports no release, a draft, or a
// prerelease.
func TestUnpublishedFirstTagIsSkipped(t *testing.T) {
	for name, make := range map[string]func(pf planFixture){
		"local only": func(pf planFixture) { pf.localTag("v0.8.0") },
		"another object on origin": func(pf planFixture) {
			pf.localTag("v0.8.0")
			pf.push("v0.8.0")
			pf.f.repo.Git("tag", "-f", "-a", "-m", "moved", "v0.8.0")
			pf.release("v0.8.0", releaseJSON("v0.8.0", false, false))
		},
		"no release": func(pf planFixture) {
			pf.localTag("v0.8.0")
			pf.push("v0.8.0")
			pf.g.fail("GET", releasePath("v0.8.0"), "", "gh: Not Found (HTTP 404)")
		},
		"draft": func(pf planFixture) {
			pf.localTag("v0.8.0")
			pf.push("v0.8.0")
			pf.release("v0.8.0", releaseJSON("v0.8.0", true, false))
		},
		"prerelease": func(pf planFixture) {
			pf.localTag("v0.8.0")
			pf.push("v0.8.0")
			pf.release("v0.8.0", releaseJSON("v0.8.0", false, true))
		},
	} {
		t.Run(name, func(t *testing.T) {
			pf := releaseFixture(t)
			make(pf)
			pf.tag("v0.9.0")
			p := pf.plan(t, "")
			if p.Release != "v0.9.0" || len(p.Entries) != 1 || p.Entries[0].FirstTag != "v0.9.0" || p.Entries[0].PatchedVersion != "0.9.0" {
				t.Fatalf("plan %+v", p)
			}
		})
	}
}

// TestPublishedFirstTagCounts: a tag origin holds at the same object, whose
// release GitHub reports published, gives the patched version.
func TestPublishedFirstTagCounts(t *testing.T) {
	pf := releaseFixture(t)
	pf.tag("v0.8.0")
	pf.tag("v0.9.0")
	p := pf.plan(t, "")
	if p.Release != "v0.9.0" || len(p.Entries) != 1 || p.Entries[0].FirstTag != "v0.8.0" {
		t.Fatalf("plan %+v", p)
	}
}

// TestFixInNoPublishedReleaseLeftOut: a fix held only by tags that are not
// published releases, up to the release, is left out with a fixed reason.
func TestFixInNoPublishedReleaseLeftOut(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	pf.tag("v0.7.0")
	r := pf.f.repo
	r.Git("switch", "-q", "-c", "side")
	fix := pf.commit("parser.go", "package parser\n", "fix on a side branch")
	pf.localTag("v0.7.1")
	r.Git("switch", "-q", "main")
	pf.resolve(issA, fix)
	pf.tag("v0.8.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	p := pf.plan(t, "")
	if p.Release != "v0.8.0" || len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaOpen), "no published release up to v0.8.0") {
		t.Fatalf("plan %+v", p)
	}
}

// TestDefaultReleaseIsHighestPublished: with no release named, the plan is
// of the highest release tag that is a published release, not the highest
// tag this checkout holds.
func TestDefaultReleaseIsHighestPublished(t *testing.T) {
	pf := releaseFixture(t)
	pf.tag("v0.9.0")
	pf.commit("other.go", "package other\n", "later")
	pf.localTag("v0.10.0")
	pf.f.repo.Git("tag", "v0.11.0")
	pf.push("v0.11.0")
	pf.release("v0.11.0", releaseJSON("v0.11.0", true, false))
	if p := pf.plan(t, ""); p.Release != "v0.9.0" {
		t.Fatalf("plan %+v", p)
	}
}

// TestNamedReleaseMustBePublished: a release named on the command line that
// is not a published release is refused, naming what it lacks and the
// remedy, and nothing is recorded.
func TestNamedReleaseMustBePublished(t *testing.T) {
	for name, tc := range map[string]struct {
		make func(pf planFixture)
		want []string
	}{
		"local only": {func(pf planFixture) { pf.originPath(); pf.localTag("v0.9.0") }, []string{"origin does not hold", "git push origin v0.9.0"}},
		"another object on origin": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.f.repo.Git("tag", "-f", "-a", "-m", "moved", "v0.9.0")
		}, []string{"another object"}},
		"no release": {func(pf planFixture) {
			pf.localTag("v0.9.0")
			pf.push("v0.9.0")
			pf.g.fail("GET", releasePath("v0.9.0"), "", "gh: Not Found (HTTP 404)")
		}, []string{"no release", "publish its release"}},
		"draft":      {func(pf planFixture) { pf.tag("v0.9.0"); pf.release("v0.9.0", releaseJSON("v0.9.0", true, false)) }, []string{"draft"}},
		"prerelease": {func(pf planFixture) { pf.tag("v0.9.0"); pf.release("v0.9.0", releaseJSON("v0.9.0", false, true)) }, []string{"prerelease"}},
	} {
		t.Run(name, func(t *testing.T) {
			pf := releaseFixture(t)
			tc.make(pf)
			before := readFile(t, pf.s.StatePath())
			err := pf.planErr("v0.9.0")
			for _, w := range append([]string{"v0.9.0 is not a published release"}, tc.want...) {
				mustRefuse(t, err, w)
			}
			if readFile(t, pf.s.StatePath()) != before {
				t.Fatal("a refused plan recorded something")
			}
		})
	}
}

// TestReleaseUnreadableRefusesPlan: when origin cannot be read, or GitHub's
// answer for a release cannot be read or names another tag or does not
// carry its two flags as booleans, whether a tag is a published release
// cannot be told: the plan refuses whole, with a remedy, rather than
// falling back on the checkout's tags, and repeats nothing either sent.
func TestReleaseUnreadableRefusesPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		make func(pf planFixture)
		want string
	}{
		"origin unreachable": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.f.repo.Git("config", "remote.origin.url", filepath.Join(pf.f.repo.Root(), ".git", "no-such-origin.git"))
		}, "git ls-remote --tags origin"},
		"gh fails": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.g.fail("GET", releasePath("v0.9.0"), "", "gh: Server Error (HTTP 502)")
		}, "whether it is published cannot be told"},
		"another tag": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.release("v0.9.0", releaseJSON("v0.9.1", false, false))
		}, "another release"},
		"flag not a boolean": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.release("v0.9.0", `{"tag_name":"v0.9.0","draft":"no","prerelease":false}`)
		}, "not the JSON this abcd reads"},
		"flag missing": {func(pf planFixture) {
			pf.tag("v0.9.0")
			pf.release("v0.9.0", `{"tag_name":"v0.9.0","prerelease":false}`)
		}, "not the JSON this abcd reads"},
	} {
		for _, release := range []string{"", "v0.9.0"} {
			t.Run(name+" "+release, func(t *testing.T) {
				pf := releaseFixture(t)
				tc.make(pf)
				before := readFile(t, pf.s.StatePath())
				err := pf.planErr(release)
				mustRefuse(t, err, tc.want)
				if strings.Contains(err.Error(), "Server Error") || strings.Contains(err.Error(), "no-such-origin") {
					t.Fatalf("the refusal repeats what was sent: %v", err)
				}
				if readFile(t, pf.s.StatePath()) != before {
					t.Fatal("a refused plan recorded something")
				}
			})
		}
	}
}

// TestFirstTagUnreadableRefusesPlan: the release a fix's first tag names is
// read as the plan's own release is; an answer that cannot be read refuses
// the plan whole.
func TestFirstTagUnreadableRefusesPlan(t *testing.T) {
	pf := releaseFixture(t)
	pf.localTag("v0.8.0")
	pf.push("v0.8.0")
	pf.g.fail("GET", releasePath("v0.8.0"), "", "gh: Server Error (HTTP 502)")
	pf.tag("v0.9.0")
	mustRefuse(t, pf.planErr(""), "v0.8.0")
}
