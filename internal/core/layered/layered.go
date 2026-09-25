// Package layered is the one layered configuration resolver: a value is taken
// from the invocation's flag, else the repository's committed file, else the
// machine's file under ~/.abcd/, else the bundled default the caller supplies,
// and it comes back with the layer and the origin that supplied it.
//
// It is the canonical primitive for every configuration that more than one
// party may set (the one-canonical-primitive rule): the model-tier routing
// table (itd-2609170822093401, internal/core/oracle), the pace and the sub-agent
// ceiling (itd-2609201925079472, pace.*), the RepoPrompt review route (itd-6,
// oracle.review), the runner per role (itd-2609201916056194,
// roles.<role>.runner) and the duplicate-match threshold (itd-2609212137116617,
// match.threshold). A consumer reads through it rather than opening a file of
// its own, so the precedence order, the guarded reads and the refusals are
// spelled once.
//
// Two file families, one shape of resolution (DECISIONS, 2026-09-25):
//
//   - Config: .abcd/config.json in the checkout and ~/.abcd/config.json on the
//     machine, holding scalar keys under a namespace (pace.work_minutes,
//     oracle.review, roles.<role>.runner, match.threshold). The repository file
//     is shared with the keys ahoy writes (docs, meta, oracle.backend, repo,
//     attribution, rules), so no reader owns the whole file: a consumer CLAIMS
//     its namespace and every key in it, and an unknown key under a claimed
//     namespace is refused.
//   - OracleRouting: .abcd/config/oracle-routing.json and
//     ~/.abcd/oracle-routing.json, the per-agent routing table the model-tier
//     intent names. It carries a top-level schema_version, and its reader
//     claims the whole file.
//
// Loudness is the contract (the loud-staging rule applied to configuration).
// An absent file is an absent layer, the ordinary case. Every other fault is an
// error naming the file: a present file that is not a regular file, is a
// symlink or sits behind one, is writable by others (machine layer), is not
// valid JSON, holds a key twice, carries trailing content, or declares the
// wrong schema_version; an unknown key under a claimed namespace; a value in
// the winning layer that does not decode to the caller's type or fails its
// check. None of these falls through to a lower layer or to the default,
// because a configuration that silently does less than it says is the failure
// this package exists to close.
//
// The package reads and never writes, never reaches a network, and never
// prints: the front door renders its values and its errors.
package layered

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Layer is where a resolved value came from. The constants ascend in
// precedence: a higher layer holding a key beats every lower one.
type Layer int

const (
	// None: no layer holds the key and no bundled value applies. Only a
	// consumer with its own activation rule reports it (the routing table
	// applies nothing until a table is accepted).
	None Layer = iota
	// Bundled: the default the binary ships, supplied by the caller.
	Bundled
	// Machine: the file under ~/.abcd/.
	Machine
	// Repo: the file committed in the checkout the session resolved.
	Repo
	// Flag: the invocation's own override, for one run.
	Flag
)

// String is the layer's name as every surface renders it.
func (l Layer) String() string {
	switch l {
	case None:
		return "none"
	case Bundled:
		return "bundled"
	case Machine:
		return "machine"
	case Repo:
		return "repo"
	case Flag:
		return "flag"
	}
	return fmt.Sprintf("layer(%d)", int(l))
}

// File names one configuration file family: where it sits in a checkout and
// where it sits under ~/.abcd/.
type File struct {
	// RepoRel is the slash path relative to the checkout root.
	RepoRel string
	// MachineRel is the slash path relative to ~/.abcd/.
	MachineRel string
	// SchemaVersion, when non-zero, is the top-level schema_version every
	// layer's file must declare.
	SchemaVersion int
}

var (
	// Config is the shared scalar-key file (see the package doc).
	Config = File{RepoRel: ".abcd/config.json", MachineRel: "config.json"}
	// OracleRouting is the per-agent model-tier routing table.
	OracleRouting = File{RepoRel: ".abcd/config/oracle-routing.json", MachineRel: "oracle-routing.json", SchemaVersion: 1}
)

