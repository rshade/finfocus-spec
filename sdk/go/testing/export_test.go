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

// RunAllocatorScenariosForTest exposes the allocator conformance scenario
// runner to external tests, so broken allocators can be asserted to fail
// specific scenarios without a fake *testing.T.
//
//nolint:gochecknoglobals // Test-only export of an unexported function.
var RunAllocatorScenariosForTest = runAllocatorScenarios

// RunContractCommitmentScenariosForTest exposes the contract commitment
// conformance scenario runner, so broken sources can be asserted to fail
// specific scenarios without a fake *testing.T.
//
//nolint:gochecknoglobals // Test-only export of an unexported function.
var RunContractCommitmentScenariosForTest = runContractCommitmentScenarios

// CopiedVocabularyForTest returns the row kinds and metric names this package
// copies from pluginsdk (which it cannot import), in a fixed order, so an
// external drift test can compare them with the pluginsdk constants.
func CopiedVocabularyForTest() []string {
	return []string{
		kindWorkload, kindIdle, kindCluster,
		metricCPURequest, metricMemRequest, metricCPUAllocatable, metricMemAllocatable,
	}
}

// RunScorerScenariosForTest exposes the scorer conformance scenario runner,
// so broken scorers can be asserted to fail specific scenarios without a fake
// *testing.T.
//
//nolint:gochecknoglobals // Test-only export of an unexported function.
var RunScorerScenariosForTest = runScorerScenarios

// RunScorerAdvertisedLimitsForTest exposes the advertised_limits scenario.
//
//nolint:gochecknoglobals // Test-only export of an unexported function.
var RunScorerAdvertisedLimitsForTest = scorerCheckAdvertisedLimits
