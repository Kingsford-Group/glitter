package main

import (
	"bufio"
    "cmp"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const VERSION_STR = "0.1"

// GlitterOptions stores global options about how to operate.
type GlitterOptions struct {
	Verbose                  int
	WeaveOutFilename         string
	Command                  string
	GivenFiles               []string
	ShowUsage                bool
	DisallowMultipleIncludes bool
}

// Options is a global variable describing how to operation.
var Options GlitterOptions

// LineType is the type of the constants that represent the type of a line.
type LineType int8

// These are the types of lines we can observe
const (
	TextStartLine LineType = iota
	CodeStartLine
	GlitterLine
	OtherLine
)

// FilePos represents a position in a file
type FilePos struct {
	filename string
	lineno   int
}

// SourceLine represents a line in the source files
type SourceLine struct {
	pos  FilePos
	line string
}

// Line returns the string for the line.
func (s *SourceLine) Line() string {
	return s.line
}

// Pos returns the position of the line.
func (s *SourceLine) Pos() FilePos {
	return s.pos
}

// includeRegex matches an include line
var includeRegex = regexp.MustCompile(`^\s*@include\s+"(.+)"\s*$`)

// textStartRegex denotes how a line should begin to start a text block. The :
// is in () so that we have a group, which is required by lineMatchesWithArg.
var textStartRegex = regexp.MustCompile(`^\s*@(:)`)
var codeStartRegex = regexp.MustCompile(`^\s*<<(.+)>>=\s*$`)
var escapeRegex = regexp.MustCompile(`@'(.)`)
var spaceRegexp = regexp.MustCompile(`\s+`)
var topLevelRegex = regexp.MustCompile(`^\*\s*(".*")?\s*(\d+)?\s*$`)
var topLevelStart = regexp.MustCompile(`^\s*\*`)
var glitterRegex = regexp.MustCompile(`^\s*@glitter(\s.*)?$`)

// codeRefRegex matches a reference to a code block. The +? operator means
// match more than one, prefer fewer. This is neded because we may have more
// than one code ref on a single line. Code refs cannot have unescaped >> in
// their label.
var codeRefRegex = regexp.MustCompile(`<<(.+?)>>`)

var errorRecursionTooDeep = fmt.Errorf("include recursion depth exceeds maximum")

// MAX_INCLUDE_DEPTH is the maximum depth of includes that GlitterScanner
// supports.
const MAX_INCLUDE_DEPTH = 20

// Extensions of known file types.
const TANGLE_OUT_EXT = ".go"
const GLITTER_EXT = ".gw"

// Void is an empty struct.
type Void struct{}

// StringSet is a set of strings. No, I don't want to make this generic.
type StringSet struct {
	items map[string]Void
}

// Global item to mark items that are present in the set.
var setMember = Void{}

// NewStringSet creates a new string set.
func NewStringSet() StringSet {
	return StringSet{
		items: make(map[string]Void),
	}
}

// Insert adds a string to the set. It cannot be removed.
func (s *StringSet) Insert(i string) {
	s.items[i] = setMember
}

// Contains return true if a string was previously Inserted.
func (s *StringSet) Contains(i string) bool {
	_, ok := s.items[i]
	return ok
}

// GlitterScanner enables recursively scanning through glitter source files,
// handling @include commands as needed.
type GlitterScanner struct {
	filenames                []string
	stack                    []FilePos
	processedFiles           StringSet
	lines                    chan *SourceLine
	err                      error
	disallowMultipleIncludes bool
}

// NewGlitterScanner creates a GlitterScanner that will read through the given
// files.
func NewGlitterScanner(filenames []string) *GlitterScanner {
	// for each file, read it and put the lines into the output channel
	scanner := GlitterScanner{
		filenames:      filenames,
		stack:          make([]FilePos, 0),
		processedFiles: NewStringSet(),
		lines:          make(chan *SourceLine),
	}
	return &scanner
}

// DisallowMultipleIncludes causes the scanner to skip any file that has
// already been read even if it is included by another file.
func (g *GlitterScanner) DisallowMultipleIncludes() {
	g.disallowMultipleIncludes = true
}

// Lines returns something that can be iterated over, returning successive
// *SourceLine.
func (g *GlitterScanner) Lines() chan *SourceLine {
	go func() {
		for _, f := range g.filenames {
			if err := g.readGlitterSourceFile(f); err != nil {
				g.err = err
				break
			}
		}
		close(g.lines)
	}()
	return g.lines
}

// Err returns the error that stopped the iteration, if any.
func (g *GlitterScanner) Err() error {
	return g.err
}

// CurrentFilePos returns the FilePos object for the file currently being read.
func (g *GlitterScanner) CurrentFilePos() *FilePos {
	return &g.stack[len(g.stack)-1]
}

// Depth returns the current include depth (1= top level)
func (g *GlitterScanner) Depth() int {
	return len(g.stack)
}

// pushFile adds a file to the reading stack.
func (g *GlitterScanner) pushFile(filename string) {
	g.stack = append(g.stack, FilePos{filename: filename, lineno: 0})
}

// popFile removes a file from the reading stack.
func (g *GlitterScanner) popFile() {
	g.stack = g.stack[:len(g.stack)-1]
}

// readGlitterSourceFile reads a file given its filename.
func (g *GlitterScanner) readGlitterSourceFile(filename string) error {
	// do not process a file we have already processed.
	filename = filepath.Clean(filename)
	if g.disallowMultipleIncludes && g.processedFiles.Contains(filename) {
		return nil
	}
	Info(1, "Processing file `%s`", filename)
	in, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer in.Close()
	// remember that we processed this file.
	g.processedFiles.Insert(filename)
	// push file info onto stack
	g.pushFile(filename)
	// recursively read it
	err = g.readGlitterStream(in)
	// pop file info from stack
	g.popFile()
	return err
}

// readGlitterStream reads a stream with source lines in it.
func (g *GlitterScanner) readGlitterStream(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		g.CurrentFilePos().lineno++

		// if this is an include line, recurse
		if include, filename := lineMatchesWithArg(line, includeRegex); include {
			if len(g.stack) >= MAX_INCLUDE_DEPTH {
				return errorRecursionTooDeep
			}
			if err := g.readGlitterSourceFile(filename); err != nil {
				return err
			}
		} else {
			g.lines <- g.newSourceLine(line)
		}
	}
	return scanner.Err()
}

