// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` once. The oracle tests skip when it is absent
// (the qemu cross-arch lanes and the Windows lane), so the deterministic suite
// alone drives the 100% gate there. The oracle additionally requires Ruby 4.0+.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI REXML oracle")
	}
	out, err := exec.Command(path, "-e", `print(RUBY_VERSION >= "4.0")`).Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		t.Skipf("ruby %s; oracle gated to >= 4.0", strings.TrimSpace(string(out)))
	}
	return path
}

// rubyREXML runs a REXML script and returns its stdout. The script reads the
// input document from stdin in binary mode and $stdout.binmode itself, so
// Windows text-mode never pollutes the XML bytes (the go-ruby-erb lesson).
func rubyREXML(t *testing.T, bin, doc, script string) string {
	t.Helper()
	full := "$stdout.binmode\n$stdin.binmode\nxml = $stdin.read\n" +
		"require 'rexml/document'\nrequire 'rexml/formatters/pretty'\n" +
		"d = REXML::Document.new(xml)\n" + script
	cmd := exec.Command(bin, "-e", full)
	cmd.Stdin = strings.NewReader(doc)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// corpus is the shared set of documents exercised by every oracle direction.
var corpus = []string{
	`<a/>`,
	`<a></a>`,
	`<a x="1"/>`,
	`<a b="1" c="2"><child>hi</child></a>`,
	`<a c="3" b="2" a="1"/>`,
	`<root><a id="1">one</a><a id="2">two</a><c/></root>`,
	`<a>x &lt; y &amp; z &gt; "q"</a>`,
	`<a b="x &lt; &amp; &gt; &quot; &apos;"/>`,
	`<a>&#65; &#x42;</a>`,
	`<a>1 > 0</a>`,
	`<a><!-- a comment --></a>`,
	`<a><![CDATA[<raw> & data]]></a>`,
	`<a><?target some data?></a>`,
	`<x:a xmlns:x="urn:X"><x:b id="9"/></x:a>`,
	`<a xmlns="d" xmlns:p="u" id="1" z:x="2" xmlns:z="w"/>`,
	`<?xml version="1.0" encoding="UTF-8"?><a/>`,
	`<?xml version="1.0" standalone="yes"?>` + "\n" + `<doc><p>text</p></doc>`,
	`<!DOCTYPE root SYSTEM "x.dtd"><a/>`,
	`<a><b><c><d>deep</d></c></b></a>`,
	`<a>  spaced text  </a>`,
	`<config><server name="web" port="8080"><option>on</option></server></config>`,
}

// TestOracleRoundTrip checks parse→serialise (compact to_s) matches MRI byte-
// for-byte across the corpus.
func TestOracleRoundTrip(t *testing.T) {
	bin := rubyBin(t)
	for _, doc := range corpus {
		d, err := ParseDocument(doc)
		if err != nil {
			t.Errorf("ParseDocument(%q): %v", doc, err)
			continue
		}
		want := rubyREXML(t, bin, doc, "print d.to_s")
		if got := d.ToString(); got != want {
			t.Errorf("to_s mismatch for %q:\n go = %q\nmri = %q", doc, got, want)
		}
	}
}

// TestOraclePretty checks the Pretty formatter (indent 2) matches MRI across the
// corpus.
func TestOraclePretty(t *testing.T) {
	bin := rubyBin(t)
	script := "out = +''\nREXML::Formatters::Pretty.new(2).write(d, out)\nprint out"
	for _, doc := range corpus {
		d, err := ParseDocument(doc)
		if err != nil {
			t.Errorf("ParseDocument(%q): %v", doc, err)
			continue
		}
		want := rubyREXML(t, bin, doc, script)
		if got := d.Pretty(2); got != want {
			t.Errorf("pretty mismatch for %q:\n go = %q\nmri = %q", doc, got, want)
		}
	}
}

// TestOraclePrettyIndent4 checks a non-default indent width matches MRI.
func TestOraclePrettyIndent4(t *testing.T) {
	bin := rubyBin(t)
	doc := `<a><b><c>x</c></b></a>`
	script := "out = +''\nREXML::Formatters::Pretty.new(4).write(d, out)\nprint out"
	want := rubyREXML(t, bin, doc, script)
	d, _ := ParseDocument(doc)
	if got := d.Pretty(4); got != want {
		t.Errorf("pretty(4):\n go = %q\nmri = %q", got, want)
	}
}

// TestOracleXPath checks the XPath subset matches MRI's REXML::XPath.match for a
// representative set of expressions against one rich document.
func TestOracleXPath(t *testing.T) {
	bin := rubyBin(t)
	doc := `<root>` +
		`<a id="1" k="x"><b>B1</b></a>` +
		`<a id="2"><b>B2</b></a>` +
		`<c><a id="3"><b>B3</b></a></c>` +
		`</root>`
	paths := []string{
		"/root/a",
		"//a",
		"//b",
		"//a/@id",
		`//a[@id="2"]`,
		"//a[@k]",
		"//a[1]",
		"//a[2]",
		"//a/b/text()",
		"/root/*",
		"/root/c/a",
	}
	d, _ := ParseDocument(doc)
	for _, p := range paths {
		// MRI prints one matched value per line; render elements by name,
		// attributes and text by value.
		script := "REXML::XPath.match(d, " + rubyQuote(p) + ").each do |n|\n" +
			"  if n.is_a?(REXML::Element) then puts n.name\n" +
			"  elsif n.is_a?(REXML::Attribute) then puts n.value\n" +
			"  else puts n.value end\nend"
		want := rubyREXML(t, bin, doc, script)
		got := strings.Join(append(xpathRender(XPathMatch(d, p)), ""), "\n")
		if normLines(got) != normLines(want) {
			t.Errorf("xpath %q:\n go = %q\nmri = %q", p, got, want)
		}
	}
}

// xpathRender renders a node set the same way the oracle Ruby script prints it.
func xpathRender(ns []Node) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		switch v := n.(type) {
		case *Element:
			out = append(out, v.Name)
		case *Text:
			out = append(out, v.Val())
		case *attrResult:
			out = append(out, AttrValue(v))
		}
	}
	return out
}

func normLines(s string) string { return strings.TrimRight(s, "\n") }

// rubyQuote produces a single-quoted Ruby string literal for path.
func rubyQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}
