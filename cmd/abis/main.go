package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

var (
	Info  = log.New(os.Stdout, "abis: ", 0)
	Error = log.New(os.Stderr, "abis: ", 0)
)

const (
	genSuffix      = "_abis"
	basicExtension = genSuffix + ".go"
	testExtension  = genSuffix + "_test.go"

	moduleName = "github.com/I-Am-Dench/abis"
)

func findInterfaceType(packageName string, typeName string) (*types.Interface, error) {
	config := packages.Config{
		Mode: packages.LoadTypes,
	}

	encodingPkg, err := packages.Load(&config, packageName)
	if err != nil {
		return nil, err
	}

	for _, pkg := range encodingPkg {
		obj := pkg.Types.Scope().Lookup(typeName)
		if obj != nil {
			if inter, ok := obj.Type().Underlying().(*types.Interface); ok {
				return inter, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to find interface %s in %s", typeName, packageName)
}

var (
	binaryAppenderType *types.Interface
	binaryAdvancerType *types.Interface
)

func init() {
	var err error

	binaryAppenderType, err = findInterfaceType("encoding", "BinaryAppender")
	if err != nil {
		Error.Fatal(err)
	}

	binaryAdvancerType, err = findInterfaceType(moduleName, "BinaryAdvancer")
	if err != nil {
		Error.Fatal(err)
	}
}

type Converter interface {
	WriteAppender(buf *bytes.Buffer, receiver, fieldName string)
	WriteAdvancer(buf *bytes.Buffer, receiver, fieldName string)
}

type basicConverter struct {
	AppenderName string
	AdvancerName string
	TypeCast     string
}

func (f basicConverter) WriteAppender(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tbuf = abis.")
	buf.WriteString(f.AppenderName)
	buf.WriteString("(buf, ")

	if len(f.TypeCast) > 0 {
		buf.WriteString(f.TypeCast)
		buf.WriteString("(")
	}

	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)

	if len(f.TypeCast) > 0 {
		buf.WriteString(")")
	}

	buf.WriteString(")\n")
}

func (f basicConverter) WriteAdvancer(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tif data, err = abis.")
	buf.WriteString(f.AdvancerName)
	buf.WriteString("(data, &")
	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)
	buf.WriteString("); err != nil {\n\t\treturn data, abis.NewAdvanceError(\"")
	buf.WriteString(fieldName)
	buf.WriteString("\", nil)\n\t}\n")
}

var basicConverters = map[types.BasicKind]basicConverter{
	types.Bool:    {"AppendBool", "AdvanceBool", ""},
	types.Int:     {"AppendUint64", "AdvanceInt", "uint64"},
	types.Int8:    {"AppendUint8", "AdvanceInt8", "uint8"},
	types.Int16:   {"AppendUint16", "AdvanceInt16", "uint16"},
	types.Int32:   {"AppendUint32", "AdvanceInt32", "uint32"},
	types.Int64:   {"AppendUint64", "AdvanceInt64", "uint64"},
	types.Uint:    {"AppendUint64", "AdvanceUint", "uint64"},
	types.Uint8:   {"AppendUint8", "AdvanceUint8", ""},
	types.Uint16:  {"AppendUint16", "AdvanceUint16", ""},
	types.Uint32:  {"AppendUint32", "AdvanceUint32", ""},
	types.Uint64:  {"AppendUint64", "AdvanceUint64", ""},
	types.Float32: {"AppendUint32", "AdvanceFloat32", "math.Float32bits"},
	types.Float64: {"AppendUint64", "AdvanceFloat64", "math.Float64bits"},
	types.String:  {"AppendString", "AdvanceString", ""},
}

type namedConverter struct{}

func (f namedConverter) WriteAppender(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tif buf, err = ")
	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)
	buf.WriteString(".AppendBinary(buf); err != nil {\n\t\treturn buf, err\n\t}\n")
}

func (f namedConverter) WriteAdvancer(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tif data, err = ")
	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)
	buf.WriteString(".AdvanceBinary(data); err != nil {\n\t\treturn data, abis.NewAdvanceError(\"")
	buf.WriteString(fieldName)
	buf.WriteString("\", err)\n\t}\n")
}

