// Package needs owns a repository's list of the outside tools it needs
// (.abcd/needs.json), the list's grammar, and the judge every copy of a
// program passes before abcd will register or start it (itd-2611041815429986,
// under the trust rule adr-2611041815445203).
//
// The list is data about what a repository needs, never about where a copy
// lives: it holds a tool's name, why it is needed and, for Go, which file
// declares its version, and no path, checksum or version of its own. Its
// grammar is strict so that nothing else can ride in on it: one JSON document,
// no repeated key, no field the grammar does not name, a name from a narrow
// alphabet that is never a shell, git, abcd, node or a command a runner
// starts, a reason that cannot carry a path to type, and a version file from
// a closed set for its tool.
//
// The judge (judge.go) is the runner's admit rule, extracted so that the
// runner calls it (Rule.Admit), with the further checks a person's
// registration is held to (Rule.Judge): the path's own name, every link on the
// way and what it points at, every directory above them, and their owners.
//
// This is a filter against mistakes, not a boundary (adr-42). It refuses what
// it reads, and nothing in it can stop a program running on the person's own
// account from editing a PATH, a home file or anything else that account can
// write. The package reads bytes it is handed and the file system's metadata;
// it never runs a program, never writes, and never reaches a network.
package needs

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
)

// ListPath is where a repository keeps its tool list, relative to its root.
const ListPath = ".abcd/needs.json"

// SchemaVersion is the one version of the list this build reads.
const SchemaVersion = 1

const (
	// MaxListBytes bounds the list, so a hostile one cannot hold a reader.
	MaxListBytes = 64 << 10
	// MaxReasonRunes bounds a reason: a sentence, not a document.
	MaxReasonRunes = 160
	// maxNameBytes bounds a name.
	maxNameBytes = 64
)

// List is a repository's tool list as read.
type List struct {
	SchemaVersion int    `json:"schema_version"`
	Tools         []Tool `json:"tools"`
}

// Tool is one entry: a name, why the repository needs it, and the file that
// declares its version, "" for a tool abcd checks no version of.
type Tool struct {
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	VersionFile string `json:"version_file,omitempty"`
}

// ParseList reads a tool list from its bytes, refusing anything outside the
// grammar. The error names where the list is wrong and never repeats a value
// it refused, since that value is a repository's text.
func ParseList(data []byte) (List, error) {
	var l List
	if len(data) > MaxListBytes {
		return List{}, fmt.Errorf("%s is %d bytes, over the %d a tool list may hold", ListPath, len(data), MaxListBytes)
	}
	// An absent schema_version reads as zero and is refused below.
	if err := jsonstrict.Decode(data, &l); err != nil {
		return List{}, fmt.Errorf("%s is not a tool list: %w", ListPath, err)
	}
	if l.SchemaVersion != SchemaVersion {
		return List{}, fmt.Errorf("%s declares schema_version %d; this build reads %d", ListPath, l.SchemaVersion, SchemaVersion)
	}
	seen := map[string]bool{}
	for i, tool := range l.Tools {
		at := fmt.Sprintf("%s: tools[%d]", ListPath, i)
		if err := CheckName(tool.Name); err != nil {
			return List{}, fmt.Errorf("%s: %w", at, err)
		}
		if seen[tool.Name] {
			return List{}, fmt.Errorf("%s: %s is listed twice", at, tool.Name)
		}
		seen[tool.Name] = true
		if err := CheckReason(tool.Reason); err != nil {
			return List{}, fmt.Errorf("%s (%s): %w", at, tool.Name, err)
		}
		if err := CheckVersionFile(tool.Name, tool.VersionFile); err != nil {
			return List{}, fmt.Errorf("%s (%s): %w", at, tool.Name, err)
		}
	}
	return l, nil
}

