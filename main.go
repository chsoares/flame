package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"unicode"

	"golang.org/x/term"

	"github.com/chsoares/flame/internal"
	"github.com/chsoares/flame/internal/tui"
	"github.com/chsoares/flame/internal/ui"
)

// Config holds the application configuration
type Config struct {
	Port      int
	Host      string
	Interface string
	IP        string // Resolved IP (from interface or direct)
}

func main() {
	// Parse command-line flags
	// flag package is Go's standard way to handle CLI arguments
	config := parseFlags()

	// Setup logging - minimal output like Penelope
	log.SetFlags(0)

	// Initialize runtime configuration
	runtimeConfig, err := internal.InitRuntimeConfig(config.IP)
	if err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Config initialization failed: %v (using defaults)", err)))
	}
	internal.GlobalRuntimeConfig = runtimeConfig

	// Initialize listener with resolved IP
	l := internal.NewListener(config.Host, config.Port)
	l.SetListenerIP(config.IP) // Set the IP for payload generation

	// Start listening for connections
	if err := l.Start(); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to start listener: %v", err)))
		os.Exit(1)
	}

	// Setup signal handling - only for cleanup, not for exit
	// Exit is only via exit/quit/q commands
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM) // Only SIGTERM, not SIGINT (Ctrl+C)

	go func() {
		<-sigChan
		fmt.Println()
		// Stop HTTP server if running
		if internal.GlobalRuntimeConfig.BinbagEnabled {
			internal.GlobalRuntimeConfig.DisableBinbag()
		}
		internal.GlobalRuntimeConfig.CleanupBinbagTmp()
		// Stop listener
		if err := l.Stop(); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Error stopping listener: %v", err)))
		}
		fmt.Println(ui.Success("Goodbye!"))
		os.Exit(0)
	}()

	// Launch TUI
	manager := l.GetSessionManager()
	listenerAddr := fmt.Sprintf("%s:%d", config.IP, config.Port)
	if err := tui.Run(manager, listenerAddr); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("TUI error: %v", err)))
		os.Exit(1)
	}

	// Cleanup tmp_* files from binbag
	internal.GlobalRuntimeConfig.CleanupBinbagTmp()

	// Print exit banner
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	fmt.Println()
	fmt.Println(tui.RenderExitBanner(width))

	// Show session log path if this instance created logs
	if logDir := manager.GetSessionLogDir(); logDir != "" {
		// Pretty path: replace home with ~
		if home, err := os.UserHomeDir(); err == nil {
			if len(logDir) > len(home) && logDir[:len(home)] == home {
				logDir = "~" + logDir[len(home):]
			}
		}
		fmt.Printf("\n\033[2mSession logs saved to: \033[36m%s\033[0m\n", logDir)
	}
	fmt.Println()
}

// parseFlags parses command-line arguments
// Go convention: unexported functions start with lowercase
func parseFlags() *Config {
	config := &Config{}

	var interfaceFlag string
	var ipFlag string

	// flag.IntVar binds the flag to a variable
	flag.IntVar(&config.Port, "port", 4444, "Port to listen on")
	flag.IntVar(&config.Port, "p", 4444, "Port to listen on (shorthand)")

	flag.StringVar(&interfaceFlag, "interface", "", "Network interface to bind to (e.g., eth0, eno1)")
	flag.StringVar(&interfaceFlag, "i", "", "Network interface to bind to (shorthand)")

	flag.StringVar(&ipFlag, "ip", "", "IP address to bind to (alternative to -i)")

	// Custom usage message with Flame styling
	flag.Usage = func() {
		// Print banner first
		fmt.Println()
		fmt.Println(renderStartupSplash())

		// Error message
		fmt.Println(ui.Error("Either -i <interface> or -ip <address> is required"))
		fmt.Println()

		// Usage instructions without box
		fmt.Println(ui.CommandHelp("usage"))
		fmt.Println(ui.Command(fmt.Sprintf("  %s -i <interface> -p <port>", os.Args[0])))
		fmt.Println(ui.Command(fmt.Sprintf("  %s -ip <address> -p <port>", os.Args[0])))
		fmt.Println()
		fmt.Println(ui.CommandHelp("options"))
		fmt.Println(ui.Command("  -i, -interface <name>    Network interface to bind to (e.g., eth0, eno1)"))
		fmt.Println(ui.Command("  -ip <address>            IP address to bind to (alternative to -i)"))
		fmt.Println(ui.Command("  -p, -port <number>       Port to listen on (default: 4444)"))
		fmt.Println()

		// Available interfaces in box
		fmt.Println(internal.FormatInterfaceList())
	}

	flag.Parse()

	// Validate that either interface or IP is provided
	if interfaceFlag == "" && ipFlag == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Both flags provided - error
	if interfaceFlag != "" && ipFlag != "" {
		// Print banner first
		fmt.Println()
		fmt.Println(renderStartupSplash())
		fmt.Println(ui.Error("Cannot specify both -i and -ip flags"))
		fmt.Println(ui.Info("Use either -i <interface> or -ip <address>, not both"))
		os.Exit(1)
	}

	// Resolve IP from interface
	if interfaceFlag != "" {
		ip, err := internal.GetIPFromInterface(interfaceFlag)
		if err != nil {
			// Print banner first
			fmt.Println()
			fmt.Println(renderStartupSplash())
			fmt.Println(ui.Error(titleCaseFirst(fmt.Sprintf("%v", err))))
			fmt.Println(internal.FormatInterfaceList())
			os.Exit(1)
		}
		config.IP = ip
		config.Interface = interfaceFlag
		config.Host = ip // Bind to the specific interface IP
	} else {
		// Validate IP address
		if !internal.IsValidIP(ipFlag) {
			// Print banner first
			fmt.Println()
			fmt.Println(renderStartupSplash())
			fmt.Println(ui.Error(fmt.Sprintf("Invalid IP address: %s", ipFlag)))
			os.Exit(1)
		}
		config.IP = ipFlag
		config.Host = ipFlag // Bind to specific IP
	}

	return config
}

func renderStartupSplash() string {
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	return tui.RenderExitBanner(width)
}

func titleCaseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
