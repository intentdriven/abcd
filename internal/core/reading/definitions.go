package reading

// definitions.go is the definition locator: it resolves a reading position to
// the definition file that holds that position's object, question, blindness
// core, regime and item shape, and reports what the file states.
//
// The regime is the DEFINITION's property, not the payload's. That is the whole
// point of reading it from here: a run's supply regime is resolved by looking up
// the position's definition and reading its `regime:` key, so no operand an
// operator types can set one. The file's stated value is what comes back — this
// package does not substitute the position table's value for it, because a
// locator that answered from the table would make the frontmatter decorative and
// a drift between the two undetectable.
//
// A drift is REFUSED instead, which is the detection that sentence asks for: a
// file stating a legal regime under the wrong position resolves to nothing at
// all rather than to a confident wrong licence. Refusing and substituting are
// different acts, and only the second would make the frontmatter decorative.
//
// The assembler never reads a definition into a bundle: `agents` is denied
// structurally to every assembly (see deny.go and the exclusion floor). Locating
// a definition and passing one to a reading are different acts.

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Definition is one cold-reading definition as this binary resolves it: where it
// is, which position it holds, the regime it states, and the hash of the bytes
// that were read. The hash is the instrument's identity for a run — two runs at
// one position under one definition hash read under the same instructions.
type Definition struct {
	// Position is the position the definition holds.
	Position Position `json:"position"`
	// Regime is the supply regime the definition states, verbatim.
	Regime string `json:"regime"`
	// Path is the definition file, repo-relative and slash-separated.
	Path string `json:"path"`
	// SHA256 is the hex digest of the whole file, frontmatter included.
	SHA256 string `json:"sha256"`
}

// DefinitionPath returns p's definition file, repo-relative. The filename is
// derived from the position rather than looked up, which is what lets a run's
// position resolve to its definition by construction.
func DefinitionPath(p Position) string {
	return DefinitionsDir + "/" + definitionPrefix + string(p) + ".md"
}

// LoadDefinition resolves p to its definition under repoRoot and reports what the
// file states. The root is a parameter rather than a discovered value so a caller
// — the ingest verb, or a test over a temporary tree — decides which repository
// is being read.
//
// A missing file is returned wrapped around os.ErrNotExist, so a caller that
// treats absence as a state can say so with errors.Is. Every other fault is a
// fault: a definition present but silent about its position or its regime is
// worse than an absent one, because it reports an instrument that is not there.
func LoadDefinition(repoRoot string, p Position) (Definition, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return Definition{}, errors.New("reading: locating a definition needs a repository root")
	}
	pos, err := ParsePosition(string(p))
	if err != nil {
		return Definition{}, fmt.Errorf("reading: %w", err)
	}
	// Read through the repository root, not through a joined path. The hash this
	// returns is the definition half of an instrument identity — the thing that
	// makes "two runs claiming the same instrument are provably the same" a proof
	// — and a symlinked ancestor under agents/ would have it hash a file that is
	// not in this repository at all. The read itself is harmless; the CLAIM about
	// what was read is not, and a claim about an unknown artefact is worse than
	// no claim.
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Definition{}, fmt.Errorf("reading: opening the repository root: %w", err)
	}
	defer root.Close()

	rel := DefinitionPath(pos)
	raw, err := fsutil.ReadGuardedInRoot(root, rel, MaxFileBytes)
	if err != nil {
		return Definition{}, fmt.Errorf("reading: the %s definition at %s: %w", pos, rel, err)
	}

	lines := strings.Split(string(raw), "\n")
	// A duplicated key is refused rather than resolved: Fields keeps the first
	// occurrence, so two `regime:` lines would silently pick one, and which one
	// is not a question a definition should leave open.
	for _, dup := range frontmatter.Duplicates(lines) {
		if dup.Key == "position" || dup.Key == "regime" {
			return Definition{}, fmt.Errorf("reading: %s states %q more than once; "+
				"a definition states its position and its regime exactly once", rel, dup.Key)
		}
	}
	fields := frontmatter.Fields(lines)
	stated := scalar(fields["position"].Value)
	regime := scalar(fields["regime"].Value)

	if stated == "" {
		return Definition{}, fmt.Errorf("reading: %s states no 'position' in its frontmatter; "+
			"a definition names the position it holds", rel)
	}
	if stated != string(pos) {
		return Definition{}, fmt.Errorf("reading: %s states position %q, which is not the position its "+
			"filename holds (%s); the filename is how a run resolves to its definition", rel, stated, pos)
	}
	if regime == "" {
		return Definition{}, fmt.Errorf("reading: %s states no 'regime' in its frontmatter; the regime is "+
			"the definition's property, and a run that cannot read one here has no regime at all", rel)
	}
	if !knownRegime(regime) {
		return Definition{}, fmt.Errorf("reading: %s states regime %q; the set is closed: %s",
			rel, regime, strings.Join(knownRegimes(), ", "))
	}
	// Membership and AGREEMENT are two questions, and a legal regime under the
	// wrong position is still the wrong licence. The locator is the one thing
	// that claims to resolve a position to its regime, and its caller is the
	// ingest verb, whose whole purpose is to catch a reading that exceeded the
	// licence it read under — so a resolver returning a confidently wrong answer
	// would make that gate enforce the wrong licence, silently, which is exactly
	// the failure shape the gate exists to close (iss-2608311145258479).
	if want := issueschema.ReadingRegime(string(pos)); regime != want {
		return Definition{}, fmt.Errorf("reading: %s states regime %q under position %s, which carries "+
			"the %s regime; the definition states the licence a reading reads under, and a definition "+
			"stating another position's licence is refused rather than resolved", rel, regime, pos, want)
	}

	return Definition{Position: pos, Regime: regime, Path: rel, SHA256: sha256Hex(raw)}, nil
}

