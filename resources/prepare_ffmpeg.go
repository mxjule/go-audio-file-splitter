package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	ffmpegURL   = "https://github.com/GyanD/codexffmpeg/releases/download/6.1.1/ffmpeg-6.1.1-essentials_build.zip"
	binariesDir = "binaries"
)

func main() {
	// Create binaries directory if it doesn't exist
	if err := os.MkdirAll(binariesDir, 0755); err != nil {
		fmt.Printf("Error creating binaries directory: %v\n", err)
		return
	}

	// Check if files already exist
	ffmpegPath := filepath.Join(binariesDir, "ffmpeg.exe")
	ffprobePath := filepath.Join(binariesDir, "ffprobe.exe")

	if fileExists(ffmpegPath) && fileExists(ffprobePath) {
		fmt.Println("FFmpeg and FFprobe already exist in binaries folder. Skipping download.")
		return
	}

	tempZip := "ffmpeg.zip"

	// Download FFmpeg
	fmt.Println("Downloading FFmpeg...")
	if err := downloadFile(ffmpegURL, tempZip); err != nil {
		fmt.Printf("Error downloading FFmpeg: %v\n", err)
		return
	}

	// Extract files
	fmt.Println("Extracting files...")
	if err := extractFFmpeg(tempZip, binariesDir); err != nil {
		fmt.Printf("Error extracting files: %v\n", err)
		// Clean up zip file
		os.Remove(tempZip)
		return
	}

	// Clean up zip file
	if err := os.Remove(tempZip); err != nil {
		fmt.Printf("Warning: Could not remove temporary zip file: %v\n", err)
	}

	fmt.Println("FFmpeg and FFprobe have been downloaded and extracted successfully!")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func downloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractFFmpeg(zipPath, destPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Check if the file is ffmpeg.exe or ffprobe.exe
		baseName := filepath.Base(file.Name)
		if strings.EqualFold(baseName, "ffmpeg.exe") || strings.EqualFold(baseName, "ffprobe.exe") {
			// Open the file from the zip
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in zip: %v", err)
			}
			defer rc.Close()

			// Create the output file
			outPath := filepath.Join(destPath, baseName)
			outFile, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("failed to create output file: %v", err)
			}
			defer outFile.Close()

			// Copy the contents
			if _, err := io.Copy(outFile, rc); err != nil {
				return fmt.Errorf("failed to extract file: %v", err)
			}

			fmt.Printf("Extracted: %s\n", baseName)
		}
	}

	return nil
}
