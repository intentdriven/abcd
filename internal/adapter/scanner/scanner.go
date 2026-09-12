package scanner

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// BundleFile is the minimal descriptor ScanBundle needs. It is defined here
// (rather than importing the launch package) so the scanner has NO dependency on
// launch — that would create an import cycle, since launch's dry-run orchestrator
// imports the scanner. The launch package adapts its IncludedFile slice to this.
type BundleFile struct {
	LogicalPath  string
	ResolvedPath string
}

// ScanResult is the outcome of scanning a bundle.
type ScanResult struct {
	Findings          []Finding `json:"findings"`
	FilesScanned      int       `json:"files_scanned"`
	HardFails         int       `json:"hard_fails"`
	Unavailable       bool      `json:"unavailable"`
	UnavailableReason string    `json:"unavailable_reason,omitempty"`
	// Unscanned lists bundle files that were present but could NOT be covered:
	// unreadable, over the byte-scan cap, a non-regular or symlinked leaf, or
	// classified binary by the content sniff (e.g. a leading-NUL file) WITHOUT
	// a reviewed skip explaining it. They are the fail-closed coverage gap —
	// the launch gate refuses on them, so a crafted binary cannot smuggle
	// unscanned content into a source bundle (GHSA-5mmm-3whv-3rqp).
	Unscanned []string `json:"unscanned,omitempty"`
	// UnscannedWhy carries, per Unscanned path, the reason it could not be
	// covered, so an asset over the cap is distinguishable from an I/O error.
	UnscannedWhy map[string]string `json:"unscanned_why,omitempty"`
	// ScannedBinary lists bundle files that matched the reviewed skip sets (a
	// binary media extension, a skip filename, or a skip fragment) AND whose
	// name is on the plaintext allow-list (plaintextNames): a skip-listed file
	// known to carry no compressed region, so the byte scan covers all of it.
	// The raw bytes run through the byte rules — every secret rule, the
	// harness-leak rules, and the long-literal identity rules — with no
	// format decoded. A skip exempts a file from the short/generic identity
	// rules that are noise on bytes; it is never an unverified allow, because
	// a file that enters the payload enters it with its bytes, whatever its
	// name says (GHSA-9wv7-88w3-f77m).
	ScannedBinary []string `json:"scanned_binary,omitempty"`
	// ContentUnverified lists every other skip-listed bundle file — the
	// DEFAULT for the byte branch, because a closed list of compressed
	// formats would label whatever it missed as verified. An archive, but
	// equally a PNG (deflate IDAT and zTXt), a JPEG entropy stream, a PDF
	// FlateDecode stream, an mp4 box, a database blob, a packed executable,
	// or any extension a repo's pii.json adds to skip_extensions. They get
	// the same byte scan as ScannedBinary (cheap, and it still catches
	// plaintext such as an uncompressed tar's entries or PNG tEXt metadata),
	// but a compressed region is invisible to it, so they are NOT counted as
	// content-verified and the report says so rather than claiming coverage
	// it does not have. It is what remains AFTER the decoder (container.go)
	// has had its turn: the formats abcd cannot decode, the streams that
	// would not inflate, and the ones a decode bound refused. Every entry
	// carries its reason in ContentUnverifiedWhy and the format detected from
	// its magic bytes in ContentFormat. They do not refuse on their own.
	ContentUnverified []string `json:"content_unverified,omitempty"`
	// ContentDecoded lists skip-listed bundle files whose container format the
	// scanner DECODES (container.go): the archive was opened or the compressed
	// regions inflated, under the decode bounds, and every decoded region was
	// scanned with the same byte rules. A file reaches this list only when the
	// walk could ACCOUNT for the whole of its content — a zip whose entries do
	// not tile it, a tar with bytes after its end marker, a stream with a
	// trailer, an entry the walk could not read, a region only part of which
	// reads as text, a structural field carrying a container signature: each of
	// those stays in ContentUnverified with the reason, because the claim this
	// list makes is that the bytes were read, not that the parts that were read
	// looked fine. The residuals the claim does NOT cover are written down at
	// container.go's coverStructuralField. So "decoded and clean" and "could
	// not be read" are two tiers and never one green (iss-2608291832160371).
	// The rules applied are the byte rules, as everywhere on this branch: a
	// text file inside an archive is scanned as bytes, not with the full
	// prose-and-identity set it would get loose in the payload.
	ContentDecoded []string `json:"content_decoded,omitempty"`
	// ContentUnverifiedWhy carries, per ContentUnverified path, why the content
	// could not be read: no decoder for the detected format, an unrecognised
	// format, a stream that would not inflate, an entry that is itself opaque,
	// or one of the decode bounds (output budget, nesting depth, entry count)
	// refusing a bomb.
	ContentUnverifiedWhy map[string]string `json:"content_unverified_why,omitempty"`
	// ContentFormat carries, per byte-branch path, the format detected from the
	// file's MAGIC BYTES — never its extension, so a zip renamed .png reports
	// "zip" and is decoded as one, which the record names as the defect in the
	// old name-keyed label.
	ContentFormat map[string]string `json:"content_format,omitempty"`
}

// maxBinaryScanBytes caps how much of a skip-listed (binary) bundle file the
// byte scan reads. It equals the sibling privacy lint's maxScanBytes
// (internal/core/repolint/rule_privacy.go), restated here because repolint
// imports this package and the constant cannot be shared without a cycle. The
// cap understates cost by ~20-30x: ScanText's percent-decode pre-pass keeps an
// 8-byte position map per input byte plus a decoded copy per line, so a
// newline-free 16 MiB file measured ~460 MiB and 10 s. A file over the cap is
// a loud Unscanned refusal rather than a quiet skip: memory stays bounded and
// coverage stays honest.
const maxBinaryScanBytes = 4 << 20

// plaintextNames is the allow-list of skip-listed names known to carry no
// compressed region, so a byte scan covers all of their content and they may
// be reported as ScannedBinary. The polarity is deliberate: the byte branch
// defaults to ContentUnverified, and only a name listed here is promoted.
// Keyed on the final path element's extension as filepath.Ext reports it, so a
// skip FILENAME such as .gitignore (a dotfile, whose Ext is the whole name)
// qualifies. Nothing on defaultSkipExtensions qualifies: .ico can embed PNG,
// .wav can carry a compressed codec, .tar holds compressed entries, .pyc is
// marshalled bytecode, and .so/.dylib/.dll/.exe/.sqlite/.db can all hold
// packed sections or blobs. A config-added skip extension (.jar, say) reaches
// this branch like any other and lands in ContentUnverified by default.
var plaintextNames = toSet([]string{".gitignore"})

