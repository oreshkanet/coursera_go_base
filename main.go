package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

func main() {
	//out := os.Stdout
	out := new(bytes.Buffer)
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
	fmt.Print(out)
}

func dirTree(out *bytes.Buffer, path string, printFiles bool) (err error) {
	// Читаем содержимое текущей директории
	return dirTreeRecursive(out, path, printFiles, 0, "")
}

func dirTreeRecursive(out *bytes.Buffer, path string, printFiles bool, level int, prefix string) (err error) {
	// Читаем содержимое текущей директории
	pathFiles, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	sort.Slice(pathFiles, func(i, j int) bool { return pathFiles[i].Name() < pathFiles[j].Name() })

	var maxDir int
	for index, val := range pathFiles {
		if val.IsDir() {
			maxDir = index
		}
	}

	for index, val := range pathFiles {

		var curRow string
		var isLast bool
		if printFiles {
			isLast = index == len(pathFiles)-1
		} else {
			isLast = index == maxDir
		}
		if isLast {
			curRow = prefix + "└───"
		} else {
			curRow = prefix + "├───"
		}
		if val.IsDir() {
			curRow = curRow + val.Name() + "\n"
			_, err = out.WriteString(curRow)

			var newPrefix string
			if isLast {
				newPrefix = prefix + "\t"
			} else {
				newPrefix = prefix + "│\t"
			}

			if err = dirTreeRecursive(out, path+string(os.PathSeparator)+val.Name(), printFiles, level+1, newPrefix); err != nil {
				return err
			}
		} else {
			if !printFiles {
				continue
			}
			var bytes string
			if val.Size() == 0 {
				bytes = "empty"
			} else {
				bytes = strconv.FormatInt(val.Size(), 10) + "b"
			}
			curRow = curRow + val.Name() + " (" + bytes + ")\n"
			_, err = out.WriteString(curRow)
		}
	}

	return nil
}
