package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"

	//"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
	"github.com/xi2/xz"
)

var GitBranch string
var Version string
var BuildDate string
var GitID string

var quietOption = false
var dOption = false

/*
 source: https://gist.github.com/indraniel/1a91458984179ab4cf80
*/
func ExtractTarGz(gzipStream io.Reader, ext string, filepacsave string) string {

	var uncompressedStream io.Reader
	var err error
	if ext == ".zst" {
		uncompressedStream, err = zstd.NewReader(gzipStream)
	}
	if ext == ".xz" {
		uncompressedStream, err = xz.NewReader(gzipStream, 0)
	}
	if ext == ".gz" {
		uncompressedStream, err = gzip.NewReader(gzipStream)
	}
	if err != nil {
		fmt.Println("ExtractTar: NewReader failed")
		return ""
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("\tERROR", err.Error())
			return ""
			//log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue //fmt.Println("\t:::dir:", header.Name)
		case tar.TypeReg:

			if filepacsave[1:] == header.Name {
				//fmt.Println("\t:::file:", header.Name)

				buf := new(bytes.Buffer)
				if nb, err := buf.ReadFrom(tarReader); err != nil {
					if err != io.EOF {
						nb = nb + 1
						return ""
						//fmt.Println("error", err.Error())
						//log.Fatalf("ExtractTarGz:  failed: %s", err.Error())
					}
				}
				//fmt.Println(string(buf.Bytes()))
				return string(buf.Bytes())
			}
		default:
			//fmt.Println("error def", header.Typeflag, header.Name)
		}
	}
	return ""
}

type Pacman struct {
	Stdout string
	Code   int
	Stderr error
}

func (self *Pacman) Run(cmd string) bool {
	cm := exec.Command("/usr/bin/pacman", cmd)
	out, err := cm.Output()
	if err != nil {
		self.Stderr = err
		self.Code = 1
	}
	self.Stdout = string(out[:])
	return self.Code == 0
}

type FilePacSave struct {
	Pkg      string
	Version  string
	Filename string
	Pkgfile  string
	Content  string
}

func (self Pacman) isValidFile(search string) bool {
	var excludes = [2]string{"/etc/gshadow", "/etc/passwd"}
	for _, value := range excludes {
		if value == search {
			return false
		}
	}
	return true
}

func (self *Pacman) GetModified() <-chan FilePacSave {
	ch := make(chan FilePacSave)
	go func() {
		defer close(ch)
		if self.Run("-Qii") {
			//fmt.Println(self.Stdout)
			pkgname := ""
			filename := ""
			version := ""
			scanner := bufio.NewScanner(strings.NewReader(self.Stdout))

			var Files []FilePacSave

			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Name") {
					s := strings.SplitN(line, ":", 2)
					pkgname = strings.TrimSpace(s[1])
					filename = ""
				}
				if strings.HasPrefix(line, "Version") {
					s := strings.SplitN(line, ":", 2)
					version = strings.TrimSpace(s[1])
				}
				if strings.HasPrefix(line, "MODIFIED") {
					s := strings.SplitN(line, "/", 2)
					filename = "/" + strings.TrimSpace(s[1])
					if self.isValidFile(filename) {
						v := FilePacSave{pkgname, version, filename, "", ""}
						Files = append(Files, v)
					}
				}
			}

			const cachepath = "/var/cache/pacman/pkg/"
			var wg sync.WaitGroup
			wg.Add(len(Files))
			for i, _ := range Files {
				go func(v *FilePacSave, ch chan FilePacSave) {
					filename := cachepath + v.Pkg + "-" + v.Version + "-*"
					matches, err := filepath.Glob(filename)
					if err == nil && len(matches) == 1 {
						v.Pkgfile = matches[0]
						//fmt.Println(v.Pkgfile, "\t scan ...")
						f, _ := os.Open(v.Pkgfile)
						v.Content = ExtractTarGz(f, filepath.Ext(v.Pkgfile), v.Filename)
						os.MkdirAll(filepath.Dir("/tmp/dconfig/"+v.Filename), os.ModePerm)
						tmp, _ := os.Create("/tmp/dconfig/" + v.Filename)
						tmp.WriteString(v.Content)
						tmp.Close()
						defer f.Close()
					}
					ch <- *v
					wg.Done()
					//
				}(&Files[i], ch)
			}
			wg.Wait()
			//fmt.Println("fin")
		} else {
			fmt.Println(self.Stderr)
		}
	}()
	return ch
}

func main() {
	fmt.Println("dconfig")
	if len(os.Args) > 1 && strings.ToUpper(os.Args[1]) == "-V" {
		fmt.Printf("\n%s Version: %v %v %v %v\n", filepath.Base(os.Args[0]), Version, GitID, GitBranch, BuildDate)
	}
	os.Setenv("LANG", "C")

	if len(os.Args) > 1 && strings.ToUpper(os.Args[1]) == "-Q" {
		quietOption = true
	}
	if len(os.Args) > 1 && strings.ToUpper(os.Args[1]) == "-D" {
		quietOption = true
		dOption = true
	}

	p := Pacman{}

	ch := p.GetModified()
	for v := range ch {
		if !quietOption {
			println("\n")
		}
		fmt.Println("\033[1m", v.Filename, "\033[0m\t", v.Pkg+"("+v.Version+")\t") //, v.Pkgfile, v.Content != "")
		if v.Content != "" && !quietOption {
			print("\033[1;34m")
			cmd := exec.Command("diff", "-dEiwZB", "/tmp/dconfig/"+v.Filename, v.Filename, "--color=auto", "--new-line-format=+ %L", "--old-line-format=", "--unchanged-line-format=")
			cmd.Stdout = os.Stdout
			cmd.Run()
			print("\033[0m")
			os.Remove("/tmp/dconfig/" + v.Filename)
		}
	}

	if dOption {
		findPointd()
	}

	//TODO remove /tmp/dconfig/
	os.Remove("/tmp/dconfig/")

}
