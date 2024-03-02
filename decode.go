package bru

import (
	"fmt"
	"strings"
)

func Read(data []byte) ([]ContentBlock, error) {
	// Check for well-formedness.
	// Avoids filling out half a data structure
	// before discovering a JSON syntax error.
	var d decodeState
	err := checkValid(data, &d.scan)
	if err != nil {
		return nil, err
	}
	d.init(data)
	return d.unmarshal()
}

func (d *decodeState) unmarshal() ([]ContentBlock, error) {
	d.scan.reset()
	blocks, err := d.value()
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

// decodeState represents the state while decoding a JSON value.
type decodeState struct {
	data   []byte
	off    int // next read offset in data
	opcode int // last read result
	scan   scanner
}

// readIndex returns the position of the last byte read.
func (d *decodeState) readIndex() int {
	return d.off - 1
}

func (d *decodeState) init(data []byte) *decodeState {
	d.data = data
	d.off = 0
	return d
}

// skip scans to the end of what was started.
func (d *decodeState) skip() {
	s, data, i := &d.scan, d.data, d.off
	depth := len(s.parseState)
	for {
		op := s.step(s, data[i])
		i++
		if len(s.parseState) < depth {
			d.off = i
			d.opcode = op
			return
		}
	}
}

// scanNext processes the byte at d.data[d.off].
func (d *decodeState) scanNext() {
	if d.off < len(d.data) {
		d.opcode = d.scan.step(&d.scan, d.data[d.off])
		d.off++
	} else {
		d.opcode = d.scan.eof()
		d.off = len(d.data) + 1 // mark processed EOF with len+1
	}
}

// scanWhile processes bytes in d.data[d.off:] until it
// receives a scan code not equal to op.
func (d *decodeState) scanWhile(op int) {
	s, data, i := &d.scan, d.data, d.off
	for i < len(data) {
		newOp := s.step(s, data[i])
		i++
		if newOp != op {
			d.opcode = newOp
			d.off = i
			return
		}
	}

	d.off = len(data) + 1 // mark processed EOF with len+1
	d.opcode = d.scan.eof()
}

// value consumes a JSON value from d.data[d.off-1:], decoding into v, and
// reads the following byte ahead. If v is invalid, the value is discarded.
// The first byte of the value has been read already.
func (d *decodeState) value() ([]ContentBlock, error) {
	var blocks []ContentBlock
	for {
		d.scanWhile(scanSkipSpace)
		if d.opcode == scanEnd {
			break
		}
		block, err := d.block()
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		d.scanNext()
	}
	return blocks, nil
}

func getBlockForTag(tag string) (ContentBlock, error) {
	// Split
	for i, t := range tags {
		if tag == t {
			tag, tagData, _ := strings.Cut(tag, ":")
			switch blockTypes[i] {
			case dictionaryBlock:
				return &DictionaryBlock{
					Name: tag,
					Type: tagData,
				}, nil
			case textBlock:
				return &TextBlock{
					Name: tag,
					Type: tagData,
				}, nil
			case arrayBlock:
				return &ArrayBlock{
					Name: tag,
					Type: tagData,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find block for tag '%s'", tag)
}

// block consumes an block from d.data[d.off-1:], decoding into v.
// The first byte of the block has been read already.
func (d *decodeState) block() (ContentBlock, error) {
	// First read the tag
	for {
		if d.opcode == scanBeginTag {
			break
		}
		d.scanNext()
	}
	start := d.readIndex()
	for {
		if d.opcode == scanEndTag {
			break
		}
		d.scanNext()
	}
	blockName := string(d.data[start:d.readIndex()])
	block, err := getBlockForTag(blockName)
	if err != nil {
		return nil, err
	}
	d.scanWhile(scanSkipSpace)

	// Get the type of data to read
	switch d.opcode {
	case scanBeginDictionary:
		var dic []DictionaryElement
		for {
			if d.opcode == scanEndBlock {
				break
			}
			d.scanWhile(scanSkipSpace)
			// Get the key
			start := d.readIndex()
			d.scanWhile(scanContinue)
			key := string(d.data[start:d.readIndex()])
			d.scanWhile(scanSkipSpace)
			value := ""
			if d.opcode != scanDictionaryKey {
				// Get the value
				start = d.readIndex()
				d.scanWhile(scanContinue)
				value = string(d.data[start:d.readIndex()])
			}
			dic = append(dic, DictionaryElement{key, value})
			d.scanNext()
		}
		return block, block.SetContent(dic)
	case scanBeginArray:
		dic := make([]string, 0)
		for {
			if d.opcode == scanEndArray {
				break
			}
			d.scanWhile(scanSkipSpace)
			// Get the value
			start = d.readIndex()
			d.scanWhile(scanContinue)
			value := string(d.data[start:d.readIndex()])
			dic = append(dic, value)
			d.scanNext()
		}
		return block, block.SetContent(dic)
	case scanBeginText:
		// Wait for start of first text line
		for {
			if d.opcode == scanEndBlock || d.opcode == scanTextLine {
				break
			}
			d.scanNext()
		}
		dic := ""
		for {
			if d.opcode == scanEndBlock {
				break
			}
			// Get the value
			start = d.readIndex()
			d.scanWhile(scanContinue)
			value := string(d.data[start:d.readIndex()])
			dic = dic + value + "\n"
			d.scanNext()
		}
		return block, block.SetContent(dic[:len(dic)-1])
	}

	return nil, nil
}
