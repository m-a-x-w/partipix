package api

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"partipix/backend/internal/database"
	"time"
)

type AdminStats struct {
	TotalUsers     int64 `json:"totalUsers"`
	TotalEvents    int64 `json:"totalEvents"`
	TotalPhotos    int64 `json:"totalPhotos"`
	ActiveEvents   int64 `json:"activeEvents"`
}

type UpdateUserRequest struct {
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	IsAdmin bool   `json:"isAdmin,omitempty"`
}

type UpdateEventRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Date        string `json:"date,omitempty"`
	Location    string `json:"location,omitempty"`
}

func (r *Router) handleGetAdminStats(ctx *fasthttp.RequestCtx) {
	var stats AdminStats

	// Get total users
	if count, err := database.CountUsers(); err == nil {
		stats.TotalUsers = count
	}

	// Get total events
	if count, err := database.CountEvents(); err == nil {
		stats.TotalEvents = count
	}

	// Get total photos
	if count, err := database.CountPhotos(); err == nil {
		stats.TotalPhotos = count
	}

	// Get active events (events that haven't passed yet)
	if count, err := database.CountActiveEvents(); err == nil {
		stats.ActiveEvents = count
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(stats)
}

func (r *Router) handleListUsers(ctx *fasthttp.RequestCtx) {
	users, err := database.ListUsers()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch users"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(users)
}

func (r *Router) handleUpdateUser(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("id").(string)

	var req UpdateUserRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "User not found"}`)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	user.IsAdmin = req.IsAdmin

	if err := database.UpdateUser(user); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to update user"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(user)
}

func (r *Router) handleDeleteUser(ctx *fasthttp.RequestCtx) {
	userID := ctx.UserValue("id").(string)

	if err := database.DeleteUser(userID); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete user"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"message": "User deleted successfully"}`)
}

func (r *Router) handleListEvents(ctx *fasthttp.RequestCtx) {
	events, err := database.ListEvents()
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch events"}`)
		return
	}

	response := make([]EventResponse, len(events))
	for i, event := range events {
		// Get participants with user info
		participants, users, err := database.GetEventParticipantsWithUsers(event.ID)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
			return
		}

		guestResponses := make([]Guest, len(participants))
		for j, p := range participants {
			userName := "Unknown User"
			if user, ok := users[p.UserID]; ok {
				userName = user.Name
			}
			guestResponses[j] = Guest{
				Name:   userName,
				Status: p.Status,
				IsHost: p.UserID == event.CreatedBy,
			}
		}

		response[i] = EventResponse{
			ID:          event.ID,
			Title:       event.Title,
			Description: event.Description,
			Date:        event.Date.Format("2006-01-02T15:04:05Z07:00"),
			Location:    event.Location,
			Participants: len(participants),
			Code:        event.Code,
			Guests:      guestResponses,
		}
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleUpdateEvent(ctx *fasthttp.RequestCtx) {
	eventID := ctx.UserValue("id").(string)

	var req UpdateEventRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error": "Invalid request body"}`)
		return
	}

	event, err := database.GetEventByID(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(`{"error": "Event not found"}`)
		return
	}

	// Update fields if provided
	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.Location != "" {
		event.Location = req.Location
	}
	if req.Date != "" {
		date, err := time.Parse("2006-01-02T15:04:05Z07:00", req.Date)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString(`{"error": "Invalid date format"}`)
			return
		}
		event.Date = date
	}

	if err := database.UpdateEvent(event); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to update event"}`)
		return
	}

	// Get updated event details for response
	participants, users, err := database.GetEventParticipantsWithUsers(eventID)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to fetch participants"}`)
		return
	}

	guestResponses := make([]Guest, len(participants))
	for i, p := range participants {
		userName := "Unknown User"
		if user, ok := users[p.UserID]; ok {
			userName = user.Name
		}
		guestResponses[i] = Guest{
			Name:   userName,
			Status: p.Status,
			IsHost: p.UserID == event.CreatedBy,
		}
	}

	response := EventResponse{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		Date:        event.Date.Format("2006-01-02T15:04:05Z07:00"),
		Location:    event.Location,
		Participants: len(participants),
		Code:        event.Code,
		Guests:      guestResponses,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	json.NewEncoder(ctx).Encode(response)
}

func (r *Router) handleDeleteEvent(ctx *fasthttp.RequestCtx) {
	eventID := ctx.UserValue("id").(string)

	if err := database.DeleteEvent(eventID); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error": "Failed to delete event"}`)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"message": "Event deleted successfully"}`)
}