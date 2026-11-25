package spotcheck

// beacon.go is the draw's seed (spc-2610301925429250 approach 4): one round
// of drand's quicknet chain, a public randomness beacon no agent can steer,
// fetched only once a window has closed and verified here, at the draw,
// under the chain's pinned public key. A round whose signature does not
// verify is no seed at all: the draw is refused and nothing is written.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudflare/circl/sign/bls"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/urlguard"
)

// ChainHash is the quicknet chain's hash, as its info reply names it, read
// on 31 October 2026 (.abcd/development/research/notes/
// 2026-10-31-drand-beacon-verification-sota.md). Every round is asked of the
// chain by this hash.
const ChainHash = "52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971"

// chainPoint is the quicknet chain's group public key, a compressed point on
// G2 (scheme bls-unchained-g1-rfc9380: signatures on G1, keys on G2), read
// from the same info reply.
const chainPoint = "83cf0f2896adee7eb8b5f01fcad3912212c437e0073e911fb90022d3e760183c8c4b450b6a0a6c3ac6a5776a2d1064510d1fec758c921cc22b0e17e63aaf4bcb5ed66304de9cf809bd274ca73bab4af5a6e9c76a4bc09e76eae8991ef5ece45a"

// The beacon's bounds.
const (
	// BeaconTimeout bounds the beacon's fetch, every host it asks together.
	BeaconTimeout = 10 * time.Second
	// maxBeaconBytes bounds one reply.
	maxBeaconBytes = 4 << 10
	// sigHexLen is a G1 signature's length in hex digits: 48 bytes.
	sigHexLen = 96
)

// beaconHosts are the hosts a round is asked of, in order: the next is asked
// only when the one before could not be reached.
var beaconHosts = []string{"api.drand.sh", "api2.drand.sh", "api3.drand.sh"}

