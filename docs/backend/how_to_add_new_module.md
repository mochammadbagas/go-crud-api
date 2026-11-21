# Step-by-Step Guide: Adding a New Module to the Starter Kit

**Document Type:** Practical Extensibility & Implementation Guide  
**Target Audience:** Developers building real-world applications using this Starter Kit  
**Example Feature:** Building a `products` catalog module with CRUD operations, ULID keys, RBAC guards, and multi-input validation

---

## 1. Overview of the Development Workflow

Adding a new domain entity or feature in this starter kit follows the **Clean Architecture** flow from data storage up to HTTP routing:

```text
[ Step 1: Migration ] ──► Write SQL DDL in db/migrations/
            │
            ▼
[ Step 2: SQLC ]      ──► Write queries in db/queries/, run 'sqlc generate'
            │
            ▼
[ Step 3: Model/DTO ] ──► Define domain struct, input DTOs, and Validate() in internal/models/
            │
            ▼
[ Step 4: Repository] ──► Create repository interface & SQLC wrapper in internal/repository/
            │
            ▼
[ Step 5: Service ]   ──► Implement business rules & ULID assignment in internal/service/
            │
            ▼
[ Step 6: Handlers ]  ──► Decode JSON, run validation, format responses in internal/handlers/
            │
            ▼
[ Step 7: Routes ]    ──► Register routes and RBAC permission guards in internal/routes/
            │
            ▼
[ Step 8: Wiring ]    ──► Instantiate & inject dependencies in cmd/server/main.go
```

Below is the complete walkthrough implementing a sample `products` feature.

---

## Step 1: Database Migration (DDL)

Create sequential migration files in `db/migrations/`:

### `db/migrations/000003_create_products.up.sql`

