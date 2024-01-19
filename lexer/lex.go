package lexer

import (
	"bufio"
    "fmt"
	"io"
	"strings"
	"unicode"
    "text/scanner"
    "unicode/utf8"
)

// A token represents an object identified by the lexer.
type Token struct {
    Type string
    Literal string
    Pos scanner.Position
}

func (t *Token) DebugPrint() {
    fmt.Printf("%s:%d:%d: %s '%s'\n", t.Pos.Filename, t.Pos.Line, t.Pos.Column, t.Type, t.Literal) 
}

// Every token has a type drawn from one of these:
const (
    // TOK_EOF is an end of file marker.
    TOK_EOF      string  = "EOF"

    // TOK_COMMAND is an @ command.
    TOK_COMMAND          = "COMMAND"

    // TOK_INDENT is a identifier
    TOK_IDENT            = "IDENT"

    // TOK_STRING is an argument to a command or variable
    TOK_STRING           = "STRING"

    // TOK_STRING is a string of cotent (either code or natural
    // language) that does not include any commands.
    TOK_CONTENT          = "CONTENT"

    TOK_VAR              = "VAR"
)

// These are syntax elements.
const (
    NEWLINE       rune    = '\n'
    CMD_CHAR      rune    = '@'
    SECTION_CHAR  rune    = '#'
    LITERAL_CHAR  rune    = '\''
    QUOTE_CHAR    rune    = '"'
    ASSIGN_CHAR   rune    = '='

    CMD_NATURAL   string    = ":"
    CMD_CODE      string    = "="
    CMD_INCLUDE   string  = "include"
    CMD_LABEL     string  = "label"
    CMD_SET       string  = "set"
    CMD_NEXT      string  = "next"
    CMD_SCOPE_START string = "scope"
    CMD_SCOPE_END string = "ends"
    CMD_C         string = "c"
    CMD_COMMENT   string = "comment"
    CMD_COMMENT_END string = "endc"
    CMD_REF_START string  = "{"
    CMD_REF_END   string  = "}"
    CMD_CODENAME_START string = "<"

    CMD_PREAMBLE string = "preamble"
    CMD_POSTAMBLE string = "postamble"
    
    COMMAND_SYMS  string  = ":='<>(){}"
)

// isCommandStr returns true iff s is a string that spells out a valid command
func isCommandStr(s string) bool {

    if len(s) == 0 {
        return false
    }

    // multi character commands
    switch (s) {
    case CMD_INCLUDE, CMD_LABEL, CMD_SET, CMD_NEXT, CMD_SCOPE_START, CMD_SCOPE_END: return true
    }

    // single character commands
    if len(s) == 1 && strings.ContainsRune(COMMAND_SYMS, FirstRune(s)) {
        return true
    }

    // section commands consist of # repeated one or more times.
    if FirstRune(s) == SECTION_CHAR {
        for _, c := range s {
            if c != SECTION_CHAR {
                return false
            }
        }
        return true
    }
    return false
}

// FirstRune reads the first UTF8 rune from the string.
func FirstRune(s string) rune {
    r, _ := utf8.DecodeRuneInString(s)
    return r
}

// Lexer represents the saved state of a lexing process.
type Lexer struct {
    stream *bufio.Reader
	ch     rune
	err    error
    pos    scanner.Position

    currentToken *Token
    nextToken  *Token
    nextErr error

    mode   int
    atEOF  bool
}

// New creates a new lexer that will parse a web stream from reader f.
func New(filename string, f io.Reader) *Lexer {
	l := Lexer{
		stream: bufio.NewReader(f),
		ch:     0,
		err:    nil,
        pos: scanner.Position{filename, 0,1,1},

        mode: MODE_NONE,
	}
	l.nextRune()
	return &l
}

// setError sets the error code to e if the error hasn't been set already.
func (l *Lexer) setError(e error) {
    if l.err == nil {
        l.err = e
    }
}

// clearError resets the error. Called implicity every NextToken().
func (l *Lexer) clearError() {
    l.err = nil
}

// lexError creates an error with the line, column number, etc.
func (l *Lexer) lexError(msg string, varg ...any) error {

    locstr := fmt.Sprintf("%s:%d:%d:", l.pos.Filename, l.pos.Line, l.pos.Column)
    msgstr := fmt.Sprintf(msg, varg...)

    err := fmt.Errorf("%s %s",locstr, msgstr)
    l.setError(err)
    fmt.Println("error:", err)
    return l.err
}

// Err returns the last recorded error.
func (l *Lexer) Err() error {
	return l.err
}