// newSourceLine creates a new source line from the current file (top of the
// stack) and the given string.
func (g *GlitterScanner) newSourceLine(line string) *SourceLine {
	return &SourceLine{
		pos:  *g.CurrentFilePos(),
		line: line,
	}
}

// lineMatchesWithArg returns ok == true, and argument equal to the first
// captured group if the line matches the given regular expression, which must
// include a single () capture group. Otherwise it returns false and ""
func lineMatchesWithArg(line string, re *regexp.Regexp) (bool, string) {
	subs := re.FindStringSubmatch(strings.TrimSpace(line))
	if subs == nil {
		return false, ""
	}
	return true, subs[1]
}

// computeLineType figures out what type the current line is.
func computeLineType(line string) (LineType, string) {
	if m, arg := lineMatchesWithArg(line, textStartRegex); m {
		return TextStartLine, arg
	} else if m, arg := lineMatchesWithArg(line, codeStartRegex); m {
		return CodeStartLine, arg
	} else if m, arg := lineMatchesWithArg(line, glitterRegex); m {
		return GlitterLine, arg
	} else {
		return OtherLine, ""
	}
}

// These constants represent the state of the parser.
const (
	Start int = iota
	InCode
	InText
)

// endBlock writes out the command to end the block according to the state.
func weaveEndBlock(state int, out *bufio.Writer) error {
	var err error
	switch state {
	case InCode:
		_, err = out.WriteString("\\glitterEndCode")
	case InText:
		_, err = out.WriteString("\\glitterEndText")
	}
	return err
}

