// Copyright 2022 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by golang.org/x/tools/cmd/bundle. DO NOT EDIT.
//go:generate bundle -o maphash.go github.com/dolthub/maphash

package lru

import (
	"unsafe"
)

// Hasher hashes values of type K.
// Uses runtime AES-based hashing.
type maphash_Hasher[K comparable] struct {
	hash maphash_hashfn
	seed uintptr
}

// NewHasher creates a new Hasher[K] with a random seed.
func maphash_NewHasher[K comparable]() maphash_Hasher[K] {
	return maphash_Hasher[K]{
		hash: maphash_getRuntimeHasher[K](),
		seed: maphash_newHashSeed(),
	}
}

// Hash hashes |key|.
func (h maphash_Hasher[K]) Hash(key K) uint64 {
	// promise to the compiler that pointer
	// |p| does not escape the stack.
	p := maphash_noescape(unsafe.Pointer(&key))
	return uint64(h.hash(p, h.seed))
}

type maphash_hashfn func(unsafe.Pointer, uintptr) uintptr

func maphash_getRuntimeHasher[K comparable]() (h maphash_hashfn) {
	a := any(make(map[K]struct{}))
	i := (*maphash_mapiface)(unsafe.Pointer(&a))
	h = i.typ.hasher
	return
}

func maphash_newHashSeed() uintptr {
	return uintptr(fastrand64())
}

// noescape hides a pointer from escape analysis. It is the identity function
// but escape analysis doesn't think the output depends on the input.
// noescape is inlined and currently compiles down to zero instructions.
// USE CAREFULLY!
// This was copied from the runtime (via pkg "strings"); see issues 23382 and 7921.
//
//go:nosplit
//go:nocheckptr
func maphash_noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

type maphash_mapiface struct {
	typ *maphash_maptype
	val *maphash_hmap
}

// go/src/runtime/type.go
type maphash_maptype struct {
	typ    maphash__type
	key    *maphash__type
	elem   *maphash__type
	bucket *maphash__type
	// function for hashing keys (ptr to key, seed) -> hash
	hasher     func(unsafe.Pointer, uintptr) uintptr
	keysize    uint8
	elemsize   uint8
	bucketsize uint16
	flags      uint32
}

// go/src/runtime/map.go
type maphash_hmap struct {
	count     int
	flags     uint8
	B         uint8
	noverflow uint16
	// hash seed
	hash0      uint32
	buckets    unsafe.Pointer
	oldbuckets unsafe.Pointer
	nevacuate  uintptr
	// true type is *mapextra
	// but we don't need this data
	extra unsafe.Pointer
}

// go/src/runtime/type.go
type maphash_tflag uint8

type maphash_nameOff int32

type maphash_typeOff int32

// go/src/runtime/type.go
type maphash__type struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      maphash_tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata     *byte
	str        maphash_nameOff
	ptrToThis  maphash_typeOff
}

//go:noescape
//go:linkname fastrand64 runtime.fastrand64
func fastrand64() uint64
