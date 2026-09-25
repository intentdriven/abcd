package lifeboat

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
)

// graveyard_abandoned.go — Layer 2 of the graveyard: what the project itself
// declared dead in its Tier-1/2 record. It is a pure function over the read-only
// SourceContext file surface (ctx.ReadFile/ctx.ListDir, both contained and
// capped) and frontmatter.Fields — it never touches git and never writes.
//
// Every qualifying record contributes a Finding keyed by the record's OWN id
// (an intent's itd-N, an ADR's adr-N, an issue's iss-N) so a layer-3 lesson can
// cite exactly the id a human reads in the record — no re-derivation. Findings
// are grouped in signalRank order (buildAbandoned appends signal by signal in
// that order) and, within a signal, in that signal's fixed deterministic order
// (sorted record id, sorted ADR id, or DECISIONS.md line order), so the
// assembled abandoned.json is byte-identical across re-plans of an unchanged
// repo and the pinned manifest hash stays stable.

// Record-id shapes. A finding is keyed by the record's own id only when that id
// has the expected shape; a record whose id is malformed (or whose id must be
// derived) is validated against these before it can key a finding. Intents and
// issues spell their id one way, so the shape gate is the whole check; ADRs go
// through gvCanonADRID below, which gates AND canonicalises.
var (
	gvIntentIDRe = regexp.MustCompile(`^itd-[0-9]+$`)
	gvIssueIDRe  = regexp.MustCompile(`^iss-[0-9]+$`)
)

// gvADRHandleRe captures an ADR id's ordinal digits. ADRs are the one family
// whose id is routinely written both padded and bare (`adr-0012` in a frontmatter
// block, `0012-slug.md` in a filename), and the rest of the record treats those
// as ONE handle — record dispatch (record.adrHandleRe) and the citation resolver
// (recordid.fileID) both compare the number, not the spelling. Layer 2 must
// agree, or one ADR keys two findings and the cross-home dedup this file
// documents cannot fire.
var gvADRHandleRe = regexp.MustCompile(`^adr-([0-9]+)$`)

// gvIntentHandleRe and gvIssueHandleRe capture an intent/issue id's ordinal digits.
// Intents and issues are not routinely written padded the way ADRs are, but nothing
// stops itd-7 and itd-007 both reaching a graveyard scan (a hand-edited record, a
// foreign convention), and they are ONE record — so the canonicaliser trims the
// padding the same way gvCanonADRID does, and the dedup collapses them.
var (
	gvIntentHandleRe = regexp.MustCompile(`^itd-([0-9]+)$`)
	gvIssueHandleRe  = regexp.MustCompile(`^iss-([0-9]+)$`)
)

// gvRejectionVerbs is the deliberately narrow set of verbs that mark a
// DECISIONS.md bullet as recording a rejected option. It is conservative on
// purpose: a broad list ("no", "instead", "not") would fire on ordinary prose,
// so only unambiguous abandonment verbs qualify. Matched as a substring of the
// lower-cased line, so "rejected" fires on "RAG rejected at this scale" but not
// on the unrelated word "rejection".
var gvRejectionVerbs = []string{
	"rejected", "dropped", "discarded", "abandoned", "ruled out", "deferred",
}

// maxAbandonedEvidencePerFinding bounds each unbounded evidence source a layer-2
// finding draws on: the alternatives-considered bullets (one line per bullet) and
// the shadowed-claimant notes gvIDClaims records. The cap keeps a hostile or
// pathological record home from ballooning a single finding. Unlike
// maxGraveyardFindingsPerSignal it promises no notice of its own — it bounds the
// evidence of a finding that is itself reported, not the findings a signal
// reports.
const maxAbandonedEvidencePerFinding = 32

// buildAbandoned reads what the project explicitly declared dead and returns the
// deterministic, evidence-only layer-2 record. Findings are grouped in
// signalRank order; the slice is never nil, so abandoned.json always marshals
// "findings": [] rather than null when the record declares nothing dead.
func buildAbandoned(ctx *SourceContext) Abandoned {
	var fs []Finding
	fs = append(fs, gvSupersededIntents(ctx)...)
	fs = append(fs, gvSupersededADRs(ctx)...)
	fs = append(fs, gvAlternativesConsidered(ctx)...)
	fs = append(fs, gvWontfixIssues(ctx)...)
	fs = append(fs, gvRejectedOptions(ctx)...)
	if fs == nil {
		fs = []Finding{}
	}
	return Abandoned{SchemaVersion: GraveyardSchemaVersion, Findings: fs}
}

