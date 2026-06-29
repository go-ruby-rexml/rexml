// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"sort"
	"strings"
)

// base carries the common parent pointer for every node type.
type base struct {
	par *Element
}

func (b *base) parent() *Element     { return b.par }
func (b *base) setParent(e *Element) { b.par = e }

// Text is character data inside an element. REXML keeps the text in its raw,
// still-escaped form: a parsed "&#65;&amp;" round-trips byte-for-byte, while
// text added programmatically (already-decoded) is escaped on output. The Raw
// flag distinguishes the two — parsed text is Raw, NewText escapes on write.
type Text struct {
	base
	// Value is the raw stored string (escaped form when Raw, plain otherwise).
	Value string
	// Raw reports that Value already holds the serialised (escaped) bytes, so
	// writeTo emits it verbatim. Parsed text sets this; NewText leaves it false.
	Raw bool
}

// NewText builds a text node from an unescaped string (REXML's add_text). The
// string is escaped on serialisation.
func NewText(s string) *Text { return &Text{Value: s} }

func (t *Text) writeTo(w *writer) {
	if t.Raw {
		w.str(t.Value)
		return
	}
	w.str(escapeText(t.Value))
}

// String returns the raw stored bytes — REXML's Text#to_s.
func (t *Text) String() string {
	if t.Raw {
		return t.Value
	}
	return escapeText(t.Value)
}

// Val returns the unescaped character value — REXML's Text#value.
func (t *Text) Val() string {
	if t.Raw {
		return unescape(t.Value)
	}
	return t.Value
}

// Comment is an XML comment; its content is emitted verbatim between <!-- -->.
type Comment struct {
	base
	Value string
}

func (c *Comment) writeTo(w *writer) {
	w.str("<!--")
	w.str(c.Value)
	w.str("-->")
}

// CData is a CDATA section; its content is emitted verbatim between
// <![CDATA[ and ]]>.
type CData struct {
	base
	Value string
}

func (c *CData) writeTo(w *writer) {
	w.str("<![CDATA[")
	w.str(c.Value)
	w.str("]]>")
}

// Instruction is a processing instruction (PI): <?target content?>.
type Instruction struct {
	base
	Target  string
	Content string
}

func (i *Instruction) writeTo(w *writer) {
	w.str("<?")
	w.str(i.Target)
	if i.Content != "" {
		w.str(" ")
		w.str(i.Content)
	}
	w.str("?>")
}

// DocType is a document type declaration. The whole declaration is kept as its
// raw bytes (without the surrounding <!DOCTYPE ... >) so it round-trips.
type DocType struct {
	base
	// Body is the text between "<!DOCTYPE " and the closing ">".
	Body string
}

func (d *DocType) writeTo(w *writer) {
	w.str("<!DOCTYPE ")
	w.str(d.Body)
	w.str(">")
}

// XMLDecl is the XML declaration: <?xml version='1.0' ...?>. REXML emits the
// version, and the encoding / standalone only when present.
type XMLDecl struct {
	base
	Version    string
	Encoding   string
	Standalone string
}

func (x *XMLDecl) writeTo(w *writer) {
	w.str("<?xml version=")
	w.quoted(x.Version)
	if x.Encoding != "" {
		w.str(" encoding=")
		w.quoted(x.Encoding)
	}
	if x.Standalone != "" {
		w.str(" standalone=")
		w.quoted(x.Standalone)
	}
	w.str("?>")
}

// Attribute is a single element attribute. Prefix is the namespace prefix (the
// part before ':' in the qualified name), empty for an unprefixed attribute.
// Value holds the normalized (still-escaped) form when Raw is set — the byte
// form a parser captured — and the plain unescaped form otherwise.
type Attribute struct {
	Prefix string
	Name   string // local name
	Value  string
	Raw    bool // Value is already in normalized form (parsed); emit near-verbatim
}

// UnescapedValue returns the attribute's decoded value — REXML's Attribute#value.
func (a *Attribute) UnescapedValue() string {
	if a.Raw {
		return unescape(a.Value)
	}
	return a.Value
}

// QName is the qualified attribute name (prefix:name, or just name).
func (a *Attribute) QName() string {
	if a.Prefix != "" {
		return a.Prefix + ":" + a.Name
	}
	return a.Name
}

// Attributes is an ordered map of attributes keyed by qualified name, matching
// REXML's source order on serialisation.
type Attributes struct {
	order []string              // qualified names in insertion order
	byKey map[string]*Attribute // qualified name -> attribute
}

func newAttributes() *Attributes {
	return &Attributes{byKey: map[string]*Attribute{}}
}

// Get returns the attribute's decoded value for a qualified name and whether it
// was set — REXML's Attributes#[] (which returns the unnormalized value).
func (a *Attributes) Get(qname string) (string, bool) {
	at, ok := a.byKey[qname]
	if !ok {
		return "", false
	}
	return at.UnescapedValue(), true
}

// GetAttr returns the *Attribute for a qualified name, or nil.
func (a *Attributes) GetAttr(qname string) *Attribute {
	return a.byKey[qname]
}

// Set inserts or replaces an attribute with an unescaped value (programmatic
// add_attribute), preserving first-seen order.
func (a *Attributes) Set(qname, value string) {
	a.set(qname, value, false)
}

// setRaw inserts or replaces an attribute whose value is already in normalized
// (parsed) form, so it round-trips byte-for-byte.
func (a *Attributes) setRaw(qname, value string) {
	a.set(qname, value, true)
}

func (a *Attributes) set(qname, value string, raw bool) {
	prefix, name := splitQName(qname)
	if at, ok := a.byKey[qname]; ok {
		at.Value = value
		at.Raw = raw
		return
	}
	at := &Attribute{Prefix: prefix, Name: name, Value: value, Raw: raw}
	a.byKey[qname] = at
	a.order = append(a.order, qname)
}

// Delete removes an attribute by qualified name; it is a no-op if absent.
func (a *Attributes) Delete(qname string) {
	if _, ok := a.byKey[qname]; !ok {
		return
	}
	delete(a.byKey, qname)
	for i, k := range a.order {
		if k == qname {
			a.order = append(a.order[:i], a.order[i+1:]...)
			break
		}
	}
}

// Len is the number of attributes.
func (a *Attributes) Len() int { return len(a.order) }

// Each calls fn for every attribute in source / insertion order — REXML's
// each_attribute, the order the Pretty formatter uses.
func (a *Attributes) Each(fn func(*Attribute)) {
	for _, k := range a.order {
		fn(a.byKey[k])
	}
}

// EachSorted calls fn for every attribute ordered by local name with a stable
// sort — the order REXML's default (compact) formatter emits.
func (a *Attributes) EachSorted(fn func(*Attribute)) {
	attrs := make([]*Attribute, len(a.order))
	for i, k := range a.order {
		attrs[i] = a.byKey[k]
	}
	sort.SliceStable(attrs, func(i, j int) bool {
		return attrs[i].Name < attrs[j].Name
	})
	for _, at := range attrs {
		fn(at)
	}
}

func splitQName(qname string) (prefix, local string) {
	if i := strings.IndexByte(qname, ':'); i >= 0 {
		return qname[:i], qname[i+1:]
	}
	return "", qname
}
