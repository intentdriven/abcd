package embargo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// resolvedRel is where a release's tree holds the resolved records, slash-separated.
const resolvedRel = ledgerRel + "/resolved"

// maxRecordBytes caps a resolved record read out of a release's tree, and
// maxTagListing a tag listing.
const (
	maxRecordBytes = 1 << 20
	maxTagListing  = 16 << 20
)

var commitRefRe = regexp.MustCompile(`\A[0-9a-f]{7,64}\z`)

// PublishRequest asks for the publication plan of the release tag Release
// (empty: the highest vMAJOR.MINOR.PATCH tag) and, with Approve equal to the
// plan's digest, for its publication. A host passes Approve only after the
// person, in that session, has seen the plan and said yes to it
// (adr-2610120558328615, decision 4). Now is the clock; nil is the system clock.
type PublishRequest struct {
	Home     string
	RepoRoot string
	Release  string
	Approve  string
	Now      func() time.Time
}

// Package is an advisory's affected package, as GitHub names it.
type Package struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// PlannedVulnerability is one affected package as the plan will set it, in
// the shape GitHub's update takes.
type PlannedVulnerability struct {
	Package                Package  `json:"package"`
	VulnerableVersionRange string   `json:"vulnerable_version_range"`
	PatchedVersions        string   `json:"patched_versions"`
	VulnerableFunctions    []string `json:"vulnerable_functions"`
}

// PlanEntry is one advisory the plan publishes: its record, the commit that
// fixed it, the first release tag holding that commit, the patched version it
// gives, and each affected package's versions. ForgeState is the state GitHub
// reported; TextDigest binds the advisory's text as read.
type PlanEntry struct {
	GHSAID          string                 `json:"ghsa_id"`
	IssueID         string                 `json:"issue_id"`
	FixCommit       string                 `json:"fix_commit"`
	FirstTag        string                 `json:"first_tag"`
	PatchedVersion  string                 `json:"patched_version"`
	ForgeState      string                 `json:"forge_state"`
	Vulnerabilities []PlannedVulnerability `json:"vulnerabilities"`
	TextDigest      string                 `json:"text_digest"`
	// StoreEvidence marks an entry whose fix commit is taken from its
	// private record, resolved in the store's own ledger, because the
	// release carries no public record of it. It is omitted when false, so a
	// public-record entry, and the digest over it, is what it always was.
	StoreEvidence bool `json:"store_evidence,omitempty"`
}

// LeftOut is an advisory judged a vulnerability that the plan does not
// publish, and why.
type LeftOut struct {
	GHSAID  string `json:"ghsa_id"`
	IssueID string `json:"issue_id"`
	Reason  string `json:"reason"`
}