```sql
CREATE TABLE IF NOT EXISTS products (
    id CHAR(26) NOT NULL,
    name VARCHAR(150) NOT NULL,
    sku VARCHAR(50) NOT NULL,
    price BIGINT NOT NULL DEFAULT 0,
    stock INT NOT NULL DEFAULT 0,
    description TEXT NULL,
    image_url VARCHAR(500) NULL,
    status ENUM('active', 'archived') NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_products_sku (sku),
    INDEX idx_products_status (status),
    INDEX idx_products_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

### `db/migrations/000003_create_products.down.sql`

```sql
DROP TABLE IF EXISTS products;
```

_Note: The migration files will be automatically detected and executed on startup because `db/embed.go` embeds all files matching `migrations/_.sql`.\*

---

## Step 2: SQL Queries & SQLC Code Generation

Create raw SQL queries with SQLC annotations in `db/queries/products.sql`:

```sql
-- name: CreateProduct :exec
INSERT INTO products (
    id, name, sku, price, stock, description, image_url, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProductByID :one
SELECT id, name, sku, price, stock, description, image_url, status, created_at, updated_at
FROM products
WHERE id = ? LIMIT 1;

-- name: GetProductBySKU :one
SELECT id, name, sku, price, stock, description, image_url, status, created_at, updated_at
FROM products
WHERE sku = ? LIMIT 1;

-- name: ListActiveProducts :many
SELECT id, name, sku, price, stock, description, image_url, status, created_at, updated_at
FROM products
WHERE status = 'active'
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CountActiveProducts :one
SELECT COUNT(*) FROM products WHERE status = 'active';

-- name: UpdateProduct :exec
UPDATE products
SET name = ?, price = ?, stock = ?, description = ?, image_url = ?, status = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = ?;
```

Now execute the SQLC compiler from your terminal:

```bash
sqlc generate
```

SQLC generates type-safe Go structs and query methods inside `internal/repository/sqlc/`.

---

## Step 3: Domain Models & Input DTOs with Validation

Create `internal/models/product.go`:

```go
package models

import (
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku"`
	Price       int64     `json:"price"`
	Stock       int32     `json:"stock"`
	Description *string   `json:"description,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductInput struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Price       int64   `json:"price"`
	Stock       int32   `json:"stock"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
}

func (i CreateProductInput) Validate() error {
	errs := make(utils.ValidationErrors)

	if strings.TrimSpace(i.Name) == "" {
		errs.Add("name", utils.ErrCodeRequired, "name is required")
	} else if len(i.Name) < 3 {
		errs.Add("name", utils.ErrCodeTooShort, "name must be at least 3 characters long")
	} else if len(i.Name) > 150 {
		errs.Add("name", utils.ErrCodeTooLong, "name must not exceed 150 characters")
	}

	if strings.TrimSpace(i.SKU) == "" {
		errs.Add("sku", utils.ErrCodeRequired, "sku is required")
	} else if len(i.SKU) < 3 {
		errs.Add("sku", utils.ErrCodeTooShort, "sku must be at least 3 characters long")
	} else if len(i.SKU) > 50 {
		errs.Add("sku", utils.ErrCodeTooLong, "sku must not exceed 50 characters")
	}

	if i.Price < 0 {
		errs.Add("price", utils.ErrCodeInvalidValue, "price cannot be negative")
	}

	if i.Stock < 0 {
		errs.Add("stock", utils.ErrCodeInvalidValue, "stock cannot be negative")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}

type UpdateProductInput struct {
	Name        string  `json:"name"`
	Price       int64   `json:"price"`
	Stock       int32   `json:"stock"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Status      string  `json:"status"`
}

func (i UpdateProductInput) Validate() error {
	errs := make(utils.ValidationErrors)

	if strings.TrimSpace(i.Name) == "" {
		errs.Add("name", utils.ErrCodeRequired, "name is required")
	}
	if i.Price < 0 {
		errs.Add("price", utils.ErrCodeInvalidValue, "price cannot be negative")
	}
	if i.Stock < 0 {
		errs.Add("stock", utils.ErrCodeInvalidValue, "stock cannot be negative")
	}
	if i.Status != "active" && i.Status != "archived" {
		errs.Add("status", utils.ErrCodeInvalidStatus, "status must be either active or archived")
	}

	if errs.HasErrors() {
		return errs
	}
	return nil
}
```

> [!NOTE]
> Always use standard validation error code constants (`utils.ErrCodeRequired`, `utils.ErrCodeTooShort`, `utils.ErrCodeTooLong`, `utils.ErrCodeInvalidFormat`, `utils.ErrCodeInvalidValue`, etc.) when calling `errs.Add(...)`. For the full list of standardized codes and their meanings, see [System Architecture](../overview/system_architecture.md#field-level-validation-error-codes-dictionary).

---

## Step 4: Repository Layer

Create `internal/repository/product_repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
)

type ProductRepository interface {
	Create(ctx context.Context, p *models.Product) error
	GetByID(ctx context.Context, id string) (*models.Product, error)
	GetBySKU(ctx context.Context, sku string) (*models.Product, error)
	ListActive(ctx context.Context, limit, offset int32) ([]*models.Product, int64, error)
	Update(ctx context.Context, p *models.Product) error
	Delete(ctx context.Context, id string) error
}

type productRepository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r *productRepository) Create(ctx context.Context, p *models.Product) error {
	var desc, img sql.NullString
	if p.Description != nil {
		desc = sql.NullString{String: *p.Description, Valid: true}
	}
	if p.ImageURL != nil {
		img = sql.NullString{String: *p.ImageURL, Valid: true}
	}

	return r.queries.CreateProduct(ctx, sqlc.CreateProductParams{
		ID:          p.ID,
		Name:        p.Name,
		Sku:         p.SKU,
		Price:       p.Price,
		Stock:       p.Stock,
		Description: desc,
		ImageUrl:    img,
		Status:      sqlc.ProductsStatus(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	})
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	row, err := r.queries.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toProductModel(row), nil
}

func (r *productRepository) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	row, err := r.queries.GetProductBySKU(ctx, sku)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return toProductModel(row), nil
}

func (r *productRepository) ListActive(ctx context.Context, limit, offset int32) ([]*models.Product, int64, error) {
	rows, err := r.queries.ListActiveProducts(ctx, sqlc.ListActiveProductsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.queries.CountActiveProducts(ctx)
	if err != nil {
		return nil, 0, err
	}

	products := make([]*models.Product, len(rows))
	for i, row := range rows {
		products[i] = toProductModel(row)
	}
	return products, total, nil
}

func (r *productRepository) Update(ctx context.Context, p *models.Product) error {
	var desc, img sql.NullString
	if p.Description != nil {
		desc = sql.NullString{String: *p.Description, Valid: true}
	}
	if p.ImageURL != nil {
		img = sql.NullString{String: *p.ImageURL, Valid: true}
	}

	return r.queries.UpdateProduct(ctx, sqlc.UpdateProductParams{
		Name:        p.Name,
		Price:       p.Price,
		Stock:       p.Stock,
		Description: desc,
		ImageUrl:    img,
		Status:      sqlc.ProductsStatus(p.Status),
		UpdatedAt:   p.UpdatedAt,
		ID:          p.ID,
	})
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteProduct(ctx, id)
}

func toProductModel(row sqlc.Product) *models.Product {
	var desc, img *string
	if row.Description.Valid {
		desc = &row.Description.String
	}
	if row.ImageUrl.Valid {
		img = &row.ImageUrl.String
	}

	return &models.Product{
		ID:          row.ID,
		Name:        row.Name,
		SKU:         row.Sku,
		Price:       row.Price,
		Stock:       row.Stock,
		Description: desc,
		ImageURL:    img,
		Status:      string(row.Status),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
```

---

## Step 5: Service Layer (Business Logic)

Create `internal/service/product_service.go`:

```go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrProductSKUExists = errors.New("product sku already exists")
)

type ProductService interface {
	Create(ctx context.Context, input models.CreateProductInput) (*models.Product, error)
	GetByID(ctx context.Context, id string) (*models.Product, error)
	ListActive(ctx context.Context, page, limit int) ([]*models.Product, *utils.PaginationMeta, error)
	Update(ctx context.Context, id string, input models.UpdateProductInput) (*models.Product, error)
	Delete(ctx context.Context, id string) error
}

type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) Create(ctx context.Context, input models.CreateProductInput) (*models.Product, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Verify SKU uniqueness
	if existing, err := s.repo.GetBySKU(ctx, input.SKU); err == nil && existing != nil {
		return nil, ErrProductSKUExists
	}

	now := time.Now().UTC()
	product := &models.Product{
		ID:          utils.NewULID(),
		Name:        input.Name,
		SKU:         input.SKU,
		Price:       input.Price,
		Stock:       input.Stock,
		Description: input.Description,
		ImageURL:    input.ImageURL,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *productService) GetByID(ctx context.Context, id string) (*models.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *productService) ListActive(ctx context.Context, page, limit int) ([]*models.Product, *utils.PaginationMeta, error) {
	p := utils.NormalizePagination(page, limit)
	items, total, err := s.repo.ListActive(ctx, int32(p.Limit), int32(p.Offset))
	if err != nil {
		return nil, nil, err
	}
	meta := utils.BuildPaginationMeta(p.Page, p.Limit, total)
	return items, meta, nil
}

func (s *productService) Update(ctx context.Context, id string, input models.UpdateProductInput) (*models.Product, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	product.Name = input.Name
	product.Price = input.Price
	product.Stock = input.Stock
	product.Description = input.Description
	product.ImageURL = input.ImageURL
	product.Status = input.Status
	product.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *productService) Delete(ctx context.Context, id string) error {
	_, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}
```

---

## Step 6: HTTP Handlers Layer

Create `internal/handlers/product_handler.go`:

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	product, err := h.svc.Create(r.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrProductSKUExists) {
			utils.ErrorCode(w, http.StatusConflict, "PRODUCT_SKU_EXISTS", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, meta, err := h.svc.ListActive(r.Context(), page, limit)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"data":       items,
		"pagination": meta,
	})
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	product, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", err.Error())
			return
		}
		utils.Error(w, http.StatusInternalServerError, "failed to get product")
		return
	}

	utils.JSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	var input models.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	product, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", err.Error())
			return
		}
		utils.ValidationError(w, err)
		return
	}

	utils.JSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !utils.IsValidULID(id) {
		utils.Error(w, http.StatusBadRequest, "invalid id, must be a 26-character ULID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			utils.ErrorCode(w, http.StatusNotFound, "PRODUCT_NOT_FOUND", err.Error())
			return
		}
		utils.Error(w, http.StatusInternalServerError, "failed to delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 7: Routing & RBAC Registration

Register your handler routes in `internal/routes/routes.go`:

```go
// Inside routes.Setup(...)
func Setup(
    // ... other parameters
    productHandler *handlers.ProductHandler,
    userRepo repository.UserRepository,
) http.Handler {
    // ...
    r.Route("/api/v1", func(r chi.Router) {
        // Public read-only endpoints
        r.Get("/products", productHandler.List)
        r.Get("/products/{id}", productHandler.GetByID)

        // Administrative protected endpoints (Requires JWT + RBAC Permission)
        r.Group(func(r chi.Router) {
            r.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

            // Create requires 'products.create' permission (or super_admin wildcard)
            r.With(middleware.RequirePermission("products.create", userRepo)).Post("/admin/products", productHandler.Create)

            // Update requires 'products.update' permission
            r.With(middleware.RequirePermission("products.update", userRepo)).Put("/admin/products/{id}", productHandler.Update)

            // Delete requires 'products.delete' permission
            r.With(middleware.RequirePermission("products.delete", userRepo)).Delete("/admin/products/{id}", productHandler.Delete)
        })
    })
    // ...
}
```

---

## Step 8: Wiring Dependencies in `main.go`

In `cmd/server/main.go`, instantiate your new repository, service, and handler:

```go
// 5. Initialize Repositories
productRepo := repository.NewProductRepository(db)

// 8. Initialize Services
productSvc := service.NewProductService(productRepo)

// 9. Initialize HTTP Handlers
productHandler := handlers.NewProductHandler(productSvc)

// 10. Pass handler into routes.Setup(...)
router := routes.Setup(
    // ... other handlers
    productHandler,
    userRepo,
)
```

---

## Step 9: Seed Initial RBAC Permissions (Optional)

If your new endpoints require custom permissions, register them in `internal/database/seeder.go`:

```go
permissions := []permissionSeed{
    // ... existing permissions
    {Name: "products.read", Description: "View products"},
    {Name: "products.create", Description: "Create new products"},
    {Name: "products.update", Description: "Update existing products"},
    {Name: "products.delete", Description: "Delete products"},
}
```

When you start the server with `go run cmd/server/main.go`, the seeder will automatically insert your new permissions and attach them to `super_admin` and assigned roles!
