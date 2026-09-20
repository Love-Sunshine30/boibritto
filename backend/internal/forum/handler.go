package forum

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"boibritto/internal/apihttp"
	"boibritto/internal/app"
	"boibritto/internal/auth"
	"boibritto/internal/books"
	"boibritto/internal/profile"
)

func Mount(r chi.Router, a *app.App) {
	store := NewStore(a.DB)
	bookStore := books.NewStore(a.DB)
	profileStore := profile.NewStore(a.DB)
	svc := NewService(store, bookStore, profileStore)
	h := &handler{svc: svc}

	r.Get("/books/{id}/forum", h.list)
	r.Post("/books/{id}/forum", h.create)
	r.Patch("/forum/{postID}", h.update)
	r.Delete("/forum/{postID}", h.delete)
}

type handler struct{ svc *Service }

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	bookID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid book id"))
		return
	}
	var cursor *time.Time
	if c := r.URL.Query().Get("cursor"); c != "" {
		t, err := time.Parse(time.RFC3339, c)
		if err != nil {
			apihttp.RespondError(w, r, apihttp.ErrValidation("invalid cursor format"))
			return
		}
		cursor = &t
	}

	posts, next, err := h.svc.ListPosts(r.Context(), bookID, cursor)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, ListPostsResponse{Posts: posts, NextCursor: next})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
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
	var body CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	post, err := h.svc.CreatePost(r.Context(), bookID, user.ID, body.Body)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusCreated, post)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	postID, err := strconv.Atoi(chi.URLParam(r, "postID"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid post id"))
		return
	}
	var body UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid request body"))
		return
	}

	post, err := h.svc.UpdatePost(r.Context(), postID, user.ID, body.Body)
	if err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	apihttp.RespondJSON(w, http.StatusOK, post)
}

func (h *handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		apihttp.RespondError(w, r, apihttp.ErrInternal("user missing from context"))
		return
	}
	postID, err := strconv.Atoi(chi.URLParam(r, "postID"))
	if err != nil {
		apihttp.RespondError(w, r, apihttp.ErrValidation("invalid post id"))
		return
	}
	if err := h.svc.DeletePost(r.Context(), postID, user.ID); err != nil {
		apihttp.RespondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
