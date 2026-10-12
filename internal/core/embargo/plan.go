package embargo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gitutil"
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
}

// LeftOut is an advisory judged a vulnerability that the plan does not
// publish, and why.
type LeftOut struct {
	GHSAID  string `json:"ghsa_id"`
	IssueID string `json:"issue_id"`
	Reason  string `json:"reason"`
}

// PublicationPlan is what publication would do. Digest is the approval it
// takes, and ApproveWith the command that gives it (empty when nothing is
// planned).
type PublicationPlan struct {
	Repo        string      `json:"repo"`
	Release     string      `json:"release"`
	Entries     []PlanEntry `json:"entries"`
	LeftOut     []LeftOut   `json:"left_out"`
	Digest      string      `json:"digest"`
	ApproveWith string      `json:"approve_with"`
}

// PlanPublication builds the publication plan and changes nothing: the only
// requests it sends are reads of the advisories it plans.
func PlanPublication(req PublishRequest) (PublicationPlan, error) {
	s, st, err := OpenExisting(req.Home, req.RepoRoot)
	if err != nil {
		return PublicationPlan{}, err
	}
	return planFor(s, st, req.Release)
}

// planFor builds the plan of release over the store's state. For each
// advisory judged a vulnerability and neither published nor closed, in id
// order: its record must be resolved in the release's tree and name its fix
// commit; the first release tag holding that commit gives the patched
// version; the advisory, read from GitHub, must still be unpublished, carry
// its package, an affected range in GitHub's form, and text the outbound scan
// passes. Each that fails a step is left out with the reason. Git is read
// before GitHub, so an advisory the release leaves out is never requested.
func planFor(s Store, st State, release string) (PublicationPlan, error) {
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
	p := PublicationPlan{Repo: st.Repo, Release: release, Entries: []PlanEntry{}, LeftOut: []LeftOut{}}
	var candidates []Advisory
	for _, a := range st.Advisories {
		if a.Verdict != nil && *a.Verdict == VerdictVulnerability && a.PublishedAt == nil && a.ClosedAt == nil {
			candidates = append(candidates, a)
		}
	}
	slices.SortFunc(candidates, func(a, b Advisory) int { return strings.Compare(a.GHSAID, b.GHSAID) })
	for _, a := range candidates {
		e, why, err := planEntry(s, st.Repo, release, resolved, a)
		if err != nil {
			return PublicationPlan{}, err
		}
		if why != "" {
			p.LeftOut = append(p.LeftOut, LeftOut{GHSAID: a.GHSAID, IssueID: a.IssueID, Reason: why})
			continue
		}
		p.Entries = append(p.Entries, e)
	}
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

// planEntry plans one advisory, or says why it is left out. An error is a
// failure that refuses the whole plan (a git or GitHub request that fails, a
// text the scanner cannot judge).
func planEntry(s Store, repo, release string, resolved []string, a Advisory) (PlanEntry, string, error) {
	rec := ""
	for _, p := range resolved {
		base := path.Base(p)
		if base == a.IssueID+".md" || (strings.HasPrefix(base, a.IssueID+"-") && strings.HasSuffix(base, ".md")) {
			rec = p
			break
		}
	}
	if rec == "" {
		return PlanEntry{}, "its record is not resolved in " + release, nil
	}
	fix, why := fixCommit(s.Checkout, release, rec)
	if why != "" {
		return PlanEntry{}, why, nil
	}
	first, why, err := firstReleaseTag(s.Checkout, fix, release)
	if err != nil || why != "" {
		return PlanEntry{}, why, err
	}
	patched := patchedVersion(first)

	remote, err := getAdvisory(s.Checkout, repo, a.GHSAID)
	if err != nil {
		return PlanEntry{}, "", err
	}
	switch remote.State {
	case "triage", "draft":
	case "published", "closed":
		return PlanEntry{}, "GitHub reports it " + remote.State + " already", nil
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
	}
	return PlanEntry{GHSAID: a.GHSAID, IssueID: a.IssueID, FixCommit: fix, FirstTag: first, PatchedVersion: patched,
		ForgeState: remote.State, Vulnerabilities: vulns, TextDigest: textDigest(remote)}, "", nil
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
