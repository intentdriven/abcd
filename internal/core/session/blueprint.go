package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The blueprint (spc-2610141709430146 step 2): `abcd session blueprint` writes
// a Render Blueprint and a Dockerfile, so a team deploys its own relay once to
// its own account. The Dockerfile downloads the released abcd binary for
// linux/amd64 and checks it against the digest that release's checksums.txt
// published, fetched and pinned when the files are written.
//
// The Blueprint is render.yaml at the repository's root (step 6). Render's
// Deploy button takes only the repository's URL, with an optional
// /tree/<branch>, and documents no parameter naming another Blueprint path
// (https://render.com/docs/deploy-to-render); Render reads the Blueprint "at
// your repository's root" by default, and only its New Blueprint form takes a
// custom Blueprint Path (https://render.com/docs/infrastructure-as-code). So
// the one-click deploy needs the root file. A root render.yaml the team wrote
// for itself is never touched: the Blueprint is then written under
// .abcd/relay/, to be picked by its Blueprint Path on that form.

// ReposEnv is the environment variable `abcd session serve` reads the
// repositories it serves from, as a comma list; the blueprint sets it.
const ReposEnv = "ABCD_SESSION_REPOS"

// OriginsEnv is the environment variable `abcd session serve` reads the page
// origins it admits from, as a comma list, when no --origin is given; the
// blueprint sets it, to DefaultPageOrigin unless told others.
const OriginsEnv = "ABCD_SESSION_ORIGINS"

// DefaultPageOrigin is the session page's own origin, the one the blueprint's
// relay admits unless it is told others.
const DefaultPageOrigin = "https://abcdesign.app"

// RelaysFile is the blueprint file naming the relays this repository trusts,
// written only when the verb is given the relay's origin. The session page
// reads it from the repository as its trust anchor for relays.
const RelaysFile = "relays.json"

// BlueprintDir is where the blueprint's other files are written, relative to
// the checkout.
const BlueprintDir = ".abcd/relay"

// RootBlueprint is the Blueprint's path when the root is free: the one
// Render's Deploy button reads.
const RootBlueprint = "render.yaml"

// renderDeploy is Render's Deploy button's target; the repository's URL
// follows it.
const renderDeploy = "https://render.com/deploy?repo="

// RenderDeployURL is the Deploy button's link for the repository owner/name
// on github.com (https://render.com/docs/deploy-to-render).
func RenderDeployURL(repo string) string { return renderDeploy + "https://github.com/" + repo }

// relayAsset is the release asset the relay runs: Render builds and runs on
// linux/amd64.
const relayAsset = "abcd-linux-amd64"

// alpineDigest pins the relay image's base, alpine:3.22, to one multi-arch
// image index, so a tag moved on the registry cannot change what the relay is
// built on. Taken on 2026-10-14 from Docker Hub's registry API (the
// Docker-Content-Digest of library/alpine's 3.22 manifest, an OCI image
// index). Renew it with `docker buildx imagetools inspect alpine:3.22` when
// Alpine publishes a fix the relay needs.
const alpineDigest = "sha256:5291449c3df73caf6ed85e649dec1b9e818b39a5d8c871e97afc13e9cd5e8fa8"

// releaseOrigin is where abcd's releases are published.
const releaseOrigin = "https://github.com/intentdriven/abcd"

