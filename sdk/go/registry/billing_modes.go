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

package registry

// allManifestBillingModes is the billing_modes enum of the plugin manifest schema's
// supported_resources entries, in schema order. It is distinct from the pricing package's
// BillingMode enum, which covers PricingSpec documents.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allManifestBillingModes = []string{
	"per_hour", "per_minute", "per_second",
	"per_gb_month", "per_gb_hour", "per_gb_day",
	"per_request", "per_operation", "per_transaction",
	"per_execution", "per_invocation",
	"flat", "per_day", "per_month", "per_year",
	"per_cpu_hour", "per_cpu_month", "per_vcpu_hour",
	"per_memory_gb_hour", "per_memory_gb_month",
	"per_iops", "per_provisioned_iops",
	"per_rcu", "per_wcu", "per_dtu", "per_ru",
	"on_demand", "reserved", "spot", "preemptible",
	"savings_plan", "committed_use", "hybrid_benefit",
	"per_data_transfer_gb", "per_bandwidth_gb",
	"per_api_call", "per_lookup", "per_query",
}

// AllManifestBillingModes returns the billing modes a plugin manifest may list under
// specification.supported_resources, in schema order.
func AllManifestBillingModes() []string {
	return allManifestBillingModes
}

// IsValidManifestBillingMode reports whether mode is one of AllManifestBillingModes.
func IsValidManifestBillingMode(mode string) bool {
	for _, valid := range allManifestBillingModes {
		if mode == valid {
			return true
		}
	}
	return false
}
