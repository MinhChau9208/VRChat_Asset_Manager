// Package category manages the two-level asset category tree.
package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound       = errors.New("category not found")
	ErrDuplicateName  = errors.New("a category with this name already exists")
	ErrParentNotFound = errors.New("parent category not found")
	ErrTooDeep        = errors.New("categories can only be nested one level deep")
	ErrHasChildren    = errors.New("category has subcategories; move or delete them first")
)

// Category is one node of the tree. ParentID is nil for top-level categories.
type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	ParentID  *int64    `json:"parent_id"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// Input is the body for creating or updating a category.
type Input struct {
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	SortOrder *int   `json:"sort_order"`
}

// Repository handles category persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a category repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List returns all categories in display order: each top-level category
// followed by its children.
func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.parent_id, c.sort_order, c.created_at
		FROM categories c
		LEFT JOIN categories p ON c.parent_id = p.id
		ORDER BY
			COALESCE(p.sort_order, c.sort_order),
			COALESCE(p.id, c.id),
			c.parent_id IS NOT NULL,
			c.sort_order,
			c.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// Get returns one category by id.
func (r *Repository) Get(ctx context.Context, id int64) (*Category, error) {
	var c Category
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, parent_id, sort_order, created_at FROM categories WHERE id = ?", id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.SortOrder, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query category: %w", err)
	}
	return &c, nil
}

// validate checks name uniqueness and that the parent keeps the tree two levels deep.
// selfID is 0 when creating.
func (r *Repository) validate(ctx context.Context, selfID int64, name string, parentID *int64) error {
	var clash int64
	err := r.db.QueryRowContext(ctx,
		"SELECT id FROM categories WHERE name = ? COLLATE NOCASE AND id != ?", name, selfID,
	).Scan(&clash)
	if err == nil {
		return ErrDuplicateName
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check category name: %w", err)
	}

	if parentID == nil {
		return nil
	}
	if *parentID == selfID {
		return ErrTooDeep
	}
	parent, err := r.Get(ctx, *parentID)
	if errors.Is(err, ErrNotFound) {
		return ErrParentNotFound
	}
	if err != nil {
		return err
	}
	if parent.ParentID != nil {
		return ErrTooDeep
	}
	if selfID != 0 {
		hasChildren, err := r.hasChildren(ctx, selfID)
		if err != nil {
			return err
		}
		if hasChildren {
			return ErrTooDeep
		}
	}
	return nil
}

func (r *Repository) hasChildren(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM categories WHERE parent_id = ?)", id,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check subcategories: %w", err)
	}
	return exists, nil
}

// Create inserts a category. Without sort_order it is placed last among its siblings.
func (r *Repository) Create(ctx context.Context, in Input) (*Category, error) {
	name := strings.TrimSpace(in.Name)
	if err := r.validate(ctx, 0, name, in.ParentID); err != nil {
		return nil, err
	}

	var sortOrder int
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	} else {
		err := r.db.QueryRowContext(ctx,
			"SELECT COALESCE(MAX(sort_order), 0) + 10 FROM categories WHERE parent_id IS ? AND sort_order < 999",
			in.ParentID,
		).Scan(&sortOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to compute sort order: %w", err)
		}
	}

	res, err := r.db.ExecContext(ctx,
		"INSERT INTO categories (name, parent_id, sort_order) VALUES (?, ?, ?)",
		name, in.ParentID, sortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert category: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get category id: %w", err)
	}
	return r.Get(ctx, id)
}

// Update renames, re-parents and/or reorders a category. Omitting sort_order keeps it.
func (r *Repository) Update(ctx context.Context, id int64, in Input) (*Category, error) {
	existing, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(in.Name)
	if err := r.validate(ctx, id, name, in.ParentID); err != nil {
		return nil, err
	}

	sortOrder := existing.SortOrder
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}

	_, err = r.db.ExecContext(ctx,
		"UPDATE categories SET name = ?, parent_id = ?, sort_order = ? WHERE id = ?",
		name, in.ParentID, sortOrder, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}
	return r.Get(ctx, id)
}

// Delete removes a category without subcategories. Its assets become uncategorized.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	if _, err := r.Get(ctx, id); err != nil {
		return err
	}
	hasChildren, err := r.hasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrHasChildren
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}
