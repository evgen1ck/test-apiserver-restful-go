package handlers

import (
	"github.com/go-chi/chi/v5"
	"test-server-go/internal/api_v1"
	"test-server-go/internal/models"
	"time"
)

const (
	timeout            = 5 * time.Second
	rateLimitRequests  = 15
	rateLimitInterval  = 10 * time.Second
	requestMaxSize     = 12 * 1024 * 1024 // 12 MB
	uriMaxLength       = 1024             // 1024 runes
	serviceUnavailable = false
)

var (
	allowedMethods        = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	allowedContentTypes   = []string{"application/json", "text/plain"}
	supportedHttpVersions = []string{"HTTP/1.1", "HTTP/2.0"}
)

type Resolver struct {
	App *models.Application
}

func (rs *Resolver) SetupRouterApiVer1(pathPrefix string) {
	r := chi.NewRouter()

	// CORS settings
	r.Use(api_v1.CorsMiddleware())

	// Error handling
	r.Use(api_v1.ServiceUnavailableMiddleware(serviceUnavailable))          // Error 503 - Service Unavailable
	r.Use(api_v1.RateLimitMiddleware(rateLimitRequests, rateLimitInterval)) // Error 429 - Too Many Requests
	r.Use(api_v1.UriLengthMiddleware(uriMaxLength))                         // Error 414 - URI Too Long
	r.Use(api_v1.RequestSizeMiddleware(requestMaxSize))                     // Error 413 - Payload Too Large
	r.Use(api_v1.UnprocessableEntityMiddleware)                             // Error 422 - Unprocessable Entity
	r.Use(api_v1.UnsupportedMediaTypeMiddleware(allowedContentTypes))       // Error 415 - Unsupported Media Type
	r.Use(api_v1.NotImplementedMiddleware(allowedMethods))                  // Error 501 - Not implemented
	r.Use(api_v1.MethodNotAllowedMiddleware)                                // Error 405 - Method Not Allowed
	r.Use(api_v1.GatewayTimeoutMiddleware(timeout))                         // Error 504 - Gateway Timeout
	r.Use(api_v1.HttpVersionCheckMiddleware(supportedHttpVersions))
	r.NotFound(api_v1.NotFoundMiddleware()) // Error 404 - Not Found
	//r.Use(api_v1.NotAcceptableMiddleware(allowedContentTypes))            // Error 406 - Not Acceptable (inactive)

	rs.registerRoutes(r)

	rs.App.Router.Mount(pathPrefix, r)
}

func (rs *Resolver) registerRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", rs.AuthSignup)
		r.Post("/signup-with-token", rs.AuthSignupWithToken)
		r.Post("/login", rs.AuthLogin)
		r.Post("/login-with-token", rs.AuthLoginWithToken)
		r.Post("/recover-password", rs.AuthRecoverPassword)
		r.Post("/logout", rs.AuthLogout)
	})
	r.Route("/goods", func(r chi.Router) {
		r.Get("/games", rs.GoodsGames)
		r.Get("/game/{name}", rs.GoodsGame)
		r.Get("/others", rs.GoodsOthers)
		r.Get("/other/{name}", rs.GoodsOther)
	})
	r.Route("/profile", func(r chi.Router) {
		r.Get("/data", rs.ProfileData)
		r.Post("/dump", rs.ProfileDump)
		r.Patch("/update", rs.ProfileUpdate)
		r.Delete("/delete", rs.ProfileDelete)
		r.Get("/orders", rs.ProfileOrders)
		r.Get("/order/{uuid}", rs.ProfileOrder)
	})
	r.Route("/admin", func(r chi.Router) {
		r.Post("/postgres-backup", rs.ServerPostgresBackup)
		r.Route("/games", func(r chi.Router) {
			r.Post("/add", rs.AdminGamesAdd)
			r.Patch("/update/{id}", rs.AdminGamesUpdate)
			r.Delete("/delete/{id}", rs.AdminGamesDelete)
		})
		r.Route("/others", func(r chi.Router) {
			r.Post("/add", rs.AdminOthersAdd)
			r.Patch("/update/{id}", rs.AdminOthersUpdate)
			r.Delete("/delete/{id}", rs.AdminOthersDelete)
		})
	})
}
