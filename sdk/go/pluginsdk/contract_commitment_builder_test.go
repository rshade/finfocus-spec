package pluginsdk_test

import (
	"strings"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// =============================================================================
// ContractCommitmentBuilder Tests (Phase 5 - User Story 3)
// =============================================================================

// TestContractCommitmentBuilder_Build_HappyPath tests a complete valid contract commitment.
func TestContractCommitmentBuilder_Build_HappyPath(t *testing.T) {
	now := time.Now()
	commitmentStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	commitmentEnd := commitmentStart.AddDate(1, 0, 0)
	contractStart := commitmentStart.AddDate(-1, 0, 0)
	contractEnd := commitmentEnd.AddDate(2, 0, 0)

	record, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commitment-ri-123456", "contract-enterprise-2024").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
		WithType("3-Year Reserved Instance").
		WithCommitmentPeriod(commitmentStart, commitmentEnd).
		WithContractPeriod(contractStart, contractEnd).
		WithFinancials(120000.00, 0, "", "USD").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify identity fields
	if record.GetContractCommitmentId() != "commitment-ri-123456" {
		t.Errorf("Expected ContractCommitmentId %q, got %q", "commitment-ri-123456", record.GetContractCommitmentId())
	}
	if record.GetContractId() != "contract-enterprise-2024" {
		t.Errorf("Expected ContractId %q, got %q", "contract-enterprise-2024", record.GetContractId())
	}

	// Verify classification fields
	if record.GetContractCommitmentCategory() != pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND {
		t.Errorf("Expected SPEND category, got %v", record.GetContractCommitmentCategory())
	}
	if record.GetContractCommitmentType() != "3-Year Reserved Instance" {
		t.Errorf("Expected CommitmentType %q, got %q", "3-Year Reserved Instance", record.GetContractCommitmentType())
	}

	// Verify financial fields
	if record.GetContractCommitmentCost() != 120000.00 {
		t.Errorf("Expected Cost 120000.00, got %f", record.GetContractCommitmentCost())
	}
	if record.GetBillingCurrency() != "USD" {
		t.Errorf("Expected BillingCurrency %q, got %q", "USD", record.GetBillingCurrency())
	}
}

// TestContractCommitmentBuilder_WithIdentity tests the WithIdentity builder method.
func TestContractCommitmentBuilder_WithIdentity(t *testing.T) {
	tests := []struct {
		name         string
		commitmentID string
		contractID   string
	}{
		{"ri commitment", "ri-123", "contract-456"},
		{"savings plan", "sp-abc", "contract-xyz"},
		{"cud commitment", "cud-def", "contract-ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidCommitmentBuilder()
			builder.WithIdentity(tt.commitmentID, tt.contractID)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractCommitmentId() != tt.commitmentID {
				t.Errorf("Expected CommitmentId %q, got %q", tt.commitmentID, record.GetContractCommitmentId())
			}
			if record.GetContractId() != tt.contractID {
				t.Errorf("Expected ContractId %q, got %q", tt.contractID, record.GetContractId())
			}
		})
	}
}

// TestContractCommitmentBuilder_WithCategory tests the WithCategory builder method.
func TestContractCommitmentBuilder_WithCategory(t *testing.T) {
	tests := []struct {
		name        string
		category    pbc.FocusContractCommitmentCategory
		expectError bool
	}{
		{"spend", pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND, false},
		{"usage", pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE, false},
		{
			"unspecified rejects",
			pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_UNSPECIFIED,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: createValidCommitmentBuilder sets SPEND category, so we start with minimal builder
			builder := pluginsdk.NewContractCommitmentBuilder().
				WithIdentity("commitment-123", "contract-456").
				WithCategory(tt.category).
				WithFinancials(100, 0, "", "USD")
			record, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for UNSPECIFIED category, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Build failed: %v", err)
				}
				if record.GetContractCommitmentCategory() != tt.category {
					t.Errorf("Expected Category %v, got %v", tt.category, record.GetContractCommitmentCategory())
				}
			}
		})
	}
}

// TestContractCommitmentBuilder_WithType tests the WithType builder method.
func TestContractCommitmentBuilder_WithType(t *testing.T) {
	tests := []struct {
		name           string
		commitmentType string
	}{
		{"reserved instance", "Reserved Instance"},
		{"savings plan", "Savings Plan"},
		{"committed use discount", "Committed Use Discount"},
		{"enterprise agreement", "Enterprise Agreement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidCommitmentBuilder()
			builder.WithType(tt.commitmentType)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractCommitmentType() != tt.commitmentType {
				t.Errorf("Expected Type %q, got %q", tt.commitmentType, record.GetContractCommitmentType())
			}
		})
	}
}

