package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The abcd download (itd-2610240823055889, criterion 4, and the product
// thinker's decision of 2026-10-24): when abcd is not on the PATH, setup puts
// it there itself in the same run, after one question naming the download,
// so the status line, which needs a PATH entry to run, never depends on the
// README's install line (iss-2610240823574047). The download is the release
// binary for this platform, checked the way the README's install line and
// abcd update check it: against the release's checksums.txt, then against
// the build provenance the release workflow signed (update.FetchVerified,
// which the front door supplies, since update imports this package). It is
// fetched only on a yes to that question, the fetch its documented meaning.

// DownloadQuestionID is the download question's id, the id an answers file
// answers it by.
const DownloadQuestionID = "abcd.download"

// DownloadedRelease is a release binary fetched and verified: its bytes, the
// sha256 checksums.txt names for them, and the signer of its build
// provenance.
type DownloadedRelease struct {
	Bytes  []byte
	SHA256 string
	Signer string
}

// DownloadRelease fetches the release tag's binary for this platform and
// verifies it before returning it; any failure is an error and nothing is
// returned.
type DownloadRelease func(tag string) (DownloadedRelease, error)

// downloadLead opens the download question's explanation.
const downloadLead = "abcd is not on your PATH, so the status line and the commit hooks have nothing to run."

// downloadAskPrefix opens the download question's one line.
const downloadAskPrefix = "Download abcd "

// downloadQuestion is the download question: the explanation, then its one
// line naming the release and where it is written.
func downloadQuestion(tag, target string) string {
	return downloadLead + " Setup downloads the abcd release " + tag + " for this computer from " + core.ReleaseOrigin +
		", checks it against the release's checksums.txt and against the build record the release workflow signed, " +
		"writes it to " + displayPath(target) + " and records it in " + abcdhome.Display("path-entry") +
		"; declining writes nothing, and the README's install line does the same by hand.\n" +
		downloadAskPrefix + tag + " to " + displayPath(target) + "?"
}

// isDownloadQuestion reports whether text is the download question.
func isDownloadQuestion(text string) bool {
	_, ask := splitConfirm(text)
	return strings.HasPrefix(text, downloadLead) && strings.HasPrefix(ask, downloadAskPrefix)
}

// releaseTagRe is the shape of a release tag.
var releaseTagRe = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// abcdModule is the module path of abcd's own source checkout.
const abcdModule = "github.com/intentdriven/abcd"

// downloadTag is the release setup downloads: the one this abcd was stamped
// as, or, in a source checkout of abcd, where a development build carries no
// release, the newest release tag the checkout holds, read from disk. "" when
// neither names one, and the download is not offered. A seam for tests.
var downloadTag = func(cwd string) string {
	if releaseTagRe.MatchString(core.Version) {
		return core.Version
	}
	if checkoutModule(cwd) != abcdModule {
		return ""
	}
	out, err := gitutil.Run(cwd, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil {
		return ""
	}
	for _, t := range strings.Split(out, "\n") {
		if t = strings.TrimSpace(t); releaseTagRe.MatchString(t) {
			return t
		}
	}
	return ""
}

// downloadOffer is the release and the target the download question offers,
// or ok false where it is not offered: under --dev (the shim is the entry),
// with no downloader wired, where the PATH target already holds something, or
// where the plugin's verified cache can supply the copy itself.
func downloadOffer(cwd string, det DetectionResult, opts InstallOptions, target string) (tag string, ok bool) {
	if opts.Dev || opts.DownloadAbcd == nil || target == "" {
		return "", false
	}
	if classifyBinTarget(target, det.pluginRoot) != binTargetAbsent {
		return "", false
	}
	if det.pluginRoot != "" && cachePresent(pluginDataDir(det.pluginRoot).dir, cwd) {
		return "", false
	}
	tag = downloadTag(cwd)
	return tag, tag != ""
}

// stepDownload asks the download question and, on a yes, writes the verified
// release to the PATH target and records it, before the status line is
// offered, so a yes wires the status line in the same run. Every way of
// writing nothing is said.
func (a *applyCtx) stepDownload() {
	tag, ok := downloadOffer(a.cwd, a.det, a.opts, a.binTarget)
	if !ok {
		return
	}
	const did = "did not put abcd on your PATH"
	if a.autoYes {
		a.refuse(did + ": --yes never downloads a program; run abcd ahoy install at a terminal to be asked")
		return
	}
	if !a.prompter.Confirm(downloadQuestion(tag, a.binTarget)) {
		a.refuse(did + ": the download of abcd " + tag + " was declined; the README's install line puts it there by hand")
		return
	}
	// Past the yes, every way of writing nothing is a step left for the
	// person in the default report, with why, not only a note.
	failed := func(why string) {
		a.refuse(did + ": " + why)
		a.needYou("Put abcd on your PATH with the README's install line, then run abcd ahoy install again",
			"you said yes to the download of abcd "+tag+", and "+why+".")
	}
	rel, err := a.opts.DownloadAbcd(tag)
	if err != nil {
		failed("the download of abcd " + tag + " failed or did not verify, so nothing was written: " + errText(err))
		return
	}
	sum := sha256.Sum256(rel.Bytes)
	got := hex.EncodeToString(sum[:])
	if rel.SHA256 != "" && got != strings.ToLower(rel.SHA256) {
		failed("the downloaded abcd " + tag + " does not hash to the checksum it was verified against, so nothing was written")
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.binTarget), 0o755); err != nil {
		failed(displayPath(filepath.Dir(a.binTarget)) + " could not be created: " + errText(err))
		return
	}
	// Re-checked at the write: an entry that appeared while the question was
	// open is never replaced.
	if classifyBinTarget(a.binTarget, a.det.pluginRoot) != binTargetAbsent {
		failed(displayPath(a.binTarget) + " appeared while the question was open, and it is left as it is")
		return
	}
	if err := fsutil.WriteFileAtomic(a.binTarget, rel.Bytes, 0o755); err != nil {
		failed(displayPath(a.binTarget) + " could not be written: " + errText(err))
		return
	}
	a.verifiedCopy = a.binTarget
	a.downloaded = true
	a.note(writeCommandEntry, a.binTarget)
	a.recordEntry(a.binTarget, got)
}

// swapDownloadTag replaces the release tag seam and returns a restore func.
func swapDownloadTag(f func(cwd string) string) (restore func()) {
	prev := downloadTag
	downloadTag = f
	return func() { downloadTag = prev }
}
