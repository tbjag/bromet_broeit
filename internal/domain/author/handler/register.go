package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/gmhafiz/go8/internal/domain/author/usecase"
	"github.com/gmhafiz/go8/internal/middleware"
)

func RegisterHTTPEndPoints(router *chi.Mux, validate *validator.Validate, useCase usecase.Author) *Handler {
	h := NewHandler(useCase, validate)

	router.Route("/api/v1/author", func(router chi.Router) {
		router.Post("/", h.Create)

		cacheGroup := router.Group(nil)
		cacheGroup.Use(middleware.CacheByURL)
		cacheGroup.Get("/", h.List)

		router.Get("/{id}", h.Get)
		router.Put("/{id}", h.Update)
		router.Delete("/{id}", h.Delete)
	})

	return h
}