// TestContractCommitmentBuilder_WithFinancials tests the WithFinancials builder method.
func TestContractCommitmentBuilder_WithFinancials(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		quantity float64
		unit     string
		currency string
	}{
		{"spend commitment", 10000.00, 0, "", "USD"},
		{"usage commitment", 0, 1000, "Hours", "EUR"},
		{"combined commitment", 5000.00, 500, "vCPU-Hours", "GBP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := createValidCommitmentBuilder()
			builder.WithFinancials(tt.cost, tt.quantity, tt.unit, tt.currency)
			record, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}
			if record.GetContractCommitmentCost() != tt.cost {
				t.Errorf("Expected Cost %f, got %f", tt.cost, record.GetContractCommitmentCost())
			}
			if record.GetContractCommitmentQuantity() != tt.quantity {
				t.Errorf("Expected Quantity %f, got %f", tt.quantity, record.GetContractCommitmentQuantity())
			}
			if record.GetContractCommitmentUnit() != tt.unit {
				t.Errorf("Expected Unit %q, got %q", tt.unit, record.GetContractCommitmentUnit())
			}
			if record.GetBillingCurrency() != tt.currency {
				t.Errorf("Expected Currency %q, got %q", tt.currency, record.GetBillingCurrency())
			}
		})
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

