package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/frantjc/external-dns-dnsserver-webhook/externaldns"
	"github.com/frantjc/external-dns-dnsserver-webhook/hosts"
	xurl "github.com/frantjc/x/net/url"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/external-dns/provider/webhook/api"
)

func NewWebhook(version string) *cobra.Command {
	var (
		port, metricsPort, dnsHealthPort, dnsReadyPort, dnsMetricsPort int
		dnsCache                                                       string
		dnsForwardServers                                              []string
		initialHosts                                                   string
		verbosity int
		cmd                                                            = &cobra.Command{
			Use: "webhook",
			SilenceErrors: true,
			SilenceUsage: true,
			Version: version,
			RunE: func(cmd *cobra.Command, args []string) error {
				caddy.AppName = cmd.Name()
				caddy.AppVersion = cmd.Version
				log.Discard()

				var (
					eg, ctx     = errgroup.WithContext(cmd.Context())
					log         = slog.New(
						slog.NewTextHandler(cmd.OutOrStdout(), &slog.HandlerOptions{
							Level: slog.Level(verbosity),
						}),
					)
					metricsAddr = fmt.Sprintf(":%d", metricsPort)
				)

				if !log.Enabled(ctx, slog.LevelDebug) {
					dnsserver.Quiet = true
					caddy.Quiet = true
				}

				dnsCacheDuration, err := time.ParseDuration(dnsCache)
				if err != nil {
					return err
				}

				log.Info("DNS cache seconds " + fmt.Sprint(int(dnsCacheDuration.Seconds())))
				log.Info("DNS forward servers " + strings.Join(dnsForwardServers, ", "))

				f, err := os.CreateTemp("", "hosts-*")
				if err != nil {
					return err
				}
				defer os.Remove(f.Name())

				log.Info("hosts file " + f.Name())

				g := new(bytes.Buffer)
				if initialHosts != "" {
					g, err := xurl.OpenContext(ctx, initialHosts)
					if err != nil {
						return err
					}
					defer g.Close()

					log.Info("opened initial hosts " + initialHosts)
				}

				h, err := hosts.Decode(g)
				if err != nil {
					return err
				}

				log.Info("parsed initial hosts")

				if err := h.Encode(f); err != nil {
					return err
				}

				log.Info("wrote initial hosts to " + f.Name())

				if err = f.Close(); err != nil {
					return err
				}

				if _, err = caddy.Start(caddy.CaddyfileInput{
					Filepath:       "Corefile",
					ServerTypeName: "dns",
					Contents: []byte(fmt.Sprintf(
						`. {
  ready :%d
  health :%d {
    lameduck 5s
  }
  prometheus :%d
  header {
    response set ra
  }
  hosts %s {
    fallthrough
  }
  forward . %s
  cache %d
  loop
  loadbalance
}
`,
						dnsReadyPort,
						dnsHealthPort,
						dnsMetricsPort,
						f.Name(),
						strings.Join(dnsForwardServers, " "),
						int(dnsCacheDuration.Seconds()),
					)),
				}); err != nil {
					return err
				}
				defer caddy.Stop() //nolint:errcheck

				var (
					startedC = make(chan struct{})
					started  bool
				)

				go func() {
					<-startedC
					started = true
				}()

				l, err := net.Listen("tcp", metricsAddr)
				if err != nil {
					return err
				}
				defer l.Close()

				mux := http.NewServeMux()

				healthz := func(w http.ResponseWriter, r *http.Request) {
					if !started {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					fmt.Fprintln(w, "ok")
				}
				mux.HandleFunc("GET /readyz", healthz)
				mux.HandleFunc("GET /livez", healthz)
				mux.HandleFunc("GET /healthz", healthz)
				srv := &http.Server{
					Addr:              metricsAddr,
					ReadHeaderTimeout: time.Second * 5,
					BaseContext: func(_ net.Listener) context.Context {
						return ctx
					},
					Handler: mux,
				}

				eg.Go(func() error {
					log.Info("listening on " + metricsAddr)
					return srv.Serve(l)
				})
				defer srv.Close()

				eg.Go(func() error {
					<-ctx.Done()
					cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*30)
					defer cancel()
					return errors.Join(srv.Shutdown(cctx), ctx.Err())
				})

				api.StartHTTPApi(
					&externaldns.StandaloneCoreDNSProvider{
						Hosts: h,
						File:  f.Name(),
					},
					startedC,
					time.Second*5,
					0,
					fmt.Sprintf(":%d", port),
				)

				return eg.Wait()
			},
		}
	)

	cmd.Flags().BoolP("help", "h", false, "Help for "+cmd.Name())
	cmd.Flags().Bool("version", false, "Version for "+cmd.Name())
	cmd.SetVersionTemplate("{{ .Name }}{{ .Version }} coredns" + coremain.CoreVersion + " " + runtime.Version() + "\n")

	cmd.Flags().CountVarP(&verbosity, "verbose", "v", fmt.Sprintf("Verbosity for %s.", cmd.Name()))

	cmd.Flags().StringVar(&dnsserver.Port, "dns-port", dnsserver.DefaultPort, "DNS port")
	cmd.Flags().IntVar(&dnsMetricsPort, "dns-metrics-port", 8181, "DNS metrics port")
	cmd.Flags().IntVar(&dnsHealthPort, "dns-health-port", 8282, "DNS health port")
	cmd.Flags().IntVar(&dnsReadyPort, "dns-ready-port", 9153, "DNS ready port")

	cmd.Flags().StringVar(&dnsCache, "dns-cache", "30s", "DNS cache time")
	cmd.Flags().StringSliceVar(&dnsForwardServers, "dns-forward-server", []string{"1.1.1.2", "1.1.1.1", "8.8.8.8", "8.8.4.4"}, "DNS servers to forward to after fallthrough")

	cmd.Flags().StringVar(&initialHosts, "init-hosts", "", "Initial hosts file")

	cmd.Flags().IntVar(&metricsPort, "metrics-port", 8080, "Metrics port")
	cmd.Flags().IntVar(&port, "port", 8888, "Port")

	return cmd
}
