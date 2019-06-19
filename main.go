package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	dataFile      = "data.json"
	hostsFile     = "hosts"
	parallelLimit = 1 << 7
	failedLimit   = 3
	randN         = 5
)

var (
	data   = make(dataType)
	tokens = make(chan bool, parallelLimit)
	status sync.Map
)

type dataType map[string]map[string]uint

type rowType struct {
	addr string
	host string
	ok   bool
}

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
	var wg sync.WaitGroup
	var mu sync.Mutex
	for host := range data {
		wg.Add(1)
		tokens <- true
		go func(host string) {
			addrs, err := net.LookupHost(host)
			if err != nil {
				log.Println(host, err)
			}
			for _, addr := range addrs {
				if strings.ContainsRune(addr, '.') {
					mu.Lock()
					data[host][addr] = data[host][addr]
					fmt.Print(rowFormat(addr, host))
					mu.Unlock()
				}
			}
			<-tokens
			wg.Done()
		}(host)
	}
	wg.Wait()
	saveData(data, true)
	autoPush()
}

func buildHosts() {
	var wg sync.WaitGroup
	hostsData := make([]rowType, 0)
	hostsCh := make(chan rowType, 16)
	go func() {
		for r := range hostsCh {
			if r.ok {
				data[r.host][r.addr] = 0
				hostsData = append(hostsData, r)
			} else if data[r.host][r.addr] >= failedLimit {
				delete(data[r.host], r.addr)
			} else {
				data[r.host][r.addr]++
			}
			wg.Done()
		}
	}()
	for host, addrs := range data {
		for addr := range addrs {
			wg.Add(2)
			tokens <- true
			go func(addr, host string) {
				hostsCh <- rowType{
					addr: addr,
					host: host,
					ok:   ok(addr),
				}
				<-tokens
				wg.Done()
			}(addr, host)
		}
	}
	wg.Wait()
	saveData(data, false)
	sort.Slice(hostsData, func(i, j int) bool {
		if hostsData[i].host == hostsData[j].host {
			return hostsData[i].addr < hostsData[j].addr
		}
		return reverse(hostsData[i].host) < reverse(hostsData[j].host)
	})
	var buf bytes.Buffer
	for _, r := range hostsData {
		row := rowFormat(r.addr, r.host)
		fmt.Print(row)
		buf.WriteString(row)
	}
	err := ioutil.WriteFile(hostsFile, buf.Bytes(), 0664)
	checkErr(err)
}

func saveData(data dataType, display bool) {
	cts, err := json.MarshalIndent(data, "", "\t")
	checkErr(err)
	err = ioutil.WriteFile(dataFile, cts, 0664)
	checkErr(err)
	if display {
		fmt.Println(string(cts))
	}
}

func ok(ip string) bool {
	if v, ok := status.Load(ip); ok {
		return v.(bool)
	}
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp4", ip+":80", timeout)
	if err != nil {
		log.Println(ip, err)
	} else {
		defer conn.Close()
	}
	status.Store(ip, conn != nil)
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

func autoPush() {
	password := os.Getenv("password")
	rand.Seed(time.Now().Unix())
	if password != "" && rand.Intn(randN) == 0 {
		err := exec.Command("git", "config", "user.name", "openset").Run()
		checkErr(err)
		err = exec.Command("git", "config", "user.email", "openset.wang@gmail.com").Run()
		checkErr(err)
		err = exec.Command("git", "config", "remote.origin.url", fmt.Sprintf("https://openset:%s@github.com/openset/hosts.git", password)).Run()
		checkErr(err)
		err = exec.Command("git", "stash").Run()
		checkErr(err)
		err = exec.Command("git", "checkout", "master").Run()
		checkErr(err)
		err = exec.Command("git", "stash", "pop").Run()
		checkErr(err)
		err = exec.Command("git", "commit", "-am", "daily update").Run()
		checkErr(err)
		err = exec.Command("git", "push", "origin", "master").Run()
		checkErr(err)
	}
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
