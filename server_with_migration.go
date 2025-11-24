package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/example/auth0-gqlgen-demo/auth"
	"github.com/example/auth0-gqlgen-demo/graph"
	"github.com/example/auth0-gqlgen-demo/migration"
	"github.com/example/auth0-gqlgen-demo/store"
)

func main() {
	// Auth0 Configuration
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	auth0Audience := os.Getenv("AUTH0_AUDIENCE")
	if auth0Domain == "" || auth0Audience == "" {
		log.Fatal("AUTH0_DOMAIN and AUTH0_AUDIENCE environment variables are required")
	}

	// Passage Configuration (for migration)
	passageAppID := os.Getenv("PASSAGE_APP_ID")
	passageAPIKey := os.Getenv("PASSAGE_API_KEY")
	
	// Auth0 credentials for user management
	auth0ClientID := os.Getenv("CLIENT_ID")
	auth0ClientSecret := os.Getenv("CLIENT_SECRET")
	auth0Connection := os.Getenv("AUTH0_CONNECTION") // e.g., "Username-Password-Authentication"
	if auth0Connection == "" {
		auth0Connection = "Username-Password-Authentication"
	}

	// Initialize storage
	memoryStore := store.NewMemoryStore()

	// Initialize GraphQL server
	resolver := &graph.Resolver{
		Store: memoryStore,
	}
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// GraphQL endpoints with Auth0 middleware
	config := auth.Auth0Config{
		Domain:   auth0Domain,
		Audience: auth0Audience,
	}
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", auth.Middleware(config)(srv))

	// Migration endpoints (if Passage credentials are provided)
	if passageAppID != "" && passageAPIKey != "" {
		log.Println("Migration endpoints enabled")
		
		// Initialize migration service
		migrationService, err := migration.NewTokenExchangeService(
			passageAppID,
			passageAPIKey,
			auth0Domain,
			auth0ClientID,
			auth0ClientSecret,
			auth0Audience,
			auth0Connection,
		)
		if err != nil {
			log.Fatalf("Failed to initialize migration service: %v", err)
		}

		migrationHandler := migration.NewHandler(migrationService)
		
		// Migration endpoints (no auth required - Passage token validates the user)
		http.HandleFunc("/migrate/exchange-token", migrationHandler.HandleExchangeToken)
		http.HandleFunc("/migrate/stats", migrationHandler.HandleMigrationStats)
		
		log.Println("  POST /migrate/exchange-token - Exchange Passage JWT for Auth0 user")
		log.Println("  GET  /migrate/stats - View migration statistics")
	} else {
		log.Println("Migration endpoints disabled (set PASSAGE_APP_ID and PASSAGE_API_KEY to enable)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