// Config is the on-disk scanner configuration (the per-repo pii.json override
// shape). Only the consumed fields are modelled.
type Config struct {
	SkipDirs           []string              `json:"skip_dirs"`
	SkipPathFragments  []string              `json:"skip_path_fragments"`
	SkipExtensions     []string              `json:"skip_extensions"`
	SkipFilenames      []string              `json:"skip_filenames"`
	Patterns           map[string]patternDef `json:"patterns"`
	IdentitySeverities map[string]Severity   `json:"identity_severities"`
}

// patternDef is one pattern definition in a config override.
type patternDef struct {
	Regex           string   `json:"regex"`
	Kind            string   `json:"kind"`
	Label           string   `json:"label"`
	Severity        Severity `json:"severity"`
	CaseInsensitive bool     `json:"case_insensitive"`
	Suggestion      string   `json:"suggestion"`
}

// Scanner holds the merged config, compiled patterns and probed identity for a
// repo. Construct it with New.
type Scanner struct {
	patterns       []Pattern
	identity       Identity
	identSev       map[string]Severity
	skipExtensions map[string]struct{}
	skipFilenames  map[string]struct{}
	skipFragments  []string
	unavailable    bool
	unavailReason  string
}

// defaultSkipExtensions / defaultSkipFilenames mirror the bundled pii.json
// binary/noise skip sets.
var (
	// defaultSkipExtensions are media/archive/artefact extensions treated as
	// reviewed skips. It deliberately omits .svg AND .lock: both are text — an
	// SVG is XML that can carry a secret in a comment or attribute, and a
	// lockfile is the same class, carrying secrets in a pinned URL, a path, or a
	// comment (GHSA-5mmm-3whv-3rqp; iss-2608270655490511). Each is scanned like
	// any other text file rather than waved through by extension.
	defaultSkipExtensions = []string{
		".png", ".jpg", ".jpeg", ".gif", ".pdf", ".ico", ".webp",
		".mp3", ".mp4", ".mov", ".webm", ".wav",
		".zip", ".tar", ".gz", ".tgz", ".bz2", ".xz", ".7z",
		".pyc", ".pyo", ".so", ".dylib", ".dll", ".exe",
		".sqlite", ".db",
	}
	defaultSkipFilenames = []string{".DS_Store", "Thumbs.db", ".gitignore"}
	defaultSkipFragments = []string{".abcd/.work.local/logs/pii-scan/", ".abcd/.work.local/logs/audit-history/"}
	repoConfigRelPath    = filepath.Join(".abcd", "config", "pii.json")
)

// maxScannerConfigBytes caps the per-repo pii.json override (trust boundary).
// It matches the cap guard.Load applies to its own committed override: both are
// short, hand-written JSON documents, and the cap exists to bound a planted file
// rather than to fit a real one.
const maxScannerConfigBytes = 256 * 1024

// New builds a Scanner for repoRoot: it starts from the built-in secret set and
// identity floors, merges the per-repo .abcd/config/pii.json override when
// present (enforcing the severity floor), and probes identity. If the per-repo
// config exists but cannot be read or parsed, the scanner is marked unavailable
// (fail-closed): New still returns a usable value with no error, and ScanBundle
// surfaces Unavailable=true.
func New(repoRoot string) (*Scanner, error) {
	s := &Scanner{
		patterns:       DefaultPatterns(),
		identity:       ProbeIdentity(repoRoot),
		identSev:       DefaultIdentitySeverities(),
		skipExtensions: toSet(defaultSkipExtensions),
		skipFilenames:  toSet(defaultSkipFilenames),
		skipFragments:  append([]string(nil), defaultSkipFragments...),
	}

	// The override lives in the repo working tree, so untrusted content can put
	// something that is not a regular file at this path — and New is reached
	// automatically by the SessionEnd hook's history capture, which holds the
	// history repo lock across the call. A bare os.ReadFile here blocks forever
	// on a FIFO and reads unbounded through a symlink to an endless device,
	// wedging transcript capture for the repo (iss-202). Every sibling
	// trust-boundary config reader — guard.Load, the banlist private store,
	// lint's config — guards the read; this one is now no different.
	//
	// The read is contained in an os.Root rather than guarded by a hand-rolled
	// ancestor check: O_NOFOLLOW binds the final path component only, and this
	// config sits two directories down (.abcd/config/pii.json), so a symlinked
	// .abcd OR .abcd/config would redirect a path-based read out of the tree.
	// os.Root resolves every component inside the root and refuses the escape
	// wherever it sits — guard.Load's single ancestor check is complete for its
	// one-deep .abcd/guard.json, but copying it here covered half the path.
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		s.unavailable = true
		s.unavailReason = "per-repo scanner config unreadable: the repo root cannot be opened for contained reads"
		return s, nil
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, repoConfigRelPath, maxScannerConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // no override — built-in defaults stand
		}
		// Every remaining failure is fail-closed, but the reason is specific:
		// a degraded scanner that cannot say WHY is the defect iss-203 tracks.
		switch {
		case errors.Is(err, fsutil.ErrNotRegular):
			s.unavailReason = "per-repo scanner config is not a regular file (a symlinked leaf is refused): " + repoConfigRelPath
		case errors.Is(err, fsutil.ErrTooBig):
			s.unavailReason = "per-repo scanner config exceeds the " + strconv.Itoa(maxScannerConfigBytes) + "-byte cap: " + repoConfigRelPath
		default:
			// Includes a path component that is a symlink or otherwise escapes
			// the repo root — os.Root refuses the escape wherever it sits.
			s.unavailReason = "per-repo scanner config unreadable: " + repoConfigRelPath
		}
		s.unavailable = true
		return s, nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		s.unavailable = true
		s.unavailReason = "per-repo scanner config is not valid JSON: " + repoConfigRelPath
		return s, nil
	}
	if err := s.mergeConfig(cfg); err != nil {
		s.unavailable = true
		s.unavailReason = err.Error()
		return s, nil
	}
	return s, nil
}

// Unavailable reports whether the scanner is in the fail-closed degraded state
// (the per-repo config exists but is unreadable, invalid JSON, or carries a bad
// override regex) and, if so, a human reason. A write-time redactor MUST consult
// this before trusting ScanText/Redact: unlike ScanBundle, those entry points
// cannot signal degradation in-band, so a caller that skips this check would
// sanitise with a silently weakened pattern set. Mirrors ScanBundle's guard.
func (s *Scanner) Unavailable() (bool, string) {
	return s.unavailable, s.unavailReason
}

