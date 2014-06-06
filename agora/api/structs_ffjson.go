// DO NOT EDIT!
// Code generated by ffjson <https://github.com/pquerna/ffjson>
// source: structs.go
// DO NOT EDIT!

package api

import (
	"bytes"

	"encoding/json"

	"unicode/utf8"
)

func (mj *App) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(1024)
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *App) MarshalJSONBuf(buf *bytes.Buffer) error {
	var err error
	var obj []byte
	var first bool = true
	_ = obj
	_ = err
	_ = first
	buf.WriteString(`{`)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"components":`)
	if mj.Components != nil {
		buf.WriteString(`[`)
		for i, v := range mj.Components {
			if i != 0 {
				buf.WriteString(`,`)
			}
			err = v.MarshalJSONBuf(buf)
			if err != nil {
				return err
			}
		}
		buf.WriteString(`]`)
	} else {
		buf.WriteString(`null`)
	}
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"name":`)
	ffjson_WriteJsonString(buf, mj.Name)
	buf.WriteString(`}`)
	return nil
}

func (mj *AppComponent) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(1024)
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *AppComponent) MarshalJSONBuf(buf *bytes.Buffer) error {
	var err error
	var obj []byte
	var first bool = true
	_ = obj
	_ = err
	_ = first
	buf.WriteString(`{`)
	if len(mj.Command) != 0 {
		if first == true {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"command":`)
		ffjson_WriteJsonString(buf, mj.Command)
	}
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"component_type":`)
	ffjson_WriteJsonString(buf, mj.ComponentType)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"cpus":`)
	ffjson_FormatBits(buf, uint64(mj.Cpus), 10, mj.Cpus < 0)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"dist_url":`)
	ffjson_WriteJsonString(buf, mj.DistURL)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"distribution":`)
	ffjson_WriteJsonString(buf, mj.Distribution)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"env":`)
	/* Falling back. type=map[string]string kind=map */
	obj, err = json.Marshal(mj.Env)
	if err != nil {
		return err
	}
	buf.Write(obj)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"mem":`)
	ffjson_FormatBits(buf, uint64(mj.Mem), 10, mj.Mem < 0)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"name":`)
	ffjson_WriteJsonString(buf, mj.Name)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"ports":`)
	/* Falling back. type=map[string]int kind=map */
	obj, err = json.Marshal(mj.Ports)
	if err != nil {
		return err
	}
	buf.Write(obj)
	if first == true {
		first = false
	} else {
		buf.WriteString(`,`)
	}
	buf.WriteString(`"version":`)
	ffjson_WriteJsonString(buf, mj.Version)
	if len(mj.WorkDir) != 0 {
		if first == true {
			first = false
		} else {
			buf.WriteString(`,`)
		}
		buf.WriteString(`"work_dir":`)
		ffjson_WriteJsonString(buf, mj.WorkDir)
	}
	buf.WriteString(`}`)
	return nil
}

func ffjson_WriteJsonString(buf *bytes.Buffer, s string) {
	const hex = "0123456789abcdef"

	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\n':
				buf.WriteByte('\\')
				buf.WriteByte('n')
			case '\r':
				buf.WriteByte('\\')
				buf.WriteByte('r')
			default:

				buf.WriteString(`\u00`)
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\ufffd`)
			i += size
			start = i
			continue
		}

		if c == '\u2028' || c == '\u2029' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString(`\u202`)
			buf.WriteByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	buf.WriteByte('"')
}

func ffjson_FormatBits(dst *bytes.Buffer, u uint64, base int, neg bool) {
	const (
		digits   = "0123456789abcdefghijklmnopqrstuvwxyz"
		digits01 = "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
		digits10 = "0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999"
	)

	var shifts = [len(digits) + 1]uint{
		1 << 1: 1,
		1 << 2: 2,
		1 << 3: 3,
		1 << 4: 4,
		1 << 5: 5,
	}

	if base < 2 || base > len(digits) {
		panic("strconv: illegal AppendInt/FormatInt base")
	}

	var a [64 + 1]byte
	i := len(a)

	if neg {
		u = -u
	}

	if base == 10 {

		for u >= 100 {
			i -= 2
			q := u / 100
			j := uintptr(u - q*100)
			a[i+1] = digits01[j]
			a[i+0] = digits10[j]
			u = q
		}
		if u >= 10 {
			i--
			q := u / 10
			a[i] = digits[uintptr(u-q*10)]
			u = q
		}

	} else if s := shifts[base]; s > 0 {

		b := uint64(base)
		m := uintptr(b) - 1
		for u >= b {
			i--
			a[i] = digits[uintptr(u)&m]
			u >>= s
		}

	} else {

		b := uint64(base)
		for u >= b {
			i--
			a[i] = digits[uintptr(u%b)]
			u /= b
		}
	}

	i--
	a[i] = digits[uintptr(u)]

	if neg {
		i--
		a[i] = '-'
	}

	dst.Write(a[i:])

	return
}