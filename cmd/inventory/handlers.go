package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type inventoryApp struct {
	repo *partRepository
}

func newInventoryApp(repo *partRepository) *inventoryApp {
	return &inventoryApp{
		repo: repo,
	}
}

func (a *inventoryApp) createPart(w http.ResponseWriter, r *http.Request) {
	var req CreatePartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "невалидный JSON", http.StatusBadRequest)
		return
	}

	part, err := a.repo.Create(r.Context(), req)
	if err != nil {
		slog.Error("не удалось создать деталь", slog.Any("ошибка", err))
		http.Error(w, "не удалось создать деталь", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(part)
}

func (a *inventoryApp) getPart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	part, err := a.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, errPartNotFound) {
			http.Error(w, errPartNotFound.Error(), http.StatusNotFound)
			return
		}

		slog.Error("не удалось получить деталь", slog.String("id", id), slog.Any("ошибка", err))
		http.Error(w, "не удалось получить деталь", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(part)
}

func (a *inventoryApp) listParts(w http.ResponseWriter, r *http.Request) {
	parts, err := a.repo.List(r.Context())
	if err != nil {
		slog.Error("не удалось получить список деталей", slog.Any("ошибка", err))
		http.Error(w, "не удалось получить список деталей", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(parts)
}
