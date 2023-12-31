package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Top-Weerapat-Mungmee/api-go-starter/dataloader"
	"github.com/Top-Weerapat-Mungmee/api-go-starter/graphql"
	customMiddleware "github.com/Top-Weerapat-Mungmee/api-go-starter/middleware"
	"github.com/Top-Weerapat-Mungmee/api-go-starter/mongodb"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// Init start server
func Init() {
	// get .env config
	GetConfig()

	// connect to db
	db := ConnectDB()

	// connect redis
	rdb := connectRedis()

	var (
		userRepo   = mongodb.UserRepo{DB: db.Collection("user")}
		deviceRepo = mongodb.DeviceRepo{DB: db.Collection("device")}
		roomRepo   = mongodb.RoomRepo{DB: db.Collection("room")}
		emailRepo  = mongodb.EmailRepo{DB: db.Collection("email")}
	)

	router := chi.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.RealIP)
	router.Use(middleware.GetHead)
	router.Use(middleware.Recoverer)
	router.Use(customMiddleware.AuthMiddleware(userRepo))

	d := &dataloader.DBLoader{
		DeviceRepo: deviceRepo,
		RoomRepo:   roomRepo,
	}
	router.Use(dataloader.DataMiddleware(d))

	c := graphql.Config{Resolvers: &graphql.Resolver{
		UserRepo:   userRepo,
		DeviceRepo: deviceRepo,
		RoomRepo:   roomRepo,
		EmailRepo:  emailRepo,
		Redis:      *rdb,
	}}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.New(graphql.NewExecutableSchema(c))
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	router.Handle("/graphql", srv)
	router.Get("/health-check", healthCheck)

	log.Printf("connect to %s for GraphQL playground", os.Getenv("CORS_DEFAULT"))
	log.Fatal(http.ListenAndServe(":"+port, router))
}
