package lexer

import (
	"fmt"
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
    const in = `
    @set
        file = foo.cc
        order = 1

    @: Global variables.
    We create a bunch of global variables. Since we're always
    compiling a single thing, we can get away with global state.
    @= Global variables
        std::vector<Tokens> _tokens;
    @:"More globals" are included here:
    @= "Global variables"
        std::list<int> _t;
    @### HEADER
    @set foo = "moo"
    @include a.ww 
    @include b.ww
    @not
    `

    l := New("test.ww", strings.NewReader(in))
    for l.NextToken() {
        err := l.Err()
        if err != nil {
            fmt.Println("foo", err)
            t.Errorf("%v", err)
            return
        }
        tok := l.CurrentToken()
        fmt.Printf("%s:%d:%d %s '%s'\n", tok.Pos.Filename, tok.Pos.Line, tok.Pos.Column, tok.Type, tok.Literal)
    }
}
