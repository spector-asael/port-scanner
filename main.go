// Filename: main.go
// Purpose: This program demonstrates how to create a TCP network connection using Go

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Address struct {
	port    string
	address string
}

func worker(wg *sync.WaitGroup, tasks chan Address, dialer net.Dialer, openPortFound *[]int) {
	defer wg.Done()
	maxRetries := 3
	for addr := range tasks {
		var success bool
		(*openPortFound)[0]++
		for i := range maxRetries {
			conn, err := dialer.Dial("tcp", addr.address)
			if err == nil {
				conn.Close()
				fmt.Printf("Connection to %s was successful\n", addr.address)
				success = true
				portNumber, _ := strconv.Atoi(addr.port)
				*openPortFound = append(*openPortFound, portNumber)
				break
			}
			backoff := time.Duration(1<<i) * time.Second
			fmt.Printf("Attempt %d to %s failed. Waiting %v...\n", i+1, addr.address, backoff)
			time.Sleep(backoff)
		}
		if !success {
			fmt.Printf("Failed to connect to %s after %d attempts\n", addr.address, maxRetries)
		}
	}
}

func main() {
	startTime := time.Now()
	var wg sync.WaitGroup
	var openPortFound []int = []int{0}

	tasks := make(chan Address, 100)

	target := flag.String("target", "", "IP address or hostname to scan (required)")

	startPort := flag.String("start-port", "1", "Enter a number from 0 to 65535")

	endPort := flag.String("end-port", "1024", "Enter a number from 0 to 65535")

	flag.Parse()

	if *target == "" {
		fmt.Println("Error: -target flag is required")
		flag.Usage()
		os.Exit(1)
	}

	startPortNumber, err := strconv.Atoi(*startPort)

	if err != nil || startPortNumber < 0 || startPortNumber > 65535 {
		fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.")
		os.Exit(1)
	}

	lastPortNumber, err := strconv.Atoi(*endPort)

	if err != nil || lastPortNumber < 0 || lastPortNumber > 65535 {
		fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.")
		os.Exit(1)
	}

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	workers := 100

	for i := 0; i <= workers; i++ {
		wg.Add(1)
		go worker(&wg, tasks, dialer, &openPortFound)
	}

	ports := lastPortNumber

	for p := startPortNumber; p <= ports; p++ {
		port := strconv.Itoa(p)
		address := net.JoinHostPort(*target, port)
		tasks <- Address{port, address}
	}
	close(tasks)
	wg.Wait()

	elapsedTime := time.Since(startTime)

	fmt.Println("Report summary.")
	fmt.Printf("Total time elapsed: %.2fs\n", elapsedTime.Seconds())
	fmt.Printf("Total number of ports scanned: %d (Port %s - %s)\n", openPortFound[0], *startPort, *endPort)
	fmt.Print("Open ports found: [ ")
	for i := 1; i < len(openPortFound); i++ {
		fmt.Printf("%d ", openPortFound[i])
	}
	print("]\n")
}
