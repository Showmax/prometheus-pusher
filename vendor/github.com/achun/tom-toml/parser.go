package toml

import (
	"errors"
	"fmt"
	"strings"
)

type Status int

const (
	SNot Status = iota
	SInvalid
	SUnexpected
	SMaybe
	SYes
	SYesKeep // SYes 并且多读了一个字符, 保持当前的字符给后续解析
)

var statusName = [...]string{
	"Not",
	"Invalid",
	"Unexpected",
	"Maybe",
	"Yes",
	"YesKeep",
}

func (t Status) String() string {
	return statusName[t]
}

const (
	BOM = 0xFEFF // UTF-8 encoded byte order mark
)

type Token uint

// don't change order
const (
	tokenEOF Token = iota
	tokenString
	tokenInteger
	tokenFloat
	tokenBoolean
	tokenDatetime
	tokenWhitespace
	tokenComment
	tokenTableName
	tokenArrayOfTables
	tokenNewLine
	tokenKey
	tokenEqual
	tokenArrayLeftBrack
	tokenArrayRightBrack
	tokenComma
	tokenError
	tokenRuneError
	tokenNothing
)

func (t Token) String() string {
	i := int(t)
	if i < 0 || i >= len(tokensName) {
		return "User defined error: " + fmt.Sprint(i)
	}
	return tokensName[i]
}

var tokensName = [...]string{
	"EOF",
	"String",
	"Integer",
	"Float",
	"Boolean",
	"Datetime",
	"Whitespace",
	"Comment",
	"TableName",
	"ArrayOfTables",
	"NewLine",
	"Key",
	"Equal",
	"ArrayLeftBrack",
	"ArrayRightBrack",
	"Comma",
	"Error",
	"EncodingError",
	"Nothing",
}

type TokenHandler func(Token, string) error

type parser interface {
	Scanner
	Handler(TokenHandler)

	Run()
	Keep()
	IsTestMode() bool

	Err(msg string)
	Token(token Token) error
	Invalid(token Token)
	NotMatch(token ...Token)
	Unexpected(token Token)
}

type parse struct {
	Scanner
	err      error
	handler  TokenHandler
	next     bool
	testMode bool // 测试模式允许不完整的 stage
}

func (p *parse) Close() {}

func (p *parse) Run() {
	stagePlay(p, openStage())
}

func (p *parse) IsTestMode() bool {
	return p.testMode
}

func (p *parse) Next() rune {
	if p.next {
		return p.Scanner.Next()
	}
	p.next = true
	return p.Scanner.Rune()
}

func (p *parse) Keep() {
	p.next = false
}

func (p *parse) Err(msg string) {
	p.err = errors.New(msg)
	p.Token(tokenError)
}

func (p *parse) NotMatch(token ...Token) {
	var msg string
	if len(token) == 1 {
		msg = "incomplete"
	} else {
		msg = "no one can match for"
	}

	for _, t := range token {
		msg += " " + t.String()
	}

	p.err = errors.New(msg)
	p.Token(tokenError)
}

func (p *parse) Invalid(token Token) {
	p.err = errors.New("invalid " + tokensName[token])
	p.Token(tokenError)
}

func (p *parse) Unexpected(token Token) {
	p.err = errors.New("unexpected token " + tokensName[token])
	p.Token(tokenError)
}

func (p *parse) Token(token Token) (err error) {
	var str string
	if token == tokenError {
		if p.err == nil {
			p.err = errors.New("invalid format.")
		}
		err = p.err
		str = err.Error()
	} else {
		str = p.Scanner.Fetch(p.next)
		if token != tokenEOF && token != tokenWhitespace {
			str = strings.TrimSpace(str)
		}
	}

	if p.handler == nil {
		fmt.Println(token.String(), str)
	} else {
		if token == tokenError {
			p.handler(token, str)
			if err == nil {
				err = errors.New("tokenError")
			}
		} else {
			err = p.handler(token, str)
		}
	}
	return
}