type sliceConverter struct{}

func (f sliceConverter) WriteAppender(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tif buf, err = abis.AppendArray(buf, ")
	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)
	buf.WriteString("); err != nil {\n\t\treturn buf, err\n\t}\n")
}

func (f sliceConverter) WriteAdvancer(buf *bytes.Buffer, receiver, fieldName string) {
	buf.WriteString("\tif data, err = abis.AdvanceArray(data, &")
	buf.WriteString(receiver)
	buf.WriteString(".")
	buf.WriteString(fieldName)
	buf.WriteString("); err != nil {\n\t\treturn data, abis.NewAdvanceError(\"")
	buf.WriteString(fieldName)
	buf.WriteString("\", err)\n\t}\n")
}

type Field struct {
	Name string
	Type types.Type
}

func (f Field) Converter() (Converter, error) {
	switch v := f.Type.(type) {
	case *types.Basic:
		converter, ok := basicConverters[v.Kind()]
		if !ok {
			return converter, fmt.Errorf("unhandled kind: %s", v.Name())
		}
		return converter, nil
	case *types.Named:
		return namedConverter{}, nil
	case *types.Slice:
		return sliceConverter{}, nil
	default:
		return nil, fmt.Errorf("unhandled type: %s", f.Type.String())
	}
}

type Packet struct {
	Name   string
	Fields []Field
	FieldsInfo

	IsBinaryAppender bool
	IsBinaryAdvancer bool
}

func (p Packet) WriteAppender(w io.Writer) error {
	receiver := string(strings.ToLower(p.Name)[0])

	buf := bytes.Buffer{}
	fmt.Fprintf(&buf, "\nfunc (%v %v) AppendBinary(buf []byte) ([]byte, error) {\n", receiver, p.Name)

	if p.HasNamedField {
		buf.WriteString("\tvar err error\n")
	}

	for _, f := range p.Fields {
		converter, err := f.Converter()
		if err != nil {
			return fmt.Errorf("get appender: %v", err)
		}

		converter.WriteAppender(&buf, receiver, f.Name)
	}

	buf.WriteString("\treturn buf, nil\n}\n")

	_, err := io.Copy(w, &buf)
	return err
}

func (p Packet) WriteAdvancer(w io.Writer) error {
	receiver := string(strings.ToLower(p.Name)[0])

	buf := bytes.Buffer{}
	fmt.Fprintf(&buf, "\nfunc (%v *%v) AdvanceBinary(data []byte) ([]byte, error) {\n\tvar err error\n", receiver, p.Name)

	for _, f := range p.Fields {
		converter, err := f.Converter()
		if err != nil {
			return fmt.Errorf("get advancer: %v", err)
		}

		converter.WriteAdvancer(&buf, receiver, f.Name)
	}

	buf.WriteString("\treturn data, nil\n}\n")

	_, err := io.Copy(w, &buf)
	return err
}

type Package struct {
	Name    string
	Packets []Packet
}

func (p Package) Imports() []string {
	imports := []string{}
	for _, pkg := range p.Packets {
		if pkg.NeedsMathPkg {
			imports = append(imports, "math")
			break
		}
	}
	imports = append(imports, moduleName)

	return imports
}

func (p Package) WriteFile(name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}

	buf := []byte("// auto-generated by abis; created on ")
	buf = time.Now().AppendFormat(buf, time.RFC3339)
	buf = append(buf, "\npackage "...)
	buf = append(buf, p.Name...)
	buf = append(buf, "\n\n"...)

	imports := p.Imports()
	if len(imports) == 1 {
		buf = append(buf, "import \""...)
		buf = append(buf, imports[0]...)
		buf = append(buf, "\"\n"...)
	} else if len(imports) > 1 {
		buf = append(buf, "import (\n"...)
		for _, im := range imports {
			buf = append(buf, "\t\""...)
			buf = append(buf, im...)
			buf = append(buf, "\"\n"...)
		}
		buf = append(buf, ")\n"...)
	}

	if _, err := file.Write(buf); err != nil {
		return err
	}

	for _, packet := range p.Packets {
		if !packet.IsBinaryAppender {
			if err := packet.WriteAppender(file); err != nil {
				return fmt.Errorf("write appender: %v", err)
			}
		}

		if !packet.IsBinaryAdvancer {
			if err := packet.WriteAdvancer(file); err != nil {
				return fmt.Errorf("write advancer: %v", err)
			}
		}
	}
	return nil
}

