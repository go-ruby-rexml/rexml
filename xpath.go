// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

import (
	"strconv"
	"strings"
)

// XPath implements the widely-used subset of XPath 1.0 that REXML's XPath.first
// / each / match cover. The supported grammar is documented on Match.
//
// Boundary (out of scope): axes other than child/descendant (no ancestor,
// following-sibling, …), functions beyond text() (no count(), position(),
// last(), name(), contains(), …), arithmetic, and nested boolean predicates
// (and/or). Predicates support [n] (1-based position), [@attr],
// [@attr='value'] / [@attr="value"], and the boolean attribute existence test.
type XPath struct{}

// XPathFirst returns the first node matching path relative to ctx, or nil —
// REXML::XPath.first.
func XPathFirst(ctx Node, path string) Node {
	got := matchXPath(ctx, path)
	if len(got) == 0 {
		return nil
	}
	return got[0]
}

// XPathEach calls fn for every node matching path — REXML::XPath.each.
func XPathEach(ctx Node, path string, fn func(Node)) {
	for _, n := range matchXPath(ctx, path) {
		fn(n)
	}
}

// XPathMatch returns every node matching path — REXML::XPath.match.
func XPathMatch(ctx Node, path string) []Node {
	return matchXPath(ctx, path)
}

// matchPath is the Element#elements[path] entry point: a relative path is
// resolved against e's children, an absolute path against the document root.
func matchPath(e *Element, path string) []Node {
	return matchXPath(e, path)
}

// node matched by XPath: either an *Element, *Text, or *attrResult (for @attr).
//
// attrResult wraps a matched attribute so callers can read its value.
type attrResult struct {
	base
	Owner *Element
	Attr  *Attribute
}

func (a *attrResult) writeTo(w *writer) { w.str(a.Attr.UnescapedValue()) }

// AttrValue returns the decoded value of a matched @attr node, the empty string
// for any other node — convenience for XPath @attr matches.
func AttrValue(n Node) string {
	if a, ok := n.(*attrResult); ok {
		return a.Attr.UnescapedValue()
	}
	return ""
}

func matchXPath(ctx Node, path string) []Node {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	absolute := false
	descendantRoot := false
	switch {
	case strings.HasPrefix(path, "//"):
		absolute, descendantRoot = true, true
		path = path[2:]
	case strings.HasPrefix(path, "/"):
		absolute = true
		path = path[1:]
	}

	// splitSteps always yields at least one step for a non-empty path.
	steps := splitSteps(path)

	var cur []Node
	startIdx := 0
	// An absolute path, or a relative path applied to a Document, seeds the
	// first step against the root element (REXML resolves a document-relative
	// path from the root).
	_, ctxIsDoc := ctx.(*Document)
	if absolute || ctxIsDoc {
		root := rootElement(ctx)
		if root == nil {
			return nil
		}
		cands := []*Element{root}
		if descendantRoot {
			// "//test" matches the root or any descendant.
			cands = append(cands, descendants(root)...)
		}
		cur = seedStep(cands, steps[0])
		startIdx = 1
	} else {
		cur = []Node{ctx}
	}

	for i := startIdx; i < len(steps); i++ {
		st := steps[i]
		if st.empty {
			continue // a "//" separator; its descendant flag rides on the next step
		}
		cur = applyStep(cur, st, st.descendant)
		if len(cur) == 0 {
			return nil
		}
	}
	return cur
}

// seedStep applies the first step against an explicit element candidate set,
// dispatching on the node test like applyStep but without re-walking children.
func seedStep(cands []*Element, st *step) []Node {
	nodes := make([]Node, len(cands))
	for i, e := range cands {
		nodes[i] = e
	}
	if strings.HasPrefix(st.name, "@") {
		return attrNodes(nodes, st.name[1:])
	}
	if st.name == "text()" {
		return textNodes(nodes)
	}
	els := filterByName(cands, st.name)
	els = applyPredicate(els, st.predicate)
	out := make([]Node, len(els))
	for i, e := range els {
		out[i] = e
	}
	return out
}

// step is one location step: a node test plus optional predicate.
type step struct {
	name       string // element name, "*", "@attr", or "text()"
	empty      bool   // produced by an empty path segment ("//")
	descendant bool   // this step searches descendants (followed a "//")
	predicate  *predicate
	next       *step
}

type predicate struct {
	index     int    // 1-based positional predicate, 0 when none
	attr      string // attribute name for [@a] / [@a='v']
	attrValue string // expected value for [@a='v'], "" when existence-only
	hasValue  bool   // distinguishes [@a='']  from [@a]
}

// splitSteps splits a path on '/' that are not inside predicates, parsing each
// step's node test and predicate. Consecutive '//' yields a descendant step.
func splitSteps(path string) []*step {
	var raw []string
	depth := 0
	last := 0
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '[':
			depth++
		case ']':
			depth--
		case '/':
			if depth == 0 {
				raw = append(raw, path[last:i])
				last = i + 1
			}
		}
	}
	raw = append(raw, path[last:])

	var steps []*step
	for _, r := range raw {
		s := parseStep(r)
		if len(steps) > 0 {
			prev := steps[len(steps)-1]
			prev.next = s
			if prev.empty {
				s.descendant = true
			}
		}
		steps = append(steps, s)
	}
	return steps
}

