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

func usage() {
	progName := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `Chaos Data Downloader - Download and process company data from Project Discovery

USAGE:
    %s [OPTIONS]

OPTIONS:
    -c string    Comma-separated list of company names to download
                 Example: -c "Tesla,Google,Microsoft"
    -i string    Path to file containing company names (one per line)
    -a           Download all available companies
    -h           Show this help message

EXAMPLES:
    # Download specific companies
    %s -c Tesla
    %s -c "Tesla,Google,Microsoft"
    
    # Download companies from file
    %s -i companies.txt
    
    # Download all available companies
    %s -a

DESCRIPTION:
    This tool downloads chaos data from Project Discovery for specified companies.
    Downloaded data is extracted to ./AllChaosData/ and all .txt files are 
    concatenated into everything.txt in the current directory.

`, progName, progName, progName, progName, progName)
}

func main() {
	inputFile := flag.String("i", "", "Path to file containing company names (one per line)")
	companies := flag.String("c", "", "Comma-separated list of company names to download (e.g., 'Tesla,Google,Microsoft')")
	all := flag.Bool("a", false, "Download all available companies")
	help := flag.Bool("h", false, "Show usage information")
	flag.Usage = usage
	flag.Parse()

	// Show usage if no arguments provided or help requested
	if *help || (*inputFile == "" && *companies == "" && !*all) {
		flag.Usage()
		return
	}

	baseDir := filepath.Join(".", "AllChaosData")
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	var companySet map[string]struct{}
	var err error

	if *companies != "" {
		companySet = parseCompanyNames(*companies)
		fmt.Printf("Selected companies: %v\n", getCompanyNames(companySet))
	} else if *inputFile != "" {
		companySet, err = readCompanyList(*inputFile)
		if err != nil {
			log.Fatalf("Failed to read input file: %v", err)
		}
	} else if *all {
		// companySet remains nil to download all companies
		fmt.Println("Downloading all available companies...")
	}

	jsonURL := "https://chaos-data.projectdiscovery.io/index.json"
	if err := processURLs(jsonURL, baseDir, companySet); err != nil {
		log.Fatalf("Failed to process URLs: %v", err)
	}

	if err := concatenateAllTxtFiles(baseDir, "."); err != nil {
		log.Fatalf("Failed to concatenate all txt files: %v", err)
	}
}

func parseCompanyNames(input string) map[string]struct{} {
	companies := make(map[string]struct{})
	names := strings.Split(input, ",")

	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			companies[strings.ToLower(trimmed)] = struct{}{}
		}
	}

	return companies
}

func getCompanyNames(companySet map[string]struct{}) []string {
	var names []string
	for name := range companySet {
		names = append(names, name)
	}
	return names
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

	processedCount := 0
	for _, entry := range entries {
		if filterSet != nil {
			if _, ok := filterSet[strings.ToLower(entry.Name)]; !ok {
				continue
			}
		}
		fmt.Printf("Processing %s...\n", entry.Name)
		if err := downloadAndUnzip(entry.URL, entry.Name, baseDir); err != nil {
			log.Printf("Failed to process %s: %v\n", entry.Name, err)
		} else {
			processedCount++
		}
	}

	fmt.Printf("\nCompleted processing %d companies.\n", processedCount)
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
