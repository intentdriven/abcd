package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// topLevelJSONFields is every key t can put at the top of its JSON object:
// its own fields' names and those of every struct it embeds without a name,
// whose fields encoding/json lifts into the same object.
func topLevelJSONFields(t reflect.Type) []string {
	var names []string
	for i := range t.NumField() {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			names = append(names, topLevelJSONFields(f.Type)...)
			continue
		}
		if !f.IsExported() || name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		names = append(names, name)
	}
	return names
}

// pageSpan is the text of commands/page from the first line holding from to
// the end of the section headed through: the next heading of its level, or
// the end of the page.
func pageSpan(t *testing.T, page, from, through string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", page))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, from)
	if start < 0 {
		t.Fatalf("commands/%s has no %q", page, from)
	}
	text = text[start:]
	head := strings.Index(text, through+"\n")
	if head < 0 {
		t.Fatalf("commands/%s has no %q section after %q", page, through, from)
	}
	if end := strings.Index(text[head+len(through):], "\n## "); end >= 0 {
		text = text[:head+len(through)+end]
	}
	return text
}

// TestTheStatusAndPeersPagesNameEveryTopLevelField is iss-2611272325440853:
// the reservations block `abcd status` and `abcd peers` share gained fields
// (release_withdrawals, lost_withdrawals, ceiling, drain_in_work) that the
// status page's --json list never named. Every key either document can carry
// at its top level is named, in backticks, where its page lists the payload.
func TestTheStatusAndPeersPagesNameEveryTopLevelField(t *testing.T) {
	for _, c := range []struct {
		page, from, through string
		doc                 reflect.Type
	}{
		{"status.md", "## Read the status", "## Read the status", reflect.TypeOf(statusOutput{})},
		{"peers.md", "peers --json", "## The reservations", reflect.TypeOf(peersOutput{})},
	} {
		text := pageSpan(t, c.page, c.from, c.through)
		for _, name := range topLevelJSONFields(c.doc) {
			if !strings.Contains(text, "`"+name+"`") {
				t.Errorf("commands/%s does not name the top-level field `%s` that its --json document carries", c.page, name)
			}
		}
	}
}
