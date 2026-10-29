package identity

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// LegacyPinRelPath is the identity pin itd-131 committed into a repository. It
// named one person as who commits there, so every other contributor was told
// to commit as them (iss-2610240830397793). Nothing gates on it any more; the
// migration below moves it to its owner's machine and out of the repository.
const LegacyPinRelPath = provenance.LegacyPinRelPath

// legacyPinLimit caps the bytes read from a legacy pin.
const legacyPinLimit = 64 << 10

// LegacyPin is what a repository's committed identity pin holds.
type LegacyPin struct {
	Name           string
	Email          string
	ProductionMode string
	// Unknown lists any member the migration does not know how to carry. A
	// pin holding one is never removed: removing it would drop a setting.
	Unknown []string
}

// loadLegacyPin reads the committed pin at root through the guarded reader
// (fsutil.ReadGuardedInRoot): resolved inside an os.Root at the working tree,
// a symlinked pin refused rather than followed, only a regular file accepted,
// and capped at 64 KiB. found is false when the repository has none; a pin
// that is there and refused is an error.
func loadLegacyPin(root string) (LegacyPin, bool, error) {
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		return LegacyPin{}, false, err
	}
	defer osRoot.Close()
	data, err := fsutil.ReadGuardedInRoot(osRoot, LegacyPinRelPath, legacyPinLimit)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return LegacyPin{}, false, nil
		}
		return LegacyPin{}, false, fmt.Errorf("reading %s: %w", LegacyPinRelPath, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return LegacyPin{}, false, fmt.Errorf("malformed %s: %w", LegacyPinRelPath, err)
	}
	var p LegacyPin
	for key, val := range raw {
		var s string
		known := key == "name" || key == "email" || key == "production_mode"
		if known && json.Unmarshal(val, &s) != nil {
			return LegacyPin{}, false, fmt.Errorf("malformed %s: %s is not a string", LegacyPinRelPath, key)
		}
		switch key {
		case "name":
			p.Name = strings.TrimSpace(s)
		case "email":
			p.Email = strings.TrimSpace(s)
		case "production_mode":
			p.ProductionMode = strings.TrimSpace(s)
		default:
			p.Unknown = append(p.Unknown, key)
		}
	}
	sort.Strings(p.Unknown)
	if p.ProductionMode != "" {
		if _, err := provenance.ParseMode(p.ProductionMode); err != nil {
			return LegacyPin{}, false, fmt.Errorf("malformed %s: %w", LegacyPinRelPath, err)
		}
	}
	return p, true, nil
}

// MigrationPlan is what moving a repository's committed pin would do, for one
// person on one machine. It is computed without writing anything.
type MigrationPlan struct {
	Pin LegacyPin
	// Owner is true when this person's git identity — the one git resolves in
	// the repository, or their global one — is the pinned identity. Only an
	// owner's machine receives the pin; on anyone else's it is not copied, and
	// nothing shown to them names it.
	Owner bool
	// Target is where an owner's record goes: the repository's own record
	// (keyed on its root commit), else the machine-wide default for a
	// repository with no commit yet.
	Target RecordRef
	// Existing is the record already at Target, if any. A different record is
	// never overwritten: the person's own record wins over the repository's pin.
	Existing      Record
	ExistingFound bool
	// Blocker, when set, is why the pin cannot be removed without losing
	// something: a member the migration cannot carry, a pin that names no
	// complete human identity, or a production mode that conflicts with the one
	// the repository already declares.
	Blocker string
}

