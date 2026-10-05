package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// boardOracle composes the board's oracle member (itd-2609170822093401 AC 6):
// one row per agent once a routing table is accepted at the repository or the
// machine layer, nil otherwise. The layers are read from layered.RootsFor, the
// root the rules loader resolves for cwd, so the board reads the table from the
// same directory the session's rules and guard come from. A table that cannot
// be read omits the member and says why on stderr, and an orphan row or a
// clamped fan-out is reported there too: the board itself never fails on a
// routing file.
func boardOracle(cwd string, stderr io.Writer) []oracle.BoardRow {
	roots, notes := layered.RootsFor(cwd)
	for _, n := range notes {
		fmt.Fprintf(stderr, "abcd: %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	l, err := oracle.Load(roots)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the oracle lines are omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	for _, d := range l.Diagnostics {
		fmt.Fprintf(stderr, "abcd: %s\n", termsafe.Sanitize(d))
	}
	rows, err := l.Board()
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the oracle lines are omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	return rows
}

// oracleRow is the oracle row of the facilitator's view: the heading and one
// line per agent, every layer holding a row as layer=tier, highest precedence
// first, the one that applies marked with *. Nil when nothing is accepted.
func oracleRow(rows []oracle.BoardRow) *board.Row {
	if len(rows) == 0 {
		return nil
	}
	r := &board.Row{Label: "oracle", Text: "routing table accepted; * marks the row that applies (flag > repo > machine > bundled)"}
	width := 0
	for _, row := range rows {
		width = max(width, len(row.Agent))
	}
	for _, row := range rows {
		cells := make([]string, 0, len(row.Layers))
		for _, lr := range row.Layers {
			cell := lr.Layer + "=" + string(lr.Tier)
			if lr.Layer == row.Winner {
				cell += "*"
			}
			cells = append(cells, cell)
		}
		r.Items = append(r.Items, fmt.Sprintf("%-*s  %s", width, row.Agent, termsafe.Sanitize(strings.Join(cells, " "))))
	}
	return r
}
