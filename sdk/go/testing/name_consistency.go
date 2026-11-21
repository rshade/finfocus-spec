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
	"errors"
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ErrPluginNameMismatch is the error of a failed name check: a plugin reports one
// name, its install name, from both the Name RPC and GetPluginInfo.
var ErrPluginNameMismatch = errors.New("plugin name mismatch")

// nameConsistencyTests returns the Basic checks that a plugin reports one name. They are
// registered only when SuiteConfig.ExpectedName is set.
func nameConsistencyTests() []ConformanceSuiteTest {
	return []ConformanceSuiteTest{
		{
			Name: "RPCCorrectness_NameMatchesPluginInfo",
			Description: "Validates that GetPluginInfo reports the name the Name RPC returns; " +
				"a plugin whose GetPluginInfo is Unimplemented is not compared",
			Category: CategoryRPCCorrectness,
			MinLevel: ConformanceLevelBasic,
			TestFunc: testNameMatchesPluginInfo,
		},
		{
			Name:        "RPCCorrectness_NameMatchesExpected",
			Description: "Validates that the Name RPC returns the install name set with WithExpectedName",
			Category:    CategoryRPCCorrectness,
			MinLevel:    ConformanceLevelBasic,
			TestFunc:    testNameMatchesExpected,
		},
	}
}

// pluginNameResult calls the Name RPC. On failure it returns the failed result.
func pluginNameResult(harness *TestHarness, start time.Time) (string, *TestResult) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTestTimeoutSeconds*time.Second)
	defer cancel()
	resp, err := harness.Client().Name(ctx, &pbc.NameRequest{})
	if err != nil {
		return "", &TestResult{Method: MethodName, Category: CategoryRPCCorrectness, Error: err,
			Duration: time.Since(start), Details: MethodName + " RPC failed"}
	}
	return resp.GetName(), nil
}

func testNameMatchesPluginInfo(harness *TestHarness) TestResult {
	start := time.Now()
	name, failed := pluginNameResult(harness, start)
	if failed != nil {
		return *failed
	}

	harness.discoverCapabilities()
	profile := &harness.capabilities
	if profile.err != nil {
		return TestResult{Method: MethodGetPluginInfo, Category: CategoryRPCCorrectness,
			Error: profile.err, Duration: time.Since(start), Details: detailsCapabilityDiscoveryFailed}
	}
	if !profile.infoProvided {
		return TestResult{Method: MethodGetPluginInfo, Category: CategoryRPCCorrectness, Success: true,
			Duration: time.Since(start), Details: "GetPluginInfo is Unimplemented; name not compared"}
	}
	if profile.infoName != name {
		return TestResult{Method: MethodGetPluginInfo, Category: CategoryRPCCorrectness,
			Error: fmt.Errorf("%w: GetPluginInfo name %q, Name %q",
				ErrPluginNameMismatch, profile.infoName, name),
			Duration: time.Since(start), Details: "GetPluginInfo name differs from Name"}
	}
	return TestResult{Method: MethodGetPluginInfo, Category: CategoryRPCCorrectness, Success: true,
		Duration: time.Since(start), Details: fmt.Sprintf("GetPluginInfo name matches Name: %s", name)}
}

func testNameMatchesExpected(harness *TestHarness) TestResult {
	start := time.Now()
	if harness.expectedName == "" {
		return TestResult{Method: MethodName, Category: CategoryRPCCorrectness, Success: true,
			Duration: time.Since(start), Details: "No expected name set (WithExpectedName); not compared"}
	}

	name, failed := pluginNameResult(harness, start)
	if failed != nil {
		return *failed
	}
	if name != harness.expectedName {
		return TestResult{Method: MethodName, Category: CategoryRPCCorrectness,
			Error: fmt.Errorf("%w: Name %q, expected install name %q",
				ErrPluginNameMismatch, name, harness.expectedName),
			Duration: time.Since(start), Details: "Name differs from the expected install name"}
	}
	return TestResult{Method: MethodName, Category: CategoryRPCCorrectness, Success: true,
		Duration: time.Since(start), Details: fmt.Sprintf("Name matches the expected install name: %s", name)}
}
