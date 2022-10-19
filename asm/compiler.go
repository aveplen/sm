package compiler

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/aveplen/sm/emu"
)

const (
	_INSTRUCTION = iota + 1
	_INTEGER
	_LABEL
	_LABELREF
)

var (
	breakpoints = []rune{'\n', ' '}
)

func runein(r rune, coll []rune) bool {
	for _, x := range coll {
		if x == r {
			return true
		}
	}
	return false
}

type runeiter struct {
	in     bufio.Reader
	opened bool
	closed bool
	n      rune
}

func newruneiter(in bufio.Reader) runeiter {
	ret := runeiter{in: in}
	ret.walk()
	ret.opened = true
	return ret
}

func (ri *runeiter) walk() {
	if ri.closed {
		return
	}

	n, _, err := ri.in.ReadRune()
	if err != nil {
		ri.n = n
		ri.closed = true
		return
	}

	if ri.n == unicode.ReplacementChar {
		panic("could not read rune: invalid code")
	}

	ri.n = n
}

func (ri *runeiter) hasnext() bool {
	return ri.opened && !ri.closed
}

func (ri *runeiter) next() rune {
	ret := ri.n
	ri.walk()
	return ret
}

type lexem struct {
	val string
	typ int
}

type runeiterator interface {
	next() rune
	hasnext() bool
}

type lexemiter struct {
	runeiter  runeiterator
	opened    bool
	closed    bool
	exhausted bool
	buf       []rune
}

func newlexemiter(runeiter runeiterator) lexemiter {
	ret := lexemiter{runeiter: runeiter}
	ret.walk()
	ret.opened = true
	return ret
}

func (li *lexemiter) walk() {
	if li.closed {
		return
	}

	if li.exhausted {
		li.closed = true
		return
	}

	li.buf = []rune{}
	for li.runeiter.hasnext() {
		nextrune := li.runeiter.next()

		if runein(nextrune, breakpoints) {
			return
		}

		li.buf = append(li.buf, nextrune)
	}

	li.exhausted = true
}

func (li *lexemiter) hasnext() bool {
	return li.opened && !li.closed
}

func isinteger(word string) bool {
	for _, l := range word {
		if !(('0' <= l) && (l <= '9')) {
			return false
		}
	}
	return true
}

func isinstruction(word string) bool {
	iset := map[string]struct{}{
		"nop":    {},
		"add":    {},
		"sub":    {},
		"and":    {},
		"or":     {},
		"xor":    {},
		"not":    {},
		"in":     {},
		"out":    {},
		"load":   {},
		"stor":   {},
		"jmp":    {},
		"jz":     {},
		"push":   {},
		"dup":    {},
		"swap":   {},
		"rol3":   {},
		"outnum": {},
		"jnz":    {},
		"drop":   {},
		"compl":  {},
	}

	_, ok := iset[strings.ToLower(word)]
	return ok
}

func islabel(word string) bool {
	for _, l := range word {
		if l == '_' {
			continue
		}

		if l == ':' {
			continue
		}

		if l < 'a' {
			return false
		}

		if l > 'z' {
			return false
		}
	}

	lensatisf := len(word) >= 2

	runes := []rune(word)
	colonend := runes[len(runes)-1] == ':'

	return lensatisf && colonend
}

func islabelref(word string) bool {
	for _, l := range word {
		if l == '_' {
			continue
		}

		if l == '&' {
			continue
		}

		if l < 'a' {
			return false
		}

		if l > 'z' {
			return false
		}
	}

	lensatisf := len(word) >= 2
	amperstart := []rune(word)[0] == '&'

	return lensatisf && amperstart
}

func decode(word string) int {
	if isinteger(word) {
		return _INTEGER
	}

	if isinstruction(word) {
		return _INSTRUCTION
	}

	if islabel(word) {
		return _LABEL
	}

	if islabelref(word) {
		return _LABELREF
	}

	panic("could not decode lexem: unknown format word")
}

func (li *lexemiter) next() lexem {
	val := string(li.buf)
	ret := lexem{
		val: val,
		typ: decode(val),
	}
	li.walk()
	return ret
}

type lexemiterator interface {
	next() lexem
	hasnext() bool
}

type compiler struct {
	in     bufio.Reader
	labels map[string]uint32
	ino    int
}

func (c *compiler) compile() []uint32 {
	buf := make([]uint32, 0)
	test := newruneiter(c.in)
	test1 := newlexemiter(&test)
	var li lexemiterator = &test1

	for li.hasnext() {
		lexem := li.next()

		var apnd uint32
		switch lexem.typ {

		case _INSTRUCTION:
			apnd = c.compileinstr(lexem.val)

		case _INTEGER:
			apnd = c.compileint(lexem.val)

		case _LABEL:
			apnd = c.compilelabel(lexem.val)

		case _LABELREF:
			apnd = c.compilelabelref(lexem.val)

		default:
			panic("unknown lexem type")
		}

		buf = append(buf, apnd)
		c.ino++
	}

	return buf
}

func (c *compiler) compileinstr(value string) uint32 {
	strmap := map[string]uint32{
		"nop":    emu.NOP,
		"add":    emu.ADD,
		"sub":    emu.SUB,
		"and":    emu.AND,
		"or":     emu.OR,
		"xor":    emu.XOR,
		"not":    emu.NOT,
		"in":     emu.IN,
		"out":    emu.OUT,
		"load":   emu.LOAD,
		"stor":   emu.STOR,
		"jmp":    emu.JMP,
		"jz":     emu.JZ,
		"push":   emu.PUSH,
		"dup":    emu.DUP,
		"swap":   emu.SWAP,
		"rol3":   emu.ROL3,
		"outnum": emu.OUTNUM,
		"jnz":    emu.JNZ,
		"drop":   emu.DROP,
		"compl":  emu.COMPL,
	}

	code, ok := strmap[strings.ToLower(value)]
	if !ok {
		panic("unkown instruction mnemonics")
	}

	return code
}

func (c *compiler) compileint(value string) uint32 {
	asint64, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		panic(fmt.Errorf("could not parse int: %w", err))
	}
	return uint32(asint64)
}

var ()

func (c *compiler) compilelabel(value string) uint32 {
	raw := value[:len(value)-1]
	_, ok := c.labels[raw]
	if ok {
		panic(fmt.Errorf("attempt to overwrite '%s' label", raw))
	}

	c.labels[raw] = uint32(c.ino)
	return emu.NOP
}

func (c *compiler) compilelabelref(value string) uint32 {
	raw := value[1:]
	labref, ok := c.labels[raw]
	if !ok {
		panic(fmt.Errorf("attempt to reference unexistant label '%s'", raw))
	}
	return labref
}

func Compile(in bufio.Reader) []uint32 {
	c := compiler{
		in:     in,
		labels: make(map[string]uint32),
		ino:    1,
	}
	return c.compile()
}
