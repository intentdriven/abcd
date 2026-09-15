package guard

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SchemaVersion is the guard config schema this build implements. It is
// independent of the rules loader's schema: the two features keep separate
// switches so a rules kill switch can never silently disable a safety guard.
const SchemaVersion = 1

// Tier values, mirroring the record-lint severity family: a blocker refuses the
// command, a warn lets it through with the warning attached.
const (
	TierBlocker = "blocker"
	TierWarn    = "warn"
)

// Verdict is the decision a front door acts on.
type Verdict string

const (
	// VerdictAllow means no entry matched: run the command.
	VerdictAllow Verdict = "allow"
	// VerdictWarn means a warn-tier entry matched: run the command with the
	// warning surfaced.
	VerdictWarn Verdict = "warn"
	// VerdictBlock means a blocker-tier entry matched: refuse, and reply with
	// the successor as the block message.
	VerdictBlock Verdict = "block"
)

// Pattern is a command-position match expressed over shell tokens, never raw
// text. All declared constraints must hold for the pattern to fire.
type Pattern struct {
	// Command is the executable that must stand in command position. It is
	// compared against the segment's argv[0] by basename, so /bin/rm matches rm.
	Command string `json:"command"`
	// Subcommand, when set, must be the first non-flag argument (git push,
	// git reset). ValueFlags lets a global option that consumes the next token
	// be skipped while looking for it.
	Subcommand string `json:"subcommand,omitempty"`
	// Subcommand2, when set, must be the SECOND non-flag argument — the
	// two-level grammar `gh repo delete`, which one Subcommand cannot express
	// (`gh repo list` and `gh repo delete` share their first level, and only one
	// of them is a hazard).
	Subcommand2 string `json:"subcommand2,omitempty"`
	// ValueFlags are flags of Command that consume the FOLLOWING token, so the
	// scan for Subcommand steps over the value rather than reading it as the
	// subcommand (`git -C /repo push`).
	ValueFlags []string `json:"value_flags,omitempty"`
	// Flags are argument constraints. Each element is one alternation group
	// written with "|" ("--force|-f"); every group must be satisfied by some
	// argument token. A long alternative also matches its --flag=value form; a
	// single-letter short alternative also matches inside a bundled cluster
	// (-rf satisfies -r and -f).
	Flags []string `json:"flags,omitempty"`
	// FlagValues constrain what a flag is SET TO, not merely that it is present.
	// `gh api -X DELETE repos/owner/repo` deletes the repository and
	// `gh api -X GET repos/owner/repo` reads it, so an entry that could only ask
	// whether -X appears would have to choose between missing the first and
	// refusing the second. Every listed constraint must be satisfied.
	FlagValues []FlagValue `json:"flag_values,omitempty"`
	// ArgPaths constrain an operand that names a slash-separated resource path,
	// by its first segment AND its exact depth. Depth is the point: DELETE on
	// `repos/{owner}/{repo}` destroys the repository, while DELETE on a deeper
	// path under it (a branch ref, a release) is ordinary work that stays
	// allowed. Every listed constraint must be satisfied.
	ArgPaths []PathArg `json:"arg_paths,omitempty"`
	// ArgPrefixes constrain an OPERAND rather than a flag: every listed prefix
	// must be carried by some non-flag argument. It is what describes a hazard
	// spelled without any flag at all — `git push origin +main:main`, where the
	// leading `+` on the refspec is a force push by another name and Flags has
	// nothing to look at.
	ArgPrefixes []string `json:"arg_prefixes,omitempty"`
	// AfterCD, when true, additionally requires that some EARLIER command in the
	// same chain is a `cd` — the cd-chain structure (`cd scratch && rm -rf *`)
	// whose hazard is that a failed cd silently redirects the command. A nil
	// pointer means false; it is a pointer so a per-repo override can turn the
	// requirement off as well as on.
	AfterCD *bool `json:"after_cd,omitempty"`
}

