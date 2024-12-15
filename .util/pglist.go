package main

// Utility to generate `git diff` commands for the Postgres source from
// comments that contain GitHub URLs. Use it on in a Postgres Git clone to
// compare changes since the last time comments were updated.
//
// go run .util/pglist.go

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func main() {
	goRegex := regexp.MustCompile(`[.]go`)
	//  https://github.com/postgres/postgres/blob/REL_17_2/src/test/regress/sql/jsonpath.sql#L52-L64
	pgRegex := regexp.MustCompile(`postgres/postgres/blob/([^/]+)/([^#]+)`)

	found := map[string][]string{}
	logErr(filepath.WalkDir("path", func(path string, info fs.DirEntry, err error) error {
		if err == nil && goRegex.MatchString(info.Name()) {
			file, err := os.Open(path)
			logErr(err)
			defer file.Close()
			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if err == io.EOF {
					break
				}
				logErr(err)
				if match := pgRegex.FindStringSubmatch(line); match != nil {
					if list, ok := found[match[1]]; ok {
						if !slices.Contains(list, match[2]) {
							found[match[1]] = append(list, strings.TrimSpace(match[2]))
						}
					} else {
						found[match[1]] = []string{strings.TrimSpace(match[2])}
					}
				}
			}
		}
		return nil
	}))

	fmt.Println("# Clone the next release tag from the Postgres repo and run these diffs:")
	for tag, files := range found {
		for _, f := range files {
			fmt.Printf("git diff %v -- %v\n", tag, f)
		}
	}
}

func logErr(err error) {
	if err != nil {
		panic(err)
	}
}
