package types

import "fmt"

type Kind int

const (
	KindVoid Kind = iota
	KindInt8
	KindInt64
	KindBool
	KindString
	KindStruct
	KindClass
	KindEnum
	KindArray
	KindSlice
	KindRaw
	KindFn
	KindNil
)

type Type interface {
	Kind() Kind
	String() string
	Size() int     // size in bytes
	IsRef() bool   // is reference-counted class type?
	IsValue() bool // is value struct or primitive?
}

// Primitive types
type VoidType struct{}

func (t *VoidType) Kind() Kind       { return KindVoid }
func (t *VoidType) String() string   { return "void" }
func (t *VoidType) Size() int        { return 0 }
func (t *VoidType) IsRef() bool      { return false }
func (t *VoidType) IsValue() bool    { return true }

type Int8Type struct{}

func (t *Int8Type) Kind() Kind       { return KindInt8 }
func (t *Int8Type) String() string   { return "i8" }
func (t *Int8Type) Size() int        { return 1 }
func (t *Int8Type) IsRef() bool      { return false }
func (t *Int8Type) IsValue() bool    { return true }

type Int64Type struct{}

func (t *Int64Type) Kind() Kind       { return KindInt64 }
func (t *Int64Type) String() string   { return "i64" }
func (t *Int64Type) Size() int        { return 8 }
func (t *Int64Type) IsRef() bool      { return false }
func (t *Int64Type) IsValue() bool    { return true }

type BoolType struct{}

func (t *BoolType) Kind() Kind       { return KindBool }
func (t *BoolType) String() string   { return "bool" }
func (t *BoolType) Size() int        { return 8 }
func (t *BoolType) IsRef() bool      { return false }
func (t *BoolType) IsValue() bool    { return true }

type StringType struct{}

func (t *StringType) Kind() Kind       { return KindString }
func (t *StringType) String() string   { return "string" }
func (t *StringType) Size() int        { return 8 }
func (t *StringType) IsRef() bool      { return false }
func (t *StringType) IsValue() bool    { return true }

type NilType struct{}

func (t *NilType) Kind() Kind       { return KindNil }
func (t *NilType) String() string   { return "nil" }
func (t *NilType) Size() int        { return 8 }
func (t *NilType) IsRef() bool      { return false }
func (t *NilType) IsValue() bool    { return false }

// Struct Field
type FieldInfo struct {
	Name   string
	Type   Type
	Offset int
}

// StructType: value type
type StructType struct {
	Name      string
	Fields    []FieldInfo
	TotalSize int
}

func (t *StructType) Kind() Kind       { return KindStruct }
func (t *StructType) String() string   { return t.Name }
func (t *StructType) Size() int        { return t.TotalSize }
func (t *StructType) IsRef() bool      { return false }
func (t *StructType) IsValue() bool    { return true }

func (t *StructType) GetField(name string) (*FieldInfo, int, bool) {
	for i, f := range t.Fields {
		if f.Name == name {
			return &t.Fields[i], i, true
		}
	}
	return nil, -1, false
}

// ClassType: reference type (header 16 bytes: [refcount: i64][type_id: i64][fields...])
type ClassType struct {
	Name         string
	TypeId       int64
	Fields       []FieldInfo
	PayloadSize  int  // fields size (not including 16-byte header)
	HasDeinit    bool // user defined deinit
}

func (t *ClassType) Kind() Kind       { return KindClass }
func (t *ClassType) String() string   { return t.Name }
func (t *ClassType) Size() int        { return 8 } // pointer size
func (t *ClassType) IsRef() bool      { return true }
func (t *ClassType) IsValue() bool    { return false }

func (t *ClassType) GetField(name string) (*FieldInfo, int, bool) {
	for i, f := range t.Fields {
		if f.Name == name {
			return &t.Fields[i], i, true
		}
	}
	return nil, -1, false
}

// RawType: Unsafe raw pointer escape hatch
type RawType struct {
	BaseType Type
}

