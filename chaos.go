package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	inputFile := flag.String("i", "", "Optional: Path to file containing company names (one per line)")
	flag.Parse()

	baseDir := filepath.Join(".", "AllChaosData")
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	var companySet map[string]struct{}
	if *inputFile != "" {
		var err error
		companySet, err = readCompanyList(*inputFile)
		if err != nil {
			log.Fatalf("Failed to read input file: %v", err)
		}
	}

	jsonURL := "https://chaos-data.projectdiscovery.io/index.json"
	if err := processURLs(jsonURL, baseDir, companySet); err != nil {
		log.Fatalf("Failed to process URLs: %v", err)
	}

	if err := concatenateAllTxtFiles(baseDir, "."); err != nil {
		log.Fatalf("Failed to concatenate all txt files: %v", err)
	}
}

func readCompanyList(path string) (map[string]struct{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	companies := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			companies[strings.ToLower(line)] = struct{}{}
		}
	}
	return companies, scanner.Err()
}

func processURLs(jsonURL, baseDir string, filterSet map[string]struct{}) error {
	resp, err := http.Get(jsonURL)
	if err != nil {
		return fmt.Errorf("error fetching JSON index: %w", err)
	}
	defer resp.Body.Close()

	var entries []struct {
		Name string `json:"name"`
		URL  string `json:"URL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("error decoding JSON index: %w", err)
	}

	for _, entry := range entries {
		if filterSet != nil {
			if _, ok := filterSet[strings.ToLower(entry.Name)]; !ok {
				continue
			}
		}
		fmt.Printf("Processing %s...\n", entry.Name)
		if err := downloadAndUnzip(entry.URL, entry.Name, baseDir); err != nil {
			log.Printf("Failed to process %s: %v\n", entry.Name, err)
		}
	}
	return nil
}

func downloadAndUnzip(url, name, baseDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "*.zip")
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err = io.Copy(tempFile, resp.Body); err != nil {
		tempFile.Close()
		return fmt.Errorf("error writing to temp file: %w", err)
	}
	tempFile.Close()

	dirPath := filepath.Join(baseDir, name)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dirPath, err)
	}

	if err := unzipFile(tempFile.Name(), dirPath); err != nil {
		return fmt.Errorf("error unzipping file: %w", err)
	}

	return nil
}

func unzipFile(zipFile, destDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("error opening zip file: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("error opening output file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("error opening zip content: %w", err)
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("error writing to output file: %w", err)
		}
	}
	return nil
}

func concatenateAllTxtFiles(baseDir, outputDir string) error {
	allTxtFiles := findAllTxtFiles(baseDir)

	destPath := filepath.Join(outputDir, "everything.txt")
	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating %s: %w", destPath, err)
	}
	defer dest.Close()

	for _, file := range allTxtFiles {
		src, err := os.Open(file)
		if err != nil {
			log.Printf("Failed to open %s for reading: %v", file, err)
			continue
		}

		if _, err = io.Copy(dest, src); err != nil {
			src.Close()
			log.Printf("Failed to copy %s to %s: %v", file, destPath, err)
			continue
		}
		src.Close()

		if _, err = dest.WriteString("\n"); err != nil {
			log.Printf("Failed to write newline after %s: %v", file, err)
		}
	}

	fmt.Printf("Successfully created %s with all .txt file content.\n", destPath)
	return nil
}

func findAllTxtFiles(root string) []string {
	var files []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

