package launch

// artefact.go — the artefact declaration (itd-2609150819432059, decision 1).
//
// A repository abcd manages says once what it ships, in
// .abcd/config/artefact.json, and the launch verbs follow the declaration: the
// bundle the preview scans, the lockstep check, the plugin-only rows and the
// scaffold's file set are chosen by the declared kind. This file is the one
// reader of that declaration. Every launch verb and ahoy go through it, so a
// kind one of them accepts is a kind all of them accept.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ArtefactRelPath is the declaration's home, repo-relative.
const ArtefactRelPath = ".abcd/config/artefact.json"

// maxArtefactBytes caps the guarded read: the declaration is a few lines, and a
// larger file is not one anybody wrote by hand.
const maxArtefactBytes = 64 << 10

// ArtefactKind is what a repository ships.
type ArtefactKind string

// The kinds the first cut accepts. Any other value is refused by name.
const (
	// KindPlugin is the shipped shape: a harness plugin whose payload, manifests
	// and marketplace listing the launch gates already judge.
	KindPlugin ArtefactKind = "plugin"
	// KindBinary is a built program, a Go binary in the first cut.
	KindBinary ArtefactKind = "binary"
	// KindApplication is an application with its own build and publish steps.
	KindApplication ArtefactKind = "application"
)

// ArtefactKinds is the accepted set, in the order a refusal names it.
var ArtefactKinds = []ArtefactKind{KindPlugin, KindBinary, KindApplication}

// artefactKindList renders the accepted set for a refusal.
func artefactKindList() string {
	names := make([]string, len(ArtefactKinds))
	for i, k := range ArtefactKinds {
		names[i] = string(k)
	}
	return strings.Join(names, ", ")
}

// ValidArtefactKind reports whether k is one of the accepted kinds.
func ValidArtefactKind(k string) bool {
	for _, known := range ArtefactKinds {
		if string(known) == k {
			return true
		}
	}
	return false
}

// LockstepFile is one file held in lockstep with the version-location primary.
// Pointer is the RFC-6901 pointer to the version inside it; empty means the
// primary's own pointer.
type LockstepFile struct {
	Path    string `json:"path"`
	Pointer string `json:"json_pointer,omitempty"`
}

// Artefact is a validated declaration.
type Artefact struct {
	Kind ArtefactKind `json:"kind"`
	// Lockstep is the declared secondaries a non-plugin kind holds in lockstep
	// with its primary (decision 2). A plugin keeps the pinned manifest table.
	Lockstep []LockstepFile `json:"lockstep,omitempty"`
	// Site is the release-rendered site opt-in (decision 9). It is read and
	// validated here and acted on by itd-2609061543533170, not by this intent.
	Site bool `json:"site,omitempty"`
}

// IsPlugin reports whether the declared kind is the plugin shape.
func (a Artefact) IsPlugin() bool { return a.Kind == KindPlugin }

// ErrNoArtefact reports a repository that has not declared its artefact kind.
// It is carried inside a PreflightError whose message names the declaration's
// home and the accepted kinds, so a caller that recognises it can say more and
// a caller that does not still relays an actionable refusal.
var ErrNoArtefact = errors.New("this repository declares no artefact kind")

// noArtefactMessage is the absent declaration's refusal.
func noArtefactMessage() string {
	return "this repository declares no artefact kind: " + ArtefactRelPath + " is where it is declared, " +
		`as {"kind": "<kind>"} with the kind one of ` + artefactKindList() +
		"; `abcd ahoy install` writes it (a repository carrying a plugin manifest adopts kind plugin without being asked)"
}

// artefactKeys are the keys the declaration admits. An unknown key is refused
// rather than ignored: a misspelt "lockstep" would otherwise switch the check
// off without a word.
var artefactKeys = map[string]struct{}{"kind": {}, "lockstep": {}, "site": {}}

// LoadArtefact reads and validates the repository's artefact declaration. An
// absent file is a PreflightError wrapping ErrNoArtefact; every other fault —
// an unreadable or malformed file, an unknown kind, a lockstep path that is not
// a contained repo-relative path — is a PreflightError naming what is wrong.
// Whether a declared lockstep file can be READ is the lockstep check's to say,
// where the refusal names the path.
func LoadArtefact(repoRoot string) (Artefact, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(ArtefactRelPath)), maxArtefactBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return Artefact{}, &PreflightError{msg: noArtefactMessage(), err: ErrNoArtefact}
		}
		return Artefact{}, preflight("the artefact declaration %s is unreadable: %v", ArtefactRelPath, pathFreeError(err))
	}
	return ParseArtefact(data)
}