// RepoOrigin and MachineOrigin are how a layer's file is named in a value's
// origin and in every refusal: repo-relative, and in the tilde form so no
// message carries the caller's home path.
func (f File) RepoOrigin() string    { return f.RepoRel }
func (f File) MachineOrigin() string { return "~/.abcd/" + f.MachineRel }

// Roots are the two places the layers are read from. Repo is the directory the
// repository layer is read at; "" means there is none and the repo layer is
// absent. Home is the caller's home directory; "" is refused, because an
// unresolved home would drop the machine layer without a word. A front door
// builds Roots with RootsFor; a test builds them by hand.
type Roots struct {
	Repo string
	Home string
}

// RootsFor is the one way a front door resolves Roots for a working directory,
// so every consumer of a layered file (the bare board, the delegating verbs'
// --route, and the pace, runner, review-route and match-threshold readers that
// follow) reads the repository layer from the same place.
//
// The repository root is the rules loader's (rules.Resolve), not git's
// toplevel, and that is the choice for a reason: .abcd/rules.json,
// .abcd/guard.json and .abcd/config.json are already read from that root, so a
// session's injected rules, its shell guard and its layered configuration can
// never come from two different directories. The two answers differ in two
// places, and the rules root is the right one in both: a nested .abcd/ inside
// the working tree (a monorepo member governs its own subtree, as it already
// does for its rules), and a repository git will not answer for, where the
// rules root is bounded by the same shape check and ownership gate (a foreign-
// uid checkout falls back to the working directory with a note, instead of
// being read on git's say-so or dropped without one). Outside any repository it
// is the working directory, walked no higher, exactly as the rules loader reads
// it.
//
// notes are the resolution's refusals (a foreign-owned checkout declined), for
// the front door to print on stderr; the home is left "" when it cannot be
// resolved, which Load refuses loudly.
func RootsFor(cwd string) (Roots, []string) {
	res := rules.Resolve(cwd)
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return Roots{Repo: res.Root, Home: home}, res.Notes
}

// MaxFileBytes caps every layer's read. A configuration file is a few hundred
// bytes; 1 MiB bounds a planted device or an endless file without ever
// refusing a real one.
const MaxFileBytes = 1 << 20

// doc is one layer's parsed file. root is nil when the layer is absent.
type doc struct {
	layer  Layer
	origin string
	root   map[string]json.RawMessage
	// flagOrigins names, per full key, the flag text that set it (flag layer).
	flagOrigins map[string]string
}

// Stack is one file family's loaded layers.
type Stack struct {
	flag    doc
	repo    doc
	machine doc
}

// Load reads the repo and machine layers of f. The flag layer starts empty and
// is filled with SetFlag.
func Load(f File, r Roots) (*Stack, error) {
	if !f.valid() {
		return nil, fmt.Errorf("layered: file family %q / %q is not a pair of clean relative paths", f.RepoRel, f.MachineRel)
	}
	if r.Home == "" {
		return nil, fmt.Errorf("%s: the home directory is unresolved, so the machine layer "+
			"cannot be read; set HOME and retry", f.MachineOrigin())
	}
	s := &Stack{
		flag:    doc{layer: Flag, flagOrigins: map[string]string{}},
		repo:    doc{layer: Repo, origin: f.RepoOrigin()},
		machine: doc{layer: Machine, origin: f.MachineOrigin()},
	}
	if r.Repo != "" {
		raw, err := readRepo(r.Repo, f.RepoRel)
		if err != nil {
			return nil, fmt.Errorf("%s (repo layer): %w", f.RepoOrigin(), err)
		}
		if raw != nil {
			if s.repo.root, err = parse(raw, f); err != nil {
				return nil, fmt.Errorf("%s (repo layer): %w", f.RepoOrigin(), err)
			}
		}
	}
	raw, err := readMachine(r.Home, f.MachineRel)
	if err != nil {
		return nil, fmt.Errorf("%s (machine layer): %w", f.MachineOrigin(), err)
	}
	if raw != nil {
		if s.machine.root, err = parse(raw, f); err != nil {
			return nil, fmt.Errorf("%s (machine layer): %w", f.MachineOrigin(), err)
		}
	}
	return s, nil
}

