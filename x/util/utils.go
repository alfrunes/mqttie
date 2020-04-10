package util

import (
	"encoding/binary"
	"fmt"
	"io"
)

func EncodeUvarint(b []byte, val uint32) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				panic(r)
			}
		}
	}()
	return binary.PutUvarint(b, uint64(val)), nil
}

func GetUvarintLen(val uint64) int {
	var length int
	for {
		length++
		val /= 128
		if val < 1 {
			break
		}
	}
	return length
}

func ReadVarint(r io.Reader) (v uint32, n int, err error) {
	var b [1]byte
	// Read up to maximum of 4 bytes
	for i := 0; i < 32; i += 8 {
		N, err := r.Read(b[:])
		n += N
		if err != nil {
			return v, n, err
		}
		v += (uint32(b[0]&0x7F) << i)
		if b[0]&0x80 == 0 {
			return v, n, err
		}
	}
	return 0, 4, fmt.Errorf("Varint too long: > 4 bytes")
}

func EncodeValue(b []byte, val interface{}) int {
	var n int
	switch v := val.(type) {
	case string:
		l := uint16(len(v))
		binary.BigEndian.PutUint16(b, l)
		n += 2
		n += copy(b[n:], v[:int(l)])

	case []byte:
		l := uint16(len(v))
		binary.BigEndian.PutUint16(b, l)
		n += 2
		n += copy(b[n:], v[:int(l)])

	case uint32:
		binary.BigEndian.PutUint32(b, v)
		n = 4

	case uint16:
		binary.BigEndian.PutUint16(b, v)
		n = 2

	case uint8:
		b[0] = v
		n = 1

	default:
		panic(fmt.Errorf("invalid argument type: %v", val))
	}
	return n
}

func EncodeUTF8(b []byte, str string) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				panic(r)
			}
		}
	}()
	var l int
	if l = len(str); l > 0xFFFF {
		return 0, fmt.Errorf("UTF-8 string too long")
	}
	binary.BigEndian.PutUint16(b, uint16(l))
	copy(b[2:], str)
	return l + 2, nil
}

// TODO: Actually extend from ascii to UTF-8
func WriteUTF8(w io.Writer, str string) (n int, err error) {
	var buf [2]byte
	l := len(str)
	if l > 0xFFFFFFFF {
		return 0, fmt.Errorf("UTF-8 string too long")
	}
	binary.BigEndian.PutUint16(buf[:], uint16(l))
	sb := []byte(str)
	return w.Write(append(buf[:], sb...))
}

func ReadUTF8(r io.Reader) (str string, n int, err error) {
	var b [2]byte
	n, err = r.Read(b[:])
	if err != nil {
		return "", n, err
	} else if n < 2 {
		return "", n, io.ErrUnexpectedEOF
	}
	l := binary.BigEndian.Uint16(b[:])

	ret := make([]byte, int(l))
	N, err := r.Read(ret)
	n += N
	if err != nil {
		return "", n, err
	} else if n < len(ret) {
		return "", n, io.ErrUnexpectedEOF
	}
	return string(ret), n, nil
}
