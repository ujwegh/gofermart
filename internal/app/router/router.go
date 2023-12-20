package router

import (
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/ujwegh/gophermart/docs"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	middlware "github.com/ujwegh/gophermart/internal/app/middleware"
	"net/http"
)

func NewAppRouter(serverAddress string,
	uh *handlers.UserHandler,
	oh *handlers.OrdersHandler,
	bh *handlers.BalanceHandler,
	am middlware.AuthMiddleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(setupCORS())
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

func setupCORS() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