// Present reports whether the layer's file exists (repo, machine) or any flag
// was set (flag).
func (s *Stack) Present(l Layer) bool {
	d := s.layer(l)
	return d != nil && d.root != nil
}

// readRepo reads rel inside an os.Root at the checkout, so a symlinked leaf or
// ancestor committed by a hostile repository cannot aim the read outside it.
// An absent file returns (nil, nil).
func readRepo(repoRoot, rel string) ([]byte, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("opening the checkout root: %w", err)
	}
	defer root.Close()
	raw, err := fsutil.ReadGuardedInRoot(root, rel, MaxFileBytes)
	switch {
	case err == nil:
		return raw, nil
	case errors.Is(err, os.ErrNotExist):
		return nil, nil
	case errors.Is(err, fsutil.ErrNotRegular):
		return nil, errors.New("it is not a regular file (a symlink, a directory or a device), so it is refused rather than followed")
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, fmt.Errorf("it is larger than %d bytes", MaxFileBytes)
	}
	return nil, fmt.Errorf("it could not be read inside the checkout: %w", err)
}

// readMachine reads ~/.abcd/<rel> through the home-declaration guard: a regular
// file, owned by the caller and writable by nobody else, because what it says
// decides which model a step reaches. An absent file returns (nil, nil).
func readMachine(home, rel string) ([]byte, error) {
	p := filepath.Join(home, ".abcd", filepath.FromSlash(rel))
	raw, refusal, err := fsutil.ReadDeclaration(p, MaxFileBytes)
	switch refusal {
	case fsutil.DeclarationOK:
		return raw, nil
	case fsutil.DeclarationAbsent:
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("it could not be examined: %s", fsutil.RedactHome(err.Error()))
	case fsutil.DeclarationNotRegular:
		return nil, errors.New("it is not a regular file (a symlink, a directory or a device)")
	case fsutil.DeclarationWritableByOthers:
		return nil, errors.New("it is writable by group or other, so its contents are not necessarily yours; chmod go-w it")
	case fsutil.DeclarationForeignOwner:
		return nil, errors.New("it is not owned by you")
	}
	if errors.Is(err, fsutil.ErrTooBig) {
		return nil, fmt.Errorf("it is larger than %d bytes", MaxFileBytes)
	}
	return nil, fmt.Errorf("it could not be read: %s", fsutil.RedactHome(err.Error()))
}

// parse decodes one layer's bytes: a single JSON object, no key twice at any
// depth, nothing after it, and the family's schema_version where it has one.
func parse(raw []byte, f File) (map[string]json.RawMessage, error) {
	if err := refuseDuplicateKeys(raw); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var root map[string]json.RawMessage
	if err := dec.Decode(&root); err != nil {
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			return nil, errors.New("it is not a JSON object")
		}
		return nil, fmt.Errorf("it is not valid JSON: %v", err)
	}
	if root == nil {
		return nil, errors.New("it is not a JSON object")
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("it carries trailing content after the JSON object")
	}
	if f.SchemaVersion != 0 {
		v, ok := root["schema_version"]
		var n int
		if !ok || json.Unmarshal(v, &n) != nil || n != f.SchemaVersion {
			shown := "absent"
			if ok {
				shown = string(v)
			}
			return nil, fmt.Errorf("schema_version is %s, want %d", shown, f.SchemaVersion)
		}
	}
	return root, nil
}

