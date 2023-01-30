package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/armon/go-socks5"
	"github.com/ryboe/trigger-captive-portal/routerdns"
)

// The SOCKS5 proxy server will listen on this address.
const proxyAddr = "localhost:1666"

// This will be set to something like 'v1.2.3' at build time.
var Version = ""

// The captive portal is triggered when you make an unsecured HTTP request,
// probably using a web browser. It has to be an unsecured HTTP request, because
// the router is going to ignore the requested URL and redirect you to the
// captive portal login. HTTPS requests can't be redirected like this. Only HTTP
// requests can.
//
// Apple provides a dedicated URL for safely making unsecured HTTP requests for
// the purpose of triggering captive portals:
//
//	http://captive.apple.com
//
// The process starts with a DNS request for apple.com. This request MUST go to
// the router's DNS server. The DNS request won't be forwarded to the router if
// the browser or OS DNS resolver is used. Other DNS resolvers maintain their
// own cache, so a request might not get generated at all. We'll force the use
// of the router's DNS resolver by by creating a SOCKS5 proxy that forwards all
// requests to the router's DNS server. The proxy will run in a background
// goroutine. Then we'll set the PROXY env var to the address the proxy server
// is listening on. Finally, we'll open Chrome in incognito mode (which disables
// DNS caching) and make an unsecured HTTP request to http://captive.apple.com.
//
// The router *should* then redirect the browser to the captive portal login
// page.
func main() {
	log.SetFlags(0) // Don't print timestamps before each log message.
	err := parseArgs()
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		os.Exit(1)
	}

	chromeApp, err := getChromeAppName() // "Google Chrome" or "Chromium" or err if neither is installed
	if err != nil {
		log.Fatalf("Error checking if Chrome is installed:\n%v", err)
	}
	log.Println("Chrome is installed.")

	// Get the name of the Wi-Fi interface, e.g. 'en0'.
	log.Println("Getting Wi-Fi network interface...")
	wifiIface, err := getWiFINetworkInterface()
	if err != nil {
		log.Fatalf("Error getting Wi-Fi network interface:\n%v", err)
	}
	log.Printf("The Wi-Fi interface is %s", wifiIface)

	// Get the router's IP address so we can send our DNS requests to it. We
	// don't want to use DNS cache, a browser's built in DNS resolver, or any
	// other DNS server. Only making a DNS request to the router's DNS server
	// will trigger the captive portal.
	log.Println("Getting router's DNS IP...")
	routerIP, err := getRouterIP(wifiIface)
	if err != nil {
		log.Fatalf("Error getting router's DNS IP:\n%v", err)
	}
	log.Printf("The router DNS server's IP is %s", routerIP)

	// Create a proxy server that forwards all DNS requests to the router's
	// DNS server.
	log.Printf("Starting SOCKS5 proxy server pointing to DNS server at %s", routerIP)
	log.Printf("Listening on %s", proxyAddr)
	err = startProxyServer(routerIP)
	if err != nil {
		log.Fatalf("Error starting SOCKS5 server:\n%v", err)
	}

	// Open Chrome in incognito mode and make an unsecured HTTP request to
	// http://captive.apple.com.
	err = makeUnsecuredHTTPRequestWithChrome(chromeApp)
	if err != nil {
		log.Fatalf("Error making unsecured HTTP request with Chrome:\n%v", err)
	}
}

// parseArgs reads command line arguments and returns an error if they are
// anything but the help flag. trigger-captive-portal doesn't take any args or
// flags. Regardless, every CLI should support the help flag.
func parseArgs() error {
	if len(os.Args) >= 2 {
		printUsage()

		for _, arg := range os.Args[1:] {
			if arg == "-h" || arg == "--help" {
				return flag.ErrHelp
			}
		}
		return fmt.Errorf("invalid arguments")
	}

	return nil
}

// printUsage prints the help message.
func printUsage() {
	programName := filepath.Base(os.Args[0])
	fmt.Printf(`
%[1]s %[2]s
Trigger the captive portal on a public Wi-Fi network.

USAGE:
    %[1]s

OPTIONS:
    -h, --help
        Print help information

`[1:], programName, Version)
}

