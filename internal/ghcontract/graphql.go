package ghcontract

import (
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// The forge's GraphQL schema, as far as abcd's queries reach into it: one
// field a line, "<Type>.<field> <named type> <kind>", and one line for each
// type a union can be, "on <Union> <Type>", read by introspection
// (graphql-fields.txt says how). Refresh it by running that query again.
//
//go:embed graphql-fields.txt
var graphqlFields string

// gqlField is one pinned field: its named type and that type's kind.
type gqlField struct{ named, kind string }

// gqlSchema is graphqlFields by type, then by field; gqlUnions is each
// union's possible types.
var gqlSchema, gqlUnions = func() (map[string]map[string]gqlField, map[string]map[string]bool) {
	s, u := map[string]map[string]gqlField{}, map[string]map[string]bool{}
	for _, line := range strings.Split(graphqlFields, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if f[0] == "on" && len(f) == 3 {
			if u[f[1]] == nil {
				u[f[1]] = map[string]bool{}
			}
			u[f[1]][f[2]] = true
			continue
		}
		typ, field, ok := strings.Cut(f[0], ".")
		if !ok || len(f) != 3 {
			panic("ghcontract: graphql-fields.txt: malformed line " + fmt.Sprintf("%q", line))
		}
		if s[typ] == nil {
			s[typ] = map[string]gqlField{}
		}
		s[typ][field] = gqlField{f[1], f[2]}
	}
	return s, u
}()

// The arguments of each pinned field: one a line, "<Type>.<field> <argument>
// <type> [default]", and a line "one <Type>.<field> <argument>..." for a field
// the forge refuses without at least one of them, read by introspection
// (graphql-args.txt says how).
//
//go:embed graphql-args.txt
var graphqlArgs string

// gqlArg is one pinned argument: its type as a query writes it, and whether
// the schema gives it a default.
type gqlArg struct {
	typ        string
	hasDefault bool
}

// required reports whether a query must pass the argument: a non-null type
// with no default, as GraphQL's validation holds it.
func (a gqlArg) required() bool { return strings.HasSuffix(a.typ, "!") && !a.hasDefault }

// gqlArgs is graphqlArgs by type, then field, then argument; gqlOneOf is each
// field's "one" lines, every set of arguments of which a query passes one.
var gqlArgs, gqlOneOf = func() (map[string]map[string]map[string]gqlArg, map[string]map[string][][]string) {
	a, o := map[string]map[string]map[string]gqlArg{}, map[string]map[string][][]string{}
	malformed := func(line string) {
		panic("ghcontract: graphql-args.txt: malformed line " + fmt.Sprintf("%q", line))
	}
	pinned := func(line, ref string) (string, string) {
		typ, field, ok := strings.Cut(ref, ".")
		if !ok {
			malformed(line)
		}
		if _, ok := gqlSchema[typ][field]; !ok {
			panic("ghcontract: graphql-args.txt: " + ref + " is not a field graphql-fields.txt pins")
		}
		return typ, field
	}
	for _, line := range strings.Split(graphqlArgs, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		if f[0] == "one" {
			if len(f) < 4 {
				malformed(line)
			}
			typ, field := pinned(line, f[1])
			if o[typ] == nil {
				o[typ] = map[string][][]string{}
			}
			o[typ][field] = append(o[typ][field], f[2:])
			continue
		}
		if len(f) != 3 && (len(f) != 4 || f[3] != "default") {
			malformed(line)
		}
		typ, field := pinned(line, f[0])
		if a[typ] == nil {
			a[typ] = map[string]map[string]gqlArg{}
		}
		if a[typ][field] == nil {
			a[typ][field] = map[string]gqlArg{}
		}
		a[typ][field][f[1]] = gqlArg{f[2], len(f) == 4}
	}
	for typ, fields := range o {
		for field, sets := range fields {
			for _, set := range sets {
				for _, name := range set {
					if _, ok := a[typ][field][name]; !ok {
						panic("ghcontract: graphql-args.txt: " + typ + "." + field + " has no argument " + name + " for its \"one\" line")
					}
				}
			}
		}
	}
	return a, o
}()

// typenameField is GraphQL's meta-field naming a selection's concrete type.
const typenameField = "__typename"

// GraphQLField reports the named type of field on the forge's GraphQL type
// typ, and whether the pinned schema has that field.
func GraphQLField(typ, field string) (string, bool) {
	f, ok := gqlSchema[typ][field]
	return f.named, ok
}

// CheckGraphQL refuses a GraphQL query that selects a field its type does not
// have in the pinned schema, as the forge refuses it, or passes a field an
// argument the field does not take, or leaves out one it requires (a
// non-null argument with no default, or one of a set the forge needs one of).
// It parses the shape abcd's queries take, and refuses any other: an optional
// `query` keyword, name and variable definitions, then nested selections of
// plain fields, each with optional arguments, and inline fragments
// (`... on Type { }`) on a union the type is pinned to be, or on the
// selection's own type. An alias (`p12: pullRequest(number:12)`) is read past
// to the field it names, which is checked as any other. `__typename`, the
// meta-field every selection may take, is read bare (no alias, arguments or
// selection). A named fragment, a directive or a mutation is refused as not
// parsed rather than passed unchecked. An argument's name is checked, its
// value only parsed: a value of the wrong type is not refused.
func CheckGraphQL(query string) error {
	p := &gqlParser{s: query}
	p.skip()
	if p.name() == "query" {
		p.skip()
		if p.peekName() {
			p.name()
			p.skip()
		}
		if p.peek() == '(' {
			if err := p.args(); err != nil {
				return err
			}
			p.skip()
		}
	} else {
		p.i = 0
		p.skip()
	}
	if err := p.selection("Query"); err != nil {
		return err
	}
	if p.skip(); p.i != len(p.s) {
		return p.unparsed()
	}
	return nil
}

// gqlParser walks one query, at byte i.
type gqlParser struct {
	s string
	i int
}

func (p *gqlParser) peek() byte {
	if p.i < len(p.s) {
		return p.s[p.i]
	}
	return 0
}

// skip passes what GraphQL ignores: white space, commas and comments.
func (p *gqlParser) skip() {
	for p.i < len(p.s) {
		switch c := p.s[p.i]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',':
			p.i++
		case c == '#':
			for p.i < len(p.s) && p.s[p.i] != '\n' {
				p.i++
			}
		default:
			return
		}
	}
}

