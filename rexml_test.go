// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"strings"
	"testing"
)

// roundTrip parses xml and returns its default serialisation.
func roundTrip(t *testing.T, xml string) string {
	t.Helper()
	d, err := ParseDocument(xml)
	if err != nil {
		t.Fatalf("ParseDocument(%q): %v", xml, err)
	}
	return d.ToString()
}

func TestRoundTripCompact(t *testing.T) {
	cases := []struct{ in, want string }{
		{`<a/>`, `<a/>`},
		{`<a></a>`, `<a/>`},
		{`<a x="1"></a>`, `<a x='1'/>`},
		{`<a><b/></a>`, `<a><b/></a>`},
		{`<a b="1" c="2"/>`, `<a b='1' c='2'/>`},
		{`<a c="1" b="2" a="3"/>`, `<a a='3' b='2' c='1'/>`}, // sort by local name
		{`<a>text</a>`, `<a>text</a>`},
		{`<a>x &lt; y &amp; z &gt; "q"</a>`, `<a>x &lt; y &amp; z &gt; "q"</a>`},
		{`<a b="x &lt; &amp; &gt; &quot; &apos;"/>`, `<a b='x &lt; &amp; &gt; &quot; &apos;'/>`},
		{`<a b='has " quote'/>`, `<a b='has " quote'/>`}, // raw " preserved
		{`<a>&#65; &#x42;</a>`, `<a>&#65; &#x42;</a>`},
		{`<a>1 > 0</a>`, `<a>1 > 0</a>`}, // bare > preserved raw
		{`<a><!-- cm --></a>`, `<a><!-- cm --></a>`},
		{`<a><![CDATA[x<y&z]]></a>`, `<a><![CDATA[x<y&z]]></a>`},
		{`<a><?pi some data?></a>`, `<a><?pi some data?></a>`},
		{`<a><?pi?></a>`, `<a><?pi?></a>`},
		{`<x:a xmlns:x="urn:X"><x:b/></x:a>`, `<x:a xmlns:x='urn:X'><x:b/></x:a>`},
		{`<?xml version="1.0" encoding="UTF-8"?><a/>`, `<?xml version='1.0' encoding='UTF-8'?><a/>`},
		{`<?xml version="1.0" standalone="yes"?>` + "\n" + `<a/>`,
			`<?xml version='1.0' standalone='yes'?>` + "\n" + `<a/>`},
		{`<?xml?><a/>`, `<?xml version='1.0'?><a/>`},
		{`  <a/>`, `  <a/>`},
		{`<a>  spaced  </a>`, `<a>  spaced  </a>`},
		{`<a xmlns="d" xmlns:p="u" id="1" z:x="2" xmlns:z="w"/>`,
			`<a id='1' xmlns:p='u' z:x='2' xmlns='d' xmlns:z='w'/>`},
	}
	for _, c := range cases {
		if got := roundTrip(t, c.in); got != c.want {
			t.Errorf("roundTrip(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDocType(t *testing.T) {
	if got := roundTrip(t, `<!DOCTYPE root SYSTEM "x.dtd"><a/>`); got != `<!DOCTYPE root SYSTEM "x.dtd"><a/>` {
		t.Errorf("doctype system: %q", got)
	}
	if got := roundTrip(t, `<!DOCTYPE a [<!ELEMENT a EMPTY>]><a/>`); got != `<!DOCTYPE a [<!ELEMENT a EMPTY>]><a/>` {
		t.Errorf("doctype subset: %q", got)
	}
	d, _ := ParseDocument(`<!DOCTYPE root SYSTEM "x.dtd"><a/>`)
	if d.DocType() == nil || d.DocType().Body != `root SYSTEM "x.dtd"` {
		t.Errorf("DocType() = %+v", d.DocType())
	}
	if NewDocument().DocType() != nil {
		t.Error("empty doc DocType should be nil")
	}
}

func TestXMLDeclAccessors(t *testing.T) {
	d, _ := ParseDocument(`<?xml version="1.1" encoding="UTF-16" standalone="no"?><a/>`)
	x := d.XMLDecl()
	if x == nil || x.Version != "1.1" || x.Encoding != "UTF-16" || x.Standalone != "no" {
		t.Fatalf("XMLDecl = %+v", x)
	}
	if NewDocument().XMLDecl() != nil {
		t.Error("empty doc XMLDecl should be nil")
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		``,
		`<a>`,
		`</a>`,
		`<a></b>`,
		`<!-- unterminated`,
		`<![CDATA[ unterminated`,
		`<?pi unterminated`,
		`<?xml version="1.0"`,
		`<!DOCTYPE a`,
		`<a x>`,
		`<a x=>`,
		`<a x="unterminated`,
		`<a `,
		`<>`,
		`</ >`,
		`<a x=y/>`,
		`<a /x>`,
		`text only`,
		`<a></a>extra`,
		`x<a/>`,
	}
	for _, b := range bad {
		if _, err := ParseDocument(b); err == nil {
			t.Errorf("ParseDocument(%q) expected error", b)
		} else if !strings.Contains(err.Error(), "parse error") {
			t.Errorf("ParseDocument(%q) error = %v", b, err)
		}
	}
}

func TestParseErrorType(t *testing.T) {
	_, err := ParseDocument(`<a>`)
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if pe.Offset < 0 || !strings.Contains(pe.Error(), "offset") {
		t.Errorf("ParseError = %+v", pe)
	}
}

func TestEpilogueWhitespaceDropped(t *testing.T) {
	// trailing whitespace after the root is dropped, matching REXML.
	if got := roundTrip(t, `<a/>  `); got != `<a/>` {
		t.Errorf("epilogue = %q, want %q", got, `<a/>`)
	}
	// a top-level comment after the root is preserved.
	if got := roundTrip(t, `<a/><!--end-->`); got != `<a/><!--end-->` {
		t.Errorf("trailing comment = %q", got)
	}
}
