package abis

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// BinaryAdvancer is implemented by an object that will deserialize a
// representation of itself based on the supplied bytes. The AdvanceBinary
// method, should return the provided data slice advanced by the number
// of bytes representing the underlying object.
type BinaryAdvancer interface {
	AdvanceBinary(data []byte) (buf []byte, err error)
}

// Options is an empty, dummy type that can be given struct tags to
// control the generation of serialization methods.
//
// The following example will tell the ABIS tool to only generate
// [BinaryAdvancer] methods:
//
//	type Example struct {
//		_ abis.Options `abis:"advancer"`
//
//		Name string
//		Type int
//	}
type Options struct{}

var order = binary.BigEndian

func AppendBool(buf []byte, b bool) []byte {
	if b {
		return append(buf, 1)
	} else {
		return append(buf, 0)
	}
}

func AppendUint8(buf []byte, v uint8) []byte {
	return append(buf, v)
}

func AppendUint16(buf []byte, v uint16) []byte {
	return order.AppendUint16(buf, v)
}

func AppendUint32(buf []byte, v uint32) []byte {
	return order.AppendUint32(buf, v)
}

func AppendUint64(buf []byte, v uint64) []byte {
	return order.AppendUint64(buf, v)
}

func AppendString(buf []byte, s string) []byte {
	buf = binary.AppendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func AppendArray[S ~[]E, E encoding.BinaryAppender](buf []byte, arr S) ([]byte, error) {
	length := uint64(len(arr))

	buf = binary.AppendUvarint(buf, length)
	for _, v := range arr {
		var err error
		if buf, err = v.AppendBinary(buf); err != nil {
			return buf, err
		}
	}
	return buf, nil
}

func AdvanceBool(buf []byte, b *bool) ([]byte, error) {
	if len(buf) < 1 {
		return buf, fmt.Errorf("advance bool: %w", io.ErrShortBuffer)
	}
	*b = buf[0] != 0

	return buf[1:], nil
}

func AdvanceInt(buf []byte, v *int) ([]byte, error) {
	if len(buf) < 8 {
		return buf, fmt.Errorf("advance int: %w", io.ErrShortBuffer)
	}
	*v = int(order.Uint64(buf))

	return buf[8:], nil
}

func AdvanceInt8(buf []byte, v *int8) ([]byte, error) {
	if len(buf) < 1 {
		return buf, fmt.Errorf("advance int8: %w", io.ErrShortBuffer)
	}
	*v = int8(buf[0])

	return buf[1:], nil
}

func AdvanceInt16(buf []byte, v *int16) ([]byte, error) {
	if len(buf) < 2 {
		return buf, fmt.Errorf("advance int16: %w", io.ErrShortBuffer)
	}
	*v = int16(order.Uint16(buf))

	return buf[2:], nil
}

func AdvanceInt32(buf []byte, v *int32) ([]byte, error) {
	if len(buf) < 4 {
		return buf, fmt.Errorf("advance int32: %w", io.ErrShortBuffer)
	}
	*v = int32(order.Uint32(buf))

	return buf[4:], nil
}

func AdvanceInt64(buf []byte, v *int64) ([]byte, error) {
	if len(buf) < 8 {
		return buf, fmt.Errorf("advance int64: %w", io.ErrShortBuffer)
	}
	*v = int64(order.Uint64(buf))

	return buf[8:], nil
}

func AdvanceUint(buf []byte, v *uint) ([]byte, error) {
	if len(buf) < 8 {
		return buf, fmt.Errorf("advance uint: %w", io.ErrShortBuffer)
	}
	*v = uint(order.Uint64(buf))

	return buf[8:], nil
}

func AdvanceUint8(buf []byte, v *uint8) ([]byte, error) {
	if len(buf) < 1 {
		return buf, fmt.Errorf("advance uint8: %w", io.ErrShortBuffer)
	}
	*v = buf[0]

	return buf[1:], nil
}

func AdvanceUint16(buf []byte, v *uint16) ([]byte, error) {
	if len(buf) < 2 {
		return buf, fmt.Errorf("advance uint16: %w", io.ErrShortBuffer)
	}
	*v = order.Uint16(buf)

	return buf[2:], nil
}

func AdvanceUint32(buf []byte, v *uint32) ([]byte, error) {
	if len(buf) < 4 {
		return buf, fmt.Errorf("advance uint32: %w", io.ErrShortBuffer)
	}
	*v = order.Uint32(buf)

	return buf[4:], nil
}

func AdvanceUint64(buf []byte, v *uint64) ([]byte, error) {
	if len(buf) < 8 {
		return buf, fmt.Errorf("advance uint64: %w", io.ErrShortBuffer)
	}
	*v = order.Uint64(buf)

	return buf[8:], nil
}

func AdvanceFloat32(buf []byte, v *float32) ([]byte, error) {
	if len(buf) < 4 {
		return buf, fmt.Errorf("advance float32: %w", io.ErrShortBuffer)
	}
	*v = math.Float32frombits(order.Uint32(buf))

	return buf[4:], nil
}

func AdvanceFloat64(buf []byte, v *float64) ([]byte, error) {
	if len(buf) < 8 {
		return buf, fmt.Errorf("advance float64: %w", io.ErrShortBuffer)
	}
	*v = math.Float64frombits(order.Uint64(buf))

	return buf[8:], nil
}

func AdvanceString(buf []byte, s *string) ([]byte, error) {
	length, n := binary.Uvarint(buf)
	if n == 0 {
		return buf, fmt.Errorf("advance string: size data is too short")
	}
	buf = buf[n:]

	if len(buf) < int(length) {
		return buf, fmt.Errorf("advance string: %w", io.ErrShortBuffer)
	}
	*s = string(buf[:length])

	return buf[length:], nil
}

type ptrToAdvancer[T any] interface {
	*T
	BinaryAdvancer
}

func AdvanceArray[S ~[]T, T any, E ptrToAdvancer[T]](buf []byte, arr *S) ([]byte, error) {
	length, n := binary.Uvarint(buf)
	if n == 0 {
		return buf, fmt.Errorf("advance array: size data is too short")
	}
	buf = buf[n:]

	if len(buf) < int(length) {
		return buf, fmt.Errorf("advance array: %w", io.ErrShortBuffer)
	}

	a := make(S, length)
	for i := range length {
		var (
			v   T
			err error
		)
		if buf, err = E(&v).AdvanceBinary(buf); err != nil {
			return buf, fmt.Errorf("advance array: %w", err)
		}
		a[i] = v
	}
	*arr = a

	return buf, nil
}