// refuseDuplicateKeys walks the token stream and refuses any object that names
// a key twice. Silent last-wins lets a block further down a reviewed file
// replace the one a reader saw first. Invalid JSON is left to the decoder,
// which reports it with a position.
func refuseDuplicateKeys(raw []byte) error {
	if !json.Valid(raw) {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	// Each frame is one open container: its key set when it is an object (nil
	// for an array), whether the next string token is a key, and its path.
	type frame struct {
		keys    map[string]bool
		wantKey bool
		path    string
		lastKey string
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil
		}
		var top *frame
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{', '[':
				p := ""
				if top != nil {
					p = top.path + "[]"
					if top.keys != nil {
						p = joinKey(top.path, top.lastKey)
					}
					if top.keys != nil {
						top.wantKey = true
					}
				}
				fr := &frame{path: p}
				if t == '{' {
					fr.keys = map[string]bool{}
					fr.wantKey = true
				}
				stack = append(stack, fr)
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1].keys != nil {
					stack[len(stack)-1].wantKey = true
				}
			}
		case string:
			if top != nil && top.keys != nil && top.wantKey {
				if top.keys[t] {
					return fmt.Errorf("it names %q more than once; the last would win silently, "+
						"so a block further down the file could replace the one a reader saw first",
						BoundKey(joinKey(top.path, t)))
				}
				top.keys[t] = true
				top.lastKey = t
				top.wantKey = false
				continue
			}
			if top != nil && top.keys != nil {
				top.wantKey = true
			}
		default:
			if top != nil && top.keys != nil {
				top.wantKey = true
			}
		}
	}
}

func joinKey(prefix, k string) string {
	if prefix == "" {
		return k
	}
	return prefix + "." + k
}

// splitKey validates a dotted key and returns its segments.
func splitKey(key string) ([]string, error) {
	if key == "" {
		return nil, errors.New("layered: empty key")
	}
	segs := strings.Split(key, ".")
	for _, s := range segs {
		if s == "" {
			return nil, fmt.Errorf("layered: key %q has an empty segment", key)
		}
	}
	return segs, nil
}

func (s *Stack) layer(l Layer) *doc {
	switch l {
	case Flag:
		return &s.flag
	case Repo:
		return &s.repo
	case Machine:
		return &s.machine
	}
	return nil
}

// SetFlag places v at key in the flag layer, for this invocation only. origin
// is the flag text as typed (for example "--pace 90/240"), which becomes the
// value's origin and is how a receipt names an override verbatim.
func (s *Stack) SetFlag(key string, v any, origin string) error {
	segs, err := splitKey(key)
	if err != nil {
		return err
	}
	enc, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("layered: flag %s: %w", origin, err)
	}
	if s.flag.root == nil {
		s.flag.root = map[string]json.RawMessage{}
	}
	if err := setPath(s.flag.root, segs, enc); err != nil {
		return fmt.Errorf("layered: flag %s: %w", origin, err)
	}
	s.flag.flagOrigins[key] = origin
	return nil
}

func setPath(m map[string]json.RawMessage, segs []string, v json.RawMessage) error {
	if len(segs) == 1 {
		m[segs[0]] = v
		return nil
	}
	child := map[string]json.RawMessage{}
	if raw, ok := m[segs[0]]; ok {
		if err := json.Unmarshal(raw, &child); err != nil {
			return fmt.Errorf("%s already holds a value that is not an object", segs[0])
		}
	}
	if err := setPath(child, segs[1:], v); err != nil {
		return err
	}
	enc, err := json.Marshal(child)
	if err != nil {
		return err
	}
	m[segs[0]] = enc
	return nil
}

// Found is one layer's raw value for a key.
type Found struct {
	Layer  Layer
	Origin string
	Raw    json.RawMessage
}

// Lookup returns every layer that holds key, highest precedence first, so the
// winner is the first entry and a board can render the rest beside it. A path
// that runs through a non-object value is an error naming the layer's file.
func (s *Stack) Lookup(key string) ([]Found, error) {
	segs, err := splitKey(key)
	if err != nil {
		return nil, err
	}
	var out []Found
	for _, d := range []*doc{&s.flag, &s.repo, &s.machine} {
		if d.root == nil {
			continue
		}
		raw, ok, err := walk(d.root, segs)
		if err != nil {
			return nil, fmt.Errorf("%s (%s layer): %s: %w", d.originFor(key), d.layer, key, err)
		}
		if ok {
			out = append(out, Found{Layer: d.layer, Origin: d.originFor(key), Raw: raw})
		}
	}
	return out, nil
}

