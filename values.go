package octosql

import (
	"log"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
)

// go-sumtype:decl Value
// type Value interface {
//	docs.Documented

func MakeNull() Value {
	return Value{Value: &Value_Null{Null: true}}
}
func ZeroNull() Value {
	return Value{Value: &Value_Null{Null: true}}
}

type Phantom struct{}

func MakePhantom() Value {
	return Value{Value: &Value_Phantom{Phantom: true}}
}
func ZeroPhantom() Value {
	return Value{Value: &Value_Phantom{Phantom: true}}
}

type Int int

func MakeInt(v int) Value {
	return Value{Value: &Value_Int{Int: int64(v)}}
}
func ZeroInt() Value {
	return Value{Value: &Value_Int{Int: int64(0)}}
}

type Float float64

func MakeFloat(v float64) Value {
	return Value{Value: &Value_Float{Float: v}}
}
func ZeroFloat() Value {
	return Value{Value: &Value_Float{Float: 0}}
}

func MakeBool(v bool) Value {
	return Value{Value: &Value_Bool{Bool: v}}
}
func ZeroBool() Value {
	return Value{Value: &Value_Bool{Bool: false}}
}

func MakeString(v string) Value {
	return Value{Value: &Value_String_{String_: v}}
}
func ZeroString() Value {
	return Value{Value: &Value_String_{String_: ""}}
}

func MakeTime(v time.Time) Value {
	t, err := ptypes.TimestampProto(v)
	if err != nil {
		panic(err)
	}
	return Value{Value: &Value_Time{Time: t}}
}
func ZeroTime() Value {
	return Value{Value: &Value_Time{Time: &timestamp.Timestamp{}}}
}

func MakeDuration(v time.Duration) Value {
	return Value{Value: &Value_Duration{Duration: ptypes.DurationProto(v)}}
}
func ZeroDuration() Value {
	return Value{Value: &Value_Duration{Duration: &duration.Duration{}}}
}

func MakeTuple(v []Value) Value {
	tuple := &Tuple{
		Fields: make([]*Value, len(v)),
	}
	for i, v := range v {
		vInternal := v
		tuple.Fields[i] = &vInternal
	}
	return Value{Value: &Value_Tuple{Tuple: tuple}}
}
func ZeroTuple() Value {
	return Value{Value: &Value_Tuple{Tuple: &Tuple{
		Fields: nil,
	}}}
}

func MakeObject(v map[string]Value) Value {
	object := &Object{
		Fields: make(map[string]*Value),
	}
	for k, v := range v {
		vInternal := v
		object.Fields[k] = &vInternal
	}

	return Value{Value: &Value_Object{Object: object}}
}
func ZeroObject() Value {
	return Value{Value: &Value_Object{Object: &Object{
		Fields: nil,
	}}}
}

// NormalizeType brings various primitive types into the type we want them to be.
// All types coming out of data sources have to be already normalized this way.
func NormalizeType(value interface{}) Value {
	switch value := value.(type) {
	case nil:
		return MakeNull()
	case bool:
		return MakeBool(value)
	case int:
		return MakeInt(value)
	case int8:
		return MakeInt(int(value))
	case int32:
		return MakeInt(int(value))
	case int64:
		return MakeInt(int(value))
	case uint8:
		return MakeInt(int(value))
	case uint32:
		return MakeInt(int(value))
	case uint64:
		return MakeInt(int(value))
	case float32:
		return MakeFloat(float64(value))
	case float64:
		return MakeFloat(value)
	case []byte:
		return MakeString(string(value))
	case string:
		return MakeString(value)
	case []interface{}:
		out := make([]Value, len(value))
		for i := range value {
			out[i] = NormalizeType(value[i])
		}
		return MakeTuple(out)
	case map[string]interface{}:
		out := make(map[string]Value)
		for k, v := range value {
			out[k] = NormalizeType(v)
		}
		return MakeObject(out)
	case *interface{}:
		if value != nil {
			return NormalizeType(*value)
		}
		return MakeNull()
	case time.Time:
		return MakeTime(value)
	case time.Duration:
		return MakeDuration(value)
	case struct{}:
		return MakePhantom()
	case Value:
		return value
	}
	log.Fatalf("invalid type to normalize: %s", reflect.TypeOf(value).String())
	panic("unreachable")
}

