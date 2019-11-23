// Copyright 2019 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := elispToMarkdown(os.Stdout, os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func elispToMarkdown(out io.Writer, filename string) error {
	w, ok := out.(*bufio.Writer)
	if !ok {
		w = bufio.NewWriter(out)
	}
	inElisp := false
	blankCnt := 0
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	writeLine := func(s string) error {
		if _, err := w.WriteString(s); err != nil {
			return err
		}
		return w.WriteByte('\n')
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			blankCnt += 1
		} else if strings.HasPrefix(line, ";;;") {
			if inElisp {
				if _, err := w.WriteString("```\n"); err != nil {
					return err
				}
				inElisp = false
			}
			if blankCnt > 0 {
				if _, err := w.WriteString(strings.Repeat("\n", blankCnt)); err != nil {
					return err
				}
				blankCnt = 0
			}
			if s := strings.TrimPrefix(line, ";;; {{"); len(s) < len(line) {
				if fn := strings.TrimSuffix(s, "}}"); len(fn) < len(s) {
					if b := filepath.Base(fn); len(b) != len(fn) {
						return fmt.Errorf("file %s should be just base name", fn)
					}
					path := filepath.Join(filepath.Dir(filename), fn)
					if err := elispToMarkdown(w, path); err != nil {
						return err
					}
					continue
				}
			}
			i := 3
			if len(line) > 3 && line[3] == ' ' {
				i = 4
			}
			if err := writeLine(untabify(line[i:])); err != nil {
				return err
			}
		} else {
			if blankCnt > 0 {
				if _, err := w.WriteString(strings.Repeat("\n", blankCnt)); err != nil {
					return err
				}
				blankCnt = 0
			}
			if !inElisp {
				if _, err := w.WriteString("```emacs-lisp\n"); err != nil {
					return err
				}
				inElisp = true
			}
			if err := writeLine(untabify(line)); err != nil {
				return err
			}
		}
	}
	if inElisp {
		if _, err := w.WriteString("```\n"); err != nil {
			return err
		}
	}
	err = w.Flush()
	if err := sc.Err(); err != nil {
		return err
	}
	return err
}

func untabify(s string) string {
	t := strings.TrimLeft(s, "\t")
	n := len(s) - len(t)
	if n == 0 {
		return s
	}
	return strings.Repeat(" ", 8*n) + t
}
