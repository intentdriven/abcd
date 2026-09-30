package oracle

// connect.go is the setup's write (itd-2609081951381895 scope 5, criteria 7
// and 8): given a provider's base URL, its first allowlist and where its key
// lives, verify the connection with one call, then write the key into its home
// and the provider block into the machine's ~/.abcd/config.json. Nothing is
// written into the repository or into the harness's settings, and a
// verification that fails writes nothing at all.
//
// The key is kept through the credential store's walkthrough
// (itd-2609221017023290): in one of its three homes, the external setup, the
// abcd-only file or the platform keychain, verified first with this adapter's
// own call. A fourth answer, none, is a local server that takes no key.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The homes a provider's key may live in (the intent's Decision 4): the
// credential store's three, and none.
const (
	// KeyHomeExternal is a setup outside abcd (an environment variable or an
	// existing tool's configuration); abcd stores only where it is.
	KeyHomeExternal = credential.HomeExternal
	// KeyHomeABCD is abcd-only: the owner-only ~/.abcd/credentials.json.
	KeyHomeABCD = credential.HomeABCD
	// KeyHomeKeychain is the platform keychain.
	KeyHomeKeychain = credential.HomeKeychain
	// KeyHomeNone is a server that takes no key (a local one).
	KeyHomeNone = "none"
)

// KeyHomes returns the homes in the order the setup offers them.
func KeyHomes() []string { return append(credential.Homes(), KeyHomeNone) }

// ConnectRequest is one provider's setup.
type ConnectRequest struct {
	// Roots are where the configuration in force is read: the oracle.denylist
	// a model is held to, and the provider blocks a name must not repeat. The
	// writes land under Roots.Home alone.
	Roots    layered.Roots
	Provider string
	BaseURL  string
	// Models is the first allowlist; the verification call asks for the first.
	Models []string
	// Home is where the key lives: one of KeyHomes.
	Home string
	// KeyName is the credential's name; "" names it after the provider.
	KeyName string
	// Key is the value, for the abcd and keychain homes. It is never echoed.
	Key string
	// Pointer is where the key is, for the external home.
	Pointer credential.Pointer
	// Timeout bounds the verification call; 0 keeps the adapter's default.
	Timeout time.Duration
}

// ConnectResult is what the setup did. It never carries the key.
type ConnectResult struct {
	Provider string   `json:"provider"`
	BaseURL  string   `json:"base_url"`
	Models   []string `json:"models"`
	KeyHome  string   `json:"key_home"`
	KeyName  string   `json:"key_name,omitempty"`
	// Verified is the verification call's record.
	Verified CallRecord `json:"verified"`
	// Wrote names each file written, in the tilde form.
	Wrote []string `json:"wrote"`
}

// verifyBrief is the verification call's brief: one short exchange, judged
// only on the provider answering in the protocol's shape.
var verifyBrief = openaiapi.Brief{
	Instructions: "abcd is verifying a provider connection. Reply with the single word: ok",
	Input:        "ok",
}

