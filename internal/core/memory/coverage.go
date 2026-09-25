package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// coverage.go — quotation-budget math + the regenerable coverage index (fn-39):
// the numbers behind the MQ lint family. The denominator is the registry's
// source_token_count (the discarded original is never re-read); the numerator is
// extracted from the retained distilled page.

const (
	numeratorTokenizerVersion = 1
	coverageIndexVersion      = 1
	coverageIndexName         = ".coverage_index.json"

	reasonNoTokenCount   = "no_source_token_count"
	reasonMalformedEntry = "malformed_registry_entry"
	reasonNoExternal     = "no_external_source"
)

// QuotationBudget is the pinned schema (fractions, not percentages); the defaults
// are the baked-in values used when config.json is absent.
type quotationBudget struct {
	PerPagePct              float64
	MaxContiguousQuoteWords int
	CumulativeWarnPct       float64
	CumulativeBlockPct      float64
}

func defaultBudget() quotationBudget {
	return quotationBudget{
		PerPagePct:              0.05,
		MaxContiguousQuoteWords: 150,
		CumulativeWarnPct:       0.15,
		CumulativeBlockPct:      0.25,
	}
}

func (b quotationBudget) asMap() map[string]any {
	return map[string]any{
		"per_page_pct":               b.PerPagePct,
		"max_contiguous_quote_words": b.MaxContiguousQuoteWords,
		"cumulative_warn_pct":        b.CumulativeWarnPct,
		"cumulative_block_pct":       b.CumulativeBlockPct,
	}
}

// loadQuotationBudget reads the store's config.json through the handle: a
// trust boundary, so a symlinked leaf is refused rather than followed and the
// read cannot leave the directory the handle vetted (iss-2608291814572914).
// Absent or unreadable is the default budget.
func loadQuotationBudget(store *storeHandle) quotationBudget {
	def := defaultBudget()
	raw, err := store.read("config.json", maxRegistryBytes)
	if err != nil {
		return def
	}
	var top map[string]any
	if json.Unmarshal(raw, &top) != nil {
		return def
	}
	block, ok := top["quotation_budget"].(map[string]any)
	if !ok {
		return def
	}
	frac := func(key string, d float64) float64 {
		v, ok := block[key].(float64)
		if !ok || v <= 0 {
			return d
		}
		return v
	}
	count := func(key string, d int) int {
		v, ok := block[key].(float64)
		if !ok || v <= 0 || v != float64(int(v)) {
			return d
		}
		return int(v)
	}
	return quotationBudget{
		PerPagePct:              frac("per_page_pct", def.PerPagePct),
		MaxContiguousQuoteWords: count("max_contiguous_quote_words", def.MaxContiguousQuoteWords),
		CumulativeWarnPct:       frac("cumulative_warn_pct", def.CumulativeWarnPct),
		CumulativeBlockPct:      frac("cumulative_block_pct", def.CumulativeBlockPct),
	}
}

// ---------------------------------------------------------------------------
// Quoted-span extraction — pinned v1 grammar
// ---------------------------------------------------------------------------

type quotedSpan struct {
	text       string
	normalized string
	tokenCount int
	line       int
}

func normalizeSpanText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

var (
	fenceDelimRe       = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	blockquoteRe       = regexp.MustCompile(`^ {0,3}>`)
	blockquoteMarkerRe = regexp.MustCompile(`^ {0,3}> ?`)
)

func fenceMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	openChar := byte(0)
	openLen := 0
	for i, line := range lines {
		if openChar == 0 {
			if m := fenceDelimRe.FindString(line); m != "" {
				run := strings.TrimLeft(m, " ")
				openChar, openLen = run[0], len(run)
				mask[i] = true
			}
		} else {
			mask[i] = true
			stripped := strings.TrimSpace(line)
			if len(stripped) >= openLen && stripped == strings.Repeat(string(openChar), len(stripped)) {
				openChar, openLen = 0, 0
			}
		}
	}
	return mask
}

