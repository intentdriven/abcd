package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestRepositoryGatesAdmitTheReleasePage is the wiring detector for the page's
// two homes. A cut writes RELEASE.md at the root and moves the outgoing page to
// the archive, so the repository's own gates must admit both: the root page is
// not a stray doc, CI treats it as prose, the archive directory says what it
// holds, and the archive is exempt from the one record-lint rule that exempts
// shipped intents (its prose is carried from them).
func TestRepositoryGatesAdmitTheReleasePage(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := fsutil.ModuleRoot(cwd)
	if err != nil {
		t.Fatal(err)
	}
	read := func(rel string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		return data
	}

	var docs struct {
		Rules struct {
			StrayRootDocs struct {
				Allowlist []string `json:"allowlist"`
			} `json:"stray_root_docs"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(read(".abcd/docs-lint.json"), &docs); err != nil {
		t.Fatalf("docs-lint.json: %v", err)
	}
	if !listHas(docs.Rules.StrayRootDocs.Allowlist, strings.TrimSuffix(PageFile, ".md")) {
		t.Errorf("stray_root_docs does not allow %s: %v", PageFile, docs.Rules.StrayRootDocs.Allowlist)
	}

	var records struct {
		Rules struct {
			ForbiddenSynonyms struct {
				ExemptPrefixes []string `json:"exempt_prefixes"`
			} `json:"forbidden_synonyms"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(read(".abcd/record-lint.json"), &records); err != nil {
		t.Fatalf("record-lint.json: %v", err)
	}
	if !listHas(records.Rules.ForbiddenSynonyms.ExemptPrefixes, ArchiveDir+"/") {
		t.Errorf("forbidden_synonyms does not exempt %s/: %v", ArchiveDir, records.Rules.ForbiddenSynonyms.ExemptPrefixes)
	}

	inert := regexp.MustCompile(`(?m)^\s*README\.md\|CHANGELOG\.md\|[^)]*\)`).Find(read(".github/workflows/ci.yml"))
	if !strings.Contains(string(inert), "|"+PageFile+"|") && !strings.Contains(string(inert), "|"+PageFile+")") {
		t.Errorf("ci.yml's inert root list does not name %s: %s", PageFile, inert)
	}

	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ArchiveDir), "README.md")); err != nil {
		t.Errorf("the archive directory has no README.md: %v", err)
	}
}

func listHas(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
