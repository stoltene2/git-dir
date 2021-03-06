package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"bufio"
	"strings"

	fs "github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	gitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
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

func walkFunc(gitDirs chan<- string) func(string, os.FileInfo, error) error {

	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		if d, err := toDir(path); err == nil && d.isGitRepo() {
			gitDirs <- d.path
			return filepath.SkipDir
		} else if err != nil || !info.IsDir() {
			return nil
		}

		return nil
	}
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

// readExcludeFile assumes that path is the root directory of a git
// repository. If a .git/info/exclude file is included then it is used as additional patterns
func readExcludeFile(path string) (ps []gitignore.Pattern, err error) {
	const commentPrefix = "#"
	f := fs.New(path)

	file, err := f.Open(f.Join(append([]string{".git", "info"}, "exclude")...))

	if err == nil {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			s := scanner.Text()
			if !strings.HasPrefix(s, commentPrefix) && len(strings.TrimSpace(s)) > 0 {
				ps = append(ps, gitignore.ParsePattern(s, []string{}))
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return
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

	gitDirs := make(chan string)

	go func() {
		filepath.Walk(dir.path, walkFunc(gitDirs))
		close(gitDirs)
	}()

	var wg sync.WaitGroup

	for dir := range gitDirs {
		wg.Add(1)

		go func(d string) {
			repo, err := git.PlainOpen(d)

			if err != nil {
				fmt.Println("error:", err)
				wg.Done()
				return
			}

			// Check to see if there is an exclude file to load up
			wt, err := repo.Worktree()

			if err != nil {
				fmt.Println("error:", err)
				wg.Done()
				return
			}


			patterns, patErr := readExcludeFile(d)

			if patErr == nil {
				wt.Excludes = append(wt.Excludes, patterns...)
			}

			st, err := wt.Status()

			if err != nil {
				fmt.Println("error:", err)
				wg.Done()
				return

			}

			if !st.IsClean() {
				fmt.Println("Dirty:", d)
			} else {
				fmt.Println("Clean:", d)
			}
			wg.Done()
		}(dir)
	}

	wg.Wait()
}