// nextRune reads the next rune from the buffered stream. It returns true if we
// succeed; if so, curRune() contains the next rune otherwise Err() will be
// non-nil.
func (l *Lexer) nextRune() bool {
	ch, _, err := l.stream.ReadRune()
	if err != nil {
        if err == io.EOF {
            l.atEOF = true
        } else {
            l.setError(err)
        }
		return false
	}

	l.ch = ch
	l.pos.Column++

	if l.ch == NEWLINE {
		l.pos.Line++
		l.pos.Column = 1
	}
	return true
}

// curRune returns the rune that was last read by nextRune.
func (l *Lexer) curRune() rune {
	return l.ch
}

// skipWhiteSpace skips until the current rune is a non-whitespace character.
func (l *Lexer) skipWhitespace() error {
	first := true
	for first || l.nextRune() {
		if !unicode.IsSpace(l.curRune()) {
			return l.Err()
		}
		first = false
	}
	return l.Err()
}

// skipWhitespaceOnLine skips whitespace up to the next non-space character or
// the next newline, whichever is first.
func (l *Lexer) skipWhitespaceOnLine() error {
	first := true
	for first || l.nextRune() {
        if l.curRune() == NEWLINE {
            return nil
        }
		if !unicode.IsSpace(l.curRune()) {
			return l.Err()
		}
		first = false
	}
	return l.Err()
}

// readQuoteString reads the quoted string. It assumes that the current rune is
// *not* part of the string (e.g. it is the opening ") and it will not include
// terminating " in the returned string. On error, the string will be nonsense.
// It consumes the final ".
//
// TODO: must implement escapes
func (l *Lexer) readQuoteString() (string, error) {
	b := make([]rune, 0)

	for l.nextRune() {
		if l.curRune() == QUOTE_CHAR {
			l.nextRune()
			return string(b), l.Err()
		} else {
			b = append(b, l.curRune())
		}
	}
	return "", l.Err()
}

// readImplictString reads a string that goes from a non-space character until
// the end of the line.
//
// TODO: must implement escapes
func (l *Lexer) readImplictString() (string, error) {
    b := []rune{l.curRune()}

    for l.nextRune() {
        if l.curRune() == NEWLINE {
            return strings.TrimSpace(string(b)), l.Err()
        } else {
            b = append(b, l.curRune())
        }
    }

    return "", l.Err()
}

// readString reads the next string, automatically determining if it's a Quote
// string or an Implicit String. 
func (l *Lexer) readString() (string, error) {
    if err := l.skipWhitespaceOnLine(); err != nil {
        return "", err
    }

    // if there is no character until the next NEWLINE, then the string is empty
    if l.curRune() == NEWLINE {
        l.nextRune()
        return "", nil
    }

    if l.curRune() == '"' {
        return l.readQuoteString()
    }

    return l.readImplictString()
}

// readIdent reads an identifier which is a continuous string of upper and
// letters.
func (l *Lexer) readIdent() (string, error) {
    err := l.skipWhitespace()
    if err != nil {
        return "", err
    }

	b := make([]rune, 0)

    for {
        if !unicode.IsLetter(l.curRune()) {
            if len(b) == 0 {
                l.lexError("empty identifier")
            }
            return string(b), l.Err()
        }
        b = append(b, l.curRune())
        if !l.nextRune() { break }
    }

	return "", l.Err()
}

// readSectionCommand reads a string of #s.
func (l *Lexer) readSectionCommand() (string, error) {
    b := []rune{l.curRune()}
    for l.nextRune() {
        if l.curRune() == SECTION_CHAR {
            b = append(b, l.curRune())
        } else {
            return string(b), l.Err()
        }
    }

    return "", l.Err()
}

// readEscapeSeq reads a command of the form 'c where c is a single character.
func (l *Lexer) readEscapeSeq() (string, error) {
    b := []rune{l.curRune()}
    if l.nextRune() {
        b = append(b, l.curRune())
        return string(b), l.Err()
    }

    return "", l.Err()
}

// readCommand reads a command string, which is one of the following forms:
//
//    - [:=<>(){}']  -- a single character from this set
//    - #####        -- a run of #
//    - uperAndLower -- a stretch of upper and lowercase letters
//    - 'c           -- where c is any character
func (l *Lexer) readCommand() (string, error) {
    b := []rune{l.curRune()}
    if strings.ContainsRune(COMMAND_SYMS, l.curRune()) {
        l.nextRune()
       return string(b), l.Err() 
    }

    switch l.curRune() {
    case SECTION_CHAR: 
        return l.readSectionCommand()
    case LITERAL_CHAR: 
        return l.readEscapeSeq()
    default:
        return l.readIdent()
    }
}

