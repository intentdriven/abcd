package openaiapi

// models.go is the model listing (spc-2610031241482088, step 1): one GET of
// {base}/models on the same pinned client as Complete, under the rules of
// itd-2610030821294016's decision 4, enforced here so that no caller can
// loosen them: no key unless the client holds one, no redirect followed, a
// short bound of its own, and the response size cap.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/termsafe"
)

const (
	// ListTimeout bounds one listing end to end, connecting and reading the
	// whole list, whatever the client's call limits are: those are sized for
	// a completion, and a list either arrives quickly or is not worth waiting
	// for.
	ListTimeout = 10 * time.Second
	// MaxListedModels is how many ids a listing keeps, in the service's order.
	MaxListedModels = 5000
	// maxListedIDBytes bounds a kept id. It is internal/core/oracle's bound
	// on a model name (validModel), so an id longer than any name the
	// configuration admits is dropped here rather than carried.
	maxListedIDBytes = 128
)

// errListTimeout is the cause ListTimeout cancels a listing with.
var errListTimeout = errors.New("list timeout")

// Listing is what a service lists. An id is bounded and free of the key and of
// every rune a terminal acts on, but not of characters a shell acts on (`;`,
// `$(`, a quote): a caller checks it as a model name (internal/core/oracle's
// validModel) before offering it or printing it in a command.
type Listing struct {
	IDs     []string // in the service's own order, each sanitised and bounded
	Dropped int      // listed names that were not usable model names
}

// ListError says why there is no list, in words a question can carry.
type ListError struct {
	NeedsKey bool   // the service answered 401 or 403
	Reason   string // "answered not found", "answered with a redirect, which abcd never follows", ...
	msg      string
}

// Error is the reason with the adapter's prefix and the host, scrubbed of
// the key like every other error the client returns.
func (e *ListError) Error() string { return e.msg }

// Models asks the service for the models it lists: one GET of {base}/models.
//
// The request carries the client's key as its bearer token and none when the
// client holds none; it follows no redirect (the pinned CheckRedirect); it is
// bounded by ListTimeout, and its body by MaxResponseBytes. The answer is
// read as the standard list, data[].id; an id is kept only when it is a
// non-empty line of valid UTF-8 within maxListedIDBytes that no reading of
// it (as sent, JSON escapes undone, character references resolved) reveals
// the key in, and that carries no rune the terminal sanitiser masks. Every
// other entry is dropped and counted, never kept redacted: a scrubbed id
// names no model. Which names the configuration admits is the caller's to
// judge. Every failure is a *ListError, its reason in plain words, and no
// part of the service's body is quoted in it.
func (c *Client) Models(ctx context.Context) (Listing, error) {
	lctx, cancel := context.WithTimeoutCause(ctx, c.listWait, errListTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(lctx, http.MethodGet, c.models, nil)
	if err != nil {
		return Listing{}, c.listFail(false, "could not be asked: the request could not be built")
	}
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return Listing{}, c.listCallError(ctx, lctx, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return Listing{}, c.listFail(true, fmt.Sprintf("answered HTTP %d: it lists its models only for a key", resp.StatusCode))
	case http.StatusNotFound:
		return Listing{}, c.listFail(false, "answered not found")
	default:
		return Listing{}, c.listFail(false, fmt.Sprintf("answered HTTP %d", resp.StatusCode))
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return Listing{}, c.listCallError(ctx, lctx, err)
	}
	if len(raw) > MaxResponseBytes {
		return Listing{}, c.listFail(false, fmt.Sprintf("answered with a list larger than %d bytes, so it is refused unread", MaxResponseBytes))
	}
	return c.decodeListing(raw)
}

// decodeListing reads data[].id from a list answer.
func (c *Client) decodeListing(raw []byte) (Listing, error) {
	var env struct {
		Data *[]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Data == nil {
		// The decoder's message can quote the body; it is dropped.
		return Listing{}, c.listFail(false, "answered with something that is not a model list")
	}
	if len(*env.Data) == 0 {
		return Listing{}, c.listFail(false, "listed no models")
	}
	var l Listing
	for _, entry := range *env.Data {
		var m struct {
			ID *string `json:"id"`
		}
		if json.Unmarshal(entry, &m) != nil || m.ID == nil || !c.usableID(*m.ID) {
			l.Dropped++
			continue
		}
		if len(l.IDs) < MaxListedModels {
			l.IDs = append(l.IDs, *m.ID)
		}
	}
	if len(l.IDs) == 0 {
		return Listing{}, c.listFail(false, fmt.Sprintf("listed no usable models (%d listed names were not usable model names)", l.Dropped))
	}
	// Each kept id is free of the key, but the key split across adjacent ids
	// passes every id's own check, so the kept ids are read once more as one
	// run. A service that splits the key so is hostile and gets no list. A key
	// split across ids that are not adjacent is not reassembled here.
	if c.carriesKey(strings.Join(l.IDs, "")) {
		return Listing{}, c.listFail(false, "listed names that together carry the key, so abcd keeps none of them")
	}
	return l, nil
}

// usableID reports whether a listed id may be kept as it is: within the
// bound, valid UTF-8, not blank, free of every rune the terminal sanitiser
// masks (a control, an escape, a bidi or zero-width rune), and carrying the
// key in no reading (carriesKey).
func (c *Client) usableID(id string) bool {
	if len(id) > maxListedIDBytes || !utf8.ValidString(id) || strings.TrimSpace(id) == "" ||
		termsafe.Sanitize(id) != id {
		return false
	}
	return !c.carriesKey(id)
}

// carriesKey reports whether s shows the client's key in any reading a
// reader of it could apply. The readings are the ones providerSaid undoes
// before its scrub: JSON escapes and HTML character references, alone and
// together. A keyless client carries no key.
func (c *Client) carriesKey(s string) bool {
	if len(c.forms) == 0 {
		return false
	}
	unescaped := unescapeJSONText(s)
	for _, r := range []string{s, unescaped, html.UnescapeString(s), html.UnescapeString(unescaped)} {
		if c.scrub(r) != r {
			return true
		}
	}
	return false
}

// listCallError is a listing that ended without an answer, each way in its
// own words. A limit is read from the cause it cancelled the listing with.
func (c *Client) listCallError(parent, lctx context.Context, err error) error {
	switch {
	case errors.Is(err, errRedirect):
		return c.listFail(false, "answered with a redirect, which abcd never follows")
	case errors.Is(parent.Err(), context.Canceled):
		return c.listFail(false, "was not waited for: the look-up was cancelled")
	case errors.Is(parent.Err(), context.DeadlineExceeded):
		return c.listFail(false, "was not waited for: the caller's deadline passed")
	case errors.Is(context.Cause(lctx), errListTimeout):
		return c.listFail(false, fmt.Sprintf("sent no list within %s", c.listWait))
	case neverConnected(err):
		return c.listFail(false, "could not be reached")
	}
	return c.listFail(false, "could not be reached: the connection failed before the list arrived")
}

// listFail is every listing failure: the reason scrubbed, and the error's
// text built through fail, so the key, in every form keyForms knows, is
// removed from both whatever built them.
func (c *Client) listFail(needsKey bool, reason string) error {
	reason = termsafe.Sanitize(c.scrub(reason))
	return &ListError{NeedsKey: needsKey, Reason: reason,
		msg: c.fail(c.host + " gave no model list: it " + reason).Error()}
}
