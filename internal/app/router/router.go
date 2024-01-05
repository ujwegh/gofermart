package router

import (
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/ujwegh/gophermart/docs"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	middlware "github.com/ujwegh/gophermart/internal/app/middleware"
)

func NewAppRouter(serverAddress string,
	uh *handlers.UserHandler,
	oh *handlers.OrdersHandler,
	bh *handlers.BalanceHandler,
	am middlware.AuthMiddleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middlware.SetupCORS())
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://"+serverAddress+"/swagger/doc.json"),
	))

	r.Group(func(r chi.Router) {
		r.Use(middlware.RequestLogger)
		r.Use(middlware.ResponseLogger)
		r.Post("/api/user/register", uh.Register)
		r.Post("/api/user/login", uh.Login)

		r.Group(func(r chi.Router) {
			r.Use(am.Authenticate)
			r.Post("/api/user/orders", oh.CreateOrder)
			r.Get("/api/user/orders", oh.GetOrders)
			r.Get("/api/user/balance", bh.GetBalance)
			r.Post("/api/user/balance/withdraw", bh.Withdraw)
			r.Get("/api/user/withdrawals", bh.GetWithdrawals)
		})
	})

	return r
}
