package spotcheck

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

// The published vector of drand's quicknet chain for round 1000
// (spc-2610301925429250 approach 4), read on 31 October 2026.
const (
	vectorRound      = 1000
	vectorSignature  = "b44679b9a59af2ec876b1a6b1ad52ea9b1615fc3982b19576350f93447cb1125e342b73a8dd2bacbe47e4b6b63ed5e39"
	vectorRandomness = "fe290beca10872ef2fb164d2aa4442de4566183ec51c56ff3cd603d930e54fdd"
)

// TestBeaconVerify: the round-1000 vector verifies under the pinned key and
// yields its published randomness; the same signature against round 1001
// does not verify.
func TestBeaconVerify(t *testing.T) {
	sig, err := hex.DecodeString(vectorSignature)
	if err != nil {
		t.Fatal(err)
	}
	r, err := verifyRound(vectorRound, sig)
	if err != nil {
		t.Fatalf("the published round-1000 vector does not verify under the pinned key: %v", err)
	}
	if got := hex.EncodeToString(r[:]); got != vectorRandomness {
		t.Fatalf("round 1000 yields randomness %s; want the published %s", got, vectorRandomness)
	}
	if _, err := verifyRound(vectorRound+1, sig); err == nil {
		t.Fatal("round 1000's signature verifies for round 1001")
	}
}

// beaconReply is a reply body as the chain serves a round.
func beaconReply(round, sig, randomness string) []byte {
	b := `{"round":` + round + `,"signature":"` + sig + `"`
	if randomness != "" {
		b += `,"randomness":"` + randomness + `"`
	}
	return []byte(b + `}`)
}

// serve sets the beacon seam for the test: each host's answer by host.
func serve(t *testing.T, answers map[string]func() ([]byte, error)) *[]string {
	t.Helper()
	var asked []string
	old := beaconGet
	beaconGet = func(ctx context.Context, url string) ([]byte, error) {
		asked = append(asked, url)
		for host, a := range answers {
			if strings.HasPrefix(url, "https://"+host+"/") {
				return a()
			}
		}
		return nil, errors.New("unreachable")
	}
	t.Cleanup(func() { beaconGet = old })
	return &asked
}

func TestBeaconFetchVerifiesTheRound(t *testing.T) {
	serve(t, map[string]func() ([]byte, error){
		"api.drand.sh": func() ([]byte, error) { return beaconReply("1000", vectorSignature, vectorRandomness), nil },
	})
	now := time.Date(2026, 11, 25, 1, 2, 3, 0, time.UTC)
	b, err := fetchBeacon(context.Background(), 1000, now)
	if err != nil {
		t.Fatal(err)
	}
	if b.Round != vectorRound || b.Signature != vectorSignature || b.Randomness != vectorRandomness || b.ChainHash != ChainHash || !b.FetchedAt.Equal(now) {
		t.Fatalf("beacon = %+v", b)
	}
	if b.URL != "https://api.drand.sh/"+ChainHash+"/public/1000" {
		t.Fatalf("the beacon names %s", b.URL)
	}
}

// TestBeaconReplyBounds: a reply over 4 KiB, a signature that is not 96 hex
// digits, a round that is not positive and a reply that is not JSON are each
// refused, and the draw is bounded by ten seconds in all.
func TestBeaconReplyBounds(t *testing.T) {
	for name, body := range map[string][]byte{
		"over the bound":   append(beaconReply("1000", vectorSignature, ""), []byte(strings.Repeat(" ", 4<<10))...),
		"short signature":  beaconReply("1000", vectorSignature[:94], ""),
		"not hex":          beaconReply("1000", strings.Repeat("z", 96), ""),
		"round zero":       beaconReply("0", vectorSignature, ""),
		"negative round":   beaconReply("-1", vectorSignature, ""),
		"round as text":    beaconReply(`"1000"`, vectorSignature, ""),
		"not json":         []byte("round 1000"),
		"two values":       append(beaconReply("1000", vectorSignature, ""), beaconReply("1000", vectorSignature, "")...),
		"no signature":     []byte(`{"round":1000}`),
		"unverified round": beaconReply("1001", vectorSignature, ""),
	} {
		t.Run(name, func(t *testing.T) {
			serve(t, map[string]func() ([]byte, error){"api.drand.sh": func() ([]byte, error) { return body, nil }})
			if b, err := fetchBeacon(context.Background(), 1000, time.Now()); err == nil {
				t.Fatalf("the reply was taken: %+v", b)
			} else if !strings.Contains(err.Error(), "beacon") {
				t.Fatalf("the refusal does not name the beacon: %v", err)
			}
		})
	}
	if BeaconTimeout != 10*time.Second || maxBeaconBytes != 4<<10 {
		t.Fatalf("the beacon's bounds are %v and %d bytes; want ten seconds and 4 KiB", BeaconTimeout, maxBeaconBytes)
	}
}

