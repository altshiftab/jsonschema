// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

// printKeywordsSort writes out the keyword sorting function.
// Sorting keywords ensures that a keyword sees any required annotations.
func printKeywordsSort(sortBuf *bytes.Buffer, keywords []keywordData) {
	found := false
	for _, k := range keywords {
		if len(k.After) > 0 {
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintln(sortBuf, "// keywordCmp is the keyword comparison routine.")
		fmt.Fprintln(sortBuf, "func keywordCmp(a, b string) int {")
		fmt.Fprintln(sortBuf, "\treturn strings.Compare(a, b)")
		fmt.Fprintln(sortBuf, "}")
		return
	}

	// Sort by name in descending order.
	sortedKeywords := slices.Clone(keywords)
	slices.SortFunc(sortedKeywords, func(a, b keywordData) int {
		return -strings.Compare(a.Name, b.Name)
	})

	// Build a list of keywords in reverse order,
	// putting each keyword that must follow other keywords in place.
	ranked := make([]keywordData, 0, len(sortedKeywords))
	for _, k := range sortedKeywords {
		ins := len(ranked)
		for _, a := range k.After {
			for i, r := range ranked {
				if r.Name == a {
					ins = min(ins, i)
				}
			}
		}
		ranked = slices.Insert(ranked, ins, k)
	}

	// Reverse the list to get the expected order.
	slices.Reverse(ranked)

	fmt.Fprintln(sortBuf, "// sortRank is the ranking of each keyword when sorting")
	fmt.Fprintln(sortBuf, "var sortRank = map[string]int{")
	for i, r := range ranked {
		fmt.Fprintf(sortBuf, "\t%q: %d,\n", r.Name, i)
	}
	fmt.Fprintln(sortBuf, "}")
	fmt.Fprintln(sortBuf)

	fmt.Fprintln(sortBuf, "// keywordCmp is the keyword comparison routine.")
	fmt.Fprintln(sortBuf, "func keywordCmp(a, b string) int {")
	fmt.Fprintln(sortBuf, "\treturn cmp.Compare(sortRank[a], sortRank[b])")
	fmt.Fprintln(sortBuf, "}")
}
