package lifeboat

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// conventionSources returns the Tier-1 adapters: brief sections derivable from
// conventional project files (README, docs/, CHANGELOG, LICENSE, CONTRIBUTING,
// build manifests, CI workflows, and ADRs wherever they live). Every adapter
// reads only through the SourceContext file surface and never touches git, so it
// grounds the same sections against any directory — a git working tree or a bare
// snapshot.
func conventionSources() []Source {
	return []Source{
		convContextSource{},
		convPressReleaseSource{},
		convScopeSource{},
		convPlatformSource{},
		convDependenciesSource{},
		convADRsSource{},
		convSurfacesSource{},
		convOutOfScopeSource{},
		convGlossarySource{},
		convNamingSource{},
		convInternalsSource{},
		convOpenQuestionsSource{},
		convIssuesSource{},
		convWhatWorkedSource{},
	}
}

// convReadmeNames are the conventional README spellings, in preference order.
var convReadmeNames = []string{
	"README.md", "README", "README.rst", "README.txt", "readme.md",
}

// convGroundedProseBytes is the body-prose threshold above which a README is
// treated as real documentation (grounded) rather than a near-empty stub
// (partial). Body prose excludes heading lines, code fences, and blank lines.
const convGroundedProseBytes = 120

// convReadme returns the first README that exists, with its contents. ok is
// false when no README is present.
func convReadme(ctx *SourceContext) (path string, data []byte, ok bool) {
	p := ctx.FindFirst(convReadmeNames...)
	if p == "" {
		return "", nil, false
	}
	d, read := ctx.ReadFile(p)
	if !read {
		return "", nil, false
	}
	return p, d, true
}

// convIsHeading reports whether a line is a Markdown heading.
func convIsHeading(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}

// convProseBytes counts the body-prose characters of a README: non-blank,
// non-heading, non-fence lines. It is the measure that separates a documented
// project from a stub.
func convProseBytes(data []byte) int {
	total := 0
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || convIsHeading(line) || strings.HasPrefix(t, "```") {
			continue
		}
		total += len(t)
	}
	return total
}

// convLede extracts a README's lede: the first heading as a title and the first
// paragraph beneath it. ok is false when there is no non-blank content at all.
func convLede(data []byte) (title, para string, ok bool) {
	lines := strings.Split(string(data), "\n")
	i := 0
	// Title: first heading, else first non-blank line.
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			continue
		}
		if convIsHeading(lines[i]) {
			title = strings.TrimSpace(strings.TrimLeft(t, "#"))
		} else {
			title = t
		}
		i++
		break
	}
	if title == "" {
		return "", "", false
	}
	// Paragraph: first block of consecutive non-blank, non-heading lines.
	var buf []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			if len(buf) > 0 {
				break
			}
			continue
		}
		if convIsHeading(lines[i]) {
			if len(buf) > 0 {
				break
			}
			continue
		}
		buf = append(buf, t)
	}
	return title, strings.Join(buf, " "), true
}

// convHasHeadingLike reports whether any Markdown heading in data contains one
// of keywords (case-insensitive).
func convHasHeadingLike(data []byte, keywords ...string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		if !convIsHeading(line) {
			continue
		}
		low := strings.ToLower(line)
		for _, k := range keywords {
			if strings.Contains(low, k) {
				return true
			}
		}
	}
	return false
}

// convHasCodeFence reports whether data contains a fenced code block, a strong
// signal of usage/CLI examples.
func convHasCodeFence(data []byte) bool {
	return strings.Contains(string(data), "\n```") || strings.HasPrefix(string(data), "```")
}

// convADRDirs are the conventional homes for ADRs, sorted so a citation is
// deterministic.
var convADRDirs = []string{
	"adr", "architecture/decisions", "docs/adr", "docs/adrs", "docs/decisions",
}

// convListADRs returns the repo-relative paths of ADR documents found under the
// conventional ADR directories, unique and sorted.
func convListADRs(ctx *SourceContext) []string {
	var out []string
	for _, dir := range convADRDirs {
		if !ctx.IsDir(dir) {
			continue
		}
		for _, name := range ctx.ListDir(dir) {
			low := strings.ToLower(name)
			if strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".markdown") {
				out = append(out, dir+"/"+name)
			}
		}
	}
	return dedupeSorted(out)
}

