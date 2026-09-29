package positioning

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DefaultBlockLocation is where a repo's identity block lands when onboarding
// does not name a location: one file in the durable record, under the heading
// the parser and every generated pointer use.
var DefaultBlockLocation = BlockLocation{
	File:    ".abcd/development/IDENTITY.md",
	Heading: "Identity (canonical)",
}

// Init refusals, so a surface can render each one as its own message.
var (
	// ErrAlreadyAdopted: the repo has a registry already, and Init was given
	// answers that would change what it points at. Repointing the canon is a
	// deliberate edit, never a side effect of re-running onboarding.
	ErrAlreadyAdopted = errors.New("this repo already records an identity block")
	// ErrAnswersRequired: no block exists to adopt, and the required answers
	// (title and tagline) were not supplied.
	ErrAnswersRequired = errors.New("an identity block needs at least a title and a tagline")
)

// InitRequest is the onboarding interview's outcome. The host asks the
// questions (the prepare surface carries the wording); this is what it hands
// over. Pitch may be empty — it is skippable at onboarding.
type InitRequest struct {
	Title    string
	Tagline  string
	Pitch    string
	Location BlockLocation // zero value means DefaultBlockLocation
}

// InitResult reports what onboarding did, so a surface can say it plainly.
type InitResult struct {
	Location BlockLocation `json:"location"`
	Block    Block         `json:"block"`
	// AdoptedExisting: a block was already there and was registered as the
	// canon rather than re-interviewed and rewritten.
	AdoptedExisting bool `json:"adoptedExisting"`
	// NoOp: the repo was already fully adopted; nothing was written.
	NoOp bool `json:"noOp"`
	// Wrote lists the repo-relative paths written, in order.
	Wrote []string `json:"wrote,omitempty"`
}

// Init adopts the positioning check for a repo: it registers an identity block
// and writes the pointer that records where it lives.
//
// It never re-interviews a repo that already has a block — an existing block at
// the target location is adopted as the canon and left byte-for-byte alone,
// because a second set of answers is exactly how a project ends up with two
// canons. A repo whose registry is already committed is a no-op when no answers
// are given, and a refusal when they are: repointing the canon is a deliberate
// edit to the registry, not a side effect of re-running onboarding.
func Init(root string, req InitRequest) (InitResult, error) {
	loc := req.Location
	if strings.TrimSpace(loc.File) == "" {
		loc.File = DefaultBlockLocation.File
	}
	if strings.TrimSpace(loc.Heading) == "" {
		loc.Heading = DefaultBlockLocation.Heading
	}
	if !validBlockFile(loc.File) {
		return InitResult{}, fmt.Errorf("%w: %q", ErrBadLocation, loc.File)
	}

	// One containment root for the whole operation: every read and every write
	// below resolves through it, so neither the block location nor the registry
	// path can land outside the repository via a symlinked ancestor.
	r, err := openRepoRoot(root)
	if err != nil {
		return InitResult{}, err
	}
	defer r.Close()

	answered := strings.TrimSpace(req.Title) != "" || strings.TrimSpace(req.Tagline) != "" || strings.TrimSpace(req.Pitch) != ""

	if existing, adopted, err := LoadConfig(root); err != nil {
		return InitResult{}, err
	} else if adopted {
		if answered {
			return InitResult{}, fmt.Errorf("%w: it points at %s (%q); change the block there, or edit %s deliberately",
				ErrAlreadyAdopted, existing.Block.File, existing.Block.Heading, ConfigRelPath)
		}
		block, err := parseBlockIn(r, existing.Block)
		if err != nil {
			return InitResult{}, err
		}
		return InitResult{Location: existing.Block, Block: block, NoOp: true}, nil
	}

	res := InitResult{Location: loc}

	// An existing block at the target location is the canon: register it,
	// never rewrite it.
	block, perr := parseBlockIn(r, loc)
	switch {
	case perr == nil:
		res.Block, res.AdoptedExisting = block, true

	case errors.Is(perr, ErrBulletMissing):
		// The heading is already there but its bullets are incomplete. Only an
		// ABSENT file or heading is fixable by appending a section; appending to
		// this one would leave two copies of the same heading, of which the
		// parser still reads the broken first — so every retry would append
		// again and none would ever succeed. Refuse before writing anything and
		// name what a human has to repair.
		return InitResult{}, fmt.Errorf("%w — complete it by hand, then re-run to register it", perr)

	case !errors.Is(perr, ErrBlockFileMissing) && !errors.Is(perr, ErrHeadingMissing):
		return InitResult{}, perr

	default:
		title, tagline := strings.TrimSpace(req.Title), strings.TrimSpace(req.Tagline)
		if title == "" || tagline == "" {
			return InitResult{}, fmt.Errorf("%w (the pitch is optional)", ErrAnswersRequired)
		}
		if err := writeBlock(r, loc, title, tagline, strings.TrimSpace(req.Pitch)); err != nil {
			return InitResult{}, err
		}
		res.Wrote = append(res.Wrote, loc.File)
		written, werr := parseBlockIn(r, loc)
		if werr != nil {
			// The block we just wrote must parse; if it does not, the scaffold
			// is broken and must not be registered as a canon.
			return InitResult{}, fmt.Errorf("the scaffolded identity block does not parse: %w", werr)
		}
		res.Block = written
	}

	if err := writeConfig(r, DefaultConfig(loc)); err != nil {
		return InitResult{}, err
	}
	res.Wrote = append(res.Wrote, ConfigRelPath)
	return res, nil
}