// Beacon is the round a cycle was drawn from, as the cycle record keeps it,
// so anyone can repeat the verification later.
type Beacon struct {
	URL        string    `json:"url"`
	ChainHash  string    `json:"chain_hash"`
	Round      uint64    `json:"round"`
	Signature  string    `json:"signature"`
	Randomness string    `json:"randomness"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// beaconGet reads one reply, bounded at maxBeaconBytes: the seam a test
// serves rounds through. The shipping one goes through urlguard's pinned
// download, to the beacon's hosts alone, with no proxy and the address policy
// on every dial.
var beaconGet = func(ctx context.Context, rawURL string) ([]byte, error) {
	dl := urlguard.NewDownload(urlguard.BlockedIP, false, BeaconTimeout, urlguard.OverrideFreeRoots(), beaconHosts...)
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if err := dl.Admit(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "abcd-spotcheck")
	resp, err := dl.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("answered %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBeaconBytes+1))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// errUnreachable marks a host the beacon's fetch could not read a reply from,
// the one failure on which the next host is asked.
var errUnreachable = errors.New("unreachable")

// fetchBeacon is the chain's latest round, asked of each host in turn until
// one answers, within BeaconTimeout, and verified under the pinned key. A
// reply that is read and refused (over its bound, malformed, of another
// chain, a signature that does not verify) refuses the fetch: no other host
// is asked, since a host that answers wrongly is no reason to trust the next.
// Every refusal names the beacon, and the round where one was read.
func fetchBeacon(parent context.Context, now time.Time) (Beacon, error) {
	ctx, cancel := context.WithTimeout(parent, BeaconTimeout)
	defer cancel()
	var last error
	for _, host := range beaconHosts {
		u := "https://" + host + "/" + ChainHash + "/public/latest"
		body, err := beaconGet(ctx, u)
		if err != nil {
			last = fmt.Errorf("%s: %w (%s)", host, errUnreachable, redact(err))
			if ctx.Err() != nil {
				break
			}
			continue
		}
		b, err := readBeacon(body)
		if err != nil {
			return Beacon{}, fmt.Errorf("the beacon at %s: %w", host, err)
		}
		b.URL, b.FetchedAt = u, now.UTC().Truncate(time.Second)
		return b, nil
	}
	return Beacon{}, fmt.Errorf("the beacon could not be reached: %w", last)
}

// beaconReplyFields are the fields of a reply the fetch reads; any other
// field is ignored, since the reply is the beacon's to extend.
type beaconReplyFields struct {
	Round      json.RawMessage `json:"round"`
	Signature  *string         `json:"signature"`
	Randomness *string         `json:"randomness"`
	ChainHash  *string         `json:"chain_hash"`
}

// readBeacon is one reply read and verified: within maxBeaconBytes, one JSON
// object with a positive whole round and a 96-hex signature, of the pinned
// chain where it names one, whose randomness, where it carries one, is the
// hash of its signature, and whose signature verifies for its round.
func readBeacon(body []byte) (Beacon, error) {
	if len(body) > maxBeaconBytes {
		return Beacon{}, fmt.Errorf("the reply is over the %d bytes a beacon reply may hold", maxBeaconBytes)
	}
	if err := jsonstrict.NoDuplicateKeys(body); err != nil {
		return Beacon{}, fmt.Errorf("the reply does not read as a beacon round: %s", redact(err))
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	var f beaconReplyFields
	if err := dec.Decode(&f); err != nil {
		return Beacon{}, fmt.Errorf("the reply does not read as a beacon round: %s", redact(err))
	}
	if _, err := dec.Token(); err != io.EOF {
		return Beacon{}, errors.New("the reply holds more than one beacon round")
	}
	if f.Round == nil || f.Signature == nil {
		return Beacon{}, errors.New("the reply names no round and signature")
	}
	round, err := parseRound(string(f.Round))
	if err != nil {
		return Beacon{}, err
	}
	if f.ChainHash != nil && *f.ChainHash != ChainHash {
		return Beacon{}, fmt.Errorf("round %d is of another chain than the pinned one", round)
	}
	sig, err := sigBytes(*f.Signature)
	if err != nil {
		return Beacon{}, fmt.Errorf("round %d: %w", round, err)
	}
	r, err := verifyRound(round, sig)
	if err != nil {
		return Beacon{}, fmt.Errorf("round %d: %w", round, err)
	}
	randomness := hex.EncodeToString(r[:])
	if f.Randomness != nil && *f.Randomness != randomness {
		return Beacon{}, fmt.Errorf("round %d carries a randomness that is not the hash of its signature", round)
	}
	return Beacon{ChainHash: ChainHash, Round: round, Signature: hex.EncodeToString(sig), Randomness: randomness}, nil
}

// parseRound is a reply's round: a positive whole number, written as a JSON
// number of digits alone (no sign, fraction, exponent or quotes).
func parseRound(s string) (uint64, error) {
	var n uint64
	if s == "" || len(s) > 20 {
		return 0, errors.New("the reply's round is not a positive whole number")
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' || n > (1<<64-1-uint64(c-'0'))/10 {
			return 0, errors.New("the reply's round is not a positive whole number")
		}
		n = n*10 + uint64(c-'0')
	}
	if n == 0 {
		return 0, errors.New("the reply's round is not a positive whole number")
	}
	return n, nil
}

// sigBytes is a signature given as 96 lower- or upper-case hex digits.
func sigBytes(s string) ([]byte, error) {
	if len(s) != sigHexLen {
		return nil, fmt.Errorf("the signature is not %d hex digits", sigHexLen)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("the signature is not %d hex digits", sigHexLen)
	}
	return b, nil
}

// verifyRound checks sig as the chain's signature of round under the pinned
// key, and answers the round's randomness, the SHA-256 of the signature. The
// message signed is the SHA-256 of the round as a big-endian uint64; circl's
// G1 domain separation tag is the one the chain's scheme signs under.
func verifyRound(round uint64, sig []byte) ([32]byte, error) {
	key, err := hex.DecodeString(chainPoint)
	if err != nil {
		return [32]byte{}, err
	}
	var pk bls.PublicKey[bls.KeyG2SigG1]
	if err := pk.UnmarshalBinary(key); err != nil {
		return [32]byte{}, fmt.Errorf("the pinned beacon key does not read: %w", err)
	}
	var be [8]byte
	binary.BigEndian.PutUint64(be[:], round)
	msg := sha256.Sum256(be[:])
	if !bls.Verify(&pk, msg[:], sig) {
		return [32]byte{}, errors.New("the beacon's signature does not verify under the chain's pinned key")
	}
	return sha256.Sum256(sig), nil
}