// TestBeaconOtherChainRefused: a reply that names a chain hash other than the
// pinned one is refused, even with a signature that verifies.
func TestBeaconOtherChainRefused(t *testing.T) {
	other := strings.Repeat("ab", 32)
	serve(t, map[string]func() ([]byte, error){"api.drand.sh": func() ([]byte, error) {
		return []byte(`{"round":1000,"signature":"` + vectorSignature + `","chain_hash":"` + other + `"}`), nil
	}})
	if _, err := fetchBeacon(context.Background(), 1000, time.Now()); err == nil || !strings.Contains(err.Error(), "chain") {
		t.Fatalf("a reply of another chain was taken (err %v)", err)
	}
}

// TestBeaconRandomnessMustMatch: a reply that carries a randomness field must
// carry the hash of its own signature.
func TestBeaconRandomnessMustMatch(t *testing.T) {
	serve(t, map[string]func() ([]byte, error){"api.drand.sh": func() ([]byte, error) {
		return beaconReply("1000", vectorSignature, strings.Repeat("0", 64)), nil
	}})
	if _, err := fetchBeacon(context.Background(), 1000, time.Now()); err == nil || !strings.Contains(err.Error(), "randomness") {
		t.Fatalf("a reply whose randomness is not its signature's hash was taken (err %v)", err)
	}
}

// TestBeaconHostFallbackOrder: an unreachable host is followed by the next,
// api.drand.sh, then api2, then api3; a reply that is read and refused is not
// asked of another host.
func TestBeaconHostFallbackOrder(t *testing.T) {
	asked := serve(t, map[string]func() ([]byte, error){
		"api3.drand.sh": func() ([]byte, error) { return beaconReply("1000", vectorSignature, ""), nil },
	})
	b, err := fetchBeacon(context.Background(), 1000, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"api.drand.sh", "api2.drand.sh", "api3.drand.sh"}
	if len(*asked) != 3 {
		t.Fatalf("asked %q; want the three hosts in order", *asked)
	}
	for i, h := range want {
		if !strings.HasPrefix((*asked)[i], "https://"+h+"/"+ChainHash+"/") {
			t.Fatalf("asked %q; want %q in order", *asked, want)
		}
	}
	if !strings.HasPrefix(b.URL, "https://api3.drand.sh/") {
		t.Fatalf("the beacon names %s; want the host that answered", b.URL)
	}

	asked = serve(t, map[string]func() ([]byte, error){
		"api.drand.sh":  func() ([]byte, error) { return beaconReply("1001", vectorSignature, ""), nil },
		"api2.drand.sh": func() ([]byte, error) { return beaconReply("1000", vectorSignature, ""), nil },
	})
	if _, err := fetchBeacon(context.Background(), 1000, time.Now()); err == nil || len(*asked) != 1 {
		t.Fatalf("a refused reply was followed by another host (asked %q, err %v)", *asked, err)
	}

	asked = serve(t, nil)
	if _, err := fetchBeacon(context.Background(), 1000, time.Now()); err == nil || len(*asked) != 3 || !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("no host answering was not refused naming why (asked %q, err %v)", *asked, err)
	}
}

// TestRoundAtTheChainsClock: a window's round is the first the chain
// publishes at or after its close, on the pinned genesis and period.
func TestRoundAtTheChainsClock(t *testing.T) {
	at := roundTime(1000)
	for when, want := range map[time.Time]uint64{at: 1000, at.Add(-time.Second): 1000, at.Add(-3 * time.Second): 999, at.Add(time.Second): 1001,
		at.Add(7 * 24 * time.Hour): 202600, time.Unix(chainGenesis, 0): 1, time.Unix(0, 0): 1} {
		if got := roundAt(when); got != want {
			t.Errorf("roundAt(%s) = %d; want %d", when, got, want)
		}
	}
	if at.Unix() != 1692806364 {
		t.Fatalf("round 1000 is published at %d", at.Unix())
	}
}

// TestFetchRefusesAnotherRound: a host that answers a verified round other
// than the one asked is refused, naming both.
func TestFetchRefusesAnotherRound(t *testing.T) {
	serve(t, map[string]func() ([]byte, error){
		"api.drand.sh": func() ([]byte, error) { return beaconReply("1000", vectorSignature, vectorRandomness), nil },
	})
	if _, err := fetchBeacon(context.Background(), 202600, time.Now()); err == nil || !strings.Contains(err.Error(), "round 1000 when round 202600") {
		t.Fatalf("another round was taken (err %v)", err)
	}
}
