package provenance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// DeclaredModeRelPath is the committed file a repository declares its DEFAULT
// production mode in: how the text of a record it mints was produced, unless
// the minting verb says otherwise (itd-178). It holds one member,
// {"production_mode": "<mode>"}.
//
// The default used to ride the committed identity pin
// (.abcd/config/identity.json). That pin left the repository when each
// person's commit identity moved to their own machine (itd-2610240918547728,
// adr-2610241022541063), and the production mode, which is the repository's
// and not a person's, moved here so it stays committed.
const DeclaredModeRelPath = ".abcd/config/provenance.json"

// LegacyPinRelPath is the identity pin the production mode used to ride. It
// is read here ONLY for its production_mode member, and only while the new
// file is absent, so a repository that has not yet run the migration keeps the
// mode it declared.
const LegacyPinRelPath = ".abcd/config/identity.json"

// declaredLimit caps the bytes read from either file.
const declaredLimit = 64 << 10

type declaredFile struct {
	ProductionMode string `json:"production_mode,omitempty"`
}

// DeclaredMode is the repo's default production mode: the member of
// DeclaredModeRelPath, else (that file absent) the production_mode member of
// the legacy identity pin, else DefaultMode. It is the ONE resolution of that
// question, so no surface re-derives "absent means hand-written" for itself.
// A present value outside the closed set, or a file that does not parse, is
// an error, never a silent default.
func DeclaredMode(root string) (Mode, error) {
	m, found, err := ReadDeclaredModeFile(root)
	if err != nil || found {
		return m, err
	}
	data, ok, err := readRepoConfig(root, LegacyPinRelPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return DefaultMode, nil
	}
	var f declaredFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", fmt.Errorf("malformed %s: %w", LegacyPinRelPath, err)
	}
	m, err = modeOrDefault(strings.TrimSpace(f.ProductionMode))
	if err != nil {
		return "", fmt.Errorf("malformed %s: %w", LegacyPinRelPath, err)
	}
	return m, nil
}

// ReadDeclaredModeFile reads DeclaredModeRelPath alone: found is false when
// the file is absent. A file that is present declares a mode; one whose member
// is missing or outside the closed set is an error.
func ReadDeclaredModeFile(root string) (Mode, bool, error) {
	data, ok, err := readRepoConfig(root, DeclaredModeRelPath)
	if err != nil || !ok {
		return "", false, err
	}
	var f declaredFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", false, fmt.Errorf("malformed %s: %w", DeclaredModeRelPath, err)
	}
	m, err := ParseMode(strings.TrimSpace(f.ProductionMode))
	if err != nil {
		return "", false, fmt.Errorf("malformed %s: %w", DeclaredModeRelPath, err)
	}
	return m, true, nil
}

// WriteDeclaredMode writes m as the repo's declared default production mode,
// atomically and inside an os.Root at the repository, so a committed symlink
// under .abcd cannot aim the write outside the working tree.
func WriteDeclaredMode(root string, m Mode) error {
	if _, err := ParseMode(string(m)); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(declaredFile{ProductionMode: string(m)}); err != nil {
		return err
	}
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer osRoot.Close()
	return fsutil.WriteFileAtomicInRoot(osRoot, DeclaredModeRelPath, buf.Bytes(), 0o644)
}

// readRepoConfig reads rel through the guarded reader
// (fsutil.ReadGuardedInRoot): inside an os.Root at root, a symlinked file
// refused rather than followed, only a regular file accepted, capped at 64 KiB.
// ok is false when the file is absent.
func readRepoConfig(root, rel string) ([]byte, bool, error) {
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer osRoot.Close()
	data, err := fsutil.ReadGuardedInRoot(osRoot, rel, declaredLimit)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("reading %s: %w", rel, err)
	}
	return data, true, nil
}
