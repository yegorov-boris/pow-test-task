package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os/signal"
	"server/check"
	"server/quotes"
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
		ttl time.Duration
	}
	Config struct {
		server ServerConfig
		cache  CacheConfig
		zeros  uint8
	}
	Handler struct {
		logger   *zap.SugaredLogger
		zeros    uint8
		quoteGen *quotes.Generator
		checker  *check.Checker
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
				return fmt.Errorf(`failed to bind command line arguments: %+v`, err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := newConfig(viper.GetViper())
			log, _ := zap.NewProduction()
			slog := log.Sugar()
			slog.Infof(`Starting http-server localhost:%s`, cfg.server.port)
			checker := check.NewChecker(cfg.cache.ttl)
			stopChecker := make(chan struct{})
			checker.Run(stopChecker)

			quoteGen, err := quotes.NewGenerator()
			if err != nil {
				return errors.Wrap(err, "failed to create quotes generator")
			}

			httpServer := newServer(cfg, slog, checker, quoteGen)
			wg := sync.WaitGroup{}
			wg.Add(1)
			ctx := cmd.Context()

			go func() {
				<-ctx.Done()
				slog.Info("Attempting to shutdown gracefully")
				ct, _ := context.WithTimeout(context.Background(), 1*time.Second)
				_ = httpServer.Shutdown(ct)
				stopChecker <- struct{}{}
				wg.Done()
			}()

			go func() {
				slog.Infof(`Serve http-server localhost:%s`, cfg.server.port)
				if err := httpServer.ListenAndServe(); err != nil {
					slog.Errorf(`error while serving: %+v`, err)
					stopChecker <- struct{}{}
					wg.Done()
				}
			}()

			wg.Wait()
			slog.Infof("Clean shutdown")

			return nil
		},
	}

	cmd.Flags().String(`server.port`, `8080`, `Server port`)
	cmd.Flags().Duration(`server.write_timeout`, time.Millisecond*100, `Maximum duration before timing out writes of the response`)
	cmd.Flags().Duration(`server.read_timeout`, time.Millisecond*50, `Maximum duration for reading the entire request`)
	cmd.Flags().Duration(`cache.ttl`, 1*time.Second, `Cache TTL for proven hashes`)
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
			ttl: viper.GetDuration(`cache.ttl`),
		},
		zeros: uint8(viper.GetUint(`zeros`)),
	}
}

func newServer(
	cfg *Config,
	logger *zap.SugaredLogger,
	checker *check.Checker,
	quoteGen *quotes.Generator,
) *http.Server {
	r := mux.NewRouter()
	r.Handle(`/{id}`, Handler{
		logger:   logger,
		zeros:    cfg.zeros,
		quoteGen: quoteGen,
		checker:  checker,
	})

	return &http.Server{
		Addr:         fmt.Sprintf("localhost:%s", cfg.server.port),
		WriteTimeout: time.Millisecond * cfg.server.writeTimeout,
		ReadTimeout:  time.Millisecond * cfg.server.readTimeout,
		Handler:      r,
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	h.logger.Infof("id: %s", id)
	idLength := len(id)
	if idLength < 13 || idLength > 16 {
		w.WriteHeader(http.StatusBadRequest)
		_, e := fmt.Fprintf(w, "invalid ID")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	idBase64, err := url.PathUnescape(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, e := fmt.Fprintf(w, "failed to decode PoW from URL on the server")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	idB, err := base64.StdEncoding.DecodeString(idBase64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, e := fmt.Fprintf(w, "failed to decode PoW bytes from base64 on the server")
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	if h.checker.CheckPoW(h.zeros, idB) {
		if !h.checker.IsUniq(idB) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, e := fmt.Fprintf(w, "this PoW has been recently used, please try again")
			if e != nil {
				h.logger.Errorf("%+v", e)
			}

			return
		}

		w.WriteHeader(http.StatusOK)
		_, e := fmt.Fprintf(w, h.quoteGen.Get())
		if e != nil {
			h.logger.Errorf("%+v", e)
		}

		return
	}

	w.WriteHeader(http.StatusForbidden)
	_, e := fmt.Fprintf(w, "PoW failed")
	if e != nil {
		h.logger.Errorf("%+v", e)
	}
}
