package lifeboat

import (
	"fmt"
	"sort"
	"strings"
)

// gitSources returns the Tier-0 adapters: every brief section derivable from
// git history alone (present in any git repository). Each adapter reads ONLY
// commit history via the SourceContext git surface and never touches the
// working tree, so it grounds the same sections against an archived, README-less
// project as against a live one.
func gitSources() []Source {
	return []Source{
		gitGraveyardSource{},
		gitSpineSource{},
		gitContextSource{},
		gitDependenciesSource{},
		gitBuildSequenceSource{},
		gitWhatDidntSource{},
	}
}

// gitManifestFiles are the dependency manifests whose add/remove churn Tier-0
// can read from history. Kept sorted so the churn citation is deterministic.
var gitManifestFiles = []string{
	"Cargo.toml", "Gemfile", "Pipfile", "composer.json", "go.mod",
	"package.json", "pyproject.toml", "requirements.txt",
}

// gitReverts returns the subjects of reverting commits in HEAD's history, in
// git-log order (reverse chronological, deterministic). A `git revert` writes a
// `Revert "..."` subject; some tools write `Revert:` — both are recognised.
func gitReverts(ctx *SourceContext) []string {
	var out []string
	for _, s := range ctx.GitLines("log", "--format=%s") {
		if isRevertSubject(s) {
			out = append(out, s)
		}
	}
	return out
}

// gitDeletedPaths returns the unique, sorted repo-relative paths that history
// shows were deleted at some point (`git log --diff-filter=D`). A path deleted
// then re-added still counts as an abandonment signal.
func gitDeletedPaths(ctx *SourceContext) []string {
	return dedupeSorted(ctx.GitLines("log", "--diff-filter=D", "--name-only", "--format="))
}

// gitGraveyardSource grounds "graveyard" from git alone: what a project
// abandoned is written in its history whether or not anyone wrote it down —
// reverted commits and files deleted after substantial history. This is the
// flagship Tier-0 section.
type gitGraveyardSource struct{}

func (gitGraveyardSource) Section() Section { return "graveyard" }
func (gitGraveyardSource) Tier() Tier       { return TierGit }

func (gitGraveyardSource) Probe(ctx *SourceContext) Evidence {
	reverts := gitReverts(ctx)
	deleted := gitDeletedPaths(ctx)
	if len(reverts) == 0 && len(deleted) == 0 {
		return blank(
			[]string{
				"reverted commits (git log --grep Revert)",
				"files deleted after substantial history (git log --diff-filter=D)",
			},
			"What did this project try and abandon, and why is it not coming back?",
		)
	}
	var sources []string
	if len(reverts) > 0 {
		sources = append(sources, fmt.Sprintf("%d reverted commits", len(reverts)))
	}
	if len(deleted) > 0 {
		sources = append(sources, fmt.Sprintf("%d files deleted (e.g. %s)", len(deleted), deleted[0]))
	}
	// A revert is an explicit, deliberate abandonment signal — it grounds the
	// graveyard. A file deletion alone is ambiguous: nearly every repo deletes a
	// file in the normal course of work, so deletions without any revert are only
	// partial evidence, not a grounded graveyard. Keeping this honest matters —
	// the cross-repo aggregate reads these statuses as the experiment's result.
	if len(reverts) > 0 {
		return Evidence{Status: StatusGrounded, Confidence: ConfidenceHigh, Sources: sources}
	}
	return partial(ConfidenceMedium, sources,
		fmt.Sprintf("Found %d deleted file(s) but no revert, and a deletion alone is ordinary work rather than a deliberate abandonment; a reverted commit would ground it.", len(deleted)))
}

// gitSpineSource grounds "rescue/spine" partially: the commit history is a
// project spine where no record exists — its length and contributor count.
type gitSpineSource struct{}

func (gitSpineSource) Section() Section { return "rescue/spine" }
func (gitSpineSource) Tier() Tier       { return TierGit }

func (gitSpineSource) Probe(ctx *SourceContext) Evidence {
	n := ctx.CommitCount()
	if n == 0 {
		return blank(
			[]string{"commit history (git log)"},
			"Is there any development history to reconstruct a spine from?",
		)
	}
	authors := map[string]bool{}
	for _, a := range ctx.GitLines("log", "--format=%an") {
		authors[a] = true
	}
	sources := []string{fmt.Sprintf("git log (%d commits)", n)}
	if len(authors) > 0 {
		sources = append(sources, fmt.Sprintf("%d contributor(s)", len(authors)))
	}
	return partial(ConfidenceMedium, sources,
		fmt.Sprintf("Found %d commit(s) of history, which give the project's sequence but not the intent behind each step; an intent corpus (.abcd/development/intents/) would ground it.", n))
}