// removeTextStart removes the text start code from the line.
func removeTextStart(line string) string {
	return textStartRegex.ReplaceAllString(line, "")
}

// weaveCodeRefs replaces a <<foo>> in a line with a call to format the code
// ref.
func weaveCodeRefs(line string) string {
	return codeRefRegex.ReplaceAllString(line, `\glitterCodeRef{$1}`)
}

// replaceEscapes substitutes the escape sequence @'x with x.
func replaceEscapes(line string) string {
	return escapeRegex.ReplaceAllString(line, "$1")
}

// TODO: add config file for specifying text

// Weave creates a typesetable stream, writing it to out.
func Weave(filenames []string, out io.Writer) error {
	w := bufio.NewWriter(out)
	defer w.Flush()

	w.WriteString("\\glitterStartBook\n")

	state := Start
	currentFilename := ""
	// for every source line
	scanner := NewGlitterScanner(filenames)
	for l := range scanner.Lines() {
		if l.Pos().filename != currentFilename {
			currentFilename = l.Pos().filename
			w.WriteString(fmt.Sprintf("%%line %d \"%s\"\n", l.Pos().lineno, currentFilename))
		}
		// depending on what type of line it is:
		t, arg := computeLineType(l.Line())
		switch t {

		// if we're starting a text block
		case TextStartLine:
			err := weaveEndBlock(state, w)
			if err != nil {
				return err
			}
			w.WriteString(`\glitterStartText`)
			w.WriteString(removeTextStart(l.Line()))
			w.WriteString("\n")
			state = InText

		// if we're starting a code block
		case CodeStartLine:
			err := weaveEndBlock(state, w)
			if err != nil {
				return err
			}
			w.WriteString(fmt.Sprintf("\\glitterStartCode{%s}\n", arg))
			InfoWithFile(2, scanner.CurrentFilePos(), "At code block `%s`", arg)
			state = InCode

		case GlitterLine:
			// do nothing

		case OtherLine:
			w.WriteString(replaceEscapes(weaveCodeRefs(l.Line())))
			w.WriteString("\n")
		}
	}

	var err error
	if err = scanner.Err(); err != nil {
		log.Println(err)
	} else {
		err = weaveEndBlock(state, w)
		w.WriteString("\n\\glitterEndBook\n")
	}
	return err
}

// canonicalCodeName converts name to a canonical form, which removes leading
// and trailiing spaces, replaces runs of whitespace with a single space, and,
// if the name does not start with *, it will be all lowercased.
func canonicalCodeName(name string) string {
	name = spaceRegexp.ReplaceAllString(strings.TrimSpace(name), " ")
	if !isTopLevelName(name) {
		name = strings.ToLower(name)
	}
	return replaceEscapes(name)
}

// isTopLevelName returns true if this is a top-level ref, meaning that the code name
// starts with *
func isTopLevelName(name string) bool {
	return topLevelStart.MatchString(name)
}

// parseTopLevelName parses a code block name of the following form:
//
//	<<* "filename" 1234>>
//
// The "filename" and 1234 are both optional, but must be in that order if
// given. If 1234 is omitted, it is 0. If "filename" is omitted, it is
// defaultFile. The "filename" must be contained in quotes.
func parseTopLevelName(name, defaultFile string) (filename string, order int, ok bool) {
	subs := topLevelRegex.FindStringSubmatch(name)
	if subs == nil {
		return
	}
	ok = true
	filename = defaultFile
	for _, g := range subs[1:] {
		if strings.HasPrefix(g, `"`) {
			// filename without the quotes
            filename = filepath.Clean(trimQuotes(g))
		} else {
			o, err := strconv.Atoi(g)
			if err == nil {
				order = o
			}
		}
	}
	return
}