// PlanMigration reads the repository's committed pin and says what moving it
// would do for the person whose home directory is home. found is false when the
// repository stores no pin.
func PlanMigration(root, home string) (MigrationPlan, bool, error) {
	pin, found, err := loadLegacyPin(root)
	if err != nil || !found {
		return MigrationPlan{}, found, err
	}
	plan := MigrationPlan{Pin: pin}
	if len(pin.Unknown) > 0 {
		plan.Blocker = LegacyPinRelPath + " holds " + strings.Join(pin.Unknown, ", ") + ", which abcd does not know where to keep, so it is left in place"
	}
	if pin.ProductionMode != "" {
		declared, has, err := provenance.ReadDeclaredModeFile(root)
		if err != nil {
			return MigrationPlan{}, true, err
		}
		if has && string(declared) != pin.ProductionMode && plan.Blocker == "" {
			plan.Blocker = LegacyPinRelPath + " declares production_mode " + pin.ProductionMode + " while " + provenance.DeclaredModeRelPath + " declares " + string(declared) + "; settle which one holds before the pin is removed"
		}
	}
	human := pin.Name != "" && pin.Email != "" && !IsToolIdentity(RoleAuthor, pin.Name, pin.Email) && !unpinnable(pin.Name) && !unpinnable(pin.Email)
	if human {
		plan.Owner, err = ownsPin(root, pin)
		if err != nil {
			return MigrationPlan{}, true, err
		}
	}
	rootSHA, err := gitutil.RootCommit(root)
	if err != nil {
		return MigrationPlan{}, true, err
	}
	plan.Target = MachineRecordRef()
	if ref, err := repoRecordRef(rootSHA); err == nil {
		plan.Target = ref
	}
	if plan.Owner {
		plan.Existing, plan.ExistingFound, err = readRecord(home, plan.Target)
		if err != nil {
			return MigrationPlan{}, true, err
		}
	}
	return plan, true, nil
}

// ownsPin reports whether the person's git identity is the pinned one: the
// identity git resolves in the repository (which the old gate held to the pin
// on its owner's machine), or their global user.name/user.email.
func ownsPin(root string, pin LegacyPin) (bool, error) {
	eff, err := EffectiveIdentity(root)
	if err != nil {
		return false, err
	}
	if eff.Name == pin.Name && eff.Email == pin.Email {
		return true, nil
	}
	name, err := gitConfigScope(root, "--global", "user.name")
	if err != nil {
		return false, err
	}
	email, err := gitConfigScope(root, "--global", "user.email")
	if err != nil {
		return false, err
	}
	return name == pin.Name && email == pin.Email, nil
}

// MigrationResult says what ApplyMigration did.
type MigrationResult struct {
	// Recorded is true when the pin was written as the owner's record at
	// RecordAt; false when an identical record was already there (Kept) or a
	// different one was (KeptDifferent).
	Recorded      bool
	Kept          bool
	KeptDifferent bool
	RecordAt      RecordRef
	// ProductionModeMoved is true when the pin's production_mode was written
	// to provenance.DeclaredModeRelPath.
	ProductionModeMoved bool
	// Removed is true when the pin left the working tree.
	Removed bool
}

// ApplyMigration carries out plan for its owner, in this order: the owner's
// record (never overwriting a different one), the production mode into its
// own committed file, then the pin's removal from the working tree. It refuses
// a plan with a Blocker, and one whose person is not the owner: on anyone
// else's machine the pin is neither copied nor removed by abcd.
func ApplyMigration(root, home string, plan MigrationPlan) (MigrationResult, error) {
	if plan.Blocker != "" {
		return MigrationResult{}, errors.New(plan.Blocker)
	}
	if !plan.Owner {
		return MigrationResult{}, errors.New("the identity this repository stores is not yours, so it is not moved to your machine")
	}
	res := MigrationResult{RecordAt: plan.Target}
	want := Record{Name: plan.Pin.Name, Email: plan.Pin.Email}
	switch {
	case plan.ExistingFound && plan.Existing == want:
		res.Kept = true
	case plan.ExistingFound:
		res.KeptDifferent = true
	default:
		if err := WriteRecord(home, plan.Target, want); err != nil {
			return res, err
		}
		res.Recorded = true
	}
	if plan.Pin.ProductionMode != "" {
		_, has, err := provenance.ReadDeclaredModeFile(root)
		if err != nil {
			return res, err
		}
		if !has {
			if err := provenance.WriteDeclaredMode(root, provenance.Mode(plan.Pin.ProductionMode)); err != nil {
				return res, err
			}
			res.ProductionModeMoved = true
		}
	}
	osRoot, err := os.OpenRoot(root)
	if err != nil {
		return res, err
	}
	defer osRoot.Close()
	if err := osRoot.Remove(filepath.FromSlash(LegacyPinRelPath)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}
	res.Removed = true
	return res, nil
}
