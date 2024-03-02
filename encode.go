package bru // Copyright 2010 The Go Authors. All rights reserved.
import (
	"bytes"
	"fmt"
	"strings"
)

type Encoder struct {
	indent             int
	lineSep            string
	addTrailingLineEnd bool
}

// Using default encoder for write
// To customize parameter create an encoder
func Write(data []ContentBlock) ([]byte, error) {
	return (&Encoder{}).Write(data)
}

func (b *Encoder) Write(data []ContentBlock) ([]byte, error) {
	var e encodeState
	err := e.marshal(data, b)
	if err != nil {
		return nil, err
	}
	// Remove one trailing \n
	toWrite := e.Bytes()
	n := len(toWrite)
	if n > 2 && toWrite[n-1] == '\n' && toWrite[n-2] == '\n' {
		toWrite = toWrite[:n-2+b.GetEndOffset()]
	}
	buf := append([]byte(nil), toWrite...)

	return buf, nil
}

// An encodeState encodes JSON into a bytes.Buffer.
type encodeState struct {
	bytes.Buffer // accumulated output
}

func (e *encodeState) marshal(data []ContentBlock, b *Encoder) (err error) {
	for _, d := range data {
		// Add the first line
		e.WriteString(d.GetName())
		if d.GetType() != "" {
			e.WriteString(":" + d.GetType())
		}
		// Add the content
		switch c := d.(type) {
		case *DictionaryBlock:
			e.WriteString(" {\n")
			for i, v := range c.Content {
				if i == len(c.Content)-1 {
					// Last
					e.WriteString(fmt.Sprintf("%s%s: %s\n", strings.Repeat(" ", b.GetIndent()), v.Key, v.Value))
				} else {
					i++
					e.WriteString(fmt.Sprintf("%s%s: %s%s\n", strings.Repeat(" ", b.GetIndent()), v.Key, v.Value, b.GetLineSep()))
				}
			}
			e.WriteString("}\n\n")
		case *TextBlock:
			e.WriteString(" {\n")
			e.WriteString(c.Content + "\n")
			e.WriteString("}\n\n")
		case *ArrayBlock:
			e.WriteString(" [\n")
			for i, v := range c.Content {
				if i == len(c.Content)-1 {
					// Last
					e.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat(" ", b.GetIndent()), v))
				} else {
					e.WriteString(fmt.Sprintf("%s%s%s\n", strings.Repeat(" ", b.GetIndent()), v, b.GetLineSep()))
				}
			}
			e.WriteString("]\n\n")
		}
	}
	return nil
}

func (b *Encoder) GetIndent() int {
	if b.indent != 0 {
		return b.indent
	}
	return 2
}

func (b *Encoder) GetEndOffset() int {
	if !b.addTrailingLineEnd {
		return 0
	}
	return 1
}

func (b *Encoder) GetLineSep() string {
	return b.lineSep
}
