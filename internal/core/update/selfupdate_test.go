package update

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/root"
)

// selfFixture is a running file at a path the test owns: the running executable
// is pinned at it, and the build's own claim is what claim says. Nothing real is
// read or written.
func selfFixture(t *testing.T, rel string, body []byte, claim string, claimErr error) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o755); err != nil {
		t.Fatal(err)
	}
	prevExe, prevRel := osExecutable, runningRelease
	osExecutable = func() (string, error) { return path, nil }
	runningRelease = func() (string, error) { return claim, claimErr }
	t.Cleanup(func() { osExecutable, runningRelease = prevExe, prevRel })
	return path
}

// TestAReleaseBuildInAPlainFileMaySelfUpdate: the one admitted shape, reached
// through a link, is judged at the file the link leads to.
func TestAReleaseBuildInAPlainFileMaySelfUpdate(t *testing.T) {
	path := selfFixture(t, "bin/abcd", []byte("release"), "v0.17.0", nil)
	link := filepath.Join(filepath.Dir(path), "abcd-link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	osExecutable = func() (string, error) { return link, nil }
	got := ResolveSelfTarget()
	if got.Refusal != nil {
		t.Fatalf("refused: %+v", got.Refusal)
	}
	if got.Path != path || got.Version != "v0.17.0" {
		t.Fatalf("target %+v, want the resolved file %s at v0.17.0", got, path)
	}
}

// TestASourceBuildNeverSelfUpdates is the "source build" row: a build claiming no
// release tag is refused in its own shape, with no release read.
func TestASourceBuildNeverSelfUpdates(t *testing.T) {
	selfFixture(t, "bin/abcd", []byte("built here"), "", errors.New("its build metadata names no release tag"))
	got := ResolveSelfTarget()
	if got.Refusal == nil || got.Refusal.Shape != "source-build" {
		t.Fatalf("refusal %+v, want source-build", got.Refusal)
	}
	if !strings.Contains(got.Refusal.Detail, "source build") {
		t.Fatalf("the detail says what it is: %q", got.Refusal.Detail)
	}
}

// TestARunningFileAnotherMechanismOwnsIsRefused is the "plugin root, Cellar, shim"
// row: refused in Plan's shapes, whatever release it claims.
func TestARunningFileAnotherMechanismOwnsIsRefused(t *testing.T) {
	t.Run("plugin root", func(t *testing.T) {
		path := selfFixture(t, "plugin/abcd", []byte("release"), "v0.17.0", nil)
		if err := os.Mkdir(filepath.Join(filepath.Dir(path), "hooks"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := ResolveSelfTarget(); got.Refusal == nil || got.Refusal.Shape != "plugin-root" {
			t.Fatalf("got %+v", got.Refusal)
		}
	})
	t.Run("cellar", func(t *testing.T) {
		path := selfFixture(t, "Cellar/abcd/0.17.0/bin/abcd", []byte("release"), "v0.17.0", nil)
		prev := brewCellarPrefixes
		brewCellarPrefixes = []string{filepath.Join(strings.SplitN(path, "Cellar", 2)[0], "Cellar") + "/"}
		t.Cleanup(func() { brewCellarPrefixes = prev })
		if got := ResolveSelfTarget(); got.Refusal == nil || got.Refusal.Shape != "package-manager" {
			t.Fatalf("got %+v", got.Refusal)
		}
	})
	t.Run("shim", func(t *testing.T) {
		selfFixture(t, "bin/abcd", []byte("#!/bin/sh\n# abcd-dev-shim (managed by `abcd ahoy install --dev`)\n"), "v0.17.0", nil)
		if got := ResolveSelfTarget(); got.Refusal == nil || got.Refusal.Shape != "dev-shim" {
			t.Fatalf("got %+v", got.Refusal)
		}
	})
}

// latestAnswers is a release origin's latest-release answer, counting the asks.
type latestAnswers struct {
	tag   string
	err   error
	calls int
}

func (l *latestAnswers) LatestTag() (string, error) {
	l.calls++
	return l.tag, l.err
}

// TestOnlyANewerStableTagIsACandidate is the core of the "not newer" row: the
// running tag, an older one and a pre-release are not candidates, one request.
func TestOnlyANewerStableTagIsACandidate(t *testing.T) {
	for _, c := range []struct {
		name, tag string
		newer     bool
	}{
		{"the running tag", "v0.17.0", false},
		{"an older tag", "v0.16.4", false},
		{"a pre-release", "v0.18.0-rc.1", false},
		{"build metadata", "v0.18.0+dirty", false},
		{"a newer stable tag", "v0.18.0", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := &latestAnswers{tag: c.tag}
			u := newUpdater("http://origin.invalid", nil, testAssetName, true, l)
			tag, newer, err := u.Candidate("v0.17.0")
			if err != nil {
				t.Fatal(err)
			}
			if tag != c.tag || newer != c.newer {
				t.Fatalf("candidate %q newer=%v, want %q newer=%v", tag, newer, c.tag, c.newer)
			}
			if l.calls != 1 {
				t.Fatalf("%d requests, want one", l.calls)
			}
		})
	}
	u := newUpdater("http://origin.invalid", nil, testAssetName, true, &latestAnswers{err: errors.New("no answer")})
	if _, newer, err := u.Candidate("v0.17.0"); err == nil || newer {
		t.Fatalf("an origin that does not answer names no candidate: newer=%v err=%v", newer, err)
	}
}

// TestStageReleaseRefusesEachFaultOnItsOwn is the first half of the "update
// package" row: the three checks each refuse on their own fault, and nothing is
// left at the staging name.
func TestStageReleaseRefusesEachFaultOnItsOwn(t *testing.T) {
	stagedName := func(target string) string {
		return filepath.Join(filepath.Dir(target), ".abcd.new")
	}
	t.Run("checksums", func(t *testing.T) {
		o := newTestOrigin(t)
		o.addRelease("v0.18.0", testAssetName, []byte("what-the-manifest-says"))
		o.assets["v0.18.0"][testAssetName] = []byte("other-bytes")
		target := writeTarget(t, []byte("old"))
		_, err := testUpdater(t, o).StageRelease(target, "v0.18.0", nil)
		if err == nil || !strings.Contains(err.Error(), "manifest") {
			t.Fatalf("got %v", err)
		}
		if _, serr := os.Lstat(stagedName(target)); !errors.Is(serr, os.ErrNotExist) {
			t.Fatalf("nothing is staged: %v", serr)
		}
	})
	t.Run("provenance", func(t *testing.T) {
		o := newTestOrigin(t)
		o.addRelease("v0.18.0", testAssetName, []byte("new"))
		u := testUpdater(t, o)
		u.provenance = refusingProvenance{}
		target := writeTarget(t, []byte("old"))
		_, err := u.StageRelease(target, "v0.18.0", nil)
		if err == nil || !strings.Contains(err.Error(), "provenance") {
			t.Fatalf("got %v", err)
		}
		if _, serr := os.Lstat(stagedName(target)); !errors.Is(serr, os.ErrNotExist) {
			t.Fatalf("nothing is staged: %v", serr)
		}
	})
	t.Run("release claim", func(t *testing.T) {
		o := newTestOrigin(t)
		o.addRelease("v0.17.0", testAssetName, []byte("older build"))
		republished(o, "v0.17.0", "v0.18.0")
		claimFirstPublished(t, o)
		target := writeTarget(t, []byte("old"))
		_, err := testUpdater(t, o).StageRelease(target, "v0.18.0", nil)
		if err == nil || !strings.Contains(err.Error(), "v0.17.0") {
			t.Fatalf("got %v", err)
		}
		if _, serr := os.Lstat(stagedName(target)); !errors.Is(serr, os.ErrNotExist) {
			t.Fatalf("nothing is staged: %v", serr)
		}
	})
}

// TestCommitStagedKeepsThePreviousFileAndRefusesAChangedDigest is the second half
// of the "update package" row.
func TestCommitStagedKeepsThePreviousFileAndRefusesAChangedDigest(t *testing.T) {
	stage := func(t *testing.T) (string, Staged) {
		t.Helper()
		o := newTestOrigin(t)
		o.addRelease("v0.18.0", testAssetName, []byte("new-bytes"))
		target := writeTarget(t, []byte("old-bytes"))
		st, err := testUpdater(t, o).StageRelease(target, "v0.18.0", nil)
		if err != nil {
			t.Fatal(err)
		}
		if body, err := os.ReadFile(st.Path); err != nil || string(body) != "new-bytes" {
			t.Fatalf("staged %q: %v", body, err)
		}
		if body, _ := os.ReadFile(target); string(body) != "old-bytes" {
			t.Fatalf("staging touched the target: %q", body)
		}
		return target, st
	}
	t.Run("kept", func(t *testing.T) {
		target, st := stage(t)
		keep := KeptPath(target)
		if err := os.WriteFile(keep, []byte("what stood there first"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := CommitStaged(target, st, keep); err != nil {
			t.Fatal(err)
		}
		if body, _ := os.ReadFile(target); string(body) != "new-bytes" {
			t.Fatalf("target holds %q", body)
		}
		if body, _ := os.ReadFile(keep); string(body) != "old-bytes" {
			t.Fatalf("kept copy holds %q", body)
		}
		if _, err := os.Lstat(st.Path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("the staging name is empty after the swap: %v", err)
		}
		if filepath.Base(keep) != ".abcd.previous" {
			t.Fatalf("the kept copy is named %s", keep)
		}
	})
	t.Run("changed", func(t *testing.T) {
		target, st := stage(t)
		if err := os.WriteFile(st.Path, []byte("written after the check"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := CommitStaged(target, st, KeptPath(target)); err == nil {
			t.Fatal("a changed staging file is refused")
		}
		if body, _ := os.ReadFile(target); string(body) != "old-bytes" {
			t.Fatalf("target holds %q", body)
		}
		if _, err := os.Lstat(st.Path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("the changed file is removed: %v", err)
		}
		if _, err := os.Lstat(KeptPath(target)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("nothing was kept, since nothing swapped: %v", err)
		}
	})
}

// TestCheckStagedRefusesALinkAndAChangedFile (F3): fileSHA256 follows a link, so the
// digest alone would pass one that points at the same bytes.
func TestCheckStagedRefusesALinkAndAChangedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "staged")
	if err := os.WriteFile(path, []byte("bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckStaged(path, sum); err != nil {
		t.Fatalf("the verified file passes: %v", err)
	}
	var none *NoVerdictError
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := CheckStaged(link, sum); !errors.As(err, &none) {
		t.Fatalf("a link: %v", err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckStaged(path, sum); !errors.As(err, &none) {
		t.Fatalf("a changed file: %v", err)
	}
	target := writeTarget(t, []byte("old"))
	stagedName := filepath.Join(filepath.Dir(target), ".abcd.new")
	if err := os.Symlink(link, stagedName); err != nil {
		t.Fatal(err)
	}
	if err := CommitStaged(target, Staged{Path: stagedName, SHA256: sum}, KeptPath(target)); !errors.As(err, &none) {
		t.Fatalf("a link at the staging name is not swapped in: %v", err)
	}
	if body, _ := os.ReadFile(target); string(body) != "old" {
		t.Fatalf("target holds %q", body)
	}
}

// TestRestoreKeptPutsThePreviousFileBack: the exec-refused and fallback rows' core.
func TestRestoreKeptPutsThePreviousFileBack(t *testing.T) {
	target := writeTarget(t, []byte("new-bytes"))
	keep := KeptPath(target)
	if _, err := ReadKept(keep); err == nil {
		t.Fatal("no kept copy is a failure, and the target stays")
	}
	if body, _ := os.ReadFile(target); string(body) != "new-bytes" {
		t.Fatalf("target holds %q", body)
	}
	if err := os.WriteFile(keep, []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := RestoreKept(target, keep, []byte("old-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(target); string(body) != "old-bytes" {
		t.Fatalf("target holds %q", body)
	}
	if want, _ := fileSHA256(target); digest != want {
		t.Fatalf("digest %q, want %q", digest, want)
	}
	if _, err := os.Lstat(keep); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the kept name is empty: %v", err)
	}
}

// TestRestoreKeptWritesTheBytesItWasGiven (F1): what goes in place is what was verified,
// whatever stands at the kept name by then.
func TestRestoreKeptWritesTheBytesItWasGiven(t *testing.T) {
	target := writeTarget(t, []byte("new-bytes"))
	keep := KeptPath(target)
	if err := os.WriteFile(keep, []byte("something else now"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreKept(target, keep, []byte("old-bytes")); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(target); string(body) != "old-bytes" {
		t.Fatalf("target holds %q", body)
	}
}

// TestVerifyKeptHoldsTheCopyToItsRelease (F1): the kept file's own build claim names the
// tag and its provenance passes, or nothing is returned; each is held on its own.
func TestVerifyKeptHoldsTheCopyToItsRelease(t *testing.T) {
	const tag = "v0.17.0"
	keptFile := func(t *testing.T, body string) string {
		t.Helper()
		keep := KeptPath(writeTarget(t, []byte("new")))
		if err := os.WriteFile(keep, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		return keep
	}
	seam := func(t *testing.T, claim error, prov func(string, []byte) (string, error)) *int {
		t.Helper()
		prevC, prevP := checkReleaseClaim, verifyKeptProvenance
		t.Cleanup(func() { checkReleaseClaim, verifyKeptProvenance = prevC, prevP })
		calls := 0
		checkReleaseClaim = func([]byte, string) error { return claim }
		verifyKeptProvenance = func(tag string, d []byte) (string, error) {
			calls++
			return prov(tag, d)
		}
		return &calls
	}
	passes := func(string, []byte) (string, error) { return "signer", nil }
	t.Run("a claim that does not name it spends no request", func(t *testing.T) {
		calls := seam(t, errors.New("it names no release"), passes)
		if _, err := VerifyKept(keptFile(t, "not-a-release"), tag); err == nil || *calls != 0 {
			t.Fatalf("got %v, %d requests", err, *calls)
		}
	})
	t.Run("provenance that does not hold", func(t *testing.T) {
		seam(t, nil, func(string, []byte) (string, error) { return "", errors.New("no attestation") })
		if _, err := VerifyKept(keptFile(t, "old"), tag); err == nil {
			t.Fatal("refused")
		}
	})
	t.Run("offline", func(t *testing.T) {
		seam(t, nil, func(string, []byte) (string, error) { return "", &NoVerdictError{Err: errors.New("no route")} })
		if _, err := VerifyKept(keptFile(t, "old"), tag); err == nil {
			t.Fatal("what cannot be verified is refused")
		}
	})
	t.Run("a link", func(t *testing.T) {
		seam(t, nil, passes)
		real := keptFile(t, "old")
		keep := KeptPath(writeTarget(t, []byte("new")))
		if err := os.Symlink(real, keep); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyKept(keep, tag); err == nil {
			t.Fatal("a link is refused")
		}
	})
	t.Run("the release", func(t *testing.T) {
		seam(t, nil, passes)
		got, err := VerifyKept(keptFile(t, "old"), tag)
		if err != nil || string(got) != "old" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
}

// TestTheDrivesUpdaterLeavesTheProcessEnvironment: `abcd update` scrubs the
// transport overrides from its own short-lived process, but a drive goes on to
// start git, gh and every agent, which need the person's proxy. The client still
// honours none of them.
func TestTheDrivesUpdaterLeavesTheProcessEnvironment(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.invalid:3128")
	u := NewSelfUpdater()
	if u == nil {
		t.Fatal("no updater")
	}
	if got := os.Getenv("HTTPS_PROXY"); got == "" {
		t.Fatal("the proxy is gone from the process, and from every child it starts")
	}
	if len(u.envIgnored) != 1 || u.envIgnored[0] != "HTTPS_PROXY" {
		t.Fatalf("what it ignored: %v", u.envIgnored)
	}
	if u.origin != releaseOrigin || u.provenance == nil || u.latest == nil {
		t.Fatalf("the updater as it ships: origin %q provenance %v latest %v", u.origin, u.provenance, u.latest)
	}
}

// faultyOrigin answers the path ending in name with fault and everything else as o does.
func faultyOrigin(t *testing.T, o *testOrigin, name string, fault http.HandlerFunc) *Updater {
	t.Helper()
	real := o.srv.Config.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/"+name) {
			fault(w, r)
			return
		}
		real.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	u := newUpdater(srv.URL, nil, testAssetName, true, nil)
	u.provenance = &recordingProvenance{}
	return u
}

// TestAFaultThatSaysNothingOfTheReleaseIsNoVerdict is the core of the "fetch fault"
// row: a status other than not-found, a connection dropped and a body cut short leave
// the release unjudged, with nothing staged.
func TestAFaultThatSaysNothingOfTheReleaseIsNoVerdict(t *testing.T) {
	status := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }
	}
	dropped := func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}
	cutShort := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("partial"))
	}
	for _, c := range []struct {
		name, file string
		fault      http.HandlerFunc
	}{
		{"5xx at the checksums", "checksums.txt", status(http.StatusServiceUnavailable)},
		{"a rate limit at the checksums", "checksums.txt", status(http.StatusTooManyRequests)},
		{"a refusal at the checksums", "checksums.txt", status(http.StatusForbidden)},
		{"a dropped connection at the checksums", "checksums.txt", dropped},
		{"checksums cut short", "checksums.txt", cutShort},
		{"5xx at the asset", testAssetName, status(http.StatusBadGateway)},
		{"a dropped connection at the asset", testAssetName, dropped},
		{"an asset cut short", testAssetName, cutShort},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := newTestOrigin(t)
			o.addRelease("v0.18.0", testAssetName, []byte("new"))
			u := faultyOrigin(t, o, c.file, c.fault)
			target := writeTarget(t, []byte("old"))
			_, err := u.StageRelease(target, "v0.18.0", nil)
			var none *NoVerdictError
			if !errors.As(err, &none) {
				t.Fatalf("got %v, want no verdict", err)
			}
			if _, serr := os.Lstat(filepath.Join(filepath.Dir(target), ".abcd.new")); !errors.Is(serr, os.ErrNotExist) {
				t.Fatalf("nothing is staged: %v", serr)
			}
		})
	}
}

// TestAStagingFileThatCannotBeWrittenIsNoVerdict: the bytes passed every check, and a
// full disk says nothing of them.
func TestAStagingFileThatCannotBeWrittenIsNoVerdict(t *testing.T) {
	o := newTestOrigin(t)
	o.addRelease("v0.18.0", testAssetName, []byte("new"))
	prev := stagingWriter
	stagingWriter = func(io.Writer) io.Writer { return failingWriter{} }
	t.Cleanup(func() { stagingWriter = prev })
	_, err := testUpdater(t, o).StageRelease(writeTarget(t, []byte("old")), "v0.18.0", nil)
	var none *NoVerdictError
	if !errors.As(err, &none) {
		t.Fatalf("got %v, want no verdict", err)
	}
}

// failingWriter is a disk with no room.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("no space left on device") }

// TestAVerdictOnTheReleaseIsNotDressedAsAFault: what is about the release stays a
// verdict, whatever else is true of the network.
func TestAVerdictOnTheReleaseIsNotDressedAsAFault(t *testing.T) {
	stage := func(t *testing.T, u *Updater) error {
		t.Helper()
		_, err := u.StageRelease(writeTarget(t, []byte("old")), "v0.18.0", nil)
		if err == nil {
			t.Fatal("a release with a fault in it is refused")
		}
		return err
	}
	for name, u := range map[string]func(t *testing.T) *Updater{
		"a digest the manifest does not name": func(t *testing.T) *Updater {
			o := newTestOrigin(t)
			o.addRelease("v0.18.0", testAssetName, []byte("what-the-manifest-says"))
			o.assets["v0.18.0"][testAssetName] = []byte("other-bytes")
			return testUpdater(t, o)
		},
		"no checksums published": func(t *testing.T) *Updater {
			o := newTestOrigin(t)
			o.addRelease("v0.18.0", testAssetName, []byte("new"))
			delete(o.assets["v0.18.0"], "checksums.txt")
			return testUpdater(t, o)
		},
		"an asset the manifest names and the release lacks": func(t *testing.T) *Updater {
			o := newTestOrigin(t)
			o.addRelease("v0.18.0", testAssetName, []byte("new"))
			delete(o.assets["v0.18.0"], testAssetName)
			return testUpdater(t, o)
		},
		"provenance that refuses": func(t *testing.T) *Updater {
			o := newTestOrigin(t)
			o.addRelease("v0.18.0", testAssetName, []byte("new"))
			u := testUpdater(t, o)
			u.provenance = refusingProvenance{}
			return u
		},
	} {
		t.Run(name, func(t *testing.T) {
			var none *NoVerdictError
			if err := stage(t, u(t)); errors.As(err, &none) {
				t.Fatalf("a verdict reads as a fault: %v", err)
			}
		})
	}
}

// TestAProvenanceServiceThatDoesNotAnswerIsNoVerdict: the attestations API and the
// trusted root failing say nothing of the release; an attestation that is not there, or
// does not verify, does.
func TestAProvenanceServiceThatDoesNotAnswerIsNoVerdict(t *testing.T) {
	digest := make([]byte, 32)
	const commit = "0123456789012345678901234567890123456789"
	world := func(t *testing.T, attestations http.HandlerFunc) *githubAttestations {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/git/") {
				_, _ = fmt.Fprintf(w, `{"ref":"refs/tags/v0.18.0","object":{"sha":%q,"type":"commit"}}`, commit)
				return
			}
			attestations(w, r)
		}))
		t.Cleanup(srv.Close)
		g := newReleaseProvenance(releaseOrigin, nil, true)
		parsed, err := url.Parse(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		g.dl.AllowHost(parsed.Host)
		g.endpoint, g.gitEndpoint = srv.URL+"/attestations/", srv.URL+"/git/"
		g.trusted = func() (root.TrustedMaterial, error) { return nil, errors.New("no route to the mirror") }
		return g
	}
	unavailable := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }
	listed := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"attestations":[{"bundle":{}}]}`))
	}
	var none *NoVerdictError
	if _, err := world(t, unavailable).verify("v0.18.0", digest); !errors.As(err, &none) {
		t.Fatalf("a 503 from the attestations API: %v", err)
	}
	if _, err := world(t, listed).verify("v0.18.0", digest); !errors.As(err, &none) {
		t.Fatalf("a trusted root that could not be fetched: %v", err)
	}
	notFound := world(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	if _, err := notFound.verify("v0.18.0", digest); err == nil || errors.As(err, &none) {
		t.Fatalf("no attestation published is a verdict: %v", err)
	}
}
