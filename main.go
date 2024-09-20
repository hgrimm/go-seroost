package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
)

func parseEntireTxtFile(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("could not read file %s: %v", filePath, err)
	}
	return string(content), nil
}

func parseEntireXmlFile(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("could not read file %s: %v", filePath, err)
	}

	decoder := xml.NewDecoder(strings.NewReader(string(content)))
	var result strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", fmt.Errorf("error parsing XML file %s: %v", filePath, err)
		}

		switch t := token.(type) {
		case xml.CharData:
			result.WriteString(string(t))
			result.WriteString(" ")
		}
	}

	return result.String(), nil
}

func parseEntirePdfFile(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open PDF file %s: %v", filePath, err)
	}
	defer f.Close()

	var result strings.Builder

	totalPage := r.NumPage()
	for i := 1; i <= totalPage; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("could not extract text from page %d in %s: %v", i, filePath, err)
		}
		result.WriteString(text)
		result.WriteString(" ")
	}

	return result.String(), nil
}

func parseEntireFileByExtension(filePath string) (string, error) {
	extension := strings.ToLower(filepath.Ext(filePath))
	switch extension {
	case ".xhtml", ".xml":
		return parseEntireXmlFile(filePath)
	case ".txt", ".md":
		return parseEntireTxtFile(filePath)
	case ".pdf":
		return parseEntirePdfFile(filePath)
	default:
		return "", fmt.Errorf("unsupported file extension %s for file %s", extension, filePath)
	}
}

func saveModelAsJSON(model *Model, indexPath string) error {
	fmt.Printf("Saving %s...\n", indexPath)
	file, err := os.Create(indexPath)
	if err != nil {
		return fmt.Errorf("could not create index file %s: %v", indexPath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(model); err != nil {
		return fmt.Errorf("could not serialize index into file %s: %v", indexPath, err)
	}

	return nil
}

func addFolderToModel(dirPath string, model *Model, processed *int) error {
	entries, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("could not open directory %s for indexing: %v", dirPath, err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		filePath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if err := addFolderToModel(filePath, model, processed); err != nil {
				return err
			}
			continue
		}

		lastModified := entry.ModTime()
		if model.RequiresReindexing(filePath, lastModified) {
			fmt.Printf("Indexing %s...\n", filePath)
			contentStr, err := parseEntireFileByExtension(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				continue
			}
			contentRunes := []rune(contentStr)
			model.AddDocument(filePath, lastModified, contentRunes)
			*processed++
		}
	}

	return nil
}

func usage(program string) {
	fmt.Fprintf(os.Stderr, "Usage: %s [SUBCOMMAND] [OPTIONS]\n", program)
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "    serve <folder> [address]       start local HTTP server with Web Interface\n")
}

func main() {
	args := os.Args
	if len(args) < 2 {
		usage(args[0])
		fmt.Fprintln(os.Stderr, "ERROR: no subcommand is provided")
		os.Exit(1)
	}

	subcommand := args[1]

	switch subcommand {
	case "serve":
		if len(args) < 3 {
			usage(args[0])
			fmt.Fprintf(os.Stderr, "ERROR: no directory is provided for %s subcommand\n", subcommand)
			os.Exit(1)
		}
		dirPath := args[2]

		indexPath := filepath.Join(dirPath, ".seroost.json")

		address := "127.0.0.1:6969"
		if len(args) > 3 {
			address = args[3]
		}

		model := NewModel()
		if _, err := os.Stat(indexPath); err == nil {
			file, err := os.Open(indexPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: could not open index file %s: %v\n", indexPath, err)
				os.Exit(1)
			}
			defer file.Close()
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&model); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: could not parse index file %s: %v\n", indexPath, err)
				os.Exit(1)
			}
		}

		modelMutex := &sync.Mutex{}
		go func() {
			processed := 0
			modelMutex.Lock()
			err := addFolderToModel(dirPath, model, &processed)
			modelMutex.Unlock()
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				return
			}
			if processed > 0 {
				if err := saveModelAsJSON(model, indexPath); err != nil {
					fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				}
			}
			fmt.Println("Finished indexing")
		}()

		startServer(address, model, modelMutex)

	default:
		usage(args[0])
		fmt.Fprintf(os.Stderr, "ERROR: unknown subcommand %s\n", subcommand)
		os.Exit(1)
	}
}
