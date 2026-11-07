package layered

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The records namespace: how long a record's slug may be in this repository.
const (
	// SlugCapKey is the key a repository sets its cap under in .abcd/config.json.
	SlugCapKey = "records.slug_cap"
	// MinSlugCap is the least a cap may be: a shorter slug names too little to
	// tell one record from the next.
	MinSlugCap = 16
	// MaxSlugCap is the most a cap may be: a filename stops at 255 bytes on every
	// filesystem abcd runs on, and the id, its family tag, the extension and a
	// temporary name's suffix come on top of the slug.
	MaxSlugCap = 200
)

// RecordSlugCap is the one value every record-minting verb cuts a slug at and
// every check of a committed record's slug holds it to: records.slug_cap in
// repoRoot's .abcd/config.json, else recordid.DefaultMaxSlugLen. A value that is
// not a whole number within the bounds, a misspelt key under records, and a file
// that cannot be read each refuse, naming the file, and a caller mints nothing.
//
// It reads the repository layer alone, so a machine's file sets nothing here. The
// cap bounds names committed for every clone, and a figure one person's machine
// held would mint a record another clone's check refuses.
func RecordSlugCap(repoRoot string) (int, error) {
	s, err := Load(Config, Roots{Repo: repoRoot, HomeRefused: true})
	if err != nil {
		return 0, err
	}
	if err := s.Claim("records", "slug_cap"); err != nil {
		return 0, err
	}
	v, err := Get(s, SlugCapKey, recordid.DefaultMaxSlugLen, func(n int) error {
		if n < MinSlugCap || n > MaxSlugCap {
			return fmt.Errorf("want a whole number of characters from %d to %d", MinSlugCap, MaxSlugCap)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return v.V, nil
}
