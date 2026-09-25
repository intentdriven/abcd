package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// oracleStore names the routing table in a refusal about where it is read.
const oracleStore = "the repository's routing table"

// boardOracle composes the board's oracle member (itd-2609170822093401 AC 6):
// one row per agent once a routing table is accepted at the repository or the
// machine layer, nil otherwise. The repository layer is read from the checkout
// root git names for cwd and from nowhere above it; outside any repository only
// the machine layer is read. A table that cannot be read omits the member and
// says why on stderr, and an orphan row or a clamped fan-out is reported there
// too: the board itself never fails on a routing file.
func boardOracle(cwd string, stderr io.Writer) []oracle.BoardRow {
	repo, err := gitutil.CheckoutRoot(cwd, oracleStore)
	if err != nil {
		if gitutil.RepoShapedRoot(cwd) != "" {
			fmt.Fprintf(stderr, "abcd: the oracle lines are omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
			return nil
		}
		repo = ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	l, err := oracle.Load(layered.Roots{Repo: repo, Home: home})
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

// renderBoardOracle writes the oracle heading and one line per agent: every
// layer holding a row as layer=tier, highest precedence first, the one that
// applies marked with *.
func renderBoardOracle(w io.Writer, rows []oracle.BoardRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(w, "  oracle:     routing table accepted; * marks the row that applies (flag > repo > machine > bundled)\n")
	width := 0
	for _, r := range rows {
		width = max(width, len(r.Agent))
	}
	for _, r := range rows {
		cells := make([]string, 0, len(r.Layers))
		for _, lr := range r.Layers {
			cell := lr.Layer + "=" + string(lr.Tier)
			if lr.Layer == r.Winner {
				cell += "*"
			}
			cells = append(cells, cell)
		}
		fmt.Fprintf(w, "    %-*s  %s\n", width, r.Agent, termsafe.Sanitize(strings.Join(cells, " ")))
	}
}
