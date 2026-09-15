package memory

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
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
	// Refuse a symlinked store DIRECTORY (GHSA-72rp): the leaf O_NOFOLLOW guards
	// below do not contain a symlinked ancestor, so a committed `.abcd/memory`
	// symlink would otherwise have its out-of-repo pages crawled and disclosed.
	mem, present, err := safeMemoryDir(repoRoot)
	if err != nil {
		return BareStatus{}, err
	}

	infos := barePageInfos(mem)
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
	if r, err := LoadRegistry(SourcesIndexPath(repoRoot)); err == nil {
		registry = r
	}
	status.LastIngest = bareLastIngest(registry)

	if contrText, ok := readOrEmpty(filepath.Join(mem, "contradictions.md")); ok {
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
			current, ok := readOrEmpty(filepath.Join(mem, name))
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

	status.Headroom = bareHeadroomLines(repoRoot, mem)
	return status, nil
}

func barePageInfos(mem string) []PageInfo {
	entries, err := os.ReadDir(mem)
	if err != nil {
		return nil
	}
	var infos []PageInfo
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Type().IsRegular() && IsMemoryPageName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if text, ok := readOrEmpty(filepath.Join(mem, name)); ok {
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

func bareHeadroomLines(repoRoot, mem string) []string {
	const header = "Quotation-budget headroom:"
	indexPath := CoverageIndexPath(repoRoot)

	raw, err := fsutil.ReadGuarded(indexPath, maxRegistryBytes)
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
	var pages []crawledPage
	_ = filepath.WalkDir(mem, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if !isTypedMemoryPagePath(mem, path) {
			return nil
		}
		if b, err := fsutil.ReadGuarded(path, maxMemoryPageBytes); err == nil {
			rel, _ := filepath.Rel(mem, path)
			pages = append(pages, crawledPage{rel: filepath.ToSlash(rel), text: string(b)})
		}
		return nil
	})
	registry, regErr := LoadRegistry(SourcesIndexPath(repoRoot))
	if regErr != nil {
		registry = nil
	}
	budget := loadQuotationBudget(repoRoot)
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

// readOrEmpty reads one store file through the guarded primitive: the store
// sits inside the repo working tree — a trust boundary — so a committed
// symlink leaf is refused rather than followed, and the read is size-capped.
func readOrEmpty(path string) (string, bool) {
	raw, err := fsutil.ReadGuarded(path, maxMemoryPageBytes)
	if err != nil {
		return "", false
	}
	return string(raw), true
}