// quotePairSpans returns (startRuneIndex, innerText) for each double-quoted
// prose span. Codepoint set: U+0022 (paired by successive occurrence) and
// U+201C/U+201D. Operates on runes so line offsets map correctly.
func quotePairSpans(runes []rune) [][2]any {
	var out [][2]any
	var straight []int
	for i, r := range runes {
		if r == '"' {
			straight = append(straight, i)
		}
	}
	for k := 0; k+1 < len(straight); k += 2 {
		a, b := straight[k], straight[k+1]
		out = append(out, [2]any{a, string(runes[a+1 : b])})
	}
	idx := 0
	for {
		a := indexRune(runes, '“', idx)
		if a == -1 {
			break
		}
		b := indexRune(runes, '”', a+1)
		if b == -1 {
			break
		}
		out = append(out, [2]any{a, string(runes[a+1 : b])})
		idx = b + 1
	}
	return out
}

func indexRune(runes []rune, target rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == target {
			return i
		}
	}
	return -1
}

func extractQuotedSpans(pageText string) []quotedSpan {
	body := pageText
	if _, b, err := splitFileFrontmatter(pageText); err == nil {
		body = b
	}
	prefix := pageText
	if body != "" {
		prefix = pageText[:len(pageText)-len(body)]
	}
	bodyStartLine := strings.Count(prefix, "\n") + 1

	lines := strings.Split(body, "\n")
	fenced := fenceMask(lines)
	var spans []quotedSpan

	appendSpan := func(text string, line int) {
		tokens := len(wordRe.FindAllString(text, -1))
		if tokens > 0 {
			spans = append(spans, quotedSpan{
				text:       text,
				normalized: normalizeSpanText(text),
				tokenCount: tokens,
				line:       line,
			})
		}
	}

	// Blockquote runs — contiguous > lines (outside fences) form one span.
	i := 0
	for i < len(lines) {
		if !fenced[i] && blockquoteRe.MatchString(lines[i]) {
			start := i
			var buf []string
			for i < len(lines) && !fenced[i] && blockquoteRe.MatchString(lines[i]) {
				buf = append(buf, blockquoteMarkerRe.ReplaceAllString(lines[i], ""))
				i++
			}
			appendSpan(strings.Join(buf, "\n"), bodyStartLine+start)
		} else {
			i++
		}
	}

	// Double-quoted prose on a line-preserving mask (fence + blockquote blanked).
	maskedLines := make([]string, len(lines))
	for j := range lines {
		if fenced[j] || blockquoteRe.MatchString(lines[j]) {
			maskedLines[j] = ""
		} else {
			maskedLines[j] = lines[j]
		}
	}
	masked := strings.Join(maskedLines, "\n")
	runes := []rune(masked)
	for _, sp := range quotePairSpans(runes) {
		start := sp[0].(int)
		inner := sp[1].(string)
		nl := 0
		for r := 0; r < start; r++ {
			if runes[r] == '\n' {
				nl++
			}
		}
		appendSpan(inner, bodyStartLine+nl)
	}

	return spans
}

// ---------------------------------------------------------------------------
// Text-based span dedup
// ---------------------------------------------------------------------------

type spanCharge struct {
	span        quotedSpan
	unambiguous bool
}

type keptSpan struct {
	normalized  string
	tokenCount  int
	unambiguous bool
}

func dedupSpans(charged []spanCharge) (int, int) {
	ordered := append([]spanCharge(nil), charged...)
	sort.SliceStable(ordered, func(i, j int) bool {
		li, lj := len(ordered[i].span.normalized), len(ordered[j].span.normalized)
		if li != lj {
			return li > lj
		}
		return ordered[i].span.normalized < ordered[j].span.normalized
	})
	var kept []*keptSpan
	for _, c := range ordered {
		if c.span.normalized == "" {
			continue
		}
		var container *keptSpan
		for _, k := range kept {
			if strings.Contains(k.normalized, c.span.normalized) {
				container = k
				break
			}
		}
		if container != nil {
			container.unambiguous = container.unambiguous || c.unambiguous
		} else {
			kept = append(kept, &keptSpan{normalized: c.span.normalized, tokenCount: c.span.tokenCount, unambiguous: c.unambiguous})
		}
	}
	total, unambiguous := 0, 0
	for _, k := range kept {
		total += k.tokenCount
		if k.unambiguous {
			unambiguous += k.tokenCount
		}
	}
	return unambiguous, total
}

func pageQuotedTokenTotal(spans []quotedSpan) int {
	charged := make([]spanCharge, len(spans))
	for i, s := range spans {
		charged[i] = spanCharge{span: s, unambiguous: true}
	}
	_, total := dedupSpans(charged)
	return total
}

// ---------------------------------------------------------------------------
// Span -> source attribution
// ---------------------------------------------------------------------------

