// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "testing"

func TestXPathSeedAttr(t *testing.T) {
	d, _ := ParseDocument(`<r><a id="1"/><a id="2"/></r>`)
	got := names(XPathMatch(d, "//@id"))
	if !eqStrs(got, []string{"1", "2"}) {
		t.Errorf("//@id = %v", got)
	}
	// /@id seeds the root's own attribute.
	d2, _ := ParseDocument(`<r id="9"/>`)
	if got := names(XPathMatch(d2, "/@id")); !eqStrs(got, []string{"9"}) {
		t.Errorf("/@id = %v", got)
	}
}

func TestXPathStepAfterNonElement(t *testing.T) {
	d, _ := ParseDocument(`<a>txt<b>x</b></a>`)
	// A text() step yields Text nodes; a following element step iterates them
	// and skips the non-elements, yielding nothing.
	if got := XPathMatch(d, "//text()/c"); got != nil {
		t.Errorf("text()/c = %v", names(got))
	}
	// @attr after a text-node set also yields nothing.
	if got := XPathMatch(d, "//text()/@x"); got != nil {
		t.Errorf("text()/@x = %v", names(got))
	}
	// text() applied to a text-node set yields nothing.
	if got := XPathMatch(d, "//text()/text()"); got != nil {
		t.Errorf("text()/text() = %v", names(got))
	}
}

func TestXPathUnknownPredicateKeepsSet(t *testing.T) {
	// A predicate that is neither positional nor an attribute test leaves the
	// element set unchanged (the documented boundary).
	d, _ := ParseDocument(`<r><a/><a/></r>`)
	got := names(XPathMatch(d, "//a[self]"))
	if !eqStrs(got, []string{"a", "a"}) {
		t.Errorf("//a[self] = %v", got)
	}
}

func TestParseMoreErrors(t *testing.T) {
	for _, b := range []string{`</a`, `<a ="x"/>`} {
		if _, err := ParseDocument(b); err == nil {
			t.Errorf("ParseDocument(%q) expected error", b)
		}
	}
}

func TestParsePseudoAttrNameThenSpace(t *testing.T) {
	// A pseudo-attr name followed only by spaces (no '=') terminates parsing.
	got := parsePseudoAttrs("version  ")
	if len(got) != 0 {
		t.Errorf("name then space = %v", got)
	}
	// Trailing whitespace after a complete pair breaks the loop on the next
	// iteration's leading skipSpace.
	got = parsePseudoAttrs(`version="1.0"   `)
	if got["version"] != "1.0" || len(got) != 1 {
		t.Errorf("trailing space pair = %v", got)
	}
}
