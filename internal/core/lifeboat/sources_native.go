package lifeboat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// nativeSources returns the Tier-2 adapters: brief sections derivable from an
// abcd record under .abcd/ (decisions, intents, specs, brief, roadmap, work
// issues and reviews, DECISIONS.md). Every adapter reports TierNative and reads
// only through the SourceContext file surface — it never touches git and never
// writes.
//
// Most brief sections are grounded by a single parameterised nativeBriefSource,
// one instance per row of the mapping Table, reading the section's own brief
// file. The sections whose native evidence lives outside the brief tree — ADRs,
// the issue ledger, the intent corpus, the conventions router, the glossary, and
// the two evidence sections synthesised from several places — get dedicated
// adapters below. "graveyard" gets none: what a project abandoned is a Tier-0
// git signal, not a record one.
func nativeSources() []Source {
	// Sections whose native evidence a dedicated adapter (or a poorer tier)
	// owns, so nativeBriefSource must NOT also speak for them.
	dedicated := map[Section]bool{
		"graveyard":               true, // Tier-0 git owns the graveyard
		"docs/adrs":               true,
		"activity/issues":         true,
		"rescue/spine":            true,
		"evidence/tradeoffs":      true,
		"evidence/open-questions": true,
		"constraints/invariants":  true,
		"constraints/naming":      true,
		"product/personas":        true,
	}

	var sources []Source
	for _, m := range Table {
		if dedicated[m.Section] {
			continue
		}
		sources = append(sources, nativeBriefSource{section: m.Section, lifeboatPath: m.LifeboatPath})
	}
	sources = append(sources,
		nativeADRsSource{},
		nativeIssuesSource{},
		nativeTradeoffsSource{},
		nativeOpenQuestionsSource{},
		nativeSpineSource{},
		nativeInvariantsSource{},
		nativeNamingSource{},
		nativePersonasSource{},
	)
	return sources
}

// nativeGroundedBodyBytes is the body-prose threshold above which a brief file
// is treated as authored content (grounded) rather than a stub (partial). Body
// prose is every non-blank line except a single leading "# Heading" line — the
// same measure used to tell a written section from a scaffolded placeholder.
const nativeGroundedBodyBytes = 200

// nativeBriefRoot is the on-disk prefix under which the brief is laid out per
// each mapping row's LifeboatPath.
const nativeBriefRoot = ".abcd/development/"

// nativeBriefFilePath resolves a mapping LifeboatPath to the brief file to read.
// A path ending in "/" is a directory whose README.md carries the section; any
// other path is the section file itself.
func nativeBriefFilePath(lifeboatPath string) string {
	p := nativeBriefRoot + lifeboatPath
	if strings.HasSuffix(lifeboatPath, "/") {
		return p + "README.md"
	}
	return p
}

// nativeBodyBytes counts the body-prose characters of a brief file: every
// non-blank line except a single leading heading. It is the measure that
// separates an authored section from a stub.
func nativeBodyBytes(data []byte) int {
	total := 0
	seenContent := false
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if !seenContent && strings.HasPrefix(t, "#") {
			seenContent = true // strip the single leading "# Heading" line
			continue
		}
		seenContent = true
		total += len(t)
	}
	return total
}

// nativeHasHeadingLike reports whether any Markdown heading in data contains one
// of keywords (case-insensitive).
func nativeHasHeadingLike(data []byte, keywords ...string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			continue
		}
		low := strings.ToLower(t)
		for _, k := range keywords {
			if strings.Contains(low, k) {
				return true
			}
		}
	}
	return false
}

// nativeIsNumbered reports whether a filename begins with a digit, the shape of
// an NNNN-slug ADR or an ordered record — used to skip a README.md sitting
// alongside numbered documents.
func nativeIsNumbered(name string) bool {
	return name != "" && name[0] >= '0' && name[0] <= '9'
}

