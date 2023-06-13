package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/mickume/llm_hacks/internal"
)

const (
	minLineLength = 20

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

func merge(path string) error {

	// create and open the merge file

	//merge_file := fmt.Sprintf("%s/input_%d.txt", path, stdlib.Now())
	merge_file := fmt.Sprintf("%s/input.txt", path)
	out, err := os.OpenFile(merge_file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// scan the dir for files to merge into
	files, err := ioutil.ReadDir(path)
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

func process(id, path string) error {
	source := fmt.Sprintf("%s/%s.txt", path, id)
	output := fmt.Sprintf("%s/%s.training.txt", path, id)

	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		if err := internal.RetrieveFromAO3(id, source); err != nil {
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

// aoc [input.txt | story ID] output_dir
func main() {
	if len(os.Args) != 3 {
		log.Fatal(fmt.Errorf("invalid arguments"))
	}

	input := os.Args[1]
	path := os.Args[2]

	if strings.HasSuffix(input, ".txt") {
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
				if err := process(id, path); err != nil {
					log.Fatal(err)
				}
			}
		}

		// merge all the texts
		if err := merge(path); err != nil {
			log.Fatal(err)
		}

	} else {
		if err := process(input, path); err != nil {
			log.Fatal(err)
		}
	}
}