func externalSourceHashes(source map[string]any) []string {
	if source == nil {
		return nil
	}
	var out []string
	if entries, ok := source["sources"].([]any); ok {
		for _, e := range entries {
			em, ok := e.(map[string]any)
			if !ok {
				continue
			}
			if isExternalClass(em["class"]) {
				if sh, ok := em["source_hash"].(string); ok && sh != "" && !contains(out, sh) {
					out = append(out, sh)
				}
			}
		}
		return out
	}
	// Non-`sources` shape: a scalar `class` and/or a plural `classes` list, with a
	// single top-level source_hash. This read only the scalar `class`, so a page
	// declaring `classes: [external_pdf]` dropped its source_hash out of coverage
	// accounting even as the lint's derivedClasses counted the external class — the
	// coverage sibling of the plural-classes licence gap (iss-2608270908341877).
	// Read the class union the way SourceClasses/derivedClasses do.
	for _, cls := range SourceClasses(source) {
		if isExternalClass(cls) {
			if sh, ok := source["source_hash"].(string); ok && sh != "" {
				out = append(out, sh)
			}
			break
		}
	}
	return out
}

func pageIsAmbiguous(source map[string]any) bool {
	return len(SourceHashes(source)) >= 2
}

// lookupSourceTokenCount returns (count, "") when the denominator is usable, or
// (0, reason) otherwise — never a divide-by-zero, never a false block.
func lookupSourceTokenCount(registry map[string]any, sourceHash string) (int, string) {
	if registry == nil {
		return 0, reasonMalformedEntry
	}
	entryAny, ok := registry[sourceHash]
	if !ok {
		return 0, reasonNoTokenCount
	}
	entry, ok := entryAny.(map[string]any)
	if !ok {
		return 0, reasonMalformedEntry
	}
	countAny, present := entry["source_token_count"]
	if !present || countAny == nil {
		return 0, reasonNoTokenCount
	}
	f, ok := countAny.(float64)
	if !ok {
		// A freshly-built (Go-native) registry may carry an int.
		if iv, ok := countAny.(int); ok {
			f = float64(iv)
		} else {
			return 0, reasonMalformedEntry
		}
	}
	if f == 0 {
		return 0, reasonNoTokenCount
	}
	if f < 0 || f != float64(int(f)) {
		return 0, reasonMalformedEntry
	}
	return int(f), ""
}

// ---------------------------------------------------------------------------
// Full-corpus coverage build + fingerprint
// ---------------------------------------------------------------------------

type sourceCoverage struct {
	SourceTokenCount        int      `json:"source_token_count"`
	QuotedTokensTotal       int      `json:"quoted_tokens_total"`
	QuotedTokensUnambiguous int      `json:"quoted_tokens_unambiguous"`
	CoverageTotal           float64  `json:"coverage_total"`
	CoverageUnambiguous     float64  `json:"coverage_unambiguous"`
	Pages                   []string `json:"pages"`
}

type coverageResult struct {
	sources     map[string]sourceCoverage
	unavailable map[string]string
	fingerprint string
}

func coveragePageSourceBlock(pageText string) map[string]any {
	fm, err := parseFrontmatter(pageText)
	if err != nil {
		return map[string]any{}
	}
	if src, ok := fm["source"].(map[string]any); ok {
		return src
	}
	return map[string]any{}
}

type crawledPage struct {
	rel  string
	text string
}