// Record locations under the abcd tree, named once so a citation is consistent.
const (
	nativeADRDir      = ".abcd/development/decisions/adrs"
	nativeIssuesDir   = ".abcd/work/issues"
	nativeIntentsDir  = ".abcd/development/intents"
	nativeGlossaryDir = ".abcd/development/brief/glossary"
	nativeDecisions   = ".abcd/work/DECISIONS.md"
	nativePersonas    = ".abcd/development/brief/01-product/05-personas.md"
)

// nativeIssueStates are the capture-ledger subdirectories, in a fixed order so
// the count citation is deterministic. It is issueschema.StatusDirs, the ledger's
// ONE status list: a lifeboat that counted a set of folders the ledger no longer
// keeps (or missed one it gained) would cite a number no surface agrees with.
var nativeIssueStates = issueschema.StatusDirs

// nativeCountRecords counts the entries directly under dir whose name has the
// given prefix and a .md suffix (e.g. "iss-" issues, "itd-" intents).
func nativeCountRecords(ctx *SourceContext, dir, prefix string) int {
	n := 0
	for _, name := range ctx.ListDir(dir) {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
			n++
		}
	}
	return n
}

// nativeCountIntents counts itd-*.md records across the intent corpus, both at
// the top level and one directory deep (drafts/, planned/, shipped/, …).
func nativeCountIntents(ctx *SourceContext) int {
	total := nativeCountRecords(ctx, nativeIntentsDir, "itd-")
	for _, sub := range ctx.ListDir(nativeIntentsDir) {
		subdir := nativeIntentsDir + "/" + sub
		if ctx.IsDir(subdir) {
			total += nativeCountRecords(ctx, subdir, "itd-")
		}
	}
	return total
}

// nativeBriefSource grounds a brief section from its own file under
// .abcd/development/. An authored file (body prose over the threshold) grounds
// the section; a stub file is partial; a missing file is a blank a human must
// fill.
type nativeBriefSource struct {
	section      Section
	lifeboatPath string
}

func (s nativeBriefSource) Section() Section { return s.section }
func (nativeBriefSource) Tier() Tier         { return TierNative }

func (s nativeBriefSource) Probe(ctx *SourceContext) Evidence {
	path := nativeBriefFilePath(s.lifeboatPath)
	data, ok := ctx.ReadFile(path)
	if !ok {
		return blank(
			[]string{path},
			"What belongs in "+string(s.section)+"? No brief file at "+path+".",
		)
	}
	if nativeBodyBytes(data) < nativeGroundedBodyBytes {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceLow,
			Sources:    []string{path + " (stub)"},
		}
	}
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    []string{path},
	}
}

// nativeADRsSource grounds "docs/adrs" from the record's ADR directory, listing
// the numbered decision documents found. Blank when the record holds no ADRs.
type nativeADRsSource struct{}

func (nativeADRsSource) Section() Section { return "docs/adrs" }
func (nativeADRsSource) Tier() Tier       { return TierNative }

func (nativeADRsSource) Probe(ctx *SourceContext) Evidence {
	var adrs []string
	for _, name := range ctx.ListDir(nativeADRDir) {
		if nativeIsNumbered(name) && strings.HasSuffix(strings.ToLower(name), ".md") {
			adrs = append(adrs, nativeADRDir+"/"+name)
		}
	}
	if len(adrs) == 0 {
		return blank(
			[]string{nativeADRDir + "/NNNN-*.md"},
			"What architectural decisions has this project recorded? No ADRs in the record.",
		)
	}
	sources := []string{fmt.Sprintf("%d ADR(s) in the record", len(adrs))}
	sources = append(sources, adrs...)
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    dedupeSorted(sources),
	}
}

// nativeIssuesSource grounds "activity/issues" from the capture ledger, counting
// the iss-*.md records in each state. Blank when no issue has been captured.
type nativeIssuesSource struct{}

func (nativeIssuesSource) Section() Section { return "activity/issues" }
func (nativeIssuesSource) Tier() Tier       { return TierNative }

