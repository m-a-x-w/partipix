package api

import (
	"log"
	"os"
	"partipix/backend/api/middleware"
	"partipix/backend/internal/facial_recognition"
	"partipix/backend/internal/storage"
	"path"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
)

var faceService *facial_recognition.Service

func init() {
	var err error
	faceService, err = facial_recognition.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize facial recognition service: %v", err)
	}
}

type Router struct {
	router *fasthttprouter.Router
}

func NewRouter() *Router {
	r := &Router{
		router: fasthttprouter.New(),
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Add a global OPTIONS handler
	r.router.OPTIONS("/*path", r.wrapHandler(middleware.CORSMiddleware(dummyHandler)))

	// Auth routes (public)
	r.router.POST("/api/auth/signup", r.wrapHandler(middleware.CORSMiddleware(r.handleSignup)))
	r.router.POST("/api/auth/login", r.wrapHandler(middleware.CORSMiddleware(r.handleLogin)))
	r.router.POST("/api/auth/refresh", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleRefreshToken))))

	// User routes (protected)
	r.router.GET("/api/users/profile", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetProfile))))
	r.router.PUT("/api/users/profile", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleUpdateProfile))))
	r.router.POST("/api/users/reference-photos", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleUploadReferencePhotos))))
	r.router.DELETE("/api/users/reference-photos/:photoId", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleDeleteReferencePhoto))))
	r.router.POST("/api/users/reference-photos/verify", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleUploadReferencePhotosForVerification))))
	r.router.POST("/api/users/reference-photos/confirm", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleConfirmReferencePhoto))))

	// Event routes (protected)
	r.router.POST("/api/events", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleCreateEvent))))
	r.router.GET("/api/events", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetEvents))))
	r.router.POST("/api/events/join", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleJoinEvent))))
	r.router.GET("/api/events/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetEvent))))
	r.router.DELETE("/api/events/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleDeleteUserEvent))))
	r.router.PUT("/api/events/:id/status", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleUpdateEventStatus))))

	// Photo routes (protected)
	r.router.POST("/api/photos", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleUploadPhoto))))
	r.router.GET("/api/photos/event/:eventId", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetEventPhotos))))
	r.router.GET("/api/photos/user", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetUserPhotos))))
	r.router.GET("/api/photos/with-user", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleGetPhotosWithUser))))
	r.router.DELETE("/api/photos/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(r.handleDeletePhoto))))

	// Admin routes (protected + admin only)
	r.router.GET("/api/admin/stats", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleGetAdminStats)))))
	r.router.GET("/api/admin/users", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleListUsers)))))
	r.router.PUT("/api/admin/users/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleUpdateUser)))))
	r.router.DELETE("/api/admin/users/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleDeleteUser)))))
	r.router.GET("/api/admin/events", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleListEvents)))))
	r.router.PUT("/api/admin/events/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleUpdateEvent)))))
	r.router.DELETE("/api/admin/events/:id", r.wrapHandler(middleware.CORSMiddleware(middleware.AuthMiddleware(middleware.AdminMiddleware(r.handleDeleteEvent)))))

	// Serve static files
	r.router.GET("/uploads/*filepath", r.wrapHandler(middleware.CORSMiddleware(r.handleStaticFiles)))
	r.router.GET("/tmp/*filepath", r.wrapHandler(middleware.CORSMiddleware(r.handleTmpFiles)))
	r.router.GET("/faces/*filepath", r.wrapHandler(r.handleFaceCrops))
}

// dummyHandler is used for OPTIONS requests
func dummyHandler(ctx *fasthttp.RequestCtx) {}

// wrapHandler converts a fasthttp.RequestHandler to a fasthttprouter.Handle
func (r *Router) wrapHandler(h fasthttp.RequestHandler) fasthttprouter.Handle {
	return func(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
		// Store URL parameters in the context
		for _, param := range ps {
			ctx.SetUserValue(param.Key, param.Value)
		}
		h(ctx)
	}
}

func (r *Router) handleFaceCrops(ctx *fasthttp.RequestCtx) {
	filepath := ctx.UserValue("filepath").(string)
	fullPath := path.Join(storage.RootDir, storage.TmpDir, filepath)

	if !strings.HasSuffix(filepath, ".jpg") {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	ctx.SendFile(fullPath)
}

func (r *Router) handleStaticFiles(ctx *fasthttp.RequestCtx) {
	filepath := ctx.UserValue("filepath").(string)
	fullPath := path.Join(storage.RootDir, storage.UploadsDir, filepath)

	if !strings.HasSuffix(filepath, ".jpg") && !strings.HasSuffix(filepath, ".png") && !strings.HasSuffix(filepath, ".webp") {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	ctx.SendFile(fullPath)
}

func (r *Router) handleTmpFiles(ctx *fasthttp.RequestCtx) {
	filepath := ctx.UserValue("filepath").(string)
	fullPath := path.Join(storage.RootDir, filepath)

	if !strings.HasSuffix(filepath, ".jpg") {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	ctx.SendFile(fullPath)
}

func (r *Router) Handler() fasthttp.RequestHandler {
	// Create file server handler for all static files
	staticFS := &fasthttp.FS{
		Root:               ".", // Serve from current directory
		IndexNames:         []string{},
		GenerateIndexPages: false,
		AcceptByteRange:    true,
	}

	// Return a handler that checks paths and routes to appropriate handler
	return func(ctx *fasthttp.RequestCtx) {
		// Always set CORS headers for all responses
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

		path := string(ctx.Path())
		switch {
		case strings.HasPrefix(path, "/tmp/") || strings.HasPrefix(path, "/uploads/"):
			// Remove the leading slash before joining with Root
			cleanPath := strings.TrimPrefix(path, "/")
			fullPath := filepath.Join(staticFS.Root, cleanPath)

			// Check if file exists
			if _, err := os.Stat(fullPath); err != nil {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				return
			}

			// Verify file extension for security
			ext := strings.ToLower(filepath.Ext(fullPath))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
				ctx.SetStatusCode(fasthttp.StatusForbidden)
				return
			}

			ctx.SendFile(fullPath)
		default:
			r.router.Handler(ctx)
		}
	}
}
