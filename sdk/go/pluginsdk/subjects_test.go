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

package pluginsdk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// TestSubjectVocabularyMatchesTesting guards the copy of the subject keys that
// sdk/go/testing keeps because it cannot import pluginsdk.
func TestSubjectVocabularyMatchesTesting(t *testing.T) {
	keys := []string{
		pluginsdk.SubjectCluster,
		pluginsdk.SubjectNamespace,
		pluginsdk.SubjectControllerKind,
		pluginsdk.SubjectController,
		pluginsdk.SubjectPod,
		pluginsdk.SubjectNode,
		pluginsdk.SubjectKind,
	}
	assert.ElementsMatch(t, keys, plugintesting.KnownSubjectKeys())
	assert.Equal(t, "label.", pluginsdk.SubjectLabelPrefix)

	first := plugintesting.KnownSubjectKeys()
	first[0] = "mutated"
	assert.NotContains(t, plugintesting.KnownSubjectKeys(), "mutated", "KnownSubjectKeys must return a copy")
}
