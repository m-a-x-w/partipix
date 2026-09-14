package middleware

import (
	"github.com/valyala/fasthttp"
)

// HTTPSMiddleware enforces HTTPS connections
func HTTPSMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Add security headers
		ctx.Response.Header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		ctx.Response.Header.Set("X-Frame-Options", "DENY")
		ctx.Response.Header.Set("X-XSS-Protection", "1; mode=block")
		ctx.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Response.Header.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:")

		// Check if the connection is secure
		if !ctx.IsTLS() {
			ctx.SetStatusCode(fasthttp.StatusPermanentRedirect)
			redirectURL := "https://" + string(ctx.Host()) + string(ctx.RequestURI())
			ctx.Response.Header.Set("Location", redirectURL)
			return
		}

		next(ctx)
	}
}
