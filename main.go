package main

import (
	"bufio"
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
	hostsTxt      = "hosts.txt"
	parallelLimit = 1 << 7
	failedLimit   = 3
	randN         = 5
)

var (
	data   = make(dataType)
	status sync.Map
	wg     sync.WaitGroup
)

type dataType map[string]map[string]int

type hostType struct {
	host  string
	addrs []string
	ok    bool
}

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
	cmdName := ""
	if len(os.Args) > 1 {
		cmdName = os.Args[1]
	}
	switch cmdName {
	case "build":
		buildHosts()
	case "import":
		importHosts()
	default:
		updateData()
	}
}

func updateData() {
	jobs := make(chan string, parallelLimit)
	results := make(chan hostType, parallelLimit)
	hostList := make([]string, 0)
	for host := range data {
		hostList = append(hostList, host)
	}
	for i := 0; i < parallelLimit; i++ {
		go func() {
			for host := range jobs {
				addrs, err := net.LookupHost(host)
				if err != nil {
					log.Println(host, err)
				}
				ok := err == nil || !strings.HasSuffix(err.Error(), "no such host")
				results <- hostType{host: host, addrs: addrs, ok: ok}
			}
		}()
	}
	go func() {
		for r := range results {
			if r.ok {
				for _, addr := range r.addrs {
					if strings.ContainsRune(addr, '.') {
						setData(r.host, addr)
						fmt.Print(rowFormat(addr, r.host))
					}
				}
			} else {
				delete(data, r.host)
			}
			wg.Done()
		}
	}()
	for _, host := range hostList {
		wg.Add(1)
		jobs <- host
	}
	wg.Wait()
	saveData(data)
	autoPush()
}

func buildHosts() {
	hostsData := make([]rowType, 0)
	jobs := make(chan rowType, parallelLimit)
	results := make(chan rowType, parallelLimit)
	for i := 0; i < parallelLimit; i++ {
		go func() {
			for j := range jobs {
				j.ok = ok(j.addr)
				results <- j
			}
		}()
	}
	go func() {
		for r := range results {
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
			wg.Add(1)
			jobs <- rowType{addr: addr, host: host}
		}
	}
	wg.Wait()
	saveData(data)
	sort.Slice(hostsData, func(i, j int) bool {
		if hostsData[i].host == hostsData[j].host {
			return hostsData[i].addr < hostsData[j].addr
		}
		return hostRev(hostsData[i].host) < hostRev(hostsData[j].host)
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

func importHosts() {
	fi, err := os.Stat(hostsTxt)
	checkErr(err)
	if !fi.Mode().IsRegular() {
		return
	}
	file, err := os.Open(hostsTxt)
	checkErr(err)
	defer file.Close()
	br := bufio.NewReader(file)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		if len(line) > 0 && line[0] != '#' {
			r := strings.Fields(line)
			if len(r) >= 2 {
				setData(r[1], r[0])
			}
		}
	}
	saveData(data)
}

func saveData(data dataType) {
	cts, err := json.MarshalIndent(data, "", "\t")
	checkErr(err)
	err = ioutil.WriteFile(dataFile, cts, 0664)
	checkErr(err)
}

func setData(host, addr string) {
	if _, ok := data[host]; !ok {
		data[host] = make(map[string]int)
	}
	data[host][addr] = data[host][addr]
}

func ok(ip string) bool {
	if v, ok := status.Load(ip); ok {
		return v.(bool)
	}
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp4", ip+":80", timeout)
	ok := conn != nil
	if err != nil {
		log.Println(ip, err)
	} else {
		_ = conn.Close()
	}
	status.Store(ip, ok)
	return ok
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

func hostRev(s string) string {
	ss := strings.Split(s, ".")
	for i, j := 0, len(ss)-1; i < j; i, j = i+1, j-1 {
		ss[i], ss[j] = ss[j], ss[i]
	}
	return strings.Join(ss, ".")
}
