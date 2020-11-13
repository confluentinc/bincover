package bincover

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Build necessary binaries before executing unit tests.
	buildTestCmd := exec.Command("go", []string{"test", "./test_bins", "-tags", "testrunmain", "-coverpkg=./...", "-c", "-o", "set_covermode"}...)
	output, err := buildTestCmd.CombinedOutput()
	if err != nil {
		log.Println(output)
		panic(err)
	}
	exitCode := m.Run()
	err = os.Remove("set_covermode")
	if err != nil {
		panic(err)
	}
	os.Exit(exitCode)
}

func TestCoverageCollector_Setup(t *testing.T) {
	type fields struct {
		MergedCoverageFilename string
		CollectCoverage        bool
		setupFinished          bool
		tmpArgsFilePrefix      string
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		errMessage string
	}{
		{
			name: "succeed setting up",
			fields: fields{
				MergedCoverageFilename: "test-file.out",
				CollectCoverage:        false,
				setupFinished:          true,
				tmpArgsFilePrefix:      defaultTmpArgsFilePrefix,
			},
			wantErr: false,
		},
		{
			name: "fail setting up with empty filename",
			fields: fields{
				MergedCoverageFilename: "",
				CollectCoverage:        true,
				setupFinished:          false,
				tmpArgsFilePrefix:      defaultTmpArgsFilePrefix,
			},
			wantErr:    true,
			errMessage: "merged coverage profile filename cannot be empty when CollectCoverage is true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CoverageCollector{
				MergedCoverageFilename: tt.fields.MergedCoverageFilename,
				CollectCoverage:        tt.fields.CollectCoverage,
			}
			if err := c.Setup(); (err != nil) != tt.wantErr {
				t.Errorf("Setup() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr && tt.errMessage != "" {
				require.EqualError(t, err, tt.errMessage)
			}
			require.Equal(t, tt.fields.setupFinished, c.setupFinished)
		})
	}
}

func TestCoverageCollector_TearDown(t *testing.T) {
	type fields struct {
		MergedCoverageFilename string
		CollectCoverage        bool
		coverMode              string
		tmpCoverageFiles       []*os.File
	}
	tests := []struct {
		name               string
		fields             fields
		wantErr            bool
		wantPanic          bool
		panicMessage       string
		errMessageContains string
		errMessage         string
		mergedFileContents string
	}{
		{
			name:    "succeed tearing down with no tests",
			fields:  fields{},
			wantErr: false,
		},
		{
			name: "succeed tearing down with tests",
			fields: fields{
				MergedCoverageFilename: "temp_merged.out",
				CollectCoverage:        true,
				tmpCoverageFiles: func() []*os.File {
					f1 := tempFileWithContent(t, "mode: set\nfirst file\n")
					f2 := tempFileWithContent(t, "mode: set\nsecond file\n")
					f3 := tempFileWithContent(t, "mode: set\nthird file\n")
					return []*os.File{f1, f2, f3}
				}(),
				coverMode: "set",
			},
			mergedFileContents: "mode: set\nfirst file\nsecond file\nthird file",
			wantErr:            false,
		},
		{
			name: "fail tearing down with missing coverage mode",
			fields: fields{
				MergedCoverageFilename: "temp_merged.out",
				CollectCoverage:        true,
				tmpCoverageFiles: func() []*os.File {
					f1 := tempFileWithContent(t, "mode: set\nfirst file\n")
					missingHeaderFile := tempFileWithContent(t, "second file\n")
					return []*os.File{f1, missingHeaderFile}
				}(),
			},
			wantErr:    true,
			errMessage: "error parsing coverage profile: missing coverage mode from coverage profile. Maybe the file got corrupted while writing?",
		},
		{
			name: "fail tearing down with missing temp coverage profiles",
			fields: fields{
				MergedCoverageFilename: "temp_merged.out",
				CollectCoverage:        true,
				tmpCoverageFiles: func() []*os.File {
					f := tempFile(t)
					require.NoError(t, f.Close())
					return []*os.File{f}
				}(),
			},
			wantErr:            true,
			errMessageContains: "error reading temp coverage profiles",
		},
		{
			name: "fail tearing down with invalid merged coverage filename",
			fields: fields{
				MergedCoverageFilename: "inval?df!l3Nam3/.%",
				CollectCoverage:        true,
				tmpCoverageFiles: func() []*os.File {
					f1 := tempFileWithContent(t, "mode: set\nfirst file\n")
					f2 := tempFileWithContent(t, "mode: set\nsecond file\n")
					f3 := tempFileWithContent(t, "mode: set\nthird file\n")
					return []*os.File{f1, f2, f3}
				}(),
			},
			wantErr:            true,
			errMessageContains: "error writing merged coverage profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CoverageCollector{
				MergedCoverageFilename: tt.fields.MergedCoverageFilename,
				CollectCoverage:        tt.fields.CollectCoverage,
				coverMode:              tt.fields.coverMode,
				tmpCoverageFiles:       tt.fields.tmpCoverageFiles,
			}
			if tt.wantPanic {
				require.PanicsWithValue(t, tt.panicMessage, func() { _ = c.TearDown() })
				return
			}
			if err := c.TearDown(); (err != nil) != tt.wantErr {
				t.Errorf("TearDown() error = %v, wantErr %v", err, tt.wantErr)
			} else if err != nil && tt.errMessageContains != "" {
				require.Contains(t, err.Error(), tt.errMessageContains)
			} else if err != nil && tt.errMessage != "" {
				require.EqualError(t, err, tt.errMessage)
			}
			if c.MergedCoverageFilename != "" && !tt.wantErr {
				defer os.Remove(c.MergedCoverageFilename)
				buf, err := ioutil.ReadFile(c.MergedCoverageFilename)
				require.NoError(t, err)
				contents := string(buf)
				require.Equal(t, tt.mergedFileContents, contents)
			}
		})
	}
}

