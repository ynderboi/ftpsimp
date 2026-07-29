package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ynderboi/ftpsimp/internal/config"
	"github.com/ynderboi/ftpsimp/internal/server"
)

func main() {
	portFlag := flag.Int("port", 0, "HTTP port (0 = from settings)")
	dirFlag := flag.String("dir", "", "shared folder (empty = from settings)")
	pinFlag := flag.String("pin", "", "fixed PIN (empty = from settings or generate)")
	openFlag := flag.Bool("open", false, "disable PIN auth (trusted LAN only)")
	roFlag := flag.Bool("readonly", false, "read-only: list and download only")
	flag.Parse()

	cfg := config.Load()

	if *portFlag > 0 {
		cfg.Port = *portFlag
	}
	if *dirFlag != "" {
		cfg.Root = *dirFlag
	}

	pin := strings.TrimSpace(*pinFlag)
	if pin == "" {
		pin = strings.TrimSpace(cfg.PIN)
	}
	// -open / -readonly apply to this process; persisted only via config.json fields.
	authOn := !cfg.Open && !*openFlag
	readOnly := cfg.ReadOnly || *roFlag

	if cfg.Root == "" {
		def, err := config.DefaultRoot()
		if err != nil {
			fatal("cannot resolve home: %v", err)
		}
		cfg.Root = def
	}

	abs, err := filepath.Abs(cfg.Root)
	if err != nil {
		fatal("invalid path: %v", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		fatal("cannot open share folder: %v", err)
	}
	cfg.Root = abs

	persist := cfg
	persist.Root = abs
	if strings.TrimSpace(*pinFlag) != "" {
		persist.PIN = pin
	}
	if err := config.Save(persist); err != nil {
		fmt.Fprintf(os.Stderr, "config save: %v\n", err)
	}

	cfgPath, _ := config.Path()
	ips := localIPs()
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := server.New(abs, addr, func(root string) error {
		persist.Root = root
		return config.Save(persist)
	}, server.Options{
		PIN:      pin,
		AuthOn:   authOn,
		ReadOnly: readOnly,
	})

	fmt.Println()
	fmt.Println("  ftpsimp — LAN file share")
	fmt.Println("  ────────────────────────────────")
	fmt.Printf("  Folder:   %s\n", abs)
	fmt.Printf("  Port:     %d\n", cfg.Port)
	if readOnly {
		fmt.Println("  Mode:     READ-ONLY")
	} else {
		fmt.Println("  Mode:     read/write")
	}
	if authOn {
		fmt.Printf("  PIN:      %s\n", srv.PIN())
		fmt.Println("  Auth:     ON (enter PIN in browser)")
	} else {
		fmt.Println("  Auth:     OFF (-open / config) — anyone on LAN can access")
	}
	if cfgPath != "" {
		fmt.Printf("  Settings: %s\n", cfgPath)
	}
	fmt.Println()
	if len(ips) == 0 {
		fmt.Println("  No LAN address found. Connect to Wi‑Fi and restart.")
	} else {
		fmt.Println("  Open in browser on another device:")
		for _, ip := range ips {
			fmt.Printf("    http://%s:%d\n", ip, cfg.Port)
		}
	}
	fmt.Println()
	fmt.Println("  Смена корневой папки — только с хоста → Settings.")
	fmt.Println("  Press Ctrl+C to stop.")
	fmt.Println()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case <-stop:
		fmt.Println("\n  Stopped.")
	case err := <-errCh:
		fatal("server error: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	fmt.Fprintln(os.Stderr, "\nPress Enter to exit...")
	fmt.Scanln()
	os.Exit(1)
}

func localIPs() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "virtual") || strings.Contains(name, "vmware") ||
			strings.Contains(name, "vbox") || strings.Contains(name, "hyper-v") ||
			strings.Contains(name, "docker") || strings.Contains(name, "vethernet") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			out = append(out, ip4.String())
		}
	}
	return out
}
