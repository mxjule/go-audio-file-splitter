package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	ffmpeg  = "ffmpeg.exe"
	ffprobe = "ffprobe.exe"
)

var embeddedBinaries = filepath.Join(os.TempDir(), "embedded-resources")
var embeddedFfmpeg = filepath.Join(embeddedBinaries, ffmpeg)
var embeddedFfprobe = filepath.Join(embeddedBinaries, ffprobe)

//go:embed binaries
var binaries embed.FS

func main() {
	inputFile := flag.String("input", "", "Input audio file")
	outputDir := flag.String("output", "chapters", "Output directory")
	removeOriginal := flag.Bool("remove", false, "Remove original file after splitting")

	flag.Parse()

	// Create binaries directory if it doesn't exist
	if err := UnpackBinaries(); err != nil {
		fmt.Printf("Error unpack binaries: %v\n", err)
		return
	}

	if *inputFile == "" {
		log.Fatal("Please provide an input file")
	}

	if err := splitAudioByChapters(*inputFile, *outputDir); err != nil {
		log.Fatalf("Failed to split audio: %v", err)
	}

	fmt.Println("Audio split completed successfully")

	if *removeOriginal {
		if err := os.Remove(*inputFile); err != nil {
			log.Printf("Failed to remove original file: %v", err)
		} else {
			fmt.Println("Original file removed")
		}
	}
}

func UnpackBinaries() error {
	// Create temp directory
	if err := os.MkdirAll(embeddedBinaries, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}

	if _, err := os.Stat(embeddedFfmpeg); err == nil {

	} else {
		entries, err := binaries.ReadDir("binaries")
		if err != nil {
			return fmt.Errorf("Error reading directory: %v\n", err)
		}

		for _, entry := range entries {
			fmt.Printf("Extract - %s \n", entry.Name())

			data, err := binaries.ReadFile("binaries/" + entry.Name())
			if err != nil {
				return fmt.Errorf("failed to read embedded binary: %v", err)
			}

			// Write to temp file
			if err := os.WriteFile(filepath.Join(embeddedBinaries, entry.Name()), data, 0755); err != nil {
				return fmt.Errorf("failed to write binary: %v", err)
			}
		}
	}

	cmd := exec.Command(embeddedFfmpeg, "-version")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Println(string(output))

	return nil
}
