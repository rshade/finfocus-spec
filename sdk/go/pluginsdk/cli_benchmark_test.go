//nolint:testpackage // Benchmarks unexported CLI routing functions (isCLIInvocation, firstPositional)
package pluginsdk

import "testing"

// BenchmarkIsCLIInvocation measures the routing check performed on every CLI
// invocation before Run() decides between the handshake path and ax.Execute.
// It benchmarks isCLIInvocation directly rather than Run/runHandshake, since
// the handshake path enters a long-lived Serve() call.
func BenchmarkIsCLIInvocation(b *testing.B) {
	b.Run("NoArgs", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			isCLIInvocation(nil)
		}
	})

	b.Run("PortOption", func(b *testing.B) {
		args := []string{"--port", "50051"}
		b.ReportAllocs()
		for range b.N {
			isCLIInvocation(args)
		}
	})

	b.Run("DryRunCommand", func(b *testing.B) {
		args := []string{"dry-run", "--provider", "aws", "--resource-type", "ec2", "--format=json"}
		b.ReportAllocs()
		for range b.N {
			isCLIInvocation(args)
		}
	})
}

// BenchmarkFirstPositional measures the argument-scanning helper that
// isCLIInvocation and parseHandshakeArgs both rely on.
func BenchmarkFirstPositional(b *testing.B) {
	b.Run("NoArgs", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			firstPositional(nil)
		}
	})

	b.Run("PortOption", func(b *testing.B) {
		args := []string{"--port", "50051"}
		b.ReportAllocs()
		for range b.N {
			firstPositional(args)
		}
	})

	b.Run("DryRunCommand", func(b *testing.B) {
		args := []string{"dry-run", "--provider", "aws", "--resource-type", "ec2", "--format=json"}
		b.ReportAllocs()
		for range b.N {
			firstPositional(args)
		}
	})
}