// TestContractCommitmentBuilder_Validation_RequiredFields tests required field validation.
func TestContractCommitmentBuilder_Validation_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		builder     func() *pluginsdk.ContractCommitmentBuilder
		expectError bool
		errContains string
	}{
		{
			name: "missing commitment id",
			builder: func() *pluginsdk.ContractCommitmentBuilder {
				return pluginsdk.NewContractCommitmentBuilder().
					WithIdentity("", "contract-123").
					WithFinancials(100, 0, "", "USD")
			},
			expectError: true,
			errContains: "contract_commitment_id",
		},
		{
			name: "missing contract id",
			builder: func() *pluginsdk.ContractCommitmentBuilder {
				return pluginsdk.NewContractCommitmentBuilder().
					WithIdentity("commitment-123", "").
					WithFinancials(100, 0, "", "USD")
			},
			expectError: true,
			errContains: "contract_id",
		},
		{
			name: "missing billing currency",
			builder: func() *pluginsdk.ContractCommitmentBuilder {
				return pluginsdk.NewContractCommitmentBuilder().
					WithIdentity("commitment-123", "contract-456").
					WithFinancials(100, 0, "", "")
			},
			expectError: true,
			errContains: "billing_currency",
		},
		{
			name: "valid minimal",
			builder: func() *pluginsdk.ContractCommitmentBuilder {
				return pluginsdk.NewContractCommitmentBuilder().
					WithIdentity("commitment-123", "contract-456").
					WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
					WithFinancials(100, 0, "", "USD")
			},
			expectError: false,
			errContains: "",
		},
		{
			name: "missing category (UNSPECIFIED)",
			builder: func() *pluginsdk.ContractCommitmentBuilder {
				return pluginsdk.NewContractCommitmentBuilder().
					WithIdentity("commitment-123", "contract-456").
					WithFinancials(100, 0, "", "USD")
			},
			expectError: true,
			errContains: "contract_commitment_category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.builder().Build()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestContractCommitmentBuilder_Validation_PeriodConsistency tests period validation.
func TestContractCommitmentBuilder_Validation_PeriodConsistency(t *testing.T) {
	now := time.Now()
	validStart := now
	validEnd := now.AddDate(1, 0, 0)
	invalidEnd := now.AddDate(-1, 0, 0) // End before start

	tests := []struct {
		name        string
		start       time.Time
		end         time.Time
		expectError bool
	}{
		{"valid period", validStart, validEnd, false},
		{"same start and end", validStart, validStart, false}, // Edge case: allowed
		{"end before start", validStart, invalidEnd, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := pluginsdk.NewContractCommitmentBuilder().
				WithIdentity("commitment-123", "contract-456").
				WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
				WithCommitmentPeriod(tt.start, tt.end).
				WithFinancials(100, 0, "", "USD")

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid period but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestContractCommitmentBuilder_Validation_NonNegativeValues tests non-negative validation.
func TestContractCommitmentBuilder_Validation_NonNegativeValues(t *testing.T) {
	tests := []struct {
		name        string
		cost        float64
		quantity    float64
		expectError bool
	}{
		{"positive cost", 100.00, 0, false},
		{"positive quantity", 0, 100, false},
		{"zero cost and quantity", 0, 0, false},
		{"negative cost", -100.00, 0, true},
		{"negative quantity", 0, -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := pluginsdk.NewContractCommitmentBuilder().
				WithIdentity("commitment-123", "contract-456").
				WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
				WithFinancials(tt.cost, tt.quantity, "Units", "USD")

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for negative values but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestContractCommitmentBuilder_Validation_Currency tests currency validation.
func TestContractCommitmentBuilder_Validation_Currency(t *testing.T) {
	tests := []struct {
		name        string
		currency    string
		expectError bool
	}{
		{"valid USD", "USD", false},
		{"valid EUR", "EUR", false},
		{"valid JPY", "JPY", false},
		{"valid XXX no currency", "XXX", false}, // XXX is valid ISO 4217 for "no currency"
		{"invalid currency", "ZZZ", true},       // ZZZ is not a valid ISO 4217 code
		{"lowercase", "usd", true},              // Must be uppercase
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := pluginsdk.NewContractCommitmentBuilder().
				WithIdentity("commitment-123", "contract-456").
				WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
				WithFinancials(100, 0, "", tt.currency)

			_, err := builder.Build()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for currency %q but got nil", tt.currency)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for currency %q: %v", tt.currency, err)
				}
			}
		})
	}
}

// TestContractCommitmentBuilder_ChainMethods tests method chaining.
func TestContractCommitmentBuilder_ChainMethods(t *testing.T) {
	now := time.Now()
	commitmentStart := now
	commitmentEnd := now.AddDate(1, 0, 0)
	contractStart := now.AddDate(-1, 0, 0)
	contractEnd := now.AddDate(2, 0, 0)

	record, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commitment-123", "contract-456").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE).
		WithType("Compute Engine CUD").
		WithCommitmentPeriod(commitmentStart, commitmentEnd).
		WithContractPeriod(contractStart, contractEnd).
		WithFinancials(0, 1000, "vCPU-Hours", "USD").
		Build()

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify all fields
	if record.GetContractCommitmentId() != "commitment-123" {
		t.Errorf("Wrong CommitmentId")
	}
	if record.GetContractId() != "contract-456" {
		t.Errorf("Wrong ContractId")
	}
	if record.GetContractCommitmentCategory() != pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE {
		t.Errorf("Wrong Category")
	}
	if record.GetContractCommitmentType() != "Compute Engine CUD" {
		t.Errorf("Wrong Type")
	}
	if record.GetContractCommitmentQuantity() != 1000 {
		t.Errorf("Wrong Quantity")
	}
	if record.GetContractCommitmentUnit() != "vCPU-Hours" {
		t.Errorf("Wrong Unit")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// createValidCommitmentBuilder creates a builder with minimal valid data.
func createValidCommitmentBuilder() *pluginsdk.ContractCommitmentBuilder {
	return pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commitment-123", "contract-456").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
		WithFinancials(100, 0, "", "USD")
}

// =============================================================================
// Benchmarks
// =============================================================================

// BenchmarkContractCommitmentBuilder_Build measures build performance.
func BenchmarkContractCommitmentBuilder_Build(b *testing.B) {
	now := time.Now()
	start := now
	end := now.AddDate(1, 0, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commitment-123", "contract-456").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
			WithType("Reserved Instance").
			WithCommitmentPeriod(start, end).
			WithFinancials(10000, 0, "", "USD").
			Build()
	}
}

// BenchmarkContractCommitmentBuilder_WithIdentity measures identity method performance.
func BenchmarkContractCommitmentBuilder_WithIdentity(b *testing.B) {
	builder := pluginsdk.NewContractCommitmentBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithIdentity("commitment-123", "contract-456")
	}
}

// BenchmarkContractCommitmentBuilder_WithFinancials measures financials method performance.
func BenchmarkContractCommitmentBuilder_WithFinancials(b *testing.B) {
	builder := pluginsdk.NewContractCommitmentBuilder()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		builder.WithFinancials(10000, 100, "Hours", "USD")
	}
}

// TestContractCommitmentBuilder_Build_DoesNotRequireFocus14 guards that a
// builder chain written before FOCUS 1.4 still builds.
func TestContractCommitmentBuilder_Build_DoesNotRequireFocus14(t *testing.T) {
	record, err := pluginsdk.NewContractCommitmentBuilder().
		WithIdentity("commitment-123", "contract-456").
		WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
		WithFinancials(100, 0, "", "USD").
		Build()
	if err != nil {
		t.Fatalf("Build() without FOCUS 1.4 columns error = %v, want nil", err)
	}
	if record == nil {
		t.Fatal("Build() returned nil record")
	}
}

func TestContractCommitmentBuilder_BuildFocus14(t *testing.T) {
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newBuilder := func() *pluginsdk.ContractCommitmentBuilder {
		return pluginsdk.NewContractCommitmentBuilder().
			WithIdentity("commitment-123", "contract-456").
			WithCategory(pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND).
			WithFinancials(100, 0, "", "USD")
	}

	if _, err := newBuilder().BuildFocus14(); err == nil {
		t.Fatal("BuildFocus14() without FOCUS 1.4 columns error = nil, want error")
	}
	if _, err := newBuilder().WithBaselineTerms(at).BuildFocus14(); err != nil {
		t.Fatalf("BuildFocus14() with baseline terms error = %v, want nil", err)
	}
	if _, err := pluginsdk.NewContractCommitmentBuilder().BuildFocus14(); err == nil {
		t.Fatal("BuildFocus14() without required base fields error = nil, want error")
	}
}
