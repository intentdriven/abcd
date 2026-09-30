package update

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The CA canary (spc-32 criterion 3, iss-2609012111162089): the release origin
// is a TLS server whose certificate no system pool trusts, and the canary is a
// CA bundle holding exactly that certificate, planted where crypto/x509 looks
// for an override (SSL_CERT_FILE, SSL_CERT_DIR). A read of the canary is
// observable: it is the only way the handshake with this server can succeed.

// canaryOrigin serves one release's checksums over TLS under the test CA.
func canaryOrigin(t *testing.T) *httptest.Server {
	t.Helper()
	sum := strings.Repeat("ab", 32) + "  " + testAssetName + "\n"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/download/v0.6.2/checksums.txt" {
			_, _ = w.Write([]byte(sum))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// plantCanary writes the test CA as a PEM bundle into a fresh directory and
// returns the bundle's path and the directory.
func plantCanary(t *testing.T, srv *httptest.Server) (string, string) {
	t.Helper()
	dir := t.TempDir()
	bundle := filepath.Join(dir, "canary-ca.pem")
	block := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(bundle, block, 0o600); err != nil {
		t.Fatal(err)
	}
	return bundle, dir
}

// isVerificationFailure reports whether err is the handshake refusing the
// server's certificate — not a refused connection, a timeout or a 404, any of
// which would make a negative assertion vacuous.
func isVerificationFailure(err error) bool {
	var cve *tls.CertificateVerificationError
	return errors.As(err, &cve)
}

// TestTransportConsultsItsPinnedRootPool: the client trusts the root pool the
// updater was built with, and only that pool. Built around a pool holding the
// test CA, the fetch succeeds; built around a pool without it, the fetch is a
// certificate verification failure even though the canary naming that very CA
// sits in SSL_CERT_FILE and SSL_CERT_DIR. So the configured bundle is what the
// handshake consults, not the environment.
func TestTransportConsultsItsPinnedRootPool(t *testing.T) {
	srv := canaryOrigin(t)
	bundle, dir := plantCanary(t, srv)
	t.Setenv("SSL_CERT_FILE", bundle)
	t.Setenv("SSL_CERT_DIR", dir)

	trusting := x509.NewCertPool()
	trusting.AddCert(srv.Certificate())
	u := buildUpdater(srv.URL, nil, testAssetName, false, nil, nil, trusting)
	if _, found, err := u.fetchChecksums("v0.6.2"); err != nil || !found {
		t.Fatalf("a pool holding the origin's CA did not verify it (found=%v): %v — the configured pool is not what the handshake consults", found, err)
	}

	bare := buildUpdater(srv.URL, nil, testAssetName, false, nil, nil, x509.NewCertPool())
	_, _, err := bare.fetchChecksums("v0.6.2")
	if err == nil {
		t.Fatal("a pool without the origin's CA verified it: the planted canary was consulted")
	}
	if !isVerificationFailure(err) {
		t.Fatalf("want a certificate verification failure, got: %v", err)
	}
}

// Child-process modes for TestShippingConstructorNeverTrustsAPlantedCA. A fresh
// process is the only honest place to test the shipping constructor:
// crypto/x509 loads the system pool once per process, so in this one the pool
// may already have been loaded, before the canary was planted, and the test
// would pass whether or not the scrub ran.
const (
	canaryChildEnv  = "ABCD_UPDATE_CA_CANARY_CHILD"
	canaryOriginEnv = "ABCD_UPDATE_CA_CANARY_ORIGIN"
	canaryResult    = "canary-result: "
)

// TestShippingConstructorNeverTrustsAPlantedCA runs two fresh processes under
// the same planted canary. The control loads the system pool the way any
// client would and handshakes with the origin: where that succeeds, a read of
// the canary is observable on this platform. The subject builds the updater
// through the shipping constructor and fetches: it must fail verification,
// because the constructor takes the overrides out of the environment before
// the pool is loaded. Where the control cannot observe the canary (a platform
// verifier that ignores the variables), the subject proves nothing and the
// test says so and skips.
func TestShippingConstructorNeverTrustsAPlantedCA(t *testing.T) {
	switch os.Getenv(canaryChildEnv) {
	case "control":
		canaryControlChild()
		return
	case "subject":
		canarySubjectChild()
		return
	}
	srv := canaryOrigin(t)
	bundle, dir := plantCanary(t, srv)
	run := func(mode string) string {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestShippingConstructorNeverTrustsAPlantedCA$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(),
			canaryChildEnv+"="+mode,
			canaryOriginEnv+"="+srv.URL,
			"SSL_CERT_FILE="+bundle,
			"SSL_CERT_DIR="+dir,
			// Go 1.27 honours the overrides on macOS too, behind this setting;
			// an older toolchain ignores it. Either way the control decides.
			"GODEBUG=x509sslcertoverrideplatform=1",
		)
		out, err := cmd.CombinedOutput()
		for _, line := range strings.Split(string(out), "\n") {
			if i := strings.Index(line, canaryResult); i >= 0 {
				return strings.TrimSpace(line[i+len(canaryResult):])
			}
		}
		t.Fatalf("the %s child reported no result (err %v):\n%s", mode, err, out)
		return ""
	}
	if got := run("control"); got != "trusted" {
		t.Skipf("the canary is not observable on this platform: a client reading the overrides did not trust it (%s)", got)
	}
	if got := run("subject"); got != "verification-failed" {
		t.Fatalf("the shipping updater's fetch under a planted CA reported %q, want verification-failed: the canary was read", got)
	}
}

// canaryControlChild handshakes through the process's default pool, loaded with
// the canary in the environment.
func canaryControlChild() {
	pool, err := x509.SystemCertPool()
	if err != nil {
		fmt.Println(canaryResult + "no-system-pool: " + err.Error())
		return
	}
	c := &http.Client{Transport: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{RootCAs: pool}}}
	reportCanary(c.Get(os.Getenv(canaryOriginEnv) + "/releases/download/v0.6.2/checksums.txt"))
}

// canarySubjectChild fetches through the shipping constructor, the first thing
// this process does.
func canarySubjectChild() {
	u := newUpdater(os.Getenv(canaryOriginEnv), nil, testAssetName, false, nil)
	_, _, err := u.fetchChecksums("v0.6.2")
	reportCanary(nil, err)
}

func reportCanary(resp *http.Response, err error) {
	if resp != nil {
		resp.Body.Close()
	}
	switch {
	case err == nil:
		fmt.Println(canaryResult + "trusted")
	case isVerificationFailure(err):
		fmt.Println(canaryResult + "verification-failed")
	default:
		fmt.Println(canaryResult + "other: " + err.Error())
	}
}
