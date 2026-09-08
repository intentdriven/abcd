package positioning

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ConfigRelPath is the committed positioning configuration, relative to the repo
// root. It sits beside the repo's other per-concern lint configuration
// (.abcd/docs-lint.json, .abcd/record-lint.json, .abcd/rules.json) rather than
// inside .abcd/config/, where identity.json already means the git commit-author
// pin — two different things must not share a filename.
const ConfigRelPath = ".abcd/positioning.json"

// maxConfigBytes caps the guarded config read. The configuration is a short
// committed registry; 256 KiB is far past any real one.
const maxConfigBytes = 256 << 10

// maxSurfaces bounds how many surfaces a committed registry may declare. The
// canonical set is three (DefaultSurfaces); a real repo that overrides them adds
// a handful more. 64 is far past any genuine registry yet closes the
// amplification: each surface triggers a guarded read capped at maxSurfaceBytes
// (1 MiB, check.go), so an unbounded count lets one audit run over a hostile
// repo hold an unbounded multiple of that cap (iss-149).
const maxSurfaces = 64

// Surface kinds — how a surface's self-description is located in its file.
const (
	// KindRegexp locates the text with the first Patterns entry that matches,
	// capturing group 1. The capture may span lines (a wrapped paragraph).
	KindRegexp = "regexp"
	// KindJSONField reads a top-level string field of a JSON manifest, so the
	// registry is not tied to one manifest format.
	KindJSONField = "json_field"
)

// Severity levels a repo may set for the whole positioning family.
const (
	SeverityWarn    = "warn"
	SeverityBlocker = "blocker"
)

// ErrConfigInvalid wraps every configuration-validation failure, so a caller can
// distinguish a malformed registry from a real drift finding.
var ErrConfigInvalid = errors.New("positioning config is invalid")

// Config is the committed positioning registry: where the identity block lives,
// how loudly the family reports, and which surfaces render from the block.
type Config struct {
	SchemaVersion int           `json:"schema_version"`
	Block         BlockLocation `json:"block"`
	// Severity is the whole family's weight: "warn" (the default — highlight,
	// never gate) or "blocker" (a repo opting into a hard gate).
	Severity string `json:"severity,omitempty"`
	// Surfaces is the registry. Empty means the three defaults; a non-empty
	// list REPLACES them, so nothing is ever registered silently.
	Surfaces []Surface `json:"surfaces,omitempty"`
}

// Surface is one rendered surface held to the block.
type Surface struct {
	ID string `json:"id"`
	// Files are candidate locations, in order; the first that exists is the one
	// checked. A surface whose candidates are all absent is skipped, never
	// reported as drifted.
	Files []string `json:"files"`
	Kind  string   `json:"kind"`
	// Patterns (KindRegexp) are tried in order; the first that matches wins.
	// Each must compile and expose exactly one capture group.
	Patterns []string `json:"patterns,omitempty"`
	// Field (KindJSONField) is the top-level JSON key holding the description.
	Field string `json:"field,omitempty"`
	// Requires names the block fields this surface must carry (title, tagline,
	// pitch). At least one is required.
	Requires []string `json:"requires"`
	// Template renders the proposed replacement, over {title}/{tagline}/{pitch}.
	// Empty means the required fields joined by a single space.
	Template string `json:"template,omitempty"`
}

// blockedSeverity resolves the family severity, defaulting to warn.
func (c Config) severity() string {
	if strings.EqualFold(strings.TrimSpace(c.Severity), SeverityBlocker) {
		return SeverityBlocker
	}
	return SeverityWarn
}

// surfaces resolves the effective registry: the declared list, or the three
// defaults when none is declared.
func (c Config) surfaces() []Surface {
	if len(c.Surfaces) > 0 {
		return c.Surfaces
	}
	return DefaultSurfaces()
}

