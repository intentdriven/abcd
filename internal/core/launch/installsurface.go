package launch

// installsurface.go — the single resolver of "what does this payload DECLARE?".
//
// # Why this is not the compatibility surface
//
// internal/core/surface already answers a different question: which cobra
// commands and which manifest KEYS exist, so a later release can be told it
// removed one. That is the compatibility surface — it records key paths and
// deliberately discards VALUES, because a changed description must not read as a
// break. The installability question is the mirror image: it cares only about
// the values (the paths a manifest points at) and not at all about which keys
// carry them. Sharing one walker would force the compatibility snapshot to start
// recording values, which would report every prose edit as a surface change —
// the exact failure its doc comment rules out. So this is a separate resolver
// over the same two manifests, and the justification is that the two answer
// disjoint questions about disjoint halves of the same JSON.
//
// It is likewise not internal/core/ahoy's plugin-root check, which asks whether
// the plugin INSTALLED ON THIS MACHINE is healthy. This asks whether the payload
// about to be published COULD install anywhere.
//
// # What "the declared surface" is
//
// The union of two registers, because the plugin-manifest schema defines every
// explicit key as declaring entries "in addition to those in the <kind>/
// directory, if it exists":
//
//   - CONVENTION — the auto-discovery roots a harness loads with no manifest
//     help at all: commands/**/*.md (nested directories namespace the command),
//     agents/*.md (a flat glob — iss-110 is the evidence: agents/README.md IS
//     registered), skills/*/SKILL.md, and hooks/hooks.json.
//   - MANIFEST — the optional commands/agents/skills/hooks keys in plugin.json,
//     each a path, a list of paths, or an inline definition.
//
// Every entry records which register it came from, so a later, stricter tier can
// treat the two differently WITHOUT re-resolving. The resolver reports what a
// harness would register, including the iss-110 mis-registrations; filtering
// those here would hide the defect that issue tracks.
//
// # Why resolution is separate from assertion
//
// Resolution answers "what does this payload declare"; smoke.go's light tier
// asserts each resolved path exists. itd-66's deep tier asserts far more about
// the SAME list — import each entrypoint, render each command's frontmatter, in
// an isolated subprocess — and does that by consuming this list rather than
// rebuilding one. PayloadTree is what makes that a drop-in: the light tier reads
// the RESOLVED BUNDLE (no materialisation), the deep tier reads a MATERIALISED
// payload directory, and TestPayloadTreeImplementationsResolveIdentically pins
// the two to the same answer.

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// pluginManifestFile is the plugin's own manifest. Unlike the VERSION location
// (which version-location.json makes negotiable per adr-19), the manifest's own
// path is fixed by the harness's discovery rule — a plugin.json anywhere else is
// not found at all — so it is a constant here rather than a contract read.
const pluginManifestFile = ".claude-plugin/plugin.json"

// maxSurfaceManifestBytes caps every guarded manifest read. A plugin manifest and
// a hooks manifest are short declarations, so a file that is not one must not
// stream unbounded input into a gate that runs in CI.
const maxSurfaceManifestBytes = 1 << 20

// PayloadTree is the read side of a payload: the only thing surface resolution
// needs from "the thing that would ship". Two implementations satisfy it — a
// resolved bundle (nothing materialised) and a rendered directory — so the same
// resolver serves the light tier and itd-66's deep tier unchanged.
type PayloadTree interface {
	// Has reports whether a FILE exists at rel, a payload-relative,
	// slash-separated path. A path escaping the payload is never present.
	Has(rel string) bool
	// Read returns the bytes at rel, guarded against oversized input.
	Read(rel string) ([]byte, error)
	// List returns every file at or beneath the directory dirRel, sorted, as
	// payload-relative slash paths. An empty result means "no such directory".
	List(dirRel string) []string
}

// SurfaceKind is one register of the installable surface.
type SurfaceKind string

const (
	SurfaceCommand SurfaceKind = "command"
	SurfaceAgent   SurfaceKind = "agent"
	SurfaceSkill   SurfaceKind = "skill"
	SurfaceHook    SurfaceKind = "hook"
)

