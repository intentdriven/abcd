package memory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// bare.go — the SD001-non-mutating bare render: page count by class,
// last-ingest, contradictions surface, drift, and read-only fingerprint-gated
// quotation headroom. This path performs ZERO writes; it reports drift, never
// heals it.

// ClassCount is one page-count-by-class row.
type ClassCount struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

// BareStatus is the structured result of Bare.
type BareStatus struct {
	StorePresent   bool         `json:"store_present"`
	Pages          int          `json:"pages"`
	ByClass        []ClassCount `json:"by_class"`
	LastIngest     string       `json:"last_ingest"`
	Contradictions []string     `json:"contradictions"`
	Drift          []string     `json:"drift"`
	Headroom       []string     `json:"headroom"`
}

// Bare renders the read-only store status.
func Bare(repoRoot string) (BareStatus, error) {
	// Every read goes through the store handle: a symlinked store DIRECTORY is
	// refused when it is opened (GHSA-72rp), and no read below can leave it
	// (iss-2608291814572914).
	store, err := openStore(repoRoot)
	if err != nil {
		return BareStatus{}, err
	}
	defer store.Close()
	present := store.present()

	infos := barePageInfos(store)
	// Seed the collections non-nil so an empty or contradiction-free store
	// marshals them as [] in --json, not bare null (every --json collection is an
	// empty list, never null; a healthy store keeps an empty contradictions list).
	status := BareStatus{
		StorePresent:   present,
		Pages:          len(infos),
		ByClass:        []ClassCount{},
		Contradictions: []string{},
		Drift:          []string{},
	}

	counts := map[string]int{}
	for _, info := range infos {
		classes := info.Classes
		if len(classes) == 0 {
			classes = []string{"(unclassified)"}
		}
		for _, cls := range classes {
			counts[cls]++
		}
	}
	for cls, n := range counts {
		status.ByClass = append(status.ByClass, ClassCount{Class: cls, Count: n})
	}
	sort.Slice(status.ByClass, func(i, j int) bool {
		if status.ByClass[i].Count != status.ByClass[j].Count {
			return status.ByClass[i].Count > status.ByClass[j].Count
		}
		return status.ByClass[i].Class < status.ByClass[j].Class
	})

	registry := map[string]any{}
	if r, err := store.registry(); err == nil {
		registry = r
	}
	status.LastIngest = bareLastIngest(registry)

	if contrText, ok := store.readText("contradictions.md"); ok {
		for _, line := range strings.Split(contrText, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "- ") {
				status.Contradictions = append(status.Contradictions, t)
			}
		}
	}

	if present {
		desired := map[string]string{
			"index.md":          RenderIndex(infos),
			"contradictions.md": RenderContradictions(infos),
		}
		stale := map[string]bool{}
		for name, want := range desired {
			current, ok := store.readText(name)
			if !ok || sha256Hex(current) != sha256Hex(want) {
				stale[name] = true
			}
		}
		if stale["index.md"] {
			status.Drift = append(status.Drift, "index stale; run an ingest")
		}
		if stale["contradictions.md"] {
			status.Drift = append(status.Drift, "contradictions register stale; run an ingest")
		}
	}

	status.Headroom = bareHeadroomLines(store)
	return status, nil
}

func barePageInfos(store *storeHandle) []PageInfo {
	var infos []PageInfo
	for _, name := range store.pageNames() {
		if text, ok := store.readText(name); ok {
			infos = append(infos, pageInfoFrom(name, text))
		}
	}
	return infos
}

func bareLastIngest(registry map[string]any) string {
	last := ""
	for _, entryAny := range registry {
		entry, ok := entryAny.(map[string]any)
		if !ok {
			continue
		}
		consumers, ok := entry["consumers"].(map[string]any)
		if !ok {
			continue
		}
		memc, ok := consumers["memory"].(map[string]any)
		if !ok {
			continue
		}
		stamp := ""
		if s, ok := entry["last_ingest"].(string); ok && s != "" {
			stamp = s
		} else if s, ok := memc["ingested_at"].(string); ok {
			stamp = s
		}
		if stamp != "" && (last == "" || stamp > last) {
			last = stamp
		}
	}
	return last
}

func fmtSignedPct(fraction float64) string {
	pct := fraction * 100.0
	if pct < 0 {
		return fmt.Sprintf("over by %.0f%%", -pct)
	}
	return fmt.Sprintf("+%.0f%%", pct)
}

func bareHeadroomLines(store *storeHandle) []string {
	const header = "Quotation-budget headroom:"

	raw, err := store.read(coverageIndexName, maxRegistryBytes)
	if err != nil {
		return []string{header + " coverage index not built yet — run `abcd memory lint`"}
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return []string{header + " headroom unavailable — run `abcd memory lint`"}
	}
	storedFP, okFP := stored["fingerprint"].(string)
	sources, okSrc := stored["sources"].(map[string]any)
	unavailable, okUn := stored["unavailable"].(map[string]any)
	budgetBlock, okBud := stored["quotation_budget"].(map[string]any)
	if !okFP || !okSrc || !okUn || !okBud {
		return []string{header + " headroom unavailable — run `abcd memory lint`"}
	}

	// Read-only crawl over the same typed pages the lint crawls.
	pages := store.typedPages()
	registry, regErr := store.registry()
	if regErr != nil {
		registry = nil
	}
	budget := loadQuotationBudget(store)
	var referenced []string
	for _, p := range pages {
		block := coveragePageSourceBlock(p.text)
		for _, sh := range SourceHashes(block) {
			if !contains(referenced, sh) {
				referenced = append(referenced, sh)
			}
		}
	}
	currentFP := computeFingerprint(pages, registry, referenced, budget)
	if currentFP != storedFP {
		return []string{header + " coverage index stale — run `abcd memory lint`"}
	}
	if len(sources) == 0 && len(unavailable) == 0 {
		return []string{header + " no external-source coverage yet"}
	}

	num := func(v any) (float64, bool) {
		f, ok := v.(float64)
		return f, ok
	}
	warnPct, ok1 := num(budgetBlock["cumulative_warn_pct"])
	if !ok1 {
		warnPct = 0.15
	}
	blockPct, ok2 := num(budgetBlock["cumulative_block_pct"])
	if !ok2 {
		blockPct = 0.25
	}
	lines := []string{header}
	var srcKeys []string
	for sh := range sources {
		srcKeys = append(srcKeys, sh)
	}
	sort.Strings(srcKeys)
	for _, sh := range srcKeys {
		cov, ok := sources[sh].(map[string]any)
		if !ok {
			return []string{header + " headroom unavailable — run `abcd memory lint`"}
		}
		covTotal, okT := num(cov["coverage_total"])
		covUnamb, okU := num(cov["coverage_unambiguous"])
		if !okT || !okU {
			return []string{header + " headroom unavailable — run `abcd memory lint`"}
		}
		lines = append(lines, fmt.Sprintf("  %s: warn %s, block %s",
			short12(sh), fmtSignedPct(warnPct-covTotal), fmtSignedPct(blockPct-covUnamb)))
	}
	var unKeys []string
	for sh := range unavailable {
		unKeys = append(unKeys, sh)
	}
	sort.Strings(unKeys)
	for _, sh := range unKeys {
		lines = append(lines, fmt.Sprintf("  %s: coverage unavailable (%v)", short12(sh), unavailable[sh]))
	}
	return lines
}
