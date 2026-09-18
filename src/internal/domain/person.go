package domain

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("person not found")

type Person struct {
	ID      int32
	Name    string
	Age     *int32
	Address *string
	Work    *string
}

type Changes struct {
	Name    *string
	Age     *int32
	Address *string
	Work    *string
}

type Repository interface {
	Create(ctx context.Context, person Person) (Person, error)
	GetByID(ctx context.Context, id int32) (Person, error)
	List(ctx context.Context) ([]Person, error)
	Update(ctx context.Context, id int32, changes Changes) (Person, error)
	Delete(ctx context.Context, id int32) error
}
