package ipc

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

func EncodeFrameOp(op OpCode, payload any) ([]byte, error) {
	j, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.LittleEndian, int32(op)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(len(j))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(j); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeFrameOp(b []byte) (OpCode, json.RawMessage, error) {
	if len(b) < 8 {
		return 0, nil, fmt.Errorf("buffer too small")
	}
	op := OpCode(binaryLittleEndianToInt32(b[:4]))
	length := int(binaryLittleEndianToInt32(b[4:8]))
	if len(b[8:]) != length {
		return 0, nil, fmt.Errorf("frame length mismatch: expected %d got %d", length, len(b[8:]))
	}
	payload := make(json.RawMessage, length)
	copy(payload, b[8:8+length])
	return op, payload, nil
}

func binaryLittleEndianToInt32(b []byte) int32 {
	var v int32
	_ = binary.Read(bytes.NewReader(b), binary.LittleEndian, &v)
	return v
}
