//go:build !windows

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

package testing_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestCostParseMockRejectsSpecialFileWithoutBlocking(t *testing.T) {
	mkfifo, lookupErr := exec.LookPath("mkfifo")
	if lookupErr != nil {
		t.Skip("mkfifo unavailable")
	}
	root := t.TempDir()
	path := filepath.Join(root, "input.fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	require.NoError(t, exec.CommandContext(ctx, mkfifo, path).Run())
	done := make(chan error, 1)
	go func() {
		_, err := pt.NewMockCostParser().
			Parse(ctx, &pbc.ParseRequest{Format: "fixture", Input: &pbc.ParseRequest_RootPath{RootPath: root}})
		done <- err
	}()
	select {
	case err := <-done:
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	case <-ctx.Done():
		// Release an old blocking reader before failing, so RED does not leak it.
		file, err := os.OpenFile(path, os.O_RDWR, 0)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		<-done
		t.Fatal("special-file open blocked beyond the request deadline")
	}
}
