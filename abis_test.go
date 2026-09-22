//go:generate go run ./cmd/abis abis_test.go
package abis_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/I-Am-Dench/abis"
)

type Environment struct {
	Type string
}

func (e Environment) AppendBinary(buf []byte) ([]byte, error) {
	var b byte

	switch e.Type {
	case "dev":
		b = 1
	case "test":
		b = 2
	default:
		b = 0
	}
	return append(buf, b), nil
}

func (e *Environment) AdvanceBinary(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("advance environment: %d", io.ErrShortBuffer)
	}

	switch data[0] {
	case 1:
		e.Type = "dev"
	case 2:
		e.Type = "test"
	default:
		e.Type = "prod"
	}
	return data[1:], nil
}

type Embedded struct {
	Name     string
	Currency float64
}

type Indicies struct {
	Lower int
	Upper int
}

type Vector struct {
	X, Y float64
}

type Stats struct {
	Speed float32
	Usage float32
	Tasks uint
}

type Packet struct {
	Embedded

	Flag      bool
	Foo       int
	Char      int8
	Short     int16
	Index     int32
	Long      int64
	Unsigned  uint
	Byte      uint8
	Word      uint16
	BigNumber uint32
	Address   uint64
	Rate      float32
	Factor    float64
	Text      string

	Indicies    Indicies
	Environment Environment
	Stats       *Stats
	Points      []Vector
}

func TestBasic(t *testing.T) {
	expected := Packet{
		Embedded: Embedded{
			Name:     "John Smith",
			Currency: 0.76,
		},

		Flag:      true,
		Foo:       123456789,
		Char:      127,
		Short:     17522,
		Index:     1137836247,
		Long:      -190301348123,
		Unsigned:  9876543210,
		Byte:      42,
		Word:      26747,
		BigNumber: 556448427,
		Address:   5192317105623498,
		Rate:      0.23516043,
		Factor:    38.40414168352489,
		Text:      "Lorem ipsum",

		Indicies: Indicies{
			Lower: -1,
			Upper: 12,
		},
		Environment: Environment{
			Type: "test",
		},
		Stats: &Stats{
			Speed: 15,
			Usage: 30.8,
			Tasks: 6,
		},
		Points: []Vector{
			{-0.5, -0.5},
			{-0.5, 0.5},
			{0.5, 0.5},
			{0.5, -0.5},
		},
	}

	data, err := expected.AppendBinary(nil)
	if err != nil {
		t.Fatal(err)
	}

	actual := Packet{}
	if _, err := actual.AdvanceBinary(data); err != nil {
		t.Fatal(err)
	}

	if expected.Embedded.Name != actual.Embedded.Name {
		t.Errorf("expected %s but got %s", expected.Embedded.Name, actual.Embedded.Name)
	}

	if expected.Embedded.Currency != actual.Embedded.Currency {
		t.Errorf("expected %f but got %f", expected.Embedded.Currency, actual.Embedded.Currency)
	}

	if expected.Flag != actual.Flag {
		t.Errorf("expected %t but got %t", expected.Flag, actual.Flag)
	}

	if expected.Foo != actual.Foo {
		t.Errorf("expected %d but got %d", expected.Foo, actual.Foo)
	}

	if expected.Char != actual.Char {
		t.Errorf("expected %d but got %d", expected.Char, actual.Char)
	}

	if expected.Short != actual.Short {
		t.Errorf("expected %d but got %d", expected.Short, actual.Short)
	}

	if expected.Index != actual.Index {
		t.Errorf("expected %d but got %d", expected.Index, actual.Index)
	}

	if expected.Long != actual.Long {
		t.Errorf("expected %d but got %d", expected.Long, actual.Long)
	}

	if expected.Unsigned != actual.Unsigned {
		t.Errorf("expected %d but got %d", expected.Unsigned, actual.Unsigned)
	}

	if expected.Byte != actual.Byte {
		t.Errorf("expected %d but got %d", expected.Byte, actual.Byte)
	}

	if expected.Word != actual.Word {
		t.Errorf("expected %d but got %d", expected.Word, actual.Word)
	}

	if expected.BigNumber != actual.BigNumber {
		t.Errorf("expected %d but got %d", expected.BigNumber, actual.BigNumber)
	}

	if expected.Address != actual.Address {
		t.Errorf("expected %d but got %d", expected.Address, actual.Address)
	}

	if expected.Rate != actual.Rate {
		t.Errorf("expected %f but got %f", expected.Rate, actual.Rate)
	}

	if expected.Factor != actual.Factor {
		t.Errorf("expected %f but got %f", expected.Factor, actual.Factor)
	}

	if expected.Text != actual.Text {
		t.Errorf("expected %s but got %s", expected.Text, actual.Text)
	}

	if expected.Indicies.Lower != actual.Indicies.Lower {
		t.Errorf("expected %d but got %d", expected.Indicies.Lower, actual.Indicies.Lower)
	}

	if expected.Indicies.Upper != actual.Indicies.Upper {
		t.Errorf("expected %d but got %d", expected.Indicies.Upper, actual.Indicies.Upper)
	}

	if expected.Environment.Type != actual.Environment.Type {
		t.Errorf("expected %s but got %s", expected.Environment.Type, actual.Environment.Type)
	}

	if actual.Stats == nil {
		t.Errorf("stats is nil")
	} else {
		if expected.Stats.Speed != actual.Stats.Speed {
			t.Errorf("expected %g but got %g", expected.Stats.Speed, actual.Stats.Speed)
		}

		if expected.Stats.Usage != actual.Stats.Usage {
			t.Errorf("expected %g but got %g", expected.Stats.Usage, actual.Stats.Usage)
		}

		if expected.Stats.Tasks != actual.Stats.Tasks {
			t.Errorf("expected %d but got %d", expected.Stats.Tasks, actual.Stats.Tasks)
		}
	}

	if len(expected.Points) != len(actual.Points) {
		t.Errorf("expected %d points but got %d", len(expected.Points), len(actual.Points))
	} else {
		for i, a := range expected.Points {
			b := actual.Points[i]

			if a.X != b.X {
				t.Errorf("point %d: expected x %g but got %g", i, a.X, b.X)
			}

			if a.Y != b.Y {
				t.Errorf("point %d: expected y %g but got %g", i, a.Y, b.Y)
			}
		}
	}
}

