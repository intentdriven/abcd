package oracle

// connect.go is the setup's write (itd-2609081951381895 scope 5, criteria 7
// and 8): given a provider's base URL, its first allowlist and where its key
// lives, verify the connection with one call, then write the key into its home
// and the provider block into the machine's ~/.abcd/config.json. Nothing is
// written into the repository or into the harness's settings, and a
// verification that fails writes nothing at all.
//
// Of the three homes a key may live in, this lane builds the one the interim
// credential source already reads: abcd-only, ~/.abcd/credentials.json at mode
// 0600. The environment-variable-or-external-tool home and the platform
// keychain are the credential store's (itd-2609221017023290, planned), which
// replaces the source's backing and not its interface; asked for either, the
// setup refuses naming it, before any call and any write. A fourth answer,
// none, is a local server that takes no key.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The homes a provider's key may live in (the intent's Decision 4).
const (
	// KeyHomeExternal is a setup outside abcd (an environment variable or an
	// existing tool's configuration); abcd would store only its name.
	KeyHomeExternal = "external"
	// KeyHomeABCD is abcd-only: the owner-only ~/.abcd/credentials.json.
	KeyHomeABCD = "abcd"
	// KeyHomeKeychain is the platform keychain.
	KeyHomeKeychain = "keychain"
	// KeyHomeNone is a server that takes no key (a local one).
	KeyHomeNone = "none"
)

// KeyHomes returns the homes in the order the setup offers them.
func KeyHomes() []string { return []string{KeyHomeExternal, KeyHomeABCD, KeyHomeKeychain, KeyHomeNone} }

// CredentialStoreIntent is the intent that builds the external and keychain
// homes, named by every deferral.
const CredentialStoreIntent = "itd-2609221017023290"