func nameStart(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func (p *gqlParser) peekName() bool { return nameStart(p.peek()) }

// name reads a name at i, or "" when none starts there.
func (p *gqlParser) name() string {
	j := p.i
	for p.i < len(p.s) && (nameStart(p.s[p.i]) || p.s[p.i] >= '0' && p.s[p.i] <= '9') {
		if p.i == j && !nameStart(p.s[p.i]) {
			break
		}
		p.i++
	}
	return p.s[j:p.i]
}

// args passes a parenthesised list (arguments or variable definitions),
// balanced, with any string in it read as a string.
func (p *gqlParser) args() error {
	depth := 0
	for ; p.i < len(p.s); p.i++ {
		switch p.s[p.i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				p.i++
				return nil
			}
		case '"':
			for p.i++; p.i < len(p.s) && p.s[p.i] != '"'; p.i++ {
				if p.s[p.i] == '\\' {
					p.i++
				}
			}
		}
	}
	return fmt.Errorf("graphql: unbalanced parentheses in %q", p.s)
}

// fieldArgs reads a field's arguments, `(name: value, ...)`, at i, and returns
// the names passed. A value is parsed, not judged: a variable, a string, a
// number, a name (true, false, null or an enum value), a list or an object.
func (p *gqlParser) fieldArgs() (map[string]bool, error) {
	p.i++
	passed := map[string]bool{}
	for p.skip(); p.peek() != ')'; p.skip() {
		name := p.name()
		if p.skip(); name == "" || p.peek() != ':' {
			return nil, p.unparsed()
		}
		p.i++
		p.skip()
		if err := p.value(); err != nil {
			return nil, err
		}
		passed[name] = true
	}
	if len(passed) == 0 {
		// GraphQL's grammar takes one or more arguments in the parentheses.
		return nil, p.unparsed()
	}
	p.i++
	return passed, nil
}

// value reads one argument value at i.
func (p *gqlParser) value() error {
	switch c := p.peek(); {
	case c == '$':
		p.i++
		if p.name() == "" {
			return p.unparsed()
		}
	case strings.HasPrefix(p.s[p.i:], `"""`):
		end := strings.Index(strings.ReplaceAll(p.s[p.i+3:], `\"""`, "xxxx"), `"""`)
		if end < 0 {
			return p.unparsed()
		}
		p.i += 3 + end + 3
	case c == '"':
		for p.i++; p.i < len(p.s) && p.s[p.i] != '"' && p.s[p.i] != '\n'; p.i++ {
			if p.s[p.i] == '\\' {
				p.i++
			}
		}
		if p.peek() != '"' {
			return p.unparsed()
		}
		p.i++
	case c == '-' || c >= '0' && c <= '9':
		j := p.i
		for p.i++; p.i < len(p.s) && strings.IndexByte("0123456789.eE+-", p.s[p.i]) >= 0; p.i++ {
		}
		if !strings.ContainsAny(p.s[j:p.i], "0123456789") {
			p.i = j
			return p.unparsed()
		}
	case nameStart(c):
		p.name()
	case c == '[':
		p.i++
		for p.skip(); p.peek() != ']'; p.skip() {
			if err := p.value(); err != nil {
				return err
			}
		}
		p.i++
	case c == '{':
		p.i++
		for p.skip(); p.peek() != '}'; p.skip() {
			name := p.name()
			if p.skip(); name == "" || p.peek() != ':' {
				return p.unparsed()
			}
			p.i++
			p.skip()
			if err := p.value(); err != nil {
				return err
			}
		}
		p.i++
	default:
		return p.unparsed()
	}
	return nil
}