// trimQuotes removes leading and trailing whitespace and a single " from the
// start and end of the string (if they exist)
func trimQuotes(s string) string {
    s = strings.TrimSpace(s)
    s, _ = strings.CutPrefix(s, `"`)
    s, _ = strings.CutSuffix(s, `"`)
    return s
}

// splitTopLevelName splits a well-formed, complete top-level name into its
// components.
func splitTopLevelName(name string) (string, int, error) {
    subs := topLevelRegex.FindStringSubmatch(name)
    if subs == nil || len(subs) != 3 {
        return "", 0, fmt.Errorf("internally incorrectly constructed top-level `%s`", name)
    }
    n, err := strconv.Atoi(subs[2])
    if err != nil {
        return "", 0, fmt.Errorf("internally incorrectly constructed top-level `%s`", name)
    }
    
    return trimQuotes(subs[1]), n, nil
}

// Info prints the message if the verbosity level is level or greater.
func Info(level int, msg string, args ...any) {
	if Options.Verbose >= level {
		log.Printf(msg+"\n", args...)
	}
}

// InfoWithFile prints the message, preceeded by the file and line number, if
// the verbosity level is level or greater.
func InfoWithFile(level int, pos *FilePos, msg string, args ...any) {
	if Options.Verbose >= level {
		log.Printf(fmt.Sprintf("%s:%d: %s\n", pos.filename, pos.lineno, msg), args...)
	}
}

// ErrorWithFile returns a new error that includes the file position.
func ErrorWithFile(pos *FilePos, msg string, args ...any) error {
	return fmt.Errorf(fmt.Sprintf("%s:%d: %s", pos.filename, pos.lineno, msg), args...)
}

// createOutputFilename returns a string with the output filename for the given
// input filename
func createOutputFilename(name string) string {
	name = filepath.Clean(name)
	// remove the suffix if it is present
	name, _ = strings.CutSuffix(name, GLITTER_EXT)
	return name + TANGLE_OUT_EXT
}

// tangleReadBlocks reads all of the given files, recursively including
// @include files and returns a map from code block name to slices of lines.
func tangleReadBlocks(filenames []string) (map[string][]string, error) {
	blocks := make(map[string][]string)

	codeName := ""
	var currentBlock []string

	finalizeBlock := func() {
		if currentBlock != nil {
			blocks[codeName] = currentBlock
			codeName = ""
			currentBlock = nil
		}
	}

	state := Start
	currentFilename := ""
	defaultFilename := ""

	scanner := NewGlitterScanner(filenames)
	for l := range scanner.Lines() {
		// if we're reading a top-level file, make sure the default filename
		if l.Pos().filename != defaultFilename && scanner.Depth() == 1 {
			defaultFilename = createOutputFilename(l.Pos().filename)
		}
		t, arg := computeLineType(l.Line())
		switch t {

		case TextStartLine:
			finalizeBlock()
			state = InText

		case CodeStartLine:
			finalizeBlock()
			state = InCode

			codeName = canonicalCodeName(arg)
			// if this looks like a top-level reference, parse it
			if isTopLevelName(codeName) {
				filename, order, ok := parseTopLevelName(codeName, currentFilename)
				if !ok {
					return nil, ErrorWithFile(
						scanner.CurrentFilePos(),
						"badly formated top-level name `%s`",
						codeName,
					)
				}
				// if the filename is empty or a single ., then switch back to
				// the main output file. Note that filepath.Clean converts an
				// empty filename to a single ., so we only have to check that.
				if filename == "." {
					filename = defaultFilename
				}
				currentFilename = filename
				codeName = fmt.Sprintf("* \"%s\" %d", currentFilename, order)
			}
			InfoWithFile(2, scanner.CurrentFilePos(), "At code block `%s`", codeName)

			// get the block if it already exists
			if b, ok := blocks[codeName]; ok {
				currentBlock = b
			} else {
				currentBlock = make([]string, 0)
			}

		case GlitterLine:
			defaultFilename = createOutputFilename(l.Pos().filename)
			currentFilename = defaultFilename

		case OtherLine:
			if state == InCode {
				currentBlock = append(currentBlock, l.Line())
			}
		}
	}

	var err error
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	finalizeBlock()
	return blocks, err
}