// mergeConfig layers a per-repo override onto the built-in defaults, enforcing
// the severity floor on both bundled patterns and identity kinds.
func (s *Scanner) mergeConfig(cfg Config) error {
	for _, e := range cfg.SkipExtensions {
		// An empty extension entry matches every extensionless file (LICENSE,
		// Makefile, …) and would silently drop them from coverage — drop the
		// entry, not the files.
		if strings.TrimSpace(e) == "" {
			continue
		}
		s.skipExtensions[strings.ToLower(e)] = struct{}{}
	}
	for _, f := range cfg.SkipFilenames {
		s.skipFilenames[f] = struct{}{}
	}
	for _, frag := range cfg.SkipPathFragments {
		// A blank or slash-only fragment is a substring of every logical path,
		// so strings.Contains would skip the whole bundle and zero the scan's
		// coverage — reject it rather than let it disable the scanner.
		if strings.Trim(frag, "/ \t\r\n") == "" {
			continue
		}
		s.skipFragments = append(s.skipFragments, frag)
	}

	floors := defaultPatternFloors()
	byName := map[string]int{}
	for i, p := range s.patterns {
		byName[p.Name] = i
	}
	// Deterministic order over override pattern names.
	names := make([]string, 0, len(cfg.Patterns))
	for name := range cfg.Patterns {
		names = append(names, name)
	}
	sort.Strings(names)

	// Pass one — compile every NEW-name override regex before mutating anything.
	// A BUNDLED pattern's regex is part of the non-negotiable floor and is NOT
	// replaceable: swapping it for a never-match regex would neuter detection while
	// the severity clamp still reports hard_fail — the exact downgrade the floor
	// forbids, achieved through the regex field instead of the severity field. So
	// a bundled name may only adjust severity/label/suggestion (new detection must
	// use a new pattern name, which merges additively). Compiling all new-name
	// regexes up front also makes the merge all-or-nothing, so a later invalid
	// regex can no longer leave an earlier override half-applied on s.patterns.
	compiled := map[string]*regexp.Regexp{}
	for _, name := range names {
		if _, bundled := floors[name]; bundled {
			continue
		}
		def := cfg.Patterns[name]
		if def.Regex == "" {
			// A NEW pattern with no regex detects nothing, and skipping it
			// silently is the one weakening a broken config can still achieve
			// without being reported: an adversarial review demonstrated a
			// pull request blanking a repo's custom detector's regex, after
			// which `abcd lint` reported "conforms" at exit 0 over a file the
			// detector had been catching. Bundled names never reach this loop
			// (the floors check above continues past them), so refusing here
			// cannot affect a config that only adjusts a built-in pattern's
			// label or severity — it binds exactly the case where an empty
			// regex is meaningless (iss-203's contract, closed here).
			return errUnreadable("pattern " + name + " has no regex")
		}
		expr := def.Regex
		if def.CaseInsensitive {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			// A malformed override regex is a config fault → fail-closed.
			return errUnreadable("pattern " + name + " has an invalid regex")
		}
		compiled[name] = re
	}

	// Pass two — apply. Bundled names adjust in place (regex/kind stay built-in);
	// new names merge additively.
	for _, name := range names {
		def := cfg.Patterns[name]
		if floor, bundled := floors[name]; bundled {
			i := byName[name]
			sev := def.Severity
			if sev == "" {
				sev = floor
			}
			s.patterns[i].Severity = applyFloor(sev, floor)
			if def.Label != "" {
				s.patterns[i].Label = def.Label
			}
			if def.Suggestion != "" {
				s.patterns[i].Suggestion = def.Suggestion
			}
			continue
		}
		re, ok := compiled[name]
		if !ok {
			continue // new pattern with no regex — nothing to add
		}
		sev := def.Severity
		if !isValidSeverity(sev) {
			sev = defaultPatternSeverity
		}
		kind := def.Kind
		if kind == "" {
			kind = name
		}
		s.patterns = append(s.patterns, Pattern{
			Name: name, Kind: kind, Label: def.Label, Re: re,
			Severity: sev, Suggestion: def.Suggestion,
		})
		byName[name] = len(s.patterns) - 1
	}

	for kind, sev := range cfg.IdentitySeverities {
		if !isValidSeverity(sev) {
			continue
		}
		if floor, ok := s.identSev[kind]; ok {
			s.identSev[kind] = applyFloor(sev, floor)
		} else {
			s.identSev[kind] = sev
		}
	}
	return nil
}

// errUnreadable is a tiny error type for config faults.
type errUnreadable string

func (e errUnreadable) Error() string { return string(e) }

// leadingFlagGroup matches an inline flag group (e.g. `(?i)`) at the very
// start of a compiled regexp's source, so adjacencyProbe can look past it
// for a leading \b — two of the bundled network patterns (lanHostRe,
// deviceHostRe) compile with `(?i)\b...`, not a bare `\b...`.
var leadingFlagGroup = regexp.MustCompile(`^\(\?[a-zA-Z]+\)`)

// adjacencyProbe returns a version of re, ANCHORED to match only at the very
// start of whatever string it is run against, with any leading \b stripped
// (after an inline flag group, if present). Every bundled secret pattern
// anchors its own start on a leading \b; when a second fixed-length token
// immediately abuts a first with no separating byte, the byte just before it
// is itself a word character (the first token's last byte), so \b can never
// hold there and the second token is silently never matched. scanAllPatterns
// runs this probe ONLY at a byte offset where a token is already known to
// begin — right after a prior match, or at a junction a greedy match ran past
// (stolenJunctions) — a hit there is anchored by the adjacency itself, not by
// \b, so dropping the anchor cannot introduce a false match anywhere else in
// the line. The `\A` anchor here is what makes that probe O(1) rather than
// O(remaining line length): without it, FindStringIndex would search the
// entire remainder of the line for a match anywhere, then discard everything
// not starting at offset 0 — turning one probe per candidate junction into a
// full rescan of the rest of the line, which a single long line (a minified
// asset, a base64 blob) can turn into a multi-second hang per call.
func adjacencyProbe(re *regexp.Regexp) *regexp.Regexp {
	flags, body := probeParts(re)
	anchored, err := regexp.Compile(flags + `\A(?:` + body + `)`)
	if err != nil {
		return re
	}
	return anchored
}

// junctionProbe is adjacencyProbe's UNANCHORED counterpart, and ONE regexp for
// the whole pattern set rather than one per pattern: every pattern's
// boundary-free body, alternated, free to match anywhere in the string it is
// run against. It is a candidate GENERATOR for stolenJunctions and nothing
// else — it answers "could ANY token begin at this offset", which is exactly
// what a backward search needs to narrow 512 offsets down to a handful, and
// answers it in one linear pass instead of one pass per pattern (the per-match
// cost of the backward search is otherwise multiplied by the size of the
// pattern set, which a scanner cannot pay on a long line). Nothing is ever
// added off a junctionProbe hit: every offset it yields is re-tested with the
// ANCHORED probe before a match is recorded, so the "an unanchored
// boundary-free match cannot be trusted on its own" reasoning above still
// holds, and a generator that over-produces only costs time.
//
// Each pattern's own inline flag group is rewritten into a SCOPED one
// (`(?i)x` → `(?i:x)`) so a case-insensitive pattern cannot fold the case of
// every alternative after it. If the alternation somehow will not compile, the
// fallback is a regexp that matches at every offset: slower, never blinder —
// the generator may over-produce, never under-produce.
func junctionProbe(patterns []Pattern) *regexp.Regexp {
	alts := make([]string, 0, len(patterns))
	for _, cp := range patterns {
		flags, body := probeParts(cp.Re)
		if flags != "" {
			alts = append(alts, flags[:len(flags)-1]+`:`+body+`)`)
			continue
		}
		alts = append(alts, `(?:`+body+`)`)
	}
	combined, err := regexp.Compile(strings.Join(alts, `|`))
	if err != nil {
		return regexp.MustCompile(`(?s).`)
	}
	return combined
}

