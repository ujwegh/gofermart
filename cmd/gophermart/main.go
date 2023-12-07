package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ujwegh/gophermart/internal/app/config"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	"github.com/ujwegh/gophermart/internal/app/logger"
	middlware "github.com/ujwegh/gophermart/internal/app/middleware"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"github.com/ujwegh/gophermart/internal/app/router"
	"github.com/ujwegh/gophermart/internal/app/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	c := config.ParseFlags()
	logger.InitLogger(c.LogLevel)

	ts := service.NewTokenService(c)
	s := repository.NewDBStorage(c)
	ur := repository.NewUserRepository(s.DbConn)
	us := service.NewUserService(ur)
	uh := handlers.NewUserHandler(us, ts, c.TokenLifetimeSec)
	am := middlware.NewAuthMiddleware(ts, us, c.ContextTimeoutSec)

	r := router.NewAppRouter(uh, am)

	// The HTTP Server
	server := &http.Server{Addr: c.ServerAddr, Handler: r}

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelFunc := context.WithTimeout(serverCtx, 30*time.Second)
		cancelFunc()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	// Run the server
	fmt.Printf("Starting server on port %s...\n", strings.Split(c.ServerAddr, ":")[1])
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	// Wait for server context to be stopped
	<-serverCtx.Done()
}
