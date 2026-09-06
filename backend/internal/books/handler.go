package books

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
)

// Mount registers this domain's routes onto the given router, pulling its
// own dependencies off App. This is the pattern every domain package follows
// — router.go just calls Mount, never wires up a domain's internals itself.
func Mount(r chi.Router, a *app.App, profileChecker profileChecker) {
	store := NewStore(a.DB)
	svc := NewService(store, profileChecker)
	h := &handler{svc: svc}

	r.Get("/books", h.list)
	r.Get("/books/{id}", h.get)
	r.Post("/books", h.create)
	r.Patch("/books/{id}", h.update)
	r.Delete("/books/{id}", h.delete)
}

type handler struct {
	svc *Service
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	var cursor *time.Time
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339, c)
		if err != nil {
			apihttp.RespondError(w, r, apihttp.ErrValidation("invalid cursor format"))
			return
		}
		cursor = &t
	}

	filter := ListBooksFilter{
		Cursor: cursor,
		Query:  r.URL.Query().Get("q"),
		Genre:  r.URL.Query().Get("genre"),
	}

	books, err := h.svc.ListBooks(r.Context(), filter)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}

	resp := ListBooksResponse{Books: books}
	if len(books) == pageSize {
		resp.NextCursor = &books[len(books)-1].CreatedAt
	}
	apihttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}

	book, err := h.svc.GetBook(r.Context(), bookID)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}

	apihttp.RespondJSON(w, http.StatusOK, book)
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}

	var req CreateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	book, err := h.svc.CreateBook(r.Context(), user.ID, req)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusCreated, book)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}

	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}

	var req UpdateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	book, err := h.svc.UpdateBook(r.Context(), bookID, user.ID, req)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, book)
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}

	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}

	if err := h.svc.DeleteBook(r.Context(), bookID, user.ID); err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