type FieldsInfo struct {
	HasNamedField bool
	NeedsMathPkg  bool
}

func GetFields(info *types.Struct) (fields []Field, fieldsInfo FieldsInfo) {
	fields = []Field{}
	for field := range info.Fields() {
		if _, ok := field.Type().(*types.Named); ok {
			fieldsInfo.HasNamedField = true
		}

		if t, ok := field.Type().(*types.Basic); ok && (t.Kind() == types.Float32 || t.Kind() == types.Float64) {
			fieldsInfo.NeedsMathPkg = true
		}

		fields = append(fields, Field{
			Name: field.Name(),
			Type: field.Type(),
		})
	}
	return fields, fieldsInfo
}

func GetPackages(name string) ([]Package, error) {
	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles | packages.NeedForTest,
		Dir:  filepath.Dir(name),
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			// We do not want to include generated files during parsing. Parsing the generated files
			// would mean that ALL structures that have already been generated would appear to implement
			// [encoding.BinaryAppender] or [abis.BinaryAdvancer].
			if strings.HasSuffix(filename, basicExtension) || strings.HasSuffix(filename, testExtension) {
				return nil, nil
			}

			// The default parsing behavior as it appears in golang.org/x/tools/go/packages
			const mode = parser.AllErrors | parser.ParseComments
			return parser.ParseFile(fset, filename, src, mode)
		},
		Tests: true,
	}

	pkgs, err := packages.Load(&cfg)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%s: no package found", name)
	}

	found := []Package{}
	for _, pkg := range pkgs {
		packets := []Packet{}
		for _, def := range pkg.TypesInfo.Defs {
			if def == nil {
				continue
			}

			position := pkg.Fset.Position(def.Pos())
			if position.Filename != name {
				continue
			}

			if info, ok := def.Type().Underlying().(*types.Struct); ok && def.Exported() && def.Parent() != nil {
				fields, fieldsInfo := GetFields(info)

				packets = append(packets, Packet{
					Name:       def.Name(),
					Fields:     fields,
					FieldsInfo: fieldsInfo,

					IsBinaryAppender: types.Implements(def.Type(), binaryAppenderType),
					IsBinaryAdvancer: types.Implements(def.Type(), binaryAdvancerType) || types.Implements(types.NewPointer(def.Type()), binaryAdvancerType),
				})
			}
		}
		slices.SortFunc(packets, func(a, b Packet) int { return strings.Compare(a.Name, b.Name) })

		if len(packets) == 0 {
			continue
		}

		found = append(found, Package{
			Name:    pkg.Name,
			Packets: packets,
		})
	}

	return found, nil
}

func main() {
	flag.Parse()

	targets := []string{}
	for i := range flag.NArg() {
		target := flag.Arg(i)
		if !strings.HasSuffix(target, ".go") {
			continue
		}

		if stat, err := os.Stat(target); err != nil {
			Error.Fatal(err)
		} else if stat.IsDir() {
			Error.Fatalf("%s: target is a directory", target)
		}

		targets = append(targets, flag.Arg(i))
	}

	if len(targets) == 0 {
		Error.Fatal("missing file names")
	}

	for _, target := range targets {
		abs, err := filepath.Abs(target)
		if err != nil {
			Error.Fatal(err)
		}

		pkgs, err := GetPackages(abs)
		if err != nil {
			Error.Fatal(err)
		}

		for _, pkg := range pkgs {
			ext := "_test.go"

			outputFile, ok := strings.CutSuffix(target, ext)
			if !ok {
				ext = filepath.Ext(target)
				outputFile, _ = strings.CutSuffix(target, ext)
			}

			outputFile += genSuffix + ext
			if err := pkg.WriteFile(outputFile); err != nil {
				Error.Fatal(err)
			}
		}
	}
}
