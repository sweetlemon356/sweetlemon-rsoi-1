package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/app"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/domain"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/httpapi"
)

type fakePersonService struct {
	create func(context.Context, app.CreateCommand) (domain.Person, error)
	get    func(context.Context, int32) (domain.Person, error)
	list   func(context.Context) ([]domain.Person, error)
	update func(context.Context, int32, app.UpdateCommand) (domain.Person, error)
	delete func(context.Context, int32) error
}

func (f fakePersonService) Create(ctx context.Context, command app.CreateCommand) (domain.Person, error) {
	return f.create(ctx, command)
}

func (f fakePersonService) Get(ctx context.Context, id int32) (domain.Person, error) {
	return f.get(ctx, id)
}

func (f fakePersonService) List(ctx context.Context) ([]domain.Person, error) {
	return f.list(ctx)
}

func (f fakePersonService) Update(ctx context.Context, id int32, command app.UpdateCommand) (domain.Person, error) {
	return f.update(ctx, id, command)
}

func (f fakePersonService) Delete(ctx context.Context, id int32) error {
	return f.delete(ctx, id)
}

type healthyDatabase struct{}

func (healthyDatabase) Ping(context.Context) error { return nil }

// personResponse describes only the public JSON contract used in assertions.
// Tests deliberately do not depend on the handler's private DTO type.
type personResponse struct {
	ID      int32   `json:"id"`
	Name    string  `json:"name"`
	Age     *int32  `json:"age"`
	Address *string `json:"address"`
	Work    *string `json:"work"`
}

func TestCreatePerson(t *testing.T) {
	service := fakePersonService{
		create: func(_ context.Context, command app.CreateCommand) (domain.Person, error) {
			if command.Name != "Alice" || command.Age == nil || *command.Age != 31 {
				t.Fatalf("unexpected create command: %+v", command)
			}
			return domain.Person{ID: 42, Name: command.Name, Age: command.Age}, nil
		},
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/persons", strings.NewReader(`{"name":"Alice","age":31}`))
	response := httptest.NewRecorder()
	newTestRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusCreated)
	}
	if location := response.Header().Get("Location"); location != "/api/v1/persons/42" {
		t.Fatalf("Location: got %q", location)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", response.Body.String())
	}
}

func TestGetPerson(t *testing.T) {
	service := fakePersonService{
		get: func(_ context.Context, id int32) (domain.Person, error) {
			if id != 42 {
				t.Fatalf("id: got %d, want 42", id)
			}
			return testPerson(), nil
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/persons/42", nil)
	response := httptest.NewRecorder()
	newTestRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusOK)
	}
	var person personResponse
	decodeResponse(t, response, &person)
	if person.ID != 42 || person.Name != "Alice" || person.Age == nil || *person.Age != 31 {
		t.Fatalf("unexpected response: %+v", person)
	}
}

func TestListPersons(t *testing.T) {
	service := fakePersonService{
		list: func(context.Context) ([]domain.Person, error) {
			return []domain.Person{testPerson()}, nil
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/persons", nil)
	response := httptest.NewRecorder()
	newTestRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusOK)
	}
	var persons []personResponse
	decodeResponse(t, response, &persons)
	if len(persons) != 1 || persons[0].ID != 42 {
		t.Fatalf("unexpected response: %+v", persons)
	}
}

func TestUpdatePerson(t *testing.T) {
	service := fakePersonService{
		update: func(_ context.Context, id int32, command app.UpdateCommand) (domain.Person, error) {
			if id != 42 || command.Name == nil || *command.Name != "Bob" || command.Age != nil {
				t.Fatalf("unexpected update: id=%d command=%+v", id, command)
			}
			person := testPerson()
			person.Name = *command.Name
			return person, nil
		},
	}

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/persons/42", strings.NewReader(`{"name":"Bob"}`))
	response := httptest.NewRecorder()
	newTestRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusOK)
	}
	var person personResponse
	decodeResponse(t, response, &person)
	if person.Name != "Bob" || person.Age == nil || *person.Age != 31 {
		t.Fatalf("unexpected response: %+v", person)
	}
}

func TestDeletePerson(t *testing.T) {
	service := fakePersonService{
		delete: func(_ context.Context, id int32) error {
			if id != 42 {
				t.Fatalf("id: got %d, want 42", id)
			}
			return nil
		},
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/persons/42", nil)
	response := httptest.NewRecorder()
	newTestRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequestErrors(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		service fakePersonService
		status  int
	}{
		{
			name:   "invalid id",
			method: http.MethodGet,
			path:   "/api/v1/persons/not-a-number",
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown JSON field",
			method: http.MethodPost,
			path:   "/api/v1/persons",
			body:   `{"name":"Alice","unknown":true}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "null PATCH field",
			method: http.MethodPatch,
			path:   "/api/v1/persons/42",
			body:   `{"address":null}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/api/v1/persons/42",
			service: fakePersonService{
				get: func(context.Context, int32) (domain.Person, error) {
					return domain.Person{}, domain.ErrNotFound
				},
			},
			status: http.StatusNotFound,
		},
		{
			name:   "internal error",
			method: http.MethodGet,
			path:   "/api/v1/persons/42",
			service: fakePersonService{
				get: func(context.Context, int32) (domain.Person, error) {
					return domain.Person{}, errors.New("database failed")
				},
			},
			status: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			response := httptest.NewRecorder()
			newTestRouter(test.service).ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status: got %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("Content-Type: got %q", contentType)
			}
		})
	}
}

func newTestRouter(service httpapi.PersonService) http.Handler {
	return httpapi.NewRouter(httpapi.NewHandler(service), healthyDatabase{})
}

func testPerson() domain.Person {
	age := int32(31)
	address := "Main Street"
	work := "Example Inc."
	return domain.Person{
		ID:      42,
		Name:    "Alice",
		Age:     &age,
		Address: &address,
		Work:    &work,
	}
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