type Custom struct {
	Environment Environment
}

func TestCustom(t *testing.T) {
	expected := Custom{
		Environment: Environment{
			Type: "dev",
		},
	}

	data, err := expected.AppendBinary(nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 1 {
		t.Errorf("expected 1 byte but got %d", len(data))
		return
	}

	if data[0] != 1 {
		t.Errorf("expected 1 but got %d", data[0])
	}
}

type Arrays struct {
	PointerToPoints []*Vector
	RawPoints       []Vector
}

func TestArrays(t *testing.T) {
	expected := Arrays{
		PointerToPoints: []*Vector{
			{1, 1},
			{2, 2},
			{3, 3},
			{4, 4},
			{5, 5},
			{6, 6},
			{7, 7},
			{8, 8},
		},
		RawPoints: []Vector{
			{3.14, 6.28},
			{1000, -1000},
			{14, 52.578990},
		},
	}

	data, err := expected.AppendBinary(nil)
	if err != nil {
		t.Fatal(err)
	}

	actual := Arrays{}
	if _, err := actual.AdvanceBinary(data); err != nil {
		t.Fatal(err)
	}

	if len(expected.PointerToPoints) != len(actual.PointerToPoints) {
		t.Errorf("expected %d PointerToPoints but got %d", len(expected.PointerToPoints), len(actual.PointerToPoints))
	} else {
		for i, a := range expected.PointerToPoints {
			b := actual.PointerToPoints[i]

			if a.X != b.X {
				t.Errorf("point %d: expected x %g but got %g", i, a.X, b.X)
			}

			if a.Y != b.Y {
				t.Errorf("point %d: expected y %g but got %g", i, a.Y, b.Y)
			}
		}
	}

	if len(expected.RawPoints) != len(actual.RawPoints) {
		t.Errorf("expected %d RawPoints but got %d", len(expected.RawPoints), len(actual.RawPoints))
	} else {
		for i, a := range expected.RawPoints {
			b := actual.RawPoints[i]

			if a.X != b.X {
				t.Errorf("point %d: expected x %g but got %g", i, a.X, b.X)
			}

			if a.Y != b.Y {
				t.Errorf("point %d: expected y %g but got %g", i, a.Y, b.Y)
			}
		}
	}
}

type WithAppender struct {
	_ abis.Options `abis:"appender"`
}

type WithAdvancer struct {
	_ abis.Options `abis:"advancer"`
}
