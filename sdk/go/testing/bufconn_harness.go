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

package testing

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// bufconnHarness serves one gRPC service over an in-memory bufconn and holds a
// client of type C for it. Every exported harness embeds it and shares Start
// and Stop; each declares its own Client so godoc shows the concrete client
// type rather than C.
type bufconnHarness[C any] struct {
	server    *grpc.Server
	listener  *bufconn.Listener
	conn      *grpc.ClientConn
	client    C
	newClient func(grpc.ClientConnInterface) C
}

// newBufconnHarness starts a server with the service that register adds.
// newClient is the generated client constructor for that service.
func newBufconnHarness[C any](
	register func(*grpc.Server), newClient func(grpc.ClientConnInterface) C,
) bufconnHarness[C] {
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	register(server)

	go func() {
		_ = server.Serve(listener)
	}()

	return bufconnHarness[C]{server: server, listener: listener, newClient: newClient}
}

// dial opens a client connection to the in-memory server.
//
//nolint:staticcheck // grpc.NewClient doesn't work with bufconn
func (h *bufconnHarness[C]) dial() (*grpc.ClientConn, error) {
	return grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return h.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// Start initializes the client connection to the in-memory server.
func (h *bufconnHarness[C]) Start(t testing.TB) {
	conn, err := h.dial()
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	h.conn = conn
	h.client = h.newClient(conn)
}

// Stop closes the client connection and stops the server. It is safe to call
// more than once, and before Start.
func (h *bufconnHarness[C]) Stop() {
	if h.conn != nil {
		_ = h.conn.Close()
	}
	if h.server != nil {
		h.server.Stop()
	}
}
