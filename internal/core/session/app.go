package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
)

// AppFile is the file, under BlueprintDir, naming the team's GitHub App's
// public client id: `{"client_id": "…"}`. Repository setup writes it; a local
// session (`abcd session start`) reads it, and offers GitHub's device flow,
// which needs the client id and no secret (adr-2610161400542916). It holds
// nothing secret, and a file that carries a client_secret is refused, so a
// secret is never committed beside it.
const AppFile = "github-app.json"

// maxAppFile bounds the file's size.
const maxAppFile = 64 << 10

// clientIDRe is the shape the page and the relay take for a client id.
var clientIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

// ValidClientID reports whether s is a client id the relay passes on.
func ValidClientID(s string) bool { return clientIDRe.MatchString(s) }

// ReadAppClientID reads the client id AppFile names in the checkout at root.
// No file is no client id and no error. The file is read through the
// checkout's root, so a symlink cannot lead it elsewhere.
func ReadAppClientID(root string) (string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer r.Close()
	name := path.Join(BlueprintDir, AppFile)
	info, err := r.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", name)
	}
	f, err := r.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxAppFile+1))
	if err != nil {
		return "", err
	}
	if len(raw) > maxAppFile {
		return "", fmt.Errorf("%s is larger than %d bytes", name, maxAppFile)
	}
	var app map[string]json.RawMessage
	if err := json.Unmarshal(raw, &app); err != nil {
		return "", fmt.Errorf("%s is not a JSON object", name)
	}
	if _, ok := app["client_secret"]; ok {
		return "", fmt.Errorf("%s holds a client_secret, which never belongs in the repository; remove it", name)
	}
	var id string
	if err := json.Unmarshal(app["client_id"], &id); err != nil || !ValidClientID(id) {
		return "", fmt.Errorf("%s names no client_id a GitHub App could have", name)
	}
	return id, nil
}
