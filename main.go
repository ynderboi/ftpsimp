package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ynderboi/ftpsimp/internal/config"
	"github.com/ynderboi/ftpsimp/internal/server"
	"github.com/ynderboi/ftpsimp/internal/tui"
)

func main() {
	portFlag := flag.Int("port", 0, "HTTP port (0 = from settings)")
	dirFlag := flag.String("dir", "", "shared folder (empty = from settings)")
	pinFlag := flag.String("pin", "", "fixed PIN (empty = from settings or generate)")
	openFlag := flag.Bool("open", false, "disable PIN auth (trusted LAN only)")
	roFlag := flag.Bool("readonly", false, "read-only: list and download only")
	plainFlag := flag.Bool("plain", false, "disable interactive TUI (classic console log)")
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
	urls := make([]string, 0, len(ips))
	for _, ip := range ips {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, cfg.Port))
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := server.New(abs, addr, func(root string) error {
		persist.Root = root
		return config.Save(persist)
	}, server.Options{
		PIN:      pin,
		AuthOn:   authOn,
		ReadOnly: readOnly,
	})

	if err := srv.Start(); err != nil {
		fatal("server error: %v", err)
	}

	opt := tui.Options{
		Server:   srv,
		Persist:  &persist,
		URLs:     urls,
		CfgPath:  cfgPath,
		LocalIPs: localIPs,
	}

	useTUI := !*plainFlag && isTerminal()
	if useTUI {
		if err := tui.Run(opt); err != nil {
			fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		}
		return
	}

	tui.PrintPlain(opt)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	fmt.Println("\n  Stopped.")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Stop(ctx)
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
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