func computeFingerprint(pages []crawledPage, registry map[string]any, referenced []string, budget quotationBudget) string {
	h := sha256.New()
	frame := func(data []byte) {
		h.Write([]byte(strconv.Itoa(len(data)) + ":"))
		h.Write(data)
	}
	frame([]byte("abcd-coverage-fingerprint-v1"))
	frame([]byte(strconv.Itoa(numeratorTokenizerVersion)))
	budgetJSON, _ := json.Marshal(budget.asMap())
	frame(budgetJSON)
	sorted := append([]crawledPage(nil), pages...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].rel != sorted[j].rel {
			return sorted[i].rel < sorted[j].rel
		}
		return sorted[i].text < sorted[j].text
	})
	for _, p := range sorted {
		frame([]byte(p.rel))
		frame([]byte(p.text))
	}
	refSet := map[string]bool{}
	for _, sh := range referenced {
		refSet[sh] = true
	}
	refs := make([]string, 0, len(refSet))
	for sh := range refSet {
		refs = append(refs, sh)
	}
	sort.Strings(refs)
	for _, sh := range refs {
		var fieldsJSON []byte
		if entry, ok := registry[sh].(map[string]any); ok {
			fields := []any{entry["source_token_count"], entry["token_count_version"]}
			fieldsJSON, _ = json.Marshal(fields)
		} else {
			fieldsJSON = []byte("null")
		}
		frame([]byte(sh))
		frame(fieldsJSON)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildCoverage(pages []crawledPage, registry map[string]any, budget quotationBudget) coverageResult {
	charged := map[string][]spanCharge{}
	contributing := map[string][]string{}
	var referenced []string
	for _, p := range pages {
		block := coveragePageSourceBlock(p.text)
		for _, sh := range SourceHashes(block) {
			if !contains(referenced, sh) {
				referenced = append(referenced, sh)
			}
		}
		externals := externalSourceHashes(block)
		if len(externals) == 0 {
			continue
		}
		spans := extractQuotedSpans(p.text)
		if len(spans) == 0 {
			continue
		}
		unambiguous := !pageIsAmbiguous(block)
		for _, sh := range externals {
			for _, span := range spans {
				charged[sh] = append(charged[sh], spanCharge{span: span, unambiguous: unambiguous})
			}
			contributing[sh] = append(contributing[sh], p.rel)
		}
	}
	sources := map[string]sourceCoverage{}
	unavailable := map[string]string{}
	var chargedKeys []string
	for sh := range charged {
		chargedKeys = append(chargedKeys, sh)
	}
	sort.Strings(chargedKeys)
	for _, sh := range chargedKeys {
		count, reason := lookupSourceTokenCount(registry, sh)
		if reason != "" {
			unavailable[sh] = reason
			continue
		}
		unambiguousTokens, totalTokens := dedupSpans(charged[sh])
		pageSet := uniqueSorted(contributing[sh])
		sources[sh] = sourceCoverage{
			SourceTokenCount:        count,
			QuotedTokensTotal:       totalTokens,
			QuotedTokensUnambiguous: unambiguousTokens,
			CoverageTotal:           float64(totalTokens) / float64(count),
			CoverageUnambiguous:     float64(unambiguousTokens) / float64(count),
			Pages:                   pageSet,
		}
	}
	return coverageResult{
		sources:     sources,
		unavailable: unavailable,
		fingerprint: computeFingerprint(pages, registry, referenced, budget),
	}
}

// ---------------------------------------------------------------------------
// .coverage_index.json IO
// ---------------------------------------------------------------------------

func readStoredFingerprint(store *storeHandle) string {
	raw, err := store.read(coverageIndexName, maxRegistryBytes)
	if err != nil {
		return ""
	}
	var data map[string]any
	if json.Unmarshal(raw, &data) != nil {
		return ""
	}
	if fp, ok := data["fingerprint"].(string); ok {
		return fp
	}
	return ""
}

func writeCoverageIndex(path string, result coverageResult, budget quotationBudget) (map[string]any, error) {
	sourcesOut := map[string]any{}
	for sh, cov := range result.sources {
		sourcesOut[sh] = map[string]any{
			"source_token_count":        cov.SourceTokenCount,
			"quoted_tokens_total":       cov.QuotedTokensTotal,
			"quoted_tokens_unambiguous": cov.QuotedTokensUnambiguous,
			"coverage_total":            cov.CoverageTotal,
			"coverage_unambiguous":      cov.CoverageUnambiguous,
			"pages":                     toAnySlice(cov.Pages),
		}
	}
	unavailableOut := map[string]any{}
	for sh, reason := range result.unavailable {
		unavailableOut[sh] = reason
	}
	payload := map[string]any{
		"version":                     coverageIndexVersion,
		"fingerprint":                 result.fingerprint,
		"numerator_tokenizer_version": numeratorTokenizerVersion,
		"quotation_budget":            budget.asMap(),
		"sources":                     sourcesOut,
		"unavailable":                 unavailableOut,
	}
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return nil, err
	}
	if err := writeStringAtomic(path, marshalIndentNoEscape(payload)); err != nil {
		return nil, err
	}
	return payload, nil
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, os.PathSeparator); i >= 0 {
		return path[:i]
	}
	return "."
}

func uniqueSorted(ss []string) []string {
	set := map[string]bool{}
	for _, s := range ss {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func fmtPct(f float64) string { return fmt.Sprintf("%.1f%%", f*100) }