// nameRe is a tool's name: a lowercase program name that starts with a letter
// or digit, so it is never an option, a hidden file, "." or "..", and holds no
// path separator, space, quote or shell character.
var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]*$`)

// reserved are the names no list may hold: the shells, git, abcd, node (the
// program a harness runs on) and the command each runner starts. Listing one
// would move a registered copy ahead on PATH of the one a harness or abcd
// itself runs on. go and make are absent on purpose: abcd runs them, but a
// repository may need them. The runner package holds its own commands to this
// set in a test, since this package cannot import it.
var reserved = []string{
	// shells
	"ash", "bash", "busybox", "csh", "dash", "elvish", "fish", "ksh", "mksh", "nu", "oksh",
	"pdksh", "powershell", "pwsh", "rc", "sh", "tcsh", "xonsh", "yash", "zsh",
	// abcd and what it is built on
	"abcd", "git",
	// what a harness runs on, and the commands runners start
	"node", "claude", "opencode",
}

// Reserved reports whether name is one no tool list may hold.
func Reserved(name string) bool {
	return slices.Contains(reserved, name)
}

// ErrReserved is CheckName's answer for a name in the reserved set.
var ErrReserved = errors.New("reserved")

// CheckName refuses a name outside the grammar or in the reserved set. The
// error repeats the name only once it has passed the grammar.
func CheckName(name string) error {
	switch {
	case name == "":
		return errors.New("a tool needs a name")
	case len(name) > maxNameBytes:
		return fmt.Errorf("a tool's name is at most %d characters", maxNameBytes)
	case !nameRe.MatchString(name):
		return errors.New("a tool's name is a lowercase program name of letters, digits and . _ + -, starting with a letter or digit")
	case Reserved(name):
		return fmt.Errorf("%s is %w: a shell, git, abcd, node and the command each runner starts are never listed", name, ErrReserved)
	}
	return nil
}

// versionFiles is each tool's closed set of version files, at the
// repository's root. A tool absent here has no version check, so its entry
// names no file.
var versionFiles = map[string][]string{
	"go": {"go.mod", "go.work"},
}

// VersionFiles is the closed set of files that may declare name's version, a
// copy the caller may change; nil for a tool abcd checks no version of.
func VersionFiles(name string) []string {
	return slices.Clone(versionFiles[name])
}

// CheckVersionFile refuses a version file outside name's closed set. A tool
// with a set must name one, since its version is what abcd checks it by; a
// tool without one must name none.
func CheckVersionFile(name, file string) error {
	set := versionFiles[name]
	switch {
	case len(set) == 0 && file == "":
		return nil
	case len(set) == 0:
		return fmt.Errorf("abcd checks no version of %s, so its entry names no version file", name)
	case slices.Contains(set, file):
		return nil
	}
	return fmt.Errorf("%s's version file is one of %s, at the repository's root", name, strings.Join(set, ", "))
}

// reasonPunct is the punctuation a reason may hold beside letters, digits and
// the space: enough to write a sentence, and no path separator, ~, $,
// backtick or quote.
const reasonPunct = ".,;:!?()[]-_+=%&#@*"

// CheckReason refuses a reason that is empty, over MaxReasonRunes, or holds a
// character outside its allowlist: letters and digits of any script, the
// space, and reasonPunct. So a reason can describe a place in words but never
// carry a path to type, and it is shown as data. The error never repeats it.
func CheckReason(reason string) error {
	if reason == "" {
		return errors.New("a tool needs a reason: why the repository needs it")
	}
	if !utf8.ValidString(reason) {
		return errors.New("a reason is UTF-8 text")
	}
	if n := utf8.RuneCountInString(reason); n > MaxReasonRunes {
		return fmt.Errorf("a reason is at most %d characters, and this one is %d", MaxReasonRunes, n)
	}
	for i, r := range reason {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || strings.ContainsRune(reasonPunct, r) {
			continue
		}
		return fmt.Errorf("a reason holds letters, digits, spaces and %s only, and character %d is none of them",
			reasonPunct, utf8.RuneCountInString(reason[:i])+1)
	}
	return nil
}
