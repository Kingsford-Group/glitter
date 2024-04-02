# Glitter Syntax

## Blocks

A **text block** starts with `^\s*@:`. Everything following the `:` is placed into the natural language file, in the order they appear in the file.

A **code block** starts with a line of the form `^\s*<<.+>>=\s*$`.  The string between  `<<`…`>>` is `S`. Leading and trailing whitespace is removed from `S`. If `S` is of the form `\*\s*f\s+n`, where `f` is a string and `n` is an integer, then this is a top-level block that will be written to file `f` in order given by `n`. If `n` is omitted, it is assumed to be 0. If `f` is omitted, it is the last `f` that occurred in some code block name. If `S` is any other form, it defines a named code block.

Code blocks with the same canonical name are concatenated in the order they appear in the file.

A block ends when another block starts or the file ends.

A source file must consist of alternating text and code blocks. 

A string in a code block of the form `<<.+>>` is a substitution call. It is recursively replaced by the code in code block with the same canonical name as given between the `<<`…`>>`. If non-whitespace characters follow `>>` on the same line as a substitution, trailing newline characters in the code referenced code block are removed before substitution.

**Canonical names** names are formed by:

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
```

If the number is omitted, it is assumed to be 0. If the filename is omitted, it is the last named file read or the default output file if no previous named file was given. If the filename is empty `“”` then it is the default output file.

This rule means that filenames are “sticky” within a top-level file (and its includes):

```
<<* “constants.go”>>=

<<*>>= // writes to constants.go
```

The current output file (and the default output file) are both reset when a new top-level file is read or when a `@glitter top` line is found.

**Marking a file as top-level**. A line, occurring as the first non-blank line in a file, of the form `@glitter top` marks the file as a top level file to be found by the tangle command when searching directories.

When weaving, the effect of a `@glitter top` line is to set the current and default output files to be the same name as the source file with its extension changed.

## Escapes

Anyplace in the file, a sequence `@'x` is treated as a single occurrence of `x`. For example, `@'@` is a single `@` and `@'<` is a single `<`. Occurrences of `@` and `<` that can’t be interpreted as one of the forms above do not need to be escaped.

Note that this rule is applied as late as possible in the processing. So ``<<foo @'>> moose >>`` defines a reference to `foo @\'` since the first `>>` still exists in the file. To include, a literal `>>`, one would say `>@'>`, since that causes `>>` to not be present.

## Includes

A line matching with `^\s*@include\s+".+"$` is an include line. It is replaced by the contents of the file named between the `“` .

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

For each directory, the tree rooted at that directory will be scanned for all files ending with `.gw`. If the first non-blank line in  a file that is found is `@glitter top` then the file will be added to the list of files to process.

If a file is given explicitly, then it is always added to the list of files that are processed.

The list is sorted lexicographically, and duplicate files are removed. The files are then processed in order. Let $F_1,F_2,\dots,F_k$ be the files.

When processing $F_i$ *and all the files included from it*, the default output file is $F_i$​ with its extension replaced by `.go`. 

If you want to tangle only given files, you can list them on the command line. Such explicitly listed files do not need to start with `@glitter top` to be processed (but they can).

If you want to tangle all `@glitter top` files in a tree, you can give the root of the tree.

If you want to create a top-level file that specifies what to tangle you can do so (as in the weave example above), so long as each of the included files is marked as `@glitter top`. Then you simply `glitter tangle topfile.gw`.

All files are read before any output is produced. 

If you give the `-forbid-multi-includes` option, then no file will be included only once, no matter how many times it’s encountered (whether it is read via an `@include` or from the list of files). Normally, you can include the same file more than once, but then they are treated exactly like processing the same file multiple times, and so should probably not contain code blocks, since multiple occurrences of a code block will be concatenated, etc., which is probably not what you want…

Code blocks with the same name are concatenated in the order they are encountered in the stream.

Top-level blocks are sorted by their `number` (the number given in `<<* “file” number>>`).

During tangling of a top-level block, the following happens in order:

1. All occurrences of code references `<< … >>` are replaced by the named block. This is done recursively until all code references are eliminated.

2. Escape characters are replaced.

## Weaving substitutions

Weaving makes some straightforward substitutions:

```
@: -> \glitterStartText
\glitterEndText

<<foo>>= ->
\glitterStartCode{name}
\glitterEndCode

... << moose >> ... ->
\glitterCodeRef{moose}
```

## Notes:



Issue: when walking a directory, need to know which files are supposed to be processed as a top-level file

Solutions:

* read the files, find the includes
* use a different suffix like .gwi
* label files with `@include_only` to skip reading unless the depth is > 1
* label files with `@glitter tags=…. ver=0.1`
* `@tangle`
* `@glitter tangle`
* a makefile like file







