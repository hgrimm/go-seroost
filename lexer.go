package main

import (
	"strings"
	"unicode"

	"github.com/reiver/go-porterstemmer"
)

type Lexer struct {
	content  []rune
	position int
}

func NewLexer(content []rune) *Lexer {
	return &Lexer{content: content}
}

func (l *Lexer) trimLeft() {
	for l.position < len(l.content) && unicode.IsSpace(l.content[l.position]) {
		l.position++
	}
}

func (l *Lexer) chop(n int) []rune {
	token := l.content[l.position : l.position+n]
	l.position += n
	return token
}

func (l *Lexer) chopWhile(predicate func(rune) bool) []rune {
	start := l.position
	for l.position < len(l.content) && predicate(l.content[l.position]) {
		l.position++
	}
	return l.content[start:l.position]
}

func (l *Lexer) NextToken() (string, bool) {
	l.trimLeft()
	if l.position >= len(l.content) {
		return "", false
	}

	c := l.content[l.position]
	if unicode.IsDigit(c) {
		tokenRunes := l.chopWhile(unicode.IsDigit)
		return string(tokenRunes), true
	}

	if unicode.IsLetter(c) {
		tokenRunes := l.chopWhile(func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r)
		})
		term := strings.ToLower(string(tokenRunes))
		stemmedTerm := porterstemmer.StemString(term)
		return stemmedTerm, true
	}

	tokenRunes := l.chop(1)
	return string(tokenRunes), true
}
