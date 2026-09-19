package photon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
)

const maxObjectDepth = 64

func MarshalObject(value Object) ([]byte, error) {
	var buf bytes.Buffer
	w := objectWriter{Buffer: &buf}
	if err := w.writeObject(value, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalObject(data []byte) (Object, error) {
	r := bytes.NewReader(data)
	value, err := readObject(r, 0)
	if err != nil {
		return nil, err
	}
	if r.Len() != 0 {
		return nil, protocolError(ErrMalformedMessage, "trailing object data")
	}
	return value, nil
}

func (d ObjectDescriptor) write(w *bytes.Buffer) error {
	w.WriteByte(d.Type)
	switch d.Type {
	case TypeCustom:
		w.WriteByte(d.CustomType)
	case TypeDictionary:
		if d.KeyType == nil || d.ValueType == nil {
			return protocolError(ErrUnsupportedObject, "dictionary type descriptor is incomplete")
		}
		if err := d.KeyType.write(w); err != nil {
			return err
		}
		if err := d.ValueType.write(w); err != nil {
			return err
		}
	}
	return nil
}

func (d ObjectDescriptor) equal(other ObjectDescriptor) bool {
	if d.Type != other.Type || d.CustomType != other.CustomType {
		return false
	}
	if d.Type != TypeDictionary {
		return true
	}
	return d.KeyType != nil && d.ValueType != nil && other.KeyType != nil && other.ValueType != nil &&
		d.KeyType.equal(*other.KeyType) && d.ValueType.equal(*other.ValueType)
}

type objectWriter struct {
	*bytes.Buffer
}

func (w *objectWriter) writeObject(value Object, depth int) error {
	if depth > maxObjectDepth {
		return protocolError(ErrUnsupportedObject, "object nesting is too deep")
	}
	if value == nil {
		return fmt.Errorf("%w: <nil>", ErrUnsupportedObject)
	}
	if err := value.photonType().write(w.Buffer); err != nil {
		return err
	}
	return value.marshalPhoton(w, depth)
}

func (w *objectWriter) writeKnownObject(value Object, expected ObjectDescriptor, depth int) error {
	if depth > maxObjectDepth {
		return protocolError(ErrUnsupportedObject, "object nesting is too deep")
	}
	if value == nil {
		return fmt.Errorf("%w: <nil>", ErrUnsupportedObject)
	}
	if expected.Type == 0 {
		return w.writeObject(value, depth)
	}
	if !value.photonType().equal(expected) {
		return protocolError(ErrUnsupportedObject, "collection contains a value of the wrong type")
	}
	return value.marshalPhoton(w, depth)
}

func (Null) photonType() ObjectDescriptor           { return ObjectDescriptor{Type: TypeNull} }
func (Null) marshalPhoton(*objectWriter, int) error { return nil }

func (Boolean) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeBoolean} }
func (v Boolean) marshalPhoton(w *objectWriter, _ int) error {
	if v {
		w.WriteByte(1)
	} else {
		w.WriteByte(0)
	}
	return nil
}

func (Byte) photonType() ObjectDescriptor                 { return ObjectDescriptor{Type: TypeByte} }
func (v Byte) marshalPhoton(w *objectWriter, _ int) error { return w.WriteByte(byte(v)) }

func (Short) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeShort} }
func (v Short) marshalPhoton(w *objectWriter, _ int) error {
	writeUint16(w, uint16(v))
	return nil
}

func (Integer) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeInteger} }
func (v Integer) marshalPhoton(w *objectWriter, _ int) error {
	writeUint32(w, uint32(v))
	return nil
}

func (Long) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeLong} }
func (v Long) marshalPhoton(w *objectWriter, _ int) error {
	writeUint64(w, uint64(v))
	return nil
}

func (Float) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeFloat} }
func (v Float) marshalPhoton(w *objectWriter, _ int) error {
	writeUint32(w, math.Float32bits(float32(v)))
	return nil
}

func (Double) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeDouble} }
func (v Double) marshalPhoton(w *objectWriter, _ int) error {
	writeUint64(w, math.Float64bits(float64(v)))
	return nil
}

func (String) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeString} }
func (v String) marshalPhoton(w *objectWriter, _ int) error {
	if len(v) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v)))
	w.WriteString(string(v))
	return nil
}

func (ByteArray) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeByteArray} }
func (v ByteArray) marshalPhoton(w *objectWriter, _ int) error {
	if len(v) > math.MaxUint32 {
		return ErrMessageTooLarge
	}
	writeUint32(w, uint32(len(v)))
	w.Write(v)
	return nil
}

func (IntegerArray) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeIntegerArray}
}
func (v IntegerArray) marshalPhoton(w *objectWriter, _ int) error {
	if len(v) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v)))
	for _, n := range v {
		writeUint32(w, n)
	}
	return nil
}

