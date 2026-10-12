package embargo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// PublishedAdvisory is one advisory GitHub confirmed published, and when
// abcd recorded it.
type PublishedAdvisory struct {
	GHSAID      string    `json:"ghsa_id"`
	IssueID     string    `json:"issue_id"`
	PublishedAt time.Time `json:"published_at"`
}

// PublishResult is what a publish showed and did: the plan, and with an
// approval, each advisory published in order and whether GitHub then lists
// none of them in draft or triage.
type PublishResult struct {
	Plan        PublicationPlan     `json:"plan"`
	Approved    bool                `json:"approved"`
	Published   []PublishedAdvisory `json:"published"`
	NoDraftLeft bool                `json:"no_draft_left"`
}

// PublishFailedError is a publication that stopped: at advisory GHSAID (the
// canary when Canary), or, with GHSAID empty, at the final check that none
// is left in draft. Published and Unpublished say where each planned
// advisory stands; nothing was sent after the failure.
type PublishFailedError struct {
	GHSAID      string   `json:"ghsa_id,omitempty"`
	Canary      bool     `json:"canary"`
	Reason      string   `json:"reason"`
	Published   []string `json:"published"`
	Unpublished []string `json:"unpublished"`
}

func (e *PublishFailedError) Error() string {
	var b strings.Builder
	b.WriteString("embargo: publication stopped")
	switch {
	case e.GHSAID != "" && e.Canary:
		b.WriteString(" at the canary, " + e.GHSAID)
	case e.GHSAID != "":
		b.WriteString(" at " + e.GHSAID)
	default:
		b.WriteString(" at the final check")
	}
	b.WriteString(": " + e.Reason + "; nothing further was sent")
	if len(e.Published) == 0 {
		b.WriteString("; nothing was published")
	} else {
		b.WriteString("; published: " + strings.Join(e.Published, ", "))
	}
	if len(e.Unpublished) > 0 {
		b.WriteString("; not published: " + strings.Join(e.Unpublished, ", "))
	}
	b.WriteString(" (look at the advisories on GitHub; running `abcd embargo publish` again plans only what is still unpublished)")
	return b.String()
}

// Publish shows the publication plan and, with Approve equal to the digest
// of the plan as it stands now, publishes it. Without approval the only
// requests are reads. With it, the plan is rebuilt and a digest that does
// not match refuses with nothing sent. Then, in id order, each advisory is
// published the same way, the first as the canary: one update sets its
// versions (accepting it to draft when it is still in triage), a read must
// show them, a second update publishes it, and a read must show it
// published with them; published_at is stamped at once. Any failure stops
// the run before the next update and says which advisories are published.
// Last, the draft and triage listings must name none of the planned ones.
func Publish(req PublishRequest) (PublishResult, error) {
	s, st, err := OpenExisting(req.Home, req.RepoRoot)
	if err != nil {
		return PublishResult{}, err
	}
	plan, err := planFor(s, st, req.Release)
	if err != nil {
		return PublishResult{}, err
	}
	res := PublishResult{Plan: plan, Published: []PublishedAdvisory{}}
	if req.Approve == "" {
		return res, nil
	}
	if req.Approve != plan.Digest {
		return PublishResult{}, refuse("the approval is not for the plan that would be published now (a record, a tag or an advisory on GitHub changed since, or it was given for another plan); nothing was sent",
			"run it again without --approve to see the current plan and its digest")
	}
	if len(plan.Entries) == 0 {
		return PublishResult{}, refuse("the plan publishes nothing; nothing was sent", "resolve and release the fixes first, then run it again without --approve")
	}
	res.Approved = true

	planned := make([]string, len(plan.Entries))
	for i, e := range plan.Entries {
		planned[i] = e.GHSAID
	}
	fail := func(i int, ghsa, why string) error {
		var done []string
		for _, p := range res.Published {
			done = append(done, p.GHSAID)
		}
		return &PublishFailedError{GHSAID: ghsa, Canary: i == 0 && ghsa != "", Reason: why, Published: done, Unpublished: planned[len(done):]}
	}

	for i, e := range plan.Entries {
		if why := publishOne(s, st.Repo, e); why != "" {
			return res, fail(i, e.GHSAID, why)
		}
		now := clock(req.Now)
		j, err := findAdvisory(st, e.GHSAID)
		if err == nil {
			st.Advisories[j].PublishedAt, st.Advisories[j].ForgeState = &now, "published"
			err = s.SaveState(st)
		}
		res.Published = append(res.Published, PublishedAdvisory{GHSAID: e.GHSAID, IssueID: e.IssueID, PublishedAt: now})
		if err != nil {
			return res, fail(i, e.GHSAID, "it is published on GitHub, but its published_at could not be recorded in the store ("+err.Error()+")")
		}
	}

	var left []string
	for _, state := range []string{"draft", "triage"} {
		items, err := listAdvisories(s.Checkout, st.Repo, state)
		if err != nil {
			return res, fail(len(planned), "", "the "+state+" listing could not be read to confirm none is left ("+err.Error()+")")
		}
		for _, a := range items {
			if slices.Contains(planned, a.GHSAID) && !slices.Contains(left, a.GHSAID) {
				left = append(left, a.GHSAID)
			}
		}
	}
	if len(left) > 0 {
		slices.Sort(left)
		return res, fail(len(planned), "", "GitHub still lists "+strings.Join(left, ", ")+" in draft or triage after reading it back published")
	}
	res.NoDraftLeft = true
	return res, nil
}

