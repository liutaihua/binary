package bufio

import (
	"encoding/binary"
	"math"
)

func GetUint16LE(b []byte) uint16 {
	return binary.LittleEndian.Uint16(b)
}

func PutUint16LE(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

func GetUint16BE(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func PutUint16BE(b []byte, v uint16) {
	binary.BigEndian.PutUint16(b, v)
}

func GetUint32LE(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func PutUint32LE(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

func GetUint32BE(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

func PutUint32BE(b []byte, v uint32) {
	binary.BigEndian.PutUint32(b, v)
}

func GetUint64LE(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}

func PutUint64LE(b []byte, v uint64) {
	binary.LittleEndian.PutUint64(b, v)
}

func GetUint64BE(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func PutUint64BE(b []byte, v uint64) {
	binary.BigEndian.PutUint64(b, v)
}

func GetFloat32BE(b []byte) float32 {
	return math.Float32frombits(GetUint32BE(b))
}

func PutFloat32BE(b []byte, v float32) {
	PutUint32BE(b, math.Float32bits(v))
}

func GetFloat32LE(b []byte) float32 {
	return math.Float32frombits(GetUint32LE(b))
}

func PutFloat32LE(b []byte, v float32) {
	PutUint32LE(b, math.Float32bits(v))
}

func GetFloat64BE(b []byte) float64 {
	return math.Float64frombits(GetUint64BE(b))
}

func PutFloat64BE(b []byte, v float64) {
	PutUint64BE(b, math.Float64bits(v))
}

func GetFloat64LE(b []byte) float64 {
	return math.Float64frombits(GetUint64LE(b))
}

func PutFloat64LE(b []byte, v float64) {
	PutUint64LE(b, math.Float64bits(v))
}