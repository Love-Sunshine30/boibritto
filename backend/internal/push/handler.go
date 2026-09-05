package push

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
)

type subscribeRequest struct {
	Platform string `json:"platform"`
	FCMToken string `json:"fcm_token"`
}

type unsubscribeRequest struct {
	FCMToken string `json:"fcm_token"`
}

func Mount(r chi.Router, a *app.App, store *Store) {
	h := &handler{store: store}
	r.Post("/push/subscribe", h.subscribe)
	r.Post("/push/unsubscribe", h.unsubscribe)
}

type handler struct {
	store *Store
}

func (h *handler) subscribe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}

	var body subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}
	if body.Platform != "web" && body.Platform != "android" {
		apihttp.RespondError(w, r, apihttp.ErrValidation("platform must be 'web' or 'android'"))
		return
	}
	if body.FCMToken == "" {
		apihttp.RespondError(w, r, apihttp.ErrValidation("fcm_token is required"))
		return
	}

	if err := h.store.Subscribe(r.Context(), user.ID, body.Platform, body.FCMToken); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrInternal("failed to save subscription"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) unsubscribe(w http.ResponseWriter, r *http.Request) {
	var body unsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}
	if err := h.store.Unsubscribe(r.Context(), body.FCMToken); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrInternal("failed to remove subscription"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
