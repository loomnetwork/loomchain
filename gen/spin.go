package gen

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	tempDownlodFilename = "__tempSpin.zip"
	LoomUrlBase         = "https://github.com/loomnetwork"
	LoomUrlEnd          = "archive/master.zip"
)

func Spin(spin string, argOutDir string, name string) error {
	outdir := getOutDir(argOutDir)
	err := os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "spinzip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	tempZip := filepath.Join(tempDir, tempDownlodFilename)
	spinTitle, spinUrl, err := getRepoPath(spin)
	if err != nil {
		return err
	}

	err = DownloadFile(tempZip, spinUrl)
	if err != nil {
		return err
	}
	files, err := Unzip(tempZip, outdir)

	os.Rename(files[0], filepath.Join(outdir, projectName(name, spinTitle)))
	return err
}

func getRepoPath(spin string) (string, string, error) {
	splitSpin := strings.Split(spin, "/")
	l := len(splitSpin)
	if l == 0 {
		return "", "", errors.New("missing spin name")
	}
	if l == 1 {
		return splitSpin[0], LoomUrlBase + "/" + splitSpin[0] + "/" + LoomUrlEnd, nil
	}
	if len(splitSpin[l-1]) < 5 {
		return "", "", fmt.Errorf("unkowon spin format %q, expectin .git or .zip", spin)
	}
	format := splitSpin[l-1][len(splitSpin[l-1])-4:]
	if format == ".zip" {
		return splitSpin[l-3], spin, nil
	} else if format == ".git" {
		return splitSpin[l-1][:len(splitSpin[l-1])-4], spin[:len(spin)-4] + "/archive/master.zip", nil
	} else {
		return "", "", fmt.Errorf("wrong spin format %q, loom project or GitHub zipfile", spin)
	}
}

func getOutDir(argOutDir string) string {
	if len(argOutDir) == 0 {
		outdir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error finding working directory %v", err)
		}
		return outdir
	} else {
		return filepath.Join(argOutDir)
	}
}

func projectName(argName string, wrapDir string) string {
	if len(argName) == 0 {
		name := wrapDir
		if len(name) > 6 && name[0:6] == "weave-" {
			name = name[6:]
		} else if len(name) > 5 && name[0:5] == "weave" {
			name = name[5:]
		}
		//if len(name) > 7 && name[len(name)-7:] == "-master" {
		//	name = name[0:len(name)-7]
		//}
		return name
	} else {
		return argName
	}

}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Problem downloading data: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	r, err := zip.OpenReader(src)
	filenames := make([]string, 0, len(r.File))
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on

		destpath, err := sanitizeExtractPath(f.Name, dest)
		if err != nil {
			return nil, err
		}

		filenames = append(filenames, destpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(destpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(destpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(destpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}

func sanitizeExtractPath(filePath string, destination string) (string, error) {
	// to avoid zip slip (writing outside of the destination), we resolve
	// the target path, and make sure it's nested in the intended
	// destination, or bail otherwise.
	destpath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destpath, destination) {
		return "", fmt.Errorf("%s: illegal file path", filePath)
	}
	return destpath, nil
}