// getTopLevelBlocks returns a list of the names of all the top-level blocks.
func getTopLevelBlocks(blocks map[string][]string) (out []string, err error) {
    out = make([]string, 0)
    for k := range blocks {
        if isTopLevelName(k) {
            out = append(out, k)
        }
    }

    // if the comparison function panics, catch the error and return in in the
    // normal way.
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("cmp error: %v", r)
        }
    }()
    slices.SortFunc(out, func(a,b string) int {
        fa, na, err := splitTopLevelName(a)
        if err != nil {
            panic(err)
        }
        fb, nb, err := splitTopLevelName(b)
        if err != nil {
            panic(err)
        }
        if n := cmp.Compare(fa, fb); n != 0 {
            return n
        }
        return cmp.Compare(na, nb)
    })
    return
}

// expandAndWriteBlock expands all << >> refs in a code block and writes the
// block to the given stream.
func expandAndWriteBlock(b []string, out *bufio.Writer) error {
    // TODO: write me!
    for _, line := range b {
        pos := codeRefRegex.FindStringSubmatchIndex(line)
        if pos == nil {
            out.WriteString(line)
            out.WriteString("\n")
        } else {
            blockName := strings.TrimSpace(line[pos[2]:pos[3]])
            fmt.Println("blockName=", blockName)
        }
    }
    return nil

}

// Tangle produces a set of source code files that can be compiled into the
// described program or library.
func Tangle(filenames []string) error {
	// read all the blocks into memory
	blocks, err := tangleReadBlocks(filenames)
	if err != nil {
		return err
	}

    topBlocks, err := getTopLevelBlocks(blocks)
    if err != nil {
        return err
    }
    if len(topBlocks) == 0 {
        return fmt.Errorf("no top-level code blocks found")
    }

    Info(2, "%d total top-level blocks found", len(topBlocks))

    currentFilename := ""
    var curOut *os.File
    var curBuff *bufio.Writer
    // need this func() indirection so the current version of curOut is closed
    defer func () {
        if curBuff != nil {
            curBuff.Flush()
        }
        if curOut != nil {
            curOut.Close()
        }
    }()

    // go through each top level block
    for _, b := range topBlocks {
        f, _, err := splitTopLevelName(b)
        if err != nil {
            return err
        }

        // if we are starting a new file, create the new output file
        if f != currentFilename {
            if curBuff != nil {
                curBuff.Flush()
            }
            if curOut != nil {
                curOut.Close()
            }
            curOut, err = os.Create(f)
            if err != nil {
                return err
            }
            curBuff = bufio.NewWriter(curOut)
            currentFilename = f
        }
        err = expandAndWriteBlock(blocks[b], curBuff)
        if err != nil {
            return err
        }
    }

	// DEBUG: the para below is just for debugging
	debugPrintBlocks(blocks, os.Stderr)
	return err
}

// whiteSpacePrefixLength returns the number of whitespace runes that prefix
// the string.
func whitespacePrefixLength(line string) int {
	for i, c := range line {
		if !unicode.IsSpace(c) {
			return i
		}
	}
	return utf8.RuneCountInString(line)
}

// deindentBlock finds the leftmost start point of a line removes whitespace
// before that point.
func deindentBlock(block []string) []string {
	minSpace := -1
	for _, line := range block {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		if minSpace < 0 {
			minSpace = len(line)
		}
		minSpace = min(minSpace, whitespacePrefixLength(line))
	}
	if minSpace < 0 {
		return block
	}
	out := make([]string, len(block))
	for i, line := range block {
		rl := []rune(line)
		if len(rl) < minSpace {
			out[i] = line
		} else {
			out[i] = string([]rune(line)[minSpace:])
		}
	}
	return out
}

