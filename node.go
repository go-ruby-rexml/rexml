// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rexml is a pure-Go (no cgo) reimplementation of the core of Ruby's
// REXML XML library — the tree model, the parser, the serialisers (the default
// compact formatter and Formatters::Pretty), and a widely-used subset of XPath.
//
// It mirrors REXML's observable behaviour rather than wrapping encoding/xml:
// REXML stores text in its raw, still-escaped form, defaults to single-quoted
// attributes, self-closes empty elements, and preserves comments / CDATA /
// processing instructions verbatim on a parse→serialise round trip. The Go
// tokenizer is used internally where convenient, but the output and value model
// match MRI's REXML, not Go's encoding/xml.
package rexml

// Node is any node in a REXML tree: Element, Text, Comment, CData,
// Instruction, DocType, XMLDecl, or the Document itself.
type Node interface {
	// parent returns the node's parent, or nil for the document / a detached node.
	parent() *Element
	// setParent records a new parent (used by the Add* helpers).
	setParent(*Element)
	// writeTo emits the node's default (compact) serialisation.
	writeTo(*writer)
}

// Child is a node that can live inside an Element's child list and also at the
// Document's top level. Every Node type implements it.
type Child = Node
