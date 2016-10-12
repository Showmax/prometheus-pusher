package toml

import (
	"unicode/utf8"
)

const (
	EOF       = 0xF8
	RuneError = 0xFFFD
)

type Scanner interface {
	Fetch(skip bool) string
	Rune() rune
	Next() rune
	Eof() bool
	LastLine() (int, int, string)
}

type scanner struct {
	buf    []byte
	pos    int
	offset int // for Get()
	size   int
	line   int
	first  int // offset to first char of line
	col    int
	r      rune
}

// first scanner
func NewScanner(source []byte) Scanner {
	p := scanner{}
	p.buf = source
	r := p.Next()

	if r == BOM {
		p.Fetch(true)
		p.Next()
	}

	p.line = 1
	p.first = 0

	if r == RuneError {
		p.buf = nil
	}
	return &p
}

func (p *scanner) Eof() bool {
	return p.r == EOF
}

func (p *scanner) Rune() rune {
	return p.r
}

// Fetch returns last char, at string(buffer[:Scan.offset - sizeOfLastChar]).

// Fetch 返回未被取出的字符串. skip 表示是否包含 pos 处的字符
func (p *scanner) Fetch(last bool) (str string) {
	e := p.pos
	if !last && p.r != EOF {
		e -= p.size
	}

	str = string(p.buf[p.offset:e])
	p.offset = e
	return
}

func (p *scanner) LastLine() (int, int, string) {
	s := p.first
	e := p.pos - p.size
	if p.col == 1 && s > 0 {
		s--
	}

	r := p.buf[s]

	for s > 0 && r != '\n' && r != '\r' && r != 0x1E {
		s--
		r = p.buf[s]
	}

	r = p.buf[e]
	for e < len(p.buf) && r != '\n' && r != '\r' && r != 0x1E {
		r = p.buf[e]
		e++
	}

	return p.line, p.col, string(p.buf[s:e])

}

// Next returns read char frome the buffer.
// b is byte(char) when size of char equal 1, otherwise it is const MultiBytes.
// r is rune value of char,If the encoding is invalid, it is RuneError.
// if end of buffer or encoding is invalid, char is error string, b equal const EOF, r equal const RuneError.
func (p *scanner) Next() rune {
	if p.r == EOF {
		return p.r
	}

	if p.pos >= len(p.buf) {
		p.r = EOF
		return p.r
	}

	r := p.r

	p.r, p.size = utf8.DecodeRune(p.buf[p.pos:])

	n := p.r
	p.col++
	if n == '\n' || n == '\r' || n == 0x1E {
		if n == r || n != r && r != '\n' && r != '\r' {
			p.col = 1
			p.line++
			p.first = p.pos
		}
	}
	p.pos += p.size

	return p.r
}
