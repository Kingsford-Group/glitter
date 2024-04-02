package executor

import (
    "fmt"
    "strings"
    "testing"
    "os"

	"monogrammedchalk.com/glitter/parser"
)

func TestWeave1(t *testing.T) {
    const in = `
    @set file = moose.cc

    @: "A test block." This is a natural language block. 
    That continues on several other lines
    and more lines
    @= Initialization.
        v string
        @<more global vars@>
        x map[string]string
    @:
    Another natural language block.
    @= Init...
        n int

    @### Section Header

    Another block of natural language text

    @label la
    @:
    This is a block of text!
    `
    b, _, err := parser.Parse("test.cc", strings.NewReader(in))
    if err != nil {
        fmt.Println(err)
        return
    }
    if b == nil {
        fmt.Println("NIL b")
        return
    }

    fmt.Println("WEAVING!")

    err = Weave(b, os.Stdout)
    if err != nil {
        fmt.Println(err)
        return
    }
}