// releaseTagRe is the release tag a blueprint may pin: vMAJOR.MINOR.PATCH with
// an optional pre-release. The tag travels into a URL and a Dockerfile, so
// nothing else is accepted.
var releaseTagRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.]+)?$`)

// ValidReleaseTag reports whether tag is one a blueprint may pin.
func ValidReleaseTag(tag string) bool { return releaseTagRe.MatchString(tag) }

// firstSessionRelease is the first abcd release that carries `abcd session`:
// the relay container runs `abcd session serve`, so a blueprint pinning an
// older release builds a container that exits at once. No release carries the
// verb yet (v0.14.1 is the newest), so this is the next release it can ship
// in as planned; THE RELEASE CUT THAT FIRST SHIPS THE VERB SETS IT to that
// release's tag. A release the running abcd is counts as carrying the verb
// whatever this says, so a binary is never refused its own version.
const firstSessionRelease = "v0.17.0"

// releaseCore is a release tag's MAJOR, MINOR and PATCH, its pre-release
// ignored: a release candidate of the first release carrying the verb carries
// it too.
func releaseCore(tag string) [3]int {
	var v [3]int
	core, _, _ := strings.Cut(strings.TrimPrefix(tag, "v"), "-")
	for i, part := range strings.SplitN(core, ".", 3) {
		v[i], _ = strconv.Atoi(part)
	}
	return v
}

// releaseBefore reports whether release a is older than release b, by their
// MAJOR.MINOR.PATCH.
func releaseBefore(a, b string) bool {
	x, y := releaseCore(a), releaseCore(b)
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

// carriesSession reports whether release carries `abcd session`: it is no
// older than firstSessionRelease, or than running, the running abcd's own
// version when that is a release.
func carriesSession(release, running string) bool {
	floor := firstSessionRelease
	if ValidReleaseTag(running) && releaseBefore(running, floor) {
		floor = running
	}
	return !releaseBefore(release, floor)
}

// AssetFetcher fetches one named asset of a release; found is false, with no
// error, when the release or the asset does not exist.
type AssetFetcher interface {
	FetchReleaseAsset(tag, name string) (data []byte, url string, found bool, err error)
}

// BlueprintRequest names the checkout and the release to pin.
type BlueprintRequest struct {
	// Root is the checkout the blueprint is written into.
	Root string
	// Release is the abcd release tag the relay runs.
	Release string
	// Running is the running abcd's own version: a release it names carries
	// `abcd session`, whatever firstSessionRelease says.
	Running string
	// Fetch reads the release's checksums.txt.
	Fetch AssetFetcher
	// Repos are the repositories the relay serves sessions for, each
	// owner/name: the checkout's own, as the verb reads it from its origin.
	Repos []string
	// RelayOrigin is the deployed relay's https origin; given, relays.json
	// names it, and absent, no relays.json is written.
	RelayOrigin string
	// Origins are the page origins the relay admits, each an https origin;
	// none is DefaultPageOrigin.
	Origins []string
	// ClientID is the team's GitHub App's public client id, which repository
	// setup records; given, render.yaml carries it as a plain value, so
	// Render's form asks only for the secret.
	ClientID string
}

// The states a blueprint file can be left in.
const (
	BlueprintWritten   = "written"
	BlueprintReplaced  = "replaced"
	BlueprintUnchanged = "unchanged"
)

// BlueprintFile is one file the verb wrote or left.
type BlueprintFile struct {
	Path  string `json:"path"`
	State string `json:"state"`
}

// BlueprintResult is what the verb did.
type BlueprintResult struct {
	Release string          `json:"release"`
	Asset   string          `json:"asset"`
	Digest  string          `json:"sha256"`
	Source  string          `json:"checksums_url"`
	Repos   []string        `json:"repos"`
	Origins []string        `json:"origins"`
	Files   []BlueprintFile `json:"files"`
	// BlueprintPath is where render.yaml was written: RootBlueprint, or
	// under BlueprintDir beside a root render.yaml the team wrote.
	BlueprintPath string `json:"blueprint_path"`
	// DeployButton reports that Render's Deploy button deploys it, which it
	// does only from the root; DeployURL is then the button's link.
	DeployButton bool   `json:"deploy_button"`
	DeployURL    string `json:"deploy_url,omitempty"`
	// Callback is the callback URL the team's GitHub App must list for the
	// relay's web sign-in, given a relay origin: RelayCallbackURL.
	Callback string `json:"callback_url,omitempty"`
}

// BlueprintRefusal is the refusal to overwrite a file someone edited, or one
// abcd did not write. Nothing is written when it is returned.
type BlueprintRefusal struct {
	Edited []string
}

func (r *BlueprintRefusal) Error() string {
	return "refusing to overwrite " + strings.Join(r.Edited, " and ") +
		": edited since abcd wrote it, or not written by abcd; move it aside or delete it to write the blueprint again (nothing written)"
}

// blueprintFiles is the order the files are written and reported in.
var blueprintFiles = []string{"Dockerfile", "render.yaml"}

// stamped reports whether data carries abcd's blueprint stamp, edited since
// or not: a file without one is someone else's.
func stamped(data []byte) bool {
	line, rest, _ := bytes.Cut(data, []byte("\n"))
	if string(line) == "{" {
		line, _, _ = bytes.Cut(rest, []byte("\n"))
		return jsonHeaderRe.Match(line)
	}
	return headerRe.Match(line)
}

// headerRe reads a blueprint file's first line: the release it pins and the
// digest of everything after that line, as written. jsonHeaderRe is the same
// stamp in relays.json, which holds no comment: the second line, after the
// opening brace, a key the page ignores, over everything after it.
var (
	headerRe     = regexp.MustCompile(`^# abcd session blueprint (\S+) sha256:([0-9a-f]{64})$`)
	jsonHeaderRe = regexp.MustCompile(`^  "abcd_session_blueprint": "(\S+) sha256:([0-9a-f]{64})",$`)
)

