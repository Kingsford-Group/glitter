# Glitter

Glitter is an implementation of literate programming system for the `Go` programming language. It supports taking one or more Glitter files (syntax defined below) and turning it into multiple `.go` files to compile or a single `.tex` file to typeset.

In fact, the system doesn’t really know anything sophisticated about Go or TeX, but has been designed to work well with those systems. And out of the box, the defaults all assume Go+TeX. But these can all be configured.

**This software is in development and not complete or ready to be used.**

# Glitter Syntax

We give the syntax using regular expressions.

## Blocks

A **text block** starts with `^\s*@:+`. Everything following the final `:` is placed into the natural language file, in the order they appear in the file. The number of `:` characters doesn’t affect anything, so long as there is one. This allows regions of the file to be separated like so:

```
@: This is a text block
of two lines
@::::::::::::::::::::::::::::::::::::::::::::::::::::::
\section{A new section}
This is a text block that happens to start with a LaTeX command
to start a new section.
```



A **code block** starts with a line of the form `^\s*<<.+>>=\s*$`.  The string between  `<<`…`>>` is `S`. Leading and trailing whitespace is removed from `S`. If `S` is of the form `\*\s*"f"\s+n`, where `f` is a string in double quotes and `n` is an integer, then this is a top-level block that will be written to file `f` in order given by `n`. If `n` is omitted, it is assumed to be 0. If `f` is omitted, it is the last `f` that occurred in some code block name. If `S` is any other form, it defines a named code block.

Code blocks with the same canonical name (see below) are concatenated in the order they appear in the input.

A block ends when another block starts or the file ends.

A string in a code block of the form `<<.+>>` is a substitution call. It is recursively replaced by the code in code block with the same canonical name as given between the `<<`…`>>`. If non-whitespace characters follow `>>` on the same line as a substitution, the final trailing newline character in the code referenced code block is removed before substitution.

**Canonical names** are formed by:

* removing leading and trailing whitespace
* replacing any sequence of whitespace with a single space
* converting the name to lowercase
* converting escaped characters (see below) to the referenced character

**Top-level blocks**: A top level block is a code block with a name that starts with `*`, which can be followed by a filename and a number, both optional. If present, they must be in the order `“filename” number`. The filename must be enclosed in `“”`. So the following are valid:

```
<<*>>=
<<* "out.go" 10>>=
<<* 15>>=
<<* "out.go">>=
<< * "moo.go"  >>=
```

If the number is omitted, it is assumed to be 0. If the filename is omitted, it is the last named file mentioned in a code block definition or the default output file if no previous named file was given. If the filename is given but empty `“”` then it is the default output file.

This rule means that filenames are “sticky” within a top-level file (and its includes):

```
<<* “constants.go”>>=
   code block 1
<<*>>=
	writes to constants.go
```

The current output file (and the default output file) are both reset when a new top-level file is read or when a `@glitter top` line is found.

**Marking a file as top-level**. A line, occurring as the first non-blank line in a file, of the form `@glitter top` marks the file as a top level file to be found by the tangle command when searching directories (see below).

When tangling, the effect of a `@glitter top` line is to set the current and default output files to be the same name as the current source file with its extension changed.

## Escapes

Anyplace, a sequence `@'x` is treated as a single occurrence of `x` but only *after* the special patterns are interpreted. For example, `@'@` is a single `@` and `@'<` is a single `<`. Occurrences of `@` and `<` that can’t be interpreted as one of the forms above do not need to be escaped. Substitution of escapes is applied as late as possible in the processing. 

Note that this rule doesn’t make characters inactive. So in ``<<foo @'>> moose >>`` the first `>>` is still used as the endpoint of the reference, and this defines a reference to `foo @\'` since the first `>>` still exists in the file. To include, a literal `>>`, one would say `>@'>`, since that causes `>>` to not be present.

## Includes

A line matching with `^\s*@include\s+".+"$` is an include line. It is replaced by the contents of the file named between the quotes. Includes act (nearly) exactly as if the lines in the included file were typed in at the point of the include. The one exception to this is the `@glitter` command which always has file scope. 

## Example

```
@: Block
<<Segment name is here>>= 
	var blockChain []string
	@': //this does not start a new block

@:Text Text Text
<<Another Segment name is here>>=

@include "defn.gw"

@:
This writes the configuration to a options file.
<<* "options.json" 0>>=
   {"verbose": 1}