// Restore is an advisory published on GitHub whose private record is
// resolved in the store and not yet in the checkout's public ledger:
// RestoreWith is the command, run from the checkout's root, that copies the
// record there; when it is empty, Reason says why none is named.
type Restore struct {
	GHSAID      string `json:"ghsa_id"`
	IssueID     string `json:"issue_id"`
	RestoreWith string `json:"restore_with,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// PublicationPlan is what publication would do. Digest is the approval it
// takes, and ApproveWith the command that gives it (empty when nothing is
// planned).
type PublicationPlan struct {
	Repo    string      `json:"repo"`
	Release string      `json:"release"`
	Entries []PlanEntry `json:"entries"`
	LeftOut []LeftOut   `json:"left_out"`
	// Unread is each held advisory whose state GitHub could not tell, left
	// as the store holds it (readHeld).
	Unread []UnreadAdvisory `json:"unread"`
	// Restores is each restore the store's state owes (restoresOwed): it is
	// not bound into the digest, since approving a plan approves what it
	// publishes.
	Restores    []Restore `json:"restores"`
	Digest      string    `json:"digest"`
	ApproveWith string    `json:"approve_with"`
}

// planPublication builds the publication plan and changes nothing on GitHub:
// the only requests it sends are reads of the advisories it plans. In the
// store it records only what those reads report: an advisory GitHub already
// holds published or closed.
func planPublication(req PublishRequest) (PublicationPlan, error) {
	s, st, err := openExisting(req.Home, req.RepoRoot)
	if err != nil {
		return PublicationPlan{}, err
	}
	return planFor(s, &st, req.Home, req.Release, clock(req.Now))
}

// planFor builds the plan of release over the store's state. First every
// advisory the store holds that is neither published nor closed, judged or
// not, is read from GitHub once (readHeld), and an end GitHub reports for
// one is recorded and saved: a run that published or closed it on GitHub but
// could not record it, or a hand on GitHub, is reconciled here instead of
// holding the advisory under embargo for ever; one GitHub cannot say is
// listed in Unread and left as it is. Then, for each advisory judged a
// vulnerability and neither published nor closed before this plan, in id
// order: one GitHub reports published or closed is left out, saying so; its
// record must be resolved in the release's tree and name its fix commit, or,
// when the release carries no public record of it (the security rule holds
// the record out of every committed tree until the advisory is published),
// its private record must be resolved in the store's own ledger naming the
// full id of its fix commit (storeFixCommit). The first release tag holding
// that commit gives the patched version; the advisory as GitHub reports it
// must carry its package, an affected range in GitHub's form, and text the
// outbound scan passes. Each that fails a step is left out with the reason.
func planFor(s Store, st *State, home, release string, now time.Time) (PublicationPlan, error) {
	if st.Repo == "" {
		return PublicationPlan{}, refuse("the embargo store does not name its repository", "run `abcd embargo intake` once, then run it again")
	}
	release, err := pickRelease(s.Checkout, release)
	if err != nil {
		return PublicationPlan{}, err
	}
	resolved, err := resolvedRecords(s.Checkout, release)
	if err != nil {
		return PublicationPlan{}, err
	}
	p := PublicationPlan{Repo: st.Repo, Release: release, Entries: []PlanEntry{}, LeftOut: []LeftOut{}, Unread: []UnreadAdvisory{}}
	var candidates []Advisory
	for _, a := range st.Advisories {
		if a.Verdict != nil && *a.Verdict == VerdictVulnerability && a.PublishedAt == nil && a.ClosedAt == nil {
			candidates = append(candidates, a)
		}
	}
	slices.SortFunc(candidates, func(a, b Advisory) int { return strings.Compare(a.GHSAID, b.GHSAID) })
	answers, ended, unread, err := readHeld(s, st, nil, now)
	if err != nil {
		return PublicationPlan{}, err
	}
	p.Unread = unread
	unreadWhy := map[string]string{}
	for _, u := range unread {
		unreadWhy[u.GHSAID] = u.Reason
	}
	for _, a := range candidates {
		remote, read := answers[a.GHSAID]
		if !read {
			remote.State = ""
		}
		e, why, err := planEntry(s, release, resolved, a, remote, unreadWhy[a.GHSAID], othersDenySet(*st, a.GHSAID))
		if err != nil {
			return PublicationPlan{}, err
		}
		if why != "" {
			p.LeftOut = append(p.LeftOut, LeftOut{GHSAID: a.GHSAID, IssueID: a.IssueID, Reason: why})
			continue
		}
		p.Entries = append(p.Entries, e)
	}
	if len(ended) > 0 {
		if err := s.SaveState(*st); err != nil {
			return PublicationPlan{}, err
		}
	}
	p.Restores = restoresOwed(home, s, *st)
	p.Digest = planDigest(p)
	if len(p.Entries) > 0 {
		p.ApproveWith = "abcd embargo publish --release " + release + " --approve " + p.Digest
	}
	return p, nil
}

// pickRelease is the release tag named, which must be a release tag this
// repository holds, or the highest one when none is named.
func pickRelease(checkout, release string) (string, error) {
	out, err := gitutil.RunCappedBytes(checkout, maxTagListing, "tag", "--list", "v*")
	if err != nil {
		return "", refuse("git could not list the release tags", "settle what git reports, then run it again")
	}
	tags := sortReleaseTags(strings.Split(string(out), "\n"))
	if release == "" {
		if len(tags) == 0 {
			return "", refuse("this repository has no release tag (vMAJOR.MINOR.PATCH) to publish from", "cut the release that carries the fixes first")
		}
		return tags[len(tags)-1], nil
	}
	if !isReleaseTag(release) {
		return "", refuse("--release is not a release tag of the form vMAJOR.MINOR.PATCH", "name the release tag, for example v0.9.0")
	}
	if !slices.Contains(tags, release) {
		return "", refuse("there is no tag "+release+" in this repository", "fetch the tags, or name a release tag this checkout holds")
	}
	return release, nil
}

// resolvedRecords lists the record files in the release's resolved/ folder.
func resolvedRecords(checkout, release string) ([]string, error) {
	out, err := gitutil.RunCappedBytes(checkout, maxTagListing, "ls-tree", "-z", "--name-only", "refs/tags/"+release, "--", resolvedRel+"/")
	if err != nil {
		return nil, refuse("git could not list the resolved records of "+release, "settle what git reports, then run it again")
	}
	var paths []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 {
			paths = append(paths, string(p))
		}
	}
	return paths, nil
}

// planEntry plans one advisory, or says why it is left out. remote is the
// advisory as readHeld read it from GitHub, its State empty when it could
// not be read (unreadWhy says why). others is every other advisory still
// under embargo, which the advisory's text must not name: publishing it
// would disclose them. An advisory in the same plan counts, since
// publication is not one atomic act. One GitHub reports published or closed
// already (readHeld has recorded it) is left out, saying so. An error is a failure that refuses the whole plan (a git
// request that fails, a text the scanner cannot judge).
func planEntry(s Store, release string, resolved []string, a Advisory, remote remoteAdvisory, unreadWhy string, others denySet) (e PlanEntry, why string, err error) {
	rec := ""
	for _, p := range resolved {
		if isRecordOf(path.Base(p), a.IssueID) {
			rec = p
			break
		}
	}
	if remote.State == "published" || remote.State == "closed" {
		return PlanEntry{}, "GitHub reports it " + remote.State + " already; its " + remote.State + "_at is recorded from that", nil
	}
	var fix, storeRec string
	if rec != "" {
		fix, why = fixCommit(s.Checkout, release, rec)
	} else if fix, storeRec, why = s.storeFixCommit(a.IssueID); why != "" {
		why = "its record is not resolved in " + release + ", " + why
	}
	if why != "" {
		return PlanEntry{}, why, nil
	}
	first, why, err := firstReleaseTag(s.Checkout, fix, release)
	if err != nil || why != "" {
		return PlanEntry{}, why, err
	}
	patched := patchedVersion(first)

	switch remote.State {
	case "triage", "draft":
	case "":
		return PlanEntry{}, unreadWhy, nil
	default:
		return PlanEntry{}, "GitHub reports it in a state this abcd does not know", nil
	}
	const noPackage = "GitHub carries no package for it; set the package on GitHub, then run it again"
	if len(remote.Vulnerabilities) == 0 {
		return PlanEntry{}, noPackage, nil
	}
	var vulns []PlannedVulnerability
	for _, v := range remote.Vulnerabilities {
		if v.Package == nil || strings.TrimSpace(v.Package.Ecosystem) == "" || strings.TrimSpace(v.Package.Name) == "" {
			return PlanEntry{}, noPackage, nil
		}
		rng, err := affectedRange(v.VulnerableVersionRange, patched)
		if err != nil {
			return PlanEntry{}, err.Error(), nil
		}
		fns := v.VulnerableFunctions
		if fns == nil {
			fns = []string{}
		}
		vulns = append(vulns, PlannedVulnerability{Package: *v.Package, VulnerableVersionRange: rng, PatchedVersions: patched, VulnerableFunctions: fns})
	}
	for _, part := range []struct{ name, text string }{{"summary", remote.Summary}, {"description", remote.description()}} {
		f, err := scanOutbound(s.Checkout, "the advisory's "+part.name, part.text, true)
		if err != nil {
			return PlanEntry{}, "", err
		}
		if f != nil {
			return PlanEntry{}, fmt.Sprintf("its %s on GitHub carries a %s on line %d; correct it there, then run it again", part.name, f.kind, f.line), nil
		}
		if m := others.matches(part.text); len(m) > 0 {
			return PlanEntry{}, fmt.Sprintf("its %s on GitHub names another advisory still under embargo (%s on line %d), which publishing it would disclose; "+
				"correct it there, or publish the other first, then run it again", part.name, m[0].kind, strings.Count(part.text[:m[0].start], "\n")+1), nil
		}
	}
	return PlanEntry{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommit: fix, FirstTag: first, PatchedVersion: patched,
		ForgeState: remote.State, Vulnerabilities: vulns, TextDigest: textDigest(remote),
		StoreEvidence: storeRec != ""}, "", nil
}

// isRecordOf reports whether a ledger file name is the record of issID.
func isRecordOf(base, issID string) bool {
	return base == issID+".md" || (strings.HasPrefix(base, issID+"-") && strings.HasSuffix(base, ".md"))
}

// storeFixCommit is the fix commit issID's private record names, resolved in
// the store's own ledger, and the record's path in the store: the evidence
// the plan takes when the security rule holds the record out of the
// release's tree. why ("" when it holds) is worded to follow "its record is
// not resolved in <release>, ".
//
// The store is a trust boundary, so nothing read from it is used before it
// is judged: the record must pass storeRecord's checks, and its
// resolved_by.commit must be a full object name (gitutil.IsFullSHA), so an
// abbreviation or anything option-shaped never reaches git, and git must
// resolve it to exactly itself as a commit, which holds it to this
// repository's hash and refuses a tag object's name.
func (s Store) storeFixCommit(issID string) (fix, rel, why string) {
	rel, fm, why := s.storeRecord(issID)
	switch {
	case why == storeRecordNone:
		return "", "", "nor is its private record resolved in the store"
	case why != "":
		return "", "", "and " + why
	}
	rb, _ := fm["resolved_by"].(map[string]any)
	ref, _ := rb["commit"].(string)
	switch {
	case ref == "":
		return "", "", "and its private record in the store names no fix commit (resolved_by.commit)"
	case !gitutil.IsFullSHA(ref):
		return "", "", "and the fix commit its private record in the store names is not a full commit id"
	}
	sha, err := gitutil.ResolveCommit(s.Checkout, ref)
	if err != nil || sha != ref {
		return "", "", "and the fix commit its private record in the store names is not a commit of this repository"
	}
	return sha, rel, ""
}

// storeRecordNone is storeRecord's why when the store's resolved/ folder
// holds no record of the issue by name.
const storeRecordNone = "its private record is not resolved in the store"

// storeRecord finds issID's private record in the store's resolved/ folder
// by its name and reads it through the checks every use of it shares: one
// record of that name (never two), read inside the store (os.Root) as a
// regular file, never a link, within its cap, parsed as a record, and
// carrying the issue's own id. rel is its path in the store,
// slash-separated, and fm its front matter; why ("" when it holds) says what
// failed, storeRecordNone when there is no record by that name.
func (s Store) storeRecord(issID string) (rel string, fm map[string]any, why string) {
	entries, err := os.ReadDir(filepath.Join(s.Root, filepath.FromSlash(resolvedRel)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, storeRecordNone
	}
	if err != nil {
		return "", nil, "the store's resolved/ folder could not be listed"
	}
	for _, e := range entries {
		if !isRecordOf(e.Name(), issID) {
			continue
		}
		if rel != "" {
			return "", nil, "the store holds its private record twice in resolved/"
		}
		rel = resolvedRel + "/" + e.Name()
	}
	if rel == "" {
		return "", nil, storeRecordNone
	}
	root, err := os.OpenRoot(s.Root)
	if err != nil {
		return "", nil, "the store could not be opened to read its private record"
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxRecordBytes)
	if err != nil {
		return "", nil, "its private record in the store could not be read as a regular file within its size limit"
	}
	fm, _, err = issuerecord.Parse(string(data))
	if err != nil {
		return "", nil, "its private record in the store is not a record this abcd reads"
	}
	if id, _ := fm["id"].(string); id != issID {
		return "", nil, "its private record in the store carries another id"
	}
	return rel, fm, ""
}

// plainPathRe is a path a shell reads as written, unquoted.
var plainPathRe = regexp.MustCompile(`\A[A-Za-z0-9._/+-]+\z`)

// restoreCommand is the command, run from the checkout's root, that copies
// the private record at rel in the store, already resolved there, into the
// public ledger's resolved/ folder: what a person runs once its advisory is
// published. why, when cmd is empty, says why none is named.
//
// The record is named under ~ whenever it sits under home: in home's own
// spelling first (the store is placed there, so a link inside abcd's home
// never matters), then through home's links. The rest follows ~/ as
// written when a shell reads it so, and single-quoted otherwise, so the
// tilde still expands and the command names no home directory, in --json
// or after a surface's home redaction. A store outside home is quoted
// whole. A path holding a character the terminal does not show as written
// (termsafe.Sanitize would change it) names no command: the one shown would
// name another file.
func restoreCommand(home string, s Store, rel string) (cmd, why string) {
	src := filepath.Join(s.Root, filepath.FromSlash(rel))
	shown := shellQuote(src)
	if r, ok := relUnder(home, src); ok {
		shown = "~/" + shellWord(r)
	} else if r, ok := relUnder(storepath.Resolve(home), storepath.Resolve(src)); ok {
		shown = "~/" + shellWord(r)
	}
	cmd = "cp -- " + shown + " " + resolvedRel + "/"
	if termsafe.Sanitize(cmd) != cmd {
		return "", "the path of its private record in the store holds a character the terminal does not show as written, so no command is named; " +
			"copy the record from the store's resolved/ folder into the public ledger's by hand"
	}
	return cmd, ""
}

// relUnder is p relative to dir, slash-separated, when p lies strictly
// under dir as both are spelled.
func relUnder(dir, p string) (string, bool) {
	if dir == "" {
		return "", false
	}
	r, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(p))
	if err != nil || r == "." || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) || filepath.IsAbs(r) {
		return "", false
	}
	return filepath.ToSlash(r), true
}

// shellWord is p as one shell word: as written when a shell reads it so,
// single-quoted otherwise.
func shellWord(p string) string {
	if plainPathRe.MatchString(p) {
		return p
	}
	return shellQuote(p)
}

// shellQuote is p single-quoted: each single quote in it ends the quoted
// run, is written escaped, and starts a new one.
func shellQuote(p string) string { return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'" }

// restoresOwed is each restore the store's state owes, in id order: every
// advisory it records published whose private record is resolved in the
// store and not yet in the checkout's public ledger (its resolved/ folder,
// by name), with the command that copies it there (restoreCommand). It is
// read from the state, never from the run that published the advisory, so
// a run that stopped, or a publication by hand on GitHub, loses nothing:
// every plan and every intake names it until the record is in the
// checkout. The store's record is held to storeRecord's checks first; one
// that fails them is listed with the reason, and no command is named.
func restoresOwed(home string, s Store, st State) []Restore {
	out := []Restore{}
	var inCheckout []os.DirEntry
	if entries, err := os.ReadDir(filepath.Join(s.Checkout, filepath.FromSlash(resolvedRel))); err == nil {
		inCheckout = entries
	}
	for _, a := range st.Advisories {
		if a.PublishedAt == nil || slices.ContainsFunc(inCheckout, func(e os.DirEntry) bool { return isRecordOf(e.Name(), a.IssueID) }) {
			continue
		}
		rel, _, why := s.storeRecord(a.IssueID)
		if why == storeRecordNone {
			continue
		}
		r := Restore{GHSAID: a.GHSAID, IssueID: a.IssueID}
		if why != "" {
			r.Reason = why + "; no command restoring it is named"
		} else {
			r.RestoreWith, r.Reason = restoreCommand(home, s, rel)
		}
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Restore) int { return strings.Compare(a.GHSAID, b.GHSAID) })
	return out
}

// recordForgeEnd stamps the end GitHub reports for a held advisory:
// published_at or closed_at, at the time given, with the forge state to
// match.
func recordForgeEnd(a *Advisory, state string, at time.Time) {
	switch state {
	case "published":
		a.PublishedAt, a.ForgeState = &at, "published"
	case "closed":
		a.ClosedAt, a.ForgeState = &at, "closed"
	}
}

// fixCommit reads the resolved_by.commit of the record at rec in the
// release's tree and resolves it to a commit of this repository. The blob
// is read with cat-file, which runs no text conversion.
func fixCommit(checkout, release, rec string) (string, string) {
	data, err := gitutil.RunCappedBytes(checkout, maxRecordBytes, "cat-file", "blob", "refs/tags/"+release+":"+rec)
	if err != nil {
		return "", "its record in " + release + " could not be read"
	}
	fm, _, err := issuerecord.Parse(string(data))
	if err != nil {
		return "", "its record in " + release + " is not a record this abcd reads"
	}
	rb, _ := fm["resolved_by"].(map[string]any)
	ref, _ := rb["commit"].(string)
	if ref == "" {
		return "", "its record in " + release + " names no fix commit (resolved_by.commit)"
	}
	if !commitRefRe.MatchString(ref) {
		return "", "its record's fix commit is not a commit id"
	}
	sha, err := gitutil.ResolveCommit(checkout, ref)
	if err != nil {
		return "", "its record's fix commit is not in this repository"
	}
	return sha, ""
}

// firstReleaseTag is the lowest release tag holding the fix: the version the
// fix was first released in, which may be older than the release being
// published. A fix in no release tag, or first released after it, is left out.
func firstReleaseTag(checkout, fix, release string) (string, string, error) {
	out, err := gitutil.RunCappedBytes(checkout, maxTagListing, "tag", "--contains", fix, "--list", "v*")
	if err != nil {
		return "", "", refuse("git could not list the tags holding a fix commit", "settle what git reports, then run it again")
	}
	tags := sortReleaseTags(strings.Split(string(out), "\n"))
	if len(tags) == 0 {
		return "", "its fix is in no release tag yet", nil
	}
	first := tags[0]
	fv, _ := launch.ParseSemver(strings.TrimPrefix(first, "v"))
	rv, _ := launch.ParseSemver(strings.TrimPrefix(release, "v"))
	if compareCore(fv, rv) > 0 {
		return "", "its fix is first released in " + first + ", after " + release, nil
	}
	return first, "", nil
}

// textDigest binds an advisory's text as read: SHA-256 over its summary and
// description.
func textDigest(a remoteAdvisory) string {
	h := sha256.New()
	h.Write([]byte(a.Summary))
	h.Write([]byte{0})
	h.Write([]byte(a.description()))
	return hex.EncodeToString(h.Sum(nil))
}

// planDigest binds an approval to exactly the plan shown: SHA-256 over the
// canonical JSON of the repository, the release tag and the included entries,
// each carrying its advisory's text digest. The left-out list is not bound:
// approving a plan approves what it publishes.
func planDigest(p PublicationPlan) string {
	// Strings, slices of strings and structs of them always marshal.
	data, _ := json.Marshal(struct {
		Repo    string      `json:"repo"`
		Release string      `json:"release"`
		Entries []PlanEntry `json:"entries"`
	}{p.Repo, p.Release, p.Entries})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
