// Package issueschema is the issue record's frontmatter schema as DATA: the
// property allow-list, the required set, and the closed enum value sets that
// issue.schema.json declares, held where every gate that asks the question can
// read them.
//
// It exists for the same reason core/changelog holds the impact enum: the writer
// and reader of the ledger (core/capture) and the lint that GATES the committed
// ledger (core/lint) must agree about what a well-formed record carries, and two
// hand-kept copies of a schema fact drift the moment one side gains a field. It
// is a leaf — it imports only the standard library, no filesystem, no transport —
// because core/capture's own tests import core/lint, so a lint that imported
// capture back would be an import cycle in test.
package issueschema

import (
	"regexp"
	"strings"
	"time"
)

// StatusDirs is the issue ledger's status-directory list, in the order a
// surface renders them. The directory a record sits in IS its status — there is
// no status field — so this list is simultaneously the set of folders the writer
// provisions, the set the readers scan, and the set the deterministic gates scope
// their pathspecs to. Those three had four spellings between them (lint's own
// list, two literals in capture's allocator, and the shell gate's pathspec), and
// the day a SIBLING family joined the ledger tree was the day the fourth spelling
// became a silent divergence: a record can only land somewhere every side agrees
// to look.
//
// It lives here for the same reason the property allow-list does — the one leaf
// both core/capture and core/lint already read the ledger's schema from. The
// shell gate cannot import Go and therefore holds the second and last spelling,
// pinned to this value by a test.
var StatusDirs = []string{"open", "resolved", "wontfix"}

// Required is every property the issue schema marks required, in the order a
// record writes them. A record missing one is not a lax record: the ledger reader
// refuses it and skips it, so it goes silently invisible to every capture surface
// while still sitting in the ledger — which is what record_schema reports.
var Required = []string{
	"schema_version", "id", "slug", "severity", "category", "source", "found_during",
}

// RequiredStrings is Required minus schema_version — the required properties
// whose value is a string. schema_version carries the integer version and is
// checked against the readers this repository ships, so it is validated on its
// own before the string-typed properties are walked. It is a slice OF Required
// rather than a second literal, so the two can never disagree about the set.
var RequiredStrings = Required[1:]

// Known is issue.schema.json's additionalProperties:false allow-list: every
// property a well-formed issue record may carry. capture's reader refuses a
// record with any key outside this set (skipping it, so it goes invisible to
// every capture surface), and record_schema reports the same key so the
// committed-ledger gate refuses what the reader refuses (iss-2608261447039180).
// It is the ONE copy both read.
var Known = map[string]bool{
	"schema_version": true, "id": true, "slug": true, "severity": true,
	"category": true, "source": true, "found_during": true, "found_at": true,
	// lapsed_at is the instant a recorded discipline gave way, RFC 3339 in UTC —
	// the lapse, not the write-up (spc-60). found_at cannot carry it (that
	// property is a LOCATION), and the timestamp-numeric record id is write-up
	// time by construction, which is the value the criterion distinguishes itself
	// from. Optional for every category and required for one; LapsedAtRequired
	// below is the single copy of which.
	"lapsed_at": true,
	"details":   true, "suggested_fix": true, "related_intents": true,
	"related_specs": true, "related_issues": true,
	"synthesis_clusters": true, "wontfix_reason": true, "resolution": true,
	"resolved_by": true, "blocked_by": true,
	// shipped_in names the release that already carried this record's work, so the
	// derivation can leave it out of a later cut (iss-2608241612087533). Optional
	// and rare — only a ledger-hygiene close, for a fix released long ago, has
	// anything to say here. It must be a KNOWN property or every write carrying
	// one is refused, which is exactly how the first draft of this feature shipped
	// a flag that could never execute.
	"shipped_in": true,
	// deferred_after and deferral_reason are the release cut's WAIVER pair: the
	// anchor tag a still-open finding was consciously deferred past, and the
	// stated reason. The unfixed-findings guardrail
	// (core/changelog.GuardFindings) refuses a cut that carries a consequential
	// finding captured since the anchor and still open, and this pair is the one
	// way past it — a deferral that is recorded and reviewable rather than
	// silent. Optional, and rare: only a finding a maintainer has decided to
	// hold over has anything to say here. Both must be KNOWN properties, or the
	// reader drops every waived record as malformed and the finding goes
	// invisible to every capture surface — the exact failure the guardrail
	// exists to prevent, reintroduced by its own escape hatch.
	"deferred_after": true, "deferral_reason": true,
	// origin and production_mode are the disclosure pair (itd-178): where an
	// item came from, and how its text was produced. Both are optional here —
	// population is forward-only, and an existing record carries neither — but
	// both must be KNOWN properties, or the reader drops every stamped record as
	// malformed and it goes invisible to every capture surface. The VALUES are
	// not judged here: the record_provenance lint rule is the gate on them, and a
	// mistyped disclosure key must not hide the finding the record carries. The
	// vocabulary itself lives in core/provenance, which reads this package for
	// the reading families' spelling — so the keys are literals here, pinned to
	// provenance's own constants by a test there.
	"origin": true, "production_mode": true,
	// impact is the product judgement the derived version and the generated
	// changelog are computed from (spc-10). It is optional here — an open issue
	// has not been judged yet, and the record-lint blocker issue_impact_valid is
	// what gates the move into resolved/ — but it must be a KNOWN property, or
	// the reader drops every judged record as malformed.
	"impact": true,
	// grounds/created/updated are no longer written, but a ledger written by an
	// older abcd still carries them. Tolerate (accept, then drop) them on read so
	// an existing committed ledger is not rejected as an unknown property; the
	// reader ignores their values entirely.
	//
	// grounds moved to the record BODY, as the append-only `## Grounds` section
	// core/grounds holds for both record families: a frontmatter scalar is SET,
	// and setting is what let a resolve destroy the conjecture the promote before
	// it recorded (iss-2608301657354776). Tolerating rather than refusing is
	// deliberate — refusing makes the reader SKIP the record, which hides it from
	// every capture surface while it still sits in the ledger. The gate that
	// notices a misplaced key is the record lint's `record_schema`,
	// which blocks a frontmatter `grounds:` and names the section, leaving the record readable meanwhile.
	"grounds": true, "created": true, "updated": true,
}

