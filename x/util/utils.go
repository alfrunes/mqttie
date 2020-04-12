package util

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/alfrunes/mqttie/mqtt"
)

var (
	ErrVarintTooLong = fmt.Errorf("varint too long: > 4 bytes")
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

func ReadVarint(r io.Reader) (v int, n int, err error) {
	var b [1]byte
	// Read up to maximum of 4 bytes
	for i := 0; i < 28; i += 7 {
		N, err := r.Read(b[:])
		n += N
		if err != nil {
			return v, n, err
		}
		v += (int(b[0]&0x7F) << i)
		if b[0]&0x80 == 0 {
			return v, n, nil
		}
	}
	return 0, 4, ErrVarintTooLong
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
		panic(fmt.Errorf("invalid argument type: %T", val))
	}
	return n
}

func ReadValue(r io.Reader, valPtr interface{}, maxLen int) (n int, err error) {
	switch val := valPtr.(type) {
	case *string:
		if maxLen < 2 {
			return 0, mqtt.ErrPacketShort
		}
		var b [2]byte
		n, err = r.Read(b[:])
		if err != nil {
			return n, err
		}
		strLen := binary.BigEndian.Uint16(b[:])
		if int(strLen)+2 > maxLen {
			return n, mqtt.ErrPacketShort
		}
		str := make([]byte, int(strLen))
		N, err := r.Read(str)
		n += N
		*val = string(str)
		return n, err

	case *[]byte:
		if maxLen < 2 {
			return 0, mqtt.ErrPacketShort
		}
		var b [2]byte
		n, err = r.Read(b[:])
		if err != nil {
			return n, err
		}
		dataLen := binary.BigEndian.Uint16(b[:])
		if int(dataLen)+2 > maxLen {
			return n, mqtt.ErrPacketShort
		}
		data := make([]byte, int(dataLen))
		N, err := r.Read(data)
		n += N
		*val = data
		return n, err

	case *uint32:
		if maxLen < 4 {
			return 0, mqtt.ErrPacketShort
		}
		var b [4]byte
		n, err = r.Read(b[:])
		*val = binary.BigEndian.Uint32(b[:])
		return n, err

	case *uint16:
		if maxLen < 2 {
			return 0, mqtt.ErrPacketShort
		}
		var b [2]byte
		n, err = r.Read(b[:])
		*val = binary.BigEndian.Uint16(b[:])
		return n, err

	case *uint8:
		if maxLen < 1 {
			return 0, mqtt.ErrPacketShort
		}
		var b [1]byte
		n, err = r.Read(b[:])
		*val = b[0]
		return n, err

	default:
		panic(fmt.Errorf("invalid argument type: %T", valPtr))
	}
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
