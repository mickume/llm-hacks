package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocolly/colly"
)

const (
	minLineLength = 20
	newLineToken  = "\n\n"

	startToken = ""
	endToken   = "\n"
)

var (
	stopWords = []string{
		"notes:",
		"summary:",
		"chapter text",
		"disclaimer:",
		"disclaimers:",
		"https://",
		"http://",
		"****",
		"....",
		". . .",
		"—--",
		"author note",
	}

	quote1 byte = '\''
)

func clean(s string) (string, int, bool) {
	line := strings.Trim(s, " ")

	if len(line) < minLineLength {
		return "", 0, true
	}

	checks := strings.ToLower(line)

	for _, word := range stopWords {
		if strings.Contains(checks, word) {
			return "", 0, true
		}
	}

	if line[0] == quote1 {
		line = "\"" + line[1:]
	}

	line = strings.ReplaceAll(line, "***", "")
	line = strings.ReplaceAll(line, "__", "")
	line = strings.ReplaceAll(line, "~*~", "")
	line = strings.ReplaceAll(line, "''", "\" ")
	line = strings.ReplaceAll(line, "‘", "\"")
	line = strings.ReplaceAll(line, "’ ", "\"")
	line = strings.ReplaceAll(line, "“", "\"")
	line = strings.ReplaceAll(line, "”", "\"")
	line = strings.ReplaceAll(line, "' ", "\" ")
	line = strings.ReplaceAll(line, " '", " \"")
	line = strings.ReplaceAll(line, ".'", ".\"")

	return line, len(line), false
}

func clean_rewrite(source, target string) (int, error) {
	n := 0

	reader, err := os.Open(source)
	if err != nil {
		return 0, err
	}

	dst, err := os.Create(target)
	if err != nil {
		return 0, err
	}
	writer := bufio.NewWriter(dst)

	defer func() {
		reader.Close()
		writer.Flush()
		dst.Close()
	}()

	scanner := bufio.NewScanner(reader)
	sentence := false

	for scanner.Scan() {
		line, l, skipped := clean(scanner.Text())
		if !skipped {
			if !sentence {
				writer.WriteString(startToken) // <|startoftext|>
				sentence = true
			}
			writer.WriteString(line)
			n = n + l
		} else {
			if sentence {
				writer.WriteString(fmt.Sprintf("%s\n", endToken)) // <|endoftext|>
				sentence = false
			}
		}
	}
	if sentence {
		writer.WriteString(endToken) // <|endoftext|>
	}

	return n, nil
}

func merge(path, name string) error {

	// create and open the merge file

	merge_file := fmt.Sprintf("%s/%s", path, name)
	out, err := os.OpenFile(merge_file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// scan the dir for files to merge into
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	num := 0
	var l int64
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".training.txt") {
			full_path := fmt.Sprintf("%s/%s", path, f.Name())

			merge, err := os.Open(full_path)
			if err != nil {
				log.Fatal(err)
			}
			defer merge.Close()

			n, err := io.Copy(out, merge)
			if err != nil {
				log.Fatal(err)
			}
			l = l + n
			num++
		}
	}

	fmt.Printf("Merged %d files into '%s'. Total length=%d characters.\n", num, merge_file, l)

	return nil
}

func fetch(id, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	c := colly.NewCollector()

	c.OnHTML("div.userstuff p", func(e *colly.HTMLElement) {
		f.WriteString(e.Text + newLineToken)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Retrieving ", r.URL)
	})

	url := fmt.Sprintf("https://archiveofourown.org/works/%s?view_full_work=true&view_adult=true", id)
	return c.Visit(url)
}

func process(id, path string) error {
	source := fmt.Sprintf("%s/%s.txt", path, id)
	output := fmt.Sprintf("%s/%s.training.txt", path, id)

	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		if err := fetch(id, source); err != nil {
			return err
		}
	}

	l, err := clean_rewrite(source, output)
	if err != nil {
		return err
	}

	fmt.Printf("Retrieved '%s'. Length=%d characters.\n", output, l)

	return nil
}

// fetch [input.txt | story ID] output_dir
func main() {

	var id string
	var input string
	var training string
	var output string

	flag.StringVar(&id, "id", "", "Story ID to fetch")
	flag.StringVar(&input, "i", "input.txt", "File with Story IDs to fetch")
	flag.StringVar(&training, "o", "training.txt", "Name of the merged file")
	flag.StringVar(&output, "dir", ".", "Output dir")
	flag.Parse()

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if input == "input.txt" {
		input = filepath.Join(path, input)
	}
	if output == "." {
		output = filepath.Join(path, output)
	}

	if id == "" {
		file, err := os.Open(input)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// read id's from the input file and retrieve the texts
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			id := strings.TrimSpace(scanner.Text())
			if len(id) > 0 && !strings.HasPrefix(id, "#") {
				if err := process(id, output); err != nil {
					log.Fatal(err)
				}
			}
		}

		// merge all the texts
		if err := merge(output, training); err != nil {
			log.Fatal(err)
		}

	} else {
		if err := process(id, output); err != nil {
			log.Fatal(err)
		}
	}
}