func (StringArray) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeStringArray}
}
func (v StringArray) marshalPhoton(w *objectWriter, _ int) error {
	if len(v) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v)))
	for _, item := range v {
		if err := String(item).marshalPhoton(w, 0); err != nil {
			return err
		}
	}
	return nil
}

func (ObjectArray) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeObjectArray}
}
func (v ObjectArray) marshalPhoton(w *objectWriter, depth int) error {
	if len(v) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v)))
	for _, item := range v {
		if err := w.writeObject(item, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (v Custom) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeCustom, CustomType: v.Code}
}
func (v Custom) marshalPhoton(w *objectWriter, _ int) error {
	if len(v.Data) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v.Data)))
	w.Write(v.Data)
	return nil
}

func (Array) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeArray} }
func (v Array) marshalPhoton(w *objectWriter, depth int) error {
	if len(v.Values) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v.Values)))
	if err := v.Type.write(w.Buffer); err != nil {
		return err
	}
	for _, item := range v.Values {
		if err := w.writeKnownObject(item, v.Type, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (Hashtable) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeHashtable}
}
func (v Hashtable) marshalPhoton(w *objectWriter, depth int) error {
	if len(v) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v)))
	for _, entry := range v {
		if err := w.writeObject(entry.Key, depth+1); err != nil {
			return err
		}
		if err := w.writeObject(entry.Value, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (v Dictionary) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeDictionary, KeyType: &v.KeyType, ValueType: &v.ValueType}
}
func (v Dictionary) marshalPhoton(w *objectWriter, depth int) error {
	if len(v.Entries) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(v.Entries)))
	for _, entry := range v.Entries {
		if err := w.writeKnownObject(entry.Key, v.KeyType, depth+1); err != nil {
			return err
		}
		if err := w.writeKnownObject(entry.Value, v.ValueType, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (OperationRequest) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeOperationRequest}
}
func (v OperationRequest) marshalPhoton(w *objectWriter, depth int) error {
	w.WriteByte(v.Code)
	return marshalParameters(w, v.Parameters, depth)
}

func (OperationResponse) photonType() ObjectDescriptor {
	return ObjectDescriptor{Type: TypeOperationResponse}
}
func (v OperationResponse) marshalPhoton(w *objectWriter, depth int) error {
	w.WriteByte(v.Code)
	writeUint16(w, uint16(v.ReturnCode))
	if v.DebugMessage == nil {
		v.DebugMessage = Null{}
	}
	typeCode := v.DebugMessage.photonType().Type
	if typeCode != TypeNull && typeCode != TypeString {
		return protocolError(ErrUnsupportedObject, "debug message is neither String nor Null")
	}
	if err := w.writeObject(v.DebugMessage, depth+1); err != nil {
		return err
	}
	return marshalParameters(w, v.Parameters, depth)
}

func (Event) photonType() ObjectDescriptor { return ObjectDescriptor{Type: TypeEventData} }
func (v Event) marshalPhoton(w *objectWriter, depth int) error {
	w.WriteByte(v.Code)
	return marshalParameters(w, v.Parameters, depth)
}

func readObject(r *bytes.Reader, depth int) (Object, error) {
	if depth > maxObjectDepth {
		return nil, protocolError(ErrMalformedMessage, "object nesting is too deep")
	}
	descriptor, err := readTypeDescriptor(r)
	if err != nil {
		return nil, err
	}
	return readObjectOfDescriptor(r, descriptor, depth)
}

