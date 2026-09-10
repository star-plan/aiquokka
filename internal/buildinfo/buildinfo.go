// Package buildinfo exposes values stamped into release binaries by GoReleaser.
package buildinfo

var (
	// Version is the release version, without a leading "v".
	Version = "dev"
	// Commit is the source revision used for the build.
	Commit = "unknown"
	// Date is the UTC date on which the binary was built.
	Date = "unknown"
)
