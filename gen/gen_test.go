package gen

import (
	"fmt"
	"testing"
	"net/http"
	"net/url"
	"io/ioutil"
	"github.com/httpmock"
	"os"
	"path/filepath"
)

var (
	servicePrefix = "http://127.0.0.1:10000/"
	box = "etherboy-core"
	testFile = "./testdata/" + box + "-master.zip"
	argOutDir = ""
	argName = "MyLoomProject"
	boxUrl = servicePrefix + "github.com/loomnetwork/" + box + "/archive/master.zip"
)

func mockServer(t *testing.T){
	// new mocking server
	mockService := httpmock.NewMockHTTPServer("127.0.0.1:10000")

	// define request->response pairs
	requestUrl, _ := url.Parse(boxUrl)
	raw, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Error("no test file")
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
}


func TestSomething(t *testing.T) {
	mockServer(t)

	willCreateDir := filepath.Join(getOutDir(argOutDir), argName)
	os.RemoveAll(willCreateDir)
	err := Unbox(boxUrl, argOutDir, argName)
	if err != nil {
		fmt.Println(err)
		t.Error("something went wrong with Unbox %s, %s, %s", boxUrl, argOutDir, argName)
	}
	if _, err := os.Stat(willCreateDir); err != nil {
		t.Error("has not made directory %s", willCreateDir)
	}


}