// gitContextSource grounds "product/context" partially: what the project does,
// inferred from non-merge commit subjects. Blank when history carries only
// empty or merge commits — nothing descriptive to infer from.
type gitContextSource struct{}

func (gitContextSource) Section() Section { return "product/context" }
func (gitContextSource) Tier() Tier       { return TierGit }

func (gitContextSource) Probe(ctx *SourceContext) Evidence {
	var subjects []string
	for _, s := range ctx.GitLines("log", "--format=%s") {
		if strings.HasPrefix(s, "Merge ") {
			continue
		}
		subjects = append(subjects, s)
	}
	if len(subjects) == 0 {
		return blank(
			[]string{"commit subjects (git log --format=%s)"},
			"What does this project do? No descriptive commit subjects to infer it from.",
		)
	}
	return partial(ConfidenceLow, []string{fmt.Sprintf("commit subjects (%d non-merge commits)", len(subjects))},
		fmt.Sprintf("Found %d non-merge commit subject(s), from which what the project does can only be inferred; a README that describes it, or an authored %s, would ground it.", len(subjects), nativeSectionBriefFile("product/context")))
}

// gitDependenciesSource grounds "constraints/dependencies" partially: the
// add/remove churn of dependency manifests across history. The authoritative
// list needs the current manifest and lockfile (a richer tier), so this is
// partial. Blank when no manifest ever appears in history.
type gitDependenciesSource struct{}

func (gitDependenciesSource) Section() Section { return "constraints/dependencies" }
func (gitDependenciesSource) Tier() Tier       { return TierGit }

func (gitDependenciesSource) Probe(ctx *SourceContext) Evidence {
	var churn []string
	for _, m := range gitManifestFiles {
		commits := ctx.GitLines("log", "--format=%h", "--", m)
		if len(commits) > 0 {
			churn = append(churn, fmt.Sprintf("%s churned in %d commit(s)", m, len(commits)))
		}
	}
	if len(churn) == 0 {
		return blank(
			[]string{"dependency manifests (" + strings.Join(gitManifestFiles, ", ") + ")"},
			"What does this project depend on? No manifest ever appears in history.",
		)
	}
	sort.Strings(churn)
	return partial(ConfidenceMedium, churn,
		"Found dependency manifests churning in history, which say dependencies changed but not the current set; the current manifest with its lockfile would ground it.")
}

// gitBuildSequenceSource grounds "delivery/build-sequence" partially: release
// tags and commit cadence. Blank when there are no tags and history is trivial.
type gitBuildSequenceSource struct{}

func (gitBuildSequenceSource) Section() Section { return "delivery/build-sequence" }
func (gitBuildSequenceSource) Tier() Tier       { return TierGit }

func (gitBuildSequenceSource) Probe(ctx *SourceContext) Evidence {
	tags := dedupeSorted(ctx.GitLines("tag"))
	if len(tags) > 0 {
		return partial(ConfidenceMedium, []string{fmt.Sprintf("%d tags (e.g. %s)", len(tags), tags[0])},
			fmt.Sprintf("Found %d release tag(s), which mark when releases happened but not what each delivered or why in that order; an authored %s would ground it.", len(tags), nativeSectionBriefFile("delivery/build-sequence")))
	}
	if n := ctx.CommitCount(); n >= 3 {
		return partial(ConfidenceLow, []string{fmt.Sprintf("commit cadence (%d commits, no tags)", n)},
			fmt.Sprintf("Found %d commits and no release tag, so only the cadence of work is known (release tags would add when releases happened, still partial); an authored %s would ground it.", n, nativeSectionBriefFile("delivery/build-sequence")))
	}
	return blank(
		[]string{"release tags (git tag)", "commit cadence"},
		"How was this project delivered? No tags and only trivial history.",
	)
}

// gitWhatDidntSource grounds "evidence/what-didnt" partially: reverts framed as
// dead ends — what was tried and reversed. Blank when history has no reverts.
type gitWhatDidntSource struct{}

func (gitWhatDidntSource) Section() Section { return "evidence/what-didnt" }
func (gitWhatDidntSource) Tier() Tier       { return TierGit }

func (gitWhatDidntSource) Probe(ctx *SourceContext) Evidence {
	reverts := gitReverts(ctx)
	if len(reverts) == 0 {
		return blank(
			[]string{"reverted commits (git log --grep Revert)"},
			"What was tried and reversed? No reverts appear in history.",
		)
	}
	return partial(ConfidenceMedium, []string{fmt.Sprintf("%d reverted commits (e.g. %s)", len(reverts), reverts[0])},
		fmt.Sprintf("Found %d reverted commit(s), which show what was reversed but not why it failed; an authored %s would ground it.", len(reverts), nativeSectionBriefFile("evidence/what-didnt")))
}
