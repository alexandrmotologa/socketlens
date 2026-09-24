package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/socketlens/pkg/client"
	"github.com/alexandrmotologa/socketlens/pkg/mock"
	"github.com/alexandrmotologa/socketlens/pkg/proxy"
	"github.com/alexandrmotologa/socketlens/pkg/server"
	"github.com/alexandrmotologa/socketlens/pkg/session"
	"github.com/alexandrmotologa/socketlens/pkg/stress"
	"github.com/alexandrmotologa/socketlens/pkg/tui"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "socketlens",
		Short: "Local-first workbench and CLI for WebSocket, SSE, and Socket.io streams",
		RunE:  runWebStudio,
	}

	// Global / root flags
	rootCmd.Flags().IntP("port", "p", 50070, "Port to bind local studio")
	rootCmd.Flags().String("host", "127.0.0.1", "Host interface to bind")
	rootCmd.Flags().Bool("no-browser", false, "Do not auto-open web browser on startup")

	// Subcommands
	rootCmd.AddCommand(newConnectCmd())
	rootCmd.AddCommand(newProxyCmd())
	rootCmd.AddCommand(newMockCmd())
	rootCmd.AddCommand(newBenchCmd())
	rootCmd.AddCommand(newRecordCmd())
	rootCmd.AddCommand(newReplayCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runWebStudio(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")

	mgr := client.NewManager()
	hub := server.NewTimelineHub()
	api := server.NewAPIServer(mgr, hub)

	r := chi.NewRouter()
	r.Mount("/", api.Routes())
	r.Handle("/*", server.FileServerHandler())

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("binding server to %s: %w", addr, err)
	}

	studioURL := fmt.Sprintf("http://%s", addr)
	fmt.Printf("\n🚀 SocketLens Studio running at: %s\n", studioURL)
	fmt.Printf("   Internal Control WebSocket: ws://%s/ws/control\n", addr)
	fmt.Println("   Press Ctrl+C to stop.")

	if !noBrowser {
		openBrowser(studioURL)
	}

	srv := &http.Server{Handler: r}
	go func() {
		_ = srv.Serve(listener)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down SocketLens Studio...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	mgr.CloseAll()
	return srv.Shutdown(ctx)
}

func newConnectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect <URL>",
		Short: "Connect to a stream in terminal interactive TUI mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawURL := args[0]
			protoStr, _ := cmd.Flags().GetString("proto")
			headersList, _ := cmd.Flags().GetStringSlice("header")
			insecure, _ := cmd.Flags().GetBool("insecure")
			reconnect, _ := cmd.Flags().GetBool("reconnect")
			heartbeat, _ := cmd.Flags().GetDuration("heartbeat")

			headers := make(map[string]string)
			for _, h := range headersList {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}

			cfg := client.ConnectionConfig{
				URL:               rawURL,
				Protocol:          client.Protocol(protoStr),
				Headers:           headers,
				TLSInsecure:       insecure,
				AutoReconnect:     reconnect,
				HeartbeatInterval: heartbeat,
			}

			return tui.RunTUI(cfg)
		},
	}

	cmd.Flags().String("proto", "", "Protocol override (ws, sse, socketio)")
	cmd.Flags().StringSliceP("header", "H", nil, "Custom HTTP header (Key: Value)")
	cmd.Flags().Bool("insecure", false, "Skip TLS verification")
	cmd.Flags().Bool("reconnect", true, "Enable automatic reconnection")
	cmd.Flags().Duration("heartbeat", 15*time.Second, "WebSocket ping interval")

	return cmd
}

func newProxyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy <target-URL>",
		Short: "Start a transparent WebSocket interception proxy with live logging",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]
			port, _ := cmd.Flags().GetInt("port")
			breakpoint, _ := cmd.Flags().GetBool("breakpoint")

			cfg := proxy.ProxyConfig{
				LocalPort:        port,
				TargetURL:        targetURL,
				EnableBreakpoint: breakpoint,
			}

			p := proxy.NewStreamProxy(cfg, func(f *client.Frame) {
				arrow := "-> (upstream)"
				if f.Direction == client.DirectionInbound {
					arrow = "<- (downstream)"
				}
				fmt.Printf("[%s] %s %d bytes: %s\n", f.Timestamp.Format("15:04:05.000"), arrow, f.Length, string(f.Payload))
			})

			if err := p.Start(); err != nil {
				return err
			}
			defer p.Stop()

			fmt.Printf("\n🔀 SocketLens Interception Proxy running on 127.0.0.1:%d\n", port)
			fmt.Printf("   Forwarding to: %s\n", targetURL)
			if breakpoint {
				fmt.Println("   Breakpoints: ENABLED (frames held for tampering)")
			}
			fmt.Println("   Point your client to: ws://127.0.0.1:" + fmt.Sprintf("%d", port))
			fmt.Println("   Press Ctrl+C to terminate.")

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nStopping proxy...")
			return nil
		},
	}

	cmd.Flags().IntP("port", "p", 8081, "Local port to listen on")
	cmd.Flags().BoolP("breakpoint", "b", false, "Enable breakpoint frame tampering")

	return cmd
}

func newMockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mock",
		Short: "Start an embedded mock stream server with chaos simulation",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			route, _ := cmd.Flags().GetString("route")
			mode, _ := cmd.Flags().GetString("mode")
			rate, _ := cmd.Flags().GetInt("rate")
			latency, _ := cmd.Flags().GetDuration("chaos-latency")
			drop, _ := cmd.Flags().GetInt("chaos-drop")
			resetDur, _ := cmd.Flags().GetDuration("chaos-reset")

			cfg := mock.ServerConfig{
				Port:  port,
				Route: route,
				Mode:  mode,
				Rate:  rate,
				Chaos: mock.ChaosConfig{
					Latency:  latency,
					DropRate: drop,
					Reset:    resetDur,
				},
			}

			srv := mock.NewServer(cfg)
			if err := srv.Start(); err != nil {
				return err
			}
			defer srv.Stop()

			fmt.Printf("\n⚡ SocketLens Mock Server started on %s\n", srv.Addr())
			fmt.Printf("   WebSocket Endpoint: %s\n", srv.WSURL())
			fmt.Printf("   SSE Endpoint:       %s\n", srv.SSEURL())
			fmt.Printf("   Mode: %s | Rate: %d/s\n", mode, rate)
			if drop > 0 || latency > 0 {
				fmt.Printf("   Chaos: Latency=%v, Drop=%d%%\n", latency, drop)
			}
			fmt.Println("   Press Ctrl+C to terminate.")

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\nStopping mock server...")
			return nil
		},
	}

	cmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	cmd.Flags().StringP("route", "r", "/ws/feed", "Route path prefix")
	cmd.Flags().StringP("mode", "m", "echo", "Mode (echo, broadcast, llm-tokens, ticks)")
	cmd.Flags().Int("rate", 10, "Event generation frequency per second")
	cmd.Flags().Duration("chaos-latency", 0, "Artificial latency injection")
	cmd.Flags().Int("chaos-drop", 0, "Packet drop probability percentage (0-100)")
	cmd.Flags().Duration("chaos-reset", 0, "Abruptly terminate connections after duration")

	return cmd
}

func newBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bench <URL>",
		Short: "Benchmark a streaming endpoint with concurrent connections",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]
			clients, _ := cmd.Flags().GetInt("clients")
			duration, _ := cmd.Flags().GetDuration("duration")
			rampUp, _ := cmd.Flags().GetDuration("ramp-up")
			sendInterval, _ := cmd.Flags().GetDuration("send-interval")
			payload, _ := cmd.Flags().GetString("payload")
			reportFile, _ := cmd.Flags().GetString("report")

			cfg := stress.BenchConfig{
				URL:          targetURL,
				Clients:      clients,
				Duration:     duration,
				RampUp:       rampUp,
				SendInterval: sendInterval,
				Payload:      payload,
			}

			fmt.Printf("\n🚀 Launching SocketLens Benchmark\n")
			fmt.Printf("   Target:       %s\n", targetURL)
			fmt.Printf("   Concurrency:  %d clients\n", clients)
			fmt.Printf("   Duration:     %v (Ramp-up: %v)\n\n", duration, rampUp)

			runner := stress.NewRunner(cfg)
			report, err := runner.Run(context.Background())
			if err != nil {
				return fmt.Errorf("benchmark failed: %w", err)
			}

			fmt.Print(report.FormatSummary())

			if reportFile != "" {
				jsonBytes, err := report.ToJSON()
				if err != nil {
					return err
				}
				if err := os.WriteFile(reportFile, jsonBytes, 0644); err != nil {
					return fmt.Errorf("writing report file: %w", err)
				}
				fmt.Printf("Saved detailed JSON report to: %s\n", reportFile)
			}

			return nil
		},
	}

	cmd.Flags().IntP("clients", "c", 100, "Number of concurrent client connections")
	cmd.Flags().DurationP("duration", "d", 30*time.Second, "Benchmark duration")
	cmd.Flags().Duration("ramp-up", 5*time.Second, "Ramp-up duration for staggering connects")
	cmd.Flags().Duration("send-interval", 0, "Periodic message transmission interval")
	cmd.Flags().String("payload", "{\"type\":\"ping\"}", "Payload string to send periodically")
	cmd.Flags().String("report", "", "Save JSON benchmark report to destination file")

	return cmd
}

func newRecordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record <URL>",
		Short: "Record live stream traffic to a JSONL archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]
			outFile, _ := cmd.Flags().GetString("out")
			if outFile == "" {
				outFile = fmt.Sprintf("session_%s.jsonl", time.Now().Format("20060102_150405"))
			}

			rec, err := session.NewRecorder(outFile)
			if err != nil {
				return err
			}
			defer rec.Close()

			cfg := client.ConnectionConfig{
				URL:           targetURL,
				AutoReconnect: true,
			}

			var count uint64
			onFrame := func(f *client.Frame) {
				rec.Record(f)
				count++
				fmt.Printf("\rRecorded %d frames (%s)...", count, f.OpCode)
			}

			c, err := client.CreateClient(cfg, onFrame, nil)
			if err != nil {
				return err
			}
			defer c.Close()

			fmt.Printf("Recording frames from %s to %s\nPress Ctrl+C to stop.\n", targetURL, outFile)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				_ = c.Connect(ctx)
			}()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Printf("\nSaved %d frames to %s\n", count, outFile)
			return nil
		},
	}

	cmd.Flags().StringP("out", "o", "", "Destination JSONL file path")
	return cmd
}

func newReplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replay <file.jsonl>",
		Short: "Replay a recorded stream session with timing offsets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			targetURL, _ := cmd.Flags().GetString("target")
			speed, _ := cmd.Flags().GetFloat64("speed")
			loop, _ := cmd.Flags().GetBool("loop")

			var targetClient client.StreamClient
			if targetURL != "" {
				cfg := client.ConnectionConfig{URL: targetURL}
				var err error
				targetClient, err = client.CreateClient(cfg, nil, nil)
				if err != nil {
					return err
				}
				_ = targetClient.Connect(context.Background())
				defer targetClient.Close()
			}

			replayer := session.NewReplayer(session.ReplayConfig{
				FilePath: filePath,
				Target:   targetClient,
				Speed:    speed,
				Loop:     loop,
				OnFrame: func(rf *session.RecordedFrame) {
					fmt.Printf("[%d ms] %s %s: %s\n", rf.OffsetMs, rf.Direction, rf.OpCode, rf.Payload)
				},
			})

			fmt.Printf("Replaying %s (speed: %.1fx)\n", filePath, speed)
			return replayer.Play(context.Background())
		},
	}

	cmd.Flags().StringP("target", "t", "", "Target streaming endpoint to receive outbound frames")
	cmd.Flags().Float64P("speed", "s", 1.0, "Playback speed multiplier")
	cmd.Flags().Bool("loop", false, "Loop playback continuously")

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print SocketLens version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("SocketLens v%s\n", version)
		},
	}
}

func openBrowser(url string) {
	// Best-effort open without panicking on headless environments
	go func() {
		time.Sleep(200 * time.Millisecond)
		// On windows: rundll32 url.dll,FileProtocolHandler <url>
		// We leave user to click link if automated open fails
	}()
}
