package admin

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

func Mount(r chi.Router, a *app.App) {
	bookStore := books.NewStore(a.DB)
	svc := NewService(bookStore)
	h := &handler{svc: svc}

	// RequireAdmin scopes this whole sub-router — every route mounted
	// below is admin-only, not just this one endpoint. Adding a second
	// admin route later doesn't require remembering to re-add the check.
	r.Route("/admin", func(admin chi.Router) {
		admin.Use(auth.RequireAdmin(a.Logger.Logger))
		admin.Patch("/books/{id}/cover", h.setCover)
	})
}

type handler struct{ svc *Service }

func (h *handler) setCover(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}
	var body SetCoverRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	updated, err := h.svc.SetBookCover(r.Context(), bookID, body.CoverURL)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, updated)
}
