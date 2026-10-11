package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type inventoryApp struct {
	store *memoryStore
}

func newInventoryApp(store *memoryStore) *inventoryApp {
	return &inventoryApp{
		store: store,
	}
}

func (a *inventoryApp) createPart(w http.ResponseWriter, r *http.Request) {
	var req CreatePartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "невалидный JSON", http.StatusBadRequest)
		return
	}

	part := a.store.Create(req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(part)
}

func (a *inventoryApp) getPart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	part, err := a.store.GetByID(id)
	if err != nil {
		http.Error(w, errPartNotFound.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(part)
}

func (a *inventoryApp) listParts(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a.store.List())
}
