package launch

import (
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// RetentionPlan is the newest-per-line prune preview for a release cut.
type RetentionPlan struct {
	Published     string   `json:"published"`
	Line          string   `json:"line"`
	Kept          []string `json:"kept"`
	Pruned        []string `json:"pruned"`
	Refused       bool     `json:"refused"`
	RefusalReason string   `json:"refusal_reason,omitempty"`
}

// ComputeRetention decides newest-per-line retention (brief §3). It is pure and
// deletes nothing — it renders the prune decision only.
//
// Rules:
//   - Each MAJOR.MINOR line keeps only its newest (max-Patch) release.
//   - The just-published release is NEVER pruned.
//   - If ANY existing release is strictly NEWER than published, refuse and prune
//     nothing (an out-of-order ship the operator must resolve manually).
//   - Comparison is core (Major, Minor, Patch) only. Tag string = "v"+version.
func ComputeRetention(published Semver, existing []Semver) RetentionPlan {
	plan := RetentionPlan{Published: published.Tag(), Line: published.Line()}

	// (1) Refuse on any strictly-newer existing release.
	for _, e := range existing {
		if CoreGreater(e, published) {
			plan.Refused = true
			plan.RefusalReason = "existing release " + e.Tag() +
				" is newer than the published " + published.Tag() +
				" (out-of-order ship; resolve manually)"
			return plan
		}
	}

	// (2) Group existing ∪ {published} by line; published filtered from existing
	// defensively so it is never double-counted or pruned.
	byLine := map[string][]Semver{}
	add := func(v Semver) { byLine[v.Line()] = append(byLine[v.Line()], v) }
	add(published)
	for _, e := range existing {
		if e.String() == published.String() {
			continue
		}
		add(e)
	}

	keptSet := map[string]struct{}{}
	var kept, pruned []string
	for _, versions := range byLine {
		// (3) Keep the max-Patch version in the line; the rest are pruned.
		best := versions[0]
		for _, v := range versions[1:] {
			if CoreGreater(v, best) {
				best = v
			}
		}
		for _, v := range versions {
			if v.String() == best.String() {
				if _, seen := keptSet[v.Tag()]; !seen {
					keptSet[v.Tag()] = struct{}{}
					kept = append(kept, v.Tag())
				}
			} else {
				pruned = append(pruned, v.Tag())
			}
		}
	}

	// (4) Guarantee published ∈ Kept and never in Pruned.
	if _, ok := keptSet[published.Tag()]; !ok {
		kept = append(kept, published.Tag())
	}
	pruned = removeTag(pruned, published.Tag())

	sort.Strings(kept)
	sort.Strings(pruned)
	plan.Kept = kept
	plan.Pruned = pruned
	return plan
}

func removeTag(tags []string, tag string) []string {
	out := tags[:0]
	for _, t := range tags {
		if t != tag {
			out = append(out, t)
		}
	}
	return out
}

// GitExistingTags is the default provider for the existing-release list: it runs
// `git tag --list v*` under repoRoot and parses the strict-SemVer core of each
// tag. Non-SemVer tags AND prerelease/build tags are ignored — retention decides
// newest-per-line over RELEASES (core versions), and Tag()/String() render the
// core only, so admitting a "v1.2.3-rc1" would surface a phantom "v1.2.3" in the
// plan and collapse it against the real "v1.2.3". It is best-effort — an error
// yields no tags.
func GitExistingTags(repoRoot string) ([]Semver, error) {
	// Isolated through gitutil.Run: an inherited GIT_DIR/GIT_WORK_TREE would
	// override the root and list a different repository's tags, skewing the
	// existing-release set that version-collision and retention decisions
	// depend on; and a repository's tag.sort can make the listing read the
	// tagged objects, so in a partial clone on a git below the lazy-fetch
	// floor the listing is refused rather than fetched (iss-2610090821527948).
	out, err := gitutil.Run(repoRoot, "tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	var vers []Semver
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "v") {
			continue
		}
		if v, err := ParseSemver(strings.TrimPrefix(line, "v")); err == nil {
			if v.Prerelease != "" || v.Build != "" {
				continue // not a release; its core tag would be a phantom in the plan
			}
			vers = append(vers, v)
		}
	}
	return vers, nil
}
