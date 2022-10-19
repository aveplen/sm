package compiler

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"

	"github.com/aveplen/sm/emu"
)

func Test_runeiter_walk(t *testing.T) {
	tests := []struct {
		name string
		ri   *runeiter
	}{
		{
			name: "should set the next rune",
			ri: &runeiter{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("b"))),
				opened: true,
				closed: false,
				n:      rune('a'),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ri.walk()

			if tt.ri.n != rune('b') {
				t.Errorf("did not set the next rune: '%v', expected: '%v'", tt.ri.n, rune('b'))
			}
		})
	}
}

func Test_runeiter_next(t *testing.T) {
	tests := []struct {
		name string
		ri   *runeiter
		want rune
	}{
		{
			name: "should return current and goto next",
			ri: &runeiter{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("b"))),
				opened: true,
				closed: false,
				n:      rune('a'),
			},
			want: rune('a'),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ri.next(); got != tt.want {
				t.Errorf("runeiter.next() = %v, want %v", got, tt.want)
			}
			if tt.ri.n != rune('b') {
				t.Errorf("did not walk to next rune: '%v', expected: '%v'", tt.ri.n, rune('b'))
			}
		})
	}
}

func Test_lexemiter_walk(t *testing.T) {
	tests := []struct {
		name string
		li   *lexemiter
		want []rune
	}{
		{
			name: "should goto next lexem",
			li: &lexemiter{
				runeiter: &runeiter{
					in:     *bufio.NewReader(bytes.NewBuffer([]byte("bc ddd"))),
					opened: true,
					closed: false,
					n:      rune('a'),
				},
				opened: true,
				closed: false,
				buf:    []rune{},
			},
			want: []rune("abc"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.li.walk()

			if !reflect.DeepEqual(tt.want, tt.li.buf) {
				t.Errorf("did not walk to next lexem: '%v', expected: '%v'", tt.li.buf, tt.want)
			}
		})
	}
}

func Test_lexemiter_next(t *testing.T) {
	tests := []struct {
		name string
		li   *lexemiter
		want lexem
	}{
		{
			name: "should goto next lexem",
			li: &lexemiter{
				buf: []rune("nop"),
			},
			want: lexem{
				val: "nop",
				typ: _INSTRUCTION,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.li.next(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lexemiter.next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_compiler_compile(t *testing.T) {
	tests := []struct {
		name    string
		c       *compiler
		want    []uint32
		wlabels map[string]uint32
	}{
		{
			name: "should compile stream of instructions",
			c: &compiler{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("add In jMp NOP"))),
				labels: make(map[string]uint32),
				ino:    1,
			},
			want: []uint32{emu.ADD, emu.IN, emu.JMP, emu.NOP},
		},
		{
			name: "should compile stream of numbers",
			c: &compiler{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("1 2 3 123 456"))),
				labels: make(map[string]uint32),
				ino:    1,
			},
			want: []uint32{1, 2, 3, 123, 456},
		},
		{
			name: "should compile stream of labels",
			c: &compiler{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("a: b: c: d:"))),
				labels: make(map[string]uint32),
				ino:    1,
			},
			want: []uint32{emu.NOP, emu.NOP, emu.NOP, emu.NOP},
			wlabels: map[string]uint32{
				"a": 1,
				"b": 2,
				"c": 3,
				"d": 4,
			},
		},
		{
			name: "should compile stream of label refs",
			c: &compiler{
				in: *bufio.NewReader(bytes.NewBuffer([]byte("&a &b &c &d"))),
				labels: map[string]uint32{
					"a": 123,
					"b": 456,
					"c": 789,
					"d": 101112,
				},
				ino: 1,
			},
			want: []uint32{123, 456, 789, 101112},
		},
		{
			name: "should reference label crated before",
			c: &compiler{
				in:     *bufio.NewReader(bytes.NewBuffer([]byte("add nop a: load &a jmp"))),
				labels: make(map[string]uint32),
				ino:    1,
			},
			want: []uint32{emu.ADD, emu.NOP, emu.NOP, emu.LOAD, 3, emu.JMP},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.compile(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("compiler.compile() = %v, want %v", got, tt.want)
			}

			if tt.wlabels != nil {
				if !reflect.DeepEqual(tt.wlabels, tt.c.labels) {
					t.Errorf("wlabels != c.labels")
				}
			}
		})
	}
}