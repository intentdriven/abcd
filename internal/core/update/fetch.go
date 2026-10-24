package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// FetchedRelease is a release binary downloaded and verified, before anything
// is written: its bytes, the sha256 checksums.txt names for them, and the
// signer of the build provenance that names them.
type FetchedRelease struct {
	Bytes  []byte
	SHA256 string
	Signer string
}

// FetchVerified downloads tag's binary for this platform and checks it the
// way Apply checks a swap and the README's install line checks a first
// install: against the release's own checksums.txt, then against the build
// provenance the release workflow signed. It writes nothing; the caller
// writes the bytes. It is the fetch `ahoy install` makes on a yes to the
// question naming the download (itd-2610240823055889), where there is no
// file at the target for Apply to prove abcd's.
func (u *Updater) FetchVerified(tag string, progress io.Writer) (FetchedRelease, error) {
	sums, found, err := u.fetchChecksums(tag)
	if err != nil {
		return FetchedRelease{}, fmt.Errorf("fetching %s checksums: %w", tag, err)
	}
	if !found {
		return FetchedRelease{}, fmt.Errorf("release %s has no checksums.txt at %s", tag, u.origin)
	}
	wantHex, ok := sums[u.assetName]
	if !ok {
		return FetchedRelease{}, fmt.Errorf("release %s publishes no %s", tag, u.assetName)
	}
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		return FetchedRelease{}, fmt.Errorf("release %s carries a malformed digest for %s", tag, u.assetName)
	}
	body, size, err := u.get(u.assetURL(tag, u.assetName), maxAssetBytes)
	if err != nil {
		return FetchedRelease{}, fmt.Errorf("downloading %s %s: %w", tag, u.assetName, err)
	}
	defer body.Close()
	reader := io.Reader(body)
	if progress != nil {
		reader = &progressReader{r: body, total: size, out: progress, label: u.assetName + " " + tag}
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return FetchedRelease{}, fmt.Errorf("downloading %s %s: %w", tag, u.assetName, err)
	}
	if progress != nil {
		fmt.Fprintln(progress)
	}
	if got := sha256.Sum256(b); !bytes.Equal(got[:], want) {
		return FetchedRelease{}, fmt.Errorf("verifying %s against the release manifest: the download's sha256 is %x, the manifest names %x", u.assetName, got, want)
	}
	signer, err := u.verifyProvenance(tag, want)
	if err != nil {
		return FetchedRelease{}, fmt.Errorf("the release's %s matches its checksums.txt, but its build provenance could not be verified: %w; %s",
			u.assetName, err, u.remedyInstallByHand(tag, "~/.local/bin/abcd"))
	}
	return FetchedRelease{Bytes: b, SHA256: wantHex, Signer: signer}, nil
}