// Handler to set TokenHandler
func (p *parse) Handler(h TokenHandler) {
	p.handler = h
}

// tokens
func itsWhitespace(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenWhitespace
	}
	if isWhitespace(r) {
		return SMaybe, 1
	}
	if flag == 1 {
		return SYesKeep, tokenWhitespace
	}
	return SNot, tokenWhitespace
}
func itsComment(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenComment
	}

	switch flag {
	case 0:
		if r == '#' {
			return SMaybe, 1
		}
	case 1:
		if isEOF(r) {
			return SYes, tokenComment
		}
		if isNewLine(r) {
			return SYesKeep, tokenComment
		}
		return SMaybe, 1
	}
	return SNot, tokenComment
}
func itsString(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenString
	}

	switch flag {
	case 0:
		if r == '"' {
			return SMaybe, 2
		}
		return SNot, tokenString
	case 1: // skip
		if !isNewLine(r) {
			return SMaybe, 2
		}
	case 2:
		if r == '"' {
			return SYes, tokenString
		}
		if r == '\\' {
			return SMaybe, 1
		}
		if !isNewLine(r) {
			return SMaybe, 2
		}
	}
	return SInvalid, tokenString
}

// 要求在 itsFlaot 的前面
func itsInteger(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenInteger
	}

	switch flag {
	case 0:
		if r == '-' {
			return SMaybe, 1
		}
		if is09(r) {
			return SMaybe, 2
		}
	case 1:
		if is09(r) {
			return SMaybe, 2
		}
	case 2:
		if is09(r) {
			return SMaybe, 2
		}
		if isSuffixOfValue(r) {
			return SYesKeep, tokenInteger
		}
	}
	return SNot, tokenInteger
}

func itsFloat(r rune, flag int, maybe bool) (Status, Token) {

	// 还是有 bug ??? 注释中可能有这些值
	switch flag {
	case 0:
		if r == '-' {
			return SMaybe, 1
		}
		if is09(r) {
			return SMaybe, 2
		}
	case 1:
		if is09(r) {
			return SMaybe, 2
		}
	case 2:
		if is09(r) {
			return SMaybe, 2
		}
		if r == '.' {
			return SMaybe, 3
		}
	case 3:
		if is09(r) {
			return SMaybe, 4
		}
		return SInvalid, tokenFloat
	case 4:
		if is09(r) {
			return SMaybe, 4
		}
		if isSuffixOfValue(r) {
			return SYesKeep, tokenFloat
		}
	}
	return SNot, tokenFloat
}

func itsBoolean(r rune, flag int, maybe bool) (Status, Token) {
	const layout = "truefalse"
	switch flag {
	case 0:
		if maybe {
			return SNot, tokenBoolean
		}
		if r == 't' {
			return SMaybe, 1
		}
		if r == 'f' {
			return SMaybe, 5
		}
	case 1, 2, 3, 5, 6, 7, 8:
		if rune(layout[flag]) == r {
			return SMaybe, Token(flag + 1)
		}
	case 4, 9:
		if isSuffixOfValue(r) {
			return SYesKeep, tokenBoolean
		}
	}
	return SNot, tokenBoolean
}
func itsDatetime(r rune, flag int, maybe bool) (Status, Token) {
	const layout = "0000-00-00T00:00:00Z"
	if flag >= 0 && flag < 20 {
		if layout[flag] == '0' && is09(r) || r == rune(layout[flag]) {
			return SMaybe, Token(flag + 1)
		}
		if flag <= 4 {
			return SNot, tokenDatetime
		}
	}
	if flag == 20 && isSuffixOfValue(r) {
		return SYesKeep, tokenDatetime
	}
	return SInvalid, tokenDatetime
}

