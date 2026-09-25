package oracle

// BoardLayer is one layer's row for an agent, as the bare board shows it.
type BoardLayer struct {
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
	Tier   Tier   `json:"tier"`
	FanOut int    `json:"fan_out"`
}

// BoardRow is one agent's routing on the bare board: every layer holding a
// row, highest precedence first, and the layer whose row applies.
type BoardRow struct {
	Agent  string       `json:"agent"`
	Winner string       `json:"winner"`
	Layers []BoardLayer `json:"layers"`
}

// Accepted reports whether a routing table exists at the repository or the
// machine layer, the condition under which any row applies.
func (l *Layered) Accepted() bool { return l.accepted }

// Board returns one row per agent in the roster, or nil when nothing is
// accepted (the board is then unchanged: every step runs through the harness
// at host-decides, which is what the board already implies). The winner is
// Resolve's own Source, so the board and a step can never disagree.
func (l *Layered) Board() ([]BoardRow, error) {
	if !l.accepted {
		return nil, nil
	}
	var out []BoardRow
	for _, agent := range Roster() {
		rows, err := l.Rows(agent)
		if err != nil {
			return nil, err
		}
		r, err := Resolve(agent, l, NoConnections{})
		if err != nil {
			return nil, err
		}
		br := BoardRow{Agent: agent, Winner: r.Source.String()}
		for _, lr := range rows {
			br.Layers = append(br.Layers, BoardLayer{Layer: lr.Layer.String(), Origin: lr.Origin, Tier: lr.Row.Tier, FanOut: lr.Row.FanOut})
		}
		out = append(out, br)
	}
	return out, nil
}
