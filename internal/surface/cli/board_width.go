package cli

import (
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/term"
)

// boardColumns is the width the board is laid at off a terminal: a pipe, a
// file or a host relaying the output (iss-2610031207397996).
const boardColumns = 80

// boardWidth is the window the board is laid at: the terminal's columns when
// stdout is one, read through internal/term, and 80 otherwise, whatever
// COLUMNS says, so piped output is the same on every machine. Tests pin it
// (the repo's package-var seam pattern).
var boardWidth = func(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(f) {
		return boardColumns
	}
	cols, _ := term.Size(f, os.Getenv)
	return cols
}

// yesNo is a board row's answer in words.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// tierList names the work tiers present, in words, or says there are none.
func tierList(tiers []string) string {
	if len(tiers) == 0 {
		return "none"
	}
	return strings.Join(tiers, ", ")
}
