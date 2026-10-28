package photon

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectMarshalXML(t *testing.T) {
	stringType := String("").photonType()
	integerType := Integer(0).photonType()
	tests := []struct {
		value Object
		want  string
	}{
		{Null{}, `<Null></Null>`},
		{Boolean(true), `<Boolean>true</Boolean>`},
		{Byte(255), `<Byte>255</Byte>`},
		{Short(1234), `<Short>1234</Short>`},
		{Integer(4294967295), `<Integer>4294967295</Integer>`},
		{Long(18446744073709551615), `<Long>18446744073709551615</Long>`},
		{Float(1.5), `<Float>1.5</Float>`},
		{Double(-2.25), `<Double>-2.25</Double>`},
		{String(`<hello>&`), `<String>&lt;hello&gt;&amp;</String>`},
		{ByteArray("ABCD"), `<ByteArray>41424344</ByteArray>`},
		{ByteArray{0, 255}, `<ByteArray>00ff</ByteArray>`},
		{IntegerArray{1, 4294967295}, `<IntegerArray><Integer>1</Integer><Integer>4294967295</Integer></IntegerArray>`},
		{StringArray{"a", "b"}, `<StringArray><String>a</String><String>b</String></StringArray>`},
		{
			ObjectArray{String("value"), Integer(4), Null{}, ObjectArray{Boolean(false)}},
			`<ObjectArray><String>value</String><Integer>4</Integer><Null></Null><ObjectArray><Boolean>false</Boolean></ObjectArray></ObjectArray>`,
		},
		{
			Custom{Code: 9, Data: []byte{0, 255}},
			`<Custom Code="SkillUnit">00ff</Custom>`,
		},
		{
			Custom{Code: 9, Data: []byte("ABCD")},
			`<Custom Code="SkillUnit">41424344</Custom>`,
		},
		{
			Custom{},
			`<Custom Code="0"></Custom>`,
		},
		{
			Array{Type: stringType, Values: []Object{String("one"), String("two")}},
			`<Array Type="s"><String>one</String><String>two</String></Array>`,
		},
		{
			Hashtable{{Key: String("key"), Value: Integer(5)}, {Key: Byte(7), Value: ByteArray("ABCD")}},
			`<Hashtable><Key><String>key</String></Key><Value><Integer>5</Integer></Value><Key><Byte>7</Byte></Key><Value><ByteArray>41424344</ByteArray></Value></Hashtable>`,
		},
		{
			Dictionary{KeyType: stringType, ValueType: integerType, Entries: []Entry{{Key: String("key"), Value: Integer(6)}, {Key: String("other"), Value: Integer(7)}}},
			`<Dictionary KeyType="s" ValueType="i"><Key><String>key</String></Key><Value><Integer>6</Integer></Value><Key><String>other</String></Key><Value><Integer>7</Integer></Value></Dictionary>`,
		},
		{
			OperationRequest{Code: 4, Parameters: Parameters{255: Short(7), 1: String("request")}, Internal: true},
			`<OperationRequest Code="4" Internal="true"><Parameter Code="1"><String>request</String></Parameter><Parameter Code="RoomName"><Short>7</Short></Parameter></OperationRequest>`,
		},
		{
			OperationResponse{Code: 5, ReturnCode: -1, DebugMessage: String("denied"), Parameters: Parameters{2: Boolean(true)}},
			`<OperationResponse Code="5" ReturnCode="-1"><DebugMessage><String>denied</String></DebugMessage><Parameter Code="2"><Boolean>true</Boolean></Parameter></OperationResponse>`,
		},
		{
			OperationResponse{},
			`<OperationResponse Code="InitEncryption" ReturnCode="0"></OperationResponse>`,
		},
		{
			OperationRequest{},
			`<OperationRequest Code="InitEncryption"></OperationRequest>`,
		},
		{
			OperationResponse{Internal: true, DebugMessage: Null{}},
			`<OperationResponse Code="InitEncryption" ReturnCode="0" Internal="true"></OperationResponse>`,
		},
		{
			OperationResponse{DebugMessage: String("")},
			`<OperationResponse Code="InitEncryption" ReturnCode="0"><DebugMessage><String></String></DebugMessage></OperationResponse>`,
		},
		{
			Event{Code: 6, Parameters: Parameters{3: Integer(7)}},
			`<Event Code="BattleContinue"><Parameter Code="3"><Integer>7</Integer></Parameter></Event>`,
		},
		{ByteArray(nil), `<ByteArray></ByteArray>`},
		{IntegerArray{}, `<IntegerArray></IntegerArray>`},
		{StringArray(nil), `<StringArray></StringArray>`},
		{ObjectArray{}, `<ObjectArray></ObjectArray>`},
		{Hashtable(nil), `<Hashtable></Hashtable>`},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			data, err := xml.Marshal(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(data))
		})
	}
}

func TestObjectMarshalXMLElement(t *testing.T) {
	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)
	start := xml.StartElement{
		Name: xml.Name{Local: "Values"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "a&b"}},
	}
	require.NoError(t, e.EncodeElement(ObjectArray{Short(1234)}, start))
	require.Equal(t, `<Values id="a&amp;b"><Short>1234</Short></Values>`, buf.String())
}

func TestObjectDescriptorMarshalXMLAttr(t *testing.T) {
	dictionaryType := Dictionary{KeyType: Byte(0).photonType(), ValueType: String("").photonType()}.photonType()
	tests := []struct {
		descriptor ObjectDescriptor
		want       string
	}{
		{Integer(0).photonType(), "i"},
		{Array{}.photonType(), "y"},
		{Null{}.photonType(), "*"},
		{Custom{Code: 0}.photonType(), "c0"},
		{Custom{Code: 9}.photonType(), "c9"},
		{Custom{Code: 255}.photonType(), "c255"},
		{dictionaryType, "Dbs"},
		{Dictionary{KeyType: Custom{Code: 9}.photonType(), ValueType: dictionaryType}.photonType(), "Dc9Dbs"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			data, err := xml.Marshal(Array{Type: tt.descriptor})
			require.NoError(t, err)
			require.Equal(t, `<Array Type="`+tt.want+`"></Array>`, string(data))
		})
	}

	data, err := xml.Marshal(Dictionary{KeyType: Custom{Code: 9}.photonType(), ValueType: dictionaryType})
	require.NoError(t, err)
	require.Equal(t, `<Dictionary KeyType="c9" ValueType="Dbs"></Dictionary>`, string(data))
}

func TestObjectDescriptorMarshalXMLAttrInvalid(t *testing.T) {
	stringType := String("").photonType()
	cycle := ObjectDescriptor{Type: TypeDictionary, KeyType: &stringType}
	cycle.ValueType = &cycle
	for _, descriptor := range []ObjectDescriptor{
		{Type: TypeDictionary},
		{Type: TypeDictionary, KeyType: &stringType},
		{Type: TypeDictionary, ValueType: &stringType},
		cycle,
	} {
		_, err := xml.Marshal(Array{Type: descriptor})
		require.ErrorIs(t, err, ErrUnsupportedObject)
	}
}
