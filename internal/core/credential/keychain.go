package credential

// keychain.go is the keychain home: the platform's own secret store, reached
// through its command-line tool as a subprocess (no Go dependency): the
// Keychain through /usr/bin/security on macOS, the secret service through
// secret-tool on Linux. The item is kept under abcd's service name with the
// credential's name as its account.
//
// The value never reaches an argv, which a process listing shows: security
// receives the add command on stdin in its interactive mode, the value as hex
// (-X), and secret-tool reads the value from stdin. The tool is run by an
// absolute path in a system directory, never resolved on PATH, so nothing a
// repository or a shell profile puts first is run. What the tool prints on
// failure is not repeated: a refusal names the exit status only.

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// keychainService is the service name every abcd item is kept under.
const keychainService = "abcd"

// The two platform tools.
const (
	keychainMacOS         = "security"
	keychainSecretService = "secret-tool"
)

// keychainTool is the platform tool located: which one, and its path.
type keychainTool struct {
	kind string
	path string
}

// errKeychainAbsent is a platform with no keychain tool abcd can run.
var errKeychainAbsent = errors.New("credential: the keychain home needs the platform keychain's tool " +
	"(/usr/bin/security on macOS, secret-tool from libsecret on Linux), and none is present here; " +
	"the external home and the abcd home remain")

// keychainTimeout bounds one keychain command; the platform may ask the
// person to unlock the keychain first.
var keychainTimeout = 60 * time.Second

// locateKeychain finds the platform's tool at its fixed system path. It is a
// variable so a test can point the home at a fake and never the real keychain.
var locateKeychain = func() (keychainTool, error) {
	var t keychainTool
	switch runtime.GOOS {
	case "darwin":
		t = keychainTool{kind: keychainMacOS, path: "/usr/bin/security"}
	case "linux":
		t = keychainTool{kind: keychainSecretService, path: "/usr/bin/secret-tool"}
	default:
		return t, errKeychainAbsent
	}
	if fi, err := os.Stat(t.path); err != nil || !fi.Mode().IsRegular() {
		return keychainTool{}, errKeychainAbsent
	}
	return t, nil
}

// runKeychain runs the tool with argv and stdin, returning stdout and the
// exit status (-1 when it did not run to an exit).
func runKeychain(t keychainTool, stdin []byte, args ...string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keychainTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, t.path, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return out.Bytes(), 0, nil
	case errors.As(err, &exit):
		return out.Bytes(), exit.ExitCode(), nil
	}
	return nil, -1, fmt.Errorf("credential: the keychain tool %s could not be run", t.kind)
}

// keychainLookup reads name's item. An absent item is ErrNotSet.
func keychainLookup(name string) (string, error) {
	t, err := locateKeychain()
	if err != nil {
		return "", err
	}
	var out []byte
	var code int
	switch t.kind {
	case keychainMacOS:
		out, code, err = runKeychain(t, nil, "find-generic-password", "-s", keychainService, "-a", name, "-w")
		if code == 44 { // errSecItemNotFound
			return "", notSetError{name: name, why: "the keychain holds no item for it"}
		}
	default:
		out, code, err = runKeychain(t, nil, "lookup", "service", keychainService, "account", name)
		if code == 1 && len(out) == 0 {
			return "", notSetError{name: name, why: "the keychain holds no item for it"}
		}
	}
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("credential: the keychain refused to read %s (%s exited %d); unlock it, or choose another home", name, t.kind, code)
	}
	v := strings.TrimSuffix(strings.TrimSuffix(string(out), "\n"), "\r")
	if v == "" {
		return "", notSetError{name: name, why: "the keychain's item for it is empty"}
	}
	if err := CheckValue(v); err != nil {
		return "", fmt.Errorf("credential: the keychain's item for %s: %w", name, err)
	}
	return v, nil
}

// keychainStore adds name's item holding value, then reads it back: the
// interactive mode security is driven through reports a failed command on
// stderr rather than in its exit, so the read is what proves the write.
func keychainStore(name, value string) error {
	t, err := locateKeychain()
	if err != nil {
		return err
	}
	var code int
	switch t.kind {
	case keychainMacOS:
		line := fmt.Sprintf("add-generic-password -s %s -a %s -l abcd:%s -X %s\n", keychainService, name, name, hex.EncodeToString([]byte(value)))
		_, code, err = runKeychain(t, []byte(line), "-i")
	default:
		_, code, err = runKeychain(t, []byte(value), "store", "--label=abcd:"+name, "service", keychainService, "account", name)
	}
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("credential: the keychain refused to store %s (%s exited %d), so nothing was stored", name, t.kind, code)
	}
	got, err := keychainLookup(name)
	switch {
	case errors.Is(err, ErrNotSet):
		return fmt.Errorf("credential: the keychain did not keep %s (it may be locked, or have refused the item), so nothing was stored", name)
	case err != nil:
		return err
	case got != value:
		return fmt.Errorf("credential: the keychain's item for %s does not read back as written; remove it by hand", name)
	}
	return nil
}
