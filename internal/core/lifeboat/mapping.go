// Package lifeboat holds the brief-to-lifeboat contract and, later, the source
// adapters that fill it.
//
// The mapping table below is the single source of truth for that contract. It
// is rendered into the brief's 00-meta.md, which calls the table "the contract"
// but has never carried one; a test asserts the two agree, so the document
// cannot drift from the code.
//
// The table is a HYPOTHESIS, not a measurement. Each row states the best status
// a lifeboat could reach for one brief section at each source tier. The probe
// (`abcd disembark probe`) measures the same sections against real repositories
// and reports the same three-valued status, so the hypothesis and the evidence
// are directly comparable — and the hypothesis is expected to lose where they
// disagree.
package lifeboat

import (
	"fmt"
	"strings"
)

// Tier names a class of source material, ordered by how much a repository has
// to have done deliberately for the tier to exist at all. Adapters degrade: a
// section is filled by the richest tier present, falling back to poorer ones.
//
// Tiers are CUMULATIVE. A repository that has conventions still has git, so the
// status quoted at a tier is what is achievable using that tier and every
// poorer one together. This is why a richer tier can never ground a section
// worse than a poorer one, and why a test enforces exactly that.
type Tier string

const (
	// TierGit reads commit history, authors, branches, reverts, file lifespans,
	// tags, and dependency churn. Present in every git repository.
	TierGit Tier = "git"
	// TierConventions reads README, docs/, CHANGELOG, LICENSE, CONTRIBUTING,
	// issue exports, and ADRs wherever they happen to live. Present in most.
	TierConventions Tier = "conventions"
	// TierNative reads .abcd/ — decisions, intents, specs, brief, roadmap,
	// issues, reviews, memory. Present only in a repository abcd manages.
	TierNative Tier = "abcd-native"
)

// Tiers lists every tier from poorest to richest.
func Tiers() []Tier { return []Tier{TierGit, TierConventions, TierNative} }

// Status is the three-valued coverage result for one brief section. A blank is
// a first-class result — it names a question a human must answer — not a
// failure of the probe.
type Status string

const (
	// StatusBlank means nothing in the repository grounds the section.
	StatusBlank Status = "blank"
	// StatusPartial means the section can be started but not completed from
	// this tier; some claims would have no source to cite.
	StatusPartial Status = "partial"
	// StatusGrounded means every claim in the section can cite a source file.
	StatusGrounded Status = "grounded"
)

// rank orders statuses so that a richer tier can be checked never to yield a
// worse result than a poorer one.
func (s Status) rank() int {
	switch s {
	case StatusBlank:
		return 0
	case StatusPartial:
		return 1
	case StatusGrounded:
		return 2
	}
	return -1
}

// Valid reports whether s is a member of the status enum.
func (s Status) Valid() bool { return s.rank() >= 0 }

// Section is a brief section, named as it appears in the coverage report.
type Section string

// Kind classifies a brief section by whether a repository could ever ground it.
// It is the durable form of the M2 gate decision (adr-36): a blank in an
// extractable section is coverage debt abcd can close with a better source or
// adapter; a blank in a human-owned section is not a failure at all — it is a
// standing prompt only a person can answer, and the lifeboat frames it that way.
type Kind string

const (
	// KindExtractable means a source or a richer adapter could ground the
	// section; an unfilled blank is coverage debt.
	KindExtractable Kind = "extractable"
	// KindHumanOwned means the section is not derivable from a repository by
	// design; the blank is a question for a human, never an extraction.
	KindHumanOwned Kind = "human-owned"
)

// humanOwnedSections are the sections the M2 cross-repo gate ruled not derivable
// from a repository — answered by a person, not extracted. Every other section
// is extractable (the zero value of the Kind lookup). Keeping the set here, off
// the positional Table literal, avoids editing 24 rows for one classification.
var humanOwnedSections = map[Section]bool{
	"product/personas":             true,
	"product/framing":              true,
	"product/mental-model":         true,
	"delivery/verification-matrix": true,
	"delivery/out-of-scope":        true,
}

// Kind reports whether the section is extractable or human-owned.
func (s Section) Kind() Kind {
	if humanOwnedSections[s] {
		return KindHumanOwned
	}
	return KindExtractable
}

// Mapping is one row of the contract: where a brief section lands in a
// lifeboat, the best status each tier could ground it to, and what a source
// adapter reads to try.
type Mapping struct {
	Section      Section
	LifeboatPath string
	Git          Status
	Conventions  Status
	Native       Status
	Reads        string
}

// StatusAt returns the hypothesised best status for this section at tier t.
func (m Mapping) StatusAt(t Tier) Status {
	switch t {
	case TierGit:
		return m.Git
	case TierConventions:
		return m.Conventions
	case TierNative:
		return m.Native
	}
	return ""
}

