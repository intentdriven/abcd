package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// diagnosticLine formats one out-of-band diagnostic, masks it, writes it to w as
// exactly one line, and returns the masked message so a caller that also carries
// it elsewhere (a hook's --json reason) says the same thing on both channels.
//
// It is the print site for every stderr line that does not pass through Run
// (iss-2609260221565656). Run masks the refusal it prints for every verb, but a
// hook writes its own diagnostics and returns nil or a message-less exit code,
// so Run never sees them; the text they interpolate — a payload's transcript
// path, a committed file's decoder error, a read error — is not abcd's. Masking
// the whole formatted line, not each argument, is deliberate: an error's text
// embeds its input in ways the format string cannot see (a JSON type error names
// the map key it failed under, raw), and the next such message must not be the
// one that reaches the terminal. Sanitize, not SanitizeBlock: a diagnostic is
// one line, and a newline inside it would forge a second.
func diagnosticLine(w io.Writer, format string, a ...any) string {
	msg := termsafe.Sanitize(fmt.Sprintf(format, a...))
	fmt.Fprintln(w, msg)
	return msg
}