func (t *RawType) Kind() Kind       { return KindRaw }
func (t *RawType) String() string   { return fmt.Sprintf("raw %s", t.BaseType.String()) }
func (t *RawType) Size() int        { return 8 }
func (t *RawType) IsRef() bool      { return false }
func (t *RawType) IsValue() bool    { return false }

// EnumVariantInfo
type EnumVariantInfo struct {
	Name        string
	Tag         int64
	PayloadType []Type
}

// EnumType: Tagged union (Value type: 8-byte tag + max payload size)
type EnumType struct {
	Name        string
	Variants    []EnumVariantInfo
	PayloadSize int
}

func (t *EnumType) Kind() Kind       { return KindEnum }
func (t *EnumType) String() string   { return t.Name }
func (t *EnumType) Size() int        { return 8 + t.PayloadSize }
func (t *EnumType) IsRef() bool      { return false }
func (t *EnumType) IsValue() bool    { return true }

func (t *EnumType) GetVariant(name string) (*EnumVariantInfo, int, bool) {
	for i, v := range t.Variants {
		if v.Name == name {
			return &t.Variants[i], i, true
		}
	}
	return nil, -1, false
}

// ArrayType: Fixed size array [T; N]
type ArrayType struct {
	ElemType Type
	Length   int64
}

func (t *ArrayType) Kind() Kind       { return KindArray }
func (t *ArrayType) String() string   { return fmt.Sprintf("[%s; %d]", t.ElemType.String(), t.Length) }
func (t *ArrayType) Size() int        { return int(t.Length) * t.ElemType.Size() }
func (t *ArrayType) IsRef() bool      { return false }
func (t *ArrayType) IsValue() bool    { return true }

// SliceType: Dynamic slice view []T (fat pointer: ptr + len)
type SliceType struct {
	ElemType Type
}

func (t *SliceType) Kind() Kind       { return KindSlice }
func (t *SliceType) String() string   { return "[]" + t.ElemType.String() }
func (t *SliceType) Size() int        { return 16 } // ptr (8) + len (8)
func (t *SliceType) IsRef() bool      { return false }
func (t *SliceType) IsValue() bool    { return true }

// FnType
type FnType struct {
	Name       string
	ParamNames []string
	ParamTypes []Type
	ReturnType Type
}

func (t *FnType) Kind() Kind       { return KindFn }
func (t *FnType) String() string   { return fmt.Sprintf("fn(%s) -> %s", t.ParamTypes, t.ReturnType) }
func (t *FnType) Size() int        { return 8 }
func (t *FnType) IsRef() bool      { return false }
func (t *FnType) IsValue() bool    { return false }

var (
	Void   = &VoidType{}
	Int8   = &Int8Type{}
	Int64  = &Int64Type{}
	Bool   = &BoolType{}
	String = &StringType{}
	Nil    = &NilType{}
)

func Equal(a, b Type) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind() != b.Kind() {
		if (a.Kind() == KindInt8 && b.Kind() == KindInt64) || (a.Kind() == KindInt64 && b.Kind() == KindInt8) {
			return true
		}
		if (a.Kind() == KindClass || a.Kind() == KindRaw) && b.Kind() == KindNil {
			return true
		}
		if (b.Kind() == KindClass || b.Kind() == KindRaw) && a.Kind() == KindNil {
			return true
		}
		return false
	}
	switch ta := a.(type) {
	case *StructType:
		tb, ok := b.(*StructType)
		return ok && ta.Name == tb.Name
	case *ClassType:
		tb, ok := b.(*ClassType)
		return ok && ta.Name == tb.Name
	case *EnumType:
		tb, ok := b.(*EnumType)
		return ok && ta.Name == tb.Name
	case *ArrayType:
		tb, ok := b.(*ArrayType)
		return ok && ta.Length == tb.Length && Equal(ta.ElemType, tb.ElemType)
	case *SliceType:
		tb, ok := b.(*SliceType)
		return ok && Equal(ta.ElemType, tb.ElemType)
	case *RawType:
		tb, ok := b.(*RawType)
		return ok && Equal(ta.BaseType, tb.BaseType)
	default:
		return true
	}
}
