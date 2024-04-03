# Glitter

Glitter is an implementation of literate programming system for the `Go` programming language. It supports taking one or more Glitter file (syntax defined below) and turning it into multiple `.go` files to compile or a single `.tex` file to typeset.

In fact, the system doesn’t really know anything sophisticated about Go or TeX, but has been designed to work well with those systems. And out of the box (currently) the defaults all assume Go+TeX.

**This software is in development and not complete or ready to be used.**

# Glitter Syntax

## Blocks

A **text block** starts with `^\s*@:`. Everything following the `:` is placed into the natural language file, in the order they appear in the file.

A **code block** starts with a line of the form `^\s*<<.+>>=\s*$`.  The string between  `<<`…`>>` is `S`. Leading and trailing whitespace is removed from `S`. If `S` is of the form `\*\s*"f"\s+n`, where `f` is a string in double quotes and `n` is an integer, then this is a top-level block that will be written to file `f` in order given by `n`. If `n` is omitted, it is assumed to be 0. If `f` is omitted, it is the last `f` that occurred in some code block name. If `S` is any other form, it defines a named code block.

Code blocks with the same canonical name (see below) are concatenated in the order they appear in the file.

A block ends when another block starts or the file ends.

A source file must consist of alternating text and code blocks. 

A string in a code block of the form `<<.+>>` is a substitution call. It is recursively replaced by the code in code block with the same canonical name as given between the `<<`…`>>`. If non-whitespace characters follow `>>` on the same line as a substitution, trailing newline characters in the code referenced code block are removed before substitution.

**Canonical names** are formed by:

* removing leading and trailing whitespace
* replacing any sequence of whitespace with a single space
* converting the name to lowercase
* converting escaped characters to the referenced character

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

**Marking a file as top-level**. A line, occurring as the first non-blank line in a file, of the form `@glitter top` marks the file as a top level file to be found by the tangle command when searching directories.

When weaving, the effect of a `@glitter top` line is to set the current and default output files to be the same name as the source file with its extension changed.

## Escapes

Anyplace in the file, a sequence `@'x` is treated as a single occurrence of `x` but only *after* the special patterns are interpreted. For example, `@'@` is a single `@` and `@'<` is a single `<`. Occurrences of `@` and `<` that can’t be interpreted as one of the forms above do not need to be escaped. Substitution of escapes is applied as late as possible in the processing. 

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

@: This writes the configuration to a options file.
<<* "options.json" 0>>=
   {"verbose": 1}
```

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

### Tangling

Tangling is more complex (but not much more so). It reads a set of files and produces a set of .go files.

```
glitter tangle dir1 dir2 file1 dir3 file2
```

Tangle may be given a list of directories and files.

For each directory, the tree rooted at that directory will be scanned for all files ending with `.gw`. If the first non-blank line in a file that is found is `@glitter top` then the file will be added to the list of files to process.

If a file is given explicitly, then it is always added to the list of files that are processed.

The list is sorted lexicographically, and duplicate files are removed. The files are then processed in order. Let $F_1,F_2,\dots,F_k$ be the files.

When processing $F_i$ *and all the files included from it*, the default output file is $F_i$​ with its `.gw` extension replaced by `.go`. (If the filename does not end with `.gw` then the `.go` extension is appended to the filename.)

If you want to tangle only given files, you can list them on the command line. Such explicitly listed files do not need to start with `@glitter top` to be processed (but they can).

If you want to tangle all `@glitter top` files in a tree, you can give the root of the tree: `glitter tangle .` will tangle everything in the current directory and its subdirectories.

If you want to create a top-level file that specifies what to tangle you can do so (as in the weave example above), so long as each of the included files is marked as `@glitter top`. Then you simply `glitter tangle topfile.gw`.

All files are read before any output is produced. Code block names are global and can be referenced from any file in the same tangle run.

Code blocks with the same name are concatenated in the order they are encountered in the stream.

If you give the `-forbid-multi-includes` option, then no file will be included only once, no matter how many times it’s encountered (whether it is read via an `@include` or from the list of files). Normally, you can include the same file more than once, but then they are treated exactly like processing the same file multiple times, and so should probably not contain code blocks, since multiple occurrences of a code block will be concatenated, etc., which is probably not what you want.

Top-level blocks are sorted by their `number` (the number given in `<<* “file” number>>`).

During tangling of a top-level block, the following happens in order:

1. All occurrences of code references `<< … >>` are replaced by the named block. This is done recursively until all code references are eliminated.

2. Escape characters are replaced.

3. The resulting expanded text is written to the file.

## Weaving substitutions

Weaving makes some straightforward substitutions:

```
start of stream -> \glitterStartBook
@: -> \glitterStartText
end of @: block -> \glitterEndText
<<name>>= -> \glitterStartCode{name}
end of <<>>= block -> \glitterEndCode
... << moose >> ... -> \glitterCodeRef{moose}
end of stream -> \glitterEndBook
```

Whitespace at the start of each line in a code block is removed in such a way to shift the code block as far to the left as possible. Blank lines at the start and end of the code block are removed.

Comments of the form `%line N “file”` are included whenever the file switches.

# Roadmap

This is a work in progress. Commits may not compile, and currently it is certainly not usable. Currently, both tangle and weave work, though are not tested in any systematic way.

1. Finish implementation of tangle
2. Options to handle some go-specific things (like automatic insertion of `package` statements)
3. Add configuration files to tweak generation of typeset and code flies.