func (d *doc) originFor(key string) string {
	if d.layer != Flag {
		return d.origin
	}
	// The flag that set key, or the longest one that set a key above or below
	// it, names the value: "--pace 90/240" for pace.work_minutes.
	best := ""
	for k := range d.flagOrigins {
		related := k == key || strings.HasPrefix(key, k+".") || strings.HasPrefix(k, key+".")
		if related && len(k) > len(best) {
			best = k
		}
	}
	if best == "" {
		return "flag"
	}
	return d.flagOrigins[best]
}

// walk follows segs from m. ok is false when a segment is absent.
func walk(m map[string]json.RawMessage, segs []string) (json.RawMessage, bool, error) {
	raw, ok := m[segs[0]]
	if !ok {
		return nil, false, nil
	}
	if len(segs) == 1 {
		return raw, true, nil
	}
	var child map[string]json.RawMessage
	if err := json.Unmarshal(raw, &child); err != nil || child == nil {
		return nil, false, fmt.Errorf("%s is %s, not an object, so it cannot hold %s",
			segs[0], compact(raw), strings.Join(segs[1:], "."))
	}
	return walk(child, segs[1:])
}

// Members returns the member names of the object at key in one layer, sorted.
// An absent layer or key returns nil; a non-object value is an error.
func (s *Stack) Members(l Layer, key string) ([]string, error) {
	d := s.layer(l)
	if d == nil || d.root == nil {
		return nil, nil
	}
	obj, ok, err := objectAt(d, key)
	if err != nil || !ok {
		return nil, err
	}
	names := make([]string, 0, len(obj))
	for k := range obj {
		names = append(names, k)
	}
	sort.Strings(names)
	return names, nil
}

// objectAt returns the object at key ("" for the file's top level).
func objectAt(d *doc, key string) (map[string]json.RawMessage, bool, error) {
	if key == "" {
		return d.root, true, nil
	}
	segs, err := splitKey(key)
	if err != nil {
		return nil, false, err
	}
	raw, ok, err := walk(d.root, segs)
	if err != nil {
		return nil, false, fmt.Errorf("%s (%s layer): %w", d.originFor(key), d.layer, err)
	}
	if !ok {
		return nil, false, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return nil, false, fmt.Errorf("%s (%s layer): %s is %s, not an object",
			d.originFor(key), d.layer, key, compact(raw))
	}
	return obj, true, nil
}

// Claim declares that the caller owns namespace and that keys are every key it
// may hold, in every layer. An unknown key is refused, naming the key, the file
// it sits in and the keys the namespace takes, so a misspelt key never lets a
// default apply unannounced. namespace "" claims the file's top level (a file
// one reader owns outright); a "*" segment claims every member at that level
// (roles.* claims each role's keys, whatever the role is called). A namespace
// no layer holds is not an error.
func (s *Stack) Claim(namespace string, keys ...string) error {
	allowed := map[string]bool{}
	for _, k := range keys {
		allowed[k] = true
	}
	for _, d := range []*doc{&s.flag, &s.repo, &s.machine} {
		if d.root == nil {
			continue
		}
		if err := claimIn(d, d.root, "", strings.Split(namespace, "."), namespace == "", allowed, keys); err != nil {
			return err
		}
	}
	return nil
}

