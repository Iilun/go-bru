package bru

import (
	"strconv"
	"sync"
)

// Bru value parser state machine.
// Just about at the limit of what is reasonable to write by hand.
// Some parts are a bit tedious, but overall it nicely factors out the
// otherwise common code from the multiple scanning functions
// in this package (Compact, Indent, checkValid, etc).
//
// This file starts with two simple examples using the scanner
// before diving into the scanner itself.

// Valid reports whether data is a valid Bru file.
func Valid(data []byte) bool {
	scan := newScanner()
	defer freeScanner(scan)
	return checkValid(data, scan) == nil
}

// checkValid verifies that data is valid Bru-encoded data.
// scan is passed in for use by checkValid to avoid an allocation.
// checkValid returns nil or a SyntaxError.
func checkValid(data []byte, scan *scanner) error {
	scan.reset()
	for _, c := range data {
		scan.bytes++
		if scan.step(scan, c) == scanError {
			return scan.err
		}
	}
	if scan.eof() == scanError {
		return scan.err
	}
	return nil
}

// A SyntaxError is a description of a Bru syntax error.
// Unmarshal will return a SyntaxError if the Bru can't be parsed.
type SyntaxError struct {
	msg    string // description of error
	Offset int64  // error occurred after reading Offset bytes
}

func (e *SyntaxError) Error() string { return e.msg }

// A scanner is a Bru scanning state machine.
// Callers call scan.reset and then pass bytes in one at a time
// by calling scan.step(&scan, c) for each byte.
// The return value, referred to as an opcode, tells the
// caller about significant parsing events like beginning
// and ending literals, objects, and arrays, so that the
// caller can follow along if it wishes.
// The return value scanEnd indicates that the scan ended successfully
type scanner struct {
	// The step is a func to be called to execute the next transition.
	// Also tried using an integer constant and a single func
	// with a switch, but using the func directly was 10% faster
	// on a 64-bit Mac Mini, and it's nicer to read.
	step func(*scanner, byte) int

	// Reached end of top-level value.
	endBlock bool

	// Stack of what we're in the middle of - array values, object keys, object values.
	parseState []int

	// Error that happened, if any.
	err error

	// total bytes consumed, updated by decoder.Decode (and deliberately
	// not set to zero by scan.reset)
	bytes   int64
	tagName []byte
}

var scannerPool = sync.Pool{
	New: func() any {
		return &scanner{}
	},
}

func newScanner() *scanner {
	scan := scannerPool.Get().(*scanner)
	// scan.reset by design doesn't set bytes to zero
	scan.bytes = 0
	scan.reset()
	return scan
}

func freeScanner(scan *scanner) {
	// Avoid hanging on to too much memory in extreme cases.
	if len(scan.parseState) > 1024 {
		scan.parseState = nil
	}
	scannerPool.Put(scan)
}

// These values are returned by the state transition functions
// assigned to scanner.state and the method scanner.eof.
// They give details about the current state of the scan that
// callers might be interested to know about.
// It is okay to ignore the return value of any particular
// call to scanner.state: if one call returns scanError,
// every subsequent call will return scanError too.
const (
	// Continue.
	scanContinue        = iota // uninteresting byte
	scanSkipSpace              // space byte; can skip; known to be last "continue" result
	scanBeginTag               // begin block tag
	scanEndTag                 // end block tag
	scanBeginArray             // begin array
	scanBeginText              // begin text block
	scanBeginDictionary        // begin dictionary block
	scanEndBlock               // end block (dictionary or text)
	scanEndArray               // end array (implies scanArrayValue if possible)
	scanArrayValue             // started scanning array value
	scanDictionaryKey          // started scanning dictionary key
	scanDictionaryValue        // started scanning dictionary value
	scanTextLine               // started scanning new text line

	// Stop.
	scanEnd   // top-level value ended *before* this byte; known to be first "stop" result
	scanError // hit an error, scanner.err.
)

// These values are stored in the parseState stack.
// They give the current state of a composite value
// being scanned. If the parser is inside a nested value
// the parseState describes the nested state, outermost at entry 0.
const (
	parseArrayValue = iota // parsing object key (before colon)
	parseDictionaryKey
	parseDictionaryValue
	parseTextValue
)

