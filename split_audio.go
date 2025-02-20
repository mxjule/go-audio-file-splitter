package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type AudioFile struct {
	chapters []Chapter
	format   string
}

type Chapter struct {
	Title     string
	StartTime string
	EndTime   string
}

func splitAudioByChapters(inputFile, outputDir string) error {

	file, err := getAudioFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to extract chapters: %v", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Use a wait group to manage concurrent goroutines
	var wg sync.WaitGroup
	// Use a semaphore to limit concurrent ffmpeg processes
	semaphore := make(chan struct{}, 4) // Limit to 4 concurrent processes
	// Channel to collect any errors
	errChan := make(chan error, len(file.chapters))

	// Split each chapter
	for i, chapter := range file.chapters {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(i int, chapter Chapter) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			outputFilename := filepath.Join(outputDir, fmt.Sprintf("%d_%s%s", i, chapter.Title, file.format))

			cmd := exec.Command(embeddedFfmpeg,
				"-i", inputFile,
				"-ss", chapter.StartTime,
				"-to", chapter.EndTime,
				"-c", "copy",
				outputFilename)

			if err := cmd.Run(); err != nil {
				errChan <- fmt.Errorf("failed to split chapter %s: %v", chapter.Title, err)
				return
			}

			fmt.Printf("Created: %s\n", outputFilename)
		}(i, chapter)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func getAudioFile(inputFile string) (*AudioFile, error) {
	// Use FFprobe to extract chapter information
	cmd := exec.Command(embeddedFfprobe,
		"-v", "quiet",
		"-print_format", "csv",
		"-show_chapters",
		inputFile)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ffprobe: %v", err)
	}

	chapters := []Chapter{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		startTime := parts[4]
		endTime := parts[6]
		title := sanitizeFilename(parts[7])

		chapters = append(chapters, Chapter{
			Title:     title,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return &AudioFile{
		chapters: chapters,
		format:   filepath.Ext(inputFile),
	}, nil
}

func sanitizeFilename(filename string) string {
	// Remove or replace characters not suitable for filenames
	replace := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\r", "\n", "_"}
	for _, char := range replace {
		filename = strings.Trim(strings.ReplaceAll(filename, char, " "), " ")
	}
	return filename
}
