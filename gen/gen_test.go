package gen

import (
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
		// loom spin <spinTestParms.spinUrl> --name <spinTestParms.name> --outdir /tmp/testspin123456789
		//
		// spinUrl describes the location of the data on the internet, either url in github format
		// or the name of a loom project with hardcoded url parts; LoomUrlBase and LoomUrlEnd.
		//
		// --name is the optional name the user wants to call the project.
		//
		// the output directory is always a random tempory directory e.g. /tmp/testspin123456789 above.
		//
		// The data atually returned and unziped into the outDir is <spinTestParms.dataFile>,
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
	// unzip data into to is actually created.
	for _, test := range spins {
		// Create a different tempory directory for each test
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
		// loom spin <spinArgument> --outdir <testDir> --name <test.name>
		err = Spin(spinArgument, testDir, test.name)

		if err != nil {
			t.Errorf("error %s while spinning url %s, name %s", err, test.spinUrl, test.name)
		}
		if _, err := os.Stat(willUnzipTo); err != nil {
			t.Errorf("has not made directory %s", willUnzipTo)
		}

		os.RemoveAll(testDir)
		mockServer.Close()
	}

}
