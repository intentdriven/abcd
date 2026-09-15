// Package recordid is abcd's canonical home for the record-id space: the
// native timestamp-numeric mint (mint.go), the read-side resolver (resolve.go),
// and the one record-filename grammar both sides parse (this file).
//
// THE MINT (adr-45; mechanics per spc-33): every minting family allocates
// <family>-<yymmddHHMMSS><4 random digits> through Minter.Mint — time-ordered,
// coordination-free, offline, and reading no maximum anywhere, so two minters
// sharing a stale view, or two current checkouts sharing the same view, can
// never converge on one id. Captures (iss), intents (itd), specs (spc), the
// reading families and the scope-condition markers all mint this way; a family
// adopts the seam by holding a Minter and naming its family tag, never by
// carrying an allocator of its own (adr-45 ruling 3). Decisions (adr) mint here
// too, per the 2026-09-01 ruling adr-45 ruling 3 deferred: an ADR filed from
// then on is `adr-<stamp>` in a `<stamp>-<slug>.md` file, while the
// hand-numbered ordinals `0001`–`0058` keep their ids and their filenames. Both
// vintages are the same `[0-9]+` grammar, and CanonADRID / ADRFileID (resolve.go)
// are the one derivation every reader of either takes.
//
// The armed record-lint uniqueness rules (issue_id_unique, intent_lifecycle,
// spec_id_unique) stay as the scheme's fail-safe — the cheap assertion that the
// scheme held — never the primary defence (adr-45 ruling 5).
package recordid

import "regexp"

// idRe matches a record filename <prefix>-<N>[-<slug>].md and captures N. The
// prefix is a fixed family tag (iss/itd/spc); it is regexp-quoted defensively so
// a caller can never inject metacharacters through it.
func idRe(prefix string) *regexp.Regexp {
	return regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-([0-9]+)(?:-[a-z0-9-]+)?\.md$`)
}

// FilenameNumRe is the canonical record-filename grammar for a prose-handle
// family (iss/itd/spc): <prefix>-<N>[-<slug>].md, capturing N. It is the ONE
// grammar the read-side resolver matches, so a second copy in another package
// would drift and let a gate accept a filename the resolver later refuses when
// the record is cited. It is exported so record-lint validates each store's
// filenames against exactly this pattern rather than a looser local copy that
// accepted an arbitrary tail (iss-2608270908346617).
func FilenameNumRe(prefix string) *regexp.Regexp { return idRe(prefix) }
