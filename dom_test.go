// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import "testing"

func TestBuildTree(t *testing.T) {
	doc := NewDocument()
	root := doc.AddElement("root")
	if doc.Root() != root {
		t.Fatal("Root mismatch")
	}
	if root.Parent() != nil {
		t.Error("root parent should be nil")
	}
	a := root.AddElement("a")
	a.AddAttribute("id", "1")
	a.AddText("hello")
	root.AddElement("b")

	if got := doc.ToString(); got != `<root><a id='1'>hello</a><b/></root>` {
		t.Errorf("built tree = %q", got)
	}
	if a.Parent() != root {
		t.Error("a.Parent")
	}
	if v, ok := a.Attr("id"); !ok || v != "1" {
		t.Errorf("Attr id = %q,%v", v, ok)
	}
	if _, ok := a.Attr("missing"); ok {
		t.Error("missing attr reported present")
	}
}

func TestStringAliases(t *testing.T) {
	d, _ := ParseDocument(`<a x="1">t</a>`)
	if d.String() != d.ToString() {
		t.Error("String != ToString")
	}
	d2, err := Parse(`<a/>`)
	if err != nil || d2.Root().Name != "a" {
		t.Errorf("Parse alias = %v %v", d2, err)
	}
	if VERSION != "3.4.4" {
		t.Errorf("VERSION = %q", VERSION)
	}
}

func TestTextAccessors(t *testing.T) {
	d, _ := ParseDocument(`<a>raw &amp; text &#65;</a>`)
	root := d.Root()
	if !root.HasText() {
		t.Fatal("HasText false")
	}
	if got := root.Text(); got != "raw & text A" {
		t.Errorf("Text() = %q", got)
	}
	gt := root.GetText()
	if gt == nil || gt.String() != `raw &amp; text &#65;` {
		t.Errorf("GetText().String() = %q", gt.String())
	}
	if gt.Val() != "raw & text A" {
		t.Errorf("Val() = %q", gt.Val())
	}
	if len(root.Texts()) != 1 {
		t.Errorf("Texts len = %d", len(root.Texts()))
	}
	// No-text element.
	empty := NewElement("x")
	if empty.HasText() || empty.Text() != "" || empty.GetText() != nil {
		t.Error("empty element text accessors")
	}
}

func TestNewTextAndVal(t *testing.T) {
	tx := NewText("a < b & c")
	if tx.String() != "a &lt; b &amp; c" {
		t.Errorf("NewText String = %q", tx.String())
	}
	if tx.Val() != "a < b & c" {
		t.Errorf("NewText Val = %q", tx.Val())
	}
}

func TestAddTextCoalesce(t *testing.T) {
	e := NewElement("a")
	e.AddText("foo")
	e.AddText("bar")
	if len(e.Children) != 1 {
		t.Fatalf("expected coalesced text, got %d children", len(e.Children))
	}
	if e.Text() != "foobar" {
		t.Errorf("coalesced text = %q", e.Text())
	}
	// A non-text child breaks coalescing.
	e.AddElement("b")
	e.AddText("baz")
	if len(e.Children) != 3 {
		t.Errorf("children = %d", len(e.Children))
	}
}

func TestSetText(t *testing.T) {
	e := NewElement("a")
	e.SetText("first") // inserts
	if e.Text() != "first" {
		t.Errorf("after insert = %q", e.Text())
	}
	e.SetText("second") // replaces existing
	if e.Text() != "second" || len(e.Texts()) != 1 {
		t.Errorf("after replace = %q (%d texts)", e.Text(), len(e.Texts()))
	}
	// SetText on an element whose first child is an element inserts at front.
	e2 := NewElement("a")
	e2.AddElement("b")
	e2.SetText("x")
	if e2.Children[0].(*Text).Value != "x" {
		t.Error("SetText should prepend when no text child exists")
	}
}

func TestExpandedNameAndNamespace(t *testing.T) {
	d, _ := ParseDocument(`<x:a xmlns:x="urn:X" xmlns="urn:D"><x:b/><c/></x:a>`)
	root := d.Root()
	if root.ExpandedName() != "x:a" || root.QName() != "x:a" {
		t.Errorf("expanded = %q", root.ExpandedName())
	}
	if root.NamespaceURI() != "urn:X" {
		t.Errorf("root ns = %q", root.NamespaceURI())
	}
	kids := root.ChildElements()
	if kids[0].NamespaceURI() != "urn:X" {
		t.Errorf("x:b ns = %q", kids[0].NamespaceURI())
	}
	if kids[1].NamespaceURI() != "urn:D" { // default namespace
		t.Errorf("c ns = %q", kids[1].NamespaceURI())
	}
	// An element with no namespace in scope resolves to "".
	plain := NewElement("z")
	if plain.NamespaceURI() != "" {
		t.Errorf("plain ns = %q", plain.NamespaceURI())
	}
	if plain.QName() != "z" {
		t.Errorf("plain QName = %q", plain.QName())
	}
}