// Table is the brief-to-lifeboat contract.
//
// Read the three status columns as the experiment's prediction. Two rows carry
// most of the thesis:
//
//   - graveyard is the only section grounded at TierGit. What a project
//     abandoned is written in its git history whether or not anyone wrote it
//     down, which is why the graveyard is worth a section of its own.
//   - product/personas is blank at every tier below abcd-native, and only
//     partial there. If the probe confirms that across the corpus, the section
//     is not derivable from a repository at all and should not be in a
//     lifeboat's brief — it is a question for a human, not an extraction.
var Table = []Mapping{
	{"product/press-release", "brief/01-product/01-press-release.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"README lede, docs/, shipped intents' press releases"},
	{"product/context", "brief/01-product/02-context.md",
		StatusPartial, StatusGrounded, StatusGrounded,
		"README, docs/, CONTRIBUTING, commit subjects"},
	{"product/mental-model", "brief/01-product/03-mental-model.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"docs/ explanation pages, ADR context sections, the brief"},
	{"product/scope", "brief/01-product/04-scope.md",
		StatusPartial, StatusPartial, StatusGrounded,
		"README features, intents' in-scope/out-of-scope sections, the code's own surface"},
	{"product/personas", "brief/01-product/05-personas.md",
		StatusBlank, StatusBlank, StatusPartial,
		"personas registry, press-release quote attributions"},
	{"product/framing", "brief/01-product/06-framing.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"the brief's framing section, the brief-creation interview's committed framing products"},
	{"constraints/platform", "brief/02-constraints/01-platform.md",
		StatusPartial, StatusGrounded, StatusGrounded,
		"build manifests, CI workflows, README requirements"},
	{"constraints/dependencies", "brief/02-constraints/02-dependencies.md",
		StatusPartial, StatusGrounded, StatusGrounded,
		"add/remove churn from git history; the authoritative list needs the manifest and lockfile"},
	{"constraints/invariants", "brief/02-constraints/03-invariants.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"CONTRIBUTING, agent-conventions router, lint configs, ADR consequences"},
	{"constraints/naming", "brief/02-constraints/04-naming.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"glossary, naming registry, reserved-vocabulary tables"},
	{"evidence/what-worked", "brief/03-evidence/01-what-worked.md",
		StatusPartial, StatusPartial, StatusGrounded,
		"CHANGELOG, code that survived, reviews and retrospectives"},
	{"evidence/what-didnt", "brief/03-evidence/02-what-didnt.md",
		StatusPartial, StatusPartial, StatusGrounded,
		"the graveyard beneath it — reverts, dead branches, superseded records"},
	{"evidence/open-questions", "brief/03-evidence/03-open-questions.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"TODO and FIXME markers, open issues, intents' open-questions sections"},
	{"evidence/tradeoffs", "brief/03-evidence/04-tradeoffs.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"ADR alternatives-considered sections wherever ADRs live"},
	{"surfaces", "brief/04-surfaces/",
		StatusPartial, StatusGrounded, StatusGrounded,
		"CLI entrypoints and help text, README usage, per-surface design files"},
	{"internals", "brief/05-internals/",
		StatusPartial, StatusPartial, StatusGrounded,
		"package layout, architecture docs, the internals chapters"},
	{"delivery/build-sequence", "brief/06-delivery/01-build-sequence.md",
		StatusPartial, StatusPartial, StatusGrounded,
		"tags and release history, CHANGELOG, roadmap phases"},
	{"delivery/verification-matrix", "brief/06-delivery/02-verification-matrix.md",
		StatusPartial, StatusPartial, StatusGrounded,
		"test files and what they target, CI workflow steps; which check covers which surface is authored judgement"},
	{"delivery/out-of-scope", "brief/06-delivery/03-out-of-scope.md",
		StatusBlank, StatusPartial, StatusGrounded,
		"README non-goals, intents' out-of-scope sections"},
	{"glossary", "brief/glossary/",
		StatusBlank, StatusPartial, StatusGrounded,
		"docs glossary pages, the bounded-context glossary"},
	{"graveyard", "graveyard/",
		StatusGrounded, StatusGrounded, StatusGrounded,
		"reverted commits, branches abandoned unmerged, files deleted after substantial history, dependencies added then removed; then superseded records"},
	{"rescue/spine", "rescue/",
		StatusPartial, StatusPartial, StatusGrounded,
		"commit history as a spine where no record exists; the intent corpus where one does"},
	{"docs/adrs", "docs/adrs/",
		StatusBlank, StatusGrounded, StatusGrounded,
		"ADRs wherever they live, copied verbatim"},
	{"activity/issues", "activity/issues/",
		StatusBlank, StatusPartial, StatusGrounded,
		"issue exports, the capture ledger"},
}

// MarkerBegin and MarkerEnd delimit the rendered table inside the brief. The
// test that guards this contract reads between them.
const (
	MarkerBegin = "<!-- BEGIN GENERATED: brief-lifeboat-mapping -->"
	MarkerEnd   = "<!-- END GENERATED: brief-lifeboat-mapping -->"
)

// Render returns the mapping table as a Markdown table, exactly as it appears
// between the markers in the brief's 00-meta.md.
func Render() string {
	var b strings.Builder
	b.WriteString("| Brief section | Lifeboat path | Tier 0 git | Tier 1 conventions | Tier 2 abcd-native | Reads |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, m := range Table {
		fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s | %s | %s |\n",
			m.Section, m.LifeboatPath, m.Git, m.Conventions, m.Native, m.Reads)
	}
	return b.String()
}