// DefaultSurfaces is the canonical three: the README strapline, the
// plugin/package manifest description, and the conventions file's opening
// paragraph. They are data, not branches — a repo overrides them wholesale in
// its own configuration, and the manifest entry lists candidate files so the
// default is not tied to one manifest format.
func DefaultSurfaces() []Surface {
	return []Surface{
		{
			ID:    "readme-strapline",
			Files: []string{"README.md"},
			Kind:  KindRegexp,
			Patterns: []string{
				// A centred HTML strapline, then the first prose paragraph
				// under the top-level heading.
				`(?s)<p[^>]*>(.*?)</p>`,
				`(?m)^#[^#\n]*\n+([^\s<#>\[!|*-][^\n]*)$`,
			},
			Requires: []string{"tagline"},
			Template: "{tagline}",
		},
		{
			ID:       "manifest-description",
			Files:    []string{".claude-plugin/plugin.json", "package.json"},
			Kind:     KindJSONField,
			Field:    "description",
			Requires: []string{"tagline"},
			Template: "{tagline}",
		},
		{
			ID:    "conventions-opening",
			Files: []string{"AGENTS.md"},
			Kind:  KindRegexp,
			Patterns: []string{
				// The first paragraph after an abcd-managed marker block, then
				// the first paragraph after the top-level heading. The
				// paragraph terminator accepts end-of-file, so a conventions
				// file whose opening paragraph is its last one still locates.
				`(?s)<!-- END ABCD -->\n+(.*?)(?:\n\n|\z)`,
				`(?m)^#[^#\n]*\n+([^\s<#>\[!|*-][^\n]*)$`,
			},
			Requires: []string{"tagline"},
			Template: "{tagline}",
		},
	}
}

// DefaultConfig is the registry a repo adopts at onboarding when it has no
// committed one: the three default surfaces, warn-tier, pointed at loc.
func DefaultConfig(loc BlockLocation) Config {
	return Config{SchemaVersion: 1, Block: loc, Severity: SeverityWarn, Surfaces: DefaultSurfaces()}
}

// LoadConfig reads and validates .abcd/positioning.json under root. It returns
// (cfg, true, nil) when the registry is present and well formed, (Config{},
// false, nil) when the repo has not adopted the check, and an ErrConfigInvalid
// error when the file is present but unusable — a malformed registry must never
// read as "no drift".
func LoadConfig(root string) (Config, bool, error) {
	r, err := openRepoRoot(root)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("%w: %s: %s", ErrConfigInvalid, ConfigRelPath, pathFreeReason(err))
	}
	defer r.Close()
	// Contained, not merely lexically checked: ConfigRelPath is a fixed path,
	// but a hostile repo can commit `.abcd` itself as a symlink and have the
	// loader read — and then act on — a registry the repository does not own.
	data, err := fsutil.ReadGuardedInRoot(r, ConfigRelPath, maxConfigBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("%w: %s: %s", ErrConfigInvalid, ConfigRelPath, pathFreeReason(err))
	}
	var cfg Config
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, false, fmt.Errorf("%w: %s: %s", ErrConfigInvalid, ConfigRelPath, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

// Validate checks the registry before any of it is used. Every field arrives as
// committed data, so each one is validated here rather than trusted downstream:
// an unparseable pattern, an escaping path, or an unknown block field is a
// refusal, not a silently skipped surface.
func (c Config) Validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("%w: schema_version must be 1, got %d", ErrConfigInvalid, c.SchemaVersion)
	}
	if !fsutil.ValidRelPath(c.Block.File) {
		return fmt.Errorf("%w: block.file %q is not a repo-relative path", ErrConfigInvalid, c.Block.File)
	}
	if strings.TrimSpace(c.Block.Heading) == "" {
		return fmt.Errorf("%w: block.heading is required", ErrConfigInvalid)
	}
	if s := strings.TrimSpace(c.Severity); s != "" && !strings.EqualFold(s, SeverityWarn) && !strings.EqualFold(s, SeverityBlocker) {
		return fmt.Errorf("%w: severity must be %q or %q, got %q", ErrConfigInvalid, SeverityWarn, SeverityBlocker, c.Severity)
	}
	if len(c.Surfaces) > maxSurfaces {
		return fmt.Errorf("%w: registry declares %d surfaces, more than the %d-surface maximum", ErrConfigInvalid, len(c.Surfaces), maxSurfaces)
	}
	seen := map[string]bool{}
	for i, s := range c.Surfaces {
		if err := s.validate(i); err != nil {
			return err
		}
		if seen[s.ID] {
			return fmt.Errorf("%w: duplicate surface id %q", ErrConfigInvalid, s.ID)
		}
		seen[s.ID] = true
	}
	return nil
}