// Connect verifies the connection with one call and, only when it succeeds,
// writes the key and the provider block. Every fault the configuration read
// would refuse is refused first, before the call.
func Connect(ctx context.Context, req ConnectRequest) (ConnectResult, error) {
	if err := checkConnect(&req); err != nil {
		return ConnectResult{}, err
	}
	cfg, err := LoadAPI(req.Roots)
	if err != nil {
		return ConnectResult{}, fmt.Errorf("%w; fix the configuration before adding a provider to it", err)
	}
	if _, exists := cfg.providers[req.Provider]; exists {
		return ConnectResult{}, fmt.Errorf("oracle adapter: provider %s is already configured in %s; "+
			"abcd never replaces a block unasked, so edit or remove it there to change it", req.Provider, layered.Config.MachineOrigin())
	}
	for _, m := range req.Models {
		if e, denied := Denied(cfg.denylist, m); denied {
			return ConnectResult{}, fmt.Errorf("oracle adapter: provider %s %s", req.Provider, deniedError(m, e))
		}
	}
	var opts []openaiapi.Option
	if req.Timeout > 0 {
		opts = append(opts, openaiapi.WithTimeout(req.Timeout))
	}
	res := ConnectResult{Provider: req.Provider, BaseURL: req.BaseURL, Models: append([]string(nil), req.Models...),
		KeyHome: req.Home}
	svc := providerService(Provider{Name: req.Provider, BaseURL: req.BaseURL, Key: req.KeyName, Models: req.Models},
		cfg.denylist, &res.Verified, opts...)
	block := map[string]any{"base_url": req.BaseURL, "models": req.Models}
	if req.Home == KeyHomeNone {
		if err := svc.Verify(ctx, ""); err != nil {
			return ConnectResult{}, fmt.Errorf("%w; the verification call failed, so nothing was written", err)
		}
	} else {
		// The store's walkthrough refuses a key it cannot keep before the
		// call, so a setup that cannot store its key is never billed for.
		walked, err := credential.Walk(ctx, req.Roots.Home, svc, credential.Choice{Home: req.Home, Value: req.Key, Pointer: req.Pointer})
		if err != nil {
			return ConnectResult{}, fmt.Errorf("oracle adapter: %w", err)
		}
		res.KeyName = req.KeyName
		block["key"] = req.KeyName
		res.Wrote = append(res.Wrote, walked.Wrote...)
	}
	if err := writeProviderBlock(req.Roots.Home, req.Provider, block); err != nil {
		if len(res.Wrote) > 0 {
			return ConnectResult{}, fmt.Errorf("%w; the key was stored in the %s home under %s, and the provider block was not written",
				err, req.Home, req.KeyName)
		}
		return ConnectResult{}, err
	}
	res.Wrote = append(res.Wrote, layered.Config.MachineOrigin())
	return res, nil
}

// checkConnect refuses a malformed request, never echoing the key.
func checkConnect(req *ConnectRequest) error {
	switch {
	case !providerNameRe.MatchString(req.Provider):
		return fmt.Errorf("oracle adapter: provider name %q is not lower case letters, digits, - and _", layered.BoundKey(req.Provider))
	case req.Provider == Harness:
		return fmt.Errorf("oracle adapter: %q names the host's own leg and is reserved", Harness)
	}
	if err := openaiapi.ValidateBaseURL(req.BaseURL); err != nil {
		return fmt.Errorf("oracle adapter: %w", err)
	}
	if len(req.Models) == 0 {
		return errors.New("oracle adapter: no model is listed; a provider serves only the models it lists, so the setup lists at least one")
	}
	if len(req.Models) > MaxModels {
		return fmt.Errorf("oracle adapter: %d models are listed; a provider lists at most %d", len(req.Models), MaxModels)
	}
	seen := map[string]bool{}
	for _, m := range req.Models {
		if !validModel(m) {
			return fmt.Errorf("oracle adapter: model %q is not a model identifier", layered.BoundKey(m))
		}
		if seen[m] {
			return fmt.Errorf("oracle adapter: model %s is listed twice", m)
		}
		seen[m] = true
	}
	switch req.Home {
	case KeyHomeNone:
		if req.Key != "" || req.Pointer != (credential.Pointer{}) {
			return errors.New("oracle adapter: a key was given for a provider set up with no key; choose a home to keep it")
		}
		req.KeyName = ""
		return nil
	case KeyHomeExternal:
		if req.Key != "" {
			return errors.New("oracle adapter: the external home keeps where the key is, never the key, and a key was given")
		}
		if req.Pointer == (credential.Pointer{}) {
			return errors.New("oracle adapter: the external home needs where the key is: an environment variable, or a file and its field")
		}
	case KeyHomeABCD, KeyHomeKeychain:
		if req.Pointer != (credential.Pointer{}) {
			return fmt.Errorf("oracle adapter: the %s home keeps the key itself, and a pointer was given", req.Home)
		}
	default:
		return fmt.Errorf("oracle adapter: key home %q is not one of external, abcd, keychain, none", layered.BoundKey(req.Home))
	}
	if req.KeyName == "" {
		req.KeyName = req.Provider
	}
	if !credential.ValidName(req.KeyName) {
		return fmt.Errorf("oracle adapter: key name %q is not a plain credential name", layered.BoundKey(req.KeyName))
	}
	if req.Home == KeyHomeExternal {
		return nil
	}
	if req.Key == "" {
		return fmt.Errorf("oracle adapter: the %s home stores a key, and none was given", req.Home)
	}
	// The store's own value check, before the call rather than after it.
	return credential.CheckValue(req.Key)
}

