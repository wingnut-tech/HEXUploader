package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ListKeys[K string, V any](m map[K]V) []K {
	o := make([]K, 0, len(m))
	for k := range m {
		o = append(o, k)
	}
	sort.Slice(o, func(i, j int) bool {
		return o[i] > o[j]
	})

	return o
}

func DownloadFile(filename string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: " + err.Error())
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error downloading file")
	}

	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("download: " + err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("download: " + err.Error())
	}

	return nil
}

func UnzipFile(filename string, dest string) ([]string, error) {
	fileNames := []string{}

	r, err := zip.OpenReader(filename)
	if err != nil {
		return fileNames, err
	}

	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fileNames, fmt.Errorf("%s: illegal filepath", fpath)
		}

		fileNames = append(fileNames, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fileNames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fileNames, err
		}

		rc, err := f.Open()
		if err != nil {
			return fileNames, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fileNames, err
		}
	}

	return fileNames, nil
}
