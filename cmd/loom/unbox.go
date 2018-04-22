package main

import (
	"errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"net/http"
	"io"
	"archive/zip"
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

func unbox(boxName string, flags unboxFlags) error {
	outdir := getOutDir(flags)
	err := os.MkdirAll(outdir, os.ModePerm)
	tempZip := filepath.Join(outdir, tempDownlodFilename)
	DownloadFile(tempZip, getRepoPath(boxName))
	Unzip(tempZip, outdir)
	os.Remove(tempZip)
	os.Rename(filepath.Join(outdir, boxName + "-master"), filepath.Join(outdir, flags.Name))
	return err
}

func getRepoPath(boxName string) (string) {
	return "https://github.com/loomnetwork/" + boxName + "/archive/master.zip"
}

func getOutDir(flags unboxFlags) (string) {
	if len(flags.OutDir) == 0 {
		outdir := filepath.Join(os.Getenv("GOPATH"),"src","github.com",os.Getenv("USER"))
		return outdir
	} else {
		return filepath.Join(flags.OutDir)
	}
	
}

// https://golangcode.com/download-a-file-from-a-url/
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

// https://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
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
