func (nativeIssuesSource) Probe(ctx *SourceContext) Evidence {
	var counts []string
	total := 0
	for _, state := range nativeIssueStates {
		n := nativeCountRecords(ctx, nativeIssuesDir+"/"+state, "iss-")
		if n > 0 {
			counts = append(counts, fmt.Sprintf("%s: %d", state, n))
			total += n
		}
	}
	if total == 0 {
		return blank(
			[]string{nativeIssuesDir + "/{open,resolved,wontfix}/iss-*.md"},
			"What issues has this project captured? The capture ledger is empty.",
		)
	}
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    []string{nativeIssuesDir + " (" + strings.Join(counts, ", ") + ")"},
	}
}

// nativeTradeoffsSource grounds "evidence/tradeoffs" from the alternatives an ADR
// weighed and the decision log. ADRs with an Alternatives-Considered section
// ground it; the decision log alone is partial. Blank when the record has
// neither.
type nativeTradeoffsSource struct{}

func (nativeTradeoffsSource) Section() Section { return "evidence/tradeoffs" }
func (nativeTradeoffsSource) Tier() Tier       { return TierNative }

func (nativeTradeoffsSource) Probe(ctx *SourceContext) Evidence {
	adrsWithAlts := 0
	for _, name := range ctx.ListDir(nativeADRDir) {
		if !nativeIsNumbered(name) || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		data, ok := ctx.ReadFile(nativeADRDir + "/" + name)
		if !ok {
			continue
		}
		if nativeHasHeadingLike(data, "alternatives considered", "alternatives", "options considered") {
			adrsWithAlts++
		}
	}
	var sources []string
	if adrsWithAlts > 0 {
		sources = append(sources, fmt.Sprintf("%d ADR(s) with an Alternatives Considered section", adrsWithAlts))
	}
	if d, ok := ctx.ReadFile(nativeDecisions); ok && nativeBodyBytes(d) > 0 {
		sources = append(sources, nativeDecisions)
	}
	if len(sources) == 0 {
		return blank(
			[]string{nativeADRDir + "/*.md (Alternatives Considered)", nativeDecisions},
			"What did this project weigh and reject, and why? No ADR alternatives or decision log in the record.",
		)
	}
	if adrsWithAlts == 0 {
		return Evidence{Status: StatusPartial, Confidence: ConfidenceMedium, Sources: sources}
	}
	return Evidence{Status: StatusGrounded, Confidence: ConfidenceHigh, Sources: sources}
}

// nativeOpenQuestionsSource grounds "evidence/open-questions" from open issues
// and the intent corpus. Open issues are concrete open questions and ground it;
// intents alone are partial. Blank when the record has neither.
type nativeOpenQuestionsSource struct{}

func (nativeOpenQuestionsSource) Section() Section { return "evidence/open-questions" }
func (nativeOpenQuestionsSource) Tier() Tier       { return TierNative }

func (nativeOpenQuestionsSource) Probe(ctx *SourceContext) Evidence {
	openDir := nativeIssuesDir + "/open"
	openCount := nativeCountRecords(ctx, openDir, "iss-")
	intents := nativeCountIntents(ctx)

	var sources []string
	if openCount > 0 {
		sources = append(sources, fmt.Sprintf("%d open issue(s) in %s", openCount, openDir))
	}
	if intents > 0 {
		sources = append(sources, fmt.Sprintf("%d intent(s) under %s", intents, nativeIntentsDir))
	}
	if len(sources) == 0 {
		return blank(
			[]string{openDir + "/iss-*.md", nativeIntentsDir},
			"What questions remain open? No open issues or intents in the record.",
		)
	}
	if openCount == 0 {
		return Evidence{Status: StatusPartial, Confidence: ConfidenceMedium, Sources: sources}
	}
	return Evidence{Status: StatusGrounded, Confidence: ConfidenceHigh, Sources: sources}
}

// nativeSpineSource grounds "rescue/spine" from the intent corpus: where a record
// exists, the sequence of intents is the project's spine. Blank when the corpus
// is empty.
type nativeSpineSource struct{}

func (nativeSpineSource) Section() Section { return "rescue/spine" }
func (nativeSpineSource) Tier() Tier       { return TierNative }

