package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"partipix/backend/internal/database"
	"time"
)

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type UserResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Avatar  string `json:"avatar"`
	IsAdmin bool   `json:"isAdmin"`
}

func generateToken(user *database.User) error {
	user.Token = uuid.New().String()
	user.TokenExp = time.Now().Add(7 * 24 * time.Hour) // Token expires in 7 days
	return database.UpdateUser(user)
}

func (r *Router) handleRefreshToken(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("userID").(string)

	user, err := database.GetUserByID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString(`{"error": "Invalid token"}`)
		return
	}

	if err := generateToken(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to refresh token"}`)
		return
	}

	response := AuthResponse{
		User: UserResponse{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Avatar:  user.Avatar,
			IsAdmin: user.IsAdmin,
		},
		Token: user.Token,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleSignup(ctx *fasthttp.RequestCtx) {
	var req SignupRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Check if user already exists
	_, err := database.GetUserByEmail(req.Email)
	if err == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Email already registered"}`)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to process password"}`)
		return
	}

	// Create user
	user := &database.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
		Avatar:   req.Avatar,
	}

	if err := database.CreateUser(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to create user"}`)
		return
	}

	// Generate token with expiration
	if err := generateToken(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to generate token"}`)
		return
	}

	response := AuthResponse{
		User: UserResponse{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Avatar:  user.Avatar,
			IsAdmin: user.IsAdmin,
		},
		Token: user.Token,
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleLogin(ctx *fasthttp.RequestCtx) {
	var req LoginRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	// Get user by email
	user, err := database.GetUserByEmail(req.Email)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString(`{"error": "Invalid email or password"}`)
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString(`{"error": "Invalid email or password"}`)
		return
	}

	// Generate new token with expiration
	if err := generateToken(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to generate token"}`)
		return
	}

	response := AuthResponse{
		User: UserResponse{
			ID:      user.ID,
			Name:    user.Name,
			Email:   user.Email,
			Avatar:  user.Avatar,
			IsAdmin: user.IsAdmin,
		},
		Token: user.Token,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}