func TestRootNodeAndDocAdd(t *testing.T) {
	d := NewDocument()
	if d.RootNode() != d {
		t.Error("RootNode")
	}
	if d.parent() != nil {
		t.Error("Document parent")
	}
	d.setParent(NewElement("ignored")) // no-op
	c := &Comment{Value: "c"}
	d.Add(c)
	if c.parent() != nil {
		t.Error("doc child element parent should be nil")
	}
}

func TestElementsAndEach(t *testing.T) {
	d, _ := ParseDocument(`<root><a>1</a><a>2</a><b/></root>`)
	root := d.Root()
	if root.ElementAt(1).Text() != "1" || root.ElementAt(2).Text() != "2" {
		t.Error("ElementAt")
	}
	if root.ElementAt(0) != nil || root.ElementAt(99) != nil {
		t.Error("ElementAt out of range")
	}
	as := root.Elements("a")
	if len(as) != 2 {
		t.Errorf("Elements(a) = %d", len(as))
	}
	if root.FirstElement("a").Text() != "1" {
		t.Error("FirstElement")
	}
	if root.FirstElement("zzz") != nil {
		t.Error("FirstElement missing")
	}
	var names []string
	root.EachElement(func(e *Element) { names = append(names, e.QName()) })
	if len(names) != 3 {
		t.Errorf("EachElement = %v", names)
	}
}

func TestAttributesMap(t *testing.T) {
	at := newAttributes()
	at.Set("a", "1")
	at.Set("b", "2")
	at.Set("a", "3") // replace
	if at.Len() != 2 {
		t.Errorf("Len = %d", at.Len())
	}
	if v, _ := at.Get("a"); v != "3" {
		t.Errorf("Get a = %q", v)
	}
	if _, ok := at.Get("zzz"); ok {
		t.Error("Get missing")
	}
	if at.GetAttr("zzz") != nil {
		t.Error("GetAttr missing")
	}
	at.Delete("a")
	at.Delete("missing") // no-op
	if at.Len() != 1 {
		t.Errorf("after delete Len = %d", at.Len())
	}
	var seen []string
	at.Each(func(a *Attribute) { seen = append(seen, a.QName()) })
	if len(seen) != 1 || seen[0] != "b" {
		t.Errorf("Each = %v", seen)
	}
	// QName with prefix.
	at.Set("ns:x", "v")
	if at.GetAttr("ns:x").QName() != "ns:x" {
		t.Error("prefixed QName")
	}
}

func TestProgrammaticEscaping(t *testing.T) {
	e := NewElement("a")
	e.AddAttribute("x", `a<b&c'"d>e`)
	if got := e.ToStringElem(); got != `<a x='a&lt;b&amp;c&apos;&quot;d&gt;e'/>` {
		t.Errorf("programmatic attr = %q", got)
	}
}

// ToStringElem renders a detached element via the document writer (test helper).
func (e *Element) ToStringElem() string {
	w := &writer{}
	e.writeTo(w)
	return w.b.String()
}

func TestLeafNodesWrite(t *testing.T) {
	cases := []struct {
		n    Node
		want string
	}{
		{&Comment{Value: " c "}, "<!-- c -->"},
		{&CData{Value: "x<y"}, "<![CDATA[x<y]]>"},
		{&Instruction{Target: "pi", Content: "data"}, "<?pi data?>"},
		{&Instruction{Target: "pi"}, "<?pi?>"},
		{&DocType{Body: `r SYSTEM "x"`}, `<!DOCTYPE r SYSTEM "x">`},
		{&XMLDecl{Version: "1.0"}, `<?xml version='1.0'?>`},
		{&XMLDecl{Version: "1.0", Encoding: "UTF-8", Standalone: "yes"},
			`<?xml version='1.0' encoding='UTF-8' standalone='yes'?>`},
	}
	for _, c := range cases {
		w := &writer{}
		c.n.writeTo(w)
		if got := w.b.String(); got != c.want {
			t.Errorf("writeTo = %q, want %q", got, c.want)
		}
	}
}