// convContextSource grounds "product/context" from a README carrying real prose.
// A near-empty README is partial; no README is a blank a human must answer.
type convContextSource struct{}

func (convContextSource) Section() Section { return "product/context" }
func (convContextSource) Tier() Tier       { return TierConventions }

func (convContextSource) Probe(ctx *SourceContext) Evidence {
	path, data, ok := convReadme(ctx)
	if !ok {
		return blank(
			[]string{"README (" + strings.Join(convReadmeNames, ", ") + ")"},
			"What does this project do? No README to describe it.",
		)
	}
	if convProseBytes(data) < convGroundedProseBytes {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceLow,
			Sources:    []string{path + " (near-empty)"},
		}
	}
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    []string{path},
	}
}

// convPressReleaseSource partially grounds "product/press-release" from the
// README lede — its first heading and paragraph read as the project's pitch.
type convPressReleaseSource struct{}

func (convPressReleaseSource) Section() Section { return "product/press-release" }
func (convPressReleaseSource) Tier() Tier       { return TierConventions }

func (convPressReleaseSource) Probe(ctx *SourceContext) Evidence {
	path, data, ok := convReadme(ctx)
	if !ok {
		return blank(
			[]string{"README lede (first heading + paragraph)"},
			"What is the one-line pitch for this project? No README lede to draw it from.",
		)
	}
	title, _, hasLede := convLede(data)
	if !hasLede {
		return blank(
			[]string{"README lede (first heading + paragraph)"},
			"What is the one-line pitch for this project? The README carries no lede.",
		)
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: ConfidenceLow,
		Sources:    []string{fmt.Sprintf("%s lede (%q)", path, title)},
	}
}

// convScopeSource partially grounds "product/scope" from a README's features or
// usage sections. Blank when no README is present.
type convScopeSource struct{}

func (convScopeSource) Section() Section { return "product/scope" }
func (convScopeSource) Tier() Tier       { return TierConventions }

func (convScopeSource) Probe(ctx *SourceContext) Evidence {
	path, data, ok := convReadme(ctx)
	if !ok {
		return blank(
			[]string{"README features/usage sections"},
			"What is in scope for this project? No README to read features from.",
		)
	}
	if convHasHeadingLike(data, "feature", "usage", "install", "getting started", "what") {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{path + " (features/usage sections)"},
		}
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: ConfidenceLow,
		Sources:    []string{path + " (no explicit features section)"},
	}
}

// convPlatformFiles are the build/CI signals that ground "constraints/platform",
// sorted so a citation is deterministic.
var convPlatformFiles = []string{
	"Dockerfile", "Makefile", "go.mod", "package.json",
}

// convPlatformSource grounds "constraints/platform" from build manifests and CI
// workflows. Blank when the project carries neither.
type convPlatformSource struct{}

func (convPlatformSource) Section() Section { return "constraints/platform" }
func (convPlatformSource) Tier() Tier       { return TierConventions }

func (convPlatformSource) Probe(ctx *SourceContext) Evidence {
	var found []string
	for _, f := range convPlatformFiles {
		if ctx.Exists(f) {
			found = append(found, f)
		}
	}
	if ctx.IsDir(".github/workflows") {
		found = append(found, ".github/workflows")
	}
	if len(found) == 0 {
		return blank(
			[]string{"build manifests (" + strings.Join(convPlatformFiles, ", ") + ")", "CI workflows (.github/workflows)"},
			"What platform and toolchain does this project require? No build manifest or CI found.",
		)
	}
	sort.Strings(found)
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    found,
	}
}

// convManifestLock pairs a dependency manifest with the lockfile that pins it. A
// present lockfile makes the dependency list authoritative (grounded); a missing
// one leaves it partial.
type convManifestLock struct {
	manifest string
	locks    []string // any one present counts as the lock
}

