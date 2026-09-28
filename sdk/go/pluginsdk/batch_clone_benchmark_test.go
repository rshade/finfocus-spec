//nolint:testpackage // Benchmarks unexported descriptorClone and result constructors.
package pluginsdk

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// benchmarkCloneSink prevents the compiler from eliminating benchmark results.
//
//nolint:gochecknoglobals // Standard benchmark sink.
var benchmarkCloneSink *pbc.ResourceCostResult

// benchmarkCloneDescriptors builds n realistic resource descriptors covering
// every ResourceDescriptor field (scalars, tags map, optional pointers).
func benchmarkCloneDescriptors(n int) []*pbc.ResourceDescriptor {
	utilization := 0.75
	growthRate := 0.05
	descriptors := make([]*pbc.ResourceDescriptor, n)
	for i := range descriptors {
		descriptors[i] = &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "m5.large",
			Region:       "us-east-1",
			Tags: map[string]string{
				"app": "web",
				"env": "production",
			},
			UtilizationPercentage: &utilization,
			Id:                    fmt.Sprintf("urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web-%d", i),
			Arn:                   fmt.Sprintf("arn:aws:ec2:us-east-1:123456789012:instance/i-%017d", i),
			GrowthType:            pbc.GrowthType_GROWTH_TYPE_LINEAR,
			GrowthRate:            &growthRate,
		}
	}
	return descriptors
}

// BenchmarkDescriptorClone measures the per-batch cost of deep-copying
// ResourceDescriptors at representative batch sizes, comparing the old
// reflection-based proto.Clone path against descriptorClone as implemented.
// The N=1000 case is the max batch size from issue #402.
func BenchmarkDescriptorClone(b *testing.B) {
	for _, n := range []int{1, MaxBatchSize} {
		descriptors := benchmarkCloneDescriptors(n)

		b.Run(fmt.Sprintf("ProtoClone/N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			var sink *pbc.ResourceDescriptor
			for range b.N {
				for _, d := range descriptors {
					//nolint:forcetypeassert // Type assertion guaranteed by proto.Clone contract.
					sink = proto.Clone(d).(*pbc.ResourceDescriptor)
				}
			}
			_ = sink
		})

		b.Run(fmt.Sprintf("DescriptorClone/N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			var sink *pbc.ResourceDescriptor
			for range b.N {
				for _, d := range descriptors {
					sink = descriptorClone(d)
				}
			}
			_ = sink
		})
	}
}

// BenchmarkBatchResultConstruction_1000 measures clone overhead in batch
// result construction at MaxBatchSize: every result embeds a cloned
// descriptor via newResourceDataResult or newResourceErrorResult (issue #402).
func BenchmarkBatchResultConstruction_1000(b *testing.B) {
	descriptors := benchmarkCloneDescriptors(MaxBatchSize)
	data := &pbc.CostData{
		Data: &pbc.CostData_Estimate{
			Estimate: &pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 1},
		},
	}
	resultErr := NewResourceError(404, "resource not found", false)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for i, d := range descriptors {
			if i%2 == 0 {
				benchmarkCloneSink = newResourceDataResult(d, data)
			} else {
				benchmarkCloneSink = newResourceErrorResult(d, resultErr)
			}
		}
	}
}
