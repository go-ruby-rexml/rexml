// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "testing"

const xpathDoc = `<root>` +
	`<a id="1"><b>B1</b></a>` +
	`<a id="2" flag="on"><b>B2</b></a>` +
	`<c><a id="3"><b>B3</b></a></c>` +
	`</root>`

// names renders a node set to comparable strings for assertions.
func names(ns []Node) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		switch v := n.(type) {
		case *Element:
			out = append(out, v.QName())
		case *Text:
			out = append(out, v.Val())
		case *attrResult:
			out = append(out, AttrValue(v))
		}
	}
	return out
}

func eqStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestXPathMatch(t *testing.T) {
	d, _ := ParseDocument(xpathDoc)
	cases := []struct {
		path string
		want []string
	}{
		{"/root/a", []string{"a", "a"}},
		{"//a", []string{"a", "a", "a"}},
		{"//b", []string{"b", "b", "b"}},
		{"//a/@id", []string{"1", "2", "3"}},
		{"//a/@*", []string{"1", "2", "on", "3"}},
		{`//a[@id="2"]`, []string{"a"}},
		{`//a[@flag='on']`, []string{"a"}},
		{"//a[@flag]", []string{"a"}},
		{"//a[2]", []string{"a"}},
		{"//a[9]", nil},
		{"//a/b/text()", []string{"B1", "B2", "B3"}},
		{"/root/*", []string{"a", "a", "c"}},
		{"/root/c/a", []string{"a"}},
		{"root/a", []string{"a", "a"}}, // relative from doc context's root? relative uses ctx
		{"//missing", nil},
		{"", nil},
		{"   ", nil},
	}
	for _, c := range cases {
		got := names(XPathMatch(d, c.path))
		if !eqStrs(got, c.want) {
			t.Errorf("XPathMatch(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestXPathRelative(t *testing.T) {
	d, _ := ParseDocument(xpathDoc)
	root := d.Root()
	got := names(XPathMatch(root, "a/b"))
	if !eqStrs(got, []string{"b", "b"}) {
		t.Errorf("relative a/b = %v", got)
	}
	// descendant from an element context.
	got = names(XPathMatch(root, "c//b"))
	if !eqStrs(got, []string{"b"}) {
		t.Errorf("c//b = %v", got)
	}
}

func TestXPathFirstEach(t *testing.T) {
	d, _ := ParseDocument(xpathDoc)
	first := XPathFirst(d, "//b")
	if first == nil || first.(*Element).Text() != "B1" {
		t.Errorf("first = %v", first)
	}
	if XPathFirst(d, "//zzz") != nil {
		t.Error("first missing should be nil")
	}
	var count int
	XPathEach(d, "//a", func(Node) { count++ })
	if count != 3 {
		t.Errorf("each count = %d", count)
	}
}

func TestXPathAttrValueHelper(t *testing.T) {
	d, _ := ParseDocument(`<a id="x&amp;y"/>`)
	n := XPathFirst(d, "/a/@id")
	if AttrValue(n) != "x&y" {
		t.Errorf("AttrValue decoded = %q", AttrValue(n))
	}
	// AttrValue on a non-attr node returns "".
	if AttrValue(d.Root()) != "" {
		t.Error("AttrValue non-attr")
	}
}

func TestXPathOnElementContextAttr(t *testing.T) {
	d, _ := ParseDocument(`<a id="7"/>`)
	got := names(XPathMatch(d.Root(), "@id"))
	if !eqStrs(got, []string{"7"}) {
		t.Errorf("@id on context = %v", got)
	}
	got = names(XPathMatch(d.Root(), "text()"))
	if len(got) != 0 {
		t.Errorf("text() empty = %v", got)
	}
}

func TestRootElementFromDetached(t *testing.T) {
	root := NewElement("r")
	child := root.AddElement("c")
	if rootElement(child) != root {
		t.Error("rootElement from child")
	}
	// rootElement of a Text node (unsupported) is nil.
	if rootElement(NewText("x")) != nil {
		t.Error("rootElement of text should be nil")
	}
}

func TestXPathAbsoluteNoRoot(t *testing.T) {
	if XPathMatch(NewText("x"), "/a") != nil {
		t.Error("absolute path with no document root should be nil")
	}
}

func TestXPathDescendantText(t *testing.T) {
	d, _ := ParseDocument(`<a>top<b>nested</b></a>`)
	got := names(XPathMatch(d, "//text()"))
	// //text() from root matches the root's own text node.
	if len(got) == 0 {
		t.Errorf("//text() = %v", got)
	}
}
