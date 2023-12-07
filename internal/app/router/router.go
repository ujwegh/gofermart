package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/ujwegh/gophermart/internal/app/handlers"
	middlware "github.com/ujwegh/gophermart/internal/app/middleware"
)

func NewAppRouter(uh *handlers.UserHandler, am middlware.AuthMiddleware) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middlware.RequestLogger)
	r.Use(middlware.ResponseLogger)

	r.Post("/api/user/register", uh.Register)
	r.Post("/api/user/login", uh.Login)

	r.Group(func(r chi.Router) {
		r.Use(am.Authenticate)

	})
	return r
}
