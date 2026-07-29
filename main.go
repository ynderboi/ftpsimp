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
	flag.Parse()

	cfg := config.Load()

	if *portFlag > 0 {
		cfg.Port = *portFlag
	}
	if *dirFlag != "" {
		cfg.Root = *dirFlag
	}
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

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config save: %v\n", err)
	}

	cfgPath, _ := config.Path()
	ips := localIPs()
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := server.New(abs, addr, func(root string) error {
		cfg.Root = root
		return config.Save(cfg)
	})

	fmt.Println()
	fmt.Println("  ftpsimp — file share over Wi‑Fi")
	fmt.Println("  ────────────────────────────────")
	fmt.Printf("  Folder:   %s\n", abs)
	fmt.Printf("  Port:     %d\n", cfg.Port)
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
	fmt.Println("  Корневую папку можно сменить в веб-интерфейсе → Настройки.")
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
