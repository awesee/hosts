package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
)

func main() {
	hosts := []string{
		"cloud.google.com",
		"code.googlesource.com",
		"go.googlesource.com",
		"golang.org",
	}
	sort.Strings(hosts)
	var buf bytes.Buffer
	for _, host := range hosts {
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.Println(host, err)
		}
		for _, addr := range addrs {
			if len(addr) < 16 {
				row := fmt.Sprintf("%-16s%s\n", addr, host)
				fmt.Print(row)
				buf.WriteString(row)
			}
		}
	}
	username := os.Getenv("username")
	password := os.Getenv("password")
	err := ioutil.WriteFile("hosts", buf.Bytes(), 0664)
	checkErr(err)
	err = exec.Command("git", "config", "user.name", username).Run()
	checkErr(err)
	err = exec.Command("git", "config", "user.email", "openset.wang@gmail.com").Run()
	checkErr(err)
	err = exec.Command("git", "config", "remote.origin.url", fmt.Sprintf("https://%s:%s@github.com/openset/hosts.git", username, password)).Run()
	checkErr(err)
	err = exec.Command("git", "commit", "-am", "weekly update").Run()
	checkErr(err)
	err = exec.Command("git", "push", "origin", "master").Run()
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