// FlagValue is one flag-with-value constraint: some argument must set one of
// Flag's "|"-separated alternatives to one of Values.
type FlagValue struct {
	// Flag is an alternation group written with "|" ("-X|--method"), matching
	// the spelling Pattern.Flags uses.
	Flag string `json:"flag"`
	// Values are the accepted settings. They are compared case-insensitively:
	// an HTTP method's case is not what makes it destructive, and a constraint a
	// change of case walks past is not a constraint.
	Values []string `json:"values"`
}

// PathArg is one operand constraint for a slash-separated resource path.
type PathArg struct {
	// Root is the first path segment the operand must have ("repos").
	Root string `json:"root"`
	// Segments is the EXACT number of segments the operand must hold, counting
	// Root: three for `repos/{owner}/{repo}`. Exact rather than a maximum,
	// because it is the deeper paths — not the shallower ones — that stay
	// allowed.
	Segments int `json:"segments"`
}

// Fixtures is an entry's proof corpus: command lines it must fire on, and
// command lines it must not. The admission gate in fixtures_test.go enforces
// both sides plus the 40% known-good floor.
type Fixtures struct {
	KnownBad  []string `json:"known_bad,omitempty"`
	KnownGood []string `json:"known_good,omitempty"`
}

// Entry is one hazard. The map key in Registry.Entries is its id; ID is filled
// from that key on load, so the key is always authoritative — a declared id is
// accepted (the documented entry shape spells one out) and then overwritten,
// never used to rename the entry.
type Entry struct {
	ID        string   `json:"id,omitempty"`
	Pattern   Pattern  `json:"pattern"`
	Tier      string   `json:"tier"`
	Successor string   `json:"successor"`
	Why       string   `json:"why"`
	Fixtures  Fixtures `json:"fixtures,omitempty"`
}

// Registry is the merged, validated hazard model: the bundled defaults overlaid
// with the per-repo override.
type Registry struct {
	SchemaVersion int              `json:"schema_version"`
	Disabled      bool             `json:"disabled"`
	Entries       map[string]Entry `json:"entries"`
}

// Decision is what core returns for one candidate command. It carries no
// formatting and no exit status — the front door decides how to render it, and
// the hook shim decides what to do with a broken guard.
type Decision struct {
	Verdict Verdict `json:"verdict"`
	// EntryID, Tier, Successor, and Why describe the winning entry; they are
	// empty on an allow.
	EntryID   string `json:"entry_id,omitempty"`
	Tier      string `json:"tier,omitempty"`
	Successor string `json:"successor,omitempty"`
	Why       string `json:"why,omitempty"`
	// Message is the ready-to-surface sentence: the why plus the successor, so
	// the refusal itself teaches the safe form.
	Message string `json:"message,omitempty"`
	// Matches lists every entry the command tripped (blocker ids first, then
	// warn, each in id order), so a warn plane can surface all of them. It
	// includes the reserved synthetic id when an execute-a-string payload raised
	// an entry-less verdict.
	Matches []string `json:"matches,omitempty"`
	// Family and Reason describe an entry-less (synthetic) verdict — an
	// execute-a-string wrapper, or a brace group the tokenizer cannot expand:
	// the family that raised it and the plain-language reason. They are empty on
	// an ordinary registry match. The raw payload is deliberately not carried —
	// nothing reads it yet.
	Family string `json:"family,omitempty"`
	Reason string `json:"reason,omitempty"`
}

//go:embed defaults/guard.json
var defaultsJSON []byte

// defaultRegistry is parsed once at init; a malformed embedded asset is a build
// error surfaced as a panic (it can never happen at runtime).
var defaultRegistry = mustParseDefaults()

func mustParseDefaults() Registry {
	r, err := parse(defaultsJSON)
	if err != nil {
		panic("guard: bundled defaults are malformed: " + err.Error())
	}
	if err := Validate(r); err != nil {
		panic("guard: bundled defaults fail validation: " + err.Error())
	}
	return r
}

// Defaults returns a deep copy of the binary-bundled registry, safe for the
// caller to mutate.
func Defaults() Registry { return cloneRegistry(defaultRegistry) }

