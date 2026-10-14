package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The blueprint (spc-2610141709430146 step 2): `abcd session blueprint` writes
// a Render Blueprint and a Dockerfile under .abcd/relay/, so a team deploys its
// own relay once to its own account. The Dockerfile downloads the released
// abcd binary for linux/amd64 and checks it against the digest that release's
// checksums.txt published, fetched and pinned when the files are written.

// BlueprintDir is where the blueprint is written, relative to the checkout.
const BlueprintDir = ".abcd/relay"

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
	// Fetch reads the release's checksums.txt.
	Fetch AssetFetcher
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
	Files   []BlueprintFile `json:"files"`
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

// headerRe reads a blueprint file's first line: the release it pins and the
// digest of everything after that line, as written.
var headerRe = regexp.MustCompile(`^# abcd session blueprint (\S+) sha256:([0-9a-f]{64})$`)

// stamp prefixes body with the header line that lets a later run tell an
// untouched file from an edited one.
func stamp(release string, body []byte) []byte {
	sum := sha256.Sum256(body)
	return append([]byte(fmt.Sprintf("# abcd session blueprint %s sha256:%s\n", release, hex.EncodeToString(sum[:]))), body...)
}

// untouched reports whether data is a blueprint file exactly as abcd wrote it.
func untouched(data []byte) bool {
	line, body, ok := bytes.Cut(data, []byte("\n"))
	if !ok {
		return false
	}
	m := headerRe.FindSubmatch(line)
	if m == nil {
		return false
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) == string(m[2])
}

// RenderBlueprint renders the two files for release, pinned to digest, the
// SHA-256 of release's abcd-linux-amd64. It reads and writes nothing.
func RenderBlueprint(release, digest string) map[string][]byte {
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
	yaml := `# The team's own brief session relay: one free web service built from the
# Dockerfile beside this file. On Render, choose New, then Blueprint, pick this
# repository and set the Blueprint path to .abcd/relay/render.yaml. One deploy
# serves all of the team's sessions. A session lives in the relay's memory
# only, so a redeploy ends every open session; that is why autoDeploy is off.
# abcd refuses to overwrite this file once it is edited.
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
      # The team's own GitHub sign-in app. Set both in Render's dashboard
      # (the service's Environment), never in git. The secret stays on the
      # relay; the client id is public.
      - key: ABCD_SESSION_GITHUB_CLIENT_ID
        sync: false
      - key: ABCD_SESSION_GITHUB_CLIENT_SECRET
        sync: false
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
	res := BlueprintResult{Release: req.Release, Asset: relayAsset}
	if !ValidReleaseTag(req.Release) {
		return res, fmt.Errorf("%q is not a release tag (vMAJOR.MINOR.PATCH)", req.Release)
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
	files := RenderBlueprint(req.Release, digest)

	if !fsutil.IsRealDir(req.Root) {
		return res, fmt.Errorf("%s is not a directory", req.Root)
	}
	// Judge both files before writing either, so a refusal writes nothing.
	ok, err = fsutil.ProbeRealDirAll(req.Root, BlueprintDir)
	if err != nil {
		return res, fmt.Errorf("%s/ is not a plain directory: %w", BlueprintDir, err)
	}
	states := map[string]string{}
	var edited []string
	if ok {
		root, err := os.OpenRoot(req.Root)
		if err != nil {
			return res, err
		}
		defer root.Close()
		for _, name := range blueprintFiles {
			rel := BlueprintDir + "/" + name
			have, err := fsutil.ReadGuardedInRoot(root, rel, maxBlueprintRead)
			switch {
			case errors.Is(err, os.ErrNotExist):
				states[name] = BlueprintWritten
			case err != nil:
				return res, fmt.Errorf("reading %s: %w", rel, err)
			case bytes.Equal(have, files[name]):
				states[name] = BlueprintUnchanged
			case untouched(have):
				states[name] = BlueprintReplaced
			default:
				edited = append(edited, rel)
			}
		}
	}
	if len(edited) > 0 {
		return res, &BlueprintRefusal{Edited: edited}
	}
	if err := fsutil.EnsureRealDirAll(req.Root, BlueprintDir, 0o755); err != nil {
		return res, fmt.Errorf("creating %s/: %w", BlueprintDir, err)
	}
	root, err := os.OpenRoot(req.Root)
	if err != nil {
		return res, err
	}
	defer root.Close()
	for _, name := range blueprintFiles {
		rel := BlueprintDir + "/" + name
		state := states[name]
		if state == "" {
			state = BlueprintWritten
		}
		if state != BlueprintUnchanged {
			if err := fsutil.WriteFileAtomicInRoot(root, rel, files[name], 0o644); err != nil {
				return res, fmt.Errorf("writing %s: %w", rel, err)
			}
		}
		res.Files = append(res.Files, BlueprintFile{Path: rel, State: state})
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