func readObjectOfDescriptor(r *bytes.Reader, descriptor ObjectDescriptor, depth int) (Object, error) {
	switch descriptor.Type {
	case 0:
		return readObject(r, depth+1)
	case TypeNull:
		return Null{}, nil
	case TypeBoolean:
		b, err := r.ReadByte()
		if err != nil || b > 1 {
			if err == nil {
				err = fmt.Errorf("invalid boolean %d", b)
			}
			return nil, malformedRead(err)
		}
		return Boolean(b == 1), nil
	case TypeByte:
		b, err := r.ReadByte()
		return Byte(b), malformedReadOrNil(err)
	case TypeShort:
		n, err := readUint16(r)
		return Short(n), err
	case TypeInteger:
		n, err := readUint32(r)
		return Integer(n), err
	case TypeLong:
		n, err := readUint64(r)
		return Long(n), err
	case TypeFloat:
		n, err := readUint32(r)
		return Float(math.Float32frombits(n)), err
	case TypeDouble:
		n, err := readUint64(r)
		return Double(math.Float64frombits(n)), err
	case TypeString:
		data, err := readShortBytes(r)
		return String(data), err
	case TypeCustom:
		data, err := readShortBytes(r)
		return Custom{Code: descriptor.CustomType, Data: data}, err
	case TypeByteArray:
		n, err := readUint32(r)
		if err != nil {
			return nil, err
		}
		data, err := readBytes(r, uint64(n))
		return ByteArray(data), err
	case TypeIntegerArray:
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		values := make(IntegerArray, n)
		for i := range values {
			value, err := readUint32(r)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	case TypeStringArray:
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		values := make(StringArray, n)
		for i := range values {
			value, err := readShortBytes(r)
			if err != nil {
				return nil, err
			}
			values[i] = string(value)
		}
		return values, nil
	case TypeObjectArray:
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		values := make(ObjectArray, n)
		for i := range values {
			values[i], err = readObject(r, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	case TypeArray:
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		subtype, err := readTypeDescriptor(r)
		if err != nil {
			return nil, err
		}
		values := make([]Object, n)
		for i := range values {
			values[i], err = readKnownObject(r, subtype, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return Array{Type: subtype, Values: values}, nil
	case TypeHashtable:
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		entries := make(Hashtable, n)
		for i := range entries {
			entries[i].Key, err = readObject(r, depth+1)
			if err != nil {
				return nil, err
			}
			entries[i].Value, err = readObject(r, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return entries, nil
	case TypeDictionary:
		if descriptor.KeyType == nil || descriptor.ValueType == nil {
			return nil, protocolError(ErrMalformedMessage, "dictionary type descriptor is incomplete")
		}
		n, err := readUint16(r)
		if err != nil {
			return nil, err
		}
		dictionary := Dictionary{
			KeyType:   *descriptor.KeyType,
			ValueType: *descriptor.ValueType,
			Entries:   make([]Entry, n),
		}
		for i := range dictionary.Entries {
			dictionary.Entries[i].Key, err = readKnownObject(r, *descriptor.KeyType, depth+1)
			if err != nil {
				return nil, err
			}
			dictionary.Entries[i].Value, err = readKnownObject(r, *descriptor.ValueType, depth+1)
			if err != nil {
				return nil, err
			}
		}
		return dictionary, nil
	case TypeOperationRequest:
		return readOperationRequest(r)
	case TypeOperationResponse:
		return readOperationResponse(r)
	case TypeEventData:
		return readEvent(r)
	default:
		return nil, protocolError(ErrMalformedMessage, fmt.Sprintf("unknown object type %#x", descriptor.Type))
	}
}

func readTypeDescriptor(r *bytes.Reader) (ObjectDescriptor, error) {
	typeCode, err := r.ReadByte()
	if err != nil {
		return ObjectDescriptor{}, malformedRead(err)
	}
	descriptor := ObjectDescriptor{Type: typeCode}
	switch typeCode {
	case TypeCustom:
		descriptor.CustomType, err = r.ReadByte()
	case TypeDictionary:
		keyType, keyErr := readTypeDescriptor(r)
		if keyErr != nil {
			return ObjectDescriptor{}, keyErr
		}
		valueType, valueErr := readTypeDescriptor(r)
		if valueErr != nil {
			return ObjectDescriptor{}, valueErr
		}
		descriptor.KeyType = &keyType
		descriptor.ValueType = &valueType
	}
	if err != nil {
		return ObjectDescriptor{}, malformedRead(err)
	}
	return descriptor, nil
}

func readKnownObject(r *bytes.Reader, descriptor ObjectDescriptor, depth int) (Object, error) {
	return readObjectOfDescriptor(r, descriptor, depth)
}

func marshalParameters(w *objectWriter, parameters Parameters, depth int) error {
	if len(parameters) > math.MaxUint16 {
		return ErrMessageTooLarge
	}
	writeUint16(w, uint16(len(parameters)))
	keys := make([]int, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, int(key))
	}
	sort.Ints(keys)
	for _, key := range keys {
		w.WriteByte(byte(key))
		if err := w.writeObject(parameters[byte(key)], depth+1); err != nil {
			return fmt.Errorf("parameter %d: %w", key, err)
		}
	}
	return nil
}

func unmarshalParameters(r *bytes.Reader) (Parameters, error) {
	n, err := readUint16(r)
	if err != nil {
		return nil, err
	}
	parameters := make(Parameters, n)
	for range n {
		key, err := r.ReadByte()
		if err != nil {
			return nil, malformedRead(err)
		}
		if _, exists := parameters[key]; exists {
			return nil, protocolError(ErrMalformedMessage, fmt.Sprintf("duplicate parameter %d", key))
		}
		parameters[key], err = readObject(r, 0)
		if err != nil {
			return nil, fmt.Errorf("parameter %d: %w", key, err)
		}
	}
	return parameters, nil
}

func MarshalOperationRequest(request OperationRequest) ([]byte, error) {
	var buf bytes.Buffer
	w := objectWriter{Buffer: &buf}
	if err := request.marshalPhoton(&w, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalOperationRequest(data []byte) (OperationRequest, error) {
	r := bytes.NewReader(data)
	request, err := readOperationRequest(r)
	if err != nil {
		return OperationRequest{}, err
	}
	if r.Len() != 0 {
		return OperationRequest{}, protocolError(ErrMalformedMessage, "trailing operation request data")
	}
	return request, nil
}

func readOperationRequest(r *bytes.Reader) (OperationRequest, error) {
	code, err := r.ReadByte()
	if err != nil {
		return OperationRequest{}, malformedRead(err)
	}
	parameters, err := unmarshalParameters(r)
	if err != nil {
		return OperationRequest{}, err
	}
	return OperationRequest{Code: code, Parameters: parameters}, nil
}

func MarshalOperationResponse(response OperationResponse) ([]byte, error) {
	var buf bytes.Buffer
	w := objectWriter{Buffer: &buf}
	if err := response.marshalPhoton(&w, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalOperationResponse(data []byte) (OperationResponse, error) {
	r := bytes.NewReader(data)
	response, err := readOperationResponse(r)
	if err != nil {
		return OperationResponse{}, err
	}
	if r.Len() != 0 {
		return OperationResponse{}, protocolError(ErrMalformedMessage, "trailing operation response data")
	}
	return response, nil
}

func readOperationResponse(r *bytes.Reader) (OperationResponse, error) {
	code, err := r.ReadByte()
	if err != nil {
		return OperationResponse{}, malformedRead(err)
	}
	returnCode, err := readUint16(r)
	if err != nil {
		return OperationResponse{}, err
	}
	debug, err := readObject(r, 0)
	if err != nil {
		return OperationResponse{}, err
	}
	typeCode := debug.photonType().Type
	if typeCode != TypeNull && typeCode != TypeString {
		return OperationResponse{}, protocolError(ErrMalformedMessage, "debug message is neither String nor Null")
	}
	parameters, err := unmarshalParameters(r)
	if err != nil {
		return OperationResponse{}, err
	}
	return OperationResponse{Code: code, ReturnCode: int16(returnCode), DebugMessage: debug, Parameters: parameters}, nil
}

func MarshalEvent(event Event) ([]byte, error) {
	var buf bytes.Buffer
	w := objectWriter{Buffer: &buf}
	if err := event.marshalPhoton(&w, 0); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalEvent(data []byte) (Event, error) {
	r := bytes.NewReader(data)
	event, err := readEvent(r)
	if err != nil {
		return Event{}, err
	}
	if r.Len() != 0 {
		return Event{}, protocolError(ErrMalformedMessage, "trailing event data")
	}
	return event, nil
}

func readEvent(r *bytes.Reader) (Event, error) {
	code, err := r.ReadByte()
	if err != nil {
		return Event{}, malformedRead(err)
	}
	parameters, err := unmarshalParameters(r)
	if err != nil {
		return Event{}, err
	}
	return Event{Code: code, Parameters: parameters}, nil
}

func readShortBytes(r *bytes.Reader) ([]byte, error) {
	n, err := readUint16(r)
	if err != nil {
		return nil, err
	}
	return readBytes(r, uint64(n))
}

func readBytes(r *bytes.Reader, n uint64) ([]byte, error) {
	if n > uint64(r.Len()) {
		return nil, malformedRead(io.ErrUnexpectedEOF)
	}
	data := make([]byte, int(n))
	_, err := io.ReadFull(r, data)
	return data, malformedReadOrNil(err)
}

func writeUint16(w io.Writer, n uint16) { _ = binary.Write(w, binary.BigEndian, n) }
func writeUint32(w io.Writer, n uint32) { _ = binary.Write(w, binary.BigEndian, n) }
func writeUint64(w io.Writer, n uint64) { _ = binary.Write(w, binary.BigEndian, n) }

func readUint16(r io.Reader) (uint16, error) {
	var n uint16
	err := binary.Read(r, binary.BigEndian, &n)
	return n, malformedReadOrNil(err)
}

func readUint32(r io.Reader) (uint32, error) {
	var n uint32
	err := binary.Read(r, binary.BigEndian, &n)
	return n, malformedReadOrNil(err)
}

func readUint64(r io.Reader) (uint64, error) {
	var n uint64
	err := binary.Read(r, binary.BigEndian, &n)
	return n, malformedReadOrNil(err)
}

func malformedRead(err error) error {
	if err == nil {
		return nil
	}
	return protocolError(ErrMalformedMessage, err.Error())
}

func malformedReadOrNil(err error) error {
	if err == nil {
		return nil
	}
	return malformedRead(err)
}

func protocolError(kind error, detail string) error {
	return &ProtocolError{Kind: kind, Detail: detail}
}
