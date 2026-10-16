package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Setting up live sessions at repository setup (spc-2610141709430146 step
// 6, adr-2610161400542916): the team registers its own GitHub App, once,
// through GitHub's App manifest flow
// (https://docs.github.com/en/apps/sharing-github-apps/registering-a-github-app-from-a-manifest).
// A form posts a pre-filled manifest to GitHub, the person clicks Create,
// GitHub redirects back with a code, and the code converts, within the hour,
// into the app's credentials. abcd keeps only the public client id, in
// .abcd/relay/github-app.json; the client secret, the webhook secret and the
// private key are never kept, written or shown.

// Where sessions may run, as repository setup records it.
const (
	WhereLocal = "local"
	WhereCloud = "cloud"
	WhereBoth  = "both"
)

// ValidWhere reports whether w is where sessions may run.
func ValidWhere(w string) bool { return w == WhereLocal || w == WhereCloud || w == WhereBoth }

// AllowsLocal reports whether w allows a session on the host's computer.
func AllowsLocal(w string) bool { return w == WhereLocal || w == WhereBoth }

// AllowsCloud reports whether w allows a session on the team's relay.
func AllowsCloud(w string) bool { return w == WhereCloud || w == WhereBoth }

// maxAppName is the longest name GitHub gives an app.
const maxAppName = 34

// AppManifest is the GitHub App manifest repository setup posts. It asks for
// no webhook, so it carries no hook_attributes and the app receives no event.
type AppManifest struct {
	Name               string            `json:"name"`
	URL                string            `json:"url"`
	RedirectURL        string            `json:"redirect_url"`
	CallbackURLs       []string          `json:"callback_urls,omitempty"`
	Description        string            `json:"description"`
	Public             bool              `json:"public"`
	DefaultPermissions map[string]string `json:"default_permissions"`
	DefaultEvents      []string          `json:"default_events"`
}

// NewAppManifest is the manifest for the repository owner/name's app, where
// sessions may run where, returning to redirectURL once GitHub has created
// it. A cloud relay signs in through GitHub's web flow, which returns to the
// session page, so the page's address is the callback for cloud and both; a
// local session signs in through the device flow, which needs none. The
// permissions are the ones the page's work needs, and no more: Contents and
// Pull requests read and write, Metadata read.
func NewAppManifest(repo, where, redirectURL string) (AppManifest, error) {
	if !ValidRepo(repo) {
		return AppManifest{}, fmt.Errorf("%q is not an owner/name repository", repo)
	}
	if !ValidWhere(where) {
		return AppManifest{}, fmt.Errorf("%q is not where sessions may run: local, cloud or both", where)
	}
	owner, name, _ := strings.Cut(repo, "/")
	m := AppManifest{
		Name:        appName(owner, name),
		URL:         "https://github.com/" + repo,
		RedirectURL: redirectURL,
		Description: "Signs the team in to live brief sessions for " + repo + ". Registered by abcd's repository setup.",
		Public:      false,
		DefaultPermissions: map[string]string{
			"contents":      "write",
			"pull_requests": "write",
			"metadata":      "read",
		},
		DefaultEvents: []string{},
	}
	if AllowsCloud(where) {
		m.CallbackURLs = []string{DefaultPageOrigin + "/" + repo + "/brief/"}
	}
	return m, nil
}

// appName is the app's name, within GitHub's length: the owner and the
// repository, then "brief". GitHub's own page lets the person change it
// before creating the app, should the name be taken.
func appName(owner, name string) string {
	n := owner + " " + name + " brief"
	if len(n) > maxAppName {
		n = strings.TrimRight(n[:maxAppName], " -_.")
	}
	return n
}

// ManifestFormURL is the page the manifest form posts to: an organisation's
// new-app page, or the person's own, carrying state, which GitHub returns.
func ManifestFormURL(owner string, org bool, state string) string {
	base := "https://github.com/settings/apps/new"
	if org {
		base = "https://github.com/organizations/" + url.PathEscape(owner) + "/settings/apps/new"
	}
	return base + "?state=" + url.QueryEscape(state)
}

// CreatedApp is what repository setup keeps of the manifest conversion's
// answer: the public client id and where the app's pages are. It has no
// field a secret could land in.
type CreatedApp struct {
	ClientID   string `json:"client_id"`
	Slug       string `json:"slug"`
	OwnerLogin string `json:"owner"`
	OwnerType  string `json:"owner_type"`
}

// slugRe is an app's slug as it travels into GitHub's addresses.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,99}$`)

// ParseConversion reads the manifest conversion's answer
// (POST /app-manifests/{code}/conversions), keeping only CreatedApp's
// fields: the secrets in the answer are never decoded into anything.
func ParseConversion(body []byte) (CreatedApp, error) {
	var raw struct {
		ClientID string `json:"client_id"`
		Slug     string `json:"slug"`
		Owner    struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return CreatedApp{}, errors.New("GitHub's answer is not the app it created")
	}
	app := CreatedApp{ClientID: raw.ClientID, Slug: raw.Slug, OwnerLogin: raw.Owner.Login, OwnerType: raw.Owner.Type}
	if !ValidClientID(app.ClientID) || !slugRe.MatchString(app.Slug) || !ValidOwner(app.OwnerLogin) {
		return CreatedApp{}, errors.New("GitHub's answer names no app abcd can use")
	}
	return app, nil
}

// SettingsURL is the app's settings page, where its device flow is switched
// on and its client secret generated.
func (a CreatedApp) SettingsURL() string {
	if a.OwnerType == "Organization" {
		return "https://github.com/organizations/" + a.OwnerLogin + "/settings/apps/" + a.Slug
	}
	return "https://github.com/settings/apps/" + a.Slug
}

// InstallURL is the page that installs the app on a repository.
func (a CreatedApp) InstallURL() string {
	return "https://github.com/apps/" + a.Slug + "/installations/new"
}

// WriteAppFile writes AppFile, naming clientID and nothing else, under
// BlueprintDir through the checkout's root, so a symlink cannot lead the
// write elsewhere.
func WriteAppFile(root, clientID string) error {
	if !ValidClientID(clientID) {
		return fmt.Errorf("%q is not a GitHub App's client id", clientID)
	}
	body, err := json.MarshalIndent(map[string]string{"client_id": clientID}, "", "  ")
	if err != nil {
		return err
	}
	if err := fsutil.EnsureRealDirAll(root, BlueprintDir, 0o755); err != nil {
		return fmt.Errorf("creating %s/: %w", BlueprintDir, err)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	return fsutil.WriteFileAtomicInRoot(r, path.Join(BlueprintDir, AppFile), append(body, '\n'), 0o644)
}
