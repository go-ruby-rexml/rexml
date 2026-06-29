// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "strings"

// writer accumulates serialised output and emits REXML's single-quoted
// attribute style.
type writer struct {
	b strings.Builder
}

func (w *writer) str(s string) { w.b.WriteString(s) }

// quotedAttr writes an attribute in single quotes. A raw (parsed) value is
// emitted verbatim apart from escaping any literal single quote; an
// unnormalized (programmatic) value is fully normalized first.
func (w *writer) quotedAttr(a *Attribute) {
	v := a.Value
	if a.Raw {
		v = escapeApos(v)
	} else {
		v = normalize(v)
	}
	w.b.WriteByte('\'')
	w.b.WriteString(v)
	w.b.WriteByte('\'')
}

// quoted writes a declaration pseudo-attribute value (XML/PI decl) in single
// quotes verbatim — REXML emits these without entity substitution.
func (w *writer) quoted(v string) {
	w.b.WriteByte('\'')
	w.b.WriteString(v)
	w.b.WriteByte('\'')
}

// normalize escapes an unnormalized string to REXML's normalized form,
// replacing & < > " ' with their predefined entities (REXML's Text::normalize
// against the default entity set). This is what programmatically-added text and
// attribute values pass through on serialisation.
func normalize(s string) string {
	if !strings.ContainsAny(s, "&<>\"'") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeText is normalize for an element-body text node.
func escapeText(s string) string { return normalize(s) }

// escapeApos escapes any literal single quote so a raw (already-normalized)
// attribute value is safe inside the single-quoted output form — REXML's
// Attribute#to_string gsub("'", "&apos;").
func escapeApos(s string) string {
	if !strings.ContainsRune(s, '\'') {
		return s
	}
	return strings.ReplaceAll(s, "'", "&apos;")
}

// unescape decodes the five predefined entities and numeric character
// references that REXML's Text#value resolves. Unknown entities are left as-is,
// matching REXML's lenient default behaviour.
func unescape(s string) string {
	if !strings.ContainsRune(s, '&') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		semi := strings.IndexByte(s[i:], ';')
		if semi < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		ent := s[i+1 : i+semi]
		if rep, ok := decodeEntity(ent); ok {
			b.WriteString(rep)
			i += semi + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func decodeEntity(ent string) (string, bool) {
	switch ent {
	case "amp":
		return "&", true
	case "lt":
		return "<", true
	case "gt":
		return ">", true
	case "quot":
		return "\"", true
	case "apos":
		return "'", true
	}
	if len(ent) >= 2 && ent[0] == '#' {
		var cp int64
		var ok bool
		if ent[1] == 'x' || ent[1] == 'X' {
			cp, ok = parseHex(ent[2:])
		} else {
			cp, ok = parseDec(ent[1:])
		}
		if ok && cp >= 0 && cp <= 0x10FFFF {
			return string(rune(cp)), true
		}
	}
	return "", false
}

func parseDec(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	var n int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return n, true
}

func parseHex(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	var n int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			n = n*16 + int64(c-'0')
		case c >= 'a' && c <= 'f':
			n = n*16 + int64(c-'a'+10)
		case c >= 'A' && c <= 'F':
			n = n*16 + int64(c-'A'+10)
		default:
			return 0, false
		}
	}
	return n, true
}
