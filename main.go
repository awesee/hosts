package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
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
	_ = ioutil.WriteFile("hosts", buf.Bytes(), 0664)
}
