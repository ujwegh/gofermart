package main

import (
	"context"
	"fmt"
	"github.com/ujwegh/gophermart/internal/app/config"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	"github.com/ujwegh/gophermart/internal/app/logger"
	middlware "github.com/ujwegh/gophermart/internal/app/middleware"
	"github.com/ujwegh/gophermart/internal/app/repository"
	"github.com/ujwegh/gophermart/internal/app/router"
	"github.com/ujwegh/gophermart/internal/app/service"
	"github.com/ujwegh/gophermart/internal/app/service/clients"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// @title           Swagger Docs for Gophermart API
// @version         1.0
// @description     This is a `gophermart` service. It allows users to create orders, credit/debit their wallets and withdraw funds from their wallets using the accrual service.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Nikita Aleksandrov
// @contact.email  nik29200018@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/user

// @securityDefinitions.apikey  ApiKeyAuth
// @in header
// @name Authorization

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	c := config.ParseFlags()
	logger.InitLogger(c.LogLevel)

	ts := service.NewTokenService(c)
	s := repository.NewDBStorage(c)
	ur := repository.NewUserRepository(s.DBConn)
	or := repository.NewOrderRepository(s.DBConn)
	wr := repository.NewWalletRepository(s.DBConn)
	wlr := repository.NewWithdrawalsRepository(s.DBConn)

	processOrderChannel := make(chan repository.Order, 100)

	ws := service.NewWalletService(wr)
	ors := service.NewOrderService(or, ws, processOrderChannel)
	oc := service.NewOrderCache(10*time.Second, 5*time.Minute, processOrderChannel)
	ac := clients.NewAccrualClient(c)
	wls := service.NewWithdrawalService(wlr, ws)
	us := service.NewUserService(ur, ws)

	uh := handlers.NewUserHandler(us, ts, c.TokenLifetimeSec)
	oh := handlers.NewOrdersHandler(c.ContextTimeoutSec, ors)
	bh := handlers.NewBalanceHandler(c.ContextTimeoutSec, ws, wls)

	am := middlware.NewAuthMiddleware(ts, us, c.ContextTimeoutSec)

	r := router.NewAppRouter(c.ServerAddr, uh, oh, bh, am)

	op := service.NewOrderProcessor(or, oc, ws, ac, processOrderChannel)
	go op.ProcessOrders(serverCtx)

	server := &http.Server{Addr: c.ServerAddr, Handler: r}

	serverErrors := make(chan error, 1)
	go func() {
		fmt.Printf("Starting server on port %s...\n", strings.Split(c.ServerAddr, ":")[1])
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-shutdown:
		log.Printf("Start shutdown %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			log.Fatalf("graceful shutdown did not complete in 30s: %v", err)
		}
		close(processOrderChannel)

	case err := <-serverErrors:
		log.Fatalf("error: listening and serving: %v", err)
	}

	serverStopCtx()
	log.Println("finished shutting down server")
}