// probeParts splits a compiled pattern's source into its leading inline flag
// group (if any) and the rest with a leading \b stripped.
func probeParts(re *regexp.Regexp) (flags, body string) {
	src := re.String()
	flags = leadingFlagGroup.FindString(src)
	return flags, strings.TrimPrefix(src[len(flags):], `\b`)
}

// patMatch is one pattern match location in a line, plus which pattern (by
// index into the patterns/probes slices passed to scanAllPatterns) it
// belongs to.
type patMatch struct {
	patIdx     int
	start, end int
}

// maxAdjacencyProbeWindow bounds how much of the line a single adjacency
// probe attempt (see scanAllPatterns) is run against. `\A` bounds the probe
// to ONE start position, but not the cost of that one attempt: two bundled
// patterns (net_lan_hostname, net_device_hostname) carry their own unbounded
// internal quantifier (`[a-z0-9-]*`) and, on a long following run with no
// terminator, scan all the way to the end of it before failing — turning
// many candidate junctions on one long line into an O(matches × remaining
// line length) cost. Slicing the probe's input to a small fixed window
// bounds every single attempt to O(window) regardless of pattern internals
// (Go's RE2-based regexp package guarantees no worse than linear cost in
// the length of the string it is run against), which is what actually
// caps the whole function at O(line length) rather than quadratic. The
// window is generously larger than every bundled FIXED-LENGTH pattern's
// real match (the longest, github_pat_finegrained, is 93 bytes) and than a
// realistic DNS hostname (~253 bytes) — the class of match this function
// exists to recover.
//
// It is the FIRST window, not a cap. A fixed cap made every match end that
// reached it an artefact of the edge rather than of the text: a probe's own
// trailing `\b` was satisfied by the artificial end of the slice, reporting a
// token the full line does not contain (iss-189), and an open-ended token
// longer than the window was recorded with an artificial end that put the next
// probe and the next junction search mid-token, breaking the chain that unwinds
// a run of abutting tokens (iss-190). gallopingFind keeps the bound where it
// pays — a FAILING attempt never grows — and doubles the window only while a
// match is still running into its edge, which costs a constant multiple of that
// match's own length.
//
// Keeping the bound on a failing attempt keeps one standing cost with it, and
// it is the half of iss-185's trade that survives: a pattern whose earliest
// REQUIRED structural marker falls past this first window matches nothing at
// all there, so such a token is still missed ENTIRELY rather than merely
// truncated — jwt_shaped needs two literal '.' separators, and a long enough
// header pushes both out of reach. Growing the window for a failure is what
// cannot be afforded (it is the O(matches × remaining line length) hang this
// constant exists to prevent), so that case is a known limit, not a defect of
// the search that feeds the probe.
const maxAdjacencyProbeWindow = 512

// maxAdjacencyBacktrack bounds how far BACK from an open-ended match's
// reported end stolenJunctions looks for the junction that match over-ran. It
// mirrors maxAdjacencyProbeWindow for the same reason and on the same terms:
// what has to stay bounded is not the length of a match (a base64 blob or a
// minified line is legitimately hundreds of kilobytes) but the work done per
// match, and a fixed window is the only thing that caps it independently of
// how long the match is. The bound is not arbitrary: over-consumption can only
// reach as far as the following token's own leading bytes that fall inside the
// first pattern's trailing character class, so the junction is at most one
// bundled token's length behind the reported end — 512 bytes clears every
// bundled pattern's longest real token by a wide margin. A junction further
// back than that would need a crafted multi-kilobyte token, which no bundled
// pattern can produce.
const maxAdjacencyBacktrack = 512

// matcher is the single regexp operation the adjacency probes need. Production
// always passes a compiled *regexp.Regexp; taking the interface is what lets the
// cost guard hand gallopingFind a counting stand-in and assert the doubling
// schedule exactly rather than time it.
type matcher interface {
	FindStringIndex(s string) []int
}

// gallopingFind runs re against a window of line that begins at at, and returns
// the leftmost match's offsets RELATIVE TO at (nil for no match). The window
// ends maxAdjacencyProbeWindow past base and DOUBLES that extent only while the
// answer is still an artefact of the window's own edge — the match ran right
// into it — stopping the moment the answer can no longer change.
//
// base is the offset the extent is measured from. It is at for the anchored
// adjacency probe; the junction search behind a match passes the match's end
// instead, because that search walks its start offset backwards while its
// forward reach has to stay measured from the end it is looking behind.
//
// Three stop conditions, and each one says the slice end is NOT an artefact:
// the window already reaches the real end of the line (whatever it found is
// what the full line holds); the match ended short of the edge (real content,
// not the edge, ended it); or nothing matched inside a bounded window. That
// last one is the cost bound the fixed window was introduced for and is
// deliberately kept: a FAILING attempt is what two bundled patterns'
// unbounded internal quantifiers make expensive, and growing the window for a
// failure would restore exactly the O(matches × remaining line length) cost.
// Growing it for a match that is genuinely still running does not: doubling
// sums to under twice the final window, and the final window is under twice the
// match, so one probe costs a small constant multiple of the match's OWN length
// — the amortised class the top-level unbounded match already pays.
func gallopingFind(re matcher, line string, at, base int, budget *int) []int {
	for w := maxAdjacencyProbeWindow; ; w *= 2 {
		hi := base + w
		if hi > len(line) {
			hi = len(line)
		}
		loc := re.FindStringIndex(line[at:hi])
		if hi == len(line) || loc == nil || at+loc[1] < hi {
			return loc
		}
		// Growing is what the shared budget pays for; when it is gone the probe
		// keeps the fixed window it already has. See gallopBudget.
		if *budget < hi-at {
			return loc
		}
		*budget -= hi - at
	}
}

