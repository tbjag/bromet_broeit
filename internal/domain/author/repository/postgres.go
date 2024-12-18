package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"go.opentelemetry.io/otel"

	"github.com/gmhafiz/go8/ent/gen"
	entAuthor "github.com/gmhafiz/go8/ent/gen/author"
	"github.com/gmhafiz/go8/ent/gen/predicate"
	"github.com/gmhafiz/go8/internal/domain/author"
	"github.com/gmhafiz/go8/internal/domain/book"
	parseTime "github.com/gmhafiz/go8/internal/utility/time"
)

type repository struct {
	ent *gen.Client
}

//go:generate mirip -rm -out postgres_mock.go . Author Searcher
type Author interface {
	Create(ctx context.Context, a *author.CreateRequest) (*author.Schema, error)
	List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error)
	Read(ctx context.Context, id uint64) (*author.Schema, error)
	Update(ctx context.Context, toAuthor *author.UpdateRequest) (*author.Schema, error)
	Delete(ctx context.Context, authorID uint64) error
}

type Searcher interface {
	Search(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error)
}

func New(ent *gen.Client) *repository {
	return &repository{
		ent: ent,
	}
}

func (r *repository) Create(ctx context.Context, request *author.CreateRequest) (*author.Schema, error) {
	if request == nil {
		return nil, errors.New("request cannot be nil")
	}
	bulk := make([]*gen.BookCreate, len(request.Books))
	for i, b := range request.Books {
		bulk[i] = r.ent.Book.Create().
			SetTitle(b.Title).
			SetDescription(b.Description).
			SetPublishedDate(parseTime.Parse(b.PublishedDate))
	}
	books, err := r.ent.Book.CreateBulk(bulk...).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("author.repository.Create bulk books: %w", err)
	}

	create, err := r.ent.Author.Create().
		SetFirstName(request.FirstName).
		SetNillableMiddleName(&request.MiddleName).
		SetLastName(request.LastName).
		AddBooks(books...).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("author.repository.Create: %w", err)
	}

	// Both created_at and updated_at are created database-side instead of ent.
	// So ent does not return both.
	created, err := r.ent.Author.Get(ctx, create.ID)
	if err != nil {
		return nil, fmt.Errorf("author not found: %w", err)
	}

	var b []*book.Schema
	for _, i := range books {

		b = append(b, &book.Schema{
			ID:            i.ID,
			Title:         i.Title,
			PublishedDate: i.PublishedDate,
			ImageURL:      i.ImageURL,
			Description:   i.Description,
			CreatedAt:     i.CreatedAt,
			UpdatedAt:     i.UpdatedAt,
			//DeletedAt:     sql.NullTime{Time: *i.DeletedAt, Valid: true},
		})
	}

	resp := &author.Schema{
		ID:         created.ID,
		FirstName:  created.FirstName,
		MiddleName: created.MiddleName,
		LastName:   created.LastName,
		CreatedAt:  created.CreatedAt,
		UpdatedAt:  created.UpdatedAt,
		DeletedAt:  created.DeletedAt,
		Books:      b,
	}

	return resp, nil
}

func (r *repository) List(ctx context.Context, f *author.Filter) ([]*author.Schema, int, error) {
	tracer := otel.Tracer("")
	ctx, span := tracer.Start(ctx, "AuthorRepoList")
	defer span.End()

	// filter by first and last names, if exists
	var predicateUser []predicate.Author
	if f.FirstName != "" {
		predicateUser = append(predicateUser, entAuthor.FirstNameContainsFold(f.FirstName))
	}
	if f.MiddleName != "" {
		predicateUser = append(predicateUser, entAuthor.MiddleNameContainsFold(f.MiddleName))
	}
	if f.LastName != "" {
		predicateUser = append(predicateUser, entAuthor.LastNameContainsFold(f.LastName))
	}

	// sort by field
	orderFunc := authorOrder(f.Base.Sort)

	total, err := r.ent.Author.Query().
		Where(entAuthor.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("get total author records: %w", err)
	}

	authors, err := r.ent.Author.Query().
		WithBooks().
		Where(predicateUser...).
		Where(entAuthor.DeletedAtIsNil()).
		Limit(f.Base.Limit).
		Offset(f.Base.Offset).
		Order(orderFunc...).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("get author records: %w", err)
	}

	resp := make([]*author.Schema, 0)

	for _, a := range authors {

		books := make([]*book.Schema, 0)
		for _, b := range a.Edges.Books {
			books = append(books, &book.Schema{
				ID:            b.ID,
				Title:         b.Title,
				PublishedDate: b.PublishedDate,
				ImageURL:      b.ImageURL,
				Description:   b.Description,
				CreatedAt:     b.CreatedAt,
				UpdatedAt:     b.UpdatedAt,
			})
		}

		resp = append(resp, &author.Schema{
			ID:         a.ID,
			FirstName:  a.FirstName,
			MiddleName: a.MiddleName,
			LastName:   a.LastName,
			CreatedAt:  a.CreatedAt,
			UpdatedAt:  a.UpdatedAt,
			DeletedAt:  a.DeletedAt,
			Books:      books,
		})
	}

	return resp, total, err
}