// configLockFileName is the lock the provider block's write takes, beside
// ~/.abcd/config.json.
const configLockFileName = ".config.json.lock"

// configLockTimeout bounds the wait for another setup writing the file.
var configLockTimeout = 5 * time.Second

// writeProviderBlock sets oracle.api.<name> in ~/.abcd/config.json, keeping
// every other key, written atomically at mode 0600. The file is read, changed
// and renamed into place under its lock (fsutil.WithFileLockIn), so concurrent
// setups never lose each other's blocks, and a block another setup wrote
// after this one's check is refused rather than replaced.
func writeProviderBlock(home, name string, block map[string]any) error {
	origin := layered.Config.MachineOrigin()
	rel := ".abcd/" + layered.Config.MachineRel
	// The machine layer refuses a file behind a symlinked ~/.abcd, so a block
	// written through the link would land wherever it points (a dotfiles
	// checkout) and never be read back.
	// ~/.abcd is created, judged and opened in one walk relative to home's
	// descriptor, and the lock and the file are reached through it, so a link
	// swapped in after the judgement is refused rather than written through
	// (iss-2609281310017733).
	dir, err := fsutil.EnsureHomeScope(home, path.Dir(rel), 0o700)
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		return fmt.Errorf("oracle adapter: the provider block was not written to %s: %v", origin, err)
	}
	if err != nil {
		return fmt.Errorf("oracle adapter: ~/.abcd could not be created, so the provider block was not written")
	}
	defer dir.Close()
	err = fsutil.WithFileLockIn(dir, configLockFileName, configLockTimeout, func() error {
		return writeProviderBlockLocked(home, dir, name, block)
	})
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("oracle adapter: %s is being written by another abcd, so the provider block was not written; retry", origin)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		// A retry cannot cure a symlinked or non-regular lock, so the
		// refusal names it rather than reading as contention.
		return fmt.Errorf("oracle adapter: the lock ~/.abcd/%s is not a regular file (a symlink, or something else), so it is refused and the provider block was not written; remove it, and the next setup creates it afresh", configLockFileName)
	}
	return err
}