func parseStep(r string) *step {
	s := &step{}
	if r == "" {
		s.empty = true
		return s
	}
	// Split off a predicate [...].
	if i := strings.IndexByte(r, '['); i >= 0 && strings.HasSuffix(r, "]") {
		s.predicate = parsePredicate(r[i+1 : len(r)-1])
		r = r[:i]
	}
	s.name = r
	return s
}

func parsePredicate(p string) *predicate {
	p = strings.TrimSpace(p)
	pr := &predicate{}
	if n, err := strconv.Atoi(p); err == nil {
		pr.index = n
		return pr
	}
	if strings.HasPrefix(p, "@") {
		body := p[1:]
		if eq := strings.IndexByte(body, '='); eq >= 0 {
			pr.attr = strings.TrimSpace(body[:eq])
			pr.attrValue = stripQuotes(strings.TrimSpace(body[eq+1:]))
			pr.hasValue = true
		} else {
			pr.attr = strings.TrimSpace(body)
		}
	}
	return pr
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// applyStep advances the node set by one step. The @attr and text() tests read
// the context element directly; element tests select children (or descendants
// for a "//" step).
func applyStep(cur []Node, st *step, descendant bool) []Node {
	// @attr and text() operate on the context nodes themselves.
	if strings.HasPrefix(st.name, "@") {
		return attrNodes(cur, st.name[1:])
	}
	if st.name == "text()" {
		return textNodes(cur)
	}
	var out []Node
	for _, n := range cur {
		e, ok := n.(*Element)
		if !ok {
			continue
		}
		var cands []*Element
		if descendant {
			cands = descendants(e)
		} else {
			cands = e.ChildElements()
		}
		els := filterByName(cands, st.name)
		els = applyPredicate(els, st.predicate)
		for _, c := range els {
			out = append(out, c)
		}
	}
	return out
}

// attrNodes matches @attr (or @* ) against each context element.
func attrNodes(cur []Node, name string) []Node {
	var out []Node
	for _, n := range cur {
		e, ok := n.(*Element)
		if !ok {
			continue
		}
		if name == "*" {
			e.Attributes.Each(func(a *Attribute) {
				out = append(out, &attrResult{Owner: e, Attr: a})
			})
			continue
		}
		if a := e.Attributes.GetAttr(name); a != nil {
			out = append(out, &attrResult{Owner: e, Attr: a})
		}
	}
	return out
}

func textNodes(cur []Node) []Node {
	var out []Node
	for _, n := range cur {
		e, ok := n.(*Element)
		if !ok {
			continue
		}
		for _, t := range e.Texts() {
			out = append(out, t)
		}
	}
	return out
}

func filterByName(cands []*Element, name string) []*Element {
	if name == "*" {
		return cands
	}
	var out []*Element
	for _, e := range cands {
		if e.QName() == name {
			out = append(out, e)
		}
	}
	return out
}

func applyPredicate(els []*Element, pr *predicate) []*Element {
	if pr == nil {
		return els
	}
	if pr.index > 0 {
		// A positional predicate selects the nth matching element within each
		// parent context, so over a flattened (descendant) set REXML groups by
		// parent and picks the nth child of each group.
		return nthPerParent(els, pr.index)
	}
	if pr.attr != "" {
		var out []*Element
		for _, e := range els {
			a := e.Attributes.GetAttr(pr.attr)
			if a == nil {
				continue
			}
			if pr.hasValue && a.UnescapedValue() != pr.attrValue {
				continue
			}
			out = append(out, e)
		}
		return out
	}
	return els
}

// nthPerParent groups els by their parent (preserving first-seen parent order)
// and returns the nth (1-based) element of each group — REXML's positional
// predicate semantics, which count within each parent context.
func nthPerParent(els []*Element, n int) []*Element {
	type group struct {
		members []*Element
	}
	groups := map[*Element]*group{}
	var order []*Element // parent keys in first-seen order
	for _, e := range els {
		g, ok := groups[e.par]
		if !ok {
			g = &group{}
			groups[e.par] = g
			order = append(order, e.par)
		}
		g.members = append(g.members, e)
	}
	var out []*Element
	for _, p := range order {
		g := groups[p]
		if n <= len(g.members) {
			out = append(out, g.members[n-1])
		}
	}
	return out
}

// descendants returns e's descendant elements in document order (self excluded).
func descendants(e *Element) []*Element {
	var out []*Element
	var walk func(*Element)
	walk = func(n *Element) {
		for _, c := range n.ChildElements() {
			out = append(out, c)
			walk(c)
		}
	}
	walk(e)
	return out
}

// rootElement finds the root element of the document containing n.
func rootElement(n Node) *Element {
	switch v := n.(type) {
	case *Document:
		return v.Root()
	case *Element:
		cur := v
		for cur.par != nil {
			cur = cur.par
		}
		return cur
	default:
		return nil
	}
}
