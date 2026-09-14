package middleware

import (
	"github.com/valyala/fasthttp"
	"strings"
	"partipix/backend/internal/database"
	"time"
)

// AuthMiddleware is a middleware that checks for a valid JWT token in the Authorization header
func AuthMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if authHeader == "" {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString(`{"error": "Authorization header is required"}`)
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString(`{"error": "Invalid authorization header format"}`)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		
		if token == "" {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString(`{"error": "Invalid token"}`)
			return
		}

		// Get user by token
		user, err := database.GetUserByToken(token)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString(`{"error": "Invalid token"}`)
			return
		}

		 // Check token expiration
		if time.Now().After(user.TokenExp) {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.SetBodyString(`{"error": "Token expired"}`)
			return
		}

		// Store the user ID in the context for later use
		ctx.SetUserValue("userID", user.ID)
		ctx.SetUserValue("isAdmin", user.IsAdmin)

		next(ctx)
	}
}

// AdminMiddleware checks if the authenticated user is an admin
func AdminMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		isAdmin, ok := ctx.UserValue("isAdmin").(bool)
		if !ok || !isAdmin {
			ctx.SetStatusCode(fasthttp.StatusForbidden)
			ctx.SetBodyString(`{"error": "Admin access required"}`)
			return
		}

		next(ctx)
	}
}