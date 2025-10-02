package serde

type BinarySerde interface {
	SerializeBinary(value any) ([]byte, error)
	DeserializeBinary(data []byte, valuePtr any) error
}
