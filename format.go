// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "strings"

// PrettyString returns the document rendered by the Pretty formatter with the
// given indent width (REXML's Formatters::Pretty.new(indent)).
func PrettyString(d *Document, indent int) string { return prettyString(d, indent) }

func prettyString(d *Document, indent int) string {
	f := &PrettyFormatter{Indentation: indent, Width: 80}
	w := &writer{}
	f.writeDocument(w, d)
	return w.b.String()
}

// PrettyFormatter is REXML::Formatters::Pretty. It re-indents the tree, drops
// whitespace-only text nodes between elements, collapses runs of whitespace in
// text, and (when Compact) keeps short text-only elements on one line.
type PrettyFormatter struct {
	Indentation int
	Width       int
	Compact     bool
}

// writeDocument mirrors Pretty#write_document: top-level Text nodes are skipped,
// and a newline separates successive top-level nodes (with REXML's first-child
// special-casing).
func (f *PrettyFormatter) writeDocument(w *writer, d *Document) {
	first := true
	for idx, c := range d.Children {
		if _, ok := c.(*Text); ok {
			continue
		}
		if !first {
			// REXML inserts a newline before every non-first emitted node,
			// except right after a non-writing first child. Our XMLDecl always
			// writes, so a plain "not first" check matches its observable output.
			if !(idx == 1 && isText(d.Children[0])) {
				w.str("\n")
			}
		}
		f.writeNode(w, c, 0)
		first = false
	}
}

func isText(n Node) bool { _, ok := n.(*Text); return ok }

func (f *PrettyFormatter) writeNode(w *writer, n Node, level int) {
	switch v := n.(type) {
	case *Element:
		f.writeElement(w, v, level)
	case *Text:
		f.writeText(w, v, level)
	case *Comment:
		w.str(indent(level))
		v.writeTo(w)
	case *CData:
		w.str(indent(level))
		v.writeTo(w)
	default:
		// XMLDecl, DocType, Instruction: emitted at column 0, verbatim.
		n.writeTo(w)
	}
}

func (f *PrettyFormatter) writeElement(w *writer, e *Element, level int) {
	w.str(indent(level))
	w.str("<")
	w.str(e.QName())
	e.writeAttrs(w, false) // Pretty keeps source order
	if len(e.Children) == 0 {
		w.str("/>")
		return
	}
	w.str(">")
	// Compact: if every child is text and the inlined render fits in Width,
	// keep it on one line.
	if f.Compact && allText(e.Children) {
		inner := &writer{}
		for _, c := range e.Children {
			f.writeNode(inner, c, 0)
		}
		s := inner.b.String()
		if len(s) < f.Width {
			w.str(s)
			w.str("</")
			w.str(e.QName())
			w.str(">")
			return
		}
	}
	w.str("\n")
	child := level + f.Indentation
	for _, c := range e.Children {
		if t, ok := c.(*Text); ok && strings.TrimSpace(t.String()) == "" {
			continue
		}
		f.writeNode(w, c, child)
		w.str("\n")
	}
	w.str(indent(level))
	w.str("</")
	w.str(e.QName())
	w.str(">")
}

// writeText mirrors Pretty#write_text: every whitespace char becomes a space,
// runs of spaces squeeze to one, the line wraps at Width-level, and each line is
// indented.
func (f *PrettyFormatter) writeText(w *writer, t *Text, level int) {
	s := t.String()
	s = squeezeSpaces(replaceWhitespace(s))
	s = wrapText(s, f.Width-level)
	s = indentText(s, level)
	w.str(indent(level))
	w.str(s)
}

func allText(children []Node) bool {
	for _, c := range children {
		if _, ok := c.(*Text); !ok {
			return false
		}
	}
	return len(children) > 0
}

func indent(level int) string {
	if level <= 0 {
		return ""
	}
	return strings.Repeat(" ", level)
}

func replaceWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n', '\f', '\v':
			return ' '
		}
		return r
	}, s)
}

func squeezeSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// wrapText breaks s at the last space before width, like Pretty#wrap.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	var parts []string
	for len(s) > width {
		place := strings.LastIndexByte(s[:width+1], ' ')
		if place < 0 {
			break
		}
		parts = append(parts, s[:place])
		s = s[place+1:]
	}
	parts = append(parts, s)
	return strings.Join(parts, "\n")
}

// indentText prefixes every newline-introduced line with level spaces, like
// Pretty#indent_text.
func indentText(s string, level int) string {
	if level < 0 {
		return s
	}
	return strings.ReplaceAll(s, "\n", "\n"+indent(level))
}
