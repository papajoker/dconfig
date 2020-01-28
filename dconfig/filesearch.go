package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

/*
 * Find Override Configuration in /xxx.d/* directories
 */

type fileToSearch struct {
	filename string
	pkg      bool
}

type filesSearch struct {
	files []fileToSearch
}

func (self *filesSearch) dbFindFiles() {
	matches, _ := filepath.Glob("/var/lib/pacman/local/*/files")
	var wg sync.WaitGroup
	wg.Add(len(matches))
	for _, m := range matches {
		go func(m string) {
			f, _ := os.Open(m)
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				for k, v := range self.files {
					if v.pkg == false && strings.HasPrefix(scanner.Text(), v.filename) {
						self.files[k].pkg = true
					}
				}
			}
			wg.Done()
			f.Close()
		}(m)
	}
	wg.Wait()
}

func (self filesSearch) get() (ret []string) {
	for _, v := range self.files {
		if !v.pkg {
			ret = append(ret, v.filename)
		}
	}
	return
}

func FindOverrideConf() {
	listFiles := filesSearch{}
	filepath.Walk("/etc",
		func(path string, info os.FileInfo, err error) error {
			if strings.Contains(path, "etc/fonts/conf.d/") ||
				strings.Contains(path, "etc/pacman.d/") ||
				strings.HasSuffix(path, "~") {
				return nil
			}
			if err != nil || !strings.Contains(path, ".d/") {
				return nil
			}
			listFiles.files = append(listFiles.files, fileToSearch{path[1:], false})
			return nil
		})

	println("")
	listFiles.dbFindFiles()
	list := listFiles.get()
	for _, v := range list {
		fmt.Println(v)
	}

	fmt.Printf("\n:: %v/%v Override configuration files in .d/ and NOT in pacman db\n", len(list), len(listFiles.files))
}
