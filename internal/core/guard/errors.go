package guard

import "errors"

// The guard sentinels. Every error the package returns wraps one of these, so a
// front door can render the failure loudly (and, for the hook shim, fail OPEN)
// without string-matching. Core never decides what a failure means for a
// session — it only names it.
var (
	// ErrUnparsableCommand is a candidate command line the shell tokenizer
	// cannot split — an unterminated quote (`'`, `"`, `$'…'`, or one in a
	// here-document delimiter word) in COMMAND text, which no shell runs
	// either. Text inside a here-document BODY is never command text, so a
	// quote there is data and does not reach this error. A state bash DOES run
	// (a trailing backslash, an unterminated here-document) is never this
	// error: the hook maps it to fail-open, so those get a verdict.
	ErrUnparsableCommand = errors.New("guard: unparsable command line")

	// ErrMalformedConfig is a per-repo .abcd/guard.json that is unreadable or
	// is not valid JSON.
	ErrMalformedConfig = errors.New("guard: malformed config")

	// ErrSchemaVersion is a config declaring a schema_version this build does
	// not implement.
	ErrSchemaVersion = errors.New("guard: unsupported schema_version")

	// ErrUnknownTier is an entry whose tier is neither blocker nor warn.
	ErrUnknownTier = errors.New("guard: unknown tier")

	// ErrInvalidEntry is an entry that fails the registry schema (bad id, no
	// pattern command, missing successor or why).
	ErrInvalidEntry = errors.New("guard: invalid entry")
)