// blockSection renders the identity block as a markdown section. The prose
// above the bullets says what the block is for, so a maintainer meeting the file
// cold knows why editing it matters.
func blockSection(heading, title, tagline, pitch string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", heading)
	b.WriteString("The single recorded home for how this project names itself. Every public\n" +
		"surface — the README strapline, the package or plugin manifest description,\n" +
		"the conventions file's opening — renders from these three lines; a surface\n" +
		"that says something else is drift, fixed by re-rendering from here or by\n" +
		"deliberately changing this block.\n\n")
	fmt.Fprintf(&b, "- **Title:** %s\n", title)
	fmt.Fprintf(&b, "- **Tagline:** %s\n", tagline)
	if pitch != "" {
		fmt.Fprintf(&b, "- **Pitch:** %s\n", pitch)
	}
	return b.String()
}

// writeBlock lands the identity section at loc: appended to the file when it
// already exists (never replacing what is there), or as a new file with a title
// heading when it does not.
// The read and both writes resolve through r, so a location behind a symlinked
// ancestor refuses instead of scaffolding the block outside the repository.
func writeBlock(r *os.Root, loc BlockLocation, title, tagline, pitch string) error {
	section := blockSection(loc.Heading, title, tagline, pitch)

	existing, err := fsutil.ReadGuardedInRoot(r, loc.File, maxBlockFileBytes)
	switch {
	case err == nil:
		body := normalizeNewlines(existing)
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		if !strings.HasSuffix(body, "\n\n") {
			body += "\n"
		}
		return fsutil.WriteFileAtomicPreserveModeInRoot(r, loc.File, []byte(body+section))
	case os.IsNotExist(err):
		return fsutil.WriteFileAtomicInRoot(r, loc.File, []byte("# Identity\n\n"+section), 0o644)
	default:
		return fmt.Errorf("%s: %s", loc.File, pathFreeReason(err))
	}
}

// writeConfig lands the registry, pretty-printed with a trailing newline,
// through the canonical atomic primitive.
func writeConfig(r *os.Root, cfg Config) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// The registry carries regexps full of characters HTML escaping would
	// mangle beyond recognition on the round trip.
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(r, ConfigRelPath, buf.Bytes(), 0o644)
}
