package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// Settings are provider sampling settings (temperature, seed, and whatever
// else a provider accepts), each a JSON scalar kept as its raw bytes so a
// number reaches the provider and the receipt exactly as written. Which keys a
// provider accepts is its adapter's judgement (itd-2609081951381895), never
// this package's: a key is shape-checked here and passed through.
type Settings map[string]json.RawMessage

// Row is one agent's routing row.
type Row struct {
	Tier Tier `json:"tier"`
	// FanOut is the most agents one step may run at once, itself included. It
	// may tighten the agent contract's ceiling and never raise it; 0 in a file
	// means "not stated", which resolves to the ceiling.
	FanOut   int      `json:"fan_out,omitempty"`
	Settings Settings `json:"settings,omitempty"`
}

// Table maps an agent's name to its row.
type Table map[string]Row

// rowFile is a row as a routing file spells it; decoded strictly, so a field
// nobody reads is refused rather than ignored.
type rowFile struct {
	Tier     string                     `json:"tier"`
	FanOut   *int                       `json:"fan_out"`
	Settings map[string]json.RawMessage `json:"settings"`
}

// flagRow is a row as --route sets it in the flag layer: a file row plus the
// connection the invocation named.
type flagRow struct {
	Tier       string                     `json:"tier"`
	Settings   map[string]json.RawMessage `json:"settings"`
	Connection string                     `json:"connection"`
}

// settingKeyRe bounds a setting's name to a provider-parameter shape.
var settingKeyRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Layered is the routing table's four layers for one invocation.
type Layered struct {
	stack *layered.Stack
	// accepted is true when a table exists at the repository or machine
	// layer: the bundled proposal applies only once one does.
	accepted bool
	// Diagnostics are the non-fatal reports a read produced, one line each,
	// for the front door to print on stderr before any step runs: an orphan
	// row (named, skipped, the rest apply) and a fan-out clamped to the
	// contract ceiling.
	Diagnostics []string
}

// Load reads the repository and machine routing files through the layered
// resolver and validates every row in them. A malformed file or row is an
// error naming the file; an orphan row and a clamped fan-out are Diagnostics.
func Load(r layered.Roots) (*Layered, error) {
	s, err := layered.Load(layered.OracleRouting, r)
	if err != nil {
		return nil, fmt.Errorf("oracle routing: %w", err)
	}
	if err := s.Claim("", "schema_version", "agents"); err != nil {
		return nil, fmt.Errorf("oracle routing: %w", err)
	}
	l := &Layered{stack: s, accepted: s.Present(layered.Repo) || s.Present(layered.Machine)}
	f := layered.OracleRouting
	for _, layer := range []struct {
		l      layered.Layer
		origin string
	}{{layered.Repo, f.RepoOrigin()}, {layered.Machine, f.MachineOrigin()}} {
		names, err := s.Members(layer.l, "agents")
		if err != nil {
			return nil, fmt.Errorf("oracle routing: %w", err)
		}
		orphans := 0
		for _, name := range names {
			if !inRoster(name) {
				// Named one line each up to maxOrphanLines, then counted: a
				// hostile file can carry thousands of rows (review-tier1 F1).
				if orphans++; orphans <= maxOrphanLines {
					l.Diagnostics = append(l.Diagnostics, fmt.Sprintf(
						"oracle routing: %s (%s layer) has a row for %q, which is not an agent in the roster; "+
							"the row is skipped and the remaining rows apply", layer.origin, layer.l, layered.BoundKey(name)))
				}
				continue
			}
			found, err := s.Lookup("agents." + name)
			if err != nil {
				return nil, fmt.Errorf("oracle routing: %w", err)
			}
			for _, fd := range found {
				if fd.Layer != layer.l {
					continue
				}
				row, clamped, err := decodeRow(name, fd.Raw)
				if err != nil {
					return nil, fmt.Errorf("oracle routing: %s (%s layer): agents.%s: %w", fd.Origin, fd.Layer, name, err)
				}
				if clamped != 0 {
					l.Diagnostics = append(l.Diagnostics, fmt.Sprintf(
						"oracle routing: %s (%s layer) sets fan_out %d for %s, above the agent contract's ceiling "+
							"of %d; a row may tighten the ceiling and never raise it, so %d applies",
						fd.Origin, fd.Layer, clamped, name, row.FanOut, row.FanOut))
				}
			}
		}
		if more := orphans - maxOrphanLines; more > 0 {
			l.Diagnostics = append(l.Diagnostics, fmt.Sprintf(
				"oracle routing: %s (%s layer) has %d more row(s) for names that are not agents in the roster; "+
					"they are skipped and the remaining rows apply", layer.origin, layer.l, more))
		}
	}
	return l, nil
}

// maxOrphanLines is the most orphan rows one layer names line by line.
const maxOrphanLines = 5

