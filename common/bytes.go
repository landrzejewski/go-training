package common

import (
	"bytes"
	"encoding/gob"
)

func ToBytes(input interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(input)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func FromBytes(data []byte, output interface{}) error {
	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	err := decoder.Decode(output)
	if err != nil {
		return err
	}
	return nil
}