func claimIn(d *doc, obj map[string]json.RawMessage, at string, rest []string, top bool, allowed map[string]bool, keys []string) error {
	if !top && len(rest) > 0 {
		seg := rest[0]
		var names []string
		if seg == "*" {
			for k := range obj {
				names = append(names, k)
			}
			sort.Strings(names)
		} else if _, ok := obj[seg]; ok {
			names = []string{seg}
		}
		for _, n := range names {
			p := joinKey(at, n)
			var child map[string]json.RawMessage
			if err := json.Unmarshal(obj[n], &child); err != nil || child == nil {
				return fmt.Errorf("%s (%s layer): %s is %s, not an object; it takes %s",
					d.originFor(p), d.layer, BoundKey(p), compact(obj[n]), strings.Join(keys, ", "))
			}
			if err := claimIn(d, child, p, rest[1:], len(rest) == 1, allowed, keys); err != nil {
				return err
			}
		}
		return nil
	}
	var unknown []string
	for k := range obj {
		if !allowed[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	// Named bounded, and at most maxEchoKeys of them: a hostile file can hold
	// tens of thousands of keys, each up to the file's size (review-tier1 F1).
	shown := unknown
	if len(shown) > maxEchoKeys {
		shown = shown[:maxEchoKeys]
	}
	full := make([]string, len(shown))
	for i, k := range shown {
		full[i] = BoundKey(joinKey(at, k))
	}
	list := strings.Join(full, ", ")
	if more := len(unknown) - len(shown); more > 0 {
		list += fmt.Sprintf(" and %d more", more)
	}
	scope := BoundKey(at)
	if scope == "" {
		scope = "the top level"
	}
	return fmt.Errorf("%s (%s layer): unknown key %s; nothing reads it, so it would change nothing. "+
		"%s takes: %s", d.originFor(joinKey(at, unknown[0])), d.layer, list, scope, strings.Join(keys, ", "))
}

// Value is a resolved configuration value with its provenance.
type Value[T any] struct {
	V      T
	Layer  Layer
	Origin string
}

// Get resolves key to a T: the highest layer holding it, else bundled. The
// winning layer's value must decode strictly into T (unknown fields refused,
// a fraction refused for an integer, null refused) and pass check, when check
// is non-nil; if it does not, Get refuses naming the key, the value and its
// origin, and never moves on to a lower layer or to the default.
func Get[T any](s *Stack, key string, bundled T, check func(T) error) (Value[T], error) {
	found, err := s.Lookup(key)
	if err != nil {
		return Value[T]{}, err
	}
	if len(found) == 0 {
		return Value[T]{V: bundled, Layer: Bundled, Origin: "bundled"}, nil
	}
	win := found[0]
	v, err := Decode[T](win.Raw)
	if err == nil && check != nil {
		err = check(v)
	}
	if err != nil {
		return Value[T]{}, fmt.Errorf("%s (%s layer): %s is %s: %v", win.Origin, win.Layer, key, compact(win.Raw), err)
	}
	return Value[T]{V: v, Layer: win.Layer, Origin: win.Origin}, nil
}

// Decode is Get's strict decode, exported so a consumer that selects among
// Lookup's layers itself (the routing table's activation rule) decodes with the
// same strictness.
func Decode[T any](raw json.RawMessage) (T, error) {
	var v T
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return v, errors.New("null is not a value")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("it does not decode as %T: %v", v, err)
	}
	return v, nil
}

// compact renders a raw value for a message, bounded so a hostile file cannot
// flood a refusal.
func compact(raw json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		b.Reset()
		b.Write(raw)
	}
	s := b.String()
	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}

// maxEcho is the most bytes of one key, name or value a message carries, and
// maxEchoKeys the most keys one message lists.
const (
	maxEcho     = 80
	maxEchoKeys = 5
)

// BoundKey renders a key or a name for a message, bounded like compact bounds a
// value: a file may be MaxFileBytes long, and a key it carries must not reach a
// refusal, a diagnostic or a stderr line whole (review-tier1 F1). It cuts on a
// rune boundary, so the result stays valid UTF-8.
func BoundKey(s string) string {
	if len(s) <= maxEcho {
		return s
	}
	cut := maxEcho - 3
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "..."
}

// valid reports whether a File's two paths are clean relative slash paths, the
// shape the contained repo read and the home join both need.
func (f File) valid() bool {
	return fsutil.ValidRelPath(f.RepoRel) && fsutil.ValidRelPath(f.MachineRel)
}
