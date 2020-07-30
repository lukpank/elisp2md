// Copyright 2019 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lukpank/elisp2md/htmlize"
)

func main() {
	output := flag.String("o", "", "output file")
	preserveHeader := flag.Bool("H", false, "preserve header")
	htmlizeFlag := flag.Bool("htmlize", false, "run htmlize on code blocks")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "error: argument required: input file name")
		os.Exit(1)
	}
	if err := run(*output, flag.Arg(0), *preserveHeader, *htmlizeFlag); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}
}

func run(outputFilename, intputFilename string, preserveHeader, htmlizeFlag bool) (err error) {
	out := os.Stdout
	header := ""
	if outputFilename != "" {
		if preserveHeader {
			header, err = readHeader(outputFilename)
			if err != nil {
				return err
			}
		}
		out, err = os.Create(outputFilename)
		if err != nil {
			return err
		}
		defer func() {
			if err2 := out.Close(); err != nil {
				if err == nil {
					err = err2
				} else {
					fmt.Fprintf(os.Stderr, "error: %v", err2)
				}
			}
		}()
	}
	if htmlizeFlag {
		var b bytes.Buffer
		if err := elispToMarkdown(&b, intputFilename, header); err != nil {
			return err
		}
		return htmlizeCodeBlocks(out, &b)
	}
	return elispToMarkdown(out, intputFilename, header)
}

func elispToMarkdown(out io.Writer, filename, header string) error {
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
	if header != "" {
		if _, err := w.WriteString(header); err != nil {
			return err
		}
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
					if err := elispToMarkdown(w, path, ""); err != nil {
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

func readHeader(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	count := 0
	lines := []string{}
	for sc.Scan() {
		line := sc.Text()
		lines = append(lines, line)
		if strings.TrimSpace(line) == "+++" {
			count++
		}
		if count == 0 {
			return "", fmt.Errorf("file %s does not start with a header", filename)
		} else if count == 2 {
			lines = append(lines, "\n")
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", sc.Err()
}

const elispInit = `
(package-initialize)
(require 'use-package nil t)
(add-hook 'emacs-lisp-mode-hook #'paren-face-mode)
`

func htmlizeCodeBlocks(w io.Writer, r io.Reader) error {
	sc := bufio.NewScanner(r)
	texts := []string{}
	blocks := []*htmlize.CodeBlock{}
	inCodeBlock := false
	lang := ""
	var b bytes.Buffer
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				block := &htmlize.CodeBlock{Lang: lang}
				block.Input = append(block.Input, b.Bytes()...)
				blocks = append(blocks, block)
			} else {
				texts = append(texts, b.String())
				lang = strings.TrimPrefix(line, "```")
			}
			b.Reset()
			inCodeBlock = !inCodeBlock
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if inCodeBlock {
		return errors.New("last code block was not closed")
	}
	texts = append(texts, b.String())
	h := htmlize.New()
	h.Init = elispInit
	if err := h.Highlight(blocks); err != nil {
		return err
	}
	for i := range blocks {
		if _, err := io.WriteString(w, texts[i]); err != nil {
			return err
		}
		if _, err := w.Write(blocks[i].Output); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, texts[len(texts)-1])
	return err
}