// gvSupersededIntents reports every intent in the superseded/ bucket, keyed by
// its own itd-N id, sorted numerically by id.
func gvSupersededIntents(ctx *SourceContext) []Finding {
	dir := intent.IntentsRelDir + "/" + intent.BucketSuperseded
	var out []Finding
	claims := newGvIDClaims()
	names, truncated := ctx.listDirNoted(dir)
	for _, name := range names {
		if !strings.HasPrefix(name, "itd-") || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		path := dir + "/" + name
		fields, ok := gvFields(ctx, path)
		if !ok {
			continue
		}
		id := gvCanonIntentID(gvUnquote(fields["id"].Value))
		if id == "" || claims.shadowed(out, id, path) {
			continue
		}
		claims.take(id, len(out))
		out = append(out, Finding{
			ID:       id,
			Signal:   SignalSupersededIntent,
			Summary:  "intent superseded",
			Evidence: []string{gvText(path)},
		})
	}
	if truncated {
		out = append(out, gvListingTruncatedFinding(SignalSupersededIntent, ctx.listCap))
	}
	gvSortByID(out)
	return capSignalFindings(out)
}

// gvSupersededADRs reports every ADR the record marks superseded — by an
// explicit status: superseded or a non-null superseded_by — across the native
// ADR home and each conventional home. Keyed by the ADR's own canonical adr-N id
// (derived from the NNNN- filename when the frontmatter carries none), deduped
// first-wins across homes (native wins) with every shadowed claimant named in the
// retained finding's evidence, sorted numerically by id.
func gvSupersededADRs(ctx *SourceContext) []Finding {
	var out []Finding
	claims := newGvIDClaims()
	truncated := gvEachADR(ctx, func(name, path string, fields map[string]frontmatter.Field) {
		status := strings.ToLower(gvUnquote(fields["status"].Value))
		supBy := gvUnquote(fields["superseded_by"].Value)
		if status != "superseded" && frontmatter.IsNull(supBy) {
			return
		}
		id := gvADRID(fields, name)
		if id == "" || claims.shadowed(out, id, path) {
			return
		}
		claims.take(id, len(out))
		var ev []string
		if !frontmatter.IsNull(supBy) {
			ev = append(ev, gvText("superseded_by: "+supBy))
		}
		ev = append(ev, gvText(path))
		out = append(out, Finding{
			ID:       id,
			Signal:   SignalSupersededADR,
			Summary:  "ADR superseded",
			Evidence: ev,
		})
	})
	if truncated {
		out = append(out, gvListingTruncatedFinding(SignalSupersededADR, ctx.listCap))
	}
	gvSortByID(out)
	return capSignalFindings(out)
}

// gvAlternativesConsidered reports every ADR carrying an Alternatives-Considered
// (or Options-Considered) section, keyed by <adr-id>-alt, with each top-level
// bullet of the section as evidence. Deduped first-wins across homes, sorted by
// the underlying ADR id.
func gvAlternativesConsidered(ctx *SourceContext) []Finding {
	var out []Finding
	claims := newGvIDClaims()
	truncated := gvEachADR(ctx, func(name, path string, fields map[string]frontmatter.Field) {
		id := gvADRID(fields, name)
		if id == "" {
			return
		}
		data, ok := ctx.ReadFile(path)
		if !ok {
			return
		}
		bullets, found := gvSectionBullets(data, "alternatives considered", "alternatives", "options considered")
		if !found {
			return
		}
		// The claim is tested only once the record has something to contribute:
		// an ADR sharing an id but carrying no such section shadows nothing here.
		if claims.shadowed(out, id, path) {
			return
		}
		claims.take(id, len(out))
		if len(bullets) > maxAbandonedEvidencePerFinding {
			bullets = bullets[:maxAbandonedEvidencePerFinding]
		}
		ev := make([]string, 0, len(bullets))
		for _, b := range bullets {
			ev = append(ev, gvText(b))
		}
		f := Finding{
			ID:      adrAltID(id),
			Signal:  SignalAlternativesConsidered,
			Summary: strings.ToUpper(id) + " weighed and rejected alternatives",
		}
		if len(ev) > 0 {
			f.Evidence = ev
		}
		out = append(out, f)
	})
	if truncated {
		out = append(out, gvListingTruncatedFinding(SignalAlternativesConsidered, ctx.listCap))
	}
	// Sort by the ADR id embedded in the <adr-id>-alt finding id.
	gvSortByID(out)
	return capSignalFindings(out)
}

