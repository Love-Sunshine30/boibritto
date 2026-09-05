package profile

import (
	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/books"
	"boibritto/internal/requests"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func Mount(r chi.Router, a *app.App) {
	store := NewStore(a.DB)
	bookStore := books.NewStore(a.DB)
	reqStore := requests.NewStore(a.DB)
	svc := NewService(store, bookStore, reqStore)
	h := &handler{svc: svc}

	r.Get("/me", h.getOwnProfile)
	r.Patch("/me", h.updateProfile)
	r.Get("/users/{id}", h.getPublicProfile)
}

type handler struct {
	svc *Service
}

func (h *handler) getOwnProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	resp, err := h.svc.GetOwnProfile(r.Context(), user.ID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *handler) getPublicProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid user id"))
		return
	}
	resp, err := h.svc.GetPublicProfile(r.Context(), userID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))

		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	if err := h.svc.UpdateProfile(r.Context(), user.ID, req); err != nil {
		apihttp.RespondError(w, r, err)
		return
	}

	updated, err := h.svc.store.GetUserByID(r.Context(), user.ID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, toUserResponse(updated))
}
