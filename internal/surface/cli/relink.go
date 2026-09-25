package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/relink"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// emitRelinked renders the links a record-moving verb repointed at the moved
// record's new path, one per line, so the operator sees every file the move
// touched beyond the record itself. Nothing is printed when no link named it.
func emitRelinked(w io.Writer, rewrites []relink.Rewrite) {
	if len(rewrites) == 0 {
		return
	}
	noun := "links"
	if len(rewrites) == 1 {
		noun = "link"
	}
	fmt.Fprintf(w, "  repointed %d %s that named the old path:\n", len(rewrites), noun)
	for _, rw := range rewrites {
		fmt.Fprintf(w, "    %s:%d  %s -> %s\n", termsafe.Sanitize(rw.File), rw.Line, termsafe.Sanitize(rw.From), termsafe.Sanitize(rw.To))
	}
}

// emitRelinkError warns, on stderr, that a repoint failed part-way. The move
// stands, so the verb does not fail; remedy says how the rest is finished.
func emitRelinkError(w io.Writer, verb, msg, remedy string) {
	if msg == "" {
		return
	}
	fmt.Fprintf(w, "WARNING: abcd %s — the record moved, but repointing the links that named its old path failed: %s; %s\n", verb, termsafe.Sanitize(msg), remedy)
}