// convManifestLocks are the recognised manifest/lock pairs, in a fixed order so
// the first match is deterministic. It spans the ecosystems the probe is likely
// to meet — Go, Node, Rust, Python (pip/poetry/pdm/uv/pipenv), Ruby, and PHP —
// so a real project is not reported as having no dependencies merely because the
// probe did not know its packaging tool.
var convManifestLocks = []convManifestLock{
	{"go.mod", []string{"go.sum"}},
	{"package.json", []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml"}},
	{"Cargo.toml", []string{"Cargo.lock"}},
	{"pyproject.toml", []string{"uv.lock", "poetry.lock", "pdm.lock"}},
	{"Pipfile", []string{"Pipfile.lock"}},
	{"requirements.txt", []string{"requirements.txt"}}, // pinned requirements is its own lock
	{"Gemfile", []string{"Gemfile.lock"}},
	{"composer.json", []string{"composer.lock"}},
}

// convDependenciesSource grounds "constraints/dependencies" from a manifest and
// its lockfile; a manifest without a lockfile is partial. Blank when no manifest
// is present.
type convDependenciesSource struct{}

func (convDependenciesSource) Section() Section { return "constraints/dependencies" }
func (convDependenciesSource) Tier() Tier       { return TierConventions }

func (convDependenciesSource) Probe(ctx *SourceContext) Evidence {
	for _, ml := range convManifestLocks {
		if !ctx.Exists(ml.manifest) {
			continue
		}
		var lock string
		for _, l := range ml.locks {
			if l == ml.manifest {
				lock = l // requirements.txt pins itself
				break
			}
			if ctx.Exists(l) {
				lock = l
				break
			}
		}
		if lock != "" && lock != ml.manifest {
			return Evidence{
				Status:     StatusGrounded,
				Confidence: ConfidenceHigh,
				Sources:    []string{ml.manifest, lock},
			}
		}
		if lock == ml.manifest {
			return Evidence{
				Status:     StatusGrounded,
				Confidence: ConfidenceMedium,
				Sources:    []string{ml.manifest},
			}
		}
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{ml.manifest + " (no lockfile)"},
		}
	}
	return blank(
		[]string{"dependency manifest + lockfile (go.mod, package.json, Cargo.toml, pyproject.toml, Pipfile, requirements.txt, Gemfile, composer.json)"},
		"What does this project depend on? No dependency manifest found.",
	)
}

// convADRsSource grounds "docs/adrs" when ADRs live under a conventional ADR
// directory, listing the documents found. Blank when none exist.
type convADRsSource struct{}

func (convADRsSource) Section() Section { return "docs/adrs" }
func (convADRsSource) Tier() Tier       { return TierConventions }

func (convADRsSource) Probe(ctx *SourceContext) Evidence {
	adrs := convListADRs(ctx)
	if len(adrs) == 0 {
		return blank(
			[]string{"ADRs under " + strings.Join(convADRDirs, ", ")},
			"What architectural decisions has this project recorded? No ADR directory found.",
		)
	}
	sources := []string{fmt.Sprintf("%d ADR(s) under conventional dirs", len(adrs))}
	sources = append(sources, adrs...)
	return Evidence{
		Status:     StatusGrounded,
		Confidence: ConfidenceHigh,
		Sources:    dedupeSorted(sources),
	}
}

// convSurfacesSource partially grounds "surfaces" from a README's usage or CLI
// sections, or its fenced usage examples. Blank when no README is present.
type convSurfacesSource struct{}

func (convSurfacesSource) Section() Section { return "surfaces" }
func (convSurfacesSource) Tier() Tier       { return TierConventions }

func (convSurfacesSource) Probe(ctx *SourceContext) Evidence {
	path, data, ok := convReadme(ctx)
	if !ok {
		return blank(
			[]string{"README usage/CLI sections", "CLI help text"},
			"What are this project's surfaces (CLI, API)? No README usage to read them from.",
		)
	}
	if convHasHeadingLike(data, "usage", "cli", "command", "api", "getting started") || convHasCodeFence(data) {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{path + " (usage/CLI sections)"},
		}
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: ConfidenceLow,
		Sources:    []string{path + " (no explicit usage section)"},
	}
}

// convOutOfScopeSource partially grounds "delivery/out-of-scope" from a README's
// non-goals or out-of-scope section. Blank when that section is absent.
type convOutOfScopeSource struct{}

func (convOutOfScopeSource) Section() Section { return "delivery/out-of-scope" }
func (convOutOfScopeSource) Tier() Tier       { return TierConventions }

