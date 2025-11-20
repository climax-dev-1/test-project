package graph

import "github.com/example/auth0-gqlgen-demo/store"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	Store *store.MemoryStore
}