// ConnectRequest is one provider's setup.
type ConnectRequest struct {
	// Roots are where the configuration in force is read: the denylist a
	// model is held to, and the provider blocks a name must not repeat. The
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
	// Key is the value, for the abcd home. It is never echoed.
	Key string
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
	if req.Home == KeyHomeABCD {
		// Refused before the call, so a setup that cannot store its key is
		// never billed for.
		stored, err := credential.Machine(req.Roots.Home).Resolve(req.KeyName)
		switch {
		case errors.Is(err, credential.ErrNotSet):
		case err != nil:
			return ConnectResult{}, err
		case stored != req.Key:
			return ConnectResult{}, fmt.Errorf("oracle adapter: %s already holds a different value for %s, and abcd never replaces a stored secret; "+
				"name another credential with --key, or remove that entry by hand", credential.StorePath, req.KeyName)
		}
	}

	var opts []openaiapi.Option
	if req.Timeout > 0 {
		opts = append(opts, openaiapi.WithTimeout(req.Timeout))
	}
	_, rec, err := complete(ctx, req.Provider, req.BaseURL, req.Key, req.Models[0], verifyBrief,
		Settings{"max_tokens": json.RawMessage(`16`)}, nil, cfg.denylist, opts...)
	if err != nil {
		return ConnectResult{}, fmt.Errorf("%w; the verification call failed, so nothing was written", err)
	}

	res := ConnectResult{Provider: req.Provider, BaseURL: req.BaseURL, Models: append([]string(nil), req.Models...),
		KeyHome: req.Home, Verified: rec}
	block := map[string]any{"base_url": req.BaseURL, "models": req.Models}
	if req.Home == KeyHomeABCD {
		res.KeyName = req.KeyName
		block["key"] = req.KeyName
		changed, err := credential.SetMachine(req.Roots.Home, req.KeyName, req.Key)
		if err != nil {
			return ConnectResult{}, fmt.Errorf("%w; the connection verified, and nothing was written", err)
		}
		if changed {
			res.Wrote = append(res.Wrote, credential.StorePath)
		}
	}
	if err := writeProviderBlock(req.Roots.Home, req.Provider, block); err != nil {
		if len(res.Wrote) > 0 {
			return ConnectResult{}, fmt.Errorf("%w; the key was stored in %s under %s, and the provider block was not written",
				err, credential.StorePath, req.KeyName)
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
	case KeyHomeExternal, KeyHomeKeychain:
		return fmt.Errorf("oracle adapter: the %s home is built by the credential store (%s), which is planned and not built; "+
			"until it lands a key lives in the abcd-only home (%s, owner-only), and nothing was written", req.Home, CredentialStoreIntent, credential.StorePath)
	case KeyHomeNone:
		if req.Key != "" {
			return errors.New("oracle adapter: a key was given for a provider set up with no key; choose the abcd home to store it")
		}
		req.KeyName = ""
		return nil
	case KeyHomeABCD:
	default:
		return fmt.Errorf("oracle adapter: key home %q is not one of external, abcd, keychain, none", layered.BoundKey(req.Home))
	}
	if req.KeyName == "" {
		req.KeyName = req.Provider
	}
	if !credential.ValidName(req.KeyName) {
		return fmt.Errorf("oracle adapter: key name %q is not a plain credential name", layered.BoundKey(req.KeyName))
	}
	if req.Key == "" {
		return errors.New("oracle adapter: the abcd home stores a key, and none was given")
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
// and renamed into place under its lock (fsutil.WithFileLock), so concurrent
// setups never lose each other's blocks, and a block another setup wrote
// after this one's check is refused rather than replaced.
func writeProviderBlock(home, name string, block map[string]any) error {
	origin := layered.Config.MachineOrigin()
	p := filepath.Join(home, ".abcd", filepath.FromSlash(layered.Config.MachineRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("oracle adapter: ~/.abcd could not be created, so the provider block was not written")
	}
	err := fsutil.WithFileLock(filepath.Join(filepath.Dir(p), configLockFileName), configLockTimeout, func() error {
		return writeProviderBlockLocked(p, name, block)
	})
	if errors.Is(err, fsutil.ErrLockContention) || errors.Is(err, fsutil.ErrLockPathUnsafe) {
		return fmt.Errorf("oracle adapter: %s is being written by another abcd, or its lock could not be taken, so the provider block was not written; retry", origin)
	}
	return err
}

// writeProviderBlockLocked is writeProviderBlock's read, change and write,
// run under the file's lock.
func writeProviderBlockLocked(p, name string, block map[string]any) error {
	origin := layered.Config.MachineOrigin()
	root := map[string]json.RawMessage{}
	raw, refusal, err := fsutil.ReadDeclaration(p, layered.MaxFileBytes)
	switch {
	case refusal == fsutil.DeclarationAbsent && errors.Is(err, os.ErrNotExist):
	case refusal != fsutil.DeclarationOK || err != nil:
		return fmt.Errorf("oracle adapter: %s could not be read safely, so the provider block was not written", origin)
	default:
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
	if err := fsutil.WriteFileAtomic(p, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("oracle adapter: %s could not be written, so the provider block was not written", origin)
	}
	return nil
}

// AdapterExplanation is what the adapter is, what abcd would use it for and
// what works without it (criterion 6), in the words every surface uses: the
// ahoy gap, `abcd ahoy --providers` and the plugin page.
const AdapterExplanation = "An aggregator (OpenRouter, for one) serves many vendors' models behind one " +
	"OpenAI-compatible address and one key, and a local OpenAI-compatible server is reached the same way. " +
	"abcd would use one for decision models and cheap judgements pointed at it by name, and never for a frontier " +
	"model, which the vendor denylist keeps on the host. Everything works without one: with no provider configured, " +
	"every delegated step runs on the host."

// KeyHomesProse is the prose above the choice of the key's home (criterion 8):
// the keychain is recommended here, in the prose, and never as a marked option.
const KeyHomesProse = "Where the key lives is your choice of three. The platform keychain is the safest home, " +
	"because the secret stays in the operating system's own store rather than in a file. A setup outside abcd " +
	"keeps it with a tool you already use, and abcd stores only its name. The abcd-only home keeps it in " +
	"~/.abcd/credentials.json, readable by you alone. This version stores a key in the abcd-only home; the other " +
	"two arrive with the credential store (" + CredentialStoreIntent + "). The key never enters the harness's " +
	"settings or the repository."
