// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"strings"
	"testing"
)

func TestPretty(t *testing.T) {
	cases := []struct{ in, want string }{
		{
			`<a b="1" c="2"><child>hi</child><!-- cm --><![CDATA[x<y]]><?pi data?></a>`,
			"<a b='1' c='2'>\n  <child>\n    hi\n  </child>\n  <!-- cm -->\n  <![CDATA[x<y]]>\n<?pi data?>\n</a>",
		},
		{`<a><b>text</b><c/></a>`, "<a>\n  <b>\n    text\n  </b>\n  <c/>\n</a>"},
		{`<a>just text</a>`, "<a>\n  just text\n</a>"},
		{`<a>  <b/>  </a>`, "<a>\n  <b/>\n</a>"},
		{`<?xml version="1.0"?><!--top--><a/>`, "<?xml version='1.0'?>\n<!--top-->\n<a/>"},
		{`<a/>`, "<a/>"},
		{`<a c="1" a="2" b="3"/>`, "<a c='1' a='2' b='3'/>"}, // Pretty keeps source order
	}
	for _, c := range cases {
		d, err := ParseDocument(c.in)
		if err != nil {
			t.Fatalf("parse %q: %v", c.in, err)
		}
		if got := d.Pretty(2); got != c.want {
			t.Errorf("Pretty(%q) =\n%q\nwant\n%q", c.in, got, c.want)
		}
	}
}

func TestPrettyIndentWidth(t *testing.T) {
	d, _ := ParseDocument(`<a><b>x</b></a>`)
	if got := PrettyString(d, 4); got != "<a>\n    <b>\n        x\n    </b>\n</a>" {
		t.Errorf("indent 4 = %q", got)
	}
}

func TestPrettyCompact(t *testing.T) {
	d, _ := ParseDocument(`<a>short text</a>`)
	f := &PrettyFormatter{Indentation: 2, Width: 80, Compact: true}
	w := &writer{}
	f.writeDocument(w, d)
	if got := w.b.String(); got != "<a>short text</a>" {
		t.Errorf("compact short = %q", got)
	}
	// A text-only element wider than Width is not compacted.
	long := strings.Repeat("word ", 30)
	d2, _ := ParseDocument("<a>" + long + "</a>")
	f2 := &PrettyFormatter{Indentation: 2, Width: 80, Compact: true}
	w2 := &writer{}
	f2.writeDocument(w2, d2)
	if !strings.Contains(w2.b.String(), "\n") {
		t.Error("long compact text should wrap")
	}
	// A mixed-content element is never compacted.
	d3, _ := ParseDocument(`<a>t<b/></a>`)
	f3 := &PrettyFormatter{Indentation: 2, Width: 80, Compact: true}
	w3 := &writer{}
	f3.writeDocument(w3, d3)
	if !strings.Contains(w3.b.String(), "\n  <b/>") {
		t.Errorf("mixed compact = %q", w3.b.String())
	}
}

func TestWriteOptions(t *testing.T) {
	d, _ := ParseDocument(`<a><b>x</b></a>`)
	if got := d.Write(WriteOptions{}); got != `<a><b>x</b></a>` {
		t.Errorf("default Write = %q", got)
	}
	if got := d.Write(WriteOptions{Pretty: true}); !strings.Contains(got, "\n  <b>") {
		t.Errorf("pretty Write = %q", got)
	}
	if got := d.Write(WriteOptions{Indent: 4}); !strings.Contains(got, "\n    <b>") {
		t.Errorf("indent Write = %q", got)
	}
}

func TestTextWrapping(t *testing.T) {
	// wrapText breaks at the last space before width.
	if got := wrapText("aaa bbb ccc", 7); got != "aaa bbb\nccc" {
		t.Errorf("wrap = %q", got)
	}
	// No space within width: emit as-is.
	if got := wrapText("aaaaaaaaaa", 4); got != "aaaaaaaaaa" {
		t.Errorf("wrap nospace = %q", got)
	}
	if got := wrapText("abc", 0); got != "abc" {
		t.Errorf("wrap zero width = %q", got)
	}
}

func TestSqueezeAndReplace(t *testing.T) {
	if got := replaceWhitespace("a\tb\nc\rd"); got != "a b c d" {
		t.Errorf("replaceWhitespace = %q", got)
	}
	if got := squeezeSpaces("a   b  c"); got != "a b c" {
		t.Errorf("squeeze = %q", got)
	}
	if got := indentText("a\nb", 2); got != "a\n  b" {
		t.Errorf("indentText = %q", got)
	}
	if got := indentText("a\nb", -1); got != "a\nb" {
		t.Errorf("indentText neg = %q", got)
	}
	if got := indent(0); got != "" {
		t.Errorf("indent 0 = %q", got)
	}
}

func TestPrettyDocFirstChildSpacing(t *testing.T) {
	// Leading whitespace text at the doc level is skipped, and the XMLDecl/root
	// spacing matches REXML.
	d, _ := ParseDocument(`  <a/>`)
	if got := d.Pretty(2); got != "<a/>" {
		t.Errorf("pretty leading ws = %q", got)
	}
}
