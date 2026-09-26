// Package issuerecord is the issue ledger's record reader: the one judgement of
// whether a file in a status directory is a record the ledger takes, and if not,
// which reader stage refused it and why.
//
// It is a leaf for the reason core/issueschema is: the ledger's surfaces
// (core/capture) and the gate on the committed ledger (core/lint) must reach
// one verdict on one file, and a record they disagree about sits in the ledger
// unread by every surface while the gate stays green (iss-2609261631132673). It
// is not inside core/capture because that package's own tests import core/lint,
// so a lint importing capture back is an import cycle.
package issuerecord

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// Layer is the reader stage that refused a ledger file, in scan order.
type Layer string

// The reader's stages, in the order a file meets them.
const (
	// LayerName: the filename claims a record and is not a well-formed one.
	LayerName Layer = "filename"
	// LayerRead: the guarded read refused the leaf (a FIFO, a symlink, an
	// oversize body, an I/O error) — nothing about the record's content.
	LayerRead Layer = "read"
	// LayerFrontmatter: the bytes do not parse as a frontmatter block.
	LayerFrontmatter Layer = "frontmatter"
	// LayerSchema: the frontmatter parses and the issue schema refuses a key or
	// value in it.
	LayerSchema Layer = "schema"
	// LayerInvariant: schema-clean, and the record disagrees with where it sits
	// — its filename, or the status folder holding it.
	LayerInvariant Layer = "invariant"
)

// Refusal is the reader's reason for skipping one file: the stage that refused
// it and that stage's error.
type Refusal struct {
	Layer Layer
	Err   error
}

func (r *Refusal) Error() string { return string(r.Layer) + ": " + r.Err.Error() }

func (r *Refusal) Unwrap() error { return r.Err }

var (
	// nameClaimRe matches a filename that CLAIMS to be a ledger record: the
	// family prefix followed by an ordinal. It is deliberately LOOSER than
	// recordid.SplitRecordFilename, and the gap between the two is the point — a
	// name that claims to be a record and then is not well-formed is reported as
	// a skip rather than dropped, while a file claiming nothing (README.md,
	// notes.md, iss-notes.md, the allocator lock) stays silently ignored. The
	// ordinal is what parts the two: prose that merely starts with the prefix is
	// not asserting an id.
	nameClaimRe = regexp.MustCompile(`^iss-[0-9]`)
	// fileNumRe is the ONE grammar that decides whether a ledger filename NAMES
	// a record — recordid.FilenameNumRe, which the read-side resolver and
	// record-lint's per-store rule match too (iss-2608280739112123).
	fileNumRe = recordid.FilenameNumRe(issFamily)
)

// Judge is the ledger reader's verdict on the file at path, which sits in the
// status directory named status (open, resolved or wontfix). claims is false
// for a file that claims no record — README.md, a stray note, the allocator
// lock — which every reader ignores. Otherwise exactly one of fm and refusal is
// set: the record's frontmatter and body when the reader takes it, the stage
// and reason when it skips it.
func Judge(path, status string) (fm map[string]any, body string, refusal *Refusal, claims bool) {
	name := filepath.Base(path)
	if !fileNumRe.MatchString(name) {
		// A file that claims to be a record (family prefix + ordinal) and is not
		// well-formed is REPORTED, not dropped: it sits in the ledger, counted by
		// nothing and reported by nothing, which is how a record gets silently
		// lost. A file claiming nothing is silently ignored.
		//
		// The stricter slug shape is still enforced — as a filename<->frontmatter
		// agreement, in ValidateInvariants — but that is a judgement on a record,
		// not the question of whether one exists.
		if filepath.Ext(name) == ".md" && nameClaimRe.MatchString(name) {
			return nil, "", &Refusal{LayerName, fmt.Errorf(
				"%w: filename %q is not a well-formed record name (iss-N[-slug].md)",
				ErrInvariantViolation, name)}, true
		}
		return nil, "", nil, false
	}
	// A well-formed name always ends .md — the grammar's pattern requires it —
	// so no separate extension check is needed on this path.
	//
	// The read is guarded, not bare: a well-formed NAME says nothing about the
	// leaf behind it, and in a hostile clone that leaf is a FIFO that would hang
	// the scan, a symlink to a file outside the ledger, or a body sized to make
	// the read unbounded. Each is a skipped record, never a hang and never
	// serialized.
	content, err := ReadGuarded(path)
	if err != nil {
		return nil, "", &Refusal{LayerRead, err}, true
	}
	fm, body, err = Parse(content)
	if err != nil {
		return nil, "", &Refusal{LayerFrontmatter, err}, true
	}
	if err := ValidateStrict(fm); err != nil {
		return nil, "", &Refusal{LayerSchema, err}, true
	}
	if err := ValidateInvariants(fm, status, path); err != nil {
		return nil, "", &Refusal{LayerInvariant, err}, true
	}
	return fm, body, nil, true
}

// ReadGuarded reads one record file through the shared trust-boundary
// primitive: fsutil.ReadGuarded opens once with O_NOFOLLOW and O_NONBLOCK and
// validates on the SAME descriptor, so no symlink swap fits between a check and
// the read, and a FIFO or device at a record name cannot block the open. The cap
// is issueschema.RecordReadLimit, the ONE cap the ledger's families share —
// core/lint applies the same value, because a cap the board applies loosely and
// the verb applies tightly makes the ledger say two things about one file.
//
// It is the reader for EVERY record family core/capture reads. The reading and
// disposition families used it from the start; the issue family read through a
// bare os.ReadFile until GHSA-fh9j-8xmg-m33f, so a committed FIFO hung `list`,
// an oversize record was serialized unbounded, and a committed symlink read an
// out-of-tree file into `list --json` (iss-2609012036271396). A record store a
// clone carries is a trust boundary whichever family the record belongs to.
//
// The sentinels are mapped to ErrPathUnsafe: a non-regular leaf, or a symlink
// refused by O_NOFOLLOW, is what every caller already tests for.
func ReadGuarded(path string) (string, error) {
	data, err := fsutil.ReadGuarded(path, issueschema.RecordReadLimit)
	if err == nil {
		return string(data), nil
	}
	if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP) {
		return "", fmt.Errorf("%w: record path is not a regular file: %s", ErrPathUnsafe, path)
	}
	if errors.Is(err, fsutil.ErrTooBig) {
		return "", fmt.Errorf("%w: record is larger than the %d-byte size cap and was left unread: %s", ErrPathUnsafe, issueschema.RecordReadLimit, path)
	}
	// A record the process may not open is a refusal with a name, not a raw open
	// error surfacing through a verb: the caller is told the ledger could not be
	// read and which file, rather than a syscall's own wording.
	if errors.Is(err, fs.ErrPermission) {
		return "", fmt.Errorf("%w: record is unreadable (permission denied): %s", ErrPathUnsafe, path)
	}
	return "", err
}
