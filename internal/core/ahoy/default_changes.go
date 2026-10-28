package ahoy

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The default-change table (spc-2610170635097509, step 4): each entry is a
// setup key and the itd-/iss- record that changed the key's default. The
// release is never written here: it is the release whose digest cites the
// record (internal/core/releasedigest), so a release is derived at the cut,
// as the changelog is, and a record no digest cites yet (a dev build) counts
// in no release. A key the repository's own layer sets is unaffected by its
// default, so Waiting lists a changed default only where the repository takes
// the default.

//go:embed defaults/default-changes.json
var embeddedDefaultChanges []byte

// defaultChangesSchema is the table's schema version; another is refused.
const defaultChangesSchema = 1

// maxDefaultChangesBytes caps the table, as every embedded default is capped.
const maxDefaultChangesBytes = 64 << 10

// maxDefaultChangeRecordBytes caps a record id, the cut's own bound.
const maxDefaultChangeRecordBytes = 32

// defaultChangeRecordRe is a record id the table may name: the grammar the
// digest's citations use, so the record can be found in a release.
var defaultChangeRecordRe = regexp.MustCompile(`\A(?:itd|iss)-[0-9]+\z`)

// DefaultChange is one entry: the setup key whose default changed and the
// record that changed it.
type DefaultChange struct {
	Key    string `json:"key"`
	Record string `json:"record"`
}

// configValueSlots are the setup keys whose answer the repository's layer
// holds in .abcd/config.json, each with its section and field: the keys
// whose "set here or taken from the default" Waiting can tell, and so the
// only keys a default change may name. They are the four the install
// collects (collectMissingValues) and loadPersistedInstallConfig reads.
var configValueSlots = map[string][2]string{
	"visibility":     {"repo", "visibility"},
	"docs_target":    {"docs", "target"},
	"oracle_backend": {"oracle", "backend"},
	"scan_deep":      {"scan", "deep"},
}

// loadDefaultChanges is the loader the binary reads its table through: it
// parses table on first use and hands every later call the same result. The
// binary loads the embedded table with it; a test hands it a table of its own
// (the embedded one with a fixture entry added) and runs that to the offer.
func loadDefaultChanges(table []byte) func() ([]DefaultChange, error) {
	return sync.OnceValues(func() ([]DefaultChange, error) {
		return parseDefaultChanges(table)
	})
}

// defaultChanges is the table this binary carries.
var defaultChanges = loadDefaultChanges(embeddedDefaultChanges)

// parseDefaultChanges reads a table strictly: one JSON document, no unknown
// field, no repeated key, nothing after it, capped. Each entry must name a
// setup question key (one SetupFixedValues or helpFor answers) whose setting
// the repository layer holds (configValueSlots), and a record id; an entry
// listed twice is refused.
func parseDefaultChanges(data []byte) ([]DefaultChange, error) {
	if len(data) > maxDefaultChangesBytes {
		return nil, fmt.Errorf("default-change table exceeds the %d-byte cap", maxDefaultChangesBytes)
	}
	var doc struct {
		SchemaVersion int             `json:"schema_version"`
		Changes       []DefaultChange `json:"changes"`
	}
	if err := jsonstrict.Decode(data, &doc); err != nil {
		return nil, fmt.Errorf("default-change table: %w", err)
	}
	switch {
	case doc.SchemaVersion == 0:
		return nil, errors.New("default-change table: schema_version is missing")
	case doc.SchemaVersion != defaultChangesSchema:
		return nil, fmt.Errorf("default-change table: schema_version %d is not %d, the one this binary reads", doc.SchemaVersion, defaultChangesSchema)
	}
	seen := map[DefaultChange]bool{}
	out := make([]DefaultChange, 0, len(doc.Changes))
	for i, c := range doc.Changes {
		at := fmt.Sprintf("default-change table: entry %d", i+1)
		if !isSetupQuestionKey(c.Key) {
			return nil, fmt.Errorf("%s: %s is not a setup question key", at, termsafe.DescribeRefused(c.Key))
		}
		if _, ok := configValueSlots[c.Key]; !ok {
			return nil, fmt.Errorf("%s: %s has no setting in .abcd/config.json, so whether a repository takes its default cannot be told", at, c.Key)
		}
		if len(c.Record) > maxDefaultChangeRecordBytes || !defaultChangeRecordRe.MatchString(c.Record) {
			return nil, fmt.Errorf("%s: %s is not a record id (want itd-N or iss-N)", at, termsafe.DescribeRefused(c.Record))
		}
		if seen[c] {
			return nil, fmt.Errorf("%s: %s and %s are listed twice", at, c.Key, c.Record)
		}
		seen[c] = true
		out = append(out, c)
	}
	return out, nil
}

// isSetupQuestionKey reports whether setup asks a question keyed key: one
// whose answers are fixed (SetupFixedValues) or one core holds help for.
func isSetupQuestionKey(key string) bool {
	if _, ok := SetupFixedValues(key); ok {
		return true
	}
	_, ok := helpFor(key)
	return ok
}
