// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"fmt"
	"strings"
)

// ParseError reports a parse failure with a byte offset, mirroring the fact
// that REXML::Document.new raises ParseException on malformed input.
type ParseError struct {
	Offset int
	Msg    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("rexml parse error at offset %d: %s", e.Offset, e.Msg)
}

// ParseDocument parses an XML string into a Document tree, mirroring
// REXML::Document.new. Text, comments, CDATA, PIs, the XML declaration and the
// DOCTYPE are preserved so a parse→serialise round trip matches MRI's REXML.
func ParseDocument(xml string) (*Document, error) {
	p := &parser{src: xml}
	doc := NewDocument()
	if err := p.parseTop(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

type parser struct {
	src string
	pos int
	// lastOpenWasEmpty records whether the start tag just read self-closed, so
	// parseTop knows not to descend into it.
	lastOpenWasEmpty bool
}

func (p *parser) errf(format string, a ...any) error {
	return &ParseError{Offset: p.pos, Msg: fmt.Sprintf(format, a...)}
}

// parseTop reads the prolog (whitespace, XML decl, DOCTYPE, comments, PIs), the
// single root element, and any trailing misc nodes.
func (p *parser) parseTop(doc *Document) error {
	var cur *Element // the element currently being filled, nil at top level
	for p.pos < len(p.src) {
		if p.src[p.pos] != '<' {
			if err := p.readText(doc, cur); err != nil {
				return err
			}
			continue
		}
		n, closing, err := p.readMarkup(cur)
		if err != nil {
			return err
		}
		if closing != "" {
			if cur == nil {
				return p.errf("unexpected closing tag </%s>", closing)
			}
			if closing != cur.QName() {
				return p.errf("mismatched closing tag: </%s> for <%s>", closing, cur.QName())
			}
			cur = cur.par
			continue
		}
		if n == nil {
			continue // a self-handled construct (decl/doctype already added)
		}
		// Attach the node.
		if cur == nil {
			doc.Add(n)
		} else {
			cur.Add(n)
		}
		// Descend into a freshly opened, non-self-closed element.
		if el, ok := n.(*Element); ok && p.lastOpenWasEmpty == false {
			cur = el
		}
		p.lastOpenWasEmpty = false
	}
	if cur != nil {
		return p.errf("unclosed element <%s>", cur.QName())
	}
	if doc.Root() == nil {
		return p.errf("no root element")
	}
	return nil
}

// lastOpenWasEmpty records whether the element just emitted by readMarkup was
// self-closing (so parseTop should not descend into it).
func (p *parser) markEmptyOpen(empty bool) { p.lastOpenWasEmpty = empty }

// readText consumes a run of character data up to the next '<'. Inside an
// element it is stored raw. At the top level REXML rejects non-whitespace text;
// whitespace before the root is preserved (prolog spacing), whitespace after
// the root is dropped.
func (p *parser) readText(doc *Document, cur *Element) error {
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '<' {
		p.pos++
	}
	raw := p.src[start:p.pos]
	if cur == nil {
		if !isWhitespace(raw) {
			p.pos = start
			return p.errf("text %q outside the root element", raw)
		}
		if doc.Root() == nil {
			doc.Add(&Text{Value: raw, Raw: true}) // prolog whitespace, preserved
		}
		// trailing whitespace after the root is discarded
		return nil
	}
	cur.Add(&Text{Value: raw, Raw: true})
	return nil
}

// readMarkup dispatches on the construct following '<'. It returns the new node
// (or nil if it self-handled, e.g. an XML decl appended to the doc directly),
// or a non-empty closing name for an end tag.
func (p *parser) readMarkup(cur *Element) (Node, string, error) {
	rest := p.src[p.pos:]
	switch {
	case strings.HasPrefix(rest, "<?xml") && (len(rest) == 5 || isXMLDeclBoundary(rest[5])):
		return p.readXMLDecl()
	case strings.HasPrefix(rest, "<?"):
		return p.readPI()
	case strings.HasPrefix(rest, "<!--"):
		return p.readComment()
	case strings.HasPrefix(rest, "<![CDATA["):
		return p.readCData()
	case strings.HasPrefix(rest, "<!DOCTYPE"):
		return p.readDocType()
	case strings.HasPrefix(rest, "</"):
		return p.readEndTag()
	default:
		return p.readStartTag(cur)
	}
}

func isXMLDeclBoundary(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '?'
}

func (p *parser) readXMLDecl() (Node, string, error) {
	end := strings.Index(p.src[p.pos:], "?>")
	if end < 0 {
		return nil, "", p.errf("unterminated XML declaration")
	}
	inner := p.src[p.pos+5 : p.pos+end] // after "<?xml"
	p.pos += end + 2
	x := &XMLDecl{}
	attrs := parsePseudoAttrs(inner)
	x.Version = attrs["version"]
	x.Encoding = attrs["encoding"]
	x.Standalone = attrs["standalone"]
	if x.Version == "" {
		x.Version = "1.0"
	}
	return x, "", nil
}

func (p *parser) readPI() (Node, string, error) {
	end := strings.Index(p.src[p.pos:], "?>")
	if end < 0 {
		return nil, "", p.errf("unterminated processing instruction")
	}
	inner := p.src[p.pos+2 : p.pos+end]
	p.pos += end + 2
	target := inner
	content := ""
	if i := strings.IndexAny(inner, " \t\r\n"); i >= 0 {
		target = inner[:i]
		content = strings.TrimLeft(inner[i:], " \t\r\n")
	}
	return &Instruction{Target: target, Content: content}, "", nil
}

func (p *parser) readComment() (Node, string, error) {
	end := strings.Index(p.src[p.pos+4:], "-->")
	if end < 0 {
		return nil, "", p.errf("unterminated comment")
	}
	val := p.src[p.pos+4 : p.pos+4+end]
	p.pos += 4 + end + 3
	return &Comment{Value: val}, "", nil
}

func (p *parser) readCData() (Node, string, error) {
	end := strings.Index(p.src[p.pos+9:], "]]>")
	if end < 0 {
		return nil, "", p.errf("unterminated CDATA section")
	}
	val := p.src[p.pos+9 : p.pos+9+end]
	p.pos += 9 + end + 3
	return &CData{Value: val}, "", nil
}

// readDocType captures the whole DOCTYPE declaration, including a bracketed
// internal subset, as raw bytes between "<!DOCTYPE " and the matching ">".
func (p *parser) readDocType() (Node, string, error) {
	start := p.pos + len("<!DOCTYPE")
	i := start
	depth := 0
	for i < len(p.src) {
		switch p.src[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '>':
			if depth == 0 {
				body := strings.TrimLeft(p.src[start:i], " \t\r\n")
				p.pos = i + 1
				return &DocType{Body: body}, "", nil
			}
		}
		i++
	}
	return nil, "", p.errf("unterminated DOCTYPE")
}

func (p *parser) readEndTag() (Node, string, error) {
	end := strings.IndexByte(p.src[p.pos:], '>')
	if end < 0 {
		return nil, "", p.errf("unterminated end tag")
	}
	name := strings.TrimSpace(p.src[p.pos+2 : p.pos+end])
	p.pos += end + 1
	return nil, name, nil
}

// readStartTag parses an element start tag and its attributes. It sets
// lastOpenWasEmpty so parseTop knows whether to descend.
func (p *parser) readStartTag(_ *Element) (Node, string, error) {
	i := p.pos + 1
	// Read the qualified name.
	nameStart := i
	for i < len(p.src) && !isNameBoundary(p.src[i]) {
		i++
	}
	if i == nameStart {
		return nil, "", p.errf("empty element name")
	}
	qname := p.src[nameStart:i]
	el := NewElement(qname)

	// Parse attributes until '>' or '/>'.
	for {
		i = skipSpace(p.src, i)
		if i >= len(p.src) {
			return nil, "", p.errf("unterminated start tag <%s>", qname)
		}
		switch p.src[i] {
		case '>':
			p.pos = i + 1
			p.markEmptyOpen(false)
			return el, "", nil
		case '/':
			if i+1 < len(p.src) && p.src[i+1] == '>' {
				p.pos = i + 2
				p.markEmptyOpen(true)
				return el, "", nil
			}
			return nil, "", p.errf("malformed empty element <%s>", qname)
		}
		// An attribute: name = "value" | 'value'.
		aStart := i
		for i < len(p.src) && !isNameBoundary(p.src[i]) && p.src[i] != '=' {
			i++
		}
		aname := p.src[aStart:i]
		if aname == "" {
			return nil, "", p.errf("malformed attribute in <%s>", qname)
		}
		i = skipSpace(p.src, i)
		if i >= len(p.src) || p.src[i] != '=' {
			return nil, "", p.errf("attribute %q missing '=' in <%s>", aname, qname)
		}
		i = skipSpace(p.src, i+1)
		if i >= len(p.src) || (p.src[i] != '"' && p.src[i] != '\'') {
			return nil, "", p.errf("attribute %q missing quoted value in <%s>", aname, qname)
		}
		quote := p.src[i]
		i++
		vStart := i
		for i < len(p.src) && p.src[i] != quote {
			i++
		}
		if i >= len(p.src) {
			return nil, "", p.errf("unterminated attribute value for %q in <%s>", aname, qname)
		}
		raw := p.src[vStart:i]
		i++
		// Store the value in its raw, still-escaped form so it round-trips
		// byte-for-byte, mirroring REXML's @normalized.
		el.Attributes.setRaw(aname, raw)
	}
}

// parsePseudoAttrs extracts name='value' / name="value" pairs from an XML / PI
// declaration body.
func parsePseudoAttrs(s string) map[string]string {
	out := map[string]string{}
	i := 0
	for i < len(s) {
		i = skipSpace(s, i)
		if i >= len(s) {
			break
		}
		ns := i
		for i < len(s) && s[i] != '=' && !isSpace(s[i]) {
			i++
		}
		name := s[ns:i]
		i = skipSpace(s, i)
		if i >= len(s) || s[i] != '=' {
			break
		}
		i = skipSpace(s, i+1)
		if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
			break
		}
		q := s[i]
		i++
		vs := i
		for i < len(s) && s[i] != q {
			i++
		}
		if i >= len(s) {
			break
		}
		out[name] = s[vs:i]
		i++
	}
	return out
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

func skipSpace(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

func isNameBoundary(c byte) bool {
	return isSpace(c) || c == '>' || c == '/' || c == '='
}
