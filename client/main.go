package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type (
	ServerConfig struct {
		host string
		port string
	}
	Config struct {
		server ServerConfig
		zeros  uint8
	}
	Handler struct {
		logger *zap.SugaredLogger
	}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGSTOP, syscall.SIGTERM)
	defer stop()

	log, _ := zap.NewProduction()
	if err := newCommand().ExecuteContext(ctx); err != nil {
		log.Sugar().Fatalf("Failed to execute command: %s", err)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			viper.AutomaticEnv()
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return fmt.Errorf(`[PaaS registry backend] failed to bind command line arguments: %+v`, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := newConfig(viper.GetViper())
			log, _ := zap.NewProduction()
			slog := log.Sugar()
			slog.Info(`[client] Starting http-server localhost`)
			httpServer := newServer(cfg, slog)
			wg := sync.WaitGroup{}
			wg.Add(1)
			ctx := cmd.Context()

			go func() {
				<-ctx.Done()
				slog.Info("[client] attempting to shutdown gracefully")
				ct, _ := context.WithTimeout(context.Background(), 1*time.Second)
				_ = httpServer.Shutdown(ct)
				wg.Done()
			}()

			go func() {
				slog.Info(`[client] Serve http-server localhost`)
				if err := httpServer.ListenAndServe(); err != nil {
					slog.Errorf(`[client] error while serving: %+v`, err)
					wg.Done()
				}
			}()

			wg.Wait()
			slog.Infof("[client] clean shutdown")

			return nil
		},
	}

	cmd.Flags().String(`server.host`, `localhost`, `Server host`)
	cmd.Flags().String(`server.port`, `8080`, `Server port`)
	cmd.Flags().Uint(`zeros`, 20, `Length of the hash zeros prefix`)

	return cmd
}

func newConfig(viper *viper.Viper) *Config {
	return &Config{
		server: ServerConfig{
			host: viper.GetString(`server.host`),
			port: viper.GetString(`server.port`),
		},
		zeros: uint8(viper.GetUint(`zeros`)),
	}
}

func newServer(cfg *Config, logger *zap.SugaredLogger) *http.Server {
	r := mux.NewRouter()
	r.Handle(`/`, Handler{logger})

	return &http.Server{
		Addr:    "localhost:80",
		Handler: r,
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, `{"client":"OK"}`)
	if err != nil {
		h.logger.Errorf("[server] %+v", err)
	}
}
