package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sort"
	"time"
)

type dataType map[string]map[string]uint

type rowType struct {
	addr, host string
}

var data = make(dataType)

const (
	dataFile    = "data.json"
	hostsFile   = "hosts"
	failedLimit = 10
)

func init() {
	cts, err := ioutil.ReadFile(dataFile)
	checkErr(err)
	err = json.Unmarshal(cts, &data)
	checkErr(err)
}

func main() {
	if len(os.Args) > 1 {
		buildHosts()
		return
	}
	hosts := [...]string{
		"cloud.google.com",
		"code.googlesource.com",
		"go.googlesource.com",
		"golang.org",
	}
	for _, host := range hosts {
		addrs, err := net.LookupHost(host)
		if err != nil {
			log.Println(host, err)
		}
		for _, addr := range addrs {
			if _, ok := data[host]; !ok {
				data[host] = make(map[string]uint)
			}
			if len(addr) < 16 {
				data[host][addr] = data[host][addr]
				fmt.Print(rowFormat(addr, host))
			}
		}
	}
	saveData(data)
}

func buildHosts() {
	hostsData := make([]rowType, 0)
	for host, addrs := range data {
		for addr, f := range addrs {
			if ok(addr) {
				hostsData = append(hostsData, rowType{addr: addr, host: host})
			} else if f > failedLimit {
				delete(data[host], addr)
			} else {
				data[host][addr]++
			}
		}
	}
	saveData(data)
	sort.Slice(hostsData, func(i, j int) bool {
		if hostsData[i].host == hostsData[j].host {
			return hostsData[i].addr < hostsData[j].addr
		}
		return hostsData[i].host < hostsData[j].host
	})
	var buf bytes.Buffer
	for _, r := range hostsData {
		row := rowFormat(r.addr, r.host)
		fmt.Println(row)
		buf.WriteString(row)
	}
	err := ioutil.WriteFile(hostsFile, buf.Bytes(), 0664)
	checkErr(err)
}

func saveData(data dataType) {
	cts, err := json.MarshalIndent(data, "", "\t")
	checkErr(err)
	err = ioutil.WriteFile(dataFile, cts, 0664)
	checkErr(err)
	fmt.Println(string(cts))
}

func ok(ip string) bool {
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp4", ip+":80", timeout)
	if err != nil {
		log.Println(ip, err)
	} else {
		defer conn.Close()
	}
	return conn != nil
}

func rowFormat(addr, host string) string {
	return fmt.Sprintf("%-16s%s\n", addr, host)
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
