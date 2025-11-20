package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/example/auth0-gqlgen-demo/auth"
	"github.com/example/auth0-gqlgen-demo/graph"
	"github.com/example/auth0-gqlgen-demo/store"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Auth0 configuration
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	auth0Audience := os.Getenv("AUTH0_AUDIENCE")

	if auth0Domain == "" || auth0Audience == "" {
		log.Fatal("AUTH0_DOMAIN and AUTH0_AUDIENCE environment variables are required")
	}

	auth0Config := auth.Auth0Config{
		Domain:   auth0Domain,
		Audience: auth0Audience,
	}

	// Initialize in-memory store
	memoryStore := store.NewMemoryStore()

	// Initialize GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Store: memoryStore,
		},
	}))

	// Setup routes
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", auth.Middleware(auth0Config)(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