// gallopBudget returns the total growth gallopingFind may spend on one line.
//
// Per probe the doubling is cheap — a constant multiple of the match's own
// length — but the SUM across a line is not automatically bounded, and the
// unbounded sum is a cost class, not a constant. Removing the fixed cap also
// removed the cap on how long a probe-recovered match may be, and every
// recovered match is then re-validated up to maxAdjacencyBacktrack times by
// stolenJunctions at O(match length) each. A line crafted so that Θ(n) probe
// positions each start a Θ(n)-long match therefore turns a linear scan
// quadratic: measured at 1.3s / 5.0s / 19.1s over 14KB / 29KB / 59KB, where the
// fixed window was 0.6s / 1.3s / 2.5s. That is a resource-exhaustion cliff on
// input this scanner exists to read from strangers.
//
// The budget restores the class without giving the fix back. It is a multiple
// of the line's own length, so a single long token — the shape the galloping
// probe exists for — is captured whole with room to spare, while a line
// engineered to grow the window at every junction exhausts it after a few and
// reverts to exactly the fixed-window behaviour, which is bounded and was never
// worse than correct-but-truncated.
func gallopBudget(line string) int {
	return 4*len(line) + 8*maxAdjacencyProbeWindow
}

// scanAllPatterns returns every match of every pattern in line, plus any
// further token — of the SAME pattern or a DIFFERENT one — that immediately
// abuts an already-found match with no separating byte (see adjacencyProbe).
// It probes two kinds of junction. A FIXED-LENGTH pattern's match always ends
// exactly where a genuinely adjacent token begins, so probing every pattern at
// that one byte offset recovers it (iss-185). A pattern whose quantifier is
// open-ended (`{36,}`) instead greedily consumes the following token's own
// leading bytes whenever they fall inside its trailing character class, which
// puts the true junction BEFORE the reported end and out of the forward
// probe's reach — that one needs stolenJunctions' bounded backward search
// (iss-188). The over-long match is left as it is rather than trimmed back to
// the junction: it only ever over-reports the first token's span, which
// over-redacts by a few bytes, while the recovered second token is what closes
// the leak.
//
// Neither probe helps a real secret preceded by content that is not itself a
// match found here — e.g. unrecognized filler text, or an identity-derived
// match from matchers.findings — since there is no existing patMatch for the
// probe to run after; that is the same pre-existing \b-boundary limitation
// every bundled pattern already accepts elsewhere in this package, not
// something this function claims to close.
func scanAllPatterns(patterns []Pattern, probes []*regexp.Regexp, junctions *regexp.Regexp, line string) []patMatch {
	var all []patMatch
	// One growth budget for the whole line, shared by every probe and the
	// junction search: see gallopBudget.
	budget := gallopBudget(line)
	seen := map[patMatch]bool{}
	add := func(m patMatch) {
		if seen[m] {
			return
		}
		seen[m] = true
		all = append(all, m)
	}
	// probeAt records every pattern whose boundary-free variant matches
	// starting exactly at byte offset at.
	probeAt := func(at int) {
		for j := range patterns {
			m := gallopingFind(probes[j], line, at, at, &budget)
			if m == nil || m[0] != 0 || m[1] == 0 {
				continue
			}
			add(patMatch{j, at, at + m[1]})
		}
	}
	for i, cp := range patterns {
		for _, loc := range cp.Re.FindAllStringIndex(line, -1) {
			add(patMatch{i, loc[0], loc[1]})
		}
	}
	// The loop reads a growing slice on purpose: a token recovered at one
	// junction is itself greedy and can have over-run the next one, so a run of
	// three or more abutting tokens unwinds one junction per iteration.
	for qi := 0; qi < len(all); qi++ {
		m := all[qi]
		probeAt(m.end)
		for _, cut := range stolenJunctions(probes[m.patIdx], junctions, line, m, &budget) {
			probeAt(cut)
		}
	}
	return all
}

// stolenJunctions returns the byte offsets INSIDE m at which a second token
// really begins — the junctions m's own greedy quantifier ran past. A cut is
// reported only when both halves hold: line[m.start:cut] is still a whole match
// for m's own pattern (so the minimum-length bound of its quantifier is still
// satisfied and the cut is a real token end, not an arbitrary byte), and some
// pattern's junction probe can start a token there.
//
// The work done per match is bounded by a constant factor independent of the
// REST OF THE LINE — proportional to the match's OWN length, since each
// candidate is validated by re-running the probe over line[m.start:cut], but
// never to what follows the match. That is what keeps a legitimately huge match
// (a base64 blob, a minified line) off a performance cliff: the whole scan
// stays linear in line length rather than quadratic. Three things do that. A
// pattern that cannot match one byte shorter has a rigid length, so its
// reported end already IS its junction and the whole search is skipped after
// one test — that is how a fixed-length pattern is told from an open-ended one
// without hand-annotating either, and a pattern added later inherits the
// classification for free. The search window is capped at maxAdjacencyBacktrack
// behind the end, so the number of candidates is capped too — the FORWARD reach
// past the end is the galloping one (gallopingFind), which grows only while a
// candidate hit is still running into its own edge, so a junction whose hit is
// longer than one fixed window is found rather than truncated away. And
// candidates come from the single COMBINED
// junction probe over that window, not from re-probing every byte in it with
// every pattern — the difference between one window-bounded search per
// candidate and a full window scan per pattern per match.
func stolenJunctions(probe, junctions *regexp.Regexp, line string, m patMatch, budget *int) []int {
	if m.end-m.start < 2 || !wholeMatch(probe, line[m.start:m.end-1]) {
		return nil
	}
	lo := m.end - maxAdjacencyBacktrack
	if lo < m.start+1 {
		lo = m.start + 1
	}
	var cuts []int
	// Walk the window one hit at a time, resuming one byte past each hit's
	// START, and stop at the first hit that is not inside m: a hit at or past
	// m.end is already the forward probe's business, so a match whose over-run
	// can only be a few bytes never pays for a scan of the whole window.
	// Resuming past a hit's END would be wrong: junctions is UNANCHORED, so a
	// hit can SPAN the true junction — start before it and end after it —
	// without starting at it. Skipping to that hit's end then steps over the
	// real junction, which nothing else revisits, and the second token's tail
	// survives redaction raw again (the very failure iss-188 closed). One byte
	// per rejected candidate costs at most maxAdjacencyBacktrack passes of a
	// window-bounded search per match — a constant, still independent of the
	// line's length.
	for off := lo; off < m.end; {
		loc := gallopingFind(junctions, line, off, m.end, budget)
		if loc == nil {
			break
		}
		cut := off + loc[0]
		if cut >= m.end {
			break
		}
		if wholeMatch(probe, line[m.start:cut]) {
			cuts = append(cuts, cut)
		}
		off = cut + 1
	}
	return cuts
}

// wholeMatch reports whether s is entirely one match of probe, an anchored
// boundary-free adjacencyProbe. The anchor makes the match start at 0, so only
// its end has to reach the end of s.
func wholeMatch(probe *regexp.Regexp, s string) bool {
	loc := probe.FindStringIndex(s)
	return loc != nil && loc[1] == len(s)
}

