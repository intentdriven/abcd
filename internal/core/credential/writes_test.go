package credential

import (
	"context"
	"reflect"
	"testing"
)

// TestWritesForMatchesTheWalk (spc-2610031241482088, step 2): what a home's
// write touches is named once, by WritesFor, and a walkthrough that stores a
// credential reports exactly that list, in that order, for every home. The
// guided path prints the list before the command runs, so the two cannot be
// allowed to drift apart.
func TestWritesForMatchesTheWalk(t *testing.T) {
	withFakeKeychain(t, keychainMacOS)
	want := map[string][]string{
		HomeABCD:     {StorePath},
		HomeKeychain: {KeychainItem("svc"), IndexPath},
		HomeExternal: {IndexPath},
	}
	for _, h := range Homes() {
		home := t.TempDir()
		ch := Choice{Home: h, Value: secretValue}
		if h == HomeExternal {
			t.Setenv("ABCD_TEST_TOKEN_FOR_WRITES", secretValue)
			ch = Choice{Home: h, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_FOR_WRITES"}}
		}
		if got := WritesFor(h, "svc"); !reflect.DeepEqual(got, want[h]) {
			t.Errorf("%s: WritesFor = %q, want %q", h, got, want[h])
		}
		res, err := Walk(context.Background(), home, testService(func(context.Context, string) error { return nil }), ch)
		if err != nil || !res.Changed {
			t.Fatalf("%s: Walk = %+v, %v", h, res, err)
		}
		if !reflect.DeepEqual(res.Wrote, WritesFor(h, "svc")) {
			t.Errorf("%s: the walk wrote %q, and WritesFor names %q", h, res.Wrote, WritesFor(h, "svc"))
		}
	}
	if got := WritesFor("none", "svc"); len(got) != 0 {
		t.Errorf("a home that stores nothing writes %q", got)
	}
}

// TestResolvePointerFollowsTheExternalHome: the pointer's resolver is the
// one the walkthrough uses, exported so a setup can list a service's models
// with the external key before the walkthrough verifies it.
func TestResolvePointerFollowsTheExternalHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ABCD_TEST_TOKEN_FOR_POINTER", secretValue)
	if v, err := ResolvePointer(home, "svc", Pointer{Env: "ABCD_TEST_TOKEN_FOR_POINTER"}); err != nil || v != secretValue {
		t.Fatalf("ResolvePointer = %v (the value: %v)", err, v == secretValue)
	}
	t.Setenv("ABCD_TEST_TOKEN_FOR_POINTER", "")
	if _, err := ResolvePointer(home, "svc", Pointer{Env: "ABCD_TEST_TOKEN_FOR_POINTER"}); err == nil {
		t.Fatal("a pointer at an unset variable resolved")
	}
}