// parse decodes registry JSON and stamps each entry's ID from its map key.
// Unknown fields and trailing content are rejected: a misspelt key would
// otherwise be dropped in silence, and the blocker a repo believed it had
// declared would simply not exist (a safety config fails closed on
// unrecognised input).
func parse(data []byte) (Registry, error) {
	var r Registry
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return Registry{}, fmt.Errorf("%w: %v", ErrMalformedConfig, err)
	}
	if dec.More() {
		return Registry{}, fmt.Errorf("%w: trailing content after the JSON object", ErrMalformedConfig)
	}
	for id, e := range r.Entries {
		e.ID = id
		r.Entries[id] = e
	}
	return r, nil
}

// Validate checks the registry schema: the schema version, every entry id, and
// every entry's tier, pattern command, successor, and why. A per-repo override
// is validated AFTER merging, so an override can never produce an entry the
// bundled defaults would not have been allowed to ship.
func Validate(r Registry) error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: schema_version must be %d, got %d", ErrSchemaVersion, SchemaVersion, r.SchemaVersion)
	}
	ids := make([]string, 0, len(r.Entries))
	for id := range r.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !validEntryID(id) {
			return fmt.Errorf("%w: entry id %q must match [a-z0-9][a-z0-9-]*", ErrInvalidEntry, id)
		}
		// The guard speaks under reserved ids of its own for verdicts that have no
		// entry behind them. An entry claiming one would be indexed out of r.Entries
		// by a synthetic winner — a blank message — and would let a repo dress an
		// ordinary hazard up as the guard's own voice.
		if containsString(reservedEntryIDs, id) {
			return fmt.Errorf("%w: entry id %q is reserved for the guard's own entry-less verdicts", ErrInvalidEntry, id)
		}
		e := r.Entries[id]
		switch e.Tier {
		case TierBlocker, TierWarn:
		default:
			return fmt.Errorf("%w: entry %s has tier %q, want %q or %q", ErrUnknownTier, id, e.Tier, TierBlocker, TierWarn)
		}
		if strings.TrimSpace(e.Pattern.Command) == "" {
			return fmt.Errorf("%w: entry %s has no pattern command", ErrInvalidEntry, id)
		}
		// The command is compared against a token's BASENAME, and a subcommand is
		// only ever a non-flag argument, so a path, a phrase, or a leading dash
		// describes a pattern nothing can satisfy. Reject it here rather than
		// ship an entry that looks armed and never fires.
		if strings.ContainsAny(e.Pattern.Command, "/ \t") {
			return fmt.Errorf("%w: entry %s pattern command %q is a path or phrase and could never match a command name", ErrInvalidEntry, id, e.Pattern.Command)
		}
		if strings.HasPrefix(e.Pattern.Subcommand, "-") {
			return fmt.Errorf("%w: entry %s subcommand %q starts with a dash and could never match a non-flag argument", ErrInvalidEntry, id, e.Pattern.Subcommand)
		}
		if strings.HasPrefix(e.Pattern.Subcommand2, "-") {
			return fmt.Errorf("%w: entry %s subcommand2 %q starts with a dash and could never match a non-flag argument", ErrInvalidEntry, id, e.Pattern.Subcommand2)
		}
		// A refusal with no successor leaves its replacement in prose only, and
		// one with no why cannot teach — both are load-time rejections, as in
		// the record-lint banned_tokens family (iss-51).
		if strings.TrimSpace(e.Successor) == "" {
			return fmt.Errorf("%w: entry %s has no successor", ErrInvalidEntry, id)
		}
		if strings.TrimSpace(e.Why) == "" {
			return fmt.Errorf("%w: entry %s has no why", ErrInvalidEntry, id)
		}
		// An empty flag group can never be satisfied, so it would silently
		// defang the entry rather than fail loudly — the one failure mode a
		// guard must not have.
		for i, group := range e.Pattern.Flags {
			if !hasAlternative(group) {
				return fmt.Errorf("%w: entry %s flag group %d is empty and could never match", ErrInvalidEntry, id, i)
			}
		}
		// An empty argument prefix is carried by every operand, so the entry
		// would fire on anything that reached it: the over-blocking twin of the
		// empty flag group, and as invisible in the file.
		for i, prefix := range e.Pattern.ArgPrefixes {
			if strings.TrimSpace(prefix) == "" {
				return fmt.Errorf("%w: entry %s argument prefix %d is empty and would match every argument", ErrInvalidEntry, id, i)
			}
			// A prefix constrains an OPERAND, and `operandIndexes` never
			// returns a token that starts with a dash — it reads those as
			// flags. A dashed
			// prefix therefore describes an argument nothing can be: the silent
			// defang again, one field along. A flag belongs in Flags.
			if strings.HasPrefix(prefix, "-") {
				return fmt.Errorf("%w: entry %s argument prefix %q starts with a dash and could never match a non-flag argument", ErrInvalidEntry, id, prefix)
			}
		}
		// A flag-value constraint with no flag, or with no accepted value, can
		// never be satisfied — the silent defang again, one field along.
		for i, fv := range e.Pattern.FlagValues {
			if !hasAlternative(fv.Flag) {
				return fmt.Errorf("%w: entry %s flag-value constraint %d names no flag and could never match", ErrInvalidEntry, id, i)
			}
			if !hasAnyValue(fv.Values) {
				return fmt.Errorf("%w: entry %s flag-value constraint %d accepts no value and could never match", ErrInvalidEntry, id, i)
			}
		}
		// A path constraint with no root would depth-limit every operand that
		// happened to look like a path, and one with no depth describes no path
		// at all.
		for i, pa := range e.Pattern.ArgPaths {
			if strings.TrimSpace(pa.Root) == "" {
				return fmt.Errorf("%w: entry %s path constraint %d names no root segment", ErrInvalidEntry, id, i)
			}
			if pa.Segments < 1 {
				return fmt.Errorf("%w: entry %s path constraint %d wants %d segments; a path has at least one", ErrInvalidEntry, id, i, pa.Segments)
			}
			// The root is compared against the FIRST segment of an operand split
			// on "/", so a root that carries a slash is not a segment and could
			// never be one. Depth is what the Segments field is for.
			if strings.Contains(pa.Root, "/") {
				return fmt.Errorf("%w: entry %s path root %q holds a slash and could never match a single path segment; use segments for depth", ErrInvalidEntry, id, pa.Root)
			}
		}
	}
	return nil
}