// ScanText scans text for every secret pattern and identity-derived match,
// returning findings sorted deterministically. It is pure: identity, patterns
// and severities are all passed in.
func ScanText(text string, id Identity, patterns []Pattern, id2sev map[string]Severity, file string) []Finding {
	return scanText(text, id, patterns, id2sev, file, false)
}

// scanText is ScanText with the byte switch: bytes says the text is a raw
// blob decoded as a string, where the home-path anchor's leading half does
// not apply (identityMatchers.bytes).
func scanText(text string, id Identity, patterns []Pattern, id2sev map[string]Severity, file string, bytes bool) []Finding {
	if id2sev == nil {
		id2sev = DefaultIdentitySeverities()
	}
	matchers := newIdentityMatchers(id)
	matchers.bytes = bytes
	probes := make([]*regexp.Regexp, len(patterns))
	for i, cp := range patterns {
		probes[i] = adjacencyProbe(cp.Re)
	}
	junctions := junctionProbe(patterns)
	var findings []Finding
	lineno := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		lineno++
		findings = append(findings, matchers.findings(line, lineno, id2sev, file)...)
		for _, m := range scanAllPatterns(patterns, probes, junctions, line) {
			cp := patterns[m.patIdx]
			matched := line[m.start:m.end]
			if cp.Skip != nil && cp.Skip(matched) {
				continue
			}
			if cp.SkipAt != nil && cp.SkipAt(line, m.start, m.end) {
				continue
			}
			findings = append(findings, Finding{
				File: file, Line: lineno, Column: m.start + 1, Kind: cp.Kind,
				Severity: cp.Severity, Snippet: snippet(line), Matched: matched,
				Suggested: cp.Suggestion, line: line,
			})
		}
		// Percent-decode pre-pass (gh-370): a URL-encoded delimiter (%3D, %2F,
		// %22) leaves a hex word-char before a literal token, defeating the
		// leading \b so the raw scan above never fires. Scan bounded
		// percent-decoded copies of the line and map every hit back to its raw
		// byte span, so Redact masks the live token where it sits on disk.
		findings = append(findings, decodedLineFindings(patterns, probes, junctions, matchers, id2sev, line, lineno, file)...)
	}
	findings = dedupFindings(findings)
	sealSnippets(findings)
	sortFindings(findings)
	return findings
}

// sealSnippets masks EVERY finding's matched token out of the shared source line
// each finding on that line carries, so a serialized snippet cannot leak a
// sibling secret found on the same line. Finding.MarshalJSON only masks a
// finding's OWN token; without this, two secrets on one line each leak the other.
// Findings on one line always come from the same ScanText call (same file), so
// grouping by line number is sufficient. Masking is by BYTE SPAN (Column is the
// 1-based byte offset, len(Matched) the length), not substring: two matches that
// partially overlap — where a substring rewrite would leave the shorter match's
// non-overlapping tail raw — are both masked span-exactly.
func sealSnippets(findings []Finding) {
	byLine := map[int][]int{}
	for i := range findings {
		byLine[findings[i].Line] = append(byLine[findings[i].Line], i)
	}
	for _, idxs := range byLine {
		if len(idxs) < 2 {
			continue // a lone finding: MarshalJSON already masks its own token
		}
		src := ""
		for _, i := range idxs {
			if findings[i].line != "" {
				src = findings[i].line
				break
			}
		}
		if src == "" {
			continue
		}
		sealed := sealLine(src, findings, idxs)
		for _, i := range idxs {
			findings[i].line = sealed
		}
	}
}

// sealSpan is a byte span to mask, plus whether it must be masked WHOLE (an
// identity or network kind, where a head/tail fingerprint is itself the leak).
type sealSpan struct {
	start, end int
	whole      bool
}

// sealLine masks the byte spans of every finding in idxs out of src. A byte
// covered by exactly one span shows that token's head/tail fingerprint; a byte in
// an OVERLAP of two or more spans is forced to '*' (so no token's raw bytes leak
// through a partner's fingerprint window); an uncovered byte is kept raw. Byte
// spans are authoritative, so overlapping matches cannot leak. O(len(src) + spans).
func sealLine(src string, findings []Finding, idxs []int) string {
	b := []byte(src)
	n := len(b)
	if n == 0 {
		return src
	}
	cov := make([]int, n+1) // difference array → per-byte coverage count in O(n+k)
	spans := make([]sealSpan, 0, len(idxs))
	for _, i := range idxs {
		start := findings[i].Column - 1
		end := start + len(findings[i].Matched)
		if start < 0 {
			start = 0
		}
		if end > n {
			end = n
		}
		if start >= end {
			continue
		}
		spans = append(spans, sealSpan{start, end, maskedWhole(findings[i].Kind)})
		cov[start]++
		cov[end]--
	}
	out := make([]byte, n)
	copy(out, b)
	// Star the middle of each span, keeping its head/tail fingerprint bytes.
	for _, s := range spans {
		fingerprintSpan(out, b, s.start, s.end, s.whole)
	}
	// Any byte covered by two or more spans is an overlap: force '*' regardless of
	// a fingerprint head/tail char, so no token's raw bytes survive in an overlap.
	run := 0
	for i := 0; i < n; i++ {
		run += cov[i]
		if run >= 2 {
			out[i] = '*'
		}
	}
	return string(out)
}

// fingerprintSpan masks out[start:end] to a token fingerprint: the first keepHead
// and last keepTail RUNES are kept (enough to triage which credential), the middle
// starred; a short span is fully starred. It is the byte-level mirror of
// maskSecret and measures the threshold in runes, not bytes, so a short multi-byte
// identity value (email, username) that is under 16 runes but at or over 16 bytes
// is fully starred here just as maskSecret fully stars it — and head/tail are kept
// on rune boundaries so a multi-byte rune is never split into invalid UTF-8. A
// whole span keeps nothing: maskMatched's trade for an identifier, applied to the
// shared source line so a sibling finding's snippet cannot leak it either.
func fingerprintSpan(out, src []byte, start, end int, whole bool) {
	const keepHead, keepTail, fingerprintBelow = 3, 2, 16
	// Star the whole span first; head/tail fingerprint runes are restored below.
	for j := start; j < end; j++ {
		out[j] = '*'
	}
	if whole {
		return // an identifier: its head and tail are what re-identify it
	}
	if utf8.RuneCount(src[start:end]) < fingerprintBelow {
		return // short value: fully starred, mirroring maskSecret
	}
	headEnd := start
	for r := 0; r < keepHead; r++ {
		_, sz := utf8.DecodeRune(src[headEnd:end])
		headEnd += sz
	}
	tailStart := end
	for r := 0; r < keepTail; r++ {
		_, sz := utf8.DecodeLastRune(src[start:tailStart])
		tailStart -= sz
	}
	for j := start; j < headEnd; j++ {
		out[j] = src[j] // keep head raw — the fingerprint
	}
	for j := tailStart; j < end; j++ {
		out[j] = src[j] // keep tail raw — the fingerprint
	}
}

