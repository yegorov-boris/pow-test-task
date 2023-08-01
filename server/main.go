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
		port         string
		writeTimeout time.Duration
		readTimeout  time.Duration
	}
	CacheConfig struct {
		size uint32
	}
	Config struct {
		server ServerConfig
		cache  CacheConfig
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
			slog.Infof(`[server] Starting http-server localhost:%s`, cfg.server.port)
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
				slog.Infof(`[server] Serve http-server localhost:%s`, cfg.server.port)
				if err := httpServer.ListenAndServe(); err != nil {
					slog.Errorf(`[server] error while serving: %+v`, err)
					wg.Done()
				}
			}()

			wg.Wait()
			slog.Infof("[server] clean shutdown")

			return nil
		},
	}

	cmd.Flags().String(`server.port`, `8080`, `Server port`)
	cmd.Flags().Duration(`server.write_timeout`, time.Millisecond*100, `Maximum duration before timing out writes of the response`)
	cmd.Flags().Duration(`server.read_timeout`, time.Millisecond*50, `Maximum duration for reading the entire request`)
	cmd.Flags().Uint32(`cache.size`, 10000, `Max elements in the cache`)
	cmd.Flags().Uint8(`zeros`, 20, `Length of the hash zeros prefix`)

	return cmd
}

func newConfig(viper *viper.Viper) *Config {
	return &Config{
		server: ServerConfig{
			port:         viper.GetString(`server.port`),
			writeTimeout: viper.GetDuration(`server.write_timeout`),
			readTimeout:  viper.GetDuration(`server.read_timeout`),
		},
		cache: CacheConfig{
			size: viper.GetUint32(`cache.size`),
		},
		zeros: uint8(viper.GetUint(`zeros`)),
	}
}

func newServer(cfg *Config, logger *zap.SugaredLogger) *http.Server {
	r := mux.NewRouter()
	r.Handle(`/`, Handler{logger})

	return &http.Server{
		Addr:         fmt.Sprintf("localhost:%s", cfg.server.port),
		WriteTimeout: time.Millisecond * cfg.server.writeTimeout,
		ReadTimeout:  time.Millisecond * cfg.server.readTimeout,
		Handler:      r,
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, `{"status":"OK"}`)
	if err != nil {
		h.logger.Errorf("[server] %+v", err)
	}
}
