package lifeboat

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The platform page once said past disembarks are recorded as "manifests (hash
// + file list + label)" (iss-2610262030288703), while the embark chapter's
// voyage layout, and voyageEntry, carry no label and record the file count, not
// the list. The page is held here to the fields the ledger line actually has.
func TestBriefPlatformPageNamesTheVoyageEntryFields(t *testing.T) {
	var tags []string
	ty := reflect.TypeOf(voyageEntry{})
	for i := 0; i < ty.NumField(); i++ {
		tags = append(tags, strings.Split(ty.Field(i).Tag.Get("json"), ",")[0])
	}
	for _, absent := range []string{"label", "file_list", "files_list"} {
		for _, tag := range tags {
			if tag == absent {
				t.Fatalf("voyageEntry carries %q; this test and the brief assume it does not", absent)
			}
		}
	}

	const rel = ".abcd/development/brief/02-constraints/01-platform.md"
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	body := strings.Join(strings.Fields(string(raw)), " ")
	if strings.Contains(body, "file list + label") {
		t.Errorf("%s still says past disembarks are recorded with a file list and a label; the voyage entry carries neither", rel)
	}
	for _, want := range []string{
		"minimal voyage entries",
		"manifest hash, source name and root SHA, destination, file and byte counts",
		"no label and no file list",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s does not say %q about how past disembarks are recorded", rel, want)
		}
	}
}
