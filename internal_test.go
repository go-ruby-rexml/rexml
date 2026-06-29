// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "testing"

func TestUnescapeEdges(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a&amp;b", "a&b"},
		{"&lt;&gt;&quot;&apos;", `<>"'`},
		{"&#65;", "A"},
		{"&#x42;", "B"},
		{"&#X43;", "C"},
		{"&unknown;", "&unknown;"},   // unknown entity kept
		{"&nosemi", "&nosemi"},       // no terminating semicolon
		{"trailing&", "trailing&"},   // bare ampersand at end
		{"&#;", "&#;"},               // empty numeric ref
		{"&#xZZ;", "&#xZZ;"},         // bad hex digits
		{"&#99x;", "&#99x;"},         // bad decimal digits
		{"&#x110000;", "&#x110000;"}, // out of Unicode range
		{"&;", "&;"},                 // empty entity name
	}
	for _, c := range cases {
		if got := unescape(c.in); got != c.want {
			t.Errorf("unescape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeAndApos(t *testing.T) {
	if got := normalize("plain"); got != "plain" {
		t.Errorf("normalize plain = %q", got)
	}
	if got := normalize(`<>&"'`); got != "&lt;&gt;&amp;&quot;&apos;" {
		t.Errorf("normalize = %q", got)
	}
	if got := escapeApos("no quote"); got != "no quote" {
		t.Errorf("escapeApos none = %q", got)
	}
	if got := escapeApos("a'b'c"); got != "a&apos;b&apos;c" {
		t.Errorf("escapeApos = %q", got)
	}
}

func TestParseDecHex(t *testing.T) {
	if v, ok := parseDec("123"); !ok || v != 123 {
		t.Errorf("parseDec = %d,%v", v, ok)
	}
	if _, ok := parseDec(""); ok {
		t.Error("parseDec empty")
	}
	if _, ok := parseDec("12a"); ok {
		t.Error("parseDec bad")
	}
	if v, ok := parseHex("1f"); !ok || v != 31 {
		t.Errorf("parseHex = %d,%v", v, ok)
	}
	if v, ok := parseHex("AB"); !ok || v != 171 {
		t.Errorf("parseHex upper = %d,%v", v, ok)
	}
	if _, ok := parseHex(""); ok {
		t.Error("parseHex empty")
	}
	if _, ok := parseHex("xy"); ok {
		t.Error("parseHex bad")
	}
}

func TestStripQuotesUnquoted(t *testing.T) {
	if got := stripQuotes("bare"); got != "bare" {
		t.Errorf("stripQuotes bare = %q", got)
	}
	if got := stripQuotes(`"q"`); got != "q" {
		t.Errorf("stripQuotes dq = %q", got)
	}
	if got := stripQuotes("'q'"); got != "q" {
		t.Errorf("stripQuotes sq = %q", got)
	}
	if got := stripQuotes("x"); got != "x" {
		t.Errorf("stripQuotes single char = %q", got)
	}
}

func TestAttrResultWriteTo(t *testing.T) {
	d, _ := ParseDocument(`<a id="v&amp;w"/>`)
	n := XPathFirst(d, "/a/@id").(*attrResult)
	w := &writer{}
	n.writeTo(w)
	if w.b.String() != "v&w" {
		t.Errorf("attrResult writeTo = %q", w.b.String())
	}
	if n.parent() != nil {
		t.Error("attrResult parent")
	}
}

func TestDocumentSetParentNoop(t *testing.T) {
	var n Node = NewDocument()
	n.setParent(NewElement("x")) // must not panic; documents ignore it
	if n.parent() != nil {
		t.Error("document parent stays nil")
	}
}

func TestParseSelfClosingDescend(t *testing.T) {
	// A self-closing element must not capture following siblings as children.
	if got := roundTrip(t, `<r><a/><b/></r>`); got != `<r><a/><b/></r>` {
		t.Errorf("self-closing siblings = %q", got)
	}
}

func TestReadStartTagSpacing(t *testing.T) {
	// Attributes with assorted spacing around '='.
	if got := roundTrip(t, `<a x = "1"  y="2" />`); got != `<a x='1' y='2'/>` {
		t.Errorf("spaced attrs = %q", got)
	}
}

func TestParsePseudoAttrsEdges(t *testing.T) {
	got := parsePseudoAttrs(`version="1.0"   encoding='UTF-8'`)
	if got["version"] != "1.0" || got["encoding"] != "UTF-8" {
		t.Errorf("pseudo attrs = %v", got)
	}
	// A trailing bare name without '=' is ignored.
	if got := parsePseudoAttrs(`version="1.0" dangling`); got["version"] != "1.0" {
		t.Errorf("dangling = %v", got)
	}
	if got := parsePseudoAttrs(``); len(got) != 0 {
		t.Errorf("empty = %v", got)
	}
	if got := parsePseudoAttrs(`x`); len(got) != 0 {
		t.Errorf("name only = %v", got)
	}
	if got := parsePseudoAttrs(`x=`); len(got) != 0 {
		t.Errorf("no value = %v", got)
	}
	if got := parsePseudoAttrs(`x="unterminated`); len(got) != 0 {
		t.Errorf("unterminated = %v", got)
	}
	if got := parsePseudoAttrs(`x=noquote`); len(got) != 0 {
		t.Errorf("noquote = %v", got)
	}
}
