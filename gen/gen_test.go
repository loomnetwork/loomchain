package gen

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newMockServer(t *testing.T, filename string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("Error reading file %s", filename)
			return
		}
		w.Write(raw)
	}))

}

func TestSpin(t *testing.T) {
	type spinTestParms struct {
		// An array of tests. The command run in each test is:
		// loom spin <sinUrl> --name <name> --outdir /tmp/testspin123456789
		// The data atually returned and unziped into the outDir is <dataFile>,
		// the output directory is a random tempory directory e.g. /tmp/testspin123456789 above.
		spinUrl,
		name,
		dataFile string
	}
	spins := []spinTestParms{
		{
			spinUrl:  "testproj",
			name:     "",
			dataFile: "./testdata/testproj-master.zip",
		},
		{
			spinUrl:  "weave-testproj",
			name:     "",
			dataFile: "./testdata/weave-testproj-master.zip",
		},
		{
			spinUrl:  "A/testproj1-core/a/master.zip",
			name:     "",
			dataFile: "./testdata/testproj-master.zip",
		},
		{
			spinUrl:  "B/testproj2-core/b/master.zip",
			name:     "mytestproject",
			dataFile: "./testdata/testproj-master.zip",
		},
		{
			spinUrl:  "C/weave-testproj3/c/master.zip",
			name:     "",
			dataFile: "./testdata/weave-testproj-master.zip",
		},
		{
			spinUrl:  "D/weave-testproj4-core/d/master.zip",
			name:     "anothertestproj",
			dataFile: "./testdata/weave-testproj-master.zip",
		},
	}

	// For each test we check that the directory that we expect the spin command to
	// unzip data to is actually created.
	// Later when the unzipped data is givin structure we can test that.
	for _, test := range spins {
		testDir, err := ioutil.TempDir("", "testspin")
		if err != nil {
			t.Errorf("error creating test directory, %v", err)
			continue
		}

		// determine the unzip directory
		spinTitle, _, err := getRepoPath(test.spinUrl)
		if err != nil {
			t.Error("bad repoPath")
			os.RemoveAll(testDir)
			continue
		}
		projName := projectName(test.name, spinTitle)
		willUnzipTo := filepath.Join(getOutDir(testDir), projName)

		// Create a mock server and ammend urls to point to the mock server.
		// If referencing by loom package name rather than URL, we ammend the hard coded value
		// in spin.go to point to the mock server.
		mockServer := newMockServer(t, test.dataFile)
		spinArgument := test.spinUrl
		if strings.Contains(test.spinUrl, "/") {
			spinArgument = mockServer.URL + "/" + test.spinUrl
		} else {
			LoomUrlBase = mockServer.URL + "/" + "someloompath"
		}

		// Run command, the same as
		// loom spin <spinArgument> --name <test.name> --outdir <testDir>
		err = Spin(spinArgument, testDir, test.name)

		if err != nil {
			fmt.Println(err)
			t.Error("error %v while spinning url %s, name %s", err, test.spinUrl, test.name)
		}
		if _, err := os.Stat(willUnzipTo); err != nil {
			t.Error("has not made directory %s", willUnzipTo)
		}

		os.RemoveAll(testDir)
		mockServer.Close()
	}

}