// stamp prefixes body with the header line that lets a later run tell an
// untouched file from an edited one.
func stamp(release string, body []byte) []byte {
	sum := sha256.Sum256(body)
	return append([]byte(fmt.Sprintf("# abcd session blueprint %s sha256:%s\n", release, hex.EncodeToString(sum[:]))), body...)
}

// stampJSON is stamp for relays.json: body is the object's members after
// the opening brace, and the stamp is the first of them.
func stampJSON(release string, body []byte) []byte {
	sum := sha256.Sum256(body)
	return append([]byte(fmt.Sprintf("{\n  \"abcd_session_blueprint\": \"%s sha256:%s\",\n", release, hex.EncodeToString(sum[:]))), body...)
}

// untouched reports whether data is a blueprint file exactly as abcd wrote it.
func untouched(data []byte) bool {
	line, body, ok := bytes.Cut(data, []byte("\n"))
	if !ok {
		return false
	}
	re := headerRe
	if string(line) == "{" {
		if line, body, ok = bytes.Cut(body, []byte("\n")); !ok {
			return false
		}
		re = jsonHeaderRe
	}
	m := re.FindSubmatch(line)
	if m == nil {
		return false
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) == string(m[2])
}

// relayHostRe is a relay origin's host: DNS labels of letters, digits and
// inner hyphens, with an optional port.
var relayHostRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*(?::[0-9]{1,5})?$`)

// NormalizeRelayOrigin reads a relay's address as an exact https origin,
// lower-cased: https://host[:port] with nothing after it but an optional
// slash. Anything else is refused, since the page trusts the relay it names.
func NormalizeRelayOrigin(s string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
		!relayHostRe.MatchString(strings.ToLower(u.Host)) {
		return "", fmt.Errorf("%q is not an https origin: give https://host[:port], with no path", s)
	}
	return "https://" + strings.ToLower(u.Host), nil
}

// renderRelays renders relays.json, naming origin as the one relay.
func renderRelays(release, origin string) []byte {
	quoted, _ := json.Marshal(origin)
	return stampJSON(release, []byte("  \"relays\": [\n    "+string(quoted)+"\n  ]\n}\n"))
}

// pageOrigins is the page origins a blueprint names: those given, each
// normalized, or DefaultPageOrigin.
func pageOrigins(given []string) ([]string, error) {
	if len(given) == 0 {
		return []string{DefaultPageOrigin}, nil
	}
	out := make([]string, 0, len(given))
	for _, o := range given {
		n, err := NormalizeRelayOrigin(o)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// renderBlueprint renders the two files for release, pinned to digest, the
// SHA-256 of release's abcd-linux-amd64, serving repos (each owner/name) and
// admitting pages from origins (each an https origin, as NormalizeRelayOrigin
// reads it; none is DefaultPageOrigin), so nothing in either needs escaping
// inside the quoted list. It reads and writes nothing.
func renderBlueprint(release, digest string, repos, origins []string) map[string][]byte {
	return renderBlueprintAt(release, digest, repos, origins, "", RootBlueprint)
}

// renderBlueprintAt is renderBlueprint with the team's client id (a plain
// value when given, which ValidClientID has checked, else asked for by
// Render's form) and the path render.yaml is written at, which its comment
// names the way to deploy from.
func renderBlueprintAt(release, digest string, repos, origins []string, clientID, at string) map[string][]byte {
	if len(origins) == 0 {
		origins = []string{DefaultPageOrigin}
	}
	docker := `# The team's own relay for abcd's live brief sessions. It downloads the
# released abcd binary for linux/amd64, checks it against the checksum that
# release published (pinned below when this file was written), and runs the
# relay behind the host's proxy. abcd refuses to overwrite this file once it is
# edited; delete it and run ` + "`abcd session blueprint`" + ` to write it again.
FROM --platform=linux/amd64 alpine:3.22@` + alpineDigest + `
RUN wget -q -O /usr/local/bin/abcd "` + releaseOrigin + `/releases/download/` + release + `/` + relayAsset + `" \
 && echo "` + digest + `  /usr/local/bin/abcd" | sha256sum -c - \
 && chmod 0755 /usr/local/bin/abcd \
 && adduser -D -H relay
