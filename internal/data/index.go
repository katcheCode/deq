package data

import (
	"bytes"
	"errors"
	"strings"
)

// IndexKey is a key for custom indexes of events. It can be marshalled and used in a key-value
// store.
//
// The marshalled format of an IndexKey is:
// IndexTag + Sep + Topic + Sep + Type + Sep + Value
type IndexKeyV1_0_0 struct {
	// Topic must not contain the null character
	Topic string
	// Value must not contain the null character.
	Value string
	ID    string
}

func (key IndexKeyV1_0_0) isKey() {}

// Size returns the length of this key's marshalled data. The result is only
// valid until the key is modified.
func (key IndexKeyV1_0_0) Size() int {
	return len(key.Topic) + len(key.Value) + len(key.ID) + 4
}

// Marshal marshals a key into a byte slice, prefixed according to the key's type.
//
// If buf is nil or has insufficient capacity, a new buffer is allocated. Marshal returns the
// slize that index was marshalled to.
func (key IndexKeyV1_0_0) Marshal(buf []byte) ([]byte, error) {

	if key.Topic == "" {
		return nil, errors.New("Topic is required")
	}
	if key.ID == "" {
		return nil, errors.New("ID is required")
	}
	if strings.ContainsRune(key.Topic, 0) {
		return nil, errors.New("Topic cannot contain null character")
	}
	if strings.ContainsRune(key.Value, 0) {
		return nil, errors.New("Value cannot contain null character")
	}

	size := key.Size()
	if cap(buf) < size {
		buf = make([]byte, 0, size)
	} else {
		buf = buf[:0]
	}

	buf = append(buf, IndexTag, Sep)
	buf = append(buf, key.Topic...)
	buf = append(buf, Sep)
	buf = append(buf, key.Value...)
	buf = append(buf, Sep)
	buf = append(buf, key.ID...)
	return buf, nil
}

// UnmarshalIndexKey unmarshals a key marshaled by key.Marshal()
func UnmarshalIndexKeyV1_0_0(buf []byte, key *IndexKeyV1_0_0) error {
	i := bytes.IndexByte(buf, Sep)
	if i == -1 {
		return errors.New("parse tag: null terminator not found")
	}
	var comparisonTag = [...]byte{IndexTag}
	if !bytes.Equal(buf[:i], comparisonTag[:]) {
		return errors.New("buf does not contain an IndexKey")
	}
	j := bytes.IndexByte(buf[i+1:], Sep) + i + 1
	if j == -1 {
		return errors.New("parse Topic: null terminator not found")
	}
	k := bytes.IndexByte(buf[j+1:], Sep) + j + 1
	if k == -1 {
		return errors.New("parse Type: null terminator not found")
	}

	key.Topic = string(buf[i+1 : j])
	key.Value = string(buf[j+1 : k])
	key.ID = string(buf[k+1:])
	return nil
}
