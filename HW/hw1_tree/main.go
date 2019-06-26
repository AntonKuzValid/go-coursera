package hw1_tree

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

const START_PADDING = "├───"
const START_LAST_PADDING = "└───"

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

func dirTree(out io.Writer, path string, printFiles bool) error {

	fileInfos, err := readDir(path, printFiles)
	if err != nil {
		return err
	}
	for ind, fi := range fileInfos {
		if err := printTree(out, "", filepath.Join(path, fi.Name()), fi, printFiles, len(fileInfos)-1 == ind); err != nil {
			return err
		}
	}
	return nil
}

func printTree(out io.Writer, padding string, root string, fi os.FileInfo, printFiles bool, islast bool) error {
	var printStr string
	if islast {
		printStr = padding + START_LAST_PADDING + fi.Name()
	} else {
		printStr = padding + START_PADDING + fi.Name()
	}
	if printFiles && !fi.IsDir() {
		if fi.Size() > 0 {
			printStr += fmt.Sprintf(" (%db)", fi.Size())
		} else {
			printStr += " (empty)"
		}
	}
	if _, err := fmt.Fprintln(out, printStr); err != nil {
		return err
	}
	if fi.IsDir() {
		fileInfos, err := readDir(root, printFiles)
		if err != nil {
			return err
		}
		for ind, val := range fileInfos {
			var newpadding string
			if islast {
				newpadding = padding + "	"
			} else {
				newpadding = padding + "│	"
			}
			if err := printTree(out, newpadding, filepath.Join(root, val.Name()), val, printFiles, len(fileInfos)-1 == ind); err != nil {
				return err
			}
		}
	}
	return nil
}

func readDir(path string, printFiles bool) ([]os.FileInfo, error) {
	fileInfos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	if !printFiles {
		fileInfos = filter(fileInfos, func(o1 os.FileInfo) bool {
			return o1.IsDir()
		})
	}

	return fileInfos, nil
}

func filter(arr []os.FileInfo, filter func(o1 os.FileInfo) bool) (res []os.FileInfo) {
	res = arr[:0]
	for _, x := range arr {
		if filter(x) {
			res = append(res, x)
		}
	}
	return
}