// ScanBundle scans the resolved content of every bundle file, reading
// ResolvedPath and reporting under LogicalPath. A file on the reviewed skip
// sets (extension, filename, fragment) is read through the guarded, capped
// primitive and its bytes scanned with the byte rules (scanBytes), reported
// under ScannedBinary (a plaintext allow-listed name) or ContentUnverified; any
// other file is sniffed (null byte + UTF-8) and scanned with the full rule
// set, or surfaced in Unscanned (with UnscannedWhy) when it cannot be. If the
// scanner is unavailable (config unreadable), it returns Unavailable=true and
// scans nothing (fail-closed).
func (s *Scanner) ScanBundle(files []BundleFile) (ScanResult, error) {
	if s.unavailable {
		return ScanResult{Unavailable: true, UnavailableReason: s.unavailReason}, nil
	}
	var res ScanResult
	unscanned := func(logical, why string) {
		res.Unscanned = append(res.Unscanned, logical)
		if res.UnscannedWhy == nil {
			res.UnscannedWhy = map[string]string{}
		}
		res.UnscannedWhy[logical] = why
	}
	secrets := secretPatterns(s.patterns)
	for _, f := range files {
		if s.skipByName(f.LogicalPath) || s.skipByFragment(f.LogicalPath) {
			// A reviewed skip exempts the file from the short/generic identity
			// rules, never from the secret, harness-leak or long-literal
			// identity rules: its bytes still ship, so its bytes are still
			// scanned (GHSA-9wv7-88w3-f77m). The read is guarded and capped — a
			// symlinked, non-regular or oversized leaf is a loud coverage gap,
			// not a quiet skip.
			data, err := fsutil.ReadGuarded(f.ResolvedPath, maxBinaryScanBytes)
			if err != nil {
				unscanned(f.LogicalPath, guardedReadWhy(err))
				continue
			}
			res.Findings = append(res.Findings, s.scanBytes(data, secrets, f.LogicalPath)...)
			if _, ok := plaintextNames[strings.ToLower(filepath.Ext(f.LogicalPath))]; ok {
				res.ScannedBinary = append(res.ScannedBinary, f.LogicalPath)
				continue
			}
			// The raw scan covers the plaintext regions only, so the decoder
			// gets its turn: it keys on the file's MAGIC BYTES, opens the
			// formats abcd knows under the decode bounds, and scans what comes
			// out with the same byte rules. A file whose content was read
			// whole is content-verified; anything the decoder could not read
			// stays ContentUnverified exactly as before, saying which it is
			// and why (iss-2608291832160371).
			format, decodedFindings, decoded, why := s.decodeContent(data, secrets, f.LogicalPath)
			res.Findings = append(res.Findings, decodedFindings...)
			if res.ContentFormat == nil {
				res.ContentFormat = map[string]string{}
			}
			res.ContentFormat[f.LogicalPath] = format
			if decoded {
				res.ContentDecoded = append(res.ContentDecoded, f.LogicalPath)
				continue
			}
			res.ContentUnverified = append(res.ContentUnverified, f.LogicalPath)
			if res.ContentUnverifiedWhy == nil {
				res.ContentUnverifiedWhy = map[string]string{}
			}
			res.ContentUnverifiedWhy[f.LogicalPath] = why
			continue
		}
		data, err := os.ReadFile(f.ResolvedPath)
		if err != nil {
			// An unreadable file is skipped, not fatal — but surfaced in Unscanned
			// with the same visibility as a binary-skipped file, so a read-skipped
			// file cannot silently vanish from the bundle's coverage.
			unscanned(f.LogicalPath, "unreadable")
			continue
		}
		if !isText(data) {
			unscanned(f.LogicalPath, "binary content without a reviewed skip")
			continue
		}
		res.FilesScanned++
		res.Findings = append(res.Findings, ScanText(string(data), s.identity, s.patterns, s.identSev, f.LogicalPath)...)
	}
	for _, fnd := range res.Findings {
		if fnd.Severity == SeverityHardFail {
			res.HardFails++
		}
	}
	sortFindings(res.Findings)
	// Zero-coverage sentinel: a bundle with files but none scanned with the
	// FULL rule set (every file skip-listed or unscannable, however that came
	// about — an over-broad skip config, an all-binary tree) means the prose
	// and identity rules effectively did not run, whatever the byte-rule pass
	// over the skip-listed files found. Fail closed so the launch path refuses
	// rather than publishing while reporting "would publish"; the reason
	// counts the byte-scanned files and the unread ones separately so neither
	// is called something it is not.
	if len(files) > 0 && res.FilesScanned == 0 && !res.Unavailable {
		byteScanned := len(res.ScannedBinary) + len(res.ContentUnverified) + len(res.ContentDecoded)
		res.Unavailable = true
		res.UnavailableReason = "scanner covered zero of " + strconv.Itoa(len(files)) +
			" bundle files with the full rule set: " + strconv.Itoa(byteScanned) + " of " +
			strconv.Itoa(len(files)) + " bundle files byte-scanned only, " +
			strconv.Itoa(len(res.Unscanned)) + " could not be read"
	}
	return res, nil
}

// guardedReadWhy names the reason a guarded read refused a skip-listed file,
// path-free, so the Unscanned entry can say it.
func guardedReadWhy(err error) string {
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		return "over the " + strconv.Itoa(maxBinaryScanBytes>>20) + " MiB scan cap"
	case errors.Is(err, fsutil.ErrNotRegular):
		return "not a regular file"
	case errors.Is(err, syscall.ELOOP):
		return "symlink"
	}
	return "unreadable"
}

