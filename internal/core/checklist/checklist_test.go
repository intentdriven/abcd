package checklist_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/checklist"
)

// sample is a checklist that uses every field the v1 schema declares, so a
// round trip or a mutation table touches each of them.
func sample() checklist.Checklist {
	return checklist.Checklist{
		Schema:    checklist.Schema,
		UpdatedAt: "2026-10-23T08:00:00Z",
		UpdatedBy: "6f1c2a9e-3b4d-4e5f-8a7b-0c1d2e3f4a5b",
		Items: []checklist.Item{
			{
				ID:        "wrap-the-store",
				Title:     "Wrap the store in the checklist verbs",
				State:     checklist.Waiting,
				WaitsOn:   "pr:42",
				NextLook:  "2026-10-23T09:30:00Z",
				Promise:   &checklist.Promise{Trigger: "pr:42 merged", Act: "add the wrapper"},
				OwnerRefs: []string{"itd-2610170638184486", "pr:42", "branch:build/checklist-step1"},
				Credential: &checklist.Credential{
					Kind:   checklist.AppInstallationToken,
					Scope:  []checklist.CredentialScope{checklist.ScopeContentsWrite, checklist.ScopePullRequests},
					Expiry: "2026-10-24T03:10:00Z",
				},
			},
			{
				ID:      "renew-signing",
				Title:   "Ask for the signing key to be renewed",
				State:   checklist.Working,
				WaitsOn: "person:product-thinker",
				Credential: &checklist.Credential{
					Kind:   checklist.SigningKey,
					Scope:  []checklist.CredentialScope{checklist.ScopeSigning},
					Expiry: checklist.NoExpiry,
				},
			},
			{ID: "slot", Title: "Wait for a build slot", State: checklist.Waiting, WaitsOn: "slot:build"},
			{ID: "done-1", Title: "Read the spec", State: checklist.Done},
			{ID: "spc-ref", Title: "Track the spec", State: checklist.Dropped, WaitsOn: "spc-2610230805429429"},
		},
	}
}

func TestSampleValidates(t *testing.T) {
	if err := sample().Validate(); err != nil {
		t.Fatalf("the sample checklist is refused: %v", err)
	}
	c := sample()
	// The cap is characters, not bytes: 200 two-byte runes are within it.
	c.Items[0].Title = strings.Repeat("é", checklist.MaxTitleRunes)
	if err := c.Validate(); err != nil {
		t.Fatalf("a title of exactly %d characters is refused: %v", checklist.MaxTitleRunes, err)
	}
}

// bogus is the value each mutation plants. No refusal may repeat it: a value
// that fails the schema is untrusted, and may be a secret in the wrong field.
const bogus = "zzBOGUSzz"

func TestValidateNamesTheOffendingField(t *testing.T) {
	cases := []struct {
		field  string
		mutate func(c *checklist.Checklist)
	}{
		{"schema", func(c *checklist.Checklist) { c.Schema = "abcd/checklist/" + bogus }},
		{"updated_at", func(c *checklist.Checklist) { c.UpdatedAt = bogus }},
		{"updated_by", func(c *checklist.Checklist) { c.UpdatedBy = "" }},
		{"updated_by", func(c *checklist.Checklist) { c.UpdatedBy = bogus + "\x1b[2J" }},
		{"items[0].id", func(c *checklist.Checklist) { c.Items[0].ID = bogus }},
		{"items[1].id", func(c *checklist.Checklist) { c.Items[1].ID = c.Items[0].ID }},
		{"items[0].title", func(c *checklist.Checklist) { c.Items[0].Title = "" }},
		{"items[0].title", func(c *checklist.Checklist) {
			c.Items[0].Title = bogus + strings.Repeat("x", checklist.MaxTitleRunes)
		}},
		{"items[0].title", func(c *checklist.Checklist) { c.Items[0].Title = bogus + "\nsecond line" }},
		{"items[0].state", func(c *checklist.Checklist) { c.Items[0].State = bogus }},
		{"items[0].waits_on", func(c *checklist.Checklist) { c.Items[0].WaitsOn = bogus }},
		{"items[0].waits_on", func(c *checklist.Checklist) { c.Items[0].WaitsOn = "person:" + bogus }},
		{"items[0].waits_on", func(c *checklist.Checklist) { c.Items[0].WaitsOn = "pr:0" }},
		{"items[0].next_look", func(c *checklist.Checklist) { c.Items[0].NextLook = bogus }},
		{"items[0].promise.trigger", func(c *checklist.Checklist) { c.Items[0].Promise.Trigger = " " }},
		{"items[0].promise.act", func(c *checklist.Checklist) { c.Items[0].Promise.Act = "" }},
		{"items[0].owner_refs[1]", func(c *checklist.Checklist) { c.Items[0].OwnerRefs[1] = bogus }},
		{"items[0].owner_refs[2]", func(c *checklist.Checklist) { c.Items[0].OwnerRefs[2] = "branch:a.." + bogus }},
		{"items[0].credential.kind", func(c *checklist.Checklist) { c.Items[0].Credential.Kind = bogus }},
		{"items[0].credential.scope", func(c *checklist.Checklist) { c.Items[0].Credential.Scope = nil }},
		{"items[0].credential.scope[1]", func(c *checklist.Checklist) {
			c.Items[0].Credential.Scope[1] = bogus
		}},
		{"items[1].credential.expiry", func(c *checklist.Checklist) { c.Items[1].Credential.Expiry = bogus }},
	}
	for _, tc := range cases {
		c := sample()
		tc.mutate(&c)
		err := c.Validate()
		var fe *checklist.FieldError
		if !errors.As(err, &fe) {
			t.Errorf("%s: want a *FieldError, got %v", tc.field, err)
			continue
		}
		if fe.Field != tc.field {
			t.Errorf("want the refusal to name %s, it names %s (%v)", tc.field, fe.Field, err)
		}
		if strings.Contains(err.Error(), bogus) {
			t.Errorf("%s: the refusal repeats the refused value: %v", tc.field, err)
		}
	}
}

func TestDecodeIsStrict(t *testing.T) {
	good, err := json.Marshal(sample())
	if err != nil {
		t.Fatal(err)
	}
	got, err := checklist.Decode(good)
	if err != nil {
		t.Fatalf("a well-formed checklist is refused: %v", err)
	}
	if got.Items[0].Credential == nil || got.Items[0].Credential.Expiry != "2026-10-24T03:10:00Z" {
		t.Fatalf("decode lost the credential: %+v", got.Items[0])
	}

	refused := map[string]string{
		"unknown top-level field": strings.Replace(string(good), `"schema"`, `"mode":"away","schema"`, 1),
		"unknown item field":      strings.Replace(string(good), `"id":"slot"`, `"id":"slot","value":"x"`, 1),
		// The ADR's point: a credential holds kind, scope and expiry, and a
		// value field is not something the decoder quietly drops.
		"credential value":  strings.Replace(string(good), `"kind":"signing-key"`, `"kind":"signing-key","value":"x"`, 1),
		"repeated key":      strings.Replace(string(good), `"schema"`, `"updated_by":"x","schema"`, 1),
		"trailing document": string(good) + "{}",
		"not json":          "{not json",
	}
	for name, doc := range refused {
		if _, err := checklist.Decode([]byte(doc)); err == nil {
			t.Errorf("%s: decoded without complaint", name)
		}
	}
}
