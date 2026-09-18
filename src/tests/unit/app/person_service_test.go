package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/app"
	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/domain"
)

type fakeRepository struct {
	create func(context.Context, domain.Person) (domain.Person, error)
	update func(context.Context, int32, domain.Changes) (domain.Person, error)
}

func (f fakeRepository) Create(ctx context.Context, person domain.Person) (domain.Person, error) {
	return f.create(ctx, person)
}

func (fakeRepository) GetByID(context.Context, int32) (domain.Person, error) {
	panic("unexpected call")
}

func (fakeRepository) List(context.Context) ([]domain.Person, error) {
	panic("unexpected call")
}

func (f fakeRepository) Update(ctx context.Context, id int32, changes domain.Changes) (domain.Person, error) {
	return f.update(ctx, id, changes)
}

func (fakeRepository) Delete(context.Context, int32) error {
	panic("unexpected call")
}

func TestCreateValidatesAndNormalizesInput(t *testing.T) {
	repository := fakeRepository{
		create: func(_ context.Context, person domain.Person) (domain.Person, error) {
			if person.Name != "Alice" {
				t.Fatalf("name: got %q, want %q", person.Name, "Alice")
			}
			person.ID = 1
			return person, nil
		},
	}

	created, err := app.NewService(repository).Create(context.Background(), app.CreateCommand{Name: "  Alice  "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("id: got %d, want 1", created.ID)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	age := int32(-1)
	repository := fakeRepository{
		create: func(context.Context, domain.Person) (domain.Person, error) {
			t.Fatal("repository must not be called for invalid input")
			return domain.Person{}, nil
		},
	}

	_, err := app.NewService(repository).Create(context.Background(), app.CreateCommand{Name: " ", Age: &age})
	var validationErr *app.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error: got %v, want ValidationError", err)
	}
	if len(validationErr.Fields) != 2 {
		t.Fatalf("validation fields: got %+v", validationErr.Fields)
	}
}

func TestUpdatePassesOnlyProvidedChanges(t *testing.T) {
	name := "  Bob  "
	repository := fakeRepository{
		update: func(_ context.Context, id int32, changes domain.Changes) (domain.Person, error) {
			if id != 7 || changes.Name == nil || *changes.Name != "Bob" {
				t.Fatalf("unexpected update: id=%d changes=%+v", id, changes)
			}
			if changes.Age != nil || changes.Address != nil || changes.Work != nil {
				t.Fatalf("unprovided fields must remain nil: %+v", changes)
			}
			return domain.Person{ID: id, Name: *changes.Name}, nil
		},
	}

	updated, err := app.NewService(repository).Update(context.Background(), 7, app.UpdateCommand{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Bob" {
		t.Fatalf("name: got %q, want Bob", updated.Name)
	}
}

func TestUpdateRejectsEmptyPatch(t *testing.T) {
	repository := fakeRepository{
		update: func(context.Context, int32, domain.Changes) (domain.Person, error) {
			t.Fatal("repository must not be called for invalid input")
			return domain.Person{}, nil
		},
	}

	_, err := app.NewService(repository).Update(context.Background(), 7, app.UpdateCommand{})
	var validationErr *app.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Fields["body"] == "" {
		t.Fatalf("error: got %v, want body ValidationError", err)
	}
}