// hasAlternative reports whether a flag group holds at least one usable
// alternative, so neither "" nor "|" can pass as a constraint.
func hasAlternative(group string) bool {
	for _, alt := range strings.Split(group, "|") {
		if strings.TrimSpace(alt) != "" {
			return true
		}
	}
	return false
}

// hasAnyValue reports whether a flag-value constraint accepts at least one
// usable setting, so neither an empty list nor a list of blanks can pass.
func hasAnyValue(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func validEntryID(id string) bool {
	if id == "" {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '-' && i > 0:
		default:
			return false
		}
	}
	return true
}

// Check evaluates one candidate command line against the registry. A disabled
// registry allows everything (the escape is committed and reviewable, never an
// in-session override). A command line the tokenizer cannot split is an error,
// not an allow — core names the failure and the caller decides what it means.
func (r Registry) Check(command string) (Decision, error) {
	if r.Disabled {
		return Decision{Verdict: VerdictAllow}, nil
	}
	segs, err := tokenize(command)
	if err != nil {
		return Decision{}, err
	}
	// Expand every execute-a-string payload ONCE, here, before the entry loop —
	// never inside a per-entry callee (that path went quadratic). Every entry then
	// sees the payload segments for free, and any payload the guard cannot read
	// raises a synthetic (entry-less) signal folded in by severity below.
	segs, signals := expandPayloads(segs)

	// git rewrites its own subcommand from configuration carried IN the command
	// line, and the operand walk was built to step exactly those values over
	// (gh-299). The pre-pass reads them and appends the command git would
	// actually run, so the entries that already exist match it. It runs after the
	// payload expansion so a git command inside an `sh -c` payload is reached too
	// (GHSA-m2r8-fx7r-rq34).
	aliasSegs, aliasSignals := r.expandGitAliases(segs)
	segs = aliasSegs
	signals = append(signals, aliasSignals...)

	// A brace group the tokenizer could not expand is folded in the same way,
	// and AFTER the payload expansion so a group hidden inside an inspectable
	// payload counts too. One signal is enough however many segments carry a
	// group: the verdict is the whole command's, and repeating the same lesson
	// per segment would only pad Matches.
	for _, s := range segs {
		if s.braceGroup {
			signals = append(signals, braceExpansionBlockSignal())
			break
		}
	}
	// An unterminated here-document is the same shape: a tokenizer state that
	// is bash grammar, not a fault, which the hook would otherwise fail open on.
	for _, s := range segs {
		if s.heredocUnterminated {
			signals = append(signals, heredocBlockSignal())
			break
		}
	}

	ids := make([]string, 0, len(r.Entries))
	for id := range r.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Tier 1: the registry match at command position. matchedSeg records WHICH
	// segments fired, because Tier 2's gate is per segment — a line-wide gate would
	// let one warn-tier command disarm the fail-safe for everything after it
	// (adr-42 decision 4).
	var blockers, warns []string
	matchedSeg := make([]bool, len(segs))
	for _, id := range ids {
		p := r.Entries[id].Pattern
		hit := false
		for i, s := range segs {
			if !matchSegment(p, s) {
				continue
			}
			if p.AfterCD != nil && *p.AfterCD && !precededByCD(segs[:i], s.chain) {
				continue
			}
			matchedSeg[i] = true
			hit = true
		}
		if hit {
			if r.Entries[id].Tier == TierBlocker {
				blockers = append(blockers, id)
			} else {
				warns = append(warns, id)
			}
		}
	}

	// Tier 2: the position-agnostic fail-safe over the segments Tier 1 left
	// unmatched. Warns only, and folded into the same severity pools below.
	signals = append(signals, r.speculate(segs, matchedSeg, ids)...)

	// The first synthetic block and warn (id order is not meaningful for the
	// entry-less verdicts, so the first encountered wins its pool).
	var synBlock, synWarn *payloadSignal
	for i := range signals {
		switch {
		case signals[i].verdict == VerdictBlock && synBlock == nil:
			synBlock = &signals[i]
		case signals[i].verdict == VerdictWarn && synWarn == nil:
			synWarn = &signals[i]
		}
	}

	// Merge by SEVERITY POOL, not a single "registry outranks synthetic" rule: a
	// synthetic block never hides behind a registry warn, and a registry blocker
	// still outranks a synthetic warn.
	blockPool := len(blockers) > 0 || synBlock != nil
	warnPool := len(warns) > 0 || synWarn != nil
	if !blockPool && !warnPool {
		return Decision{Verdict: VerdictAllow}, nil
	}
	// Every entry-less id that CONTRIBUTED is listed, not just the two that won
	// their pools. The merge keeps one signal per pool for the message, which is
	// right — the first is as good a lesson as the fifth — but Matches is the
	// record of what fired, and a Tier 2 hit that loses the slot to an unrelated
	// payload warn would vanish from it entirely. That is exactly what the warn
	// rate is measured from, so the blind spot would have hidden itself.
	matches := append(append([]string(nil), blockers...), warns...)
	for i := range signals {
		if id := signals[i].entryID(); !containsString(matches, id) {
			matches = append(matches, id)
		}
	}

	// Within the winning pool a concrete registry entry supplies the message and
	// successor (its lesson is more specific); a synthetic verdict supplies them
	// only when it is the sole member of the winning pool. A synthetic id must NOT
	// index r.Entries — that yields a zero Entry and a blank message — so the
	// winner construction branches on it.
	if blockPool {
		if len(blockers) > 0 {
			return decisionFromEntry(VerdictBlock, r.Entries[blockers[0]], matches), nil
		}
		return syntheticDecision(VerdictBlock, *synBlock, matches), nil
	}
	if len(warns) > 0 {
		return decisionFromEntry(VerdictWarn, r.Entries[warns[0]], matches), nil
	}
	return syntheticDecision(VerdictWarn, *synWarn, matches), nil
}