func TestCoverageCollector_removeTempFiles(t *testing.T) {
	tests := []struct {
		name             string
		tmpCoverageFiles []*os.File
		tmpArgsFile      *os.File
		stdErrOutputFmt  string
	}{
		{
			name: "succeed removing temp files",
			tmpCoverageFiles: func() []*os.File {
				return []*os.File{tempFile(t)}
			}(),
			tmpArgsFile: func() *os.File {
				return tempFile(t)
			}(),
			stdErrOutputFmt: "",
		},
		{
			name: "fail silently removing nonexistent temp files",
			tmpCoverageFiles: func() []*os.File {
				f := removedTempFile(t)
				return []*os.File{f}
			}(),
			tmpArgsFile: func() *os.File {
				return removedTempFile(t)
			}(),
			stdErrOutputFmt: ".*error removing.*\n.*error removing temp arg file.*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CoverageCollector{}
			c.tmpCoverageFiles = tt.tmpCoverageFiles
			c.tmpArgsFile = tt.tmpArgsFile
			var buf bytes.Buffer
			log.SetOutput(&buf)
			c.removeTempFiles()
			log.SetOutput(os.Stderr)
			require.Regexp(t, tt.stdErrOutputFmt, buf.String())
			for _, file := range c.tmpCoverageFiles {
				_, err := os.Stat(file.Name())
				require.True(t, os.IsNotExist(err))
			}
			_, err := os.Stat(c.tmpArgsFile.Name())
			require.True(t, os.IsNotExist(err))
		})
	}
}