func (convOutOfScopeSource) Probe(ctx *SourceContext) Evidence {
	path, data, ok := convReadme(ctx)
	if ok && convHasHeadingLike(data, "non-goal", "non goal", "out of scope", "out-of-scope", "not a goal") {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{path + " (non-goals / out-of-scope section)"},
		}
	}
	return blank(
		[]string{"README non-goals / out-of-scope section"},
		"What did this project deliberately leave out? No out-of-scope section found.",
	)
}

// convGlossaryDocNames are the glossary spellings checked outside docs/.
var convGlossaryDocNames = []string{"GLOSSARY.md", "GLOSSARY", "glossary.md"}

// convDocsEntryWithPrefix returns the repo-relative path of the first entry
// directly under docs/ whose lower-cased name starts with prefix, or "". It is
// the docs/ convention several adapters share: a project that keeps its glossary
// or its naming rules under docs/ names the page after the thing it documents.
func convDocsEntryWithPrefix(ctx *SourceContext, prefix string) string {
	for _, name := range ctx.ListDir("docs") {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			return "docs/" + name
		}
	}
	return ""
}

// convGlossaryDoc returns the glossary document a repository carries — a
// conventional spelling at the root, else a docs/glossary* page — or "". It is
// the single definition of "where the glossary lives", read both by the glossary
// adapter that owns the section and by the naming adapter that falls back to it.
func convGlossaryDoc(ctx *SourceContext) string {
	if p := ctx.FindFirst(convGlossaryDocNames...); p != "" {
		return p
	}
	return convDocsEntryWithPrefix(ctx, "glossary")
}

// convGlossarySource partially grounds "glossary" from a glossary document under
// docs/ or at the repo root. Blank when none exists.
type convGlossarySource struct{}

func (convGlossarySource) Section() Section { return "glossary" }
func (convGlossarySource) Tier() Tier       { return TierConventions }

func (convGlossarySource) Probe(ctx *SourceContext) Evidence {
	if p := convGlossaryDoc(ctx); p != "" {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{p},
		}
	}
	return blank(
		[]string{"GLOSSARY.md", "docs/glossary*"},
		"What terms does this project define? No glossary document found.",
	)
}

// convNamingDocNames are the dedicated naming-document spellings checked outside
// docs/. The list is short because there is no strong standard: a project that
// rules on names usually writes one file and calls it NAMING.
var convNamingDocNames = []string{"NAMING.md", "NAMING", "naming.md"}

// convNamingSource partially grounds "constraints/naming" from a dedicated
// naming document, falling back to the glossary. A project that never wrote a
// naming registry usually encodes its reserved vocabulary in its glossary, so
// the glossary is real evidence for naming — weaker, because a glossary defines
// terms rather than ruling on what may be renamed, and cited with that
// qualifier so the section is never mistaken for a copy of the glossary row.
//
// The ceiling is StatusPartial in both cases: neither a naming page nor a
// glossary enumerates a project's full reserved vocabulary. The strength of the
// signal is carried by Confidence, not by inflating the status.
type convNamingSource struct{}

func (convNamingSource) Section() Section { return "constraints/naming" }
func (convNamingSource) Tier() Tier       { return TierConventions }

func (convNamingSource) Probe(ctx *SourceContext) Evidence {
	p := ctx.FindFirst(convNamingDocNames...)
	if p == "" {
		p = convDocsEntryWithPrefix(ctx, "naming")
	}
	if p != "" {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{p},
		}
	}
	if g := convGlossaryDoc(ctx); g != "" {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceLow,
			Sources:    []string{g + " (glossary fallback — no dedicated naming document)"},
		}
	}
	return blank(
		[]string{
			"naming document (" + strings.Join(convNamingDocNames, ", ") + ")",
			"docs/naming*",
			"GLOSSARY.md", "docs/glossary*",
		},
		"What names and reserved vocabulary are fixed? No naming document and no glossary found.",
	)
}

// convArchitectureDocNames are the architecture-document spellings checked as a
// single file, in preference order.
var convArchitectureDocNames = []string{
	"ARCHITECTURE.md", "ARCHITECTURE", "architecture.md", "docs/architecture.md",
}