// publishOne publishes one planned advisory, reading it back after each of
// its two updates, and says why it failed ("" on success).
func publishOne(s Store, repo string, e PlanEntry) string {
	body := map[string]any{"vulnerabilities": e.Vulnerabilities}
	if e.ForgeState == "triage" {
		body["state"] = "draft"
	}
	data, err := plainJSON(body)
	if err != nil {
		return "its update could not be composed"
	}
	if err := patchAdvisory(s.Checkout, repo, e.GHSAID, data); err != nil {
		return "setting its versions failed (" + err.Error() + ")"
	}
	back, err := getAdvisory(s.Checkout, repo, e.GHSAID)
	if err != nil {
		return "it could not be read back after setting its versions (" + err.Error() + ")"
	}
	if back.State != "draft" {
		return fmt.Sprintf("after setting its versions GitHub reports it %s, not draft", forgeStateWord(back.State))
	}
	if why := versionsHeld(back, e.Vulnerabilities); why != "" {
		return "after setting its versions, " + why
	}
	if err := patchAdvisory(s.Checkout, repo, e.GHSAID, []byte(`{"state":"published"}`)); err != nil {
		return "publishing it failed (" + err.Error() + ")"
	}
	back, err = getAdvisory(s.Checkout, repo, e.GHSAID)
	if err != nil {
		return "it could not be read back after publishing (" + err.Error() + ")"
	}
	if back.State != "published" {
		return fmt.Sprintf("after publishing GitHub reports it %s, not published", forgeStateWord(back.State))
	}
	if why := versionsHeld(back, e.Vulnerabilities); why != "" {
		return "after publishing, " + why
	}
	return ""
}

// versionsHeld compares GitHub's copy of an advisory's affected packages
// with the planned ones, entry by entry, and says what differs.
func versionsHeld(a remoteAdvisory, want []PlannedVulnerability) string {
	if len(a.Vulnerabilities) != len(want) {
		return fmt.Sprintf("GitHub's copy carries %d affected packages where %d were planned", len(a.Vulnerabilities), len(want))
	}
	for i, v := range a.Vulnerabilities {
		w := want[i]
		switch {
		case v.Package == nil || *v.Package != w.Package:
			return fmt.Sprintf("GitHub's copy names another package in entry %d", i+1)
		case v.VulnerableVersionRange == nil || *v.VulnerableVersionRange != w.VulnerableVersionRange:
			return fmt.Sprintf("GitHub's copy does not carry the planned affected range in entry %d", i+1)
		case v.PatchedVersions == nil || *v.PatchedVersions != w.PatchedVersions:
			return fmt.Sprintf("GitHub's copy does not carry the planned patched version in entry %d", i+1)
		}
	}
	return ""
}

// plainJSON is v as compact JSON without HTML escaping, so a range such as
// "< 0.9.0" is sent as written rather than as "\u003c 0.9.0".
func plainJSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