// readContent reads until the next command or EOF.
func (l *Lexer) readContent() (string, error) {
    b := []rune{l.curRune()}
    
    for l.nextRune() {
        if l.curRune() == CMD_CHAR {
            return string(b), nil
        } else {
            b = append(b, l.curRune())
        }
    }
    return string(b), nil
}

// readAssignOp reads up to just past the next ASSIGN_CHAR (=), which may be preceeded by
// whitespace.
func (l *Lexer) readAssignOp() (string, error) {
    l.skipWhitespace()
    if l.curRune() != ASSIGN_CHAR {
        return "", l.lexError("expected assignment operator (%c), got `%c`", ASSIGN_CHAR, l.curRune())
    }
    l.nextRune()
    return string(ASSIGN_CHAR), nil
}

// newToken creates a new Token object.
func (l *Lexer) newToken(t, val string) *Token {
    return &Token{
        Type: t,
        Literal: val,
        Pos: l.pos,
    }
}

// These are the modes the lexer can be in: either we are reading content,
// which is any non-command and non Var=Value pairs, or we are reading a list
// of Var = "Value" pairs. All other states are handled by doing a lookahead
// for at most 2 tokens (2 for Var="Value" pairs, and 1 for all ""
// arguments).
const (
    MODE_CONTENT = iota + 1
    MODE_SET
    MODE_NONE
)

// NextToken returns the next token.
func (l *Lexer) NextToken() bool {

    // reset the errors for this token
    l.clearError()

    // if we have a cached token, return it and reset it.
    if l.nextToken != nil {
        l.currentToken = l.nextToken
        l.err = l.nextErr
        l.nextToken = nil
        l.nextErr = nil
        return true
    }

    // if we've been marked as at the end, we just return an EOF token forever.
    if l.atEOF {
        l.currentToken = l.newToken(TOK_EOF, "")
        return false
    }

    // in NONE mode, only whitespace is allowed, so we eat it up
    if l.mode == MODE_NONE {
        // mode NONE allows only whitespace charaqcters. 
        if !unicode.IsSpace(l.curRune()) {
            l.lexError("non-whitespace (%c) in forbidden location", l.curRune())
            return false
        }
        err := l.skipWhitespace()
        if err != nil {
            return false
        }
    }

    switch (l.curRune()) {

    // if we start a command
    case CMD_CHAR:
        l.nextRune()
        // read the command
        s, err := l.readCommand()
        if err != nil {
            l.lexError("error: %v", err)
            return false
        }

        // commands are case insensitive:
        s = strings.ToLower(s)

        if !isCommandStr(s) {
            l.lexError("unknown command `@%s`", s)
            return false
        }

        // switch to the mode that should follow this command.
        if l.mode == MODE_SET {
            l.mode = MODE_NONE
        }
        switch (s) {
        case CMD_SET, CMD_NEXT: l.mode = MODE_SET
        case CMD_NATURAL, CMD_CODE, CMD_CODENAME_START: l.mode = MODE_CONTENT
        }
        if FirstRune(s) == '#' {
            l.mode = MODE_CONTENT
        }
        // A { or label command expects an identifier next
        if s == CMD_REF_START || s == CMD_LABEL {
            ns, err := l.readIdent()
            l.nextToken = l.newToken(TOK_IDENT, ns)
            l.nextErr = err
        }

        // for any command that expects a string, read the string. We know that
        // length(s) > 0 because readCommand() returns an error if we end up
        // with an empty string.
        if s == CMD_NATURAL || s == CMD_CODE || FirstRune(s) == SECTION_CHAR || s == CMD_INCLUDE {
            ns, err := l.readString();
            l.nextToken = l.newToken(TOK_STRING, ns)
            l.nextErr = err
        }
        
        l.currentToken = l.newToken(TOK_COMMAND, s)

    default:
        switch l.mode {

        // we're reading content (meaning either natural language or code)
        case MODE_CONTENT: 
            s, err := l.readContent()
            if err != nil {
                return false
            }
            l.currentToken = l.newToken(TOK_CONTENT, s)

        // we're reading var="value" pairs of a @set block.
        case MODE_SET:
            // read a var 
            v, err := l.readIdent()
            if err != nil {
                return false
            }
            l.currentToken = l.newToken(TOK_VAR, v)

            // read a =, return an error if the next token is not a =, and then
            // discard the =
            _, err = l.readAssignOp()
            if err != nil {
                return false
            }

            // read the value
            s, err := l.readString()
            l.nextToken = l.newToken(TOK_STRING, s)
            l.nextErr = err

            // skip past any whitespace
            err = l.skipWhitespace()
            if err != nil {
                return false
            }

        }
    }

    return true
}

// Return the current token.
func (l *Lexer) CurrentToken() *Token {
    return l.currentToken
}


