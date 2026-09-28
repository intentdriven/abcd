package credential

// external.go is the external home: a pointer at a setup outside abcd, which
// abcd follows on every Resolve and never copies. A pointer names either an
// environment variable, or a field of a tool's JSON configuration file under
// the home directory, written in the tilde form (~/.config/tool/auth.json)
// with a dotted field path (auth.token).
//
// The file is read under the declaration guards (a regular file, never a
// symlink, owned by the caller, writable by nobody else, bounded), and a file
// inside a git working tree is refused: a secret there is one a commit can
// carry, and the store never points at one.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// fileRe is a pointer's file in the tilde form: plain segments, no dot-dot.
var fileRe = regexp.MustCompile(`^~/[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)

// fieldRe is a pointer's dotted field path.
var fieldRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}(\.[A-Za-z0-9_-]{1,64}){0,7}$`)

// maxToolFileBytes bounds a tool's configuration file read.
const maxToolFileBytes = 256 << 10

// checkPointer refuses a malformed pointer: exactly one of an environment
// variable or a file with its field.
func checkPointer(p Pointer) error {
	switch {
	case p.Env != "" && (p.File != "" || p.Field != ""):
		return errors.New("credential: an external pointer names an environment variable or a file, not both")
	case p.Env != "":
		if !envNameRe.MatchString(p.Env) {
			return errors.New("credential: the environment variable's name is not letters, digits and '_'")
		}
	case p.File != "" || p.Field != "":
		if !fileRe.MatchString(p.File) || strings.Contains(p.File, "/../") || strings.HasSuffix(p.File, "/..") || strings.Contains(p.File, "/./") {
			return errors.New("credential: the file is not a plain path under the home directory, written from ~/")
		}
		if !fieldRe.MatchString(p.Field) {
			return errors.New("credential: the file's field is not a dotted path of plain keys (auth.token)")
		}
	default:
		return errors.New("credential: the external home needs a pointer: an environment variable, or a file and its field")
	}
	return nil
}

// resolvePointer follows p for name. What it points at being empty or absent
// is ErrNotSet, naming what was followed; anything unsafe is refused.
func resolvePointer(home, name string, p Pointer) (string, error) {
	if err := checkPointer(p); err != nil {
		return "", err
	}
	if p.Env != "" {
		v := os.Getenv(p.Env)
		if v == "" {
			return "", notSetError{name: name, why: "the environment variable " + p.Env + " it points at is not set"}
		}
		if err := CheckValue(v); err != nil {
			return "", fmt.Errorf("credential: %s, through the environment variable %s: %w", name, p.Env, err)
		}
		return v, nil
	}
	// A directory on the way to the file that is a symlink (~/.config linked
	// into a dotfiles repository, say) is refused first, naming the link: the
	// working-tree check below judges the lexical path and cannot see a
	// repository the link leads into. The read then goes through the
	// descriptor walk of the directories judged (fsutil.ReadHomeDeclaration),
	// never the path again.
	rel := strings.TrimPrefix(p.File, "~/")
	if err := fsutil.HomeScopeLink(home, rel); err != nil {
		return "", fmt.Errorf("credential: %s points at %s, which is not read: %v", name, p.File, err)
	}
	path := filepath.Join(home, filepath.FromSlash(rel))
	if tree := workingTreeAbove(filepath.Dir(path)); tree != "" {
		return "", fmt.Errorf("credential: %s points at %s, which lies inside a git working tree, where a commit could carry it, so it is not read", name, p.File)
	}
	raw, refusal, err := fsutil.ReadHomeDeclaration(home, rel, maxToolFileBytes)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
		return "", notSetError{name: name, why: "the file " + p.File + " it points at does not exist"}
	case refusal == fsutil.DeclarationBehindSymlink:
		return "", fmt.Errorf("credential: %s points at %s, which is not read: %v", name, p.File, err)
	case refusal == fsutil.DeclarationNotRegular:
		return "", fmt.Errorf("credential: %s points at %s, which is not a regular file (a symlink is never followed), so it is not read", name, p.File)
	case refusal == fsutil.DeclarationWritableByOthers:
		return "", fmt.Errorf("credential: %s points at %s, which group or other can write, so it is not read", name, p.File)
	case refusal == fsutil.DeclarationForeignOwner:
		return "", fmt.Errorf("credential: %s points at %s, which is not owned by you, so it is not read", name, p.File)
	case err != nil:
		return "", fmt.Errorf("credential: %s points at %s, which could not be read safely, so it is not read", name, p.File)
	}
	if err := jsonstrict.NoDuplicateKeys(raw); err != nil {
		return "", fmt.Errorf("credential: %s points at %s, which names one key twice, so it is not read", name, p.File)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		// The decoder's message can quote the file's bytes; it is dropped.
		return "", fmt.Errorf("credential: %s points at %s, which is not JSON, so it is not read", name, p.File)
	}
	for _, key := range strings.Split(p.Field, ".") {
		obj, ok := doc.(map[string]any)
		if !ok {
			return "", notSetError{name: name, why: "the field " + p.Field + " of " + p.File + " is absent"}
		}
		if doc, ok = obj[key]; !ok {
			return "", notSetError{name: name, why: "the field " + p.Field + " of " + p.File + " is absent"}
		}
	}
	v, ok := doc.(string)
	switch {
	case !ok:
		return "", fmt.Errorf("credential: %s points at the field %s of %s, which is not a string, so it is not read", name, p.Field, p.File)
	case v == "":
		return "", notSetError{name: name, why: "the field " + p.Field + " of " + p.File + " is empty"}
	}
	if err := CheckValue(v); err != nil {
		return "", fmt.Errorf("credential: %s, through the field %s of %s: %w", name, p.Field, p.File, err)
	}
	return v, nil
}
