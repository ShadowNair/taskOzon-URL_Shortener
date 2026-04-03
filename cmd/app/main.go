package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/config"
	db "url-shortener/internal/connections"
	linkhandler "url-shortener/internal/httpapi/link"
	"url-shortener/internal/httpapi/router"
	memory "url-shortener/internal/repository/memory/link"
	postgres "url-shortener/internal/repository/postgres/link"
	linkusecase "url-shortener/internal/usecase/link"
	"url-shortener/utils/generator"
)

func main() {
	cfg := config.GetConfig()

	var repo linkusecase.RepoI
	var cleanup func()

	storageType := flag.String("storage", "memory", "storage type: memory|postgres")
	flag.Parse()


	switch *storageType {
	case "memory":
		repo = memory.New()
	case "postgres":
		connCfg, err := db.New(cfg)
		if err != nil {
			log.Fatalf("failed to connect to DB: %v", err)
		}
		repo = postgres.New(connCfg.PostgresSQL)
		cleanup = connCfg.CloseAll
	default:
		log.Fatalf("unsupported storage type: %s", *storageType)
	}

	if cleanup != nil {
		defer cleanup()
	}

	uc := linkusecase.New(repo, &generator.RandomGenerator{})
	h := linkhandler.New(uc)
	rout := router.Setup(h)

	server := &http.Server{
		Addr:              cfg.AppConfig.Addr(),
		Handler:           rout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("starting server on %s with storage=%s", cfg.AppConfig.Addr(), *storageType)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}