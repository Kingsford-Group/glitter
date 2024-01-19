package parser

import (
    "fmt"
    "strings"
    "testing"
)

func TestParser(t *testing.T) {
    const in = `
    @set
        file = mooose.cc    
    @: "" A section without a header can be written this way.
    @=Great
        v string
        @<rest of global variables@>
        moose
    @:
    Or like this.

    @label foo

    great

    @### Header
    Moose moose @{Order@} moose! are great.
    Really really great.

    @label test
    @: Another block.
    This is a block of text!
    `

    b, _, err := Parse("test.cc", strings.NewReader(in))
    if err != nil {
        fmt.Println(err)
    }
    if b == nil {
        fmt.Println("NIL b")
    } else {
        debugPrintList(b)
    }
}
