elisp2md
========

`elisp2md` converts Emacs lisp files with Markdown comments (such as
[my Emacs configuration](https://github.com/lukpank/.emacs.d)) into
Markdown documents, optionally using Emacs for syntax highlighting
(the final HTML result for above `init.el` is
[here](https://lupan.pl/dotemacs/).


Installation
------------

Having [Go](https://golang.org/) installed run (outside of any Go
module, for example in your home directory)

```
$ GO111MODULE=on go get github.com/lukpank/elisp2md@latest
```

or run

```
$ go get github.com/lukpank/elisp2md
```


Usage
-----

For the following `example.el` file

```
;;; Example
;;; -------
;;;
;;; Some text.

(require 'test)
```

run the command

```
$ elisp2md -o OUTPUT.md example.el
```

to turns it into the following Markdown document

    Example
    -------

    Some text.

    ```emacs-lisp
    (require 'test)
    ```


### Use Emacs for syntax highlighting

You can use option `--htmlize` to use Emacs for syntax highlighting
(requires installing Emacs [htmlize](https://melpa.org/#/htmlize) and
[paren-face](https://melpa.org/#/paren-face) packages, for example
from [Melpa](https://melpa.org/)). Run the command

```
$ elisp2md --htmlize -o OUTPUT.md example.el
```

to obtain

```
Example
-------

Some text.

<pre>
<span class="parenthesis">(</span><span class="keyword">require</span> '<span class="constant">test</span><span class="parenthesis">)</span>
</pre>
```


### Preserve output document header

If the output file (given as argument of option `-o`) exists and
contains a TOML header of the form

```
+++
date = "2020-01-25"
title = "My example"
+++
```

you can use option `-H` to preserve the header when writing new output
file.