```

## Summary of syntax:

* `@:` as the first non-whitespace on a line starts a text block
* `<<code block name>>=` on a line of its own starts a code block.
* `<<* “file” 10>>=` on a line of its own starts a top-level block that will be written to “file”; blocks written to that file will be sorted by the number given in the 3rd position (e.g. 10). The `“file”` and/or the number may be omitted, in which case defaults will be used.
* `@include "file"` is (recursively) replaced by the contents of “file”.
* `@glitter top` as the first non-blank line in a file does two things: (1) marks the file for inclusion when a directory is given to tangle; and (2) sets the default output filename to a modification of the current glitter filename (`.gw` → `.go`). This command is scoped to the file and its include subtree. [The odd syntax is to leave room for other options besides `top` in the future.]
* `@'x` means: at the last possible moment, just before writing to a file, replace with `x`, where `x` can be any character.
* `<<code block name>>` inside of a code block is (recursively) substituted with the content of the named code block during tangle.

## Command line usage

### Weaving

Weaving creates a single .tex file:

```
glitter -out out.tex weave file1 file2 file3 file4 …
```

This will read the given files and produce the out.tex for typesetting. You must specify the files explicitly, though those files can include other files.

You must list the files explicitly so that glitter knows what order to typeset them in. A good technique (but not required), would be to create a file in the root directory of your project that includes the other files in the order they should be typeset:

```
@include “foo.gw”
@include “lexer/what.gw”
@include “why.gw”
```

Whitespace at the start of each line in a code block is removed in such a way to shift the code block as far to the left as possible. Blank lines at the start and end of the code block are removed. [NYI]

Comments of the form `%line N “file”` are included whenever the file switches.

Any lines before the first block encountered in a run are output between the command that starts the file (`Start` below) and the command that starts the content (`StartBook` below). This means that lines before the first block encountered are added to the LaTeX preamble (assuming you are using LaTeX).

### Tangling

Tangling is more complex (but not much more so). It reads a set of files and produces a set of .go files.

```
glitter tangle dir1 dir2 file1 dir3 file2
```

Tangle may be given a list of directories and files.

For each directory, the tree rooted at that directory will be scanned for all files ending with `.gw`. If the first non-blank line in a file that is found is `@glitter top` then the file will be added to the list of files to process.

If a file is given explicitly, then it is always added to the list of files that are processed.

The list is sorted lexicographically, and duplicate files are removed. The files are then processed in order. Let $F_1,F_2,\dots,F_k$ be the files.

When processing $F_i$ *and all the files included from it* (unless they start with `@glitter top`), the default output file is $F_i$​ with its `.gw` extension replaced by `.go`. (If the filename does not end with `.gw` then the `.go` extension is appended to the filename.)

If you want to tangle only given files, you can list them on the command line. Such explicitly listed files do not need to start with `@glitter top` to be processed (but they can).

If you want to tangle all `@glitter top` files in a tree, you can give the root of the tree: `glitter tangle .` will tangle everything labeled with `@glitter top` in the current directory and its subdirectories.

If you want to create a top-level file that specifies what to tangle you can do so (as in the weave example above), so long as each of the included files is marked as `@glitter top`. Then you simply `glitter tangle topfile.gw`.

All files are read before any output is produced. Code block names are global and can be referenced from any file in the same tangle run.

Code blocks with the same name are concatenated in the order they are encountered in the stream.

If you give the `-forbid-multi-includes` option, then no file will be included more than once, no matter how many times it’s encountered (whether it is read via an `@include` or from the list of files). Normally, you can include the same file more than once, but then they are treated exactly like processing the same file multiple times, and so should probably not contain code blocks, since multiple occurrences of a code block will be concatenated, etc., which is probably not what you want.

A reasonable use case for including multiple files would be to create a `copyright.gw` file:

```
<<*>>=
  // (c) Great Company, Inc. 2024. All rights reserved
  package foo
```

and then include the following at the top of every top-level file:

```
@glitter top
@include "copyright.gw"
```

Then every generated source file will contain the copyright and the package command. This lets you edit the copyright message only once. 

Top-level blocks are sorted by their `number` (the number given in `<<* “file” number>>`).

During tangling of a top-level block, the following happens in order:

1. All occurrences of code references `<< … >>` are replaced by the named block. This is done recursively until all code references are eliminated. This expansion happens in such a way as to make a reasonably formatted and indented file.

