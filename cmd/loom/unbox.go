package main

import (
	"errors"
	"strings"
	"os"
	"io"
	"fmt"
	"path/filepath"
	"net/http"
	"archive/zip"

	"github.com/spf13/cobra"
)

var (
	defaultProjectName = "MyLoomProject"
	tempDownlodFilename = "__tempBox.zip"
)

type unboxFlags struct {
	OutDir string 	`json:"outDir"`
	Name string 	`json:"name"`
}

func newUnboxCommand() *cobra.Command {
	var flags unboxFlags

	unboxCmd := &cobra.Command{
		Use:   "unbox",
		Short: "Unbox a loom project from a github repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return errors.New("No repository entered")
			}

			return unbox(args[0], flags)
		},
	}
	unboxCmd.Flags().StringVar(&flags.OutDir, "outDir", "", "output directory")
	unboxCmd.Flags().StringVar(&flags.Name, "name", defaultProjectName, "project name")
	return unboxCmd

}

func unbox(box string, flags unboxFlags) error {
	outdir := getOutDir(flags)
	err := os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		return err
	}
	tempZip := filepath.Join(outdir, tempDownlodFilename)
	boxTitle, boxUrl, err := getRepoPath(box)
	if err != nil {
		return err
	}
	err = DownloadFile(tempZip, boxUrl)
	if err != nil {
		return err
	}
	_, err = Unzip(tempZip, outdir)
	os.Remove(tempZip)
	os.Rename(filepath.Join(outdir, boxTitle + "-master"), filepath.Join(outdir, flags.Name))
	return err
}
//https://github.com/loomnetwork/cryptozombie-lessons.git
//https://github.com/loomnetwork/cryptozombie-lessons/archive/master.zip
func getRepoPath(box string) (string, string, error) {
	splitBox := strings.Split(box, "/")
	l := len(splitBox)
	if l == 0 {
		return "", "", errors.New("missing box name")
	}
	if l == 1 {
		return splitBox[0], "https://github.com/loomnetwork/" + splitBox[0] + "/archive/master.zip", nil
	}
	if len(splitBox[l-1]) < 5 {
		return "", "", fmt.Errorf("unkowon box format %q, expectin .git or .zip", box)
	}
	format := splitBox[l-1][len(splitBox[l-1])-4:]
	if format == ".zip" {
		return splitBox[l-3], box, nil
	} else if format == ".git" {
		return splitBox[l-1][:len(splitBox[l-1])-4], box[:len(box)-4] + "/archive/master.zip", nil
	} else {
		return "", "", fmt.Errorf("wrong box format %q, loom project or GitHub zipfile", box)
	}
}

func getOutDir(flags unboxFlags) (string) {
	if len(flags.OutDir) == 0 {
		outdir := filepath.Join(os.Getenv("GOPATH"),"src","github.com",os.Getenv("USER"))
		return outdir
	} else {
		return filepath.Join(flags.OutDir)
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

	var filenames []string

	r, err := zip.OpenReader(src)
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
		fpath := filepath.Join(dest, f.Name)
		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
























