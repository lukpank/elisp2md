// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package htmlize

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"unicode"
)

type CodeBlock struct {
	Lang   string
	Input  []byte
	Output []byte
}

const elispInit = `
(defun my-htmlize-file (filename mode-function)
  (find-file filename)
  (funcall mode-function)
  (font-lock-ensure)
  (with-current-buffer (htmlize-buffer)
    (write-file (concat filename ".html"))))
`

// New returns Htmlize with default Emacs command and argumens
func New() *Htmlize {
	return &Htmlize{Command: "emacs", Args: []string{"--batch", "-l"}}
}

type Htmlize struct {
	Command string
	Args    []string
	Init    string
}

// Highlight runs Emacs to syntax highlight the slice of code blocks.
func (h *Htmlize) Highlight(codes []*CodeBlock) error {
	tmpDir, err := ioutil.TempDir("", "htmlize")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	filenames := make([]string, len(codes))
	for _, c := range codes {
		if !validLang(c.Lang) {
			return fmt.Errorf("htmlize: invalid language name: %s", c.Lang)
		}
	}

	for i, c := range codes {
		filenames[i] = path.Join(tmpDir, fmt.Sprintf("%04d", i))
		if err := ioutil.WriteFile(filenames[i], c.Input, 0600); err != nil {
			return err
		}
	}
	elisp := path.Join(tmpDir, "init.el")
	f, err := os.Create(elisp)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s%s(progn\n", elispInit, h.Init)
	if err != nil {
		return err
	}
	for i, fn := range filenames {
		if codes[i].Lang != "" {
			_, err = fmt.Fprintf(f, "(my-htmlize-file %q '%s-mode)\n", fn, codes[i].Lang)
		} else {
			_, err = fmt.Fprintf(f, "(htmlize-file %q)\n", fn)
		}
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(f, ")")
	if err != nil {
		return err
	}
	f.Close()

	n := len(h.Args)
	if err := exec.Command(h.Command, append(h.Args[:n:n], elisp)...).Run(); err != nil {
		return err
	}
	for i, c := range codes {
		b, err := ioutil.ReadFile(filenames[i] + ".html")
		if err != nil {
			return err
		}
		if c.Output, err = extractPre(b); err != nil {
			return err
		}
	}
	return nil
}

func validLang(name string) bool {
	for _, c := range name {
		if !unicode.IsLetter(c) && c != '-' && c != '/' {
			return false
		}
	}
	return true
}

var startPre = []byte("<pre>")
var endPre = []byte("</pre>\n")

func extractPre(b []byte) ([]byte, error) {
	i := bytes.Index(b, startPre)
	j := bytes.LastIndex(b, endPre)
	if i == -1 || j == -1 {
		return nil, errors.New("htmlize: could not find <pre> and </pre> pair")
	}
	return b[i : j+len(endPre)], nil
}