// writeProviderBlockLocked is writeProviderBlock's read, change and write,
// run under the file's lock.
func writeProviderBlockLocked(home string, dir *os.Root, name string, block map[string]any) error {
	origin := layered.Config.MachineOrigin()
	rel := ".abcd/" + layered.Config.MachineRel
	root := map[string]json.RawMessage{}
	// Read through dir, the directory the write below goes through, never by
	// walking ~/.abcd again: a same-uid swap of ~/.abcd between the two walks
	// would otherwise read one directory's file and write it, with the new
	// block, into the other (iss-2609290300313698).
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, rel, layered.MaxFileBytes, 0)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
	case refusal != fsutil.DeclarationOK || err != nil:
		return fmt.Errorf("oracle adapter: %s could not be read safely, so the provider block was not written", origin)
	default:
		// The bytes re-read under the lock are the ones the rewrite trusts, so
		// they meet LoadAPI's duplicate-key check again: a key named twice would
		// otherwise collapse last-wins here and be rewritten without its other
		// spelling (iss-2609261312108500).
		if err := jsonstrict.NoDuplicateKeys(raw); err != nil {
			return fmt.Errorf("oracle adapter: %s is refused: %v; the provider block was not written", origin, err)
		}
		if err := json.Unmarshal(raw, &root); err != nil || root == nil {
			return fmt.Errorf("oracle adapter: %s is not a JSON object, so the provider block was not written", origin)
		}
	}
	oracleObj := map[string]json.RawMessage{}
	if v, ok := root["oracle"]; ok {
		if err := json.Unmarshal(v, &oracleObj); err != nil || oracleObj == nil {
			return fmt.Errorf("oracle adapter: %s: oracle is not an object, so the provider block was not written", origin)
		}
	}
	api := map[string]json.RawMessage{}
	if v, ok := oracleObj["api"]; ok {
		if err := json.Unmarshal(v, &api); err != nil || api == nil {
			return fmt.Errorf("oracle adapter: %s: oracle.api is not an object, so the provider block was not written", origin)
		}
	}
	if _, exists := api[name]; exists {
		return fmt.Errorf("oracle adapter: provider %s is already configured in %s; "+
			"abcd never replaces a block unasked, so edit or remove it there to change it", name, origin)
	}
	enc, err := json.Marshal(block)
	if err != nil {
		return err
	}
	api[name] = enc
	if oracleObj["api"], err = json.Marshal(api); err != nil {
		return err
	}
	if root["oracle"], err = json.Marshal(oracleObj); err != nil {
		return err
	}
	body, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomicInRoot(dir, path.Base(rel), append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("oracle adapter: %s could not be written, so the provider block was not written", origin)
	}
	return nil
}

// AdapterExplanation is what the adapter is, what abcd would use it for and
// what works without it (criterion 6), in the words every surface uses: the
// ahoy gap, `abcd ahoy --providers` and the plugin page.
const AdapterExplanation = "An aggregator (OpenRouter, for one) serves many vendors' models behind one " +
	"OpenAI-compatible address and one key, and a local OpenAI-compatible server is reached the same way. " +
	"abcd would use one for decision models and cheap judgements pointed at it by name, and only for the models its " +
	"list names: a model the person does not list, a frontier model included, stays on the host. " +
	"Everything works without one: with no provider configured, " +
	"every delegated step runs on the host."

// KeyHomesProse is the prose above the choice of the key's home (criterion 8):
// the credential store's, which recommends the keychain in the prose and never
// as a marked option.
const KeyHomesProse = credential.HomesProse

// providerService is the credential walkthrough's service for a provider's
// key: what it unlocks, what works without it, and the adapter's own
// verification call, one short exchange with the first model listed. The
// call's record is written to rec when rec is non-nil.
func providerService(p Provider, denylist []DenyEntry, rec *CallRecord, opts ...openaiapi.Option) credential.Service {
	return credential.Service{
		Name:      p.Key,
		Unlocks:   "calls to the model provider " + p.Name + " at " + p.BaseURL + ", for the models its allowlist names",
		WithoutIt: "everything: every delegated step runs on the host",
		Verify: func(ctx context.Context, key string) error {
			_, r, err := complete(ctx, p.Name, p.BaseURL, key, p.Models[0], verifyBrief,
				Settings{"max_tokens": json.RawMessage(`16`)}, nil, denylist, opts...)
			r.Credential = p.Key
			if rec != nil {
				*rec = r
			}
			return err
		},
	}
}

// CredentialService is the walkthrough's service for the credential name, when
// a configured provider names it as its key: the walkthrough then verifies a
// key with that provider's own call. A name no provider names is not the
// adapter's.
func CredentialService(roots layered.Roots, name string) (credential.Service, bool, error) {
	cfg, err := LoadAPI(roots)
	if err != nil {
		return credential.Service{}, false, err
	}
	for _, p := range cfg.Providers() {
		if p.Key == name && len(p.Models) > 0 {
			return providerService(p, cfg.denylist, nil), true, nil
		}
	}
	return credential.Service{}, false, nil
}
