package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/confluentinc/bincover"
)

var (
	collector *bincover.CoverageCollector
	binPath   string
)

func TestMainMethod(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)
	binPath = path.Join(dir, "instr_bin")
	collector = bincover.NewCoverageCollector("echo_arg_coverage.out", true)
	err = collector.Setup()
	require.NoError(t, err)
	defer func() {
		err := collector.TearDown()
		if err != nil {
			panic(err)
		}
	}()
	tests := []struct {
		name          string
		args          []string
		wantOutput    string
		outputPattern *regexp.Regexp
		wantExitCode  int
	}{
		{
			name:         "succeed running main with one arg",
			args:         []string{"hello"},
			wantOutput:   "Your argument is \"hello\"\n",
			wantExitCode: 0,
		},
		{
			name:          "fail running main with two args",
			args:          []string{"hello", "world"},
			wantOutput:    "",
			outputPattern: regexp.MustCompile(".*panic.*More than 2 arguments provided!"),
			wantExitCode:  1,
		},
		{
			name:         "fail running main with no args",
			args:         []string{""},
			wantOutput:   "Please provide an argument\n",
			wantExitCode: 1,
		},
	}
	for _, tt := range tests {
		fmt.Println(tt.name)
		output, exitCode, err := collector.RunBinary(binPath, "TestBincoverRunMain", []string{}, tt.args)
		require.NoError(t, err)
		if tt.outputPattern != nil {
			require.Regexp(t, tt.outputPattern, output)
		} else {
			require.Equal(t, tt.wantOutput, output)
		}
		require.Equal(t, tt.wantExitCode, exitCode)
	}
	require.NoError(t, err)
}
