package update

// selfupdate.go is what a running drive takes a release through
// (itd-2611062149308723, adr-2611062149324491, proposed): the running file judged
// once, the latest release's tag, the verified bytes staged beside the target, and
// the swap keeping the previous file. Apply stays the person's verb and is not
// used: its receipt and its removal of the old file are for someone at a terminal.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/minio/selfupdate"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/urlguard"
)

// ShapeSourceBuild is the refusal of a build that claims no release tag.
const ShapeSourceBuild = "source-build"

// SelfTarget is the running file as a drive reads it once at its start: its
// resolved path, the release its own build claims, and why it never self-updates,
// nil when it may.
type SelfTarget struct {
	Path    string
	Version string
	Refusal *Refusal
}

// Staged is a release's verified bytes at the staging name beside the target.
type Staged struct {
	Path   string
	SHA256 string
	Signer string
}

// runningRelease is the release this process's own build claims, a seam so a test
// can stand in a build.
var runningRelease = ahoy.RunningRelease

// ResolveSelfTarget judges the running file. Nothing on the network, and nothing a
// stage can write but the file's own kind, which is read before any stage starts.
func ResolveSelfTarget() SelfTarget {
	exe, err := osExecutable()
	if err != nil {
		return SelfTarget{Refusal: selfRefusal("this abcd cannot name its own file, so it does not self-update")}
	}
	// macOS reports the path abcd was invoked through; the file is what it leads to.
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return SelfTarget{Path: exe, Refusal: selfRefusal("this abcd cannot resolve its own file, so it does not self-update")}
	}
	t := SelfTarget{Path: resolved}
	if t.Version, err = runningRelease(); err != nil {
		t.Refusal = &Refusal{Shape: ShapeSourceBuild, Detail: "this abcd is a source build, which never self-updates",
			Remedy: "start the drive from a release binary to have it take new releases"}
		return t
	}
	kind := ahoy.UpdateTargetFile
	switch fi, err := os.Lstat(resolved); {
	case err != nil || !fi.Mode().IsRegular():
		kind = ahoy.UpdateTargetForeign
	case ahoy.IsDevShim(resolved):
		kind = ahoy.UpdateTargetDevShim
	default:
		if _, under := ahoy.PluginRootContaining(resolved); under {
			kind = ahoy.UpdateTargetPluginRoot
		}
	}
	t.Refusal = Plan(ahoy.UpdateTarget{Path: resolved, ResolvedPath: resolved, Kind: kind})
	return t
}

func selfRefusal(detail string) *Refusal {
	return &Refusal{Shape: "unclassified-target", Detail: detail, Remedy: "see `abcd ahoy` for the install's health"}
}

// NewSelfUpdater is the updater a running drive asks through: NewGitHubUpdater's
// origin, policy and checks, over a client that holds the transport overrides out
// without unsetting them. A drive goes on to start git, gh and every agent, which
// need the person's proxy, so the process-wide scrub `abcd update` makes in its own
// short life would change what they run on.
func NewSelfUpdater() *Updater {
	roots := urlguard.OverrideFreeRoots()
	u := buildUpdater(releaseOrigin, urlguard.BlockedIP, "abcd-"+runtime.GOOS+"-"+runtime.GOARCH, false,
		vintage.NewGitHubReleaseFetcherWithRoots(roots), setOverrides(), roots)
	u.provenance = newGitHubAttestations(u)
	u.allowAssetHosts()
	return u
}

// Candidate asks the origin for its latest release, one request, and reports
// whether it is a newer stable release than current.
func (u *Updater) Candidate(current string) (tag string, newer bool, err error) {
	tag, err = u.ResolveTag("")
	if err != nil {
		return "", false, err
	}
	return tag, ahoy.ReleaseOlder(current, tag) && !strings.ContainsAny(tag, "-+"), nil
}

// StageRelease downloads tag, holds it to FetchVerified's three checks, and writes
// it to the staging name beside target. A failure leaves nothing there.
func (u *Updater) StageRelease(target, tag string, progress io.Writer) (Staged, error) {
	r, err := u.FetchVerified(tag, progress)
	if err != nil {
		return Staged{}, err
	}
	path, err := stageVerified(target, r.Bytes)
	if err != nil {
		return Staged{}, &NoVerdictError{Err: fmt.Errorf("staging %s beside %s: %w", u.assetName, fsutil.RedactHome(target), err)}
	}
	return Staged{Path: path, SHA256: r.SHA256, Signer: r.Signer}, nil
}