// ParseArtefact validates a declaration's bytes. It is exported for the writer
// in ahoy, which proves what it is about to write reads back.
func ParseArtefact(data []byte) (Artefact, error) {
	var raw map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&raw); err != nil || raw == nil || dec.More() {
		return Artefact{}, preflight("the artefact declaration %s is not a JSON object", ArtefactRelPath)
	}
	for key := range raw {
		if _, ok := artefactKeys[key]; !ok {
			return Artefact{}, preflight("the artefact declaration %s carries an unknown key %q (it admits kind, lockstep and site)",
				ArtefactRelPath, key)
		}
	}

	var art Artefact
	var kind string
	if err := json.Unmarshal(raw["kind"], &kind); err != nil || kind == "" {
		return Artefact{}, preflight("the artefact declaration %s declares no kind: it must name one of %s",
			ArtefactRelPath, artefactKindList())
	}
	if !ValidArtefactKind(kind) {
		return Artefact{}, preflight("the artefact declaration %s names the kind %q, which abcd does not know: the accepted kinds are %s",
			ArtefactRelPath, kind, artefactKindList())
	}
	art.Kind = ArtefactKind(kind)

	if v, ok := raw["site"]; ok {
		if err := json.Unmarshal(v, &art.Site); err != nil {
			return Artefact{}, preflight("the artefact declaration %s: site is not a boolean", ArtefactRelPath)
		}
	}

	if v, ok := raw["lockstep"]; ok {
		files, err := parseLockstep(v)
		if err != nil {
			return Artefact{}, err
		}
		if len(files) > 0 && art.IsPlugin() {
			return Artefact{}, preflight("the artefact declaration %s declares a lockstep list for kind plugin: "+
				"a plugin's lockstep is the pinned manifest table (adr-20), so the list belongs to the other kinds", ArtefactRelPath)
		}
		art.Lockstep = files
	}
	return art, nil
}

// parseLockstep validates the lockstep list: each entry is a repo-relative path,
// or an object naming the path and the pointer to the version inside it.
func parseLockstep(v json.RawMessage) ([]LockstepFile, error) {
	var entries []json.RawMessage
	if err := json.Unmarshal(v, &entries); err != nil {
		return nil, preflight("the artefact declaration %s: lockstep is not a list", ArtefactRelPath)
	}
	var files []LockstepFile
	seen := map[string]struct{}{}
	for _, e := range entries {
		var f LockstepFile
		var p string
		obj := json.NewDecoder(bytes.NewReader(e))
		obj.DisallowUnknownFields()
		if err := json.Unmarshal(e, &p); err == nil {
			f.Path = p
		} else if err := obj.Decode(&f); err != nil {
			return nil, preflight("the artefact declaration %s: a lockstep entry is neither a path nor {\"path\", \"json_pointer\"}", ArtefactRelPath)
		}
		// The path is committed configuration data joined onto the repository
		// root, so it is held to the containment the version-location
		// manifest_path is (gh-488), and it may not name the record namespace,
		// which never ships and so can never carry a released version.
		if !fsutil.ValidRelPath(f.Path) || pathContainsDeniedSegment(f.Path) {
			return nil, preflight("the artefact declaration %s: the lockstep path %q is not a contained repo-relative path outside the record namespace",
				ArtefactRelPath, f.Path)
		}
		if f.Pointer != "" && !strings.HasPrefix(f.Pointer, "/") {
			return nil, preflight("the artefact declaration %s: the lockstep json_pointer %q for %s is not an RFC-6901 pointer",
				ArtefactRelPath, f.Pointer, f.Path)
		}
		if _, dup := seen[f.Path]; dup {
			return nil, preflight("the artefact declaration %s names the lockstep path %s twice", ArtefactRelPath, f.Path)
		}
		seen[f.Path] = struct{}{}
		files = append(files, f)
	}
	return files, nil
}

// MarshalArtefact renders a declaration the way ahoy writes it: indented, with
// an empty lockstep list and the site opt-in spelled out, so the file shows the
// keys it admits.
func MarshalArtefact(art Artefact) []byte {
	files := art.Lockstep
	if files == nil {
		files = []LockstepFile{}
	}
	doc := struct {
		Kind     ArtefactKind   `json:"kind"`
		Lockstep []LockstepFile `json:"lockstep"`
		Site     bool           `json:"site"`
	}{art.Kind, files, art.Site}
	out, _ := json.MarshalIndent(doc, "", "  ")
	return append(out, '\n')
}
