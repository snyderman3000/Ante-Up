package main

// Balatro save files (.jkr) are raw-deflate-compressed Lua table literals:
//   return {["GAME"]={["round_resets"]={["ante"]=3,...},...},}
// This is a small parser for exactly that subset: tables with [key]= / name=
// / positional entries, strings, numbers, booleans and nil.

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// T is a parsed Lua table. Keys are strings; numeric keys are stored as their
// decimal text ("1", "2"), which is how the save code reads them back anyway.
type T map[string]any

// ReadJKR decompresses and parses a .jkr file.
func ReadJKR(path string) (T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeJKR(raw)
}

func DecodeJKR(raw []byte) (T, error) {
	text, err := io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
	if err != nil && len(text) == 0 {
		// Not compressed? (Balatro can also read plain text saves.)
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("return")) {
			text = raw
		} else {
			return nil, fmt.Errorf("decompress: %w", err)
		}
	}
	return ParseLua(text)
}

type lparser struct {
	s   []byte
	pos int
}

func ParseLua(src []byte) (T, error) {
	p := &lparser{s: src}
	p.ws()
	if bytes.HasPrefix(p.s[p.pos:], []byte("return")) {
		p.pos += 6
	}
	p.ws()
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	t, ok := v.(T)
	if !ok {
		return nil, fmt.Errorf("top level is not a table")
	}
	return t, nil
}

func (p *lparser) errf(f string, a ...any) error {
	return fmt.Errorf("lua at %d: %s", p.pos, fmt.Sprintf(f, a...))
}

func (p *lparser) ws() {
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
			continue
		}
		if c == '-' && p.pos+1 < len(p.s) && p.s[p.pos+1] == '-' { // comment
			for p.pos < len(p.s) && p.s[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		return
	}
}

func (p *lparser) value() (any, error) {
	p.ws()
	if p.pos >= len(p.s) {
		return nil, p.errf("unexpected end")
	}
	c := p.s[p.pos]
	switch {
	case c == '{':
		return p.table()
	case c == '"' || c == '\'':
		return p.str()
	case c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return p.num()
	case isIdent(c):
		id := p.ident()
		switch id {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "nil":
			return nil, nil
		case "inf", "math.huge":
			return math.Inf(1), nil
		}
		return id, nil // unknown bare word: keep as text
	}
	return nil, p.errf("unexpected %q", c)
}

func isIdent(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (p *lparser) ident() string {
	st := p.pos
	for p.pos < len(p.s) && (isIdent(p.s[p.pos]) || (p.s[p.pos] >= '0' && p.s[p.pos] <= '9') || p.s[p.pos] == '.') {
		p.pos++
	}
	return string(p.s[st:p.pos])
}

func (p *lparser) num() (any, error) {
	st := p.pos
	if p.s[p.pos] == '-' {
		p.pos++
		p.ws()
		if p.pos < len(p.s) && isIdent(p.s[p.pos]) { // -inf / -math.huge
			if id := p.ident(); id == "inf" || id == "math.huge" {
				return math.Inf(-1), nil
			}
			return nil, p.errf("bad number")
		}
	}
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == 'x' || c == 'X' ||
			((c == '+' || c == '-') && (p.s[p.pos-1] == 'e' || p.s[p.pos-1] == 'E')) ||
			(c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			p.pos++
			continue
		}
		break
	}
	txt := strings.ReplaceAll(string(p.s[st:p.pos]), " ", "")
	if strings.HasPrefix(txt, "0x") || strings.HasPrefix(txt, "-0x") {
		n, err := strconv.ParseInt(txt, 0, 64)
		return float64(n), err
	}
	f, err := strconv.ParseFloat(txt, 64)
	if err != nil {
		return nil, p.errf("bad number %q", txt)
	}
	return f, nil
}

func (p *lparser) str() (string, error) {
	q := p.s[p.pos]
	p.pos++
	var b strings.Builder
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == q {
			p.pos++
			return b.String(), nil
		}
		if c == '\\' && p.pos+1 < len(p.s) {
			p.pos++
			e := p.s[p.pos]
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\n':
				b.WriteByte('\n')
			default:
				if e >= '0' && e <= '9' {
					n := 0
					for k := 0; k < 3 && p.pos < len(p.s) && p.s[p.pos] >= '0' && p.s[p.pos] <= '9'; k++ {
						n = n*10 + int(p.s[p.pos]-'0')
						p.pos++
					}
					b.WriteByte(byte(n))
					continue
				}
				b.WriteByte(e)
			}
			p.pos++
			continue
		}
		b.WriteByte(c)
		p.pos++
	}
	return "", p.errf("unterminated string")
}

func keyText(v any) string {
	switch k := v.(type) {
	case string:
		return k
	case float64:
		return strconv.FormatFloat(k, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(k)
	}
	return fmt.Sprint(v)
}

func (p *lparser) table() (T, error) {
	p.pos++ // {
	t := T{}
	idx := 1
	for {
		p.ws()
		if p.pos >= len(p.s) {
			return nil, p.errf("unterminated table")
		}
		if p.s[p.pos] == '}' {
			p.pos++
			return t, nil
		}
		var key string
		haveKey := false
		if p.s[p.pos] == '[' {
			p.pos++
			k, err := p.value()
			if err != nil {
				return nil, err
			}
			p.ws()
			if p.pos >= len(p.s) || p.s[p.pos] != ']' {
				return nil, p.errf("expected ]")
			}
			p.pos++
			p.ws()
			if p.pos >= len(p.s) || p.s[p.pos] != '=' {
				return nil, p.errf("expected =")
			}
			p.pos++
			key, haveKey = keyText(k), true
		} else if isIdent(p.s[p.pos]) {
			save := p.pos
			id := p.ident()
			p.ws()
			if p.pos < len(p.s) && p.s[p.pos] == '=' && (p.pos+1 >= len(p.s) || p.s[p.pos+1] != '=') {
				p.pos++
				key, haveKey = id, true
			} else {
				p.pos = save
			}
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if !haveKey {
			key = strconv.Itoa(idx)
			idx++
		}
		if v != nil {
			t[key] = v
		}
		p.ws()
		if p.pos < len(p.s) && (p.s[p.pos] == ',' || p.s[p.pos] == ';') {
			p.pos++
		}
	}
}

// ---- typed access helpers (missing or wrong-typed values give zero values) ----

func (t T) Tab(path ...string) T {
	cur := t
	for _, k := range path {
		if cur == nil {
			return nil
		}
		n, _ := cur[k].(T)
		cur = n
	}
	return cur
}

func (t T) Num(path ...string) float64 {
	if len(path) == 0 || t == nil {
		return 0
	}
	parent := t.Tab(path[:len(path)-1]...)
	if parent == nil {
		return 0
	}
	switch v := parent[path[len(path)-1]].(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	}
	return 0
}

func (t T) Int(path ...string) int { return int(t.Num(path...)) }

func (t T) Str(path ...string) string {
	if len(path) == 0 || t == nil {
		return ""
	}
	parent := t.Tab(path[:len(path)-1]...)
	if parent == nil {
		return ""
	}
	switch v := parent[path[len(path)-1]].(type) {
	case string:
		return v
	case float64:
		return keyText(v)
	}
	return ""
}

func (t T) Bool(path ...string) bool {
	if len(path) == 0 || t == nil {
		return false
	}
	parent := t.Tab(path[:len(path)-1]...)
	if parent == nil {
		return false
	}
	b, _ := parent[path[len(path)-1]].(bool)
	return b
}