// SurfaceOrigin records HOW an entry came to be declared. It is carried on every
// entry so a stricter tier can weigh the registers differently without
// re-resolving them.
type SurfaceOrigin string

const (
	// OriginConvention — found by auto-discovery under a well-known directory.
	OriginConvention SurfaceOrigin = "convention"
	// OriginManifest — named by an explicit plugin.json key.
	OriginManifest SurfaceOrigin = "manifest"
	// OriginHookCommand — referenced by a hook's command string.
	OriginHookCommand SurfaceOrigin = "hook-command"
)

// SurfaceRequirement says who is expected to supply an entry's path.
type SurfaceRequirement string

const (
	// RequirePayload — the payload must carry it; absence is a failed install.
	RequirePayload SurfaceRequirement = "payload"
	// RequireInstalled — the install supplies it, so the payload never carries
	// it and its absence proves nothing.
	RequireInstalled SurfaceRequirement = "installed"
)

// SurfaceEntry is one declared piece of the installable surface.
type SurfaceEntry struct {
	Kind        SurfaceKind        `json:"kind"`
	Path        string             `json:"path"`
	Origin      SurfaceOrigin      `json:"origin"`
	Requirement SurfaceRequirement `json:"requirement"`
	// DeclaredAs is the raw declaration this entry was expanded from, when it
	// differs from Path (a directory declaration, a hook command string).
	DeclaredAs string `json:"declared_as,omitempty"`
	// Reason explains a requirement other than RequirePayload.
	Reason string `json:"reason,omitempty"`
}

// SourceKind classifies a marketplace entry's source.
type SourceKind string

const (
	// SourceLocal — a path inside the payload; resolvable offline.
	SourceLocal SourceKind = "local"
	// SourceExternal — a remote or non-path source; resolving it needs the
	// network, so no offline gate asserts against it.
	SourceExternal SourceKind = "external"
	// SourceMissing — the entry declares no source at all.
	SourceMissing SourceKind = "missing"
	// SourceArchive — a pinned release archive ({"source": "archive", "url",
	// "sha256"}). The archive is rendered from this payload's root (archive.go),
	// so offline it resolves to the payload root, exactly like "./"; the digest
	// it pins is judged by the release gate, which re-renders the archive.
	SourceArchive SourceKind = "archive"
)

// MarketplaceEntry is one plugin listing in the marketplace manifest.
type MarketplaceEntry struct {
	Name       string     `json:"name"`
	Source     string     `json:"source,omitempty"`
	SourceKind SourceKind `json:"source_kind"`
	// Root is the payload-relative plugin root a local source resolves to; the
	// empty string is the payload root itself (adr-28: the single repo is its
	// own marketplace, so the canonical source is "./").
	Root string `json:"root"`
	// Pin is the archive a SourceArchive entry names; nil for every other kind.
	Pin *ArchivePin `json:"pin,omitempty"`
}

// InstallSurface is everything a payload declares about what installing it
// would register.
type InstallSurface struct {
	PluginName  string             `json:"plugin_name"`
	Marketplace []MarketplaceEntry `json:"marketplace"`
	Entries     []SurfaceEntry     `json:"entries"`
}

// conventionRules are the auto-discovery roots a harness loads without any
// manifest declaration. match receives the payload-relative path, so a rule can
// pin depth: agents are a FLAT glob (iss-110) while commands nest to namespace
// themselves.
var conventionRules = []struct {
	kind  SurfaceKind
	root  string
	match func(rel string) bool
}{
	{SurfaceCommand, "commands", func(rel string) bool { return strings.HasSuffix(rel, ".md") }},
	{SurfaceAgent, "agents", func(rel string) bool {
		return strings.HasSuffix(rel, ".md") && strings.Count(rel, "/") == 1
	}},
	{SurfaceSkill, "skills", func(rel string) bool {
		return path.Base(rel) == "SKILL.md" && strings.Count(rel, "/") == 2
	}},
	{SurfaceHook, "hooks", func(rel string) bool { return rel == "hooks/hooks.json" }},
}

// declarationKeys are the plugin.json keys that name additional surface.
var declarationKeys = []struct {
	key  string
	kind SurfaceKind
}{
	{"commands", SurfaceCommand},
	{"agents", SurfaceAgent},
	{"skills", SurfaceSkill},
	{"hooks", SurfaceHook},
}

// ResolveInstallSurface returns everything tree declares: the plugin's name, the
// marketplace listings with their sources resolved, and the union of the
// convention and manifest surface entries.
//
// It returns an error only when a manifest or a hooks config is PRESENT and
// cannot be read or parsed — a payload whose declarations cannot even be
// enumerated. Everything
// else, including a declaration pointing at nothing, is DATA: it becomes an
// entry, and judging it is the assertion tier's job, not resolution's.
//
// An ABSENT manifest is an absent declaration, not a broken payload
// (iss-2609100506255436). The constant above fixes WHERE a plugin manifest
// lives, because the harness discovers it at one location only; it says nothing
// about WHETHER this artefact has one, and the two are separate facts. adr-19 is
// the precedent for the distinction inside this same file: the neighbouring
// release-shaped fact — where the VERSION lives — was made a per-repo declared
// contract (version-location.json) instead of an assumption. So a payload whose
// artefact is a binary, an application bundle or a library resolves to a surface
// that carries no plugin name and no marketplace listing, rather than failing
// resolution before anything else runs.
func ResolveInstallSurface(tree PayloadTree) (InstallSurface, error) {
	var surface InstallSurface

	plugin, _, err := readOptionalManifest(tree, pluginManifestFile)
	if err != nil {
		return surface, err
	}
	surface.PluginName, _ = plugin["name"].(string)

	market, marketPresent, err := readOptionalManifest(tree, marketplaceFile)
	if err != nil {
		return surface, err
	}
	if marketPresent {
		surface.Marketplace, err = resolveMarketplace(market)
		if err != nil {
			return surface, err
		}
	}

	entries := conventionEntries(tree)
	entries = append(entries, manifestEntries(tree, plugin)...)
	hookEntries, err := hookCommandEntries(tree, plugin, surface.PluginName, entries)
	if err != nil {
		return surface, err
	}
	entries = append(entries, hookEntries...)
	surface.Entries = dedupeEntries(entries)
	return surface, nil
}

// readOptionalManifest reads one manifest the payload MAY carry. present is
// false, with no error, when the payload simply does not have it; every other
// failure — an unreadable file, a body that is not a JSON object — is the error
// readManifest already returns, because a manifest that is there and cannot be
// enumerated is a broken payload.
//
// Absence is distinguished by asking the tree, not by classifying the read's
// error: PayloadTree.Has is the interface's own answer to "is this file in the
// payload", both implementations already give it, and an errors.Is over
// fs.ErrNotExist would have to hold for every future implementation's error
// wrapping as well.
func readOptionalManifest(tree PayloadTree, rel string) (map[string]any, bool, error) {
	if !tree.Has(rel) {
		return nil, false, nil
	}
	doc, err := readManifest(tree, rel)
	if err != nil {
		return nil, true, err
	}
	return doc, true, nil
}

// readManifest reads and decodes one JSON manifest object from the payload.
func readManifest(tree PayloadTree, rel string) (map[string]any, error) {
	data, err := tree.Read(rel)
	if err != nil {
		return nil, fmt.Errorf("%s is not readable in the payload: %w", rel, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s does not parse as a JSON object: %w", rel, err)
	}
	return doc, nil
}

// resolveMarketplace classifies every plugin listing's source. A source is local
// only when it is an explicit relative path; anything else (a remote object, a
// shorthand, a URL) needs the network, and an offline gate must record that
// rather than pretend to have checked it.
func resolveMarketplace(market map[string]any) ([]MarketplaceEntry, error) {
	raw, ok := market["plugins"].([]any)
	if !ok {
		return nil, fmt.Errorf("%s declares no plugins array", marketplaceFile)
	}
	out := make([]MarketplaceEntry, 0, len(raw))
	for i, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: plugins[%d] is not an object", marketplaceFile, i)
		}
		entry := MarketplaceEntry{SourceKind: SourceMissing}
		entry.Name, _ = obj["name"].(string)
		switch src := obj["source"].(type) {
		case string:
			entry.Source = src
			switch {
			case src == "." || src == "./":
				entry.SourceKind, entry.Root = SourceLocal, ""
			case strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../"):
				entry.SourceKind, entry.Root = SourceLocal, normaliseDeclared(src)
			default:
				entry.SourceKind = SourceExternal
			}
		case nil:
			// left as SourceMissing
		case map[string]any:
			entry.SourceKind = SourceExternal
			if src["source"] == archiveSourceKind {
				pin := ArchivePin{}
				pin.URL, _ = src["url"].(string)
				pin.SHA256, _ = src["sha256"].(string)
				entry.Source, entry.SourceKind, entry.Root, entry.Pin = archiveSourceKind, SourceArchive, "", &pin
			}
		default:
			entry.SourceKind = SourceExternal
		}
		out = append(out, entry)
	}
	return out, nil
}

