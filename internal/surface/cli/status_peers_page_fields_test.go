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

// listLeads is the lead of every top-level list item in text: the item's
// text, its continuation lines joined, up to its first colon outside a code
// span and outside parentheses, or the whole item when it has none. A field
// named there is one the item is about, never a sub-field or a value the
// item lists after its colon, nor one a nested item names.
func listLeads(text string) []string {
	var items []string
	open := false
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "- "):
			items = append(items, strings.TrimPrefix(line, "- "))
			open = true
		case open && strings.HasPrefix(line, "  ") && !strings.HasPrefix(strings.TrimLeft(line, " "), "- "):
			items[len(items)-1] += " " + strings.TrimSpace(line)
		default:
			open = false
		}
	}
	leads := make([]string, 0, len(items))
	for _, item := range items {
		code, depth, end := false, 0, len(item)
	scan:
		for j, r := range item {
			switch {
			case r == '`':
				code = !code
			case code:
			case r == '(':
				depth++
			case r == ')' && depth > 0:
				depth--
			case r == ':' && depth == 0:
				end = j
				break scan
			}
		}
		leads = append(leads, item[:end])
	}
	return leads
}

// TestTheStatusAndPeersPagesNameEveryTopLevelField is iss-2611272325440853:
// the reservations block `abcd status` and `abcd peers` share gained fields
// (release_withdrawals, lost_withdrawals, ceiling, drain_in_work) that the
// status page's --json list never named. Every key either document can carry
// at its top level leads a top-level item of the list its page gives the
// payload in, so a field named only as another's sub-field or value is not
// counted as named.
func TestTheStatusAndPeersPagesNameEveryTopLevelField(t *testing.T) {
	for _, c := range []struct {
		page, from, through string
		doc                 reflect.Type
		// never is each field the struct can hold that this verb never
		// fills, with why, so the page is not held to list it as carried.
		never map[string]string
	}{
		{"status.md", "## Read the status", "## Read the status", reflect.TypeOf(statusOutput{}), map[string]string{
			"settled": "only `abcd peers` settles ended releases (addReservations); `abcd status` fills the shared block through fillView, which leaves it empty, and it is omitempty",
		}},
		{"peers.md", "peers --json", "## The reservations", reflect.TypeOf(peersOutput{}), nil},
	} {
		leads := strings.Join(listLeads(pageSpan(t, c.page, c.from, c.through)), "\n")
		for _, name := range topLevelJSONFields(c.doc) {
			if why, ok := c.never[name]; ok {
				t.Logf("commands/%s: `%s` is not required: %s", c.page, name, why)
				continue
			}
			if !strings.Contains(leads, "`"+name+"`") {
				t.Errorf("commands/%s names the top-level field `%s` its --json document carries at the head of no top-level item of its payload list", c.page, name)
			}
		}
	}
}
