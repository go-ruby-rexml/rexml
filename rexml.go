// Copyright (c) the go-ruby-rexml/rexml authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rexml

// VERSION reports the REXML release whose observable behaviour this package
// targets (MRI 4.0.5 ships rexml 3.4.4).
const VERSION = "3.4.4"

// Parse is an alias for ParseDocument, matching the common REXML idiom
// REXML::Document.new(xml).
func Parse(xml string) (*Document, error) { return ParseDocument(xml) }
