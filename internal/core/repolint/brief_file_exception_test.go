package repolint

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// A site-gates finding with no source span has no File, and File is omitempty,
// so its JSON carries no `file` key. commands/lint.md says so; the brief's lint
// chapter described the same JSON and said each finding carries a `file`
// (iss-2611040554385570). While the field can be omitted, every page that
// lists a finding's fields states the exception.
func TestLintPagesStateTheFileException(t *testing.T) {
	field, ok := reflect.TypeOf(Finding{}).FieldByName("File")
	if !ok {
		t.Fatal("Finding has no File field")
	}
	if !strings.Contains(field.Tag.Get("json"), "omitempty") {
		t.Skip("Finding.File is no longer omitempty: the pages need no exception")
	}
	root := filepath.Join("..", "..", "..")
	// The parenthesis after the `file` field must say the key is omitted and
	// name site-gates.
	exception := regexp.MustCompile("a `file` [(]omitted[^)]*`site-gates`[^)]*[)]")
	for _, page := range []string{
		"commands/lint.md",
		".abcd/development/brief/04-surfaces/16-lint.md",
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
		if err != nil {
			t.Fatalf("read %s: %v", page, err)
		}
		text := strings.Join(strings.Fields(string(raw)), " ")
		if !exception.MatchString(text) {
			t.Errorf("%s lists a finding's `file` without the exception: a `site-gates` finding that names no source span omits it", page)
		}
	}
}
