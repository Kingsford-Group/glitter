package executor

import (
    "io"
    "fmt"

	"monogrammedchalk.com/glitter/lexer"
	"monogrammedchalk.com/glitter/parser"
)

func Weave(front *parser.Block, out io.Writer) error {
    front, err := replaceVariables(front)
    if err != nil {
        return err
    }

    front, err = moveAmbles(front)
    if err != nil {
        return err
    }

    // for every block, execute it
    return nil
}

type Stack []map[string]string

func newVarStack() Stack {
    var s Stack
    return pushStackFrame(s)
}

func pushStackFrame(stack Stack) Stack {
    return append(stack, make(map[string]string))
}

func popStackFrame(stack Stack) (Stack, error) {
    if len(stack) == 0 {
        return stack, fmt.Errorf("unexpected end of scope")
    }
    return stack[:len(stack)-1], nil
}

func defineVariable(stack Stack, name, value string) error {
    if len(stack) == 0 {
        return fmt.Errorf("variable defined outside of any scope")
    }
    stack[len(stack)-1][name] = value
    return nil
}

func varValue(stack Stack, name string) (string, error) {
    if len(stack) == 0 {
        return "", fmt.Errorf("unknown variable `%s`", name)
    }
    for i := len(stack)-1; i > 0; i-- {
        if val, ok := stack[i][name]; ok {
            return val, nil
        }
    }
    return "", fmt.Errorf("unknown variable `%s`", name)
}

func replaceVariables(front *parser.Block) (*parser.Block, error) {
    var err error
    stack := newVarStack()
    p := front
    for p != nil {
        switch p.Type {
        case lexer.CMD_SCOPE_START:
            stack = pushStackFrame(stack)

        case lexer.CMD_SCOPE_END:
            stack, err = popStackFrame(stack)
            if err != nil {
                return nil, parser.Errorf(p.Token, "%v", err)
            }

        case lexer.TOK_VAR:
            err := defineVariable(stack, p.Arguments[0], p.Arguments[1])
            if err != nil {
                return nil, parser.Errorf(p.Token, "%v", err)
            }

        // if this is a reference to a variable, look up the variable, and
        // replace this block with a content block that contains that value.
        case lexer.CMD_REF_START:
            // look up the variable value
            varname := p.Arguments[0]
            val, err := varValue(stack, varname)
            if err != nil {
                return nil, parser.Errorf(p.Token, "%v", err)
            }

            // create the new content node
            b := parser.NewBlock(lexer.TOK_CONTENT, p.Token)
            b.Content = val

            // splice it in
            b.Next = p.Next
            b.Prev = p.Prev
            if p.Prev == nil {
                front = b
            }
        }
        p = p.Next
    }
    return front, nil
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
