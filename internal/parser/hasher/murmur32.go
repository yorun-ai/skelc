package hasher

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"

	"go.yorun.ai/skelc/internal/util/checkutil"
)

type _NamedValue struct {
	SkelName string `json:"skelName"`
	Hash     string `json:"hash"`
}

const (
	murmurC1 uint32 = 0xcc9e2d51
	murmurC2 uint32 = 0x1b873593
)

func hashValue(value any) string {
	serialized, err := json.Marshal(value)
	checkutil.CheckNilError(err, "marshal hash value failed")
	return fmt.Sprintf("%08x", murmur32(serialized))
}

func cycleHash(kind string, skelName string) string {
	return hashValue(struct {
		Kind     string `json:"kind"`
		SkelName string `json:"skelName"`
	}{
		Kind:     kind,
		SkelName: skelName,
	})
}

func buildNamedValues[T any](items []*T, skelNameOf func(*T) string, hashOf func(*T) string) []*_NamedValue {
	values := make([]*_NamedValue, 0, len(items))
	for _, item := range items {
		values = append(values, &_NamedValue{
			SkelName: skelNameOf(item),
			Hash:     hashOf(item),
		})
	}
	return values
}

// Compatibility hashes are only used to compare the same schema node across
// skelc builds. They are change detectors, not globally unique identifiers,
// protocol authentication values, or security boundaries, so the lightweight
// 32-bit MurmurHash3 is intentional.
//
// murmur32 implements MurmurHash3_x86_32 with a zero seed.
// The original MurmurHash3 algorithm was written by Austin Appleby and
// released into the public domain:
// https://github.com/aappleby/smhasher/blob/master/src/MurmurHash3.cpp
func murmur32(data []byte) uint32 {
	var hash uint32
	blockCount := len(data) / 4
	for i := 0; i < blockCount; i++ {
		k := binary.LittleEndian.Uint32(data[i*4:])
		k *= murmurC1
		k = bits.RotateLeft32(k, 15)
		k *= murmurC2

		hash ^= k
		hash = bits.RotateLeft32(hash, 13)
		hash = hash*5 + 0xe6546b64
	}

	var tail uint32
	for i, b := range data[blockCount*4:] {
		tail ^= uint32(b) << (i * 8)
	}
	if tail != 0 {
		tail *= murmurC1
		tail = bits.RotateLeft32(tail, 15)
		tail *= murmurC2
		hash ^= tail
	}

	hash ^= uint32(len(data))
	hash ^= hash >> 16
	hash *= 0x85ebca6b
	hash ^= hash >> 13
	hash *= 0xc2b2ae35
	hash ^= hash >> 16
	return hash
}