func (s Surface) validate(i int) error {
	where := fmt.Sprintf("surfaces[%d]", i)
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("%w: %s.id is required", ErrConfigInvalid, where)
	}
	where = fmt.Sprintf("surface %q", s.ID)
	if len(s.Files) == 0 {
		return fmt.Errorf("%w: %s.files must list at least one candidate", ErrConfigInvalid, where)
	}
	for _, f := range s.Files {
		if !fsutil.ValidRelPath(f) {
			return fmt.Errorf("%w: %s.files entry %q is not a repo-relative path", ErrConfigInvalid, where, f)
		}
		// ValidRelPath accepts ".git/config" — it is clean, relative, and inside
		// the containment root — but a surface pointed there would quote the
		// git directory (a credential-bearing remote URL in .git/config) into
		// identity output. Nothing under .git is a rendered positioning surface.
		// The refusal itself lives in fsutil, shared with the site composer's
		// manifest gate, which faces the same threat from the same primitive
		// (iss-150, iss-2609081940475662).
		if fsutil.InsideGitDir(f) {
			return fmt.Errorf("%w: %s.files entry %q is inside .git", ErrConfigInvalid, where, f)
		}
	}
	switch s.Kind {
	case KindRegexp:
		if len(s.Patterns) == 0 {
			return fmt.Errorf("%w: %s.patterns must list at least one pattern", ErrConfigInvalid, where)
		}
		for _, p := range s.Patterns {
			re, err := regexp.Compile(p)
			if err != nil {
				return fmt.Errorf("%w: %s pattern %q does not compile: %s", ErrConfigInvalid, where, p, err)
			}
			if n := re.NumSubexp(); n != 1 {
				return fmt.Errorf("%w: %s pattern %q has %d capture groups, want exactly 1", ErrConfigInvalid, where, p, n)
			}
		}
	case KindJSONField:
		if strings.TrimSpace(s.Field) == "" {
			return fmt.Errorf("%w: %s.field is required for kind %q", ErrConfigInvalid, where, KindJSONField)
		}
	default:
		return fmt.Errorf("%w: %s.kind must be %q or %q, got %q", ErrConfigInvalid, where, KindRegexp, KindJSONField, s.Kind)
	}
	if len(s.Requires) == 0 {
		return fmt.Errorf("%w: %s.requires must name at least one block field", ErrConfigInvalid, where)
	}
	var probe Block
	for _, f := range s.Requires {
		if _, ok := probe.Field(f); !ok {
			return fmt.Errorf("%w: %s.requires names unknown block field %q (want title, tagline, or pitch)", ErrConfigInvalid, where, f)
		}
	}
	for _, ph := range placeholderRe.FindAllStringSubmatch(s.Template, -1) {
		if _, ok := probe.Field(ph[1]); !ok {
			return fmt.Errorf("%w: %s.template uses unknown block field {%s}", ErrConfigInvalid, where, ph[1])
		}
	}
	return nil
}

// placeholderRe matches a {field} placeholder in a surface template.
var placeholderRe = regexp.MustCompile(`\{([a-zA-Z]+)\}`)

// render substitutes the block's fields into the surface template, defaulting to
// the required fields joined by a single space when no template is declared.
func (s Surface) render(b Block) string {
	tmpl := strings.TrimSpace(s.Template)
	if tmpl == "" {
		parts := make([]string, 0, len(s.Requires))
		for _, f := range s.Requires {
			if v, ok := b.Field(f); ok && v != "" {
				parts = append(parts, v)
			}
		}
		return strings.Join(parts, " ")
	}
	return placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		v, _ := b.Field(placeholderRe.FindStringSubmatch(m)[1])
		return v
	})
}
