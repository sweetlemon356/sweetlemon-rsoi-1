package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sweetlemon356/sweetlemon-rsoi-1/src/internal/domain"
)

const personColumns = "id, name, age, address, work"

type PersonRepository struct {
	pool *pgxpool.Pool
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewPersonRepository(pool *pgxpool.Pool) *PersonRepository {
	return &PersonRepository{pool: pool}
}

func (r *PersonRepository) Create(ctx context.Context, person domain.Person) (domain.Person, error) {
	query := `
		INSERT INTO persons (name, age, address, work)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + personColumns

	created, err := scanPerson(r.pool.QueryRow(ctx, query,
		person.Name,
		person.Age,
		person.Address,
		person.Work,
	))
	if err != nil {
		return domain.Person{}, fmt.Errorf("create person: %w", err)
	}
	return created, nil
}

func (r *PersonRepository) GetByID(ctx context.Context, id int32) (domain.Person, error) {
	query := `SELECT ` + personColumns + ` FROM persons WHERE id = $1`
	person, err := scanPerson(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Person{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Person{}, fmt.Errorf("get person: %w", err)
	}
	return person, nil
}

func (r *PersonRepository) List(ctx context.Context) ([]domain.Person, error) {
	query := `SELECT ` + personColumns + ` FROM persons ORDER BY id`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list persons: %w", err)
	}
	defer rows.Close()

	persons := make([]domain.Person, 0)
	for rows.Next() {
		person, scanErr := scanPerson(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan person: %w", scanErr)
		}
		persons = append(persons, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate persons: %w", err)
	}
	return persons, nil
}

func (r *PersonRepository) Update(ctx context.Context, id int32, changes domain.Changes) (domain.Person, error) {
	set := make([]string, 0, 4)
	args := []any{id}

	add := func(column string, value any) {
		args = append(args, value)
		set = append(set, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if changes.Name != nil {
		add("name", *changes.Name)
	}
	if changes.Age != nil {
		add("age", *changes.Age)
	}
	if changes.Address != nil {
		add("address", *changes.Address)
	}
	if changes.Work != nil {
		add("work", *changes.Work)
	}
	if len(set) == 0 {
		return domain.Person{}, errors.New("update person: no changes provided")
	}

	query := `UPDATE persons SET ` + strings.Join(set, ", ") + ` WHERE id = $1 RETURNING ` + personColumns
	person, err := scanPerson(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Person{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Person{}, fmt.Errorf("update person: %w", err)
	}
	return person, nil
}

func (r *PersonRepository) Delete(ctx context.Context, id int32) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM persons WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete person: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanPerson(row rowScanner) (domain.Person, error) {
	var person domain.Person
	err := row.Scan(
		&person.ID,
		&person.Name,
		&person.Age,
		&person.Address,
		&person.Work,
	)
	return person, err
}