USER relay
ENV PORT=10000
EXPOSE 10000
CMD ["sh", "-c", "exec abcd session serve --listen \"0.0.0.0:${PORT}\" --behind-proxy"]
`
	how := `# Dockerfile under .abcd/relay/. Deploy it with Render's Deploy button,
# ` + RenderDeployURL(repos[0]) + `,
# or on Render choose New, then Blueprint, and pick this repository.`
	if at != RootBlueprint {
		how = `# Dockerfile beside this file. On Render, choose New, then Blueprint, pick this
# repository and set the Blueprint Path to ` + at + `.`
	}
	clientEnv := `      - key: ABCD_SESSION_GITHUB_CLIENT_ID
        sync: false`
	if clientID != "" {
		clientEnv = `      - key: ABCD_SESSION_GITHUB_CLIENT_ID
        value: "` + clientID + `"`
	}
	yaml := `# The team's own brief session relay: one free web service built from the
` + how + `
# One deploy serves all of the team's sessions. A session lives in the relay's
# memory only, so a redeploy ends every open session; that is why autoDeploy
# is off. abcd refuses to overwrite this file once it is edited.
services:
  - type: web
    name: abcd-session-relay
    runtime: docker
    plan: free
    dockerfilePath: ./.abcd/relay/Dockerfile
    dockerContext: ./.abcd/relay
    healthCheckPath: /healthz
    autoDeploy: false
    envVars:
      # The team's own GitHub App, the only kind of sign-in app the relay
      # takes. Render's form asks for the secret when the relay is created;
      # it is never in git and stays on the relay. The client id is public.
` + clientEnv + `
      - key: ABCD_SESSION_GITHUB_CLIENT_SECRET
        sync: false
      # The origins of the session pages this relay admits, as a comma
      # list of https origins: abcdesign.app unless the blueprint was told
      # others. It is not a secret.
      - key: ` + OriginsEnv + `
        value: "` + strings.Join(origins, ",") + `"
      # The repositories this relay serves sessions for, read from the
      # checkout's origin when this file was written: a comma list of
      # owner/name, or of owners for all of an owner's repositories. It is
      # not a secret. Listening beyond loopback, the relay refuses to start
      # without it.
      - key: ` + ReposEnv + `
        value: "` + strings.Join(repos, ",") + `"
`
	return map[string][]byte{
		"Dockerfile":  stamp(release, []byte(docker)),
		"render.yaml": stamp(release, []byte(yaml)),
	}
}

// maxBlueprintRead caps the read of an existing blueprint file.
const maxBlueprintRead = 1 << 20