// LoadDefinitions resolves every position's definition under repoRoot, in the
// order Positions renders them, skipping the positions whose file is absent.
//
// Absence is a state: a repository with no definitions has none, and reporting
// that is the status render's job. A definition that IS present and does not
// parse is a fault, and stops the whole resolution — the alternative is a render
// that quietly lists three instruments where four were meant.
func LoadDefinitions(repoRoot string) ([]Definition, error) {
	out := []Definition{}
	for _, p := range Positions() {
		def, err := LoadDefinition(repoRoot, p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		out = append(out, def)
	}
	return out, nil
}

// scalar reads a frontmatter value the way the record's own readers do: whether
// the value is double-quoted at all is the caller's question, and the shared
// decoder is handed the INNER text once the pair has been stripped. Handing it
// the raw value keeps the quote characters, and `position: "detection"` then
// refuses itself with a message reading detection against detection.
//
// This is the THIRD copy of the strip-then-decode idiom — capture's reader and
// record-lint's schema gate hold the other two — and it belongs in
// internal/core/frontmatter beside Unquote rather than in any of the three.
// Consolidating it is captured; this call site cannot wait for that, because
// without it the locator refuses well-formed definitions.
func scalar(value string) string {
	v := strings.TrimSpace(value)
	if len(v) >= 2 && strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
		return frontmatter.Unquote(v[1 : len(v)-1])
	}
	return v
}

// knownRegime reports whether token is one of the four supply regimes. The
// vocabulary is read off the reading-record schema rather than restated here.
//
// It answers MEMBERSHIP alone. Agreement with the position is the second
// question, asked separately by LoadDefinition against the same schema table, so
// a definition naming a regime that exists but is not its own is refused with a
// message about the position rather than about the vocabulary.
func knownRegime(token string) bool {
	for _, p := range issueschema.ReadingPositions {
		if p.Regime == token {
			return true
		}
	}
	return false
}

// knownRegimes lists the closed regime vocabulary for a refusal message.
func knownRegimes() []string {
	out := make([]string, 0, len(issueschema.ReadingPositions))
	for _, p := range issueschema.ReadingPositions {
		out = append(out, p.Regime)
	}
	sort.Strings(out)
	return out
}
