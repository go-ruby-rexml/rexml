// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

// Document is the root of a REXML tree. Its Children hold the top-level nodes:
// an optional XMLDecl, DocType, comments / PIs, and exactly one root element.
type Document struct {
	base
	Children []Node
}

// NewDocument returns an empty document.
func NewDocument() *Document { return &Document{} }

func (d *Document) parent() *Element   { return nil }
func (d *Document) setParent(*Element) {}
func (d *Document) writeTo(w *writer) {
	for _, c := range d.Children {
		c.writeTo(w)
	}
}

// Add appends a top-level node. Element children get the document recorded as a
// nil parent (REXML root elements have no parent element).
func (d *Document) Add(n Node) Node {
	n.setParent(nil)
	d.Children = append(d.Children, n)
	return n
}

// AddElement creates the root element, appends it, and returns it.
func (d *Document) AddElement(qname string) *Element {
	e := NewElement(qname)
	d.Add(e)
	return e
}

// Root returns the document's root element, or nil — REXML's Document#root.
func (d *Document) Root() *Element {
	for _, c := range d.Children {
		if e, ok := c.(*Element); ok {
			return e
		}
	}
	return nil
}

// RootNode returns the document itself — REXML's root_node returns the
// containing Document for a Document.
func (d *Document) RootNode() *Document { return d }

// XMLDecl returns the document's XML declaration, or nil.
func (d *Document) XMLDecl() *XMLDecl {
	for _, c := range d.Children {
		if x, ok := c.(*XMLDecl); ok {
			return x
		}
	}
	return nil
}

// DocType returns the document's DOCTYPE declaration, or nil.
func (d *Document) DocType() *DocType {
	for _, c := range d.Children {
		if dt, ok := c.(*DocType); ok {
			return dt
		}
	}
	return nil
}

// ToString returns the document's default (compact) serialisation — REXML's
// Document#to_s.
func (d *Document) ToString() string {
	w := &writer{}
	d.writeTo(w)
	return w.b.String()
}

// String makes Document satisfy fmt.Stringer, aliasing ToString.
func (d *Document) String() string { return d.ToString() }

// WriteOptions configures Document.Write. The zero value selects the default
// compact formatter (REXML's Document#write with no indent).
type WriteOptions struct {
	// Indent selects the Pretty formatter with this indent width when > 0.
	// REXML uses -1 to mean "no indentation" (the compact default).
	Indent int
	// Pretty forces the Pretty formatter even when Indent is 0 (REXML default
	// indent of 2). Ignored when Indent > 0.
	Pretty bool
}

// Write serialises the document with the chosen formatter and returns the text.
func (d *Document) Write(opts WriteOptions) string {
	switch {
	case opts.Indent > 0:
		return prettyString(d, opts.Indent)
	case opts.Pretty:
		return prettyString(d, 2)
	default:
		return d.ToString()
	}
}

// Pretty renders the document with the Pretty formatter at the given indent —
// a convenience for REXML::Formatters::Pretty.new(indent).write(doc, out).
func (d *Document) Pretty(indent int) string { return prettyString(d, indent) }