// tags is an array listing all allowed bru tag names
var tags = []string{"meta", "vars:secret", "body", "tests", "get", "post", "put", "delete",
	"options", "trace", "connect", "head", "query", "headers", "body:text", "body:xml",
	"body:form-urlencoded", "body:multipart-form", "body:graphql", "body:graphql:vars", "script:pre-request",
	"script:post-response", "body:test", "body:json", "assert", "vars"}

// blockTypes is an array listing the types of the aforementioned tags
// to access a tags type, juste use blockTypes[<index of tag>]
var blockTypes = []int{dictionaryBlock, arrayBlock, textBlock, textBlock, dictionaryBlock, dictionaryBlock, dictionaryBlock, dictionaryBlock,
	dictionaryBlock, dictionaryBlock, dictionaryBlock, dictionaryBlock, dictionaryBlock, dictionaryBlock, textBlock, textBlock,
	dictionaryBlock, dictionaryBlock, textBlock, textBlock, textBlock,
	textBlock, textBlock, textBlock, dictionaryBlock, dictionaryBlock}

// The types of block in Bru
const (
	dictionaryBlock = iota
	textBlock
	arrayBlock
)

// No nesting should take place, enforced to prevent stack overflow.
const maxNestingDepth = 1

// reset prepares the scanner for use.
// It must be called before calling s.step.
func (s *scanner) reset() {
	s.step = stateBeginBlockLine
	s.parseState = s.parseState[0:0]
	s.err = nil
	s.endBlock = false
	s.tagName = nil
}

// eof tells the scanner that the end of input has been reached.
// It returns a scan status just as s.step does.
func (s *scanner) eof() int {
	if s.err != nil {
		return scanError
	}
	if s.endBlock {
		return scanEnd
	}
	s.step(s, ' ')
	if s.endBlock {
		return scanEnd
	}
	if s.err == nil {
		s.err = &SyntaxError{"unexpected end of Bru input", s.bytes}
	}
	return scanError
}

// pushParseState pushes a new parse state p onto the parse stack.
// an error state is returned if maxNestingDepth was exceeded, otherwise successState is returned.
func (s *scanner) pushParseState(c byte, newParseState int, successState int) int {
	s.parseState = append(s.parseState, newParseState)
	if len(s.parseState) <= maxNestingDepth {
		return successState
	}
	return s.error(c, "exceeded max depth")
}

// popParseState pops a parse state (already obtained) off the stack
// and updates s.step accordingly.
func (s *scanner) popParseState() {
	n := len(s.parseState) - 1
	s.parseState = s.parseState[0:n]
	s.endBlock = true
	s.step = stateBeginBlockLine
}

func isSpace(c byte) bool {
	return c <= ' ' && (c == ' ' || c == '\t' || c == '\r' || c == '\n')
}

// stateBeginValueOrEmpty is the state after reading `[`.
func stateBeginValueOrEmpty(s *scanner, c byte) int {
	if isSpace(c) {
		return scanSkipSpace
	}
	if c == ']' {
		return stateEndValue(s, c)
	}
	return stateBeginBlockLine(s, c)
}

// checkTag checks if the given tag is a valid bru tag (this includes tag types)
func (s *scanner) checkTag(c byte) int {
	tagName := string(s.tagName)
	s.tagName = nil
	for i, tag := range tags {
		if tagName == tag {
			s.step = stateWaitingForOpenBlock
			// Tag found, determine what to parse next
			switch blockTypes[i] {
			case dictionaryBlock:
				return s.pushParseState(c, parseDictionaryKey, scanEndTag)
			case arrayBlock:
				return s.pushParseState(c, parseArrayValue, scanEndTag)
			case textBlock:
				return s.pushParseState(c, parseTextValue, scanEndTag)
			}
		}
	}
	return s.error(c, "invalid tag name: "+tagName)
}

