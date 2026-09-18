package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/domain"
)

type Service struct {
	repository domain.Repository
}

type CreateCommand struct {
	Name    string
	Age     *int32
	Address *string
	Work    *string
}

type UpdateCommand struct {
	Name    *string
	Age     *int32
	Address *string
	Work    *string
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

func NewService(repository domain.Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, command CreateCommand) (domain.Person, error) {
	command.Name = strings.TrimSpace(command.Name)
	if validationErr := validateCreate(command); validationErr != nil {
		return domain.Person{}, validationErr
	}

	return s.repository.Create(ctx, domain.Person{
		Name:    command.Name,
		Age:     command.Age,
		Address: command.Address,
		Work:    command.Work,
	})
}

func (s *Service) Get(ctx context.Context, id int32) (domain.Person, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]domain.Person, error) {
	return s.repository.List(ctx)
}

func (s *Service) Update(ctx context.Context, id int32, command UpdateCommand) (domain.Person, error) {
	if command.Name != nil {
		trimmed := strings.TrimSpace(*command.Name)
		command.Name = &trimmed
	}
	if validationErr := validateUpdate(command); validationErr != nil {
		return domain.Person{}, validationErr
	}

	return s.repository.Update(ctx, id, domain.Changes{
		Name:    command.Name,
		Age:     command.Age,
		Address: command.Address,
		Work:    command.Work,
	})
}

func (s *Service) Delete(ctx context.Context, id int32) error {
	return s.repository.Delete(ctx, id)
}

func validateCreate(command CreateCommand) *ValidationError {
	fields := make(map[string]string)
	if command.Name == "" {
		fields["name"] = "must not be empty"
	}
	validateAge(command.Age, fields)

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

func validateUpdate(command UpdateCommand) *ValidationError {
	fields := make(map[string]string)
	if command.Name == nil && command.Age == nil && command.Address == nil && command.Work == nil {
		fields["body"] = "must contain at least one field"
	}
	if command.Name != nil && *command.Name == "" {
		fields["name"] = "must not be empty"
	}
	validateAge(command.Age, fields)

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

func validateAge(age *int32, fields map[string]string) {
	if age != nil && *age < 0 {
		fields["age"] = fmt.Sprintf("must be non-negative, got %d", *age)
	}
}