// convArchitectureDirs are the conventional homes for architecture prose spread
// across several files, in preference order. The last is the Diátaxis
// explanation quadrant, which is where a repo following that scheme puts it.
var convArchitectureDirs = []string{
	"docs/architecture", "docs/design", "docs/explanation",
}

// convLayoutRoots are the directory names that hold a project's own packages —
// the second, independent internals signal, and the one every codebase has
// whether or not it wrote its architecture down.
var convLayoutRoots = []string{"internal", "pkg", "src", "lib", "cmd", "app"}

// maxLayoutCitations caps how many package entries the layout scan cites. Beyond
// it the scan keeps counting — the headline stays truthful — but stops citing,
// so a monorepo with thousands of packages cannot dump them all into a brief
// section.
const maxLayoutCitations = 50

// convDirHasMarkdown reports whether a repo-relative directory holds at least one
// Markdown file — the test that separates a real documentation tree from an
// empty or placeholder one.
func convDirHasMarkdown(ctx *SourceContext, dir string) bool {
	if !ctx.IsDir(dir) {
		return false
	}
	for _, name := range ctx.ListDir(dir) {
		low := strings.ToLower(name)
		if strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".markdown") {
			return true
		}
	}
	return false
}

// convLayoutPackages returns the "<root>/<pkg>/" entries implied by walked file
// paths: the top two segments of every file beneath a recognised source root,
// unique and sorted. A file sitting directly in a root names no package, so it
// contributes nothing.
func convLayoutPackages(paths []string) []string {
	var out []string
	for _, p := range paths {
		seg := strings.Split(p, "/")
		if len(seg) < 3 {
			continue
		}
		for _, root := range convLayoutRoots {
			if seg[0] == root {
				out = append(out, seg[0]+"/"+seg[1]+"/")
				break
			}
		}
	}
	return dedupeSorted(out)
}

// convInternalsSource partially grounds "internals" from what a team wrote about
// its own architecture and from the package layout a rescuer must navigate. The
// two signals are independent: either alone is evidence, and together they are
// the strongest reading available without a record.
//
// The ceiling is StatusPartial by construction: an architecture document plus a
// package listing describes the shape of a system, not its internals chapters.
// Which signals were found moves the confidence instead.
type convInternalsSource struct{}

func (convInternalsSource) Section() Section { return "internals" }
func (convInternalsSource) Tier() Tier       { return TierConventions }

func (s convInternalsSource) Probe(ctx *SourceContext) Evidence {
	return s.probeLimited(ctx, maxWalkFiles)
}

// probeLimited is Probe with the walk cap injected, so the truncation branch is
// exercisable by a test at an affordable scale. The shipped cap stays a const:
// adapters run concurrently, and a mutable package-level cap would be shared
// state between them.
func (convInternalsSource) probeLimited(ctx *SourceContext, walkLimit int) Evidence {
	docPath := ctx.FindFirst(convArchitectureDocNames...)
	docDir := ""
	if docPath == "" {
		for _, dir := range convArchitectureDirs {
			if convDirHasMarkdown(ctx, dir) {
				docDir = dir
				break
			}
		}
	}
	paths, truncated := ctx.walkFilesLimited(".", walkLimit)
	pkgs := convLayoutPackages(paths)

	// Loud staging: a walk the bounds cut short saw only part of the tree, so a
	// rescuer never mistakes a partial layout for the whole one — and the note is
	// owed whether or not the truncated walk happened to reach a package first.
	// Each note names the bound that fired and what it cut (iss-133).
	truncatedNotes := truncated.notes()

	if docPath == "" && docDir == "" && len(pkgs) == 0 {
		searched := []string{
			"architecture document (" + strings.Join(convArchitectureDocNames, ", ") + ")",
			"architecture trees (" + strings.Join(convArchitectureDirs, ", ") + ")",
			"package layout under " + strings.Join(convLayoutRoots, ", "),
		}
		// A blank is a first-class result only while it is trustworthy (adr-35),
		// so a scan that never reached the source roots says so rather than
		// claiming there are none.
		question := "How is this system built internally? No architecture document and no recognisable package layout."
		if truncated.Any() {
			searched = append(searched, truncatedNotes...)
			question = "How is this system built internally? No architecture document, and no package layout in the part of the tree the scan reached — it did not reach all of it."
		}
		return blank(searched, question)
	}

	var sources []string
	// Layout alone is the weakest reading: it says where the code is, never why.
	confidence := ConfidenceLow
	switch {
	case docPath != "":
		// The same prose measure convContextSource applies to a README, so a
		// scaffolded ARCHITECTURE.md is not mistaken for a written one.
		if data, ok := ctx.ReadFile(docPath); ok && convProseBytes(data) >= convGroundedProseBytes {
			sources = append(sources, docPath)
			confidence = ConfidenceHigh
		} else {
			sources = append(sources, docPath+" (near-empty)")
			confidence = ConfidenceMedium
		}
	case docDir != "":
		sources = append(sources, docDir+"/")
		confidence = ConfidenceMedium
	}

	sources = append(sources, truncatedNotes...)
	if len(pkgs) > 0 {
		if len(pkgs) > maxLayoutCitations {
			sources = append(sources, fmt.Sprintf("%d further package(s) counted but not cited (citation cap %d)", len(pkgs)-maxLayoutCitations, maxLayoutCitations))
			pkgs = pkgs[:maxLayoutCitations]
		}
		sources = append(sources, pkgs...)
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: confidence,
		Sources:    dedupeSorted(sources),
	}
}