// debugPrintBlocks writes all the blocks out in a simple format.
func debugPrintBlocks(blocks map[string][]string, out io.Writer) {
	for n, c := range blocks {
		fmt.Fprintf(out, "<<%s>>= {\n", n)
		for _, ll := range deindentBlock(c) {
			fmt.Fprintf(out, "\t%s\n", ll)
		}
		fmt.Fprintln(out, "}")
	}
}

// hasGlitterProp returns true if the first non-empty line in the given file is
// a @glitter line that contains the word given by property. If there is any
// error reading the file, we return false.
func hasGlitterProp(filename, property string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		subs := glitterRegex.FindStringSubmatch(line)
		// non-empty, but not @gitter line or a gitter line with no props
		if len(subs) <= 1 {
			return false
		}

		for _, p := range strings.Fields(subs[1]) {
			if p == property {
				return true
			}
		}
		return false
	}
	return false
}

// findTopFiles searches for top-level files. If filename exists but is not a
// directory, then it is a top-level file and the only file returned. If it is
// a directory, then we walk the tree rooted at that directory looking for
// files that end with GLITTER_EXT and that contain a `@glitter top` line as
// their first non-empty line.
func findTopFiles(filename string) ([]string, error) {
	filename = filepath.Clean(filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	if stat.IsDir() {
		err := filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && filepath.Ext(d.Name()) == GLITTER_EXT {
				if hasGlitterProp(d.Name(), "top") {
					out = append(out, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		out = append(out, filename)
	}
	return out, nil
}

// findTangleFiles creates a list of files to tangle. Non-directories are added to the
// list directly. Directories are searched using findTopFiles. The list of files is
// de-duped and sorted.
func findTangleFiles(filenames []string) ([]string, error) {
	out := make([]string, 0)
	for _, f := range filenames {
		list, err := findTopFiles(f)
		if err != nil {
			return nil, err
		}
		out = append(out, list...)
	}
	sort.Strings(out)
	fmt.Println(out)
	return slices.Compact(out), nil
}

// printBanner prints a 1 line name/version info to os.Stderr.
func printBanner() {
	fmt.Fprintf(os.Stderr, "glitter version %s (c) 2024 Carl Kingsford.\n", VERSION_STR)
}

// printUsage prints a 1 line usage help and then info about the command line
// options to os.Stderr.
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: glitter [options] [weave|tangle] file...")
	flag.PrintDefaults()
}

// init sets up the command line processing.
func init() {
	flag.IntVar(&Options.Verbose, "v", 0, "how much info to print")
	flag.StringVar(&Options.WeaveOutFilename, "out", "default.tex", "output for weave command")
	flag.BoolVar(&Options.ShowUsage, "h", false, "show usage and quit")
	flag.BoolVar(&Options.DisallowMultipleIncludes, "forbid-multiple-includes", false, "read every file only once")
}

func main() {
	log.SetPrefix("glitter: ")
	log.SetFlags(0)

	printBanner()

	flag.Parse()
	if Options.ShowUsage || len(flag.Args()) < 2 {
		printUsage()
		os.Exit(0)
	}
	Options.Command = flag.Arg(0)
	Options.GivenFiles = flag.Args()[1:]

	var err error
	switch Options.Command {
	case "weave":
		f, err := os.Create(Options.WeaveOutFilename)
		if err == nil {
			err = Weave(Options.GivenFiles, f)
			f.Close()
		}

	case "tangle":
		files, err := findTangleFiles(Options.GivenFiles)
		if err == nil {
			err = Tangle(files)
		}

	default:
		log.Printf("unknown command `%s`\n", Options.Command)
		os.Exit(1)
	}

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
