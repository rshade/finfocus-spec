package jsonld

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// IDGenerator defines the strategy for generating @id values in JSON-LD output.
//
// Implementations can use different strategies: user-provided IDs,
// content-based hashes, or deterministic composite keys.
//
// Security Note: The generated @id values are DETERMINISTIC and based on
// predictable composite keys. They are NOT cryptographically secure and
// should NOT be used for authentication, authorization, or as security tokens.
type IDGenerator interface {
	// Generate creates a unique identifier for a FocusCostRecord.
	Generate(record *pbc.FocusCostRecord) string

	// GenerateCommitment creates a unique identifier for a ContractCommitment.
	GenerateCommitment(record *pbc.ContractCommitment) string
}

// ConfigurableIDGenerator extends IDGenerator with configuration methods.
// The default SHA256IDGenerator implements this interface.
// Custom IDGenerator implementations that don't support configuration
// should not implement this interface.
type ConfigurableIDGenerator interface {
	IDGenerator

	// WithUserIDField configures the generator to use a specific field as the ID source.
	WithUserIDField(field string) IDGenerator

	// WithIDPrefix sets a custom prefix for generated IDs.
	WithIDPrefix(prefix string) IDGenerator
}

// SHA256IDGenerator implements IDGenerator using SHA256-based composite keys.
type SHA256IDGenerator struct {
	prefix      string
	userIDField string
}

// NewIDGenerator creates a default IDGenerator.
//
// Default configuration:
//   - Prefix: "urn:focus:cost:"
//   - No user-provided ID field (uses composite key hash)
func NewIDGenerator() IDGenerator {
	return &SHA256IDGenerator{
		prefix:      "urn:focus:cost:",
		userIDField: "",
	}
}

// WithUserIDField configures the IDGenerator to use a specific field from the record.
//
// If the field has a non-empty value, it will be used as the @id prefix.
// Otherwise, falls back to composite key hash.
//
// Thread Safety: This method returns a new IDGenerator instance with the
// updated configuration, leaving the original unchanged (copy-on-write).
func (g *SHA256IDGenerator) WithUserIDField(field string) IDGenerator {
	// Copy-on-write: create a new instance to avoid race conditions
	copied := *g
	copied.userIDField = field
	return &copied
}

// WithIDPrefix sets a custom prefix for generated IDs.
//
// Default prefix is "urn:focus:cost:" for cost records and
// "urn:focus:commitment:" for commitments.
//
// Thread Safety: This method returns a new IDGenerator instance with the
// updated configuration, leaving the original unchanged (copy-on-write).
func (g *SHA256IDGenerator) WithIDPrefix(prefix string) IDGenerator {
	// Copy-on-write: create a new instance to avoid race conditions
	copied := *g
	copied.prefix = prefix
	return &copied
}

// Generate creates a unique identifier for a FocusCostRecord.
//
// Algorithm:
//  1. If record is nil → returns prefix + "nil-record"
//  2. If userIDField is set and record has non-empty value → use that value
//  3. Otherwise, compute SHA256(billing_account_id + "|" + charge_period_start + "|" + resource_id)
//  4. Return prefix + hex(hash) (full 32 bytes = 64 hex chars)
func (g *SHA256IDGenerator) Generate(record *pbc.FocusCostRecord) string {
	// Handle nil record
	if record == nil {
		return g.prefix + "nil-record"
	}

	// Check for user-provided ID if configured
	if g.userIDField != "" {
		if userID := getUserProvidedID(record, g.userIDField); userID != "" {
			return fmt.Sprintf("%s%s", g.prefix, userID)
		}
	}

	// Defensive nil check for timestamp - use zero time if nil
	periodStart := record.GetChargePeriodStart()
	periodDate := time.Time{} // zero time
	if periodStart != nil {
		periodDate = periodStart.AsTime()
	}

	// Compute composite key hash
	compositeKey := fmt.Sprintf("%s|%s|%s",
		record.GetBillingAccountId(),
		periodDate.Format(time.RFC3339),
		record.GetResourceId(),
	)

	hash := sha256.Sum256([]byte(compositeKey))
	hashHex := hex.EncodeToString(hash[:]) // Full 64 hex chars (32 bytes = 256 bits)

	return fmt.Sprintf("%s%s", g.prefix, hashHex)
}

// GenerateCommitment creates a unique identifier for a ContractCommitment.
//
// Uses SHA256 of contract_commitment_id as the unique key.
// Returns prefix + "nil-commitment" if record is nil.
// Returns prefix + "empty-commitment-id" if ContractCommitmentId is empty.
func (g *SHA256IDGenerator) GenerateCommitment(record *pbc.ContractCommitment) string {
	commitmentPrefix := "urn:focus:commitment:"

	// Handle nil record
	if record == nil {
		return commitmentPrefix + "nil-commitment"
	}

	// Check for user-provided ID if configured
	if g.userIDField != "" {
		if userID := getCommitmentUserID(record, g.userIDField); userID != "" {
			return fmt.Sprintf("%s%s", commitmentPrefix, userID)
		}
	}

	// Handle empty commitment ID to avoid collision (all empty strings hash to same value)
	commitmentID := record.GetContractCommitmentId()
	if commitmentID == "" {
		return commitmentPrefix + "empty-commitment-id"
	}

	// Compute hash of commitment ID
	hash := sha256.Sum256([]byte(commitmentID))
	hashHex := hex.EncodeToString(hash[:]) // Full 64 hex chars (32 bytes = 256 bits)

	return fmt.Sprintf("%s%s", commitmentPrefix, hashHex)
}

func billingPeriodDocumentID(record *pbc.BillingPeriod) string {
	const prefix = "urn:focus:billing-period:"
	if record == nil {
		return prefix + "nil"
	}
	start := ""
	if ts := record.GetBillingPeriodStart(); ts != nil {
		start = ts.AsTime().UTC().Format(time.RFC3339)
	}
	sum := sha256.Sum256([]byte(record.GetInvoiceIssuerName() + "|" + start))
	return prefix + hex.EncodeToString(sum[:])
}

func invoiceDetailDocumentID(record *pbc.InvoiceDetail) string {
	const prefix = "urn:focus:invoice-detail:"
	if record == nil {
		return prefix + "nil"
	}
	key := record.GetInvoiceId() + "|" + record.GetInvoiceDetailId()
	if key == "|" {
		return prefix + "empty"
	}
	sum := sha256.Sum256([]byte(key))
	return prefix + hex.EncodeToString(sum[:])
}

// getUserProvidedID extracts a user-provided ID from a FocusCostRecord.
//
// This is a simplified version - in production, use reflection to access
// arbitrary fields by name.
func getUserProvidedID(record *pbc.FocusCostRecord, field string) string {
	// Common user-provided ID fields
	switch field {
	case "invoice_id":
		return record.GetInvoiceId()
	case "resource_id":
		return record.GetResourceId()
	default:
		return ""
	}
}

// getCommitmentUserID extracts a user-provided ID from a ContractCommitment.
func getCommitmentUserID(record *pbc.ContractCommitment, field string) string {
	switch field {
	case "contract_commitment_id":
		return record.GetContractCommitmentId()
	case "contract_id":
		return record.GetContractId()
	default:
		return ""
	}
}