// checkArgs refuses the arguments passed to typ.field (none when passed is
// nil) as the forge refuses them: one the field does not take, a required one
// left out, or none of a set the forge needs one of.
func checkArgs(typ, field string, passed map[string]bool) error {
	pins := gqlArgs[typ][field]
	for _, name := range slices.Sorted(maps.Keys(passed)) {
		if _, ok := pins[name]; !ok {
			return fmt.Errorf("graphql: %s.%s takes no argument %q (the forge's schema as ghcontract pins it)", typ, field, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(pins)) {
		if a := pins[name]; a.required() && !passed[name] {
			return fmt.Errorf("graphql: %s.%s needs the argument %q, a %s (the forge's schema as ghcontract pins it)", typ, field, name, a.typ)
		}
	}
	for _, set := range gqlOneOf[typ][field] {
		if !slices.ContainsFunc(set, func(name string) bool { return passed[name] }) {
			return fmt.Errorf("graphql: %s.%s needs one of the arguments %s, which the forge refuses it without", typ, field, strings.Join(set, ", "))
		}
	}
	return nil
}

func (p *gqlParser) unparsed() error {
	return fmt.Errorf("graphql: %q at byte %d is not parsed by ghcontract's checker (only plain fields with arguments are)", p.s[p.i:min(p.i+16, len(p.s))], p.i)
}

// selection reads one selection set on typ, checking every field against the
// pinned schema and descending into each field's own selection.
func (p *gqlParser) selection(typ string) error {
	if p.peek() != '{' {
		return p.unparsed()
	}
	p.i++
	fields, union := gqlSchema[typ], gqlUnions[typ]
	if fields == nil && union == nil {
		return fmt.Errorf("graphql: the fields of %s are not pinned in ghcontract", typ)
	}
	n := 0
	for p.skip(); p.peek() != '}'; p.skip() {
		if p.i >= len(p.s) {
			return fmt.Errorf("graphql: unclosed selection on %s in %q", typ, p.s)
		}
		if strings.HasPrefix(p.s[p.i:], "...") {
			if err := p.fragment(typ, union); err != nil {
				return err
			}
			n++
			continue
		}
		name := p.name()
		if name == typenameField {
			// The meta-field every selection may take, on a union too: a
			// type's name, a String, with no arguments and no selection.
			if p.skip(); p.peek() == ':' || p.peek() == '(' || p.peek() == '{' || p.peek() == '@' {
				return p.unparsed()
			}
			n++
			continue
		}
		if union != nil && name != "" {
			return fmt.Errorf("graphql: %q is selected directly on the union %s, which takes only fragments on its types", name, typ)
		}
		if p.skip(); name != "" && p.peek() == ':' {
			// An alias: the field it names is the one read and checked.
			p.i++
			p.skip()
			name = p.name()
			p.skip()
		}
		if name == "" || p.peek() == ':' {
			return p.unparsed()
		}
		f, ok := fields[name]
		if !ok {
			return fmt.Errorf("graphql: %s has no field %q (the forge's schema as ghcontract pins it)", typ, name)
		}
		n++
		var passed map[string]bool
		if p.peek() == '(' {
			var err error
			if passed, err = p.fieldArgs(); err != nil {
				return err
			}
			p.skip()
		}
		if err := checkArgs(typ, name, passed); err != nil {
			return err
		}
		composite := f.kind == "OBJECT" || f.kind == "INTERFACE" || f.kind == "UNION"
		switch {
		case p.peek() == '{' && !composite:
			return fmt.Errorf("graphql: %s.%s is a %s and takes no selection", typ, name, f.named)
		case p.peek() == '{':
			if err := p.selection(f.named); err != nil {
				return err
			}
		case composite:
			return fmt.Errorf("graphql: %s.%s is a %s and needs a selection", typ, name, f.named)
		case p.peek() == '@':
			return p.unparsed()
		}
	}
	p.i++
	if n == 0 {
		return fmt.Errorf("graphql: an empty selection on %s", typ)
	}
	return nil
}

// fragment reads one inline fragment, `... on T { }`, in a selection on typ
// (union is typ's possible types when it is a union), and checks its own
// selection on T. T must be typ itself or, on a union, one of its types.
func (p *gqlParser) fragment(typ string, union map[string]bool) error {
	p.i += 3
	p.skip()
	if p.name() != "on" {
		return p.unparsed()
	}
	p.skip()
	on := p.name()
	if on == "" {
		return p.unparsed()
	}
	if on != typ && !union[on] {
		return fmt.Errorf("graphql: a fragment on %s can't be spread inside %s (the forge's schema as ghcontract pins it)", on, typ)
	}
	p.skip()
	return p.selection(on)
}