// [ASCII] http://en.wikipedia.org/wiki/ASCII#ASCII_printable_characters
func itsTableName(r rune, flag int, maybe bool) (Status, Token) {
	switch flag {
	case 0:
		if !maybe && r == '[' {
			return SMaybe, 1
		}
	case 1:
		if r == '[' {
			return SNot, tokenTableName
		}
		if isNewLine(r) || isWhitespace(r) || r == ']' || r == '.' {
			return SInvalid, tokenTableName
		}
		return SMaybe, 2
	case 2:
		if isNewLine(r) || isWhitespace(r) {
			return SInvalid, tokenTableName
		}
		if r != ']' {
			return SMaybe, 2
		}
		return SYes, tokenTableName
	}
	return SNot, tokenTableName
}

func itsArrayOfTables(r rune, flag int, maybe bool) (Status, Token) {
	switch flag {
	case 0, 1:
		if r == '[' {
			return SMaybe, Token(flag + 1)
		}
	case 2:
		if isNewLine(r) || isWhitespace(r) || r == ']' || r == '.' {
			return SInvalid, tokenArrayOfTables
		}
		return SMaybe, 3
	case 3:
		if isNewLine(r) || isWhitespace(r) {
			return SInvalid, tokenArrayOfTables
		}
		if r != ']' {
			return SMaybe, 3
		}
		return SMaybe, 4
	case 4:
		if r != ']' {
			return SInvalid, tokenArrayOfTables
		}
		return SYes, tokenArrayOfTables
	}
	return SNot, tokenArrayOfTables
}
func itsNewLine(r rune, flag int, maybe bool) (Status, Token) {
	if flag == 0 && isNewLine(r) {
		return SYes, tokenNewLine
	}
	return SNot, tokenNewLine
}

func itsKey(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenKey
	}
	switch flag {
	case 0:
		return SMaybe, 1
	case 1:
		if isNewLine(r) || isEOF(r) {
			return SInvalid, tokenKey
		}
		if r == '=' || isWhitespace(r) {
			return SYesKeep, tokenKey
		}
		return SMaybe, 1
	}
	return SNot, tokenKey
}
func itsEqual(r rune, flag int, maybe bool) (Status, Token) {
	if maybe && flag == 0 {
		return SNot, tokenEqual
	}
	if r == '=' {
		return SYes, tokenEqual
	}
	return SNot, tokenEqual
}

func itsComma(r rune, flag int, maybe bool) (Status, Token) {
	if !maybe && r == ',' {
		return SYes, tokenComma
	}
	return SNot, tokenComma
}

func itsArrayLeftBrack(r rune, flag int, maybe bool) (Status, Token) {
	if !maybe && r == '[' {
		return SYes, tokenArrayLeftBrack
	}
	return SNot, tokenArrayLeftBrack
}
func itsArrayRightBrack(r rune, flag int, maybe bool) (Status, Token) {
	if !maybe && r == ']' {
		return SYes, tokenArrayRightBrack
	}
	return SNot, tokenArrayRightBrack
}

func itsEOF(r rune, flag int, maybe bool) (Status, Token) {
	if r == EOF {
		return SYes, tokenEOF
	}
	return SNot, tokenEOF
}

func itsSNot(r rune, flag int, maybe bool) (Status, Token) {
	return SNot, tokenEOF
}

func isEOF(r rune) bool {
	return r == EOF
}
func is09(r rune) bool {

	return r >= '0' && r <= '9'
}

// Spec: Whitespace means tab (0x09) or space (0x20).
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' // || r == '\v' || r == '\f' // 0x85, 0xA0 ?
}

// [NewLine](http://en.wikipedia.org/wiki/Newline)
//	LF    = 0x0A   // Line feed, \n
//	CR    = 0x0D   // Carriage return, \r
//	LFCR  = 0x0A0D // \n\r
//	CRLF  = 0x0D0A // \r\n
//	RS    = 0x1E   // QNX pre-POSIX implementation.
func isNewLine(r rune) bool {
	return r == '\n' || r == '\r' || r == 0x1E
}
func isSuffixOfValue(r rune) bool {
	return isWhitespace(r) || isNewLine(r) || isEOF(r) || r == '#' || r == ',' || r == ']'
}