// decisionFromEntry builds the decision a concrete registry match produces.
func decisionFromEntry(v Verdict, e Entry, matches []string) Decision {
	return Decision{
		Verdict:   v,
		EntryID:   e.ID,
		Tier:      e.Tier,
		Successor: e.Successor,
		Why:       e.Why,
		Message:   message(v, e),
		Matches:   matches,
	}
}

// syntheticDecision builds the decision an entry-less execute-a-string verdict
// produces. It populates Why/Successor (the fields writeGuardDecision reads) as
// well as Family/Reason and the ready-to-surface Message, so the human report is
// never blank.
func syntheticDecision(v Verdict, sig payloadSignal, matches []string) Decision {
	tier := TierBlocker
	if v == VerdictWarn {
		tier = TierWarn
	}
	return Decision{
		Verdict:   v,
		EntryID:   sig.entryID(),
		Tier:      tier,
		Successor: sig.successor,
		Why:       sig.reason,
		Message:   syntheticMessage(v, sig),
		Matches:   matches,
		Family:    sig.family,
		Reason:    sig.reason,
	}
}

// syntheticMessage renders the sentence for an entry-less verdict, in the same
// shape as message() so a front door treats the two uniformly.
func syntheticMessage(v Verdict, sig payloadSignal) string {
	lead := "Blocked"
	if v == VerdictWarn {
		lead = "Warning"
	}
	return fmt.Sprintf("%s by the abcd guard (%s): %s Run instead: %s", lead, sig.entryID(), sig.reason, sig.successor)
}

