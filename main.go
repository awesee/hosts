package main

import (
	"fmt"
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
	for _, host := range hosts {
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.Println(host, err)
		}
		for _, addr := range addrs {
			if len(addr) < 16 {
				fmt.Println(addr, "\t", host)
			}
		}
	}
}
