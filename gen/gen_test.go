package gen

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/goware/httpmock"
)

// Warning this test will modify the following directories add and removing
// the loom projects created in the tests. It will also create these directories
// if they do not exist yet.
// filepath.Join(os.Getenv("HOME"), "Documents", "gentests") and
// filepath.Join(os.Getenv("GOPATH"),"src","github.com",os.Getenv("USER"))
var (
	ip = "127.0.0.1:10000"
	testDirectory = filepath.Join(os.Getenv("HOME"), "Documents", "gentests")
)

func add(mockService *httpmock.MockHTTPServer, spinUrl string, testFile string) error {
	// define request->response pairs
	requestUrl, _ := url.Parse(spinUrl)
	raw, err := ioutil.ReadFile(testFile)
	if err != nil {
		return err
	}
	mockService.AddResponses([]httpmock.MockResponse{
		{
			Request: http.Request{
				Method: "GET",
				URL:    requestUrl,
			},
			Response: httpmock.Response{
				StatusCode: 200,
				Body:       string(raw),
			},
		},
	})
	return nil
}

func TestSpin(t *testing.T) {
	type spinTestParms struct {
		spinUrl, outDir, name, dataFile string
	}

	spins := []spinTestParms{
		{
			"http://127.0.0.1:10000/github.com/loomnetwork/etherboy-core/archive/master.zip",
			"", "", "./testdata/etherboy-core-master.zip",
		},
		{
			"http://127.0.0.1:10000/github.com/loomnetwork/weave-etherboy-core/archive/master.zip",
			"", "", "./testdata/weave-etherboy-core-master.zip",
		},
		{
			"http://127.0.0.1:10000/github.com/loomnetwork/weave-etherboy-core/archive/master.zip",
			"", "myetherboyproject",
			"./testdata/weave-etherboy-core-master.zip",
		},
		{
			"http://127.0.0.1:10000/github.com/loomnetwork/weave-etherboy-core/archive/master.zip",
			testDirectory, "",
			"./testdata/weave-etherboy-core-master.zip",
		},
		{
			"http://127.0.0.1:10000/github.com/loomnetwork/etherboy-core/archive/master.zip",
			testDirectory, "anotherboyproj",
			"./testdata/etherboy-core-master.zip",
		},
	}

	mockService := httpmock.NewMockHTTPServer(ip)

	for _, tests := range spins {

		add(mockService, tests.spinUrl, tests.dataFile)

		spinTitle, _, err := getRepoPath(tests.spinUrl)
		if err != nil {
			t.Error("bad repoPath")
		}
		projName := projectName(tests.name, spinTitle)
		willCreateDir := filepath.Join(getOutDir(tests.outDir), projName)
		os.RemoveAll(willCreateDir)

		err = Spin(tests.spinUrl, tests.outDir, tests.name)
		if err != nil {
			fmt.Println(err)
			t.Error("something went wrong with spinning %s, %s, %s", tests.spinUrl, tests.outDir, tests.name)
		}
		if _, err := os.Stat(willCreateDir); err != nil {
			t.Error("has not made directory %s", willCreateDir)
		}
	}

}