2. Escape characters are replaced.

3. The resulting expanded text is written to the file.

## Weaving substitutions

By default, the output of weave is a text file that uses the LaTeX class `glittertex`. If you are happy with this, there is nothing you need to change. You can typeset the file using `pdflatex foo.tex`. But much of this output can be customized.

The default substitutions are given in the table below. The `\glitter…` commands are defined in the `glittertex` LaTeX class.

| Glitter                     | Option        | Default                                     |
| --------------------------- | ------------- | ------------------------------------------- |
| `@:`                        | StartText     | `\glitterStartText`                         |
| end of `@:` block           | EndText       | `\glitterEndText$n`                         |
| `<<code block name>>=`      | StartCode     | `\glitterStartCode{$1}$n\begin{lstlisting}` |
| end of `<<…>>=` code block  | EndCode       | `\end{lstlisting}\glitterEndCode$n`         |
| `<< … >>` in code block     | CodeCodeRef   | `#\glitterCodeRef{$1}#`                     |
| `<< … >>` in text block     | TextCodeRef   | `\glitterCodeRef{$1}`                       |
| Start of output             | Start         | `\documentclass{glittertex}`                |
| Before start of first block | StartBook     | `\glitterStartBook`                         |
| End of output               | EndBook       | `\glitterEndBook`                           |
|                             | CodeEscape    | `#`                                         |
| CodeEscape in code block    | CodeEscapeSub | `#\glitterHash#`                            |

Any of these substitutions can be changed by reading a configuration file with lines of the form:

```
%%glitter OPTION REPLACEMENT TEXT
```

where `OPTION` is one of the options given in column 2 of the above table. The rest of the line gives what weave should output at that event. The options `StartCode`, `CodeCodeRef` and `TextCodeRef` take one argument, and the replacement text can refer to it (once) using `$1`. Any occurrence of `$n` in the replacement text is replaced by a newline.

Here is a configuration file that mimics the default options:

```
%%glitter Start         \documentclass{glittertex}
%%glitter StartBook     \glitterStartBook
%%glitter EndBook       \glitterEndBook
%%glitter StartText     \glitterStartText
%%glitter EndText       \glitterEndText$n
%%glitter StartCode     \glitterStartCode{$1}$n\begin{lstlisting}
%%glitter EndCode       \end{lstlisting}\glitterEndCode$n
%%glitter CodeEscape    #
%%glitter CodeCodeRef   #\glitterCodeRef{$1}#
%%glitter TextCodeRef   \glitterCodeRef{$1}
%%glitter CodeEscapeSub #\glitterHash#
```

In fact, you will find these lines in the `glittertex.cls` file that defines the LaTeX class that is used. When reading the configuration, any lines that do not start with `%%glitter ` are ignored. This means that the `glittertex.cls` file contains its own weave configuration, making it easy to co-edit the weave options and the corresponding LaTeX definitions. 

You can read the configuration with:

```
glitter -config glittertex.cls weave ...
```

If you don’t specify the `-config` option, glitter will use its built-in defaults.

The substitution of `#` to `#\glitterHash#` deserves some explanation. In the default templates, the code blocks are typeset using the `listings` LaTeX package. To typeset a code ref inside of listings, we enable escaping to LaTeX with the `#` character. Hence, in a code block, a ref is output as `#\glitterCodeRef{foo}#`. But your code might have a `#` in it (in a comment, or string literal for example). If it does, it is replaced by `#\glitterHash#`, and `\glitterHash` by default is defined to be `\texttt{\char35}`, which is a `#` (using standard font encodings)! This will make the `#` appear, but won’t confuse `listings` with a `#` character.

# Roadmap

This is a work in progress. Commits may not compile, and currently it is just barely usable. Both tangle and weave work, though are not tested in any systematic way.

Things to do:

1. Output `//line n “file”` comments in tangle
2. Options to handle some go-specific things (like automatic insertion of `package` statements)
3. Improve error messages
4. Special syntax (like noweb’s `[[ … ]]`) to typeset code inside a text block, e.g. by replacing with `\lstinline`
5. Indexing and cross-referencing (i.e. “code is used in …” or forward page references. Ideally leveraging LaTeX’s facilities.)
6. Hidden chunks that exist, but aren’t displayed in the typeset document.
