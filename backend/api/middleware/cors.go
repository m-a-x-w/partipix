package middleware

import (
	"github.com/valyala/fasthttp"
)

// CORSMiddleware adds CORS headers to allow cross-origin requests
func CORSMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Set CORS headers to allow development server
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "http://localhost:5173")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin, X-Requested-With")
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		ctx.Response.Header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")

		// Handle preflight requests
		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusOK)
			return
		}

		// Call the next handler
		next(ctx)
	}
}
