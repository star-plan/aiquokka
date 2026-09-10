package cmd

import (
	"bytes"
	"testing"

	"github.com/star-plan/aiquokka/internal/buildinfo"
)

func TestVersionOutput(t *testing.T) {
	originalVersion, originalCommit, originalDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	t.Cleanup(func() {
		buildinfo.Version, buildinfo.Commit, buildinfo.Date = originalVersion, originalCommit, originalDate
	})

	buildinfo.Version = "0.1.0"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-09-10"

	root := newRootCmd()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	const want = "aiquokka 0.1.0\ncommit abc1234\nbuilt 2026-09-10\n"
	if got := output.String(); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestDevelopmentVersionOutput(t *testing.T) {
	originalVersion := buildinfo.Version
	t.Cleanup(func() { buildinfo.Version = originalVersion })
	buildinfo.Version = "dev"

	if got, want := versionOutput(), "aiquokka dev\n"; got != want {
		t.Fatalf("version template = %q, want %q", got, want)
	}
}

func TestBuildDateFormatsGoReleaserTimestamp(t *testing.T) {
	originalDate := buildinfo.Date
	t.Cleanup(func() { buildinfo.Date = originalDate })
	buildinfo.Date = "2026-09-10T12:34:56Z"

	if got, want := buildDate(), "2026-09-10"; got != want {
		t.Fatalf("build date = %q, want %q", got, want)
	}
}
