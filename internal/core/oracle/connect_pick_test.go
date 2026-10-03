package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/credential"
)

// The model-less form of Connect (spc-2610031241482088, step 2): with no
// model named and a picker given, Connect lists the service's models with the
// key it holds, offers the picker only the ids a configuration admits, and
// then runs today's path unchanged: a real completion to the model picked
// verifies the connection, and only then is anything written.

// listingFake is a service that answers its model list (keyless with
// keylessCode, keyed with ids) and its chat completions, and remembers what
// each request carried.
type listingFake struct {
	srv         *httptest.Server
	keylessCode int
	keyedCode   int // 0 lists
	ids         []string
	chatCode    int

	mu        sync.Mutex
	listAuth  []string
	chatModel []string
}

func newListingFake(t *testing.T, keylessCode int, ids []string, chatCode int) *listingFake {
	t.Helper()
	f := &listingFake{keylessCode: keylessCode, ids: ids, chatCode: chatCode}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		auth := r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/models":
			f.listAuth = append(f.listAuth, auth)
			if code := f.keyedCode; auth != "" && code != 0 {
				w.WriteHeader(code)
				return
			}
			if auth == "" && f.keylessCode != http.StatusOK {
				w.WriteHeader(f.keylessCode)
				_, _ = io.WriteString(w, `{"error":{"message":"a key is needed to list"}}`)
				return
			}
			data := make([]map[string]string, len(f.ids))
			for i, id := range f.ids {
				data[i] = map[string]string{"id": id}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions":
			var req struct {
				Model string `json:"model"`
			}
			_ = json.Unmarshal(raw, &req)
			f.chatModel = append(f.chatModel, req.Model)
			w.WriteHeader(f.chatCode)
			if f.chatCode != http.StatusOK {
				_, _ = io.WriteString(w, `{"error":{"message":"the completion failed"}}`)
				return
			}
			_, _ = io.WriteString(w, chat(req.Model, "ok"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *listingFake) base() string { return f.srv.URL + "/v1" }

func (f *listingFake) seen() (listAuth, chatModel []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.listAuth...), append([]string(nil), f.chatModel...)
}

// pickReq is connectReq with no model and the picker given.
func pickReq(f *fx, base string, pick func(context.Context, []string) (string, error)) ConnectRequest {
	req := connectReq(f, base)
	req.Models = nil
	req.Pick = pick
	return req
}

// assertNothingWritten fails when the setup wrote the provider block or a key.
func assertNothingWritten(t *testing.T, f *fx, why string) {
	t.Helper()
	for _, name := range []string{"config.json", credential.StoreFileName, credential.IndexFileName} {
		if _, err := os.Lstat(machineFile(f, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: %s was written", why, name)
		}
	}
}

// TestConnectPicksFromAKeyedListing (G5, the core half): the list request
// carries the key, the picker is offered the listed ids in the service's
// order, the completion asks for the model picked, and both writes land with
// the picked model as the allowlist.
func TestConnectPicksFromAKeyedListing(t *testing.T) {
	svc := newListingFake(t, http.StatusUnauthorized, []string{"vendor/coder-large", "vendor/coder-small"}, http.StatusOK)
	f := newFx(t)
	var offered []string
	res, err := Connect(context.Background(), pickReq(f, svc.base(), func(_ context.Context, listed []string) (string, error) {
		offered = listed
		return "vendor/coder-small", nil
	}))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	listAuth, chatModel := svc.seen()
	if !reflect.DeepEqual(listAuth, []string{"Bearer " + callKey}) {
		t.Fatalf("list requests carried %q, want one, keyed", listAuth)
	}
	if !reflect.DeepEqual(offered, []string{"vendor/coder-large", "vendor/coder-small"}) {
		t.Fatalf("the picker was offered %q", offered)
	}
	if !reflect.DeepEqual(chatModel, []string{"vendor/coder-small"}) {
		t.Fatalf("the completion asked for %q, want the model picked", chatModel)
	}
	if !reflect.DeepEqual(res.Models, []string{"vendor/coder-small"}) ||
		!reflect.DeepEqual(res.Wrote, []string{credential.StorePath, "~/.abcd/config.json"}) {
		t.Fatalf("result = %+v", res)
	}
	got, ok := f.loadAPI().Provider("openrouter")
	if !ok || !reflect.DeepEqual(got.Models, []string{"vendor/coder-small"}) {
		t.Fatalf("provider read back = %+v, %v", got, ok)
	}
	if v, err := credential.Machine(f.roots.Home).Resolve("openrouter"); err != nil || v != callKey {
		t.Fatal("the key does not resolve after a picked setup")
	}
}

// TestConnectNeverOffersAnIdValidModelRefuses: the adapter keeps ids that
// carry characters a shell acts on, and the guided path prints a shell
// command, so Connect offers the picker only the ids validModel admits (and
// the denylist does not refuse): an id with `;`, `$(`, a quote or a space is
// never offered.
func TestConnectNeverOffersAnIdValidModelRefuses(t *testing.T) {
	listed := []string{"vendor/ok-1", "vendor/m;rm -rf ~", "vendor/$(id)", `vendor/q"uote`, "vendor/q'uote",
		"vendor/a b", "denied/model", "~vendor/ok-2:free"}
	svc := newListingFake(t, http.StatusOK, listed, http.StatusOK)
	f := newFx(t)
	f.machineConfig(`{"oracle":{"denylist":["denied/*"]}}`)
	var offered []string
	_, err := Connect(context.Background(), pickReq(f, svc.base(), func(_ context.Context, ids []string) (string, error) {
		offered = ids
		return "vendor/ok-1", nil
	}))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !reflect.DeepEqual(offered, []string{"vendor/ok-1", "~vendor/ok-2:free"}) {
		t.Fatalf("the picker was offered %q", offered)
	}
	for _, id := range offered {
		if strings.ContainsAny(id, ";$'\" ") {
			t.Fatalf("an id a shell acts on was offered: %q", id)
		}
	}
}

// TestConnectWritesNothingWhenTheListingOrThePickFails: a listing that
// fails, a listing with no usable id, a pick the person cancels, and a pick
// that names an id the service did not list each write nothing, make no
// completion, and say which.
func TestConnectWritesNothingWhenTheListingOrThePickFails(t *testing.T) {
	cancelled := errors.New("the person chose to decide later")
	for _, tc := range []struct {
		name     string
		keyless  int
		ids      []string
		pick     func(context.Context, []string) (string, error)
		want     string
		wantPick bool
	}{
		{"the keyed listing is refused", http.StatusUnauthorized, nil, nil, "the model list could not be read", false},
		{"no listed id is usable", http.StatusOK, []string{"vendor/m;x", "vendor/$(y)"}, nil, "listed no usable models", false},
		{"the pick is cancelled", http.StatusOK, []string{"vendor/ok"}, func(context.Context, []string) (string, error) {
			return "", cancelled
		}, "no model was picked", true},
		{"the pick names an id not listed", http.StatusOK, []string{"vendor/ok"}, func(context.Context, []string) (string, error) {
			return "vendor/other", nil
		}, "not one the service listed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newListingFake(t, tc.keyless, tc.ids, http.StatusOK)
			if tc.name == "the keyed listing is refused" {
				svc.mu.Lock()
				svc.keyedCode = http.StatusUnauthorized
				svc.mu.Unlock()
			}
			f := newFx(t)
			picked := false
			pick := func(ctx context.Context, ids []string) (string, error) {
				picked = true
				if tc.pick != nil {
					return tc.pick(ctx, ids)
				}
				return ids[0], nil
			}
			_, err := Connect(context.Background(), pickReq(f, svc.base(), pick))
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "nothing was written") {
				t.Fatalf("Connect = %v; want it to say %q and that nothing was written", err, tc.want)
			}
			if strings.Contains(err.Error(), callKey) {
				t.Fatal("the refusal carries the key")
			}
			if tc.pick != nil && tc.name == "the pick is cancelled" && !errors.Is(err, cancelled) {
				t.Errorf("the picker's own error is not kept: %v", err)
			}
			if picked != tc.wantPick {
				t.Errorf("the picker was called: %v, want %v", picked, tc.wantPick)
			}
			if _, chat := svc.seen(); len(chat) != 0 {
				t.Errorf("a completion was made: %q", chat)
			}
			assertNothingWritten(t, f, tc.name)
		})
	}
}

// TestConnectListsWithTheKeyItsHomeHolds: the external home lists with the
// key its pointer resolves to, and a keyless server lists with none.
func TestConnectListsWithTheKeyItsHomeHolds(t *testing.T) {
	t.Run("external", func(t *testing.T) {
		svc := newListingFake(t, http.StatusUnauthorized, []string{"vendor/ok"}, http.StatusOK)
		f := newFx(t)
		t.Setenv("ABCD_TEST_PROVIDER_KEY", callKey)
		req := pickReq(f, svc.base(), func(_ context.Context, ids []string) (string, error) { return ids[0], nil })
		req.Home, req.Key, req.Pointer = KeyHomeExternal, "", credential.Pointer{Env: "ABCD_TEST_PROVIDER_KEY"}
		if _, err := Connect(context.Background(), req); err != nil {
			t.Fatalf("Connect: %v", err)
		}
		if listAuth, _ := svc.seen(); !reflect.DeepEqual(listAuth, []string{"Bearer " + callKey}) {
			t.Fatalf("list requests carried %q", listAuth)
		}
	})
	t.Run("none", func(t *testing.T) {
		svc := newListingFake(t, http.StatusOK, []string{"vendor/ok"}, http.StatusOK)
		f := newFx(t)
		req := pickReq(f, svc.base(), func(_ context.Context, ids []string) (string, error) { return ids[0], nil })
		req.Home, req.Key = KeyHomeNone, ""
		if _, err := Connect(context.Background(), req); err != nil {
			t.Fatalf("Connect: %v", err)
		}
		if listAuth, _ := svc.seen(); !reflect.DeepEqual(listAuth, []string{""}) {
			t.Fatalf("list requests carried %q, want one, keyless", listAuth)
		}
	})
	t.Run("a pointer at nothing lists nothing", func(t *testing.T) {
		svc := newListingFake(t, http.StatusOK, []string{"vendor/ok"}, http.StatusOK)
		f := newFx(t)
		t.Setenv("ABCD_TEST_PROVIDER_KEY", "")
		req := pickReq(f, svc.base(), func(_ context.Context, ids []string) (string, error) { return ids[0], nil })
		req.Home, req.Key, req.Pointer = KeyHomeExternal, "", credential.Pointer{Env: "ABCD_TEST_PROVIDER_KEY"}
		if _, err := Connect(context.Background(), req); err == nil || !errors.Is(err, credential.ErrNotSet) {
			t.Fatalf("Connect = %v, want the pointer's refusal", err)
		}
		if listAuth, _ := svc.seen(); len(listAuth) != 0 {
			t.Fatalf("a listing was made: %q", listAuth)
		}
	})
}

// TestConnectRefusesBeforeListing: a request Connect would refuse is refused
// before the listing, so no request is sent for a setup that cannot finish,
// and a request with neither a model nor a picker is refused as it always was.
func TestConnectRefusesBeforeListing(t *testing.T) {
	svc := newListingFake(t, http.StatusOK, []string{"vendor/ok"}, http.StatusOK)
	f := newFx(t)
	pick := func(_ context.Context, ids []string) (string, error) { return ids[0], nil }
	bad := pickReq(f, svc.base(), pick)
	bad.Home = "elsewhere"
	if _, err := Connect(context.Background(), bad); err == nil {
		t.Fatal("an unknown home was not refused")
	}
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://api.example.com/v1","models":["m"]}}}}`)
	if _, err := Connect(context.Background(), pickReq(f, svc.base(), pick)); err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("a provider already configured = %v", err)
	}
	if listAuth, _ := svc.seen(); len(listAuth) != 0 {
		t.Fatalf("a refused setup listed the models: %q", listAuth)
	}
	none := pickReq(f, svc.base(), nil)
	none.Provider = "other"
	if _, err := Connect(context.Background(), none); err == nil || !strings.Contains(err.Error(), "no model is listed") {
		t.Fatalf("no model and no picker = %v", err)
	}
}

