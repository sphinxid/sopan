package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const (
	defaultTestURL = "https://kodelatte.com/"
	defaultTimeout = 5 * time.Second
)

// ProxyInfo holds proxy connection details
type ProxyInfo struct {
	Host     string
	Port     string
	Username string
	Password string
	Raw      string
}

// TestResult holds the result of a proxy test
type TestResult struct {
	Proxy   string
	Success bool
	Latency time.Duration
	Error   string
}

var (
	proxyFlag    = flag.String("proxy", "", "Single proxy to test (format: socks5://[user:pass@]host:port)")
	fileFlag     = flag.String("file", "", "File containing list of proxies (one per line)")
	timeoutFlag  = flag.Int("timeout", 5, "Timeout in seconds for each proxy test")
	threadsFlag  = flag.Int("threads", 10, "Number of concurrent threads")
	outputFlag   = flag.String("output", "", "Output file for successful proxies (optional)")
	verboseFlag  = flag.Bool("verbose", false, "Verbose output (show all results)")
	testURLFlag  = flag.String("url", defaultTestURL, "URL to test proxies against")
)

func main() {
	flag.Parse()

	if *proxyFlag == "" && *fileFlag == "" {
		fmt.Println("Error: Either -proxy or -file must be specified")
		flag.Usage()
		os.Exit(1)
	}

	timeout := time.Duration(*timeoutFlag) * time.Second
	
	var proxies []string
	
	// Load proxies
	if *proxyFlag != "" {
		proxies = []string{*proxyFlag}
	} else {
		var err error
		proxies, err = loadProxiesFromFile(*fileFlag)
		if err != nil {
			fmt.Printf("Error loading proxies from file: %v\n", err)
			os.Exit(1)
		}
	}

	if len(proxies) == 0 {
		fmt.Println("No proxies to test")
		os.Exit(1)
	}

	fmt.Printf("Testing %d proxies with %d threads (timeout: %v)\n", len(proxies), *threadsFlag, timeout)
	fmt.Printf("Test URL: %s\n", *testURLFlag)
	fmt.Println(strings.Repeat("-", 80))

	// Test proxies
	results := testProxies(proxies, *threadsFlag, timeout, *testURLFlag)

	// Display results
	displayResults(results)

	// Save successful proxies if output file specified
	if *outputFlag != "" {
		saveSuccessfulProxies(results, *outputFlag)
	}
}

// loadProxiesFromFile reads proxies from a file
func loadProxiesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			proxies = append(proxies, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return proxies, nil
}

// parseProxy parses a proxy string into ProxyInfo
func parseProxy(proxyStr string) (*ProxyInfo, error) {
	// Handle both with and without protocol prefix
	if !strings.HasPrefix(proxyStr, "socks5://") && !strings.HasPrefix(proxyStr, "socks4://") {
		proxyStr = "socks5://" + proxyStr
	}

	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy format: %v", err)
	}

	info := &ProxyInfo{
		Host: u.Hostname(),
		Port: u.Port(),
		Raw:  proxyStr,
	}

	if u.User != nil {
		info.Username = u.User.Username()
		info.Password, _ = u.User.Password()
	}

	if info.Host == "" || info.Port == "" {
		return nil, fmt.Errorf("invalid proxy format: missing host or port")
	}

	return info, nil
}

// testProxy tests a single proxy
func testProxy(proxyStr string, timeout time.Duration, testURL string) TestResult {
	result := TestResult{
		Proxy:   proxyStr,
		Success: false,
	}

	proxyInfo, err := parseProxy(proxyStr)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	start := time.Now()

	// Create SOCKS5 dialer
	var auth *proxy.Auth
	if proxyInfo.Username != "" {
		auth = &proxy.Auth{
			User:     proxyInfo.Username,
			Password: proxyInfo.Password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(proxyInfo.Host, proxyInfo.Port), auth, proxy.Direct)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create dialer: %v", err)
		return result
	}

	// Create HTTP client with SOCKS5 proxy
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make request
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 SOPAN/1.0")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Read response body (to ensure full connection)
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		return result
	}

	result.Latency = time.Since(start)
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400
	
	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// testProxies tests multiple proxies concurrently
func testProxies(proxies []string, threads int, timeout time.Duration, testURL string) []TestResult {
	var wg sync.WaitGroup
	resultsChan := make(chan TestResult, len(proxies))
	proxyChan := make(chan string, len(proxies))

	// Start worker goroutines
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for proxyStr := range proxyChan {
				result := testProxy(proxyStr, timeout, testURL)
				resultsChan <- result
			}
		}()
	}

	// Send proxies to workers
	for _, p := range proxies {
		proxyChan <- p
	}
	close(proxyChan)

	// Wait for all workers to finish
	wg.Wait()
	close(resultsChan)

	// Collect results
	var results []TestResult
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// displayResults displays test results
func displayResults(results []TestResult) {
	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
			if *verboseFlag {
				fmt.Printf("✓ [SUCCESS] %s (latency: %v)\n", result.Proxy, result.Latency)
			}
		} else {
			failCount++
			if *verboseFlag {
				fmt.Printf("✗ [FAILED]  %s - %s\n", result.Proxy, result.Error)
			}
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Results: %d tested | %d successful | %d failed\n", len(results), successCount, failCount)

	if !*verboseFlag && successCount > 0 {
		fmt.Println("\nSuccessful proxies:")
		for _, result := range results {
			if result.Success {
				fmt.Printf("  %s (latency: %v)\n", result.Proxy, result.Latency)
			}
		}
	}
}

// saveSuccessfulProxies saves successful proxies to a file
func saveSuccessfulProxies(results []TestResult, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	count := 0

	for _, result := range results {
		if result.Success {
			fmt.Fprintf(writer, "%s\n", result.Proxy)
			count++
		}
	}

	writer.Flush()
	fmt.Printf("\n%d successful proxies saved to %s\n", count, filename)
}
