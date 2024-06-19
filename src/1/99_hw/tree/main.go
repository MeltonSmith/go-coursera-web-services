package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func startStringFunc(isLast bool) string {
	if isLast {
		return "└───"
	}
	return "├───"
}

func specialStringFunc(isLast bool) string {
	if isLast {
		return "|"
	}
	return "<"
}

func sizeFuncStr(size int64) string {
	if size != 0 {
		return strconv.FormatInt(size, 10) + "b"
	}
	return "empty"
}

func dirTree(out io.Writer, path string, files bool) error {
	filterLambda := func(r rune) bool {
		return r == '<' || r == '|'
	}

	res, runes := replaceSpecialChars(path, filterLambda, filepath.Separator)

	file, err := os.Open(res)
	if err != nil {
		log.Fatalf("Failed to read the file %s: %v", path, err)
	}

	fileSlice, err := file.Readdir(-1)
	if !files {
		var buf []os.FileInfo
		for _, s := range fileSlice {
			if s.IsDir() {
				buf = append(buf, s)
			}
		}
		fileSlice = buf
	}

	if err == nil {

		sort.Slice(fileSlice, func(i, j int) bool {
			return fileSlice[i].Name() < fileSlice[j].Name()
		})
		for idx, value := range fileSlice {

			if value.IsDir() {
				fmt.Fprintln(out, formIndents(runes)+startStringFunc(idx == len(fileSlice)-1)+value.Name())
				//out.WriteString(formIndents(runes) + startStringFunc(idx == len(fileSlice)-1) + value.Name() + "\n")
				dirTree(out, path+specialStringFunc(idx == len(fileSlice)-1)+value.Name(), files)
			} else if files {
				fmt.Fprintln(out, formIndents(runes)+startStringFunc(idx == len(fileSlice)-1)+value.Name()+" ("+sizeFuncStr(value.Size())+")")
				//out.WriteString(formIndents(runes) + startStringFunc(idx == len(fileSlice)-1) + value.Name() + " (" + sizeFuncStr(value.Size()) + ") \n")
			}

		}
	}
	return err
}

func formIndents(runes []rune) string {
	result := []string{}
	for _, ru := range runes {
		if ru == '|' {
			result = append(result, "\t")
		} else {
			result = append(result, "│\t")
		}
	}
	return strings.Join(result, "")
}

func replaceSpecialChars(source string, f func(rune) bool, replacer rune) (string, []rune) {
	result := []rune(source)
	specialRunes := make([]rune, 0)
	for idx, value := range source {
		if f(value) {
			result[idx] = replacer
			specialRunes = append(specialRunes, value)
		}

	}
	return string(result), specialRunes
}