// conventionEntries enumerates the auto-discovered surface.
func conventionEntries(tree PayloadTree) []SurfaceEntry {
	var out []SurfaceEntry
	for _, rule := range conventionRules {
		for _, rel := range tree.List(rule.root) {
			if rule.match(rel) {
				out = append(out, SurfaceEntry{
					Kind: rule.kind, Path: rel,
					Origin: OriginConvention, Requirement: RequirePayload,
				})
			}
		}
	}
	return out
}

// manifestEntries expands the explicit declaration keys.
func manifestEntries(tree PayloadTree, plugin map[string]any) []SurfaceEntry {
	var out []SurfaceEntry
	for _, dk := range declarationKeys {
		for _, decl := range declaredPaths(plugin[dk.key]) {
			out = append(out, expandDeclared(tree, dk.kind, decl)...)
		}
	}
	return out
}

// declaredPaths flattens one declaration value into the paths it names. A string
// is a path; an array is a list of them; an object is an inline definition,
// which names a path only through a member's "source".
func declaredPaths(v any) []string {
	switch d := v.(type) {
	case string:
		return []string{d}
	case []any:
		var out []string
		for _, item := range d {
			out = append(out, declaredPaths(item)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, member := range d {
			obj, ok := member.(map[string]any)
			if !ok {
				continue
			}
			if src, ok := obj["source"].(string); ok {
				out = append(out, src)
			}
		}
		sort.Strings(out) // map iteration is unordered; entries must be stable
		return out
	default:
		return nil
	}
}

// expandDeclared turns one declaration into entries: the file it names, or every
// file beneath the directory it names.
//
// A declaration that matches NOTHING in the payload still yields an entry at its
// own path. Dropping it would turn the loudest failure this gate exists to catch
// — a manifest pointing at a file that does not ship — into a silent pass.
func expandDeclared(tree PayloadTree, kind SurfaceKind, decl string) []SurfaceEntry {
	norm := normaliseDeclared(decl)
	if tree.Has(norm) {
		return []SurfaceEntry{{Kind: kind, Path: norm, Origin: OriginManifest, Requirement: RequirePayload}}
	}
	if files := tree.List(norm); len(files) > 0 {
		out := make([]SurfaceEntry, 0, len(files))
		for _, rel := range files {
			out = append(out, SurfaceEntry{
				Kind: kind, Path: rel, Origin: OriginManifest,
				Requirement: RequirePayload, DeclaredAs: decl,
			})
		}
		return out
	}
	return []SurfaceEntry{{Kind: kind, Path: norm, Origin: OriginManifest, Requirement: RequirePayload}}
}

// normaliseDeclared makes a declaration comparable with a payload-relative path.
// It does NOT neutralise an escaping path: "../x" stays "../x" so the entry
// survives and fails its existence check by name, rather than being silently
// rewritten into something that might exist.
func normaliseDeclared(decl string) string {
	d := strings.TrimSpace(decl)
	d = strings.TrimPrefix(d, "./")
	if d == "" {
		return d
	}
	return path.Clean(d)
}

// hookCommandEntries resolves the payload files a hook's command string invokes.
//
// This is the class of break an existence check over declared files alone cannot
// see: a hook that runs a script the payload does not carry installs cleanly and
// then fails at runtime. Only the `$CLAUDE_PLUGIN_ROOT`-rooted references are
// resolvable — anything else is a PATH lookup on the user's machine, which no
// release gate can assert.
//
// A hooks config the payload carries and that cannot be read or parsed is an
// error, exactly as an unparseable plugin manifest is: the host registers no
// hook from it on any install, so skipping it would let the loudest hook
// failure read as a payload that declares no hooks (iss-2609251827104081).
func hookCommandEntries(tree PayloadTree, plugin map[string]any, pluginName string, found []SurfaceEntry) ([]SurfaceEntry, error) {
	var docs []any
	if len(plugin) > 0 {
		docs = append(docs, plugin["hooks"])
	}
	for _, e := range found {
		if !isHookConfig(e) || !tree.Has(e.Path) {
			continue // absence is the existence check's finding, not this one's
		}
		doc, err := readHookConfig(tree, e.Path)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	var out []SurfaceEntry
	for _, doc := range docs {
		for _, command := range collectCommandStrings(doc) {
			for _, ref := range pluginRootRefs(command) {
				entry := SurfaceEntry{
					Kind: SurfaceHook, Path: ref, Origin: OriginHookCommand,
					Requirement: RequirePayload, DeclaredAs: command,
				}
				if ref == pluginName && pluginName != "" {
					entry.Requirement = RequireInstalled
					entry.Reason = "the plugin executable is supplied by the install, not carried in the payload"
				}
				out = append(out, entry)
			}
		}
	}
	return out, nil
}

// isHookConfig reports whether an entry is a hooks config file the payload is
// responsible for: a hook entry named directly (by convention or a manifest
// key), or a JSON file swept from a declared hooks directory. Any other file a
// directory declaration sweeps up (a script beside the config) is not a
// config and is never parsed as one.
func isHookConfig(e SurfaceEntry) bool {
	if e.Kind != SurfaceHook || e.Origin == OriginHookCommand || e.Requirement != RequirePayload {
		return false
	}
	return e.DeclaredAs == "" || strings.EqualFold(path.Ext(e.Path), ".json")
}

// readHookConfig reads and decodes one hooks config from the payload.
func readHookConfig(tree PayloadTree, rel string) (any, error) {
	data, err := tree.Read(rel)
	if err != nil {
		return nil, fmt.Errorf("%s is not readable in the payload: %w", rel, err)
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s does not parse as JSON, so the host would register none of its hooks: %w", rel, err)
	}
	return doc, nil
}

// collectCommandStrings walks a decoded hooks document and returns every
// "command" string in it, at any depth. Walking generically rather than
// following the hooks schema means a future event name or nesting level costs
// this gate nothing.
func collectCommandStrings(doc any) []string {
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys) // map iteration is unordered; entries must be stable
			for _, k := range keys {
				if k == "command" {
					if s, ok := t[k].(string); ok {
						out = append(out, s)
						continue
					}
				}
				walk(t[k])
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(doc)
	return out
}

// pluginRootRefs extracts every path a command string reads out of the plugin
// root, in either `$CLAUDE_PLUGIN_ROOT/x` or `${CLAUDE_PLUGIN_ROOT}/x` form.
//
// It deliberately does not emulate a shell: the marker plus a literal suffix is
// the whole contract a hook command needs to name a shipped file, and a path
// assembled at runtime is not knowable at release time. A reference this misses
// yields no entry, never a false failure.
func pluginRootRefs(command string) []string {
	const name = "CLAUDE_PLUGIN_ROOT"
	var out []string
	for i := 0; i < len(command); {
		j := strings.Index(command[i:], name)
		if j < 0 {
			break
		}
		start := i + j
		i = start + len(name)
		if !dollarRooted(command, start) {
			continue
		}
		rest := command[i:]
		rest = strings.TrimPrefix(rest, "}")
		if !strings.HasPrefix(rest, "/") {
			continue
		}
		ref := rest[1:]
		if k := strings.IndexAny(ref, " \t\"';)|&"); k >= 0 {
			ref = ref[:k]
		}
		if ref = normaliseDeclared(ref); ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

// dollarRooted reports whether the variable name at start is an actual expansion
// (`$NAME` or `${NAME}`) rather than the literal word appearing in prose.
func dollarRooted(command string, start int) bool {
	if start >= 1 && command[start-1] == '$' {
		return true
	}
	return start >= 2 && command[start-1] == '{' && command[start-2] == '$'
}

// dedupeEntries collapses paths declared twice (a convention file also named
// explicitly) and sorts the result, so the entry list is a stable set that two
// PayloadTree implementations can be compared on.
func dedupeEntries(entries []SurfaceEntry) []SurfaceEntry {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Path < entries[j].Path
	})
	out := entries[:0]
	var lastKind SurfaceKind
	var lastPath string
	for i, e := range entries {
		if i > 0 && e.Kind == lastKind && e.Path == lastPath {
			continue
		}
		lastKind, lastPath = e.Kind, e.Path
		out = append(out, e)
	}
	return out
}

// ---------------------------------------------------------------------------
// PayloadTree implementations
// ---------------------------------------------------------------------------

// bundleTree reads a payload that has not been materialised: the file list is
// the bundle's Included set and each read goes to the file's resolved source.
type bundleTree struct {
	files  map[string]string // logical path -> resolved on-disk path
	sorted []string
}

// NewBundleTree views a resolved bundle as a payload tree, so the light tier can
// assert against WHAT WOULD SHIP without writing anything. A file present in the
// working tree but excluded from the payload is absent here — which is exactly
// the bug this gate catches.
func NewBundleTree(b Bundle) PayloadTree {
	t := &bundleTree{files: make(map[string]string, len(b.Included))}
	for _, f := range b.Included {
		t.files[f.LogicalPath] = f.ResolvedPath
	}
	t.sorted = make([]string, 0, len(t.files))
	for rel := range t.files {
		t.sorted = append(t.sorted, rel)
	}
	sort.Strings(t.sorted)
	return t
}

func (t *bundleTree) Has(rel string) bool {
	_, ok := t.files[rel]
	return ok
}

func (t *bundleTree) Read(rel string) ([]byte, error) {
	resolved, ok := t.files[rel]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fsutil.ReadGuarded(resolved, maxSurfaceManifestBytes)
}

func (t *bundleTree) List(dirRel string) []string {
	return withinDir(t.sorted, dirRel)
}

// dirTree reads a materialised payload directory — the shape itd-66's deep tier
// works from, since it runs its imports in a subprocess rooted at the rendered
// snapshot.
type dirTree struct{ root string }

// NewDirTree views a rendered payload directory as a payload tree.
func NewDirTree(root string) PayloadTree { return &dirTree{root: root} }

func (t *dirTree) resolve(rel string) (string, bool) {
	// The canonical lexical guard, not a hand-rolled one: rejecting only an
	// absolute path or a leading "../" let an embedded a/../../b traversal
	// through, which filepath.Join then cleans into an escape of t.root
	// (iss-2608270655490198). ValidRelPath refuses every unclean or escaping
	// form.
	if !fsutil.ValidRelPath(rel) {
		return "", false
	}
	return filepath.Join(t.root, filepath.FromSlash(rel)), true
}

func (t *dirTree) Has(rel string) bool {
	abs, ok := t.resolve(rel)
	if !ok {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && info.Mode().IsRegular()
}

func (t *dirTree) Read(rel string) ([]byte, error) {
	abs, ok := t.resolve(rel)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fsutil.ReadGuarded(abs, maxSurfaceManifestBytes)
}

func (t *dirTree) List(dirRel string) []string {
	abs, ok := t.resolve(dirRel)
	if !ok {
		return nil
	}
	var out []string
	_ = filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil //nolint:nilerr // an unreadable branch is "not present", not a gate fault
		}
		rel, rerr := filepath.Rel(t.root, p)
		if rerr != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

// withinDir returns the members of a sorted path list that lie at or beneath
// dirRel.
func withinDir(sorted []string, dirRel string) []string {
	if dirRel == "" {
		return nil
	}
	prefix := dirRel + "/"
	var out []string
	for _, rel := range sorted {
		if rel == dirRel || strings.HasPrefix(rel, prefix) {
			out = append(out, rel)
		}
	}
	return out
}
