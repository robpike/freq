// Copyright 2012 The rspace Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Freq (frequency) counts how many times each distinct
// Unicode code point appears in the input. The -bytes
// option counts bytes instead. The table is then printed
// to standard output, one count per line. Nothing is
// printed for a code point if its count is zero.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	countBytes bool
)

func init() {
	flag.BoolVar(&countBytes, "bytes", false, "count bytes (default is runes)")
	flag.BoolVar(&countBytes, "b", false, "alias for -bytes")
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		read("<stdin>", os.Stdin)
	}
	for _, file := range flag.Args() {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "freq:", err)
			os.Exit(1)
		}
		read(file, f)
		f.Close()
	}
	print()
}

// We lazily fill in the intermediate arrays, each 256 entries long.
// Unicode is 22 bits, so we only need 3 levels max.
// Indexing starts with the uppermost byte, so the innermost array
// (of uint64 elements) represents 256 consecutive code points.
type Counts [256]*[256]*[256]uint64

var counts = new(Counts) // Allocate the top level; we know we'll need it unless the input is empty.
var errors uint64        // Special count to distinguish FFFD from real errors.

func (c *Counts) Inc(r rune) {
	b2 := (r >> 16) & 0xFF
	b1 := (r >> 8) & 0xFF
	b0 := (r >> 0) & 0xFF
	c2 := (*c)[b2]
	if c2 == nil {
		c2 = new([256]*[256]uint64)
		(*c)[b2] = c2
	}
	c1 := c2[b1]
	if c1 == nil {
		c1 = new([256]uint64)
		c2[b1] = c1
	}
	c1[b0]++
}

func read(file string, f *os.File) {
	if countBytes {
		readBytes(file, f)
	} else {
		readRunes(file, f)
	}
}

func readBytes(file string, f *os.File) {
	buf := bufio.NewReader(f)
	for {
		byte, err := buf.ReadByte()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "freq: %s: %s\n", file, err)
			os.Exit(1)
		}
		counts.Inc(rune(byte))
	}
}

func readRunes(file string, f *os.File) {
	buf := bufio.NewReader(f)
	for {
		rune, width, err := buf.ReadRune()
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "freq: %s: %s\n", file, err)
			os.Exit(1)
		}
		if rune == 0xFFFD && width == 1 {
			errors++
		} else {
			counts.Inc(rune)
		}
	}
}

func print() {
	if countBytes {
		printCounts("%.2x %c\t%d\n", "%.2x -\t%d\n")
	} else {
		printCounts("%.4x %c\t%d\n", "%.4x -\t%d\n")
	}
}

func printCounts(printable, unprintable string) {
	for b2 := range *counts {
		c2 := (*counts)[b2]
		if c2 == nil {
			continue
		}
		for b1 := range c2 {
			c1 := c2[b1]
			if c1 == nil {
				continue
			}
			for b0, count := range c1 {
				if count == 0 {
					continue
				}
				var r rune = rune((b2 << 16) | (b1 << 8) | b0)
				if r != ' ' && strconv.IsPrint(r) {
					fmt.Printf(printable, r, r, count)
				} else {
					fmt.Printf(unprintable, r, count)
				}
			}
		}
	}
	if errors > 0 {
		fmt.Printf("error -\t%d\n", errors)
	}
}