// convMarkerNames are the in-code work markers recognised as open questions,
// uppercase only. NOTE and OPTIMIZE are deliberately absent: NOTE marks
// explanation rather than unfinished work, and OPTIMIZE is rare enough that its
// false-positive cost exceeds its value. The set splits by how the pattern
// anchors each marker's trailing boundary (see convMarkerRe).
var convMarkerNames = append(append([]string{}, convDelimitedMarkers...), convBareMarkers...)

// convDelimitedMarkers must carry a trailing ':' or '(' to be recognised. They
// are the markers a project documents by name, so their bare-word form (the
// marker followed by whitespace or end-of-line) is indistinguishable from prose
// that merely mentions the marker — the dominant false-positive class on a
// repository that documents its own conventions (iss-111). Requiring the
// conventional trailing delimiter — a colon or an open-paren, as in an authored
// marker naming its owner — keeps the genuine markers and drops the mentions.
// (This comment states the markers as slice values below rather than inline, so
// the scan does not cite its own prose.)
var convDelimitedMarkers = []string{"TODO", "FIXME"}

// convBareMarkers additionally match a bare word (a trailing whitespace or
// end-of-line). These three are rarely written as bare uppercase words in running
// prose, so their bare form carries no measured false-positive cost, and that
// bare spelling is how they are conventionally written — requiring a delimiter
// would lose real markers for no precision gain. (Named only as slice values
// below, for the same reason as convDelimitedMarkers.)
var convBareMarkers = []string{"XXX", "HACK", "BUG"}

// convMarkerRe matches one recognised marker on a line. The leading class is the
// word boundary that stops a marker matching inside a longer identifier (TODOS,
// todo_list); the hyphen is excluded from it so the redaction placeholder shape
// (XXX-XXX-XXX) is rejected at every one of its triples — without that exclusion
// the last triple matches on its leading hyphen and a support phone number
// becomes a fabricated open question. The two marker classes differ only in their
// trailing boundary: convDelimitedMarkers (captured in group 2) require ':' or
// '('; convBareMarkers (group 3) also accept whitespace or end-of-line. Exactly
// one of the two marker groups is non-empty on a match — convMarkerName reads
// whichever fired. Built from the marker slices so the set and the pattern cannot
// drift, and compiled once.
var convMarkerRe = regexp.MustCompile(
	`(^|[^A-Za-z0-9_-])` +
		`(?:(` + strings.Join(convDelimitedMarkers, "|") + `)(?::|\()` +
		`|(` + strings.Join(convBareMarkers, "|") + `)(?::|\(|\s|$))`)

// convMarkerName returns the marker a convMarkerRe match captured. The pattern
// puts a delimited marker in group 2 and a bare-allowed marker in group 3, and
// exactly one is non-empty.
func convMarkerName(m []string) string {
	if m[2] != "" {
		return m[2]
	}
	return m[3]
}

