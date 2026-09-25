package banlist

import (
	"fmt"
	"strconv"
	"strings"
)

// MigrateResult is the outcome of a legacy-store migration.
type MigrateResult struct {
	// Path is the private store, repo-relative.
	Path string `json:"path"`
	// Migrated reports that the store was legacy and is now keyed; a store that
	// already declared the keyed format is left byte-for-byte alone (false).
	Migrated bool `json:"migrated"`
	// Entries counts the entries the store holds after the call.
	Entries int `json:"entries"`
}

// MigratePrivate converts a legacy private store — no format declaration, every
// entry a whole-line pattern — to the keyed format in place, so the verbs that
// refuse a legacy store (add, remove, a generated-block sync) can act on it.
//
// Nothing a line matches changes. Each entry keeps its exact pattern bytes under
// the key the guard already prints for it, entry-<its line in the legacy file>, so
// a refusal names the same key before and after. The declaration becomes line 1;
// every comment and blank line survives in place, and only a leading byte-order
// mark and the trailing blanks of an entry line (which neither reader ever
// matched) are dropped. Each composed line is proved to parse back to exactly its
// key and pattern before anything is written, and a line that would not is refused
// by number. A keyed store is left alone; an absent one is ErrNoStore. The store's
// contract holds as for AddPrivate: it must be gitignored, and the write is
// contained, atomic, 0600 and under the store's lock. No pattern is ever quoted.
func MigratePrivate(repoRoot string) (MigrateResult, error) {
	res := MigrateResult{Path: PrivateRelPath}
	data, err := readPrivate(repoRoot)
	if err != nil {
		return res, err
	}
	entries, keyed, err := parse(data)
	if err != nil {
		return res, err
	}
	if keyed {
		res.Entries = countParsed(entries)
		return res, nil
	}
	if err := requireIgnoredStore(repoRoot); err != nil {
		return res, err
	}
	err = withPrivateLock(repoRoot, func() error {
		data, err := readPrivate(repoRoot)
		if err != nil {
			return err
		}
		entries, keyed, err := parse(data)
		if err != nil {
			return err
		}
		res.Entries = countParsed(entries)
		if keyed {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		out := make([]string, 0, len(lines)+1)
		out = append(out, privateFormatDecl)
		for i, raw := range lines {
			if i == 0 {
				raw = strings.TrimPrefix(raw, utf8BOM)
			}
			line := trimLead(trimTrail(raw))
			if line == "" || strings.HasPrefix(line, "#") {
				out = append(out, raw)
				continue
			}
			key := "entry-" + strconv.Itoa(i+1)
			if !composedLineRoundTrips(key, line) {
				return fmt.Errorf("%w: line %d of %s does not survive keying as %s (the value is withheld); key it by hand",
					ErrInvalidPattern, i+1, PrivateRelPath, key)
			}
			out = append(out, key+" "+line)
		}
		if err := writePrivateStore(repoRoot, []byte(strings.Join(out, "\n"))); err != nil {
			return err
		}
		res.Migrated = true
		return nil
	})
	if err != nil {
		return MigrateResult{Path: PrivateRelPath}, err
	}
	return res, nil
}
