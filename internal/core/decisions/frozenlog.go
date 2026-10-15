package decisions

// frozenlog.go reads a managed repository's frozen decision log: the single
// file `.abcd/work/DECISIONS.md` a repository adopted before the folder
// replaced it. The product thinker ruled that such a file stays where it is, as
// frozen history (spc-2610110931381316, decisions 7 and 20): abcd reads it
// wherever it exists and never writes it, and new decisions go to the folder.
// FrozenLog is the one constant naming the retired path; every reader reaches
// the file through it.

import (
	"regexp"
	"strings"
)

// FrozenLog is a managed repository's frozen decision log, repo-relative and
// slash-separated: read-only history, never written.
const FrozenLog = ".abcd/work/DECISIONS.md"

// FrozenEntry is one entry of a frozen log: a top-level list item and the
// indented lines that continue it, as written.
type FrozenEntry struct {
	// Line is the 1-based line the entry's bullet opens on, stable for an
	// unchanged file.
	Line int
	// Text is the entry, its bullet and continuation lines joined by newlines.
	Text string
}

// FrozenEntries splits a frozen log into its entries. An entry opens on a line
// starting `- ` and takes every following line that starts with a space or a
// tab; any other line ends it. The header and any prose between entries belong
// to no entry.
func FrozenEntries(log string) []FrozenEntry {
	var out []FrozenEntry
	var cur *FrozenEntry
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	for i, raw := range strings.Split(log, "\n") {
		line := strings.TrimRight(raw, "\r")
		switch {
		case strings.HasPrefix(line, "- "):
			flush()
			cur = &FrozenEntry{Line: i + 1, Text: line}
		case cur != nil && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")):
			cur.Text += "\n" + line
		default:
			flush()
		}
	}
	flush()
	return out
}

// Names reports whether text names any of ids as a whole word: an id that only
// starts a longer one (itd-10 inside itd-100) is not named. Empty ids are
// ignored.
func Names(text string, ids ...string) bool {
	for _, id := range ids {
		if id == "" {
			continue
		}
		if regexp.MustCompile(`(^|[^0-9A-Za-z-])` + regexp.QuoteMeta(id) + `($|[^0-9A-Za-z-])`).MatchString(text) {
			return true
		}
	}
	return false
}
