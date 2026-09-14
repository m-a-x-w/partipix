package server

import (
	"fmt"
	"log"
	"partipix/backend/api"
	"partipix/backend/internal/database"
	"strings"

	"github.com/valyala/fasthttp"
)

func Main() {
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	router := api.NewRouter()
	maxSize := 100 * 1024 * 1024 // 100MB

	server := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			// Check if request body exceeds the limit before calling the handler
			if len(ctx.Request.Body()) > maxSize {
				ctx.SetStatusCode(fasthttp.StatusRequestEntityTooLarge)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"error": "Upload size too large. Please upload fewer or smaller photos at once."}`)
				return
			}

			router.Handler()(ctx)
		},
		MaxRequestBodySize: maxSize,
		Name:               "Partipix API",
		// Custom error handler for other errors
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			if err != nil && strings.Contains(err.Error(), "body size exceeds") {
				ctx.SetStatusCode(fasthttp.StatusRequestEntityTooLarge)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"error": "Upload size too large. Please upload fewer or smaller photos at once."}`)
				return
			}
			// Default error handling
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		},
	}

	port := ":8080"

	fmt.Printf("Starting HTTP server on port %s...\n", port)
	if err := server.ListenAndServe(port); err != nil {
		log.Fatalf("Error in ListenAndServe: %v", err)
	}
}
