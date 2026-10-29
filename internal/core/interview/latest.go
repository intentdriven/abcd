package interview

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// maxRecordBytes caps one answers record as Latest reads it: the cap an
// answers file is held to, which no record Write composes approaches.
const maxRecordBytes = MaxAnswersBytes

// maxRecordsRead bounds how many of one interview's records Latest reads:
// the newest are read, which are the ones whose answers stand.
const maxRecordsRead = 1000

// recordNameRe is a record's file name as Write composes it: the interview,
// the stamp, and the suffix a second record in the same second takes.
var recordNameRe = regexp.MustCompile(`\A([a-z][a-z0-9-]{0,39})-([0-9]{8}T[0-9]{6}Z)(?:-([1-9][0-9]?))?\.json\z`)

// Latest returns, per question id, the value the newest of the interview
// name's records at p gives it (spc-2610170635097509): an id only an older
// record answers keeps that answer. Records are ordered by their names,
// stamp then suffix, which is the order Write gave them; a place holding no
// records directory has answered nothing, and that is no error. Every record
// read is a regular file, read strictly and capped; one that is a link, does
// not parse, or names another interview or schema is refused, never skipped,
// so an answer is never read from a file abcd did not write.
func Latest(p Place, name string) (map[string]string, error) {
	if (p.Repo == "") == (p.Home == "") {
		return nil, errors.New("interview: records are read from a repository or from the home, exactly one")
	}
	if !nameRe.MatchString(name) {
		return nil, errors.New("interview: not an interview name")
	}
	root, dir, err := openRecords(p)
	if err != nil || root == nil {
		return map[string]string{}, err
	}
	defer root.Close()

	entries, err := fs.ReadDir(root.FS(), dir)
	if err != nil {
		return nil, err
	}
	type named struct {
		file  string
		stamp string
		n     int
	}
	var records []named
	for _, e := range entries {
		m := recordNameRe.FindStringSubmatch(e.Name())
		if m == nil || m[1] != name {
			continue
		}
		n := 1
		if m[3] != "" {
			n, _ = strconv.Atoi(m[3])
		}
		records = append(records, named{file: e.Name(), stamp: m[2], n: n})
	}
	slices.SortFunc(records, func(a, b named) int {
		if a.stamp != b.stamp {
			if a.stamp < b.stamp {
				return -1
			}
			return 1
		}
		return a.n - b.n
	})
	if len(records) > maxRecordsRead {
		records = records[len(records)-maxRecordsRead:]
	}
	// Asked once for the directory, not once a record: a record git tracks
	// came with a commit, not from an interview here, and refuses them all.
	// The home's records are no checkout's tier, whatever repository holds it.
	if len(records) > 0 && p.Repo != "" {
		if err := localtier.Untracked(root.Name(), dir); err != nil {
			return nil, fmt.Errorf("interview: %w", err)
		}
	}
	out := map[string]string{}
	for _, r := range records {
		data, err := fsutil.ReadGuardedInRoot(root, path.Join(dir, r.file), maxRecordBytes)
		if err != nil {
			return nil, fmt.Errorf("interview: record %s: %w", r.file, err)
		}
		var rec Record
		if err := jsonstrict.Decode(data, &rec); err != nil {
			return nil, fmt.Errorf("interview: record %s: %w", r.file, err)
		}
		switch {
		case rec.SchemaVersion != SchemaVersion:
			return nil, fmt.Errorf("interview: record %s has schema_version %d; this abcd reads %d", r.file, rec.SchemaVersion, SchemaVersion)
		case rec.Interview != name:
			return nil, fmt.Errorf("interview: record %s is not a %s record", r.file, name)
		}
		for _, a := range rec.Answers {
			out[a.ID] = a.Value
		}
	}
	return out, nil
}

// openRecords opens the root the records at p are read through and the
// records' directory within it, or returns a nil root where no records
// directory stands. A repository's levels must each be a real directory; the
// home's are opened through the guarded home-scope walk the writer uses.
func openRecords(p Place) (*os.Root, string, error) {
	if p.Repo != "" {
		ok, err := fsutil.ProbeRealDirAll(p.Repo, RecordsRel)
		if err != nil || !ok {
			return nil, "", err
		}
		r, err := os.OpenRoot(p.Repo)
		if err != nil {
			return nil, "", err
		}
		return r, RecordsRel, nil
	}
	r, err := fsutil.OpenHomeScope(p.Home, abcdhome.Rel(homeRecordsLeaf))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return r, ".", nil
}