// octosql.AreEqual checks the equality of the given values, returning false if the types don't match.
func AreEqual(left, right Value) bool {
	return proto.Equal(&left, &right)
}

type Comparison int

const (
	LessThan    Comparison = -1
	Equal       Comparison = 0
	GreaterThan            = 1
)

func Compare(x, y Value) (Comparison, error) {
	switch x.GetType() {
	case TypeInt:
		if y.GetType() != TypeInt {
			return 0, errors.Errorf("type mismatch between values")
		}

		x := x.AsInt()
		y := y.AsInt()

		if x == y {
			return 0, nil
		} else if x < y {
			return -1, nil
		}

		return 1, nil
	case TypeFloat:
		if y.GetType() != TypeFloat {
			return 0, errors.Errorf("type mismatch between values")
		}
		x := x.AsFloat()
		y := y.AsFloat()

		if x == y {
			return 0, nil
		} else if x < y {
			return -1, nil
		}

		return 1, nil
	case TypeString:
		if y.GetType() != TypeString {
			return 0, errors.Errorf("type mismatch between values")
		}

		x := x.AsString()
		y := y.AsString()

		if x == y {
			return 0, nil
		} else if x < y {
			return -1, nil
		}

		return 1, nil
	case TypeTime:
		if y.GetType() != TypeTime {
			return 0, errors.Errorf("type mismatch between values")
		}

		x := x.AsTime()
		y := y.AsTime()

		if x == y {
			return 0, nil
		} else if x.Before(y) {
			return -1, nil
		}

		return 1, nil
	case TypeBool:
		if y.GetType() != TypeBool {
			return 0, errors.Errorf("type mismatch between values")
		}

		x := x.AsBool()
		y := y.AsBool()

		if x == y {
			return 0, nil
		} else if !x && y {
			return -1, nil
		}

		return 1, nil

	case TypeNull, TypePhantom, TypeDuration, TypeTuple, TypeObject:
		return 0, errors.Errorf("unsupported type in sorting")
	}

	panic("unreachable")
}

func ZeroValue() Value {
	return Value{}
}

func (v Value) AsInt() int {
	return int(v.GetInt())
}

func (v Value) AsFloat() float64 {
	return v.GetFloat()
}

func (v Value) AsBool() bool {
	return v.GetBool()
}

func (v Value) AsString() string {
	return v.GetString_()
}

func (v Value) AsTime() time.Time {
	t, err := ptypes.Timestamp(v.GetTime())
	if err != nil {
		panic(err)
	}
	return t
}

func (v Value) AsDuration() time.Duration {
	d, err := ptypes.Duration(v.GetDuration())
	if err != nil {
		panic(err)
	}
	return d
}

func (v Value) AsSlice() []Value {
	t := v.GetTuple()
	out := make([]Value, len(t.Fields))
	for i := range out {
		out[i] = *t.Fields[i]
	}
	return out
}

type Type int

const (
	TypeZero Type = iota
	TypeNull
	TypePhantom
	TypeInt
	TypeFloat
	TypeBool
	TypeString
	TypeTime
	TypeDuration
	TypeTuple
	TypeObject
)

// Można na tych Value pod spodem zdefiniowac GetType i użyć wirtualnych metod, a nie type switch
func (v Value) GetType() Type {
	switch v.Value.(type) {
	case *Value_Null:
		return TypeNull
	case *Value_Phantom:
		return TypePhantom
	case *Value_Int:
		return TypeInt
	case *Value_Float:
		return TypeFloat
	case *Value_Bool:
		return TypeBool
	case *Value_String_:
		return TypeString
	case *Value_Time:
		return TypeTime
	case *Value_Duration:
		return TypeDuration
	case *Value_Tuple:
		return TypeTuple
	case *Value_Object:
		return TypeObject
	default:
		return TypeZero
	}
}