// gvWontfixIssues reports every issue in the wontfix/ ledger bucket, keyed by its
// own iss-N id, evidence being the wontfix_reason (or the slug when absent),
// sorted numerically by id.
func gvWontfixIssues(ctx *SourceContext) []Finding {
	dir := capture.LedgerRelPath + "/wontfix"
	var out []Finding
	claims := newGvIDClaims()
	names, truncated := ctx.listDirNoted(dir)
	for _, name := range names {
		if !strings.HasPrefix(name, "iss-") || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		path := dir + "/" + name
		fields, ok := gvFields(ctx, path)
		if !ok {
			continue
		}
		id := gvCanonIssueID(gvUnquote(fields["id"].Value))
		if id == "" || claims.shadowed(out, id, path) {
			continue
		}
		claims.take(id, len(out))
		var ev []string
		if reason := gvUnquote(fields["wontfix_reason"].Value); reason != "" {
			ev = append(ev, gvText("wontfix_reason: "+reason))
		} else if slug := gvUnquote(fields["slug"].Value); slug != "" {
			ev = append(ev, gvText("slug: "+slug))
		}
		f := Finding{ID: id, Signal: SignalWontfixIssue, Summary: "issue closed wontfix"}
		if len(ev) > 0 {
			f.Evidence = ev
		}
		out = append(out, f)
	}
	if truncated {
		out = append(out, gvListingTruncatedFinding(SignalWontfixIssue, ctx.listCap))
	}
	gvSortByID(out)
	return capSignalFindings(out)
}

