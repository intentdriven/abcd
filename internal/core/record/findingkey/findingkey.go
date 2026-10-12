// Package findingkey derives the stable key of a consistency finding (G1 of
// itd-2609291923559186, spc-2610021456200324): `cfk-` and sixteen hex digits,
// the same for the same contradiction on every run.
//
// It is the ONE derivation. The consistency pass keys every finding and report
// row with it and capture writes it into the issue a finding files
// (`finding_key:`), so a contradiction found again is joined to the record that
// already holds it; record-lint keys its cross-document findings with it too.
// It sits under core/record beside match because both core/intent and core/lint
// call it, and lint cannot import intent. It is a leaf: it reads nothing,
// writes nothing and never prints.
package findingkey

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// Prefix opens every key.
const Prefix = "cfk-"

// domain separates this derivation from every other hash over the same text.
// Bump the version, and every key with it, only when the derivation changes.
const domain = "abcd/finding-key/v1"

// keyRe is the key's one shape: the prefix and sixteen lowercase hex digits.
var keyRe = regexp.MustCompile(`^cfk-[0-9a-f]{16}$`)

// Valid reports whether s is verbatim a key's shape. Capture refuses a
// `finding_key` that is not, and record_schema reports one, so the writer and
// the committed-ledger gate agree.
func Valid(s string) bool { return keyRe.MatchString(s) }

// End is one end of a finding: the repo-relative, slash-separated path of the
// document it sits in, and the text it quotes (empty for a finding that quotes
// nothing, such as a record-lint rule's).
type End struct {
	Path  string
	Quote string
}

// statusStores are the record stores whose folder is the record's status: the
// folder a record sits in changes over its life while the record stays the
// same record, so the key drops it. Each store's folders are pinned to the
// owning package's own list by a test there (core/intent's buckets, core/spec's
// statuses); the issue ledger's are core/issueschema's.
var statusStores = []struct {
	root    string
	folders []string
}{
	{".abcd/development/intents", []string{"drafts", "planned", "shipped", "disciplines", "superseded"}},
	{".abcd/development/specs", []string{"open", "closed"}},
	{recordid.IssuesRelDir, issueschema.StatusDirs},
}

// StatusFoldersForTest returns the status folders this package drops from a path
// under root, or nil when root is no status-foldered store. It exists for the
// tests that pin each list to its owner's.
func StatusFoldersForTest(root string) []string {
	for _, s := range statusStores {
		if s.root == root {
			return append([]string(nil), s.folders...)
		}
	}
	return nil
}

// identityPath is p with the status folder dropped when p names a record
// directly inside a status folder of a status-foldered store:
// `.abcd/development/intents/planned/itd-N-slug.md` becomes
// `.abcd/development/intents/itd-N-slug.md`. Any other path is kept as it is,
// surrounding space trimmed.
func identityPath(p string) string {
	p = strings.TrimSpace(p)
	dir, file := path.Split(p)
	folderDir, folder := path.Split(strings.TrimSuffix(dir, "/"))
	root := strings.TrimSuffix(folderDir, "/")
	for _, s := range statusStores {
		if s.root != root {
			continue
		}
		for _, f := range s.folders {
			if f == folder && file != "" {
				return root + "/" + file
			}
		}
	}
	return p
}

// collapse is strings.Fields joined by one space: the whitespace collapse the
// consistency ingest applies to every quote.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// Derive is the key of a finding of class between a and b. Each end's path
// becomes its identity path and each quote is collapsed; the two ends are
// ordered by identity path, then by quote; and the key is the prefix and the
// first sixteen hex digits of the SHA-256 of the domain, the class and the two
// ordered (path, quote) pairs, with NUL between every field.
func Derive(class string, a, b End) string {
	ends := [2]End{
		{Path: identityPath(a.Path), Quote: collapse(a.Quote)},
		{Path: identityPath(b.Path), Quote: collapse(b.Quote)},
	}
	if ends[1].Path < ends[0].Path || (ends[1].Path == ends[0].Path && ends[1].Quote < ends[0].Quote) {
		ends[0], ends[1] = ends[1], ends[0]
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		domain, class, ends[0].Path, ends[0].Quote, ends[1].Path, ends[1].Quote,
	}, "\x00")))
	return Prefix + hex.EncodeToString(sum[:])[:16]
}
