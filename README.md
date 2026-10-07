<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rexml/brand/main/social/go-ruby-rexml-rexml.png" alt="go-ruby-rexml/rexml" width="720"></p>

# rexml — go-ruby-rexml

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rexml.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.1%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the core of Ruby's
[REXML](https://docs.ruby-lang.org/en/master/REXML.html) XML library** — the tree
model, the parser, the serialisers (the compact default formatter and
`Formatters::Pretty`), and a widely-used subset of XPath — matching MRI 4.0.5's
`rexml` 3.4.4 behaviour **without any Ruby runtime**.

It is built from scratch rather than wrapping `encoding/xml`: REXML's API,
whitespace handling and round-trip semantics differ. REXML keeps text and
attribute values in their **raw, still-escaped form**, defaults to
**single-quoted** attributes, **sorts** attributes by local name in the compact
formatter (but keeps source order in Pretty), **self-closes** empty elements, and
preserves comments / CDATA / processing instructions verbatim — so
`ParseDocument(xml).ToString()` reproduces what `REXML::Document.new(xml).to_s`
emits, byte-for-byte.

It is the REXML backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-yaml](https://github.com/go-ruby-yaml/yaml) (Psych),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) (Onigmo) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb) (ERB).

## Features

Faithful port of REXML's DOM + parse + serialise + XPath, validated against the
`ruby` binary on every supported platform:

- **Parse** — `ParseDocument` builds a tree of `Document` / `Element` /
  `Attribute` / `Text` / `Comment` / `CData` / `Instruction` (PI) / `DocType` /
  `XMLDecl`, preserving raw text, entities, the XML declaration and the DOCTYPE
  (including a bracketed internal subset). Malformed input returns a `*ParseError`
  with a byte offset, mirroring `REXML::ParseException`.
- **DOM** — `Element{Prefix, Name, Attributes, Children, …}` with
  `Add` / `AddElement` / `AddAttribute` / `AddText` / `SetText`, `Elements(path)`,
  `EachElement`, `ChildElements`, `Text` / `GetText` / `Texts`, `ElementAt(n)`
  (1-based), `FirstElement`, `QName` / `ExpandedName` / `Prefix` / `NamespaceURI`,
  `Root` / `RootNode`. `Attributes` is an **ordered** map.
- **Serialise** — `(*Document).ToString` (compact, REXML's `to_s`),
  `Write(WriteOptions)`, and the Pretty formatter `Pretty(indent)` /
  `PrettyString` — matching REXML's exact output: single-quoted attributes,
  local-name attribute ordering (compact) vs source order (Pretty), entity
  escaping (`& < >` in text, `& < > " '` for programmatic values, literal `'`
  escaped to `&apos;` inside the single-quoted form), numeric-reference and
  CDATA / comment / PI round-trip, and self-closing empty elements.
- **XPath subset** — `XPathFirst` / `XPathEach` / `XPathMatch` over the common
  subset (see the boundary below).

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x).

## Install

```sh
go get github.com/go-ruby-rexml/rexml
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-rexml/rexml"
)

func main() {
	d, _ := rexml.ParseDocument(`<config><server name="web" port="8080"/></config>`)

	fmt.Println(d.ToString())
	// <config><server name='web' port='8080'/></config>   (attrs sorted, single-quoted)

	fmt.Println(d.Pretty(2))
	// <config>
	//   <server name='web' port='8080'/>
	// </config>

	srv := rexml.XPathFirst(d, "//server").(*rexml.Element)
	name, _ := srv.Attr("name")
	fmt.Println(name) // web

	// Build a tree programmatically.
	doc := rexml.NewDocument()
	root := doc.AddElement("root")
	root.AddElement("item").AddText("hello & <world>")
	fmt.Println(doc.ToString())
	// <root><item>hello &amp; &lt;world&gt;</item></root>
}
```

## Node model

`ParseDocument` returns a `*Document` whose `Children` hold the top-level nodes
(optional `*XMLDecl`, `*DocType`, comments / PIs, and the single root `*Element`).
Every node implements the unexported `Node` interface; the concrete types a host
binds are:

| REXML class               | Go type          | Notes                                        |
| ------------------------- | ---------------- | -------------------------------------------- |
| `REXML::Document`         | `*Document`      | `Root`, `RootNode`, `XMLDecl`, `DocType`     |
| `REXML::Element`          | `*Element`       | `Prefix`, `Name`, `Attributes`, `Children`   |
| `REXML::Attribute`        | `*Attribute`     | `Prefix`, `Name`, `Value`, `UnescapedValue`  |
| `REXML::Attributes`       | `*Attributes`    | ordered; `Get`, `Set`, `Each`, `EachSorted`  |
| `REXML::Text`             | `*Text`          | raw round-trip; `Val` decodes, `String` raw  |
| `REXML::Comment`          | `*Comment`       | verbatim content                             |
| `REXML::CData`            | `*CData`         | verbatim content                             |
| `REXML::Instruction` (PI) | `*Instruction`   | `Target`, `Content`                          |
| `REXML::DocType`          | `*DocType`       | raw body, round-trips                         |
| `REXML::XMLDecl`          | `*XMLDecl`       | `Version`, `Encoding`, `Standalone`          |

## XPath subset boundary

`XPathMatch` / `XPathFirst` / `XPathEach` implement the widely-used subset REXML's
`XPath` covers:

**Supported** — absolute (`/a/b/c`) and descendant (`//tag`, `a//b`) paths;
the wildcard `*`; the child and descendant axes; attribute steps `@attr` and
`@*`; the `text()` node test; and predicates `[n]` (1-based **per-parent**
position, matching REXML's `//a[1]` selecting the first `a` within *each* parent),
`[@attr]` (existence) and `[@attr='v']` / `[@attr="v"]` (value, compared against
the decoded attribute value). A relative path applied to a `*Document` resolves
from the root element, like REXML.

**Out of scope** (documented boundary) — axes other than child / descendant (no
`ancestor`, `following-sibling`, `parent`, `..`); functions beyond `text()` (no
`count()`, `position()`, `last()`, `name()`, `contains()`, …); arithmetic; and
compound boolean predicates (`and` / `or`). An unrecognised predicate leaves the
node set unchanged.

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: a wide XML corpus is parsed and re-serialised here
and by the system `ruby` (`REXML::Document.new(xml).to_s`,
`Formatters::Pretty`, `REXML::XPath.match`), and the bytes are compared exactly.
The oracle feeds XML through **stdin** and `$stdout.binmode` / `$stdin.binmode`
so Windows text-mode never pollutes the bytes; it is gated to Ruby ≥ 4.0 and
skips itself where `ruby` is absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-rexml/rexml authors.

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```