// getChromeAppName returns the name of the installed Chrome app, which will be
// `Google Chrome` or `Chromium`. It returns an error if neither is installed.
func getChromeAppName() (string, error) {
	for _, app := range []string{"Google Chrome", "Chromium"} {
		installed, err := isInstalled(app)
		if err != nil {
			return "", err
		}
		if installed {
			return app, nil
		}
	}

	return "", fmt.Errorf("neither Chrome nor Chromium is installed")
}

// isInstalled returns true if an app with the given name is installed. It
// searches for the app with Spotlight, so if the app isn't indexed by
// Spotlight, it won't be found.
func isInstalled(app string) (bool, error) {
	path, err := runCmd("mdfind", "-name", app)
	if err != nil {
		return false, err
	}
	return path != "", nil
}

// getWiFINetworkInterface returns the name of the Wi-Fi network interface, e.g.
// 'en0'. It returns an error if the Wi-Fi network interface is not found. It
// gets the interface by running `system_profiler SPNetworkDataType`.
func getWiFINetworkInterface() (wifi string, err error) {
	// The output of `system_profiler -json -timeout 10 SPNetworkDataType` looks like this:
	// {
	// 	"SPNetworkDataType": [
	// 		{
	// 			"_name": "Wi-Fi",
	// 			"interface": "en0",
	// 			...
	// 		},
	// 		...
	// 	]
	// }
	type SPNetworkDataType struct {
		SPNetworkDataType []struct {
			Name      string `json:"_name"`
			Interface string `json:"interface"`
		}
	}

	stdout, err := runCmd("system_profiler", "-json", "-timeout", "5", "SPNetworkDataType")
	if err != nil {
		return "", err
	}

	data := SPNetworkDataType{}
	err = json.Unmarshal([]byte(stdout), &data)
	if err != nil {
		return "", err
	}

	for _, iface := range data.SPNetworkDataType {
		if iface.Name == "Wi-Fi" {
			return iface.Interface, nil
		}
	}

	return "", fmt.Errorf("Wi-Fi network interface not found")
}

// runCmd runs a shell command with a 5s timeout and returns its stdout.
func runCmd(cmd string, args ...string) (stdout string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, cmd, args...).Output()
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(output)), nil
}

// getRouterIP returns the IP of the router's DNS server. It gets the
// IP by running `ipconfig getoption <iface> domain_name_server`.
func getRouterIP(iface string) (ip string, err error) {
	ip, err = runCmd("ipconfig", "getoption", iface, "domain_name_server")
	if err != nil {
		return "", err
	}

	validIP := net.ParseIP(ip)
	if validIP == nil {
		return "", fmt.Errorf(`The command 'ipconfig getoption %s domain_name_server' returned
'%s' for the IP of router's DNS server, but that's not a valid IP`, iface, ip)
	}

	return ip, nil
}

// startProxyServer starts a SOCKS5 proxy server that forwards traffic to the
// given IP.
func startProxyServer(ip string) error {
	dialer := &net.Dialer{}
	routerDNSResolver := routerdns.NewResolver(ip, dialer)

	cfg := socks5.Config{
		Resolver: routerDNSResolver,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
	}

	srv, err := socks5.New(&cfg)
	if err != nil {
		return err
	}

	go func() {
		if err := srv.ListenAndServe("tcp", proxyAddr); err != nil {
			log.Fatalf("Error starting SOCKS5 proxy server:\n%v", err)
		}
	}()

	return nil
}

// makeUnsecuredHTTPRequestWithChrome opens Chrome and makes a request to
// http://captive.apple.com.
func makeUnsecuredHTTPRequestWithChrome(chromeApp string) error {
	// Command taken from captive-browser here:
	//   https://github.com/FiloSottile/captive-browser/blob/main/captive-browser-mac-chrome.toml
	//
	// --wait-apps    block until Chrome is closed
	// --new          open a new Chrome instance even if Chrome is already running
	// --background   don't bring this new Chrome to the foreground
	openChromeCmd := `
		open --new \
		  --wait-apps \
		  -a "Google Chrome" \
		  --background \
		  --args \
		    --user-data-dir="$HOME/Library/Application Support/Google/Captive" \
		    --proxy-server="socks5://$PROXY" \
		    --host-resolver-rules="MAP * ~NOTFOUND , EXCLUDE localhost" \
		    --no-first-run \
			--new-window \
			--incognito \
		http://captive.apple.com
`

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", openChromeCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PROXY="+proxyAddr)

	return cmd.Run()
}