func (r *repository) Read(ctx context.Context, id uint64) (*author.Schema, error) {
	found, err := r.ent.Author.Query().
		WithBooks().
		Where(entAuthor.ID(id)).
		Where(entAuthor.DeletedAtIsNil()).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving book: %w", err)
	}

	books := make([]*book.Schema, 0)

	for _, b := range found.Edges.Books {
		books = append(books, &book.Schema{
			ID:            b.ID,
			Title:         b.Title,
			PublishedDate: b.PublishedDate,
			ImageURL:      b.ImageURL,
			Description:   b.Description,
			CreatedAt:     b.CreatedAt,
			UpdatedAt:     b.UpdatedAt,
		})
	}

	return &author.Schema{
		ID:         found.ID,
		FirstName:  found.FirstName,
		MiddleName: found.MiddleName,
		LastName:   found.LastName,
		CreatedAt:  found.CreatedAt,
		UpdatedAt:  found.UpdatedAt,
		DeletedAt:  found.DeletedAt,
		Books:      books,
	}, err
}

func (r *repository) Update(ctx context.Context, a *author.UpdateRequest) (*author.Schema, error) {
	updated, err := r.ent.Author.UpdateOneID(a.ID).
		SetFirstName(a.FirstName).
		SetMiddleName(a.MiddleName).
		SetLastName(a.LastName).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	books := make([]*book.Schema, 0)
	for _, b := range updated.Edges.Books {
		books = append(books, &book.Schema{
			ID:            b.ID,
			Title:         b.Title,
			PublishedDate: b.PublishedDate,
			ImageURL:      b.ImageURL,
			Description:   b.Description,
			CreatedAt:     b.CreatedAt,
			UpdatedAt:     b.UpdatedAt,
		})
	}

	return &author.Schema{
		ID:         updated.ID,
		FirstName:  updated.FirstName,
		MiddleName: updated.MiddleName,
		LastName:   updated.LastName,
		CreatedAt:  updated.CreatedAt,
		UpdatedAt:  updated.UpdatedAt,
		DeletedAt:  updated.DeletedAt,
		Books:      books,
	}, nil
}

func (r *repository) Delete(ctx context.Context, authorID uint64) error {
	_, err := r.ent.Author.UpdateOneID(authorID).
		SetDeletedAt(time.Now()).
		Save(ctx)

	return err
}

func authorOrder(sorts map[string]string) []entAuthor.OrderOption {
	var orderFunc []entAuthor.OrderOption
	for col, ord := range sorts {
		switch col {
		case entAuthor.FieldFirstName:
			if ord == "ASC" {
				orderFunc = append(orderFunc, entAuthor.ByFirstName(sql.OrderAsc()))
			} else {
				orderFunc = append(orderFunc, entAuthor.ByFirstName(sql.OrderDesc()))
			}
		case entAuthor.FieldLastName:
			if ord == "ASC" {
				orderFunc = append(orderFunc, entAuthor.ByLastName(sql.OrderAsc()))
			} else {
				orderFunc = append(orderFunc, entAuthor.ByLastName(sql.OrderDesc()))
			}
		case entAuthor.FieldCreatedAt:
			if ord == "ASC" {
				orderFunc = append(orderFunc, entAuthor.ByCreatedAt(sql.OrderAsc()))
			} else {
				orderFunc = append(orderFunc, entAuthor.ByCreatedAt(sql.OrderDesc()))
			}
		case entAuthor.FieldUpdatedAt:
			if ord == "ASC" {
				orderFunc = append(orderFunc, entAuthor.ByUpdatedAt(sql.OrderAsc()))
			} else {
				orderFunc = append(orderFunc, entAuthor.ByUpdatedAt(sql.OrderDesc()))
			}
		}
	}

	return orderFunc
}
