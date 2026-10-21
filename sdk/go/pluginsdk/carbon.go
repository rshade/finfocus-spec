// Package pluginsdk provides historical emissions validation and optional dispatch.
package pluginsdk

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// DefaultCarbonPageSize is the default historical emissions page size.
	DefaultCarbonPageSize int32 = 50
	// MaxCarbonPageSize bounds historical emissions response pages.
	MaxCarbonPageSize int32 = 1000
	// MaxCarbonPageTokenBytes bounds opaque continuation state.
	MaxCarbonPageTokenBytes = 16 * 1024
	// CarbonEmissionsUnit is grams of carbon dioxide equivalent.
	CarbonEmissionsUnit = "gCO2e"
)

// HistoricalCarbonProvider optionally supplies provider-reported monthly emissions.
// Implementations must preserve source provenance and never synthesize missing data.
// Advertise HISTORICAL_CARBON explicitly through PluginInfo capabilities. It is
// never inferred: existing plugins embedding the generated unimplemented server
// also satisfy this interface. Advertising it does not advertise projected CARBON.
type HistoricalCarbonProvider interface {
	GetCarbonEmissions(context.Context, *pbc.GetCarbonEmissionsRequest) (*pbc.GetCarbonEmissionsResponse, error)
}

// ApplyGetCarbonEmissionsDefaults resolves unspecified values in place.
// Call validation first: unknown values are never silently normalized.
func ApplyGetCarbonEmissionsDefaults(req *pbc.GetCarbonEmissionsRequest) {
	if req == nil {
		return
	}
	if req.GetMethod() == pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_UNSPECIFIED {
		req.Method = pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_MARKET_BASED
	}
	if req.GetScope() == pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_UNSPECIFIED {
		req.Scope = pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_TOTAL
	}
	if req.GetPageSize() == 0 {
		req.PageSize = DefaultCarbonPageSize
	}
}

func validateCarbonPeriod(start, end *timestamppb.Timestamp, monthly bool) error {
	for _, timestamp := range []*timestamppb.Timestamp{start, end} {
		if timestamp == nil || timestamp.CheckValid() != nil {
			return errors.New("carbon period requires valid timestamps")
		}
		at := timestamp.AsTime()
		if at.Day() != 1 || at.Hour() != 0 || at.Minute() != 0 || at.Second() != 0 || at.Nanosecond() != 0 {
			return errors.New("carbon period requires UTC month boundaries")
		}
	}
	if !end.AsTime().After(start.AsTime()) {
		return errors.New("carbon period end must follow start")
	}
	if monthly && !end.AsTime().Equal(start.AsTime().AddDate(0, 1, 0)) {
		return errors.New("carbon record must cover exactly one month")
	}
	return nil
}

// ValidateGetCarbonEmissionsRequest checks monthly boundaries, enums and page bounds.
// Unspecified accounting method and scope are valid request defaults.
func ValidateGetCarbonEmissionsRequest(req *pbc.GetCarbonEmissionsRequest) error {
	if req == nil {
		return errors.New("carbon request is required")
	}
	if err := validateCarbonPeriod(req.GetStart(), req.GetEnd(), false); err != nil {
		return err
	}
	if req.GetMethod() < 0 || req.GetMethod() > pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_LOCATION_BASED {
		return errors.New("unknown carbon accounting method")
	}
	if req.GetScope() < 0 || req.GetScope() > pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_SCOPE_3 {
		return errors.New("unknown carbon emissions scope")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > MaxCarbonPageSize {
		return errors.New("carbon page size must be between zero and 1000")
	}
	if len(req.GetPageToken()) > MaxCarbonPageTokenBytes {
		return errors.New("carbon page token exceeds 16384 bytes")
	}
	return nil
}

// ValidateCarbonEmissionRecord checks a monthly aggregate and its provenance.
// Explicit zero is valid. Missing values must never be invented by a provider.
func ValidateCarbonEmissionRecord(record *pbc.CarbonEmissionRecord) error {
	if record == nil {
		return errors.New("carbon record is required")
	}
	for _, value := range []string{record.GetId(), record.GetProvider(), record.GetSource(), record.GetAccountId(), record.GetRegion(), record.GetService(), record.GetModelVersion()} {
		if strings.TrimSpace(value) == "" {
			return errors.New("carbon record identity, grouping and provenance are required")
		}
	}
	if err := validateCarbonPeriod(record.GetStart(), record.GetEnd(), true); err != nil {
		return err
	}
	if record.GetMethod() < pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_MARKET_BASED ||
		record.GetMethod() > pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_LOCATION_BASED {
		return errors.New("carbon record requires resolved accounting method")
	}
	if record.GetScope() < pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_TOTAL ||
		record.GetScope() > pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_SCOPE_3 {
		return errors.New("carbon record requires resolved emissions scope")
	}
	if record.GetUnit() != CarbonEmissionsUnit || math.IsNaN(record.GetValue()) || math.IsInf(record.GetValue(), 0) ||
		record.GetValue() < 0 {
		return errors.New("carbon record requires finite nonnegative gCO2e")
	}
	if record.GetRetrievedAt() == nil || record.GetRetrievedAt().CheckValid() != nil {
		return errors.New("carbon record requires valid retrieval timestamp")
	}
	return nil
}

// ValidateGetCarbonEmissionsResponse validates every record, preserving empty pages.
func ValidateGetCarbonEmissionsResponse(resp *pbc.GetCarbonEmissionsResponse) error {
	if resp == nil {
		return errors.New("carbon response is required")
	}
	for i, record := range resp.GetRecords() {
		if err := ValidateCarbonEmissionRecord(record); err != nil {
			return fmt.Errorf("carbon record %d: %w", i, err)
		}
	}
	return nil
}

// GetCarbonEmissions dispatches the optional historical emissions RPC.
func (s *Server) GetCarbonEmissions(
	ctx context.Context,
	req *pbc.GetCarbonEmissionsRequest,
) (*pbc.GetCarbonEmissionsResponse, error) {
	provider, ok := s.plugin.(HistoricalCarbonProvider)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "historical carbon reporting is not implemented")
	}
	if err := ValidateGetCarbonEmissionsRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	normalized := proto.CloneOf(req)
	ApplyGetCarbonEmissionsDefaults(normalized)
	resp, err := provider.GetCarbonEmissions(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if err = ValidateGetCarbonEmissionsResponse(resp); err != nil {
		return nil, status.Error(codes.Internal, "invalid historical carbon response")
	}
	return resp, nil
}
