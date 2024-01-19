package parser

import (
	"fmt"
	"io"
	"monogrammedchalk.com/glitter/lexer"
	"strings"
)

// Block is one unit of the list of commmands to process
type Block struct {
	// command or speical command "content"
	Type      string
	Arguments []string
	Content   string
	Labels    []string

	Prev *Block
	Next *Block

	token *lexer.Token
}

// debugPrint is used to print out the value of a block for debugging.
func (b *Block) debugPrint() {
	args := strings.Join(b.Arguments, " ")
	fmt.Printf("BLOCK: %s %s\n%s\n", b.Type, args, b.Content)
}

// NewBlock creates a new block of the given type for the given token.
func NewBlock(t string, tok *lexer.Token) *Block {
	return &Block{
		Type:  t,
		token: tok,
	}
}

// addArgument adds s as an argument to the block.
func (b *Block) addArgument(s string) {
	b.Arguments = append(b.Arguments, s)
}

// addLabels adds label s to the block.
func (b *Block) addLabel(s string) {
	b.Labels = append(b.Labels, s)
}

// appendContent appends s to the content block, separated by a space.
func (b *Block) appendContent(s string) {
	b.Content = b.Content + " " + s
}

// isContentBlock returns true if this is a content block.
func isContentBlock(b *Block) bool {
	return b != nil && b.Type == lexer.TOK_CONTENT
}

// supportsLabel returns true if the command supports adding a label to it.
func supportsLabel(b *Block) bool {
	if b == nil {
		return false
	}
	switch b.Type {
	case lexer.CMD_NATURAL:
		return true
	}
	if lexer.FirstRune(b.Type) == lexer.SECTION_CHAR {
		return true
	}
	return false
}

// appendBlock appends a block to the list.
func appendBlock(end, n *Block) *Block {
	if end != nil {
		end.Next = n
	}
	n.Prev = end
	return n
}

// findEnd walks down from a given block to return the last Block on the list.
func findEnd(p *Block) *Block {
	if p == nil {
		return nil
	}
	for p.Next != nil {
		p = p.Next
	}
	return p
}

// deleteBlock removes the node from the list. It returns the new start in case
// we're removing the first node.
func deleteBlock(start, p *Block) *Block {
	if p.Prev != nil {
		p.Prev.Next = p.Next
	} else {
		start = p.Next
	}
	if p.Next != nil {
		p.Next.Prev = p.Prev
	}
	return start
}

// parserError creates and returns a formated error message.
func parserError(tok *lexer.Token, msg string, vargs ...any) error {
	var pstr string
	if tok != nil {
		pstr = fmt.Sprintf("%s:%d:%d:", tok.Pos.Filename, tok.Pos.Line, tok.Pos.Column)
	}
	msgstr := fmt.Sprintf(msg, vargs...)
	return fmt.Errorf("error: %s %s", pstr, msgstr)
}

// debugPrintList prints out all the blocks on the list starting at front.
func debugPrintList(front *Block) {
	for front != nil {
		front.debugPrint()
		front = front.Next
	}
}

// buildList constructs the initial list of blocks from the lexer.
func buildList(lex *lexer.Lexer) (front *Block, end *Block, err error) {
	for lex.NextToken() {
		if lex.Err() != nil {
			err = lex.Err()
			return
		}
		tok := lex.CurrentToken()

		switch tok.Type {

		// @ and var tokens are action blocks. For @ commands, we create a block.
		case lexer.TOK_COMMAND:
			// we filter out set commands, which we don't need now
			if tok.Literal != lexer.CMD_SET {
				end = appendBlock(end, NewBlock(tok.Literal, tok))
			}

		// var tokens have a variable name to define as their literal; we
		// create a node of type TOK_VAR, with that variable name as the first
		// assignment. (The lexer will always produce a STRING token after a VAR token
		// if the input is well formed; that will be added as the second argument.
		case lexer.TOK_VAR:
			end = appendBlock(end, NewBlock(tok.Type, tok))
			end.addArgument(tok.Literal)

		// arguments are added to the active action block
		case lexer.TOK_IDENT, lexer.TOK_STRING:
			if end == nil {
				err = parserError(tok, "argument without command!")
				return
			} else {
				end.addArgument(tok.Literal)
			}

		// adjacent CONTENT blocks are merged
		case lexer.TOK_CONTENT:
			// if the list doesn't end with a content node, append one
			if end == nil || !isContentBlock(end) {
				end = appendBlock(end, NewBlock(tok.Type, tok))
			}
			// add the content to the node
			end.appendContent(tok.Literal)
		}

		if front == nil {
			front = end
		}
	}
	return
}

