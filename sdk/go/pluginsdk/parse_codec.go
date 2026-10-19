// Copyright 2026 The FinFocus Authors
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

package pluginsdk

import (
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	parseFilesFieldNumber    protowire.Number = 2
	parseRootPathFieldNumber protowire.Number = 3
)

// conflictingParseInput scans only top-level tags. The stock decoder remains
// authoritative for malformed data and repeated occurrences of the same arm.
func conflictingParseInput(data []byte) bool {
	var files, root bool
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 {
			return false
		}
		data = data[n:]
		size := protowire.ConsumeFieldValue(number, kind, data)
		if size < 0 {
			return false
		}
		if kind == protowire.BytesType {
			files = files || number == parseFilesFieldNumber
			root = root || number == parseRootPathFieldNumber
		}
		data = data[size:]
	}
	return files && root
}

type parseGRPCCodec struct{ encoding.CodecV2 }

// Unmarshal delegates stock decoding and preserves input conflict evidence.
func (c *parseGRPCCodec) Unmarshal(data mem.BufferSlice, message any) error {
	if err := c.CodecV2.Unmarshal(data, message); err != nil {
		return err
	}
	if req, isParse := message.(*pbc.ParseRequest); isParse && conflictingParseInput(data.Materialize()) {
		req.Input = nil
	}
	return nil
}

type parseConnectCodec struct{}

// Name selects the protobuf codec for this parsing handler.
func (*parseConnectCodec) Name() string { return "proto" }

// Marshal uses ordinary protobuf response encoding.
func (*parseConnectCodec) Marshal(message any) ([]byte, error) {
	m, ok := message.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("cannot marshal %T as protobuf", message)
	}
	return proto.Marshal(m)
}

// Unmarshal rejects conflicting parsing inputs through typed validation.
func (*parseConnectCodec) Unmarshal(data []byte, message any) error {
	m, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("cannot unmarshal %T as protobuf", message)
	}
	if err := proto.Unmarshal(data, m); err != nil {
		return err
	}
	if req, isParse := message.(*pbc.ParseRequest); isParse && conflictingParseInput(data) {
		req.Input = nil
	}
	return nil
}
