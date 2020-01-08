package bincover

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunTest(t *testing.T) {
	type args struct {
		f func()
	}
	tests := []struct {
		name       string
		args       args
		argsFile   *os.File
		wantOutput string
		wantArgs   []string
		wantPanic  bool
	}{
		{
			name: "succeed running test",
			args: args{f: func() {
				fmt.Println("The worst thing about prison was the Dementors")
			}},
			argsFile: func() *os.File {
				f := tempFileWithContent(t, "first\nsecond\nthird\n")
				return f
			}(),
			wantArgs: []string{"first", "second", "third"},
			wantOutput: "The worst thing about prison was the Dementors\n" +
				startOfMetadataMarker + "\n{\"cover_mode\":\"" + testing.CoverMode() + "\",\"exit_code\":0}\n" + endOfMetadataMarker + "\n",
		},
		{
			name: "fail running test when error parsing args file",
			args: args{f: func() {
				fmt.Println("Well, well, well, how the turntables")
			}},
			argsFile: func() *os.File {
				f := removedTempFile(t)
				return f
			}(),
			wantArgs:  []string{},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Remove(tt.argsFile.Name())
			s := tt.argsFile.Name()
			argsFilename = &s
			resetArgsFileName := func() {
				var empty string
				argsFilename = &empty
			}
			defer resetArgsFileName()
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()
			tempStdout := tempFile(t)
			defer os.Remove(tempStdout.Name())
			os.Stdout = tempStdout
			if tt.wantPanic {
				require.Panics(t, func() { RunTest(tt.args.f) })
			} else {
				RunTest(tt.args.f)
			}
			_, err := tempStdout.Seek(0, 0)
			require.NoError(t, err)
			buf, err := ioutil.ReadAll(tempStdout)
			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, string(buf))
			require.Equal(t, tt.wantArgs, os.Args[len(os.Args)-len(tt.wantArgs):])
		})
	}
}

func Test_parseCustomArgs(t *testing.T) {
	tests := []struct {
		name     string
		argsFile *os.File
		want     []string
		wantErr  bool
	}{

		{
			name: "succeed parsing args",
			argsFile: func() *os.File {
				return tempFileWithContent(t, "first\nsecond\nthird\n")
			}(),
			want:    []string{"first", "second", "third"},
			wantErr: false,
		},
		{
			name: "fail parsing args when error reading from args file",
			argsFile: func() *os.File {
				return removedTempFile(t)
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Remove(tt.argsFile.Name())
			s := tt.argsFile.Name()
			argsFilename = &s
			resetArgsFileName := func() {
				var empty string
				argsFilename = &empty
			}
			defer resetArgsFileName()
			got, err := parseCustomArgs()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCustomArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCustomArgs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_printMetadata(t *testing.T) {
	type args struct {
		metadata *testMetadata
	}
	tests := []struct {
		name       string
		args       args
		wantOutput string
	}{
		{
			name: "succeed printing metadata",
			args: args{
				metadata: &testMetadata{
					CoverMode: "set",
					ExitCode:  0,
				}},
			wantOutput: startOfMetadataMarker + "\n{\"cover_mode\":\"set\",\"exit_code\":0}\n" + endOfMetadataMarker + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			defer func() { os.Stdout = oldStdout }()
			tempStdout := tempFile(t)
			defer os.Remove(tempStdout.Name())
			os.Stdout = tempStdout
			printMetadata(tt.args.metadata)
			_, err := tempStdout.Seek(0, 0)
			require.NoError(t, err)
			buf, err := ioutil.ReadAll(tempStdout)
			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, string(buf))
		})
	}
}
