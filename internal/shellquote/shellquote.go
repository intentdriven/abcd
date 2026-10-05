// Package shellquote is the one POSIX-shell quoting primitive: the form a
// command abcd prints, or writes into a script, takes so a value reaches the
// shell as one literal word whatever it holds.
//
// Before it, five packages each spelled the same one-liner (the dev shim's
// paths and the status line in internal/core/ahoy, the orphan remedy in
// internal/core/capture, a lab harvest's capture line in internal/core/lab,
// the connect guide's command in internal/core/oracle, and one worktree's
// repair in internal/abcdhome); TestOnlySingleQuotesForTheShell holds every
// other package to this one.
//
// It is a leaf that imports only the standard library, outside internal/core,
// so internal/abcdhome, which is documented to import nothing else, can use it.
package shellquote

import "strings"

// Single is s as one single-quoted POSIX shell word, the quotes included.
// Inside single quotes a shell interprets nothing, not `$`, a backquote, a
// backslash or, in an interactive bash or zsh, `!word` history, so the one
// character that needs spelling is the quote itself: close, escaped quote,
// reopen, which is the four bytes
//
//	'\''
//
// The empty string is the empty word, two quotes with nothing between. (The
// escape's spelling sits in a code block because gofmt rewrites a doubled
// apostrophe in doc-comment prose to a typographic quote.)
func Single(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