// Parse is the main entry: it lexes the given stream, creates a list of
// blocks, and then does some simplification of the list. It returns a doubly
// linked list of Blocks.
//
// Parse resolves @label commands, and removes @} variable termination symbols.
// It merges adjacent CONTENT blocks. It removes `@set` commands as well.
//
// At the end of this process, only the blocks of the following types are
// present:
//
//	:
//	=
//	VAR
//	CONTENT
//	@{
//	@<
//	@>
//	@include
//	@scope
//	@ends
//	@(
//	@)
//	@####
//	@preamble
//	@postamble
func Parse(filename string, in io.Reader) (front *Block, end *Block, err error) {
	lex := lexer.New(filename, in)
	front, end, err = buildList(lex)
	if err != nil {
		return
	}
	front, err = assignLabels(front)
	if err != nil {
		return
	}
	front, err = smoothVariableRefs(front)
	if err != nil {
		return
	}
	front = mergeContent(front)
	end = findEnd(front)
	return
}

// assignLabels processes @label commands, asigning the labels to the
// next appropriate block.
func assignLabels(start *Block) (*Block, error) {
	p := start

	// for every block
	for p != nil {
		// for every label
		if p.Type == lexer.CMD_LABEL {
			assigned := false

			// find the next command that can support a label
			n := p.Next
			for n != nil {
				if supportsLabel(n) {
					n.addLabel(p.Arguments[0])
					assigned = true
					break
				}
				n = n.Next
			}

			// if we couldn't assign the label to a block, we stop with a
			// parser error.
			if !assigned {
				return nil, parserError(p.token, "couldn't find block for label `%s`", p.Arguments[0])
			}

			start = deleteBlock(start, p)
		}

		p = p.Next
	}
	return start, nil
}

// mergeContent merges adjacent CONTENT blocks.
func mergeContent(start *Block) *Block {
	p := start
	for p != nil {
		if isContentBlock(p) && isContentBlock(p.Prev) {
			p.Prev.appendContent(p.Content)
			start = deleteBlock(start, p)
		}

		p = p.Next
	}
	return start
}

// smoothVariableRefs deals with the pattern:
//
//	"@{ VAR" CONTENT CONTENT CONTENT @}
//
// where CONTENT is a content block that is all whitespace. This pattern
// is transformed to:
//
//	@{ VAR
//
// If CONTENT is anything but a whitespace content block, a parse error
// is returned.
func smoothVariableRefs(start *Block) (*Block, error) {
	p := start
	for p != nil {
		if p.Type == lexer.CMD_REF_START {

			// walk down the list, deleting empty content nodes until we get to
			// the end or find the @}.
			q := p.Next
			for q != nil && q.Type != lexer.CMD_REF_END {
				// if we're at an empty content node
				if q.Type == lexer.TOK_CONTENT && strings.TrimSpace(q.Content) == "" {
					// delete it
					start = deleteBlock(start, q)
					q = q.Next
				} else {
					// otherwise, we've found the end of the run
					break
				}
			}

			// we ended up at a @} command: delete that command
			if q != nil && q.Type == lexer.CMD_REF_END {
				p = q.Next
				start = deleteBlock(start, q)
			} else {
				// we didn't end at a @}, so this is an error
				return nil, parserError(p.token, "unterminated variable reference")
			}
		} else {
			p = p.Next
		}
	}
	return start, nil
}
