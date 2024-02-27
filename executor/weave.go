package executor

import (
    "io"

	"monogrammedchalk.com/glitter/lexer"
	"monogrammedchalk.com/glitter/parser"
)

// Weave is the main interface that parses a list of blocks and writes out
// natural language explaination to the given out stream. This function does
// some preprocessing, and then calls `weave` (internal version) to do the real
// work.
func Weave(front *parser.Block, out io.Writer) error {
    front, err := replaceVariables(front)
    if err != nil {
        return err
    }

    front, err = moveAmbles(front)
    if err != nil {
        return err
    }

    return weave(front, out)
}

// weave executes the list of commands and writes the result to the given
// stream.
func weave(front *parser.Block, out io.Writer) error {
    stack := newVarStack()
    p := front
    for p != nil {
        p.DebugPrint()
        // if this is not handled by the scopes (@scope, @ends, VAR)
        if stack, handled, err := handleScopes(stack, p); !handled {
            switch p.Type {

            case lexer.CMD_NATURAL: // :
                p, err = weaveNatural(p, stack)
                if err != nil {
                    return err
                }

            case lexer.CMD_CODE: // =
                p, err = weaveCode(p, stack)
                if err != nil {
                    return err
                }

            case lexer.CMD_PREAMBLE, lexer.CMD_POSTAMBLE:
                p, err = weaveAmble(p)
                if err != nil {
                    return err
                }

            case lexer.CMD_SECTION:
                p, err = weaveSection(p)
                if err != nil {
                    return err
                }

            // we don't yet support include
            case lexer.CMD_INCLUDE:
                return notYetImplemented(p)

            // these commands should be handled by one of the handlers above.
            case lexer.CMD_CODENAME_START, lexer.CMD_CODENAME_END,
            lexer.CMD_INLINE_START, lexer.CMD_INLINE_END, lexer.TOK_CONTENT:

                return misplacedCommandError(p)
            }

        } else if err != nil {
            return err
        } else {
            p = p.Next
        }
    }
    return nil
}

// notYetImplemented returns an error saying that the feature is NYI.
func notYetImplemented(p *parser.Block) error {
    return parser.Errorf(p.Token, "not yet implemented: %s", p.Type)
}

// misplacedCommandError returns an error saying that we shouldn't have seen
// this command.
func misplacedCommandError(p *parser.Block) error {
    return parser.Errorf(p.Token, "parser error: command out of place %s", p.Type)
}

func weaveSection(p *parser.Block) (*parser.Block, error) {
    return nil, nil
}

func weaveNatural(p *parser.Block, stack Stack) (*parser.Block, error) {
    return nil, nil
}

func weaveCode(p *parser.Block, stack Stack) (*parser.Block, error) {
    return nil, nil
}

func weaveAmble(p *parser.Block) (*parser.Block, error) {
    return nil, nil
}

// moveAmbles processes all the @preamble and @postamble commands and their
// content to two nodes at the start and end of the list.
func moveAmbles(front *parser.Block) (*parser.Block, error) {
    if front == nil {
        return nil, nil
    }

    preamble := parser.NewBlock(lexer.CMD_PREAMBLE, nil)
    postamble := parser.NewBlock(lexer.CMD_POSTAMBLE, nil)

    var end *parser.Block

    state := "NONE"
    p := front
    for p != nil {
        switch front.Type {
        case lexer.CMD_PREAMBLE:
            state = lexer.CMD_PREAMBLE
            front = parser.DeleteBlock(front, p)

        case lexer.CMD_POSTAMBLE:
            state = lexer.CMD_POSTAMBLE
            front = parser.DeleteBlock(front, p)

        case lexer.TOK_CONTENT:
            if state == lexer.CMD_PREAMBLE {
                preamble.AppendContent(p.Content)
                front = parser.DeleteBlock(front, p)
            } else if state == lexer.CMD_POSTAMBLE {
                postamble.AppendContent(p.Content)
                front = parser.DeleteBlock(front, p)
            } else {
                state = "NONE"
            }
        default: state = "NONE"
        }

        if p.Next == nil {
            end = p
        }
        p = p.Next
    }

    if len(preamble.Content) > 0 {
        preamble.Next = front
        front = preamble
    }

    if len(postamble.Content) > 0 {
        preamble.Prev = end
        end.Next = preamble
    }
    return front, nil
}