// maxMarkerCitations caps how many path:line citations the marker scan reports.
// Beyond it the scan keeps counting — the headline stays truthful — but stops
// citing, so a repo with ten thousand markers cannot dump ten thousand lines
// into a brief section.
const maxMarkerCitations = 200

// maxMarkerScanBytes caps how much file content the marker scan reads across the
// whole tree. The per-file cap (maxProbeReadBytes) bounds one read and
// maxWalkFiles bounds one walk, but their product does not bound the scan: a
// hostile tree of many large files would hold the probe for hours reading and
// matching them. It reuses maxPlanTotalBytes, the ceiling the pack planner
// already puts on the same multiplication, rather than adding a third spelling
// of it. Spending it is reported in the cited evidence, exactly as reaching the
// walk cap is.
const maxMarkerScanBytes = maxPlanTotalBytes // 512 MiB

// convBinarySniffBytes is how far into a file the NUL-byte binary heuristic
// looks. It is the conventional prefix test, and needs no extension allow-list.
const convBinarySniffBytes = 8 << 10

// convMarkerMediumConfidence is the marker count at which the scan is confident
// the markers are a real seam of open questions rather than a stray note.
const convMarkerMediumConfidence = 10

// convIsBinary reports whether data looks like a binary blob: a NUL byte within
// the sniffed prefix. A blob carries no readable markers, so it is skipped.
func convIsBinary(data []byte) bool {
	head := data
	if len(head) > convBinarySniffBytes {
		head = head[:convBinarySniffBytes]
	}
	return bytes.IndexByte(head, 0) >= 0
}

// convOpenQuestionsSource partially grounds "evidence/open-questions" from the
// work markers a team left in its source — the one place every codebase records
// what it knows is unfinished, and the only surviving trace of it in a project
// that never kept a record. Blank when the tree carries none.
//
// The ceiling is StatusPartial by construction: a marker says something is
// unfinished, not what the question is, so markers are a thread to pull rather
// than a framed set of open questions. Volume moves the confidence instead.
type convOpenQuestionsSource struct{}

func (convOpenQuestionsSource) Section() Section { return "evidence/open-questions" }
func (convOpenQuestionsSource) Tier() Tier       { return TierConventions }

func (s convOpenQuestionsSource) Probe(ctx *SourceContext) Evidence {
	return s.probeLimited(ctx, maxMarkerScanBytes)
}

