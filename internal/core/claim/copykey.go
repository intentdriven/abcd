package claim

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The working copy's local state, in the checkout's local tier. The tier is
// never created on the way to a reservation; `abcd ahoy` creates it.
const (
	tierRelDir   = ".abcd/.work.local"
	stateRelDir  = tierRelDir + "/reservation"
	copyKeyRel   = stateRelDir + "/copy-key"
	fetchedAtRel = stateRelDir + "/fetched-at"
	copyKeyLen   = 32
	stateMode    = 0o600
)

// validCopyKey reports whether s is a working-copy key: 32 lower-case hex
// characters, naming neither a path nor a host.
func validCopyKey(s string) bool {
	if len(s) != copyKeyLen {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// tierPresent refuses a checkout without its local tier.
func tierPresent(root string) error {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(tierRelDir)))
	if err != nil || !fi.IsDir() {
		return refuse("%s/ is not a directory in this checkout, so there is nowhere to keep this copy's key; run abcd ahoy to set it up; %s",
			tierRelDir, nothingPushed)
	}
	return nil
}

// copyKey is this working copy's key: minted from crypto/rand the first time a
// reservation verb runs in the checkout at root, and read back every time
// after. The mint is an exclusive create at mode 0600, so two first runs
// cannot mint two keys: the loser reads the winner's.
func copyKey(root string) (string, error) {
	if err := tierPresent(root); err != nil {
		return "", err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", refuse("cannot open the checkout: %s; %s", redacted(err), nothingPushed)
	}
	defer r.Close()
	if err := r.Mkdir(stateRelDir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", refuse("cannot make %s/: %s; %s", stateRelDir, redacted(err), nothingPushed)
	}
	var raw [copyKeyLen / 2]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", refuse("cannot mint this copy's key: %v; %s", err, nothingPushed)
	}
	minted := hex.EncodeToString(raw[:])
	err = fsutil.CreateExclusiveIn(r, copyKeyRel, []byte(minted+"\n"), stateMode)
	if err == nil {
		return minted, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return "", refuse("cannot write %s: %s; %s", copyKeyRel, redacted(err), nothingPushed)
	}
	// Another run minted it first, or an earlier one did. A racing writer's
	// create precedes its write by an instant, so an empty read is retried.
	for attempt := 0; ; attempt++ {
		data, err := fsutil.ReadGuardedInRoot(r, copyKeyRel, 256)
		if err != nil {
			return "", refuse("cannot read %s: %s; %s", copyKeyRel, redacted(err), nothingPushed)
		}
		key := strings.TrimSpace(string(data))
		if validCopyKey(key) {
			return key, nil
		}
		if key != "" || attempt >= 20 {
			return "", refuse("%s does not hold a working-copy key (32 lower-case hex characters); remove it and retry, "+
				"which mints a new one, and this copy's earlier reservations then read as another copy's; %s", copyKeyRel, nothingPushed)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