// NoVerdictError is a failure that says nothing about the release: a network fault,
// a status other than not-found, a body cut short, a trusted root that could not be
// fetched, a staging file that could not be written. A drive tries again at its next
// check, where a verdict on the release takes that release off the table for the run.
// Its text is the fault alone, without what the caller wrapped it in.
type NoVerdictError struct{ Err error }

func (e *NoVerdictError) Error() string { return e.Err.Error() }

// Unwrap lets a caller still read the fault underneath.
func (e *NoVerdictError) Unwrap() error { return e.Err }

// faultReader marks a body that breaks off as no verdict: the end of a body is the
// one thing a read of it may report and still have said something of the release.
type faultReader struct{ r io.Reader }

func (f faultReader) Read(p []byte) (int, error) {
	n, err := f.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		err = &NoVerdictError{Err: err}
	}
	return n, err
}

// verifyKeptProvenance is VerifyReleaseProvenance behind a seam for this package's tests.
var verifyKeptProvenance = VerifyReleaseProvenance

// ReadKept reads the kept copy, refusing anything but a regular file at that name.
func ReadKept(keepAs string) ([]byte, error) {
	f, err := os.OpenFile(keepAs, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxAssetBytes {
		return nil, fmt.Errorf("%s is not a regular file of a binary's size", fsutil.RedactHome(keepAs))
	}
	return io.ReadAll(io.LimitReader(f, maxAssetBytes))
}

// VerifyKept reads the kept copy and holds it to the release it was: its own build claim
// must name tag and the build provenance for exactly those bytes must verify. What cannot
// be verified, offline included, is refused. The bytes it returns are the ones it verified.
func VerifyKept(keepAs, tag string) ([]byte, error) {
	data, err := ReadKept(keepAs)
	if err != nil {
		return nil, err
	}
	if err := checkReleaseClaim(data, tag); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if _, err := verifyKeptProvenance(tag, sum[:]); err != nil {
		return nil, err
	}
	return data, nil
}

// KeptPath is the name the previous binary is kept under, beside target.
func KeptPath(target string) string {
	return filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".previous")
}

// CheckStaged holds the file at path to sum and to being a regular file, immediately
// before it is run or swapped.
func CheckStaged(path, sum string) error {
	changed := &NoVerdictError{Err: errors.New("the staging file changed after it was verified")}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return changed
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return changed
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil || hex.EncodeToString(h.Sum(nil)) != strings.ToLower(sum) {
		return changed
	}
	return nil
}

// CommitStaged swaps the staged file into place keeping the previous one at keepAs,
// after reading the staging file's digest again: the proof ran on the same account,
// so what was verified is not taken to be what stands there now. A digest that
// differs removes the file and swaps nothing.
func CommitStaged(target string, st Staged, keepAs string) error {
	if err := CheckStaged(st.Path, st.SHA256); err != nil {
		_ = os.Remove(st.Path)
		return &NoVerdictError{Err: fmt.Errorf("%w, so it was removed and nothing swapped", err)}
	}
	if err := selfupdate.CommitBinary(selfupdate.Options{TargetPath: target, OldSavePath: keepAs}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("the swap failed and so did putting the previous file back, so the binary at %s may be missing; the previous file is at %s: %v (rollback: %v)",
				fsutil.RedactHome(target), fsutil.RedactHome(keepAs), err, rerr)
		}
		_ = os.Remove(st.Path)
		return fmt.Errorf("the swap failed and the previous file is back in place: %w", err)
	}
	return nil
}

// RestoreKept puts data, the kept copy's bytes as they were checked, back over target,
// empties the kept name, and returns the digest for the install record.
func RestoreKept(target, keepAs string, data []byte) (string, error) {
	staged, err := stageVerified(target, data)
	if err != nil {
		return "", err
	}
	if err := os.Rename(staged, target); err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	_ = os.Remove(keepAs)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