func (nativeSpineSource) Probe(ctx *SourceContext) Evidence {
	n := nativeCountIntents(ctx)
	if n == 0 {
		return blank(
			[]string{nativeIntentsDir},
			"Is there an intent corpus to reconstruct a spine from? None in the record.",
		)
	}
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    []string{fmt.Sprintf("%s (%d intent(s))", nativeIntentsDir, n)},
	}
}

// nativeInvariantsSource grounds "constraints/invariants" from the conventions
// router (AGENTS.md / CLAUDE.md) and any lint configuration under .abcd/. The
// router grounds it; lint config alone is partial. Blank when neither exists.
type nativeInvariantsSource struct{}

func (nativeInvariantsSource) Section() Section { return "constraints/invariants" }
func (nativeInvariantsSource) Tier() Tier       { return TierNative }

func (nativeInvariantsSource) Probe(ctx *SourceContext) Evidence {
	var sources []string
	hasRouter := false
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		if ctx.Exists(f) {
			sources = append(sources, f)
			hasRouter = true
		}
	}
	for _, name := range ctx.ListDir(".abcd") {
		if strings.HasSuffix(name, ".json") {
			sources = append(sources, ".abcd/"+name)
		}
	}
	if len(sources) == 0 {
		return blank(
			[]string{"AGENTS.md", "CLAUDE.md", ".abcd/*.json"},
			"What invariants must always hold? No conventions router or lint config in the record.",
		)
	}
	sort.Strings(sources)
	if !hasRouter {
		return Evidence{Status: StatusPartial, Confidence: ConfidenceMedium, Sources: sources}
	}
	return Evidence{Status: StatusGrounded, Confidence: ConfidenceHigh, Sources: sources}
}

// nativeNamingSource grounds "constraints/naming" from the brief glossary. A
// glossary carrying real prose grounds it; an empty or stub glossary is partial;
// no glossary directory is a blank.
type nativeNamingSource struct{}

func (nativeNamingSource) Section() Section { return "constraints/naming" }
func (nativeNamingSource) Tier() Tier       { return TierNative }

func (nativeNamingSource) Probe(ctx *SourceContext) Evidence {
	if !ctx.IsDir(nativeGlossaryDir) {
		return blank(
			[]string{nativeGlossaryDir},
			"What names and reserved vocabulary are fixed? No brief glossary in the record.",
		)
	}
	var files []string
	body := 0
	for _, name := range ctx.ListDir(nativeGlossaryDir) {
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		rel := nativeGlossaryDir + "/" + name
		files = append(files, rel)
		if d, ok := ctx.ReadFile(rel); ok {
			body += nativeBodyBytes(d)
		}
	}
	if len(files) == 0 {
		return blank(
			[]string{nativeGlossaryDir + "/*.md"},
			"What names and reserved vocabulary are fixed? The brief glossary is empty.",
		)
	}
	if body < nativeGroundedBodyBytes {
		return Evidence{Status: StatusPartial, Confidence: ConfidenceMedium, Sources: dedupeSorted(files)}
	}
	return Evidence{Status: StatusGrounded, Confidence: ConfidenceHigh, Sources: dedupeSorted(files)}
}

// nativePersonasSource is the deliberately hard case: "product/personas" is a
// human question rarely written down anywhere. It reaches at most PARTIAL, and
// only when an authored personas file exists; otherwise it returns a blank —
// the expected, correct result for a section a repository cannot supply.
type nativePersonasSource struct{}

func (nativePersonasSource) Section() Section { return "product/personas" }
func (nativePersonasSource) Tier() Tier       { return TierNative }

func (nativePersonasSource) Probe(ctx *SourceContext) Evidence {
	data, ok := ctx.ReadFile(nativePersonas)
	if !ok || nativeBodyBytes(data) < nativeGroundedBodyBytes {
		return blank(
			[]string{nativePersonas},
			"Who are the personas this product serves? Personas are a human question, rarely derivable from a repository.",
		)
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: ConfidenceLow,
		Sources:    []string{nativePersonas},
	}
}
