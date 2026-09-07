package messages

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/profile"
)

func Mount(r chi.Router, a *app.App, notifier Notifier) {
	store := NewStore(a.Firebase.Firestore)
	users := profile.NewStore(a.DB)
	svc := NewService(store, users, notifier)
	h := &handler{svc: svc}

	r.Get("/threads", h.list)
	r.Post("/threads/{id}/messages", h.send)
}

type handler struct{ svc *Service }

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	threads, err := h.svc.ListThreads(r.Context(), user.FirebaseUID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, threads)
}

func (h *handler) send(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	requestID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request id"))
		return
	}
	var body SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	resp, err := h.svc.SendMessage(r.Context(), requestID, user.ID, user.FirebaseUID, user.Name, body.Body)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusCreated, resp)
}
