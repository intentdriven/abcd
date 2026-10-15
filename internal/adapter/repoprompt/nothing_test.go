package repoprompt

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestTheAdapterStartsNothingAndReadsNothing is scope 4 held by structure:
// the adapter's own files import no process, file, environment or
// system-call API, so it cannot start RepoPrompt or any process, read or
// write RepoPrompt's configuration, or look for a server on the machine. The
// one thing it can do is connect to the address its caller hands it.
func TestTheAdapterStartsNothingAndReadsNothing(t *testing.T) {
	forbidden := map[string]string{
		"os":                    "files, the environment and processes",
		"os/exec":               "processes",
		"os/user":               "the machine's accounts",
		"syscall":               "system calls",
		"io/fs":                 "files",
		"io/ioutil":             "files",
		"path/filepath":         "the machine's paths",
		"plugin":                "loaded code",
		"net/http/cgi":          "processes",
		"golang.org/x/sys/unix": "system calls",
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	read := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if what, ok := forbidden[p]; ok {
				t.Fatalf("%s imports %s, which reaches %s: the adapter connects to the address it is handed and nothing else", name, p, what)
			}
		}
		read++
	}
	if read == 0 {
		t.Fatal("no file of the adapter was read")
	}
}
