package bincover

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

const (
	set                   = "set"
	count                 = "count"
	atomic                = "atomic"
	tmpArgsFilePrefix     = "integ_args"
	tmpCoverageFilePrefix = "temp_coverage"
)

type CoverageCollector struct {
	MergedCoverageFilename string
	CollectCoverage        bool
	testNum                int
	tmpArgsFile            *os.File
	coverMode              string
	tmpCoverageFilenames   []string
	setupFinished          bool
}

// NewCoverageCollector initializes a CoverageCollector with the specified
// merged coverage filename. CollectCoverage can be set to true to collect coverage,
// or set to false to skip coverage collection. This is provided in order to enable reuse of CoverageCollector
// for tests where coverage measurement is not needed.
func NewCoverageCollector(mergedCoverageFilename string, collectCoverage bool) *CoverageCollector {
	return &CoverageCollector{
		MergedCoverageFilename: mergedCoverageFilename,
		CollectCoverage:        collectCoverage,
	}
}

func (c *CoverageCollector) Setup() error {
	if c.MergedCoverageFilename == "" {
		return errors.New("merged coverage profile filename cannot be empty")
	}
	var err error
	c.tmpArgsFile, err = ioutil.TempFile("", tmpArgsFilePrefix)
	if err != nil {
		return errors.Wrap(err, "error creating temporary args file")
	}
	c.setupFinished = true
	return nil
}

// TearDown merges the coverage profiles collecting from repeated runs of RunBinary.
// It must be called at the teardown stage of the test suite, otherwise no merged coverage profile will be created. 
func (c *CoverageCollector) TearDown() error {
	if c.testNum == 0 {
		return nil
	}
	header := fmt.Sprintf("mode: %s", c.coverMode)
	var parsedProfiles []string
	for _, filename := range c.tmpCoverageFilenames {
		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			return errors.Wrap(err, "error reading temp coverage profiles")
		}
		profile := string(buf)
		loc := strings.Index(profile, header)
		if loc == -1 {
			panic("unexpected missing coverage mode from coverage profile")
		}
		parsedProfile := strings.TrimSpace(profile[loc+len(header):])
		parsedProfiles = append(parsedProfiles, parsedProfile)
	}
	mergedProfile := fmt.Sprintf("%s\n%s", header, strings.Join(parsedProfiles, "\n"))
	err := ioutil.WriteFile(c.MergedCoverageFilename, []byte(mergedProfile), 0600)
	if err != nil {
		return errors.Wrap(err, "error writing merged coverage profile")
	}
	return nil
}

// RunBinary runs the instrumented binary at binPath with env environment variables, executing only the test with mainTestName with the specified args.  
func (c *CoverageCollector) RunBinary(binPath string, mainTestName string, env []string, args []string) (output string, exitCode int, err error) {
	if !c.setupFinished {
		panic("RunBinary called before Setup")
	}
	err = c.writeArgs(args)
	if err != nil {
		return "", -1, err
	}
	var binArgs string
	if c.CollectCoverage {
		f, err := ioutil.TempFile("", tmpCoverageFilePrefix)
		if err != nil {
			return "", -1, err
		}
		c.tmpCoverageFilenames = append(c.tmpCoverageFilenames, f.Name())
		binArgs = fmt.Sprintf("-test.run=%s -test.coverprofile=%s -args-file=%s", mainTestName, f.Name(), c.tmpArgsFile.Name())
	} else {
		binArgs = fmt.Sprintf("-test.run=%s -args-file=%s", mainTestName, c.tmpArgsFile.Name())
	}
	_, _ = fmt.Println(binPath, args)
	cmd := exec.Command(binPath, strings.Split(binArgs, " ")...)
	cmd.Env = append(os.Environ(), env...)
	combinedOutput, err := cmd.CombinedOutput()
	if err != nil {
		// This exit code testing requires 1.12 - https://stackoverflow.com/a/55055100/337735.
		if exitError, ok := err.(*exec.ExitError); ok {
			binExitCode := exitError.ExitCode()
			if binExitCode != 0 {
				format := "unexpected error occurred in RunTest: %s\nRunTest exit code: %d\nRunTest output:\n%s\n"
				log.Panicf(format, err, binExitCode, string(combinedOutput))
			}
		} else {
			format := "unexpected error retrieving RunTest exit code\nRunTest exit error: %s\nRunTest output:\n%s\n"
			log.Panicf(format, err, string(combinedOutput))
		}
	}
	cmdOutput, coverMode, exitCode, err := parseCommandOutput(string(combinedOutput))
	if err != nil {
		return "", -1, errors.Wrap(err, "error parsing instrumented binary output")
	}
	if c.CollectCoverage {
		if c.coverMode == "" {
			c.coverMode = coverMode
		}
		if c.coverMode == "" {
			panic("test coverage must be enabled when CollectCoverage is set to true")
		}
		// https://github.com/wadey/gocovmerge/blob/b5bfa59ec0adc420475f97f89b58045c721d761c/gocovmerge.go#L18
		if c.coverMode != coverMode {
			panic("cannot merge profiles with different modes")
		}
		if c.coverMode != set && c.coverMode != count && c.coverMode != atomic {
			log.Panicf("unexpected coverage mode \"%s\" encountered. Coverage mode must be set, count, or atomic", c.coverMode)
		}
		c.testNum++
	}
	return cmdOutput, exitCode, err
}

func (c *CoverageCollector) writeArgs(args []string) error {
	err := c.tmpArgsFile.Truncate(0)
	if err != nil {
		return err
	}
	_, err = c.tmpArgsFile.Seek(0, 0)
	if err != nil {
		return err
	}
	argStr := strings.Join(args, "\n")
	_, err = c.tmpArgsFile.WriteAt([]byte(argStr), 0)
	if err != nil {
		return err
	}
	return err
}

func parseCommandOutput(output string) (cmdOutput string, coverMode string, exitCode int, err error) {
	startIndex := strings.Index(output, startOfMetadataMarker)
	if startIndex == -1 {
		panic("metadata start marker is unexpectedly missing")
	}
	endIndex := strings.Index(output, endOfMetadataMarker)
	if endIndex == -1 {
		panic("metadata end marker is unexpectedly missing")
	}
	cmdOutput = output[:startIndex]
	tail := output[startIndex+len(startOfMetadataMarker) : endIndex]
	// Trim extra newline after cmd output.
	metadataStr := strings.TrimSpace(tail)
	var metadata testMetadata
	err = json.Unmarshal([]byte(metadataStr), &metadata)
	if err != nil {
		return "", "", -1, err
	}
	return cmdOutput, metadata.CoverMode, metadata.ExitCode, nil
}
