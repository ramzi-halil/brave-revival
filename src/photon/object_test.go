package photon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectRoundTrip(t *testing.T) {
	stringType := String("").photonType()
	customType := Custom{Code: 9}.photonType()
	values := []Object{
		Null{},
		Boolean(true),
		Byte(7),
		Short(1234),
		Integer(123456),
		Long(123456789),
		Float(1.5),
		Double(-2.25),
		String("hello"),
		ByteArray{1, 2, 3},
		IntegerArray{1, 2, 3},
		StringArray{"a", "b"},
		ObjectArray{String("value"), Integer(4), Null{}},
		Custom{Code: 9, Data: []byte{8, 7}},
		Array{Type: stringType, Values: []Object{String("one"), String("two")}},
		Array{Type: customType, Values: []Object{Custom{Code: 9, Data: []byte{1}}}},
		Hashtable{{Key: String("key"), Value: Integer(5)}},
		Dictionary{KeyType: stringType, ValueType: Integer(0).photonType(), Entries: []Entry{{Key: String("key"), Value: Integer(6)}}},
		Dictionary{KeyType: stringType, ValueType: customType, Entries: []Entry{{Key: String("key"), Value: Custom{Code: 9, Data: []byte{2}}}}},
		OperationRequest{Code: 4, Parameters: Parameters{1: String("request")}},
		OperationResponse{Code: 5, ReturnCode: -1, DebugMessage: Null{}, Parameters: Parameters{2: Boolean(true)}},
		Event{Code: 6, Parameters: Parameters{3: Integer(7)}},
	}

	for _, value := range values {
		encoded, err := MarshalObject(value)
		require.NoErrorf(t, err, "MarshalObject(%T)", value)
		decoded, err := UnmarshalObject(encoded)
		require.NoErrorf(t, err, "UnmarshalObject(%T)", value)
		require.Equalf(t, value, decoded, "round trip %T", value)
	}
}

func TestArrayOfDictionaryEncoding(t *testing.T) {
	keyType := Byte(0).photonType()
	valueType := Short(0).photonType()
	dictionaryType := Dictionary{KeyType: keyType, ValueType: valueType}.photonType()
	want := Array{
		Type: dictionaryType,
		Values: []Object{
			Dictionary{KeyType: keyType, ValueType: valueType, Entries: []Entry{{Key: Byte(4), Value: Short(5)}}},
			Dictionary{KeyType: keyType, ValueType: valueType, Entries: []Entry{{Key: Byte(6), Value: Short(7)}}},
		},
	}
	wantBytes := []byte{'y', 0, 2, 'D', 'b', 'k', 0, 1, 4, 0, 5, 0, 1, 6, 0, 7}

	encoded, err := MarshalObject(want)
	require.NoError(t, err)
	require.Equal(t, wantBytes, encoded)

	decoded, err := UnmarshalObject(wantBytes)
	require.NoError(t, err)
	require.Equal(t, want, decoded)

	empty := Array{Type: dictionaryType, Values: []Object{}}
	emptyBytes := []byte{'y', 0, 0, 'D', 'b', 'k'}
	encoded, err = MarshalObject(empty)
	require.NoError(t, err)
	require.Equal(t, emptyBytes, encoded)
	decoded, err = UnmarshalObject(emptyBytes)
	require.NoError(t, err)
	require.Equal(t, empty, decoded)
}

func TestGoPrimitiveTypesAreNotObjects(t *testing.T) {
	values := []any{
		true,
		byte(1),
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		int(1),
		uint16(1),
		uint32(1),
		uint64(1),
		float32(1),
		float64(1),
		"value",
		[]byte{1},
		[]int32{1},
		[]uint32{1},
		[]string{"value"},
		[]any{String("value")},
		map[string]any{"key": String("value")},
	}
	for _, value := range values {
		require.NotImplementsf(t, (*Object)(nil), value, "%T unexpectedly implements Object", value)
	}
}

func TestOperationResponseRoundTrip(t *testing.T) {
	want := OperationResponse{
		Code:         230,
		ReturnCode:   -3,
		DebugMessage: String("not allowed"),
		Parameters: Parameters{
			1: String("message"),
			2: ByteArray{3, 4},
		},
	}
	encoded, err := MarshalOperationResponse(want)
	require.NoError(t, err)
	got, err := UnmarshalOperationResponse(encoded)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
