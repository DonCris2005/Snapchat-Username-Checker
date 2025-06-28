package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type checkRequest struct {
	Usernames  []string `json:"usernames"`
	Goroutines int      `json:"goroutines"`
}

type checkResponse struct {
	Available   []string `json:"available"`
	Unavailable []string `json:"unavailable"`
}

func readTargets(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var targets []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t != "" {
			targets = append(targets, t)
		}
	}
	return targets, scanner.Err()
}

func main() {
	var targetsFile string
	var goroutines int
	flag.StringVar(&targetsFile, "targets", "targets.txt", "file with usernames")
	flag.IntVar(&goroutines, "goroutines", 100, "total goroutines")
	flag.Parse()

	targets, err := readTargets(targetsFile)
	if err != nil {
		fmt.Println("failed to read targets:", err)
		return
	}

	srvEnv := os.Getenv("SERVERS")
	if srvEnv == "" {
		fmt.Println("SERVERS environment variable not set")
		return
	}
	srvList := strings.Split(srvEnv, ",")
	var servers []string
	for _, s := range srvList {
		s = strings.TrimSpace(s)
		resp, err := http.Get(s + "/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			servers = append(servers, s)
		}
	}
	if len(servers) == 0 {
		fmt.Println("no servers available")
		return
	}

	fmt.Printf("Detected %d servers\n", len(servers))

	perServer := len(targets) / len(servers)
	gorPerServer := goroutines / len(servers)
	if gorPerServer == 0 {
		gorPerServer = 1
	}
	idx := 0
	var allAvailable []string

	for i, srv := range servers {
		start := idx
		end := idx + perServer
		if i == len(servers)-1 {
			end = len(targets)
		}
		part := targets[start:end]
		idx = end

		reqBody := checkRequest{Usernames: part, Goroutines: gorPerServer}
		data, _ := json.Marshal(reqBody)
		resp, err := http.Post(srv+"/check", "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Printf("server %s error: %v\n", srv, err)
			continue
		}
		var res checkResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			fmt.Printf("server %s invalid response\n", srv)
			continue
		}
		allAvailable = append(allAvailable, res.Available...)
	}

	fmt.Printf("Available usernames (%d):\n", len(allAvailable))
	for _, u := range allAvailable {
		fmt.Println(u)
	}
}