// probeLimited is Probe with the scan budget injected, so the exhaustion branch
// is exercisable by a test at an affordable scale. The shipped budget stays a
// const: adapters run concurrently, and a mutable package-level budget would be
// shared state between them.
func (convOpenQuestionsSource) probeLimited(ctx *SourceContext, budget int) Evidence {
	paths, truncated := ctx.WalkFiles(".")
	var (
		citations []string
		markers   int
		files     int
		scanned   int
		unread    int
	)
	for i, p := range paths {
		if scanned >= budget {
			// Loud staging: the walk found more of the tree than the scan could
			// afford to read, so the count below is partial and says so.
			unread = len(paths) - i
			break
		}
		// ReadFile carries the whole read contract already — containment, the
		// per-file byte cap, regular-file-only, non-blocking open — so an
		// oversized or unreadable file is simply skipped.
		data, ok := ctx.ReadFile(p)
		if !ok {
			continue
		}
		// Charged before the binary test: the bytes are already read and copied,
		// so a tree of blobs spends the budget exactly as a tree of source does.
		scanned += len(data)
		if convIsBinary(data) {
			continue
		}
		hits := 0
		for i, line := range strings.Split(string(data), "\n") {
			// One citation per line: the first recognised marker on it. A line
			// carrying two markers is still one place to look.
			m := convMarkerRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			hits++
			markers++
			if len(citations) < maxMarkerCitations {
				citations = append(citations, fmt.Sprintf("%s:%d (%s)", p, i+1, convMarkerName(m)))
			}
		}
		if hits > 0 {
			files++
		}
	}

	if markers == 0 {
		// The scope is stated, both ways round. A reader of a packed lifeboat has
		// to be able to tell whether an absent citation means "nothing there" or
		// "not looked at" — and the answer now depends on a caller's choice, so
		// naming the choice is part of the result rather than a footnote to it
		// (iss-2608241828356533).
		scope := "every regular file in the tree except " + strings.Join(walkSkipDirs, ", ")
		if ctx.IgnoredAreIncluded() {
			scope += ", INCLUDING files git ignores"
		} else {
			scope += ", and excluding files git ignores"
		}
		searched := []string{
			"in-code work markers (" + strings.Join(convMarkerNames, ", ") + ")",
			scope,
		}
		// Loud staging on the blank too: a scan the bounds cut short read only
		// part of the tree, so "no markers" would be a claim about files that
		// were never opened. A blank is a first-class result only while it is
		// trustworthy (adr-35), so it says what it did not read.
		question := "What did this project know was unfinished? Its source carries no work markers."
		if truncated.Any() || unread > 0 {
			searched = append(searched, truncated.notes()...)
			if unread > 0 {
				searched = append(searched, fmt.Sprintf("stopped at the %d-byte read budget; %d further file(s) were not read", budget, unread))
			}
			question = "What did this project know was unfinished? No work markers in the part of the tree the scan reached, and the scan did not reach all of it."
		}
		return blank(searched, question)
	}

	sources := []string{fmt.Sprintf("%d work marker(s) across %d file(s)", markers, files)}
	// Loud staging: a partial scan says so in its own evidence, so a rescuer
	// never mistakes a truncated count for the whole tree — naming which bound
	// cut it and what it cut (iss-133).
	sources = append(sources, truncated.notes()...)
	if unread > 0 {
		sources = append(sources, fmt.Sprintf("scan stopped at the %d-byte read budget; %d further file(s) were not read", budget, unread))
	}
	if markers > len(citations) {
		sources = append(sources, fmt.Sprintf("%d further marker(s) counted but not cited (citation cap %d)", markers-len(citations), maxMarkerCitations))
	}
	sources = append(sources, citations...)

	confidence := ConfidenceLow
	if markers >= convMarkerMediumConfidence {
		confidence = ConfidenceMedium
	}
	return Evidence{
		Status:     StatusPartial,
		Confidence: confidence,
		Sources:    dedupeSorted(sources),
	}
}

// convIssuesSource partially grounds "activity/issues" from a checked-in issues
// ledger, an issues directory, or issue templates. Blank when none is present.
type convIssuesSource struct{}

func (convIssuesSource) Section() Section { return "activity/issues" }
func (convIssuesSource) Tier() Tier       { return TierConventions }

func (convIssuesSource) Probe(ctx *SourceContext) Evidence {
	if p := ctx.FindFirst("ISSUES.md", "ISSUES"); p != "" {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{p},
		}
	}
	if ctx.IsDir("issues") {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{"issues/"},
		}
	}
	if ctx.IsDir(".github/ISSUE_TEMPLATE") {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceLow,
			Sources:    []string{".github/ISSUE_TEMPLATE"},
		}
	}
	return blank(
		[]string{"ISSUES.md", "issues/", ".github/ISSUE_TEMPLATE"},
		"What issues has this project tracked? No issue ledger or templates found.",
	)
}

// convChangelogNames are the conventional changelog spellings.
var convChangelogNames = []string{
	"CHANGELOG.md", "CHANGELOG", "CHANGELOG.rst", "CHANGES.md", "HISTORY.md",
}

// convWhatWorkedSource partially grounds "evidence/what-worked" from a CHANGELOG
// — what shipped and survived. Blank when no changelog exists.
type convWhatWorkedSource struct{}

func (convWhatWorkedSource) Section() Section { return "evidence/what-worked" }
func (convWhatWorkedSource) Tier() Tier       { return TierConventions }

func (convWhatWorkedSource) Probe(ctx *SourceContext) Evidence {
	if p := ctx.FindFirst(convChangelogNames...); p != "" {
		return Evidence{
			Status:     StatusPartial,
			Confidence: ConfidenceMedium,
			Sources:    []string{p},
		}
	}
	return blank(
		[]string{"CHANGELOG (" + strings.Join(convChangelogNames, ", ") + ")"},
		"What has this project shipped that worked? No changelog found.",
	)
}
