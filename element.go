// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "strings"

// Element is an XML element node. Name is the local name; Prefix is the
// namespace prefix (empty when unprefixed). Attributes preserves source order;
// Children holds the ordered child nodes.
type Element struct {
	base
	Prefix     string
	Name       string
	Attributes *Attributes
	Children   []Node
}

// NewElement builds a detached element from a qualified name (prefix:name).
func NewElement(qname string) *Element {
	prefix, name := splitQName(qname)
	return &Element{Prefix: prefix, Name: name, Attributes: newAttributes()}
}

func (e *Element) writeTo(w *writer) { e.writeOpen(w) }

// QName is the element's qualified name (prefix:name or name).
func (e *Element) QName() string {
	if e.Prefix != "" {
		return e.Prefix + ":" + e.Name
	}
	return e.Name
}

// ExpandedName is REXML's expanded_name: prefix:name when prefixed, else name.
func (e *Element) ExpandedName() string { return e.QName() }

// Parent returns the parent element, or nil at the root.
func (e *Element) Parent() *Element { return e.par }

// NamespaceURI resolves the element's own prefix to a namespace URI by walking
// up the xmlns declarations, mirroring REXML's Element#namespace.
func (e *Element) NamespaceURI() string { return e.resolvePrefix(e.Prefix) }

// resolvePrefix finds the xmlns[:prefix] declaration in scope for prefix.
func (e *Element) resolvePrefix(prefix string) string {
	key := "xmlns"
	if prefix != "" {
		key = "xmlns:" + prefix
	}
	for cur := e; cur != nil; cur = cur.par {
		if v, ok := cur.Attributes.Get(key); ok {
			return v
		}
	}
	return ""
}

// Add appends any node as a child, setting its parent.
func (e *Element) Add(n Node) Node {
	n.setParent(e)
	e.Children = append(e.Children, n)
	return n
}

// AddElement creates a child element with the given qualified name, appends it,
// and returns it (REXML's add_element).
func (e *Element) AddElement(qname string) *Element {
	child := NewElement(qname)
	e.Add(child)
	return child
}

// AddAttribute sets an attribute by qualified name (REXML's add_attribute).
func (e *Element) AddAttribute(qname, value string) {
	e.Attributes.Set(qname, value)
}

// AddText appends an unescaped text node (REXML's add_text). Consecutive
// add_text calls in REXML coalesce; this mirrors that for raw==raw text nodes.
func (e *Element) AddText(s string) {
	if n := len(e.Children); n > 0 {
		if t, ok := e.Children[n-1].(*Text); ok && !t.Raw {
			t.Value += s
			return
		}
	}
	e.Add(NewText(s))
}

// Attr returns an attribute value by qualified name and whether it is present.
func (e *Element) Attr(qname string) (string, bool) {
	return e.Attributes.Get(qname)
}

// SetText replaces the element's first text node (or inserts one) with s,
// matching REXML's Element#text=.
func (e *Element) SetText(s string) {
	for i, c := range e.Children {
		if _, ok := c.(*Text); ok {
			e.Children[i] = &Text{base: base{par: e}, Value: s}
			return
		}
	}
	// REXML inserts the text as the first child.
	t := &Text{base: base{par: e}, Value: s}
	e.Children = append([]Node{t}, e.Children...)
}

// GetText returns the first child Text node, or nil — REXML's get_text.
func (e *Element) GetText() *Text {
	for _, c := range e.Children {
		if t, ok := c.(*Text); ok {
			return t
		}
	}
	return nil
}

// Text returns the unescaped value of the first text node, or "" if none —
// REXML's Element#text (which returns nil; "" is the empty-text proxy here).
func (e *Element) Text() string {
	if t := e.GetText(); t != nil {
		return t.Val()
	}
	return ""
}

// HasText reports whether the element has at least one text child.
func (e *Element) HasText() bool { return e.GetText() != nil }

// Texts returns every direct Text child — REXML's texts.
func (e *Element) Texts() []*Text {
	var out []*Text
	for _, c := range e.Children {
		if t, ok := c.(*Text); ok {
			out = append(out, t)
		}
	}
	return out
}

// ChildElements returns the direct child elements in order.
func (e *Element) ChildElements() []*Element {
	var out []*Element
	for _, c := range e.Children {
		if el, ok := c.(*Element); ok {
			out = append(out, el)
		}
	}
	return out
}

// EachElement calls fn for every direct child element (REXML's each_element).
func (e *Element) EachElement(fn func(*Element)) {
	for _, el := range e.ChildElements() {
		fn(el)
	}
}

// Elements resolves an XPath-style path against this element and returns the
// matching elements — REXML's Element#elements[path] / each. A bare positive
// integer selects the 1-based nth child element.
func (e *Element) Elements(path string) []*Element {
	return e.elementsByPath(path)
}

// ElementAt returns the element at the 1-based child-element index, or nil —
// REXML's elements[n] with an Integer.
func (e *Element) ElementAt(n int) *Element {
	kids := e.ChildElements()
	if n >= 1 && n <= len(kids) {
		return kids[n-1]
	}
	return nil
}

// FirstElement returns the first element matching the path, or nil.
func (e *Element) FirstElement(path string) *Element {
	if got := e.elementsByPath(path); len(got) > 0 {
		return got[0]
	}
	return nil
}

func (e *Element) elementsByPath(path string) []*Element {
	matches := matchPath(e, path)
	var out []*Element
	for _, m := range matches {
		if el, ok := m.(*Element); ok {
			out = append(out, el)
		}
	}
	return out
}

// writeOpen emits the element in REXML's compact form:
// <name a='1'>children</name>, self-closing when empty.
func (e *Element) writeOpen(w *writer) {
	w.str("<")
	w.str(e.QName())
	e.writeAttrs(w, true)
	if len(e.Children) == 0 {
		w.str("/>")
		return
	}
	w.str(">")
	for _, c := range e.Children {
		c.writeTo(w)
	}
	w.str("</")
	w.str(e.QName())
	w.str(">")
}

// writeAttrs emits the attributes. The default (compact) formatter sorts by
// local name; the Pretty formatter passes sorted=false to keep source order.
func (e *Element) writeAttrs(w *writer, sorted bool) {
	emit := func(a *Attribute) {
		w.str(" ")
		w.str(a.QName())
		w.str("=")
		w.quotedAttr(a)
	}
	if sorted {
		e.Attributes.EachSorted(emit)
	} else {
		e.Attributes.Each(emit)
	}
}

// isWhitespace reports whether s is empty or only XML whitespace.
func isWhitespace(s string) bool {
	return strings.TrimLeft(s, " \t\r\n") == ""
}