func TestNewCoverageCollector(t *testing.T) {
	type args struct {
		mergedCoverageFilename string
		collectCoverage        bool
	}
	tests := []struct {
		name string
		args args
		want *CoverageCollector
	}{
		{
			name: "succeed creating CoverageCollector instance",
			args: args{
				mergedCoverageFilename: "fake.file",
				collectCoverage:        false,
			},
			want: &CoverageCollector{
				MergedCoverageFilename: "fake.file",
				CollectCoverage:        false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCoverageCollector(tt.args.mergedCoverageFilename, tt.args.collectCoverage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCoverageCollector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseCommandOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name          string
		args          args
		wantCmdOutput string
		wantCoverMode string
		wantExitCode  int
		wantPanic     bool
		panicMessage  string
	}{
		{
			name:         "panic if metadata start marker is missing",
			args:         args{output: ""},
			wantPanic:    true,
			panicMessage: "metadata start marker is unexpectedly missing",
		},
		{
			name:         "panic if metadata end marker is missing",
			args:         args{output: startOfMetadataMarker},
			wantPanic:    true,
			panicMessage: "metadata end marker is unexpectedly missing",
		},
		{
			name:         "panic if error occurs while unmarshalling testMetadata",
			args:         args{output: startOfMetadataMarker + "invalid" + endOfMetadataMarker},
			wantPanic:    true,
			panicMessage: "error unmarshalling testMetadata struct from RunTest",
		},
		{
			name: "succeed parsing command output",
			args: args{
				output: "test output" + startOfMetadataMarker + "\n{\"cover_mode\":\"set\",\"exit_code\":1}\n" + endOfMetadataMarker + "\n",
			},
			wantCmdOutput: "test output",
			wantCoverMode: "set",
			wantExitCode:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.PanicsWithValue(t, tt.panicMessage, func() { parseCommandOutput(tt.args.output) })
				return
			}
			gotCmdOutput, gotCoverMode, gotExitCode := parseCommandOutput(tt.args.output)
			if gotCmdOutput != tt.wantCmdOutput {
				t.Errorf("parseCommandOutput() gotCmdOutput = %v, want %v", gotCmdOutput, tt.wantCmdOutput)
			}
			if gotCoverMode != tt.wantCoverMode {
				t.Errorf("parseCommandOutput() gotCoverMode = %v, want %v", gotCoverMode, tt.wantCoverMode)
			}
			if gotExitCode != tt.wantExitCode {
				t.Errorf("parseCommandOutput() gotExitCode = %v, want %v", gotExitCode, tt.wantExitCode)
			}
		})
	}
}

func TestCoverageCollector_writeArgs(t *testing.T) {
	type fields struct {
		MergedCoverageFilename string
		CollectCoverage        bool
		tmpArgsFile            *os.File
		coverMode              string
		tmpCoverageFiles       []*os.File
		setupFinished          bool
	}
	type args struct {
		args []string
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantErr            bool
		wantArgFileContent string
	}{
		{
			name: "fail when writing args to closed file",
			fields: fields{
				tmpArgsFile: func() *os.File {
					f := tempFile(t)
					require.NoError(t, f.Close())
					return f
				}(),
			},
			wantErr: true,
		},
		{
			name: "succeed writing args",
			fields: fields{
				tmpArgsFile: func() *os.File {
					return tempFile(t)
				}(),
			},
			args:               args{args: []string{"first", "second", "third"}},
			wantArgFileContent: "first\nsecond\nthird",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Remove(tt.fields.tmpArgsFile.Name())
			c := &CoverageCollector{
				MergedCoverageFilename: tt.fields.MergedCoverageFilename,
				CollectCoverage:        tt.fields.CollectCoverage,
				tmpArgsFile:            tt.fields.tmpArgsFile,
				coverMode:              tt.fields.coverMode,
				tmpCoverageFiles:       tt.fields.tmpCoverageFiles,
				setupFinished:          tt.fields.setupFinished,
			}
			if err := c.writeArgs(tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("writeArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				buf, err := ioutil.ReadAll(c.tmpArgsFile)
				require.NoError(t, err)
				require.Equal(t, tt.wantArgFileContent, string(buf))
			}
		})
	}
}

func TestCoverageCollector_RunBinary(t *testing.T) {
	type fields struct {
		MergedCoverageFilename string
		CollectCoverage        bool
		coverMode              string
	}
	type args struct {
		binPath      string
		mainTestName string
		env          []string
		args         []string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantOutput   string
		wantExitCode int
		wantErr      bool
		errMessage   string
		wantPanic    bool
		panicMessage string
		skipSetup    bool
		stdinInput   string
	}{
		{
			name:         "panic if Setup not called",
			wantPanic:    true,
			panicMessage: "RunBinary called before Setup",
			skipSetup:    true,
		},
		{
			name:       "fail if error exists and is not an ExitError when executing command at 'binPath'",
			wantErr:    true,
			errMessage: "unexpected error running command \"invalid.exec\": exec: \"invalid.exec\": executable file not found in $PATH",
			args: args{
				binPath:      "invalid.exec",
				mainTestName: "",
				env:          nil,
				args:         nil,
			},
			wantExitCode: -1,
		},
		{
			name:       "fail if error exists and is an ExitError when executing command at 'binPath'",
			wantErr:    true,
			errMessage: "unsuccessful exit by command \"./test_bins/exit_1.sh\"\nExit code: 1\nOutput:\nHello world\n: exit status 1",
			args: args{
				binPath:      "./test_bins/exit_1.sh",
				mainTestName: "",
				env:          nil,
				args:         nil,
			},
			wantExitCode: 1,
		},
		{
			name: "succeed running binary when coverage is disabled",
			args: args{
				binPath:      "./set_covermode",
				mainTestName: "TestRunMain",
				env:          nil,
				args:         nil,
			},
			fields: fields{
				MergedCoverageFilename: "",
				CollectCoverage:        false,
			},
			wantOutput:   "Hello world\n",
			wantExitCode: 1,
		},
		{
			name: "succeed running binary when coverage is enabled",
			args: args{
				binPath:      "./set_covermode",
				mainTestName: "TestRunMain",
				env:          nil,
				args:         nil,
			},
			fields: fields{
				MergedCoverageFilename: "temp_coverage.out",
				CollectCoverage:        true,
			},
			wantOutput:   "Hello world\n",
			wantExitCode: 1,
		},
		{
			name: "panic running binary which outputs empty coverage mode",
			args: args{
				binPath:      "./test_bins/empty_covermode.sh",
				mainTestName: "",
				env:          nil,
				args:         nil,
			},
			fields: fields{
				MergedCoverageFilename: "temp_coverage.out",
				CollectCoverage:        true,
			},
			wantPanic:    true,
			panicMessage: "coverage mode cannot be empty. test coverage must be enabled when CollectCoverage is set to true",
		},
		{
			name: "panic running binary which outputs different coverage mode",
			args: args{
				binPath:      "./set_covermode",
				mainTestName: "TestRunMain",
				env:          nil,
				args:         nil,
			},
			fields: fields{
				MergedCoverageFilename: "temp_coverage.out",
				CollectCoverage:        true,
				coverMode:              "atomic",
			},
			wantPanic:    true,
			panicMessage: "cannot merge profiles with different coverage modes",
		},
		{
			name: "panic running binary which outputs unexpected coverage mode",
			args: args{
				binPath:      "./test_bins/unexpected_covermode.sh",
				mainTestName: "",
				env:          nil,
				args:         nil,
			},
			fields: fields{
				MergedCoverageFilename: "temp_coverage.out",
				CollectCoverage:        true,
			},
			wantPanic:    true,
			panicMessage: "unexpected coverage mode \"evil\" encountered. Coverage mode must be set, count, or atomic",
		},
		{
			name: "fail running binary if there are no tests to run",
			args: args{
				binPath:      "./test_bins/no_tests.sh",
				mainTestName: "",
				env:          nil,
				args:         nil,
			},
			wantErr:      true,
			errMessage:   "testing: warning: no tests to run\n",
			wantExitCode: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCoverageCollector(tt.fields.MergedCoverageFilename, tt.fields.CollectCoverage)
			c.coverMode = tt.fields.coverMode
			if !tt.skipSetup {
				require.NoError(t, c.Setup())
			}
			if tt.wantPanic {
				require.PanicsWithValue(t,
					tt.panicMessage,
					func() { _, _, _ = c.RunBinary(tt.args.binPath, tt.args.mainTestName, tt.args.env, tt.args.args, tt.stdinInput) },
				)
				return
			}
			gotOutput, gotExitCode, err := c.RunBinary(tt.args.binPath, tt.args.mainTestName, tt.args.env, tt.args.args, tt.stdinInput)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil && tt.errMessage != err.Error() {
				t.Errorf("RunBinary() error =\n%v\nerrMessage =\n%v", err, tt.errMessage)
			}
			if tt.fields.CollectCoverage {
				require.Equal(t, 1, len(c.tmpCoverageFiles))
				buf, err := ioutil.ReadAll(c.tmpCoverageFiles[0])
				require.NoError(t, err)
				require.NotZero(t, len(buf))
			} else {
				require.Zero(t, len(c.tmpCoverageFiles))
			}
			if gotOutput != tt.wantOutput {
				t.Errorf("RunBinary() gotOutput = %v, want %v", gotOutput, tt.wantOutput)
			}
			if gotExitCode != tt.wantExitCode {
				t.Errorf("RunBinary() gotExitCode = %v, want %v", gotExitCode, tt.wantExitCode)
			}
		})
	}
}

func tempFile(t *testing.T) *os.File {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	return f
}

func removedTempFile(t *testing.T) *os.File {
	f := tempFile(t)
	require.NoError(t, os.Remove(f.Name()))
	return f
}

func tempFileWithContent(t *testing.T, content string) *os.File {
	f := tempFile(t)
	_, err := f.WriteString(content)
	require.NoError(t, err)
	_, err = f.Seek(0, 0)
	require.NoError(t, err)
	return f
}
