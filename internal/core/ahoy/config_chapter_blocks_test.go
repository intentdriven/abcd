package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestConfigurationChapterNamesThisRepositorysBlocks (iss-2611060809301279):
// the configuration chapter's sentence on this repository's own config names
// as many blocks, and the same blocks, as .abcd/config.json carries, and the
// blocks it says this repository sets by hand include status, whose
// scheduled_workflow key the chapter's hand-set list names.
func TestConfigurationChapterNamesThisRepositorysBlocks(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".abcd", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	var want []string
	for k := range cfg {
		want = append(want, k)
	}
	sort.Strings(want)

	b, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "05-internals", "03-configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(b)), " ")
	m := regexp.MustCompile("This repository's own config carries (\\w+) of these blocks — ([^.]*)\\. All but ([^,]*(?:, [^,]*)*?), which this repository sets by hand").FindStringSubmatch(text)
	if m == nil {
		t.Fatal("the chapter's sentence on this repository's own config blocks was not found")
	}
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten",
		"eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen"}
	if len(want) >= len(words) || m[1] != words[len(want)] {
		t.Errorf("the chapter says this repository's config carries %q blocks; .abcd/config.json carries %d", m[1], len(want))
	}
	tick := regexp.MustCompile("`([a-z_]+)`")
	var listed []string
	for _, s := range tick.FindAllStringSubmatch(m[2], -1) {
		listed = append(listed, s[1])
	}
	sort.Strings(listed)
	if strings.Join(listed, " ") != strings.Join(want, " ") {
		t.Errorf("the chapter lists blocks %v; .abcd/config.json carries %v", listed, want)
	}
	byHand := map[string]bool{}
	for _, s := range tick.FindAllStringSubmatch(m[3], -1) {
		byHand[s[1]] = true
	}
	for _, k := range []string{"facilitator", "release", "status"} {
		if !byHand[k] {
			t.Errorf("the chapter does not name %s among the blocks this repository sets by hand", k)
		}
		if _, ok := cfg[k]; !ok {
			t.Errorf(".abcd/config.json carries no %s block", k)
		}
	}
}

// TestInternalsIndexDoesNotPromiseSeamOrRegistryConfig (iss-2610262038210994):
// the internals index row for the configuration chapter once said the `meta`
// setup block holds "seam-backend + adapter-registry config". The chapter
// itself lists the seam backend selectors as staged keys nothing reads and
// says there is no `adapters` registry block, so the row may promise neither.
func TestInternalsIndexDoesNotPromiseSeamOrRegistryConfig(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "05-internals", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "| 3 | [`03-configuration.md`]") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("the internals index has no row for 03-configuration.md")
	}
	if !strings.Contains(row, "`meta` setup block") {
		t.Errorf("the configuration row no longer names the `meta` setup block: %s", row)
	}
	for _, wrong := range []string{"seam-backend", "adapter-registry"} {
		if strings.Contains(row, wrong) {
			t.Errorf("the configuration row still says %q: the backend selectors are staged keys nothing reads and no adapters registry block exists", wrong)
		}
	}
}

// TestConfigurationChapterKeyCountMatchesSample (iss-2611060809294152): every
// place the configuration chapter counts the keys the binary reads ("<N> keys
// plus a `meta` block", "Beyond those <N>", "not one of the <N>") names the
// same number, and it is the number of keys outside `meta` in the chapter's
// own JSON sample of .abcd/config.json. The chapter once said twenty-two at
// one site while the others and the sample said twenty-three.
func TestConfigurationChapterKeyCountMatchesSample(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".abcd", "development", "brief", "05-internals", "03-configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	chapter := string(b)

	const heading = "### The keys the binary reads"
	i := strings.Index(chapter, heading)
	if i < 0 {
		t.Fatalf("the chapter has no %q section", heading)
	}
	rest := chapter[i:]
	start := strings.Index(rest, "```json\n")
	if start < 0 {
		t.Fatal("the keys section holds no JSON sample")
	}
	rest = rest[start+len("```json\n"):]
	end := strings.Index(rest, "\n```")
	if end < 0 {
		t.Fatal("the keys section's JSON sample is not closed")
	}
	comment := regexp.MustCompile(`(?m)\s*//.*$`)
	sample := comment.ReplaceAllString(rest[:end], "")
	var blocks map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(sample), &blocks); err != nil {
		t.Fatalf("the keys section's JSON sample does not parse once its comments are removed: %v", err)
	}
	if _, ok := blocks["meta"]; !ok {
		t.Fatal("the JSON sample carries no meta block")
	}
	keys := 0
	for name, block := range blocks {
		if name != "meta" {
			keys += len(block)
		}
	}

	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
		"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen",
		"eighteen", "nineteen", "twenty", "twenty-one", "twenty-two", "twenty-three", "twenty-four",
		"twenty-five", "twenty-six", "twenty-seven", "twenty-eight", "twenty-nine", "thirty"}
	if keys >= len(words) {
		t.Fatalf("the JSON sample holds %d keys outside meta, more than this test can spell", keys)
	}
	want := words[keys]

	text := strings.Join(strings.Fields(chapter), " ")
	for _, site := range []struct{ name, pattern string }{
		{"keys plus a `meta` block", "(?i)\\b([a-z-]+) keys plus a `meta` block"},
		{"Beyond those", "Beyond those ([a-z-]+),"},
		{"not one of the", "is not one of the ([a-z-]+)\\."},
	} {
		m := regexp.MustCompile(site.pattern).FindStringSubmatch(text)
		if m == nil {
			t.Errorf("the chapter's %q count was not found", site.name)
			continue
		}
		if got := strings.ToLower(m[1]); got != want {
			t.Errorf("the chapter's %q count says %q; its JSON sample holds %d keys outside meta (%s)", site.name, m[1], keys, want)
		}
	}
}