// decodeRow decodes and validates one file row, clamping its fan-out to the
// agent's ceiling. clamped is the stated fan-out when it was above the
// ceiling, 0 otherwise.
func decodeRow(agent string, raw json.RawMessage) (Row, int, error) {
	rf, err := layered.Decode[rowFile](raw)
	if err != nil {
		return Row{}, 0, err
	}
	if rf.Tier == "" {
		return Row{}, 0, fmt.Errorf("the row names no tier; it takes one of %s", tierList())
	}
	tier, err := ParseTier(rf.Tier)
	if err != nil {
		return Row{}, 0, err
	}
	settings, err := checkSettings(rf.Settings)
	if err != nil {
		return Row{}, 0, err
	}
	ceiling, _ := Ceiling(agent)
	row := Row{Tier: tier, FanOut: ceiling, Settings: settings}
	clamped := 0
	if rf.FanOut != nil {
		switch n := *rf.FanOut; {
		case n < 0:
			return Row{}, 0, fmt.Errorf("fan_out %d is negative; it takes a whole number from 1 to the contract ceiling (%d), or 0 for the ceiling", n, ceiling)
		case n > ceiling:
			clamped = n
		case n > 0:
			row.FanOut = n
		}
	}
	return row, clamped, nil
}

// checkSettings refuses a setting whose name is not parameter-shaped or whose
// value is not a JSON scalar (a string, a number or a boolean).
func checkSettings(in map[string]json.RawMessage) (Settings, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > MaxSettings {
		return nil, fmt.Errorf("%d settings are given; a row or a --route carries at most %d", len(in), MaxSettings)
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(Settings, len(in))
	for _, k := range keys {
		if !settingKeyRe.MatchString(k) {
			return nil, fmt.Errorf("setting %q is not a parameter name (lower case, digits and underscores, starting with a letter)", layered.BoundKey(k))
		}
		v := bytes.TrimSpace(in[k])
		if !scalar(v) {
			return nil, fmt.Errorf("setting %s is %s; a setting takes a string, a number or a boolean", k, layered.BoundKey(string(v)))
		}
		// Bounded and clean where it is read, because it reaches the request
		// block and the receipt as it is (review-tier1 F6): a receipt records a
		// setting as sent, so it is refused here rather than truncated there.
		if len(v) > MaxSettingBytes {
			return nil, fmt.Errorf("setting %s is %d bytes; a setting's value is at most %d", k, len(v), MaxSettingBytes)
		}
		var str string
		if v[0] == '"' && json.Unmarshal(v, &str) == nil && termsafe.Sanitize(str) != str {
			return nil, fmt.Errorf("setting %s carries a control, bidirectional or zero-width character; "+
				"a setting's value is plain text", k)
		}
		out[k] = append(json.RawMessage(nil), v...)
	}
	return out, nil
}

// scalar reports whether v is a JSON string, number or boolean.
func scalar(v []byte) bool {
	if len(v) == 0 || !json.Valid(v) {
		return false
	}
	switch v[0] {
	case '{', '[', 'n':
		return false
	}
	return true
}

// LayerRow is one layer's row for an agent, as a board renders it.
type LayerRow struct {
	Layer  layered.Layer
	Origin string
	Row    Row
	// Connection is the connection a --route named (flag layer only).
	Connection string
}

// Rows returns every layer's row for agent, highest precedence first, with
// the bundled proposal last. Whether the bundled row applies is Resolve's
// answer (it does only once a table is accepted); Rows lists it regardless, so
// the board can show what an acceptance would change.
func (l *Layered) Rows(agent string) ([]LayerRow, error) {
	if !inRoster(agent) {
		return nil, notInRoster(agent)
	}
	found, err := l.stack.Lookup("agents." + agent)
	if err != nil {
		return nil, err
	}
	var out []LayerRow
	for _, fd := range found {
		if fd.Layer == layered.Flag {
			fr, err := layered.Decode[flagRow](fd.Raw)
			if err != nil {
				return nil, err
			}
			tier, _ := ParseTier(fr.Tier)
			// A --route states no fan-out: the step runs at the bound the
			// layer beneath resolves to, clamped, which is Resolve's own
			// answer (review-tier1 F5), so the board and the step agree.
			r, err := Resolve(agent, l, NoConnections{})
			if err != nil {
				return nil, err
			}
			out = append(out, LayerRow{Layer: fd.Layer, Origin: fd.Origin, Row: Row{Tier: tier, FanOut: r.Row.FanOut, Settings: fr.Settings}, Connection: fr.Connection})
			continue
		}
		row, _, err := decodeRow(agent, fd.Raw)
		if err != nil {
			return nil, err
		}
		out = append(out, LayerRow{Layer: fd.Layer, Origin: fd.Origin, Row: row})
	}
	out = append(out, LayerRow{Layer: layered.Bundled, Origin: "bundled", Row: Proposal()[agent]})
	return out, nil
}

func notInRoster(agent string) error {
	return fmt.Errorf("%q is not an agent in the roster (%d agents: see agents/)", layered.BoundKey(agent), len(proposal))
}
