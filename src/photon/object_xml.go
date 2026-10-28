package photon

import (
	"cmp"
	"encoding/hex"
	"encoding/xml"
	"encoding/json/v2"
	"encoding/json/jsontext"
	"slices"
	"strconv"
)

func (d ObjectDescriptor) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	value, err := d.appendXMLType(nil, 0)
	if err != nil {
		return xml.Attr{}, err
	}
	return xml.Attr{Name: name, Value: string(value)}, nil
}

func (d ObjectDescriptor) appendXMLType(value []byte, depth int) ([]byte, error) {
	if depth > maxObjectDepth {
		return nil, protocolError(ErrUnsupportedObject, "object nesting is too deep")
	}
	value = append(value, byte(d.Type))
	switch d.Type {
	case TypeCustom:
		value = strconv.AppendUint(value, uint64(d.CustomType), 10)
	case TypeDictionary:
		if d.KeyType == nil || d.ValueType == nil {
			return nil, protocolError(ErrUnsupportedObject, "dictionary type descriptor is incomplete")
		}
		var err error
		value, err = d.KeyType.appendXMLType(value, depth+1)
		if err != nil {
			return nil, err
		}
		return d.ValueType.appendXMLType(value, depth+1)
	}
	return value, nil
}

func (v ByteArray) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(hex.EncodeToString(v), start)
}

func (v IntegerArray) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalXMLSequence(e, start, v)
}

func (v StringArray) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalXMLSequence(e, start, v)
}

func (v ObjectArray) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalXMLSequence(e, start, v)
}

func (v Custom) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	customContent, err := json.Marshal(v.CustomData, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return e.EncodeElement(struct {
		Code CustomCode `xml:"Code,attr"`
		Data []byte     `xml:",cdata"`
	}{v.CustomCode(), customContent}, start)
}

func (v Array) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	attr, err := v.Type.MarshalXMLAttr(xml.Name{Local: "Type"})
	if err != nil {
		return err
	}
	start.Attr = append(start.Attr, attr)
	return marshalXMLSequence(e, start, v.Values)
}

func (v Entry) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	if err := e.EncodeElement(ObjectArray{v.Key}, xml.StartElement{Name: xml.Name{Local: "Key"}}); err != nil {
		return err
	}
	return e.EncodeElement(ObjectArray{v.Value}, xml.StartElement{Name: xml.Name{Local: "Value"}})
}

func (v Hashtable) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return marshalXMLSequence(e, start, v)
}

func (v Parameters) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	type parameter struct {
		Code  ParamCode
		Value Object
	}
	params := make([]parameter, 0, len(v))
	for code, value := range v {
		params = append(params, parameter{Code: code, Value: value})
	}

	slices.SortFunc(params, func(a, b parameter) int {
		return cmp.Compare(a.Code, b.Code)
	})

	element := xml.StartElement{
		Name: xml.Name{Local: "Parameter"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "Code"}, Value: ""}},
	}
	for _, param := range params {
		element.Attr[0].Value = param.Code.String()
		if err := e.EncodeToken(element); err != nil {
			return err
		}
		if err := e.Encode(param.Value); err != nil {
			return err
		}
		if err := e.EncodeToken(element.End()); err != nil {
			return err
		}
	}
	return nil
}

func (v OperationResponse) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var debug ObjectArray
	switch v.DebugMessage.(type) {
	case nil, Null:
	default:
		debug = ObjectArray{v.DebugMessage}
	}
	return e.EncodeElement(struct {
		Code         OpCode      `xml:"Code,attr"`
		ReturnCode   int16       `xml:"ReturnCode,attr"`
		DebugMessage ObjectArray `xml:"DebugMessage,omitempty"`
		Parameters   Parameters
		Internal     bool `xml:"Internal,attr,omitempty"`
	}{v.Code, v.ReturnCode, debug, v.Parameters, v.Internal}, start)
}

func marshalXMLSequence[T any](e *xml.Encoder, start xml.StartElement, values []T) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, value := range values {
		if err := e.Encode(value); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}
