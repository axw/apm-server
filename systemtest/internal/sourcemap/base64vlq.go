// Copyright (c) 2016 The github.com/go-sourcemap/sourcemap Contributors.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package sourcemap

import "io"

const encodeStd = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

const (
	vlqBaseShift       = 5
	vlqBase            = 1 << vlqBaseShift
	vlqBaseMask        = vlqBase - 1
	vlqSignBit         = 1
	vlqContinuationBit = vlqBase
)

var decodeMap [256]byte

func init() {
	for i := 0; i < len(encodeStd); i++ {
		decodeMap[encodeStd[i]] = byte(i)
	}
}

func toVLQSigned(n int32) int32 {
	if n < 0 {
		return -n<<1 + 1
	}
	return n << 1
}

func fromVLQSigned(n int32) int32 {
	isNeg := n&vlqSignBit != 0
	n >>= 1
	if isNeg {
		return -n
	}
	return n
}

type vlqDecoder struct {
	r io.ByteReader
}

func newVLQDecoder(r io.ByteReader) vlqDecoder {
	return vlqDecoder{r: r}
}

func (dec vlqDecoder) Decode() (n int32, err error) {
	shift := uint(0)
	for continuation := true; continuation; {
		c, err := dec.r.ReadByte()
		if err != nil {
			return 0, err
		}

		c = decodeMap[c]
		continuation = c&vlqContinuationBit != 0
		n += int32(c&vlqBaseMask) << shift
		shift += vlqBaseShift
	}
	return fromVLQSigned(n), nil
}
