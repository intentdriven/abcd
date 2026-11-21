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
