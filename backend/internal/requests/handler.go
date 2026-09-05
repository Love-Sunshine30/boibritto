package requests

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/books"
)

func Mount(r chi.Router, a *app.App, profileChecker profileChecker, notifier Notifier) {
	reqStore := NewStore(a.DB)
	bookStore := books.NewStore(a.DB) // requests imports books package here
	svc := NewService(a.DB, reqStore, bookStore, notifier, profileChecker, a.Logger.Logger)
	h := &handler{svc: svc}

	r.Post("/books/{id}/requests", h.create)
	r.Get("/requests/sent", h.listSent)
	r.Get("/requests/incoming", h.listIncoming)
	r.Patch("/requests/{id}", h.updateStatus)
	r.Post("/requests/{id}/confirm", h.confirmHandoff)
	r.Post("/requests/{id}/return", h.markReturned)
}

type handler struct {
	svc *Service
}

func currentUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return 0, false
	}
	return user.ID, true
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}

	var body CreateRequestBody

	// extract the message from request body
	_ = json.NewDecoder(r.Body).Decode(&body)

	book, err := h.svc.bookStore.GetBookByID(r.Context(), bookID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}

	resp, err := h.svc.CreateRequest(r.Context(), bookID, requesterID, book.OwnerID, body.Message)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusCreated, resp)
}

func (h *handler) listSent(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	reqs, err := h.svc.ListSent(r.Context(), userID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, reqs)
}

func (h *handler) listIncoming(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	reqs, err := h.svc.ListIncoming(r.Context(), userID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, reqs)
}

func (h *handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	requestID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request id"))
		return
	}

	var body UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	resp, err := h.svc.UpdateStatus(r.Context(), requestID, ownerID, body.Status)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *handler) confirmHandoff(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	requestID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request id"))
		return
	}

	resp, err := h.svc.ConfirmHandoff(r.Context(), requestID, userID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *handler) markReturned(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	requestID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request id"))
		return
	}

	resp, err := h.svc.MarkReturned(r.Context(), requestID, ownerID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}
