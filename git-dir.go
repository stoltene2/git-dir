package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type dir struct {
	name string
	path string
}

func (dir *dir) isGitRepo() bool {
	gitDir := filepath.Join(dir.path, ".git")
	_, err := toDir(gitDir)
	return err == nil
}

func walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return filepath.SkipDir
	}

	if d, err := toDir(path); err == nil && d.isGitRepo() {
		fmt.Println(d.path)
		return filepath.SkipDir
	} else if err != nil || !info.IsDir() {
		return nil
	}

	return nil
}

func toDir(path string) (dir, error) {
	var d dir
	dirname, err := os.Open(path)
	defer dirname.Close()

	if err != nil {
		return d, err
	}

	stat, err := dirname.Stat()

	if !stat.IsDir() {
		return d, errors.New(fmt.Sprint(stat.Name(), " is not a directory"))
	}

	dirPath, err := filepath.Abs(dirname.Name())

	if err != nil {
		return d, err
	}

	d = dir{
		name: dirname.Name(),
		path: dirPath,
	}

	return d, nil
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("You must pass a directory argument")
	}

	dir, err := toDir(os.Args[1])

	if err != nil {
		log.Fatal("Error opening directory: ", err)
	} else {
		fmt.Println(dir)
	}

	if dir.isGitRepo() {
		fmt.Println("it's a git repo")
	} else {
		fmt.Println("not a git repo")
	}

	filepath.Walk(dir.path, walkFunc)
}