// stateReadingTag is when the scanner is reading a tag
func stateReadingTag(s *scanner, c byte) int {
	if isSpace(c) {
		// Checking that tag exists
		return s.checkTag(c)
	}
	s.tagName = append(s.tagName, c)
	return scanContinue
}

// stateReadingTag is when the scanner finished reading a tag and waits for the block opening
func stateWaitingForOpenBlock(s *scanner, c byte) int {
	if isSpace(c) {
		return scanSkipSpace
	}
	n := len(s.parseState)
	ps := s.parseState[n-1]
	switch ps {
	case parseDictionaryKey:
		if c == '{' {
			s.step = stateOpenBlock
			return scanBeginDictionary
		}
	case parseTextValue:
		if c == '{' {
			s.step = stateOpenBlock
			return scanBeginText
		}
	case parseArrayValue:
		if c == '[' {
			s.step = stateOpenBlock
			return scanBeginArray
		}
	}
	return s.error(c, "unexpected char after block name")
}

// stateBeginBlockLine is the state when trying to read a new block
func stateBeginBlockLine(s *scanner, c byte) int {
	if isSpace(c) {
		return scanSkipSpace
	}
	if c > 96 && c < 123 {
		// Start of a tagName
		s.tagName = make([]byte, 0)
		s.tagName = append(s.tagName, c)
		s.endBlock = false
		s.step = stateReadingTag
		return scanBeginTag
	}
	// No block matched, end
	return scanEnd
}

// stateOpenBlock is the state after reading `{` or `[`.
func stateOpenBlock(s *scanner, c byte) int {
	n := len(s.parseState)
	// Parse expected block
	ps := s.parseState[n-1]
	switch ps {
	case parseDictionaryKey: // Expecting to read a dictionary key
		if isSpace(c) {
			return scanSkipSpace
		}
		return stateNewDictionaryPair(s, c)
	case parseArrayValue:
		if isSpace(c) {
			return scanSkipSpace
		}
		return stateNewArrayValue(s, c)
	case parseTextValue:
		// Ignore first newline
		s.step = stateNewTextLine
		return scanSkipSpace
	}
	return s.error(c, "no state for block")
}

// stateEndValue is the state after completing a value,
func stateEndValue(s *scanner, c byte) int {
	n := len(s.parseState)
	ps := s.parseState[n-1]
	switch ps {
	case parseTextValue:
		if c == '\n' {
			s.step = stateNewTextLine
			return scanTextLine
		}
		if c == '}' {
			s.popParseState()
			return scanEndBlock
		}
		return s.error(c, "after text line")
	case parseDictionaryKey:
		if isSpace(c) {
			s.step = stateEndValue
			return scanSkipSpace
		}
		if c == ':' {
			s.parseState[n-1] = parseDictionaryValue
			s.step = stateBeginDictionaryValue
			return scanDictionaryValue
		}
		return s.error(c, "after dictionary key ")
	case parseDictionaryValue:
		if c == ',' || c == '\n' {
			s.parseState[n-1] = parseDictionaryKey
			s.step = stateNewDictionaryPair
			return scanDictionaryKey
		}
		if c == '}' {
			s.popParseState()
			return scanEndBlock
		}
		return s.error(c, "after dictionary key:value pair")
	case parseArrayValue:
		if isSpace(c) {
			s.step = stateEndValue
			return scanSkipSpace
		}
		if c == ',' {
			s.step = stateNewArrayValue
			return scanArrayValue
		}
		if c == ']' {
			s.popParseState()
			return scanEndArray
		}
		return s.error(c, "after array element")
	}
	return s.error(c, "")
}

// stateNewDictionaryPair is the state when trying to read a new dictionary block line
func stateNewDictionaryPair(s *scanner, c byte) int {
	if isSpace(c) {
		return scanSkipSpace
	}
	// First char is an end block
	if c == '}' {
		s.popParseState()
		return scanEndBlock
	}
	s.step = stateInKey
	return stateInKey(s, c)
}

// stateNewDictionaryPair is the state when trying to read a new array block line
func stateNewArrayValue(s *scanner, c byte) int {
	if isSpace(c) {
		return scanSkipSpace
	}
	// First char is an end block
	if c == ']' {
		s.popParseState()
		return scanEndBlock
	}
	s.step = stateInValue
	return stateInValue(s, c)
}

