package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ffmpegBaseURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/%s/"
	binDir        = "bin"
)

func downloadFFmpeg() error {

	fileName, err := getSourceDir()
	if err != nil {
		return err
	}

	// Construct download URL
	filePath := filepath.Join(os.TempDir(), fileName)

	// Check if file already exists before downloading
	if _, err := os.Stat(filePath); err == nil {

	} else {
		err = downloadArchive(filePath)
		if err != nil {
			return err
		}
	}

	// Extract files
	switch runtime.GOOS {
	case "windows":
		err = unzipFile(filePath, binDir)
	case "darwin", "linux":
		err = untarFile(filePath, binDir)
	}
	if err != nil {
		return err
	}

	return nil
}

func downloadArchive(filePath string) error {

	// Create local file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	url := fmt.Sprintf(ffmpegBaseURL, filepath.Base(filePath))

	// Create output directory
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("download URL not found (404): %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Save downloaded file
	_, err = io.Copy(out, resp.Body)

	return err
}

func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func untarFile(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

func isFFmpegInstalled() bool {
	_, err1 := exec.LookPath("ffmpeg")
	_, err2 := exec.LookPath("ffprobe")
	return err1 == nil && err2 == nil
}

func configureFFmpeg(download *bool) bool {
	if *download {
		binaryDir := getBinaryDir()

		os.Setenv("PATH", os.Getenv("PATH")+";"+binaryDir)
		if isFFmpegInstalled() {
			fmt.Println("Use downloaded ffmpeg")
			return true
		}

		fmt.Println("FFmpeg not found. Downloading...")
		if err := downloadFFmpeg(); err != nil {
			log.Fatalf("Failed to download FFmpeg: %v", err)
		}

		return true
	} else {
		log.Fatal("FFmpeg not installed. Use -download-ffmpeg flag to download.")
		return false
	}
}

func getSourceDir() (string, error) {
	var fileName string
	switch runtime.GOOS {
	case "windows":
		fileName = fmt.Sprintf("ffmpeg-master-latest-win64-gpl-shared.zip")
	case "darwin":
		fileName = fmt.Sprintf("ffmpeg-master-macos64-gpl.zip")
	case "linux":
		fileName = fmt.Sprintf("ffmpeg-master-linux64-gpl.tar.xz")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return fileName, nil
}

func getBinaryDir() string {
	file, _ := getSourceDir()
	path := filepath.Join(binDir, strings.TrimSuffix(file, filepath.Ext(file)), "bin")
	abs, _ := filepath.Abs(path)
	return abs
}
