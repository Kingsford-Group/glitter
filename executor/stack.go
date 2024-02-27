package executor

import (
    "fmt"

   	"monogrammedchalk.com/glitter/lexer"
	"monogrammedchalk.com/glitter/parser"
)

// Stack is a variable stack with multiple frames.
type Stack []map[string]string

// newVarStack creates a new variable stack. 
func newVarStack() Stack {
    var s Stack
    return pushStackFrame(s)
}

// pushStackFrame creates a new, empty stack frame on the stack.
// You must use it like Go's append:
//
//   stack = pushStackFrame(stack)
//
// since it may have to move the stack to allocate a frame.
func pushStackFrame(stack Stack) Stack {
    return append(stack, make(map[string]string))
}

// popStackFrame removes the topmost stack frame on the stack; throws an error
// if the stack is empty. You must use it like Go's append:
//
//   stack, err = popStackFrame(stack)
//
func popStackFrame(stack Stack) (Stack, error) {
    if len(stack) == 0 {
        return stack, fmt.Errorf("unexpected end of scope")
    }
    return stack[:len(stack)-1], nil
}

// defineVariable adds the definition of a variable to the topmost stack frame,
// shadowing previous definitions.
func defineVariable(stack Stack, name, value string) error {
    if len(stack) == 0 {
        return fmt.Errorf("variable defined outside of any scope")
    }
    stack[len(stack)-1][name] = value
    return nil
}

// varValue returns the value of a variable. The value is the value in the
// topmost stack frame. Returns an error if no definition of the variable is
// found.
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

// handleScopes handles commands related to scopes and defining variables. This
// is used when processing command lists to take care of scope-related
// commands.
// 
// Returns the new stack, and the second return value is `true` if the command
// p was handled. Throws an error if a non-existant scope is encountered.
func handleScopes(stack Stack, p *parser.Block) (Stack, bool, error) {
    switch p.Type {
    case lexer.CMD_SCOPE_START:
        stack = pushStackFrame(stack)
        return stack, true, nil

    case lexer.CMD_SCOPE_END:
        stack, err := popStackFrame(stack)
        if err != nil {
            return stack, true, parser.Errorf(p.Token, "%v", err)
        }
        return stack, true, nil

    case lexer.TOK_VAR:
        err := defineVariable(stack, p.Arguments[0], p.Arguments[1])
        if err != nil {
            return stack, true, parser.Errorf(p.Token, "%v", err)
        }
        return stack, true, nil
    }
    return stack, false, nil
}

// replaceVariables parses the list, replacing variable references with CONTENT
// blocks that contain their value. 
func replaceVariables(front *parser.Block) (*parser.Block, error) {
    stack := newVarStack()
    p := front
    for p != nil {
        // if this is not handled by the scopes
        if stack, handled, err := handleScopes(stack, p); !handled {

            switch p.Type {
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
        } else if err != nil {
            return nil, err
        }
        p = p.Next
    }
    return front, nil
}

