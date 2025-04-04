// Filename: main.go

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PortScanResult struct {
	Target string `json:"target"`
	Port   int    `json:"port"`
	Status string `json:"status"`
	Banner string `json:"banner,omitempty"`
}

func worker(wg *sync.WaitGroup, tasks chan string, dialer net.Dialer, openPorts *[]PortScanResult, mu *sync.Mutex, totalPorts, scanned *int) {
	defer wg.Done()
	maxRetries := 3

	for addr := range tasks {
		var success bool
		var banner string
		parts := strings.Split(addr, ":")
		port, _ := strconv.Atoi(parts[1])
		target := parts[0]

		for i := 0; i < maxRetries; i++ {
			conn, err := dialer.Dial("tcp", addr)
			if err == nil {
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				buffer := make([]byte, 1024)
				n, err := conn.Read(buffer)
				if err != nil {
					fmt.Printf("Error reading from %s:%d: %v\n", target, port, err)
					banner = "" // Handle empty or error response
				}
				if n > 0 {
					banner = strings.TrimSpace(string(buffer[:n]))
					fmt.Printf(`Response from %s: %s\n`, addr, banner)
				} else {
					fmt.Printf(`No response from %s, bytes read: %d\n`, addr, n)
				}
				conn.Close()

				// Locking the mutex to safely append to the openPorts slice
				mu.Lock()
				*openPorts = append(*openPorts, PortScanResult{Target: target, Port: port, Status: "open", Banner: banner})
				mu.Unlock()

				fmt.Printf("\r[OPEN] %s:%d %s\n", target, port, banner)
				success = true
				break
			}

			backoff := time.Duration(1<<i) * time.Second
			time.Sleep(backoff)
		}

		if !success {
			fmt.Printf("\rFailed to connect to %s:%d\n", target, port)
		}

		// Locking the mutex for safely updating the 'scanned' counter
		mu.Lock()
		*scanned++
		fmt.Printf("\rScanning port %d/%d...", *scanned, *totalPorts)
		mu.Unlock()
	}
}

func main() {
	startTime := time.Now() // Helps keep track of how long the scanning process takes.
	var wg sync.WaitGroup
	var openPortFound []PortScanResult // Stores all open ports found
	var mu sync.Mutex                  // Mutex to safely update shared variables for the port scan

	// Initialization of flags
	tasks := make(chan string, 100)
	target := flag.String("target", "", "IP address or hostname to scan (required)")
	targets := flag.String("targets", "", "List of IP address or hostnames to scan using comma seperators")
	startPort := flag.String("start-port", "", "Enter a number from 0 to 65535")
	endPort := flag.String("end-port", "", "Enter a number from 0 to 65535")
	timeout := flag.String("timeout", "5", "Enter a timeout for each connection attempt (in seconds)")
	portsFlag := flag.String("ports", "", "Enter ports using commas as seperators")
	workersFlag := flag.String("workers", "100", "Enter the number of workers you'd like to use.")
	jsonFlag := flag.Bool("json", false, "Use this flag to ouput scan results in json format")

	flag.Parse()

	// Error handling for all flags
	if *target == "" && *targets == "" {
		fmt.Println("Error: -target flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Having a target flag is required, but you can only use either -target or -targets
	if *target != "" && *targets != "" {
		fmt.Println("Error: Cannot use both -target and -targets flags")
		os.Exit(1)
	}

	var targetList []string

	if *targets == "" {
		targetList = []string{*target}
	} else {
		targetList = strings.Split(*targets, ",")
	}

	// The default if no -port or -ports flag was provided.
	// The default is overrided if -ports was provided
	// This is to not force users to use the default 1-1024
	// if they don't use the -start-port and -endport flag
	if *portsFlag == "" {
		if *startPort == "" {
			*startPort = "1"
		}

		if *endPort == "" {
			*endPort = "1024"
		}
	}

	var portsList []string
	if *portsFlag != "" {
		portsList = strings.Split(*portsFlag, ",")
	}

	// Validation to check if ports are a valid number
	for _, p := range portsList {
		j, err := strconv.Atoi(p)
		if err != nil || j < 0 || j > 65535 {
			fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.", err)
			os.Exit(1)
		}
	}

	workers, err := strconv.Atoi(*workersFlag)

	if err != nil && workers <= 0 {
		fmt.Println("Error: Invalid number of works used.")
		os.Exit(1)
	}

	var startPortNumber, lastPortNumber int

	if *startPort != "" && *endPort != "" {
		startPortNumber, err = strconv.Atoi(*startPort)

		if err != nil || startPortNumber < 0 || startPortNumber > 65535 {
			fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.", err)
			os.Exit(1)
		}

		lastPortNumber, err = strconv.Atoi(*endPort)

		if err != nil || lastPortNumber < 0 || lastPortNumber > 65535 {
			fmt.Println("Error: Invalid port. Ports must be a number between 0 and 65535.", err)
			os.Exit(1)
		}
	} else {
		startPortNumber = 0
		lastPortNumber = 0
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

	totalPorts := (lastPortNumber - startPortNumber + 1) * len(targetList)
	scanned := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker(&wg, tasks, dialer, &openPortFound, &mu, &totalPorts, &scanned)
	}

	// Set the total number of ports to scan
	ports := lastPortNumber
	for _, target := range targetList {
		for p := startPortNumber; p <= ports; p++ {
			port := strconv.Itoa(p)
			address := net.JoinHostPort(target, port)
			tasks <- address // Send address as a string
		}

		for _, p := range portsList {
			address := net.JoinHostPort(target, p)
			tasks <- address // Send address as a string
		}
	}

	close(tasks)
	wg.Wait()

	// Once the scan finishes, calculate how much time it has been since the scanning started
	elapsedTime := time.Since(startTime)

	// If the user wants JSON output, print the results and exit
	if *jsonFlag {
		// Include the summary report in JSON format
		summary := struct {
			Elapsed      string           `json:"elapsed_time"`
			TotalScanned int              `json:"total_ports_scanned"`
			OpenPorts    []PortScanResult `json:"open_ports"`
		}{
			Elapsed:      fmt.Sprintf("%.2fs", elapsedTime.Seconds()),
			TotalScanned: totalPorts,
			OpenPorts:    openPortFound,
		}

		output, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Println("Error encoding JSON output:", err)
			os.Exit(1)
		}

		fmt.Println(string(output))
		return
	}

	fmt.Println("\nReport summary.")
	fmt.Printf("Time elapsed: %.2fs\n", elapsedTime.Seconds())
	fmt.Printf("Total number of ports scanned: %d\n", totalPorts)
	fmt.Print("Open ports found: [ ")
	for i := 0; i < len(openPortFound); i++ {
		fmt.Printf("%s:%d ", openPortFound[i].Target, openPortFound[i].Port) // Fixed format to match PortScanResult
	}
	fmt.Println("]")
}