// Retired maps a frontmatter key a ledger record once carried to the key that
// replaced it. The promote join's forward stamp was the scalar `promoted_to`
// until itd-4 AC3 renamed it: the promoted intent is a member of the record's
// `related_intents`, and the intent names the record back in `related_issues`.
//
// A retired key is NOT in Known, so the reader refuses a record carrying it —
// no reader tolerates the old spelling (DECISIONS.md, 2026-09-23). This map
// exists so that refusal, the committed-ledger gate's and the drift check's all
// name the successor and the one verb that rewrites it, `abcd capture migrate`,
// instead of reporting an unexplained unknown property.
var Retired = map[string]string{"promoted_to": "related_intents"}

// MigrateHint is the remedy every refusal of a retired key names.
const MigrateHint = "run `abcd capture migrate --apply` to rewrite it"

// The closed enum value sets from issue.schema.json. capture validates a record's
// severity/category/source against these, and record_schema mirrors the same
// membership check for the committed-ledger gate (iss-2608270908342889). Held
// once here so the two gates cannot disagree about what a legal value is.
var (
	// Severities is the capture-time severity enum.
	Severities = []string{"nitpick", "minor", "major", "critical"}
	// Categories is the loose issue taxonomy.
	Categories = []string{
		"bug", "documentation", "drift", "inconsistency", "tech-debt", "security",
		"ux", "process", "architectural-insight", "future-work-seed", "observation",
		// lapse is the ledger's lapse log — a recording step skipped, disclosed
		// as a value in this list rather than a separate enum or store.
		"lapse",
	}
	// Sources is the surfacing channel enum.
	Sources = []string{
		"plan-review", "impl-review", "manual-test", "review-followup",
		"agent-finding", "agent-observation", "user-observation",
		"drift-detection", "memory-curation",
	}
)

// SlugRe is the kebab-case slug pattern (lower-case alphanumerics joined by
// single hyphens). A slug becomes a filename, so both the ledger writer and the
// record-lint gate hold it to exactly this shape.
var SlugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// CategoryLapse is the category whose records may state WHEN the discipline
// gave way, on lapsed_at. spc-60 obliged the value on this category; the
// obligation is parked (iss-2609091009111294) until the rethink of the reading work
// settles what a lapse record must carry, so lapsed_at is optional everywhere
// and format-checked wherever it is present. It is spelled once here because
// two gates read it — the ledger reader (core/capture) and the committed-ledger
// gate (core/lint) — and a second copy would let one of them go on refusing
// what the other accepts.
const CategoryLapse = "lapse"

// ValidLapsedAt reports whether a non-empty lapsed_at value is well formed: an
// RFC 3339 instant. A bare date names a day rather than a moment and free text
// names nothing a reader can order, so neither can carry the claim the property
// makes. The offset is not constrained to Z — RFC 3339 fixes the instant either
// way — while the convention the record pages state is UTC.
func ValidLapsedAt(v string) bool {
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	return err == nil
}