// scanBytes runs the byte-rule set over a skip-listed file's raw bytes: the
// secret patterns (secretPatterns) plus the identity rules whose literal is
// the caller's own and long enough that a chance collision with binary
// content is negligible — the home path, the email, and a real name that is
// multi-word or 8+ characters. A home path in PDF /Creator, an /Author stamp,
// or a session URL in PNG tEXt metadata is the same release-blocking leak it
// is in prose, and renaming deck.md to deck.pdf must not change the verdict.
// The short/generic identity kinds (local_username, github_username,
// home_path_other, a short single-token real_name) are dropped by
// byteScanPolicy after the scan — home_path_other runs unguarded inside the
// identity matcher, so switching it off by identity value is not possible —
// unless the repo's pii.json raised that kind above its built-in default,
// which is a judgement the byte scan honours.
func (s *Scanner) scanBytes(data []byte, secrets []Pattern, logical string) []Finding {
	// GitRemoteUsername and the emails travel with the names so the matcher
	// tells a public handle from a real name exactly as it does on text
	// (isPublicHandle): a user.name equal to the public handle is reported as
	// github_username (dropped here), never promoted to a hard-fail real_name
	// by the file's extension (iss-283).
	long := Identity{
		HomePath: s.identity.HomePath, GitUserEmail: s.identity.GitUserEmail,
		GitUserName: s.identity.GitUserName, GitRemoteUsername: s.identity.GitRemoteUsername,
		OtherGitUserNames: s.identity.OtherGitUserNames, OtherGitUserEmails: s.identity.OtherGitUserEmails,
	}
	all := scanText(string(data), long, secrets, s.identSev, logical, true)
	out := all[:0]
	for _, f := range all {
		if s.byteScanDrops(f) {
			continue
		}
		out = append(out, f)
	}
	return out
}

// bytePolicy is what the byte scan does with a finding of one identity kind.
type bytePolicy int

const (
	// bytePolicyKeep: the finding stands on bytes as it does on text.
	bytePolicyKeep bytePolicy = iota
	// bytePolicyDrop: noise on bytes; dropped unless the repo raised the kind.
	bytePolicyDrop
	// bytePolicyKeepLongLiteral: kept when the matched literal is multi-word or
	// at least byteScanLongLiteral bytes, dropped when it is a short token.
	bytePolicyKeepLongLiteral
)

// byteScanLongLiteral is the length from which a single-token real name no
// longer collides with binary content by chance.
const byteScanLongLiteral = 8

// byteScanPolicy is the EXPLICIT byte-scan classification of every identity
// kind. It is a table, not a derivation: a kind that is not listed reports
// ok=false, and TestEveryIdentityKindIsClassifiedForBytes fails until it is
// classified, so a new kind cannot land on either side silently. The default
// for an unlisted kind is keep — the same polarity secretPatterns gives a new
// hard-fail pattern — so an omission fails closed rather than open.
func byteScanPolicy(kind string) (bytePolicy, bool) {
	switch kind {
	case kindHomeSelf, kindRealEmail:
		return bytePolicyKeep, true
	case kindRealName:
		return bytePolicyKeepLongLiteral, true
	case kindLocalUser, kindGithubUser, kindHomeOther:
		return bytePolicyDrop, true
	case kindNetIPv4, kindNetIPv6, kindNetMAC, kindNetLANHost, kindNetDeviceHost:
		// Never produced on bytes — the network patterns are not in the byte
		// rule set — but classified so the table is total.
		return bytePolicyDrop, true
	}
	return bytePolicyKeep, false
}

// byteScanDrops applies byteScanPolicy to one finding, honouring a repo-raised
// severity: when pii.json made the kind stricter than its built-in default,
// the repo has judged that kind a leak wherever it sits, and the drop yields.
func (s *Scanner) byteScanDrops(f Finding) bool {
	policy, ok := byteScanPolicy(f.Kind)
	if !ok {
		return false
	}
	if floor, has := DefaultIdentitySeverities()[f.Kind]; has && isDowngrade(floor, s.identSev[f.Kind]) {
		return false
	}
	switch policy {
	case bytePolicyDrop:
		return true
	case bytePolicyKeepLongLiteral:
		return len(f.Matched) < byteScanLongLiteral && strings.IndexFunc(f.Matched, unicode.IsSpace) < 0
	}
	return false
}

// secretPatterns returns the subset of patterns that is meaningful on raw
// bytes: every hard-fail rule that is not an identity kind — the token and
// private-key rules, the harness-leak rules, and a hard-fail override of the
// same shape. A session URL is a long literal with no chance collision, and
// AGENTS.md declares the harness-leak class as one definition reaching launch,
// so both its rules run on bytes: the session URL is caught anywhere on a
// line, and the attribution footer fires on bytes exactly as it does on text
// when a line holds it on its own (PDF and PNG metadata are line-shaped), which
// is the wanted behaviour. IsIdentityKind already covers the network kinds, so
// no separate network clause is needed. A username, an address or a hostname
// inside binary bytes is noise, not a leak, and the identity rules a byte
// scan does keep are added by scanBytes through the identity value, not here.
func secretPatterns(patterns []Pattern) []Pattern {
	out := make([]Pattern, 0, len(patterns))
	for _, p := range patterns {
		if p.Severity != SeverityHardFail || IsIdentityKind(p.Kind) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (s *Scanner) skipByName(logical string) bool {
	ext := strings.ToLower(filepath.Ext(logical))
	if _, ok := s.skipExtensions[ext]; ok {
		return true
	}
	base := path_base(logical)
	_, ok := s.skipFilenames[base]
	return ok
}

func (s *Scanner) skipByFragment(logical string) bool {
	l := filepath.ToSlash(logical)
	for _, frag := range s.skipFragments {
		if strings.Contains(l, frag) {
			return true
		}
	}
	return false
}

// path_base returns the final path element of a POSIX/OS logical path.
func path_base(p string) string {
	p = filepath.ToSlash(p)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// isText sniffs the first 8KB: a null byte or invalid UTF-8 means binary. When
// the file is longer than the sniff window the cut can land mid-rune; a dangling
// partial trailing rune (at most UTFMax-1 bytes) is trimmed before validating, so
// a valid multibyte file whose rune straddles the boundary is not misread as
// binary. A genuinely invalid encoding still fails.
func isText(data []byte) bool {
	const sniff = 8192
	chunk := data
	truncated := false
	if len(chunk) > sniff {
		chunk = chunk[:sniff]
		truncated = true
	}
	for _, b := range chunk {
		if b == 0 {
			return false
		}
	}
	if truncated {
		for i := 0; i < utf8.UTFMax-1 && len(chunk) > 0 && !utf8.Valid(chunk); i++ {
			chunk = chunk[:len(chunk)-1]
		}
	}
	return utf8.Valid(chunk)
}

func toSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, it := range items {
		m[it] = struct{}{}
	}
	return m
}

// sortFindings orders findings deterministically: file → line → column → kind.
func sortFindings(f []Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].File != f[j].File {
			return f[i].File < f[j].File
		}
		if f[i].Line != f[j].Line {
			return f[i].Line < f[j].Line
		}
		if f[i].Column != f[j].Column {
			return f[i].Column < f[j].Column
		}
		if f[i].Kind != f[j].Kind {
			return f[i].Kind < f[j].Kind
		}
		return f[i].Matched < f[j].Matched
	})
}

// MaxBinaryScanBytesForTest exposes the byte-scan cap so the sibling privacy
// lint's tests can hold the two caps equal (repolint imports this package, so
// the constant cannot be shared the other way).
func MaxBinaryScanBytesForTest() int { return maxBinaryScanBytes }
