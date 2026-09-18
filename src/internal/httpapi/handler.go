package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/app"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/domain"
)

const maxRequestBodyBytes = 1 << 20

type PersonService interface {
	Create(ctx context.Context, command app.CreateCommand) (domain.Person, error)
	Get(ctx context.Context, id int32) (domain.Person, error)
	List(ctx context.Context) ([]domain.Person, error)
	Update(ctx context.Context, id int32, command app.UpdateCommand) (domain.Person, error)
	Delete(ctx context.Context, id int32) error
}

type Handler struct {
	service PersonService
}

type createPersonRequest struct {
	Name    string  `json:"name"`
	Age     *int32  `json:"age"`
	Address *string `json:"address"`
	Work    *string `json:"work"`
}

type updatePersonRequest struct {
	Name    optionalField[string] `json:"name"`
	Age     optionalField[int32]  `json:"age"`
	Address optionalField[string] `json:"address"`
	Work    optionalField[string] `json:"work"`
}

type optionalField[T any] struct {
	Value T
	Set   bool
	Null  bool
}

func (f *optionalField[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		f.Null = true
		return nil
	}
	return json.Unmarshal(data, &f.Value)
}

func (f optionalField[T]) pointer() *T {
	if !f.Set || f.Null {
		return nil
	}
	return &f.Value
}

type personResponse struct {
	ID      int32   `json:"id"`
	Name    string  `json:"name"`
	Age     *int32  `json:"age,omitempty"`
	Address *string `json:"address,omitempty"`
	Work    *string `json:"work,omitempty"`
}

type errorResponse struct {
	Message string `json:"message"`
}

type validationErrorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors"`
}

func NewHandler(service PersonService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request createPersonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, validationErrorResponse{
			Message: "invalid request body",
			Errors:  map[string]string{"body": err.Error()},
		})
		return
	}

	created, err := h.service.Create(r.Context(), app.CreateCommand{
		Name:    request.Name,
		Age:     request.Age,
		Address: request.Address,
		Work:    request.Work,
	})
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/persons/%d", created.ID))
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	person, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPersonResponse(person))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	persons, err := h.service.List(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	response := make([]personResponse, 0, len(persons))
	for _, person := range persons {
		response = append(response, toPersonResponse(person))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var request updatePersonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, validationErrorResponse{
			Message: "invalid request body",
			Errors:  map[string]string{"body": err.Error()},
		})
		return
	}
	if nullErrors := request.nullErrors(); len(nullErrors) != 0 {
		writeJSON(w, http.StatusBadRequest, validationErrorResponse{
			Message: "validation failed",
			Errors:  nullErrors,
		})
		return
	}

	updated, err := h.service.Update(r.Context(), id, app.UpdateCommand{
		Name:    request.Name.pointer(),
		Age:     request.Age.pointer(),
		Address: request.Address.pointer(),
		Work:    request.Work.pointer(),
	})
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPersonResponse(updated))
}

func (r updatePersonRequest) nullErrors() map[string]string {
	errors := make(map[string]string)
	fields := []struct {
		name string
		null bool
	}{
		{name: "name", null: r.Name.Null},
		{name: "age", null: r.Age.Null},
		{name: "address", null: r.Address.Null},
		{name: "work", null: r.Work.Null},
	}
	for _, field := range fields {
		if field.null {
			errors[field.name] = "must not be null"
		}
	}
	return errors
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *app.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, validationErrorResponse{
			Message: "validation failed",
			Errors:  validationErr.Fields,
		})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Message: "person not found"})
	default:
		slog.ErrorContext(r.Context(), "request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Message: "internal server error"})
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, validationErrorResponse{
			Message: "validation failed",
			Errors:  map[string]string{"id": "must be a positive integer"},
		})
		return 0, false
	}
	return int32(id), true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("body must contain a single JSON object")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func toPersonResponse(person domain.Person) personResponse {
	return personResponse{
		ID:      person.ID,
		Name:    person.Name,
		Age:     person.Age,
		Address: person.Address,
		Work:    person.Work,
	}
}