// TestConnectReportsOnlyTheWritesItMade: the list of writes is the store's
// own (credential.WritesFor) when the key is stored, and names no store file
// when the same key was already held there, since nothing was written to it.
func TestConnectReportsOnlyTheWritesItMade(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", "ok"))
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "shared", callKey); err != nil {
		t.Fatal(err)
	}
	req := connectReq(f, p.base())
	req.KeyName = "shared"
	res, err := Connect(context.Background(), req)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !reflect.DeepEqual(res.Wrote, []string{"~/.abcd/config.json"}) {
		t.Fatalf("wrote = %q, want only the provider block", res.Wrote)
	}
	req.Provider, req.KeyName = "second", "second"
	if res, err = Connect(context.Background(), req); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if want := append(credential.WritesFor(KeyHomeABCD, "second"), "~/.abcd/config.json"); !reflect.DeepEqual(res.Wrote, want) {
		t.Fatalf("wrote = %q, want %q", res.Wrote, want)
	}
}

// TestConnectRefusesOfferedIdsThatTogetherCarryTheKey (security review
// finding 1): the adapter reads the ids it keeps as one run for a key split
// across them, and the ids offered are the ones a configuration admits, so the
// run read is the offered one. A service that separates the key's halves with
// an id validModel refuses gets no list: the halves would otherwise be offered
// as adjacent lines, and either one picked would be printed and written.
func TestConnectRefusesOfferedIdsThatTogetherCarryTheKey(t *testing.T) {
	half := len(callKey) / 2
	for _, between := range []string{"vendor/m;x", "denied/model"} {
		t.Run(between, func(t *testing.T) {
			svc := newListingFake(t, http.StatusOK, []string{callKey[:half], between, callKey[half:]}, http.StatusOK)
			f := newFx(t)
			const denylist = `{"oracle":{"denylist":["denied/*"]}}`
			f.machineConfig(denylist)
			var offered []string
			_, err := Connect(context.Background(), pickReq(f, svc.base(), func(_ context.Context, ids []string) (string, error) {
				offered = ids
				return ids[0], nil
			}))
			if err == nil || !strings.Contains(err.Error(), "together carry the key") || !strings.Contains(err.Error(), "nothing was written") {
				t.Fatalf("Connect = %v; want the listing refused as carrying the key together, and nothing written", err)
			}
			if strings.Contains(err.Error(), callKey) || strings.Contains(err.Error(), callKey[:half]) || strings.Contains(err.Error(), callKey[half:]) {
				t.Fatal("the refusal carries the key or a half of it")
			}
			if len(offered) != 0 {
				t.Fatalf("the picker was offered %q", offered)
			}
			if _, chat := svc.seen(); len(chat) != 0 {
				t.Errorf("a completion was made: %q", chat)
			}
			if raw, err := os.ReadFile(machineFile(f, "config.json")); err != nil || string(raw) != denylist {
				t.Errorf("the configuration was changed: %q, %v", raw, err)
			}
			for _, name := range []string{credential.StoreFileName, credential.IndexFileName} {
				if _, err := os.Lstat(machineFile(f, name)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("%s was written", name)
				}
			}
		})
	}
}

// TestCheckConnectNeedsNoKey: the check a front door runs before asking for
// the key admits a well-formed request that carries no key yet, and refuses
// what Connect would refuse without one, a provider already configured
// included.
func TestCheckConnectNeedsNoKey(t *testing.T) {
	f := newFx(t)
	req := connectReq(f, "https://api.example.com/v1")
	req.Key = ""
	if err := CheckConnect(req); err != nil {
		t.Fatalf("CheckConnect of a keyless abcd-home request = %v, want nil", err)
	}
	req.Models = nil
	req.Pick = func(_ context.Context, ids []string) (string, error) { return ids[0], nil }
	if err := CheckConnect(req); err != nil {
		t.Fatalf("CheckConnect of a picking request = %v, want nil", err)
	}
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://api.example.com/v1","models":["m"]}}}}`)
	if err := CheckConnect(req); err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("CheckConnect of a provider already configured = %v", err)
	}
	req.Provider = Harness
	if err := CheckConnect(req); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("CheckConnect of the reserved name = %v", err)
	}
}