// WriteBlueprint fetches the release's checksum, renders the blueprint and
// writes it into the checkout. It refuses, writing nothing, when the release
// tag is not one, when the release publishes no checksum for the linux/amd64
// binary, and when either file exists in a form abcd did not write.
func WriteBlueprint(req BlueprintRequest) (BlueprintResult, error) {
	res := BlueprintResult{Release: req.Release, Asset: relayAsset, Repos: req.Repos}
	if !ValidReleaseTag(req.Release) {
		return res, fmt.Errorf("%q is not a release tag (vMAJOR.MINOR.PATCH)", req.Release)
	}
	if !carriesSession(req.Release, req.Running) {
		return res, fmt.Errorf("release %s predates `abcd session`, which the relay container runs; name %s or later", req.Release, firstSessionRelease)
	}
	if len(req.Repos) == 0 {
		return res, errors.New("the relay must serve at least one repository")
	}
	for _, r := range req.Repos {
		if !ValidRepo(r) {
			return res, fmt.Errorf("%q is not an owner/name repository", r)
		}
	}
	origins, err := pageOrigins(req.Origins)
	if err != nil {
		return res, err
	}
	res.Origins = origins
	relayOrigin := ""
	if req.RelayOrigin != "" {
		o, err := NormalizeRelayOrigin(req.RelayOrigin)
		if err != nil {
			return res, err
		}
		relayOrigin = o
	}
	if req.ClientID != "" && !ValidClientID(req.ClientID) {
		return res, fmt.Errorf("%q is not a GitHub App's client id", req.ClientID)
	}
	if req.Fetch == nil {
		return res, errors.New("no way to read the release's checksums")
	}
	raw, url, found, err := req.Fetch.FetchReleaseAsset(req.Release, "checksums.txt")
	res.Source = url
	if err != nil {
		return res, fmt.Errorf("reading %s's checksums.txt: %w", req.Release, err)
	}
	if !found {
		return res, fmt.Errorf("release %s publishes no checksums.txt at %s", req.Release, url)
	}
	sums, err := parseChecksums(raw)
	if err != nil {
		return res, fmt.Errorf("%s's checksums.txt: %w", req.Release, err)
	}
	digest, ok := sums[relayAsset]
	if !ok {
		return res, fmt.Errorf("%s's checksums.txt names no %s", req.Release, relayAsset)
	}
	res.Digest = digest
	if !fsutil.IsRealDir(req.Root) {
		return res, fmt.Errorf("%s is not a directory", req.Root)
	}
	root, err := os.OpenRoot(req.Root)
	if err != nil {
		return res, err
	}
	defer root.Close()
	// The Blueprint goes to the root, where the Deploy button reads it,
	// unless the root holds a render.yaml the team wrote.
	at := RootBlueprint
	switch have, err := fsutil.ReadGuardedInRoot(root, RootBlueprint, maxBlueprintRead); {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return res, fmt.Errorf("reading %s: %w", RootBlueprint, err)
	case !stamped(have):
		at = BlueprintDir + "/" + RootBlueprint
	}
	res.BlueprintPath = at
	if at == RootBlueprint {
		res.DeployButton, res.DeployURL = true, RenderDeployURL(req.Repos[0])
	}
	byName := renderBlueprintAt(req.Release, digest, req.Repos, origins, req.ClientID, at)
	paths := map[string]string{"Dockerfile": BlueprintDir + "/Dockerfile", "render.yaml": at}
	files := map[string][]byte{}
	var names []string
	for _, name := range blueprintFiles {
		files[paths[name]] = byName[name]
		names = append(names, paths[name])
	}
	if relayOrigin != "" {
		rel := BlueprintDir + "/" + RelaysFile
		files[rel] = renderRelays(req.Release, relayOrigin)
		names = append(names, rel)
	}

	// Judge every file before writing any, so a refusal writes nothing.
	ok, err = fsutil.ProbeRealDirAll(req.Root, BlueprintDir)
	if err != nil {
		return res, fmt.Errorf("%s/ is not a plain directory: %w", BlueprintDir, err)
	}
	states := map[string]string{}
	var edited []string
	for _, rel := range names {
		if !ok && strings.HasPrefix(rel, BlueprintDir+"/") {
			states[rel] = BlueprintWritten
			continue
		}
		have, err := fsutil.ReadGuardedInRoot(root, rel, maxBlueprintRead)
		switch {
		case errors.Is(err, os.ErrNotExist):
			states[rel] = BlueprintWritten
		case err != nil:
			return res, fmt.Errorf("reading %s: %w", rel, err)
		case bytes.Equal(have, files[rel]):
			states[rel] = BlueprintUnchanged
		case untouched(have):
			states[rel] = BlueprintReplaced
		default:
			edited = append(edited, rel)
		}
	}
	if len(edited) > 0 {
		return res, &BlueprintRefusal{Edited: edited}
	}
	if err := fsutil.EnsureRealDirAll(req.Root, BlueprintDir, 0o755); err != nil {
		return res, fmt.Errorf("creating %s/: %w", BlueprintDir, err)
	}
	for _, rel := range names {
		state := states[rel]
		if state == "" {
			state = BlueprintWritten
		}
		if state != BlueprintUnchanged {
			if err := fsutil.WriteFileAtomicInRoot(root, rel, files[rel], 0o644); err != nil {
				return res, fmt.Errorf("writing %s: %w", rel, err)
			}
		}
		res.Files = append(res.Files, BlueprintFile{Path: rel, State: state})
	}
	if relayOrigin != "" {
		res.Callback = RelayCallbackURL(relayOrigin)
	}
	return res, nil
}

// parseChecksums reads a sha256sum manifest ("<64 hex>  <name>" per line, a
// '*' binary marker tolerated). A malformed line refuses the whole manifest.
func parseChecksums(raw []byte) (map[string]string, error) {
	sums := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 2 || len(f[0]) != 64 {
			return nil, errors.New("a malformed line")
		}
		if _, err := hex.DecodeString(f[0]); err != nil {
			return nil, errors.New("a malformed digest")
		}
		sums[strings.TrimPrefix(f[1], "*")] = strings.ToLower(f[0])
	}
	if len(sums) == 0 {
		return nil, errors.New("no assets")
	}
	return sums, nil
}
