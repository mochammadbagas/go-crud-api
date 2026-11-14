package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

type PostRepository interface {
	Create(ctx context.Context, post *models.Post) error
	Update(ctx context.Context, post *models.Post) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Post, error)
	GetBySlug(ctx context.Context, slug string) (*models.Post, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error)
	ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error)
	ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error)
	IncrementViews(ctx context.Context, postID string) error
	Count(ctx context.Context) (int64, error)
}

type sqlPostRepository struct {
	sqlDB *sql.DB
	q     *sqlc.Queries
}

func NewPostRepository(sqlDB *sql.DB) PostRepository {
	return &sqlPostRepository{
		sqlDB: sqlDB,
		q:     sqlc.New(sqlDB),
	}
}

func (r *sqlPostRepository) Create(ctx context.Context, post *models.Post) error {
	now := time.Now().UTC()
	if post.ID == "" {
		post.ID = utils.NewULID()
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	if post.UpdatedAt.IsZero() {
		post.UpdatedAt = now
	}

	tagsJSON, err := json.Marshal(post.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	var excerpt sql.NullString
	if post.Excerpt != nil {
		excerpt = sql.NullString{String: *post.Excerpt, Valid: true}
	}

	var coverImage sql.NullString
	if post.CoverImage != nil {
		coverImage = sql.NullString{String: *post.CoverImage, Valid: true}
	}

	var categoryID sql.NullString
	if post.CategoryID != nil && *post.CategoryID != "" {
		categoryID = sql.NullString{String: *post.CategoryID, Valid: true}
	}

	var metaTitle sql.NullString
	if post.MetaTitle != nil {
		metaTitle = sql.NullString{String: *post.MetaTitle, Valid: true}
	}

	var metaDesc sql.NullString
	if post.MetaDescription != nil {
		metaDesc = sql.NullString{String: *post.MetaDescription, Valid: true}
	}

	var publishedAt sql.NullTime
	if post.PublishedAt != nil {
		publishedAt = sql.NullTime{Time: *post.PublishedAt, Valid: true}
	}

	err = r.q.CreatePost(ctx, sqlc.CreatePostParams{
		ID:              post.ID,
		Type:            sqlc.PostsType(post.Type),
		Title:           post.Title,
		Slug:            post.Slug,
		Excerpt:         excerpt,
		Content:         post.Content,
		CoverImage:      coverImage,
		CategoryID:      categoryID,
		Tags:            tagsJSON,
		MetaTitle:       metaTitle,
		MetaDescription: metaDesc,
		IsFeatured:      post.IsFeatured,
		Status:          sqlc.PostsStatus(post.Status),
		PublishedAt:     publishedAt,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}

	return nil
}

func (r *sqlPostRepository) Update(ctx context.Context, post *models.Post) error {
	post.UpdatedAt = time.Now().UTC()

	tagsJSON, err := json.Marshal(post.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	var excerpt sql.NullString
	if post.Excerpt != nil {
		excerpt = sql.NullString{String: *post.Excerpt, Valid: true}
	}

	var coverImage sql.NullString
	if post.CoverImage != nil {
		coverImage = sql.NullString{String: *post.CoverImage, Valid: true}
	}

	var categoryID sql.NullString
	if post.CategoryID != nil && *post.CategoryID != "" {
		categoryID = sql.NullString{String: *post.CategoryID, Valid: true}
	}

	var metaTitle sql.NullString
	if post.MetaTitle != nil {
		metaTitle = sql.NullString{String: *post.MetaTitle, Valid: true}
	}

	var metaDesc sql.NullString
	if post.MetaDescription != nil {
		metaDesc = sql.NullString{String: *post.MetaDescription, Valid: true}
	}

	var publishedAt sql.NullTime
	if post.PublishedAt != nil {
		publishedAt = sql.NullTime{Time: *post.PublishedAt, Valid: true}
	}

	err = r.q.UpdatePost(ctx, sqlc.UpdatePostParams{
		Type:            sqlc.PostsType(post.Type),
		Title:           post.Title,
		Slug:            post.Slug,
		Excerpt:         excerpt,
		Content:         post.Content,
		CoverImage:      coverImage,
		CategoryID:      categoryID,
		Tags:            tagsJSON,
		MetaTitle:       metaTitle,
		MetaDescription: metaDesc,
		IsFeatured:      post.IsFeatured,
		Status:          sqlc.PostsStatus(post.Status),
		PublishedAt:     publishedAt,
		UpdatedAt:       post.UpdatedAt,
		ID:              post.ID,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlPostRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeletePost(ctx, id)
}

func (r *sqlPostRepository) GetByID(ctx context.Context, id string) (*models.Post, error) {
	row, err := r.q.GetPostByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return mapRowToPost(
		row.ID, string(row.Type), row.Title, row.Slug, row.Excerpt, row.Content,
		row.CoverImage, row.CategoryID, row.CategoryName, row.CategorySlug,
		row.Tags, row.MetaTitle, row.MetaDescription, row.IsFeatured,
		string(row.Status), row.PublishedAt, row.CreatedAt, row.UpdatedAt, row.Views,
	), nil
}

func (r *sqlPostRepository) GetBySlug(ctx context.Context, slug string) (*models.Post, error) {
	row, err := r.q.GetPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return mapRowToPost(
		row.ID, string(row.Type), row.Title, row.Slug, row.Excerpt, row.Content,
		row.CoverImage, row.CategoryID, row.CategoryName, row.CategorySlug,
		row.Tags, row.MetaTitle, row.MetaDescription, row.IsFeatured,
		string(row.Status), row.PublishedAt, row.CreatedAt, row.UpdatedAt, row.Views,
	), nil
}

func (r *sqlPostRepository) GetPublishedBySlug(ctx context.Context, slug string) (*models.Post, error) {
	row, err := r.q.GetPublishedPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return mapRowToPost(
		row.ID, string(row.Type), row.Title, row.Slug, row.Excerpt, row.Content,
		row.CoverImage, row.CategoryID, row.CategoryName, row.CategorySlug,
		row.Tags, row.MetaTitle, row.MetaDescription, row.IsFeatured,
		string(row.Status), row.PublishedAt, row.CreatedAt, row.UpdatedAt, row.Views,
	), nil
}

func (r *sqlPostRepository) IncrementViews(ctx context.Context, postID string) error {
	return r.q.IncrementPostViews(ctx, sqlc.IncrementPostViewsParams{
		ID:        utils.NewULID(),
		PostID:    postID,
		UpdatedAt: time.Now().UTC(),
	})
}

func (r *sqlPostRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountPosts(ctx)
}

func (r *sqlPostRepository) ListPublic(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	whereClauses := []string{"p.status = 'published'"}
	var args []interface{}

	if filter.Type != "" {
		whereClauses = append(whereClauses, "p.type = ?")
		args = append(args, filter.Type)
	}

	if filter.CategorySlug != "" {
		whereClauses = append(whereClauses, "c.slug = ?")
		args = append(args, filter.CategorySlug)
	} else if filter.CategoryID != nil && *filter.CategoryID != "" {
		whereClauses = append(whereClauses, "p.category_id = ?")
		args = append(args, *filter.CategoryID)
	}

	if filter.Featured != nil {
		whereClauses = append(whereClauses, "p.is_featured = ?")
		args = append(args, *filter.Featured)
	}

	if filter.Tag != "" {
		whereClauses = append(whereClauses, "JSON_CONTAINS(p.tags, JSON_QUOTE(?))")
		args = append(args, filter.Tag)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM posts p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE %s`, whereSQL)

	var totalItems int64
	if err := r.sqlDB.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, 0, err
	}

	offset := utils.CalculateOffset(filter.Page, filter.Limit)
	query := fmt.Sprintf(`
		SELECT 
			p.id, p.type, p.title, p.slug, p.excerpt, p.content, p.cover_image,
			p.category_id, p.tags, p.meta_title, p.meta_description,
			p.is_featured, p.status, p.published_at, p.created_at, p.updated_at,
			c.name AS category_name, c.slug AS category_slug,
			COALESCE(pv.views, 0) AS views
		FROM posts p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN post_views pv ON p.id = pv.post_id
		WHERE %s
		ORDER BY p.published_at DESC, p.id DESC
		LIMIT ? OFFSET ?`, whereSQL)

	queryArgs := append(args, filter.Limit, offset)
	rows, err := r.sqlDB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}

	return posts, totalItems, nil
}

func (r *sqlPostRepository) ListAdmin(ctx context.Context, filter models.PostFilter) ([]models.Post, int64, error) {
	whereClauses := []string{"1=1"}
	var args []interface{}

	if filter.Status != "" {
		whereClauses = append(whereClauses, "p.status = ?")
		args = append(args, filter.Status)
	}

	if filter.Type != "" {
		whereClauses = append(whereClauses, "p.type = ?")
		args = append(args, filter.Type)
	}

	if filter.CategoryID != nil && *filter.CategoryID != "" {
		whereClauses = append(whereClauses, "p.category_id = ?")
		args = append(args, *filter.CategoryID)
	}

	if filter.Featured != nil {
		whereClauses = append(whereClauses, "p.is_featured = ?")
		args = append(args, *filter.Featured)
	}

	if filter.Tag != "" {
		whereClauses = append(whereClauses, "JSON_CONTAINS(p.tags, JSON_QUOTE(?))")
		args = append(args, filter.Tag)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM posts p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE %s`, whereSQL)

	var totalItems int64
	if err := r.sqlDB.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, 0, err
	}

	offset := utils.CalculateOffset(filter.Page, filter.Limit)
	query := fmt.Sprintf(`
		SELECT 
			p.id, p.type, p.title, p.slug, p.excerpt, p.content, p.cover_image,
			p.category_id, p.tags, p.meta_title, p.meta_description,
			p.is_featured, p.status, p.published_at, p.created_at, p.updated_at,
			c.name AS category_name, c.slug AS category_slug,
			COALESCE(pv.views, 0) AS views
		FROM posts p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN post_views pv ON p.id = pv.post_id
		WHERE %s
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT ? OFFSET ?`, whereSQL)

	queryArgs := append(args, filter.Limit, offset)
	rows, err := r.sqlDB.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}

	return posts, totalItems, nil
}

func scanPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var (
			id              string
			postType        string
			title           string
			slug            string
			excerpt         sql.NullString
			content         string
			coverImage      sql.NullString
			categoryID      sql.NullString
			tagsRaw         json.RawMessage
			metaTitle       sql.NullString
			metaDescription sql.NullString
			isFeatured      bool
			status          string
			publishedAt     sql.NullTime
			createdAt       time.Time
			updatedAt       time.Time
			categoryName    sql.NullString
			categorySlug    sql.NullString
			views           uint64
		)

		if err := rows.Scan(
			&id, &postType, &title, &slug, &excerpt, &content, &coverImage,
			&categoryID, &tagsRaw, &metaTitle, &metaDescription,
			&isFeatured, &status, &publishedAt, &createdAt, &updatedAt,
			&categoryName, &categorySlug, &views,
		); err != nil {
			return nil, err
		}

		p := mapRowToPost(
			id, postType, title, slug, excerpt, content, coverImage,
			categoryID, categoryName, categorySlug, tagsRaw,
			metaTitle, metaDescription, isFeatured, status,
			publishedAt, createdAt, updatedAt, views,
		)
		posts = append(posts, *p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func mapRowToPost(
	id string,
	postType string,
	title string,
	slug string,
	excerpt sql.NullString,
	content string,
	coverImage sql.NullString,
	categoryID sql.NullString,
	categoryName sql.NullString,
	categorySlug sql.NullString,
	tagsRaw json.RawMessage,
	metaTitle sql.NullString,
	metaDescription sql.NullString,
	isFeatured bool,
	status string,
	publishedAt sql.NullTime,
	createdAt time.Time,
	updatedAt time.Time,
	views uint64,
) *models.Post {
	p := &models.Post{
		ID:         id,
		Type:       postType,
		Title:      title,
		Slug:       slug,
		Content:    content,
		IsFeatured: isFeatured,
		Status:     status,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Views:      views,
		Tags:       []string{},
	}

	if excerpt.Valid {
		p.Excerpt = &excerpt.String
	}
	if coverImage.Valid {
		p.CoverImage = &coverImage.String
	}
	if categoryID.Valid {
		catID := categoryID.String
		p.CategoryID = &catID
	}
	if categoryName.Valid {
		p.CategoryName = &categoryName.String
	}
	if categorySlug.Valid {
		p.CategorySlug = &categorySlug.String
	}
	if metaTitle.Valid {
		p.MetaTitle = &metaTitle.String
	}
	if metaDescription.Valid {
		p.MetaDescription = &metaDescription.String
	}
	if publishedAt.Valid {
		p.PublishedAt = &publishedAt.Time
	}

	if len(tagsRaw) > 0 {
		var tags []string
		if err := json.Unmarshal(tagsRaw, &tags); err == nil && tags != nil {
			p.Tags = tags
		}
	}

	p.ComputeSEOMetadata()

	return p
}