// gvRejectedOptions reports every top-level DECISIONS.md bullet whose text
// carries a conservative rejection verb, keyed by dec-L<line> (1-based line
// number, stable for an unchanged append-only file), in file (line) order.
func gvRejectedOptions(ctx *SourceContext) []Finding {
	data, ok := ctx.ReadFile(nativeDecisions)
	if !ok {
		return nil
	}
	var out []Finding
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if !strings.HasPrefix(line, "- ") { // top-level bullet only (no indentation)
			continue
		}
		low := strings.ToLower(line)
		matched := false
		for _, v := range gvRejectionVerbs {
			if strings.Contains(low, v) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		out = append(out, Finding{
			ID:       decisionID(i + 1),
			Signal:   SignalRejectedOption,
			Summary:  "decision log records a rejected option",
			Evidence: []string{gvText(strings.TrimSpace(line))},
		})
	}
	return capSignalFindings(out)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// gvIDClaims tracks which finding holds which record id while a signal is being
// built, so a second record claiming an id already taken is ANNOUNCED rather than
// dropped in silence. Two records genuinely can claim one ADR id — the same ADR
// copied into a second home (the dedup this file documents) and, because a number
// is not a unique name, two distinct ADRs numbered alike (adr-tools branches
// colliding, or log4brains' YYYYMMDD- filenames for two ADRs written the same
// day). Nothing in the id can tell those apart, so both still dedupe first-wins;
// what the loud-staging rule forbids is the drop leaving no marker at all, since
// a shadowed record is then permanently uncitable by the layer-3 interpreter and
// nothing in the packed lifeboat says why.
type gvIDClaims struct {
	at      map[string]int // canonical id -> index of the finding that holds it
	shadows map[string]int // canonical id -> notes already recorded against it
}

func newGvIDClaims() *gvIDClaims {
	return &gvIDClaims{at: map[string]int{}, shadows: map[string]int{}}
}

// take records that the finding at index i in the signal's slice holds id.
func (c *gvIDClaims) take(id string, i int) { c.at[id] = i }

// shadowed reports whether id is already claimed and, when it is, notes path in
// the Notices of the finding that holds it — the retained finding names every
// claimant it shadowed, the path quoted as an operand of the binary's notice. The notes are bounded like any other unbounded evidence source, so a
// hostile record home carrying thousands of same-numbered files cannot balloon
// one finding.
func (c *gvIDClaims) shadowed(out []Finding, id, path string) bool {
	i, dup := c.at[id]
	if !dup {
		return false
	}
	if c.shadows[id] < maxAbandonedEvidencePerFinding {
		c.shadows[id]++
		f := out[i]
		f.Notices = append(append([]string(nil), f.Notices...),
			fmt.Sprintf("shadowed: %s also claims %s and is not separately reported", gvQuoted(path), id))
		out[i] = f
	}
	return true
}

// gvListingTruncatedFinding is the per-scan sibling of noteFindingsOmitted: when a
// record home holds more entries than the per-directory listing cap, the scanner
// read only a prefix of it, so the signal it feeds is INCOMPLETE. A cheap synthetic
// finding says so, so a packed abandoned.json never presents a cap-truncated input
// as a complete scan (iss-2608270908348796). It carries the signal it belongs to (so
// it groups and survives capSignalFindings), and a fixed non-numeric id so gvSortByID
// orders it stably ahead of the signal's record-keyed findings. The statement is the
// binary's own, so it travels in Notices; the finding cites no evidence.
func gvListingTruncatedFinding(sig Signal, listCap int) Finding {
	return Finding{
		ID:      "gv-listing-truncated-" + idClean(string(sig)),
		Signal:  sig,
		Summary: "record listing truncated at the per-directory cap; some records were not scanned",
		Notices: []string{fmt.Sprintf(
			"a record home held more than %d entries; only %d of them were scanned", listCap, listCap)},
	}
}

// gvADRHomes lists the ADR directories in dedup priority order: the native home
// first (so it wins a first-writer-wins tie), then the conventional homes.
func gvADRHomes() []string {
	return append([]string{nativeADRDir}, convADRDirs...)
}

// gvEachADR calls fn for every ADR document (*.md/*.markdown) under every ADR
// home, in home order then sorted-name order, having parsed its frontmatter. A
// file that cannot be read is skipped.
func gvEachADR(ctx *SourceContext, fn func(name, path string, fields map[string]frontmatter.Field)) (truncated bool) {
	for _, dir := range gvADRHomes() {
		names, trunc := ctx.listDirNoted(dir)
		if trunc {
			truncated = true
		}
		for _, name := range names {
			low := strings.ToLower(name)
			if !strings.HasSuffix(low, ".md") && !strings.HasSuffix(low, ".markdown") {
				continue
			}
			path := dir + "/" + name
			fields, ok := gvFields(ctx, path)
			if !ok {
				continue
			}
			fn(name, path, fields)
		}
	}
	return truncated
}

// gvADRID resolves an ADR's id: its frontmatter id when it is a valid adr-N,
// else the id derived from a leading NNNN- filename, else "" (skip). BOTH paths
// return the canonical spelling — an id read verbatim from the frontmatter while
// the filename path stripped padding made `id: adr-012` and a filename-derived
// `adr-12` two ids for one ADR, which is precisely the case the cross-home dedup
// exists to collapse.
func gvADRID(fields map[string]frontmatter.Field, name string) string {
	if id := gvCanonADRID(gvUnquote(fields["id"].Value)); id != "" {
		return id
	}
	return gvCanonADRID(gvADRIDFromFilename(name))
}

// gvCanonADRID canonicalises an ADR id to the one spelling every claimant of that
// ADR resolves to: adr-<ordinal> with its leading zeros trimmed (an all-zero
// ordinal collapsing to adr-0). "" when the string is not an ADR id at all.
//
// The trim is TEXTUAL, never an integer parse: an ordinal wider than an int is
// still a well-formed adr-N, but a parse of it fails, and an id that canonicalises
// to "" is a record dropped in silence — the very failure the dedup around it
// exists to announce. Trimming text has no such edge, and agrees with
// record.adrHandleRe and recordid.fileID for every ordinal they can represent.
func gvCanonADRID(id string) string { return gvCanonNumID(gvADRHandleRe, "adr-", id) }

// gvCanonIntentID / gvCanonIssueID are the itd/iss twins of gvCanonADRID: they
// canonicalise a record id to the one spelling every claimant resolves to,
// trimming leading zeros textually (never an int parse, so an over-int ordinal
// keeps its identity rather than collapsing to "" and being dropped in silence).
// "" when the string is not an id of that family.
func gvCanonIntentID(id string) string { return gvCanonNumID(gvIntentHandleRe, "itd-", id) }
func gvCanonIssueID(id string) string  { return gvCanonNumID(gvIssueHandleRe, "iss-", id) }

// gvCanonNumID is the shared textual canonicaliser behind gvCanonADRID and its
// itd/iss twins: it matches <prefix><ordinal>, trims the ordinal's leading zeros
// (an all-zero ordinal collapsing to <prefix>0), and never parses the ordinal as an
// integer — so an ordinal wider than any int is still a well-formed handle and keeps
// its identity. "" when id does not match the family's shape.
func gvCanonNumID(re *regexp.Regexp, prefix, id string) string {
	m := re.FindStringSubmatch(id)
	if m == nil {
		return ""
	}
	ordinal := strings.TrimLeft(m[1], "0")
	if ordinal == "" { // the id was all zeros; <prefix>000 and <prefix>0 are one handle
		ordinal = "0"
	}
	return prefix + ordinal
}

// gvADRIDFromFilename derives adr-N from a leading run of digits in an NNNN-slug
// ADR filename, or "" when the name is not numbered. The derivation is TEXTUAL —
// "adr-" + the raw digit run, leaving the leading-zero trim to gvCanonADRID — and
// never an strconv.Atoi: a filename ordinal wider than an int is still a well-formed
// adr-N, but a parse of it fails and an "" id is a record dropped in silence, the
// very failure the cross-home dedup exists to announce (iss-2608270945469978).
func gvADRIDFromFilename(name string) string {
	i := 0
	for i < len(name) && name[i] >= '0' && name[i] <= '9' {
		i++
	}
	if i == 0 {
		return ""
	}
	return "adr-" + name[:i]
}

// gvFields reads a record and parses its leading frontmatter block. ok is false
// when the file is absent/oversize/non-regular (the ReadFile guards).
func gvFields(ctx *SourceContext, path string) (map[string]frontmatter.Field, bool) {
	data, ok := ctx.ReadFile(path)
	if !ok {
		return nil, false
	}
	return frontmatter.Fields(strings.Split(string(data), "\n")), true
}

// gvSectionBullets finds the first Markdown section whose heading contains any
// keyword (case-insensitive) and returns that section's top-level bullet lines
// (each trimmed) up to the next heading. found reports whether such a section
// exists at all — a section present but bullet-free still counts as found.
func gvSectionBullets(data []byte, keywords ...string) (bullets []string, found bool) {
	inSection := false
	for _, raw := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "#") { // a heading
			if inSection {
				break // the next heading closes the section
			}
			low := strings.ToLower(trimmed)
			for _, k := range keywords {
				if strings.Contains(low, k) {
					inSection = true
					found = true
					break
				}
			}
			continue
		}
		if !inSection {
			continue
		}
		// A top-level bullet has no leading indentation.
		if len(raw) > 0 && raw[0] != ' ' && raw[0] != '\t' &&
			(strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			bullets = append(bullets, trimmed)
		}
	}
	return bullets, found
}

// gvUnquote strips one layer of surrounding matching quotes (frontmatter values
// are sometimes quoted — issues quote their id/slug — and sometimes bare —
// intents and ADRs). It trims surrounding whitespace first.
func gvUnquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// gvSortByID sorts findings numerically by the first integer in their id, then
// lexically — so itd-2 precedes itd-10 within a signal. Every finding in one
// slice shares a signal (and therefore an id prefix), so the numeric key orders
// them the way a human reads a record.
func gvSortByID(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		ki, kj := gvIDNum(fs[i].ID), gvIDNum(fs[j].ID)
		if ki != kj {
			return ki < kj
		}
		return fs[i].ID < fs[j].ID
	})
}

// gvIDNum returns the value of the first run of digits in id, or -1 if none.
func gvIDNum(id string) int {
	start := -1
	for i := 0; i < len(id); i++ {
		if id[i] >= '0' && id[i] <= '9' {
			start = i
			break
		}
	}
	if start < 0 {
		return -1
	}
	end := start
	for end < len(id) && id[end] >= '0' && id[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(id[start:end])
	return n
}