// message renders the sentence a front door surfaces: what happened, why in
// plain language, and what to run instead. The refusal is the lesson, so the
// successor is always part of it.
func message(v Verdict, e Entry) string {
	lead := "Blocked"
	if v == VerdictWarn {
		lead = "Warning"
	}
	return fmt.Sprintf("%s by the abcd guard (%s): %s Run instead: %s", lead, e.ID, e.Why, e.Successor)
}

func cloneRegistry(r Registry) Registry {
	out := Registry{SchemaVersion: r.SchemaVersion, Disabled: r.Disabled}
	if r.Entries != nil {
		out.Entries = make(map[string]Entry, len(r.Entries))
		for id, e := range r.Entries {
			out.Entries[id] = cloneEntry(e)
		}
	}
	return out
}

func cloneEntry(e Entry) Entry {
	out := e
	out.Pattern.ValueFlags = append([]string(nil), e.Pattern.ValueFlags...)
	out.Pattern.Flags = append([]string(nil), e.Pattern.Flags...)
	out.Pattern.ArgPrefixes = append([]string(nil), e.Pattern.ArgPrefixes...)
	out.Pattern.ArgPaths = append([]PathArg(nil), e.Pattern.ArgPaths...)
	out.Pattern.FlagValues = cloneFlagValues(e.Pattern.FlagValues)
	if e.Pattern.AfterCD != nil {
		v := *e.Pattern.AfterCD
		out.Pattern.AfterCD = &v
	}
	out.Fixtures.KnownBad = append([]string(nil), e.Fixtures.KnownBad...)
	out.Fixtures.KnownGood = append([]string(nil), e.Fixtures.KnownGood...)
	return out
}

// cloneFlagValues deep-copies flag-value constraints: each one holds its own
// Values slice, so the copy goes a level further than the other pattern fields.
func cloneFlagValues(in []FlagValue) []FlagValue {
	if in == nil {
		return nil
	}
	out := make([]FlagValue, len(in))
	for i, fv := range in {
		out[i] = FlagValue{Flag: fv.Flag, Values: append([]string(nil), fv.Values...)}
	}
	return out
}
