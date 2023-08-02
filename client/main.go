package main

import (
	"client/pow"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
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
		logger    *zap.SugaredLogger
		serverURL string
		zeros     uint8
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
				return fmt.Errorf("failed to bind command line arguments: %+v", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := newConfig(viper.GetViper())
			log, _ := zap.NewProduction()
			slog := log.Sugar()
			slog.Info(`Starting http-server localhost`)
			httpServer := newServer(cfg, slog)
			wg := sync.WaitGroup{}
			wg.Add(1)
			ctx := cmd.Context()

			go func() {
				<-ctx.Done()
				slog.Info("Attempting to shutdown gracefully")
				ct, _ := context.WithTimeout(context.Background(), 1*time.Second)
				_ = httpServer.Shutdown(ct)
				wg.Done()
			}()

			go func() {
				slog.Info(`Serve http-server localhost`)
				if err := httpServer.ListenAndServe(); err != nil {
					slog.Errorf(`error while serving: %+v`, err)
					wg.Done()
				}
			}()

			wg.Wait()
			slog.Infof("Clean shutdown")

			return nil
		},
	}

	cmd.Flags().String(`server.host`, `http://localhost`, `Server host`)
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
	r.Handle("/pow", Handler{
		logger:    logger,
		serverURL: fmt.Sprintf("%s:%s", cfg.server.host, cfg.server.port),
		zeros:     cfg.zeros,
	})

	return &http.Server{
		Addr:    "localhost:80",
		Handler: r,
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("----- generating")
	id, err := pow.Generate(h.zeros)
	if err != nil {
		h.logger.Errorf("error generating ID: %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, e := fmt.Fprintf(w, "failed to compute a Proof of Work challenge, please try again")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}
	h.logger.Infof("====== generated %s", id)
	id = url.QueryEscape(id)
	reqURL := fmt.Sprintf("%s/pow?id=%s", h.serverURL, id)
	h.logger.Info(reqURL)
	res, err := http.Get(reqURL)
	if err != nil {
		h.logger.Errorf("error making http request: %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, e := fmt.Fprintf(w, "failed to make an http request to the server")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	defer func() {
		e := res.Body.Close()
		if e != nil {
			h.logger.Errorf("%+v", e)
		}
	}()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		h.logger.Errorf("error reading http response: %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, e := fmt.Fprintf(w, "failed to read an http response body from the server")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	w.WriteHeader(res.StatusCode)
	_, err = fmt.Fprintf(w, string(resBody))
	if err != nil {
		h.logger.Errorf("%+v", err)
	}
}
