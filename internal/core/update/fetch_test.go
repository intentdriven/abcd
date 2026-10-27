package update

import (
	"errors"
	"strings"
	"testing"
)

// TestFetchVerifiedChecksBothRecords: the download setup offers is the
// release's own bytes, checked against its checksums.txt and its build
// provenance; a mismatch or an unverified provenance returns nothing.
func TestFetchVerifiedChecksBothRecords(t *testing.T) {
	o := newTestOrigin(t)
	bin := []byte("release-binary-bytes")
	o.addRelease("v0.7.0", testAssetName, bin)
	u := testUpdater(t, o)

	got, err := u.FetchVerified("v0.7.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Bytes) != string(bin) || len(got.SHA256) != 64 {
		t.Fatalf("fetched %q / %q", got.Bytes, got.SHA256)
	}

	o.addRelease("v0.7.1", testAssetName, []byte("what-the-manifest-says"))
	o.assets["v0.7.1"][testAssetName] = []byte("other-bytes")
	if _, err := u.FetchVerified("v0.7.1", nil); err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Errorf("a checksum mismatch returned %v", err)
	}

	u.provenance = refusingProvenance{}
	if _, err := u.FetchVerified("v0.7.0", nil); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Errorf("an unverified provenance returned %v", err)
	}
}

type refusingProvenance struct{}

func (refusingProvenance) verify(string, []byte) (string, error) {
	return "", errors.New("no attestation names these bytes")
}
