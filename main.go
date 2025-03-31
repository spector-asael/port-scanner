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
		// The 0th element represents the count.
		(*openPortFound)[0]++
		for i := range maxRetries {
			conn, err := dialer.Dial("tcp", addr.address)
			if err == nil {
				defer conn.Close()
				fmt.Printf("\nConnection to %s was successful\n", addr.address)
				success = true
				portNumber, _ := strconv.Atoi(addr.port)
				// Once an open port is found, it gets appended to the array slice.
				*openPortFound = append(*openPortFound, portNumber)

				buffer := make([]byte, 1024)
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				numberOfBytes, err := conn.Read(buffer)
				if err == nil && numberOfBytes > 0 {
					fmt.Printf("[Banner] %s: %s\n", addr.address, string(buffer[:numberOfBytes]))
				} else {
					fmt.Printf("[Banner] %s: No response\n", addr.address)
				}
				fmt.Println("")
				break
			}
			backoff := time.Duration(1<<i) * time.Second
			// fmt.Printf("Attempt %d to %s failed. Waiting %v...\n", i+1, addr.address, backoff)
			time.Sleep(backoff)
		}
		if !success {
			// fmt.Printf("Failed to connect to %s after %d attempts\n", addr.address, maxRetries)
		}
	}
}

func main() {
	startTime := time.Now() // Helps keep track of how long the scanning process takes.
	var wg sync.WaitGroup
	var openPortFound []int = []int{0}
	// An array slice. The 0th element keeps track of the count of elements.
	// Anything else appended to the slice is an open port found.

	// Initialization of flags
	tasks := make(chan Address, 100)
	target := flag.String("target", "", "IP address or hostname to scan (required)")
	startPort := flag.String("start-port", "1", "Enter a number from 0 to 65535")
	endPort := flag.String("end-port", "1024", "Enter a number from 0 to 65535")
	timeout := flag.String("timeout", "5", "Enter a timeout for each connection attempt (in seconds)")

	flag.Parse()

	// Error handling for all flags
	if *target == "" {
		fmt.Println("Error: -target flag is required")
		flag.Usage()
		os.Exit(1)
	}

	startPortNumber, err := strconv.Atoi(*startPort)

	if err != nil || startPortNumber < 0 || startPortNumber > 65535 {
		fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.", err)
		os.Exit(1)
	}

	lastPortNumber, err := strconv.Atoi(*endPort)

	if err != nil || lastPortNumber < 0 || lastPortNumber > 65535 {
		fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.", err)
		os.Exit(1)
	}

	timeoutNumber, err := strconv.Atoi(*timeout)

	if err != nil || timeoutNumber < 0 {
		fmt.Println("Error: Timeout must be a valid number.", err)
		os.Exit(1)
	}

	// Dialer uses a timeoutNumber given by the user.
	// Defaults to 5 if no value was provided
	dialer := net.Dialer{
		Timeout: time.Duration(timeoutNumber) * time.Second,
	}

	workers := 100

	for i := 0; i <= workers; i++ {
		wg.Add(1)
		go worker(&wg, tasks, dialer, &openPortFound)
		// Adjusted worker to take in the memory address of the openPortFound slice
	}

	// Since the amount of ports is ranged based
	// Set this value to the lastPortNumber entered by the user
	// Defaults to 1024 if no value was provided
	ports := lastPortNumber
	totalPorts := lastPortNumber - startPortNumber + 1
	processedPorts := 0

	// startPortNumber defaults to 1 if no port was found.
	for p := startPortNumber; p <= ports; p++ {
		port := strconv.Itoa(p)
		address := net.JoinHostPort(*target, port)
		tasks <- Address{port, address}

		processedPorts++
		fmt.Printf("\rScanning port %d/%d", processedPorts, totalPorts)
	}
	close(tasks)

	wg.Wait()

	// Once the scan finishes, calculate how much time it has been since the scanning started
	elapsedTime := time.Since(startTime)

	fmt.Println("Report summary.")
	fmt.Printf("Time elapsed: %.2fs\n", elapsedTime.Seconds())
	fmt.Printf("Total number of ports scanned: %d (Port %s - %s)\n", openPortFound[0], *startPort, *endPort)
	fmt.Print("Open ports found: [ ")
	for i := 1; i < len(openPortFound); i++ {
		fmt.Printf("%d ", openPortFound[i])
	}
	print("]\n")
}