// stateNewDictionaryPair is the state when trying to read a new text block line
func stateNewTextLine(s *scanner, c byte) int {
	// If first char is end, end
	if c == '}' {
		s.popParseState()
		return scanEndBlock
	}
	s.step = stateInText
	return scanTextLine
}

// stateInText is the state when reading a text block line
func stateInText(s *scanner, c byte) int {
	if c == '\n' {
		return stateEndValue(s, c)
	}
	//fmt.Printf("%c stateInText\n", c)
	return scanContinue
}

// stateInKey is the state when reading a key in a dictionary block line
func stateInKey(s *scanner, c byte) int {
	if c == ':' {
		return stateEndValue(s, c)
	}
	if c == '\\' {
		s.step = stateInStringEsc
		return scanContinue
	}
	if c < 0x20 {
		return s.error(c, "in key")
	}
	//fmt.Printf("%c stateInKey\n", c)
	return scanContinue
}

// stateBeginDictionaryValue is the state after reading a key in a dictionary block line
func stateBeginDictionaryValue(s *scanner, c byte) int {
	// A key without a value
	if c == '\n' {
		return stateEndValue(s, c)
	}
	if isSpace(c) {
		return scanSkipSpace
	}
	s.step = stateInValue
	return stateInValue(s, c)
}

// stateInValue is the state when reading a value from a dictionary or array block line
func stateInValue(s *scanner, c byte) int {
	if c == '\\' {
		s.step = stateInStringEsc
		return scanContinue
	}
	if c == '\n' {
		return stateEndValue(s, c)
	}
	if c < 0x20 {
		return s.error(c, "in value literal")
	}
	//fmt.Printf("%c stateInValue\n", c)
	return scanContinue
}

// stateInStringEsc is the state after reading `"\` during a quoted string.
// TODO: Not sure this is needed for bru
func stateInStringEsc(s *scanner, c byte) int {
	//fmt.Printf("%c stateInEscape\n", c)
	switch c {
	case 'b', 'f', 'n', 'r', 't', '\\', '/', '"':
		s.step = stateInValue
		return scanContinue
	case 'u':
		s.step = stateInStringEscU
		return scanContinue
	}
	return s.error(c, "in string escape code")
}

// stateInStringEscU is the state after reading `"\u` during a quoted string.
func stateInStringEscU(s *scanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		s.step = stateInStringEscU1
		return scanContinue
	}
	// numbers
	return s.error(c, "in \\u hexadecimal character escape")
}

// stateInStringEscU1 is the state after reading `"\u1` during a quoted string.
func stateInStringEscU1(s *scanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		s.step = stateInStringEscU12
		return scanContinue
	}
	// numbers
	return s.error(c, "in \\u hexadecimal character escape")
}

// stateInStringEscU12 is the state after reading `"\u12` during a quoted string.
func stateInStringEscU12(s *scanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		s.step = stateInStringEscU123
		return scanContinue
	}
	// numbers
	return s.error(c, "in \\u hexadecimal character escape")
}

// stateInStringEscU123 is the state after reading `"\u123` during a quoted string.
func stateInStringEscU123(s *scanner, c byte) int {
	if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
		s.step = stateInValue
		return scanContinue
	}
	// numbers
	return s.error(c, "in \\u hexadecimal character escape")
}

// stateError is the state after reaching a syntax error,
func stateError(s *scanner, c byte) int {
	return scanError
}

// error records an error and switches to the error state.
func (s *scanner) error(c byte, context string) int {
	s.step = stateError
	s.err = &SyntaxError{"invalid character " + quoteChar(c) + " " + context, s.bytes}
	return scanError
}

// quoteChar formats c as a quoted character literal.
func quoteChar(c byte) string {
	// special cases - different from quoted strings
	if c == '\'' {
		return `'\''`
	}
	if c == '"' {
		return `'"'`
	}

	// use quoted string with different quotation marks
	s := strconv.Quote(string(c))
	return "'" + s[1:len(s)-1] + "'"
}
