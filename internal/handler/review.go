package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dimerin1/cloudtalk-review-system/internal/model"
	"github.com/dimerin1/cloudtalk-review-system/internal/service"
)

type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req model.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.FirstName == "" || req.LastName == "" {
		writeError(w, http.StatusBadRequest, "first_name and last_name are required")
		return
	}
	if req.ReviewText == "" {
		writeError(w, http.StatusBadRequest, "review_text is required")
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	review, err := h.svc.Create(r.Context(), productID, req)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	writeJSON(w, http.StatusCreated, review)
}

func (h *ReviewHandler) GetByProductID(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	reviews, err := h.svc.GetByProductID(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get reviews")
		return
	}

	writeJSON(w, http.StatusOK, reviews)
}

func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	var req model.UpdateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Rating != nil && (*req.Rating < 1 || *req.Rating > 5) {
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	review, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "review not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update review")
		return
	}

	writeJSON(w, http.StatusOK, review)
}

func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeError(w, http.StatusNotFound, "review not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete review")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
