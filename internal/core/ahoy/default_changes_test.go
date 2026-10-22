package ahoy

import (
	"strings"
	"testing"
)

// TestEmbeddedDefaultChangesLoad holds the table the binary carries to its
// own reader: it parses, and every entry names a setup key whose repository
// setting Waiting can read and a well-formed record id.
func TestEmbeddedDefaultChangesLoad(t *testing.T) {
	changes, err := defaultChanges()
	if err != nil {
		t.Fatalf("the embedded default-change table does not load: %v", err)
	}
	for _, c := range changes {
		if _, ok := configValueSlots[c.Key]; !ok {
			t.Errorf("entry %s/%s names a key with no repository setting", c.Key, c.Record)
		}
	}
}

// TestDefaultChangesRefuseWhatIsNotASetupDefault: a key setup does not ask, a
// setup key whose setting the repository layer does not hold, and a record
// id that is not itd-N or iss-N are each refused, as are an unknown field, a
// repeated entry and another schema. A well-formed entry is accepted.
func TestDefaultChangesRefuseWhatIsNotASetupDefault(t *testing.T) {
	doc := func(entries string) string {
		return `{"schema_version": 1, "changes": [` + entries + `]}`
	}
	ok, err := parseDefaultChanges([]byte(doc(`{"key": "docs_target", "record": "itd-2610161616086564"}`)))
	if err != nil {
		t.Fatalf("a well-formed entry was refused: %v", err)
	}
	if len(ok) != 1 || ok[0].Key != "docs_target" || ok[0].Record != "itd-2610161616086564" {
		t.Fatalf("parsed %+v", ok)
	}
	if empty, err := parseDefaultChanges([]byte(doc(``))); err != nil || len(empty) != 0 {
		t.Fatalf("an empty table: %+v, %v", empty, err)
	}
	for name, c := range map[string]struct{ data, want string }{
		"not a setup key":      {doc(`{"key": "no_such_question", "record": "itd-1"}`), "not a setup question key"},
		"a setup key, no slot": {doc(`{"key": "statusline.repo", "record": "itd-1"}`), "no setting in .abcd/config.json"},
		"an approval":          {doc(`{"key": "approve.config-change", "record": "iss-1"}`), "no setting in .abcd/config.json"},
		"malformed record":     {doc(`{"key": "docs_target", "record": "dcn-1"}`), "not a record id"},
		"record with suffix":   {doc(`{"key": "docs_target", "record": "itd-1-x"}`), "not a record id"},
		"empty record":         {doc(`{"key": "docs_target", "record": ""}`), "not a record id"},
		"overlong record":      {doc(`{"key": "docs_target", "record": "itd-` + strings.Repeat("1", 40) + `"}`), "not a record id"},
		"repeated entry":       {doc(`{"key": "docs_target", "record": "itd-1"}, {"key": "docs_target", "record": "itd-1"}`), "twice"},
		"unknown field":        {doc(`{"key": "docs_target", "record": "itd-1", "release": "v0.16.0"}`), "unknown field"},
		"another schema":       {`{"schema_version": 2, "changes": []}`, "schema_version"},
		"no schema":            {`{"changes": []}`, "schema_version"},
		"trailing content":     {doc(``) + `{}`, ""},
		"over the cap":         {doc(``) + strings.Repeat(" ", maxDefaultChangesBytes), "cap"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseDefaultChanges([]byte(c.data))
			if err == nil {
				t.Fatal("accepted")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused with %q, want it to say %q", err, c.want)
			}
		})
	}
}
