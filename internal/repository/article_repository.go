package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"
)

var (
	ErrArticleNotFound   = errors.New("article not found")
	ErrArticleSlugExists = errors.New("article slug is already in use")
)

type ArticleRepository interface {
	Create(ctx context.Context, article *models.Article, tagIDs []string, mediaIDs []string, relatedIDs []string) error
	Update(ctx context.Context, article *models.Article, tagIDs []string, mediaIDs []string, relatedIDs []string, userID string) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Article, error)
	GetBySlug(ctx context.Context, slug string) (*models.Article, error)
	GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error)
	ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error)
	ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error)
	PublishScheduled(ctx context.Context) error
	Count(ctx context.Context) (int64, error)
	ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error)
	CheckArticlePublishedExists(ctx context.Context, id string) (bool, error)
}

type sqlArticleRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewArticleRepository(db *sql.DB) ArticleRepository {
	return &sqlArticleRepository{
		db: db,
		q:  sqlc.New(db),
	}
}

func (r *sqlArticleRepository) Create(ctx context.Context, article *models.Article, tagIDs []string, mediaIDs []string, relatedIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	if article.ID == "" {
		article.ID = utils.NewULID()
	}
	now := time.Now().UTC()
	article.CreatedAt = now
	article.UpdatedAt = now

	params := sqlc.CreateArticleParams{
		ID:           article.ID,
		AuthorID:     article.AuthorID,
		Title:        article.Title,
		Slug:         article.Slug,
		Content:      article.Content,
		Status:       sqlc.ArticlesStatus(article.Status),
		IsFeatured:   article.IsFeatured,
		RobotsIndex:  article.RobotsIndex,
		RobotsFollow: article.RobotsFollow,
		CreatedAt:    article.CreatedAt,
		UpdatedAt:    article.UpdatedAt,
	}

	if article.CategoryID != nil && *article.CategoryID != "" {
		params.CategoryID = sql.NullString{String: *article.CategoryID, Valid: true}
	}
	if article.FeaturedImageID != nil && *article.FeaturedImageID != "" {
		params.FeaturedImageID = sql.NullString{String: *article.FeaturedImageID, Valid: true}
	}
	if article.OgImageID != nil && *article.OgImageID != "" {
		params.OgImageID = sql.NullString{String: *article.OgImageID, Valid: true}
	}
	if article.Excerpt != nil && *article.Excerpt != "" {
		params.Excerpt = sql.NullString{String: *article.Excerpt, Valid: true}
	}
	if article.ScheduledAt != nil {
		params.ScheduledAt = sql.NullTime{Time: *article.ScheduledAt, Valid: true}
	}
	if article.PublishedAt != nil {
		params.PublishedAt = sql.NullTime{Time: *article.PublishedAt, Valid: true}
	}
	if article.MetaTitle != nil && *article.MetaTitle != "" {
		params.MetaTitle = sql.NullString{String: *article.MetaTitle, Valid: true}
	}
	if article.MetaDescription != nil && *article.MetaDescription != "" {
		params.MetaDescription = sql.NullString{String: *article.MetaDescription, Valid: true}
	}
	if article.CanonicalURL != nil && *article.CanonicalURL != "" {
		params.CanonicalUrl = sql.NullString{String: *article.CanonicalURL, Valid: true}
	}
	if len(article.Meta) > 0 {
		params.Meta = article.Meta
	}

	if err := qtx.CreateArticle(ctx, params); err != nil {
		if IsDuplicateKeyError(err) {
			return ErrArticleSlugExists
		}
		return err
	}

	// Link Tags
	for _, tagID := range tagIDs {
		if err := qtx.AddArticleTag(ctx, sqlc.AddArticleTagParams{
			ArticleID: article.ID,
			TagID:     tagID,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}

	// Link Extracted Media
	for _, mediaID := range mediaIDs {
		if err := qtx.AddArticleMedia(ctx, sqlc.AddArticleMediaParams{
			ArticleID: article.ID,
			MediaID:   mediaID,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}

	// Link Related Articles
	relatedIDs = utils.DeduplicateStrings(relatedIDs)
	for _, relatedID := range relatedIDs {
		if relatedID == article.ID {
			continue
		}
		if err := qtx.AddRelatedArticle(ctx, sqlc.AddRelatedArticleParams{
			ArticleID:        article.ID,
			RelatedArticleID: relatedID,
			CreatedAt:        now,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *sqlArticleRepository) Update(
	ctx context.Context,
	article *models.Article,
	tagIDs []string,
	mediaIDs []string,
	relatedIDs []string,
	userID string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	// Fetch existing article to check revision requirement
	existing, err := qtx.GetArticleByID(ctx, article.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrArticleNotFound
		}
		return err
	}

	now := time.Now().UTC()

	// If existing article was published, record revision snapshot before applying update
	if existing.Status == sqlc.ArticlesStatusPublished {
		snapshotBytes, _ := json.Marshal(existing)
		if revErr := qtx.CreateArticleRevision(ctx, sqlc.CreateArticleRevisionParams{
			ID:        utils.NewULID(),
			ArticleID: article.ID,
			UserID:    userID,
			Snapshot:  snapshotBytes,
			CreatedAt: now,
		}); revErr != nil {
			slog.WarnContext(ctx, "Failed to record article revision snapshot", slog.String("article_id", article.ID), slog.Any("error", revErr))
		}
	}

	article.UpdatedAt = now

	params := sqlc.UpdateArticleParams{
		ID:           article.ID,
		Title:        article.Title,
		Slug:         article.Slug,
		Content:      article.Content,
		Status:       sqlc.ArticlesStatus(article.Status),
		IsFeatured:   article.IsFeatured,
		RobotsIndex:  article.RobotsIndex,
		RobotsFollow: article.RobotsFollow,
		UpdatedAt:    article.UpdatedAt,
	}

	if article.CategoryID != nil && *article.CategoryID != "" {
		params.CategoryID = sql.NullString{String: *article.CategoryID, Valid: true}
	}
	if article.FeaturedImageID != nil && *article.FeaturedImageID != "" {
		params.FeaturedImageID = sql.NullString{String: *article.FeaturedImageID, Valid: true}
	}
	if article.OgImageID != nil && *article.OgImageID != "" {
		params.OgImageID = sql.NullString{String: *article.OgImageID, Valid: true}
	}
	if article.Excerpt != nil && *article.Excerpt != "" {
		params.Excerpt = sql.NullString{String: *article.Excerpt, Valid: true}
	}
	if article.ScheduledAt != nil {
		params.ScheduledAt = sql.NullTime{Time: *article.ScheduledAt, Valid: true}
	}
	if article.PublishedAt != nil {
		params.PublishedAt = sql.NullTime{Time: *article.PublishedAt, Valid: true}
	}
	if article.MetaTitle != nil && *article.MetaTitle != "" {
		params.MetaTitle = sql.NullString{String: *article.MetaTitle, Valid: true}
	}
	if article.MetaDescription != nil && *article.MetaDescription != "" {
		params.MetaDescription = sql.NullString{String: *article.MetaDescription, Valid: true}
	}
	if article.CanonicalURL != nil && *article.CanonicalURL != "" {
		params.CanonicalUrl = sql.NullString{String: *article.CanonicalURL, Valid: true}
	}
	if len(article.Meta) > 0 {
		params.Meta = article.Meta
	}

	if err := qtx.UpdateArticle(ctx, params); err != nil {
		if IsDuplicateKeyError(err) {
			return ErrArticleSlugExists
		}
		return err
	}

	// Sync tags: Delete and re-insert
	if err := qtx.DeleteArticleTags(ctx, article.ID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := qtx.AddArticleTag(ctx, sqlc.AddArticleTagParams{
			ArticleID: article.ID,
			TagID:     tagID,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}

	// Sync media: Delete and re-insert
	if err := qtx.DeleteArticleMedia(ctx, article.ID); err != nil {
		return err
	}
	for _, mediaID := range mediaIDs {
		if err := qtx.AddArticleMedia(ctx, sqlc.AddArticleMediaParams{
			ArticleID: article.ID,
			MediaID:   mediaID,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}

	// Sync related: Delete and re-insert
	if err := qtx.DeleteRelatedArticles(ctx, article.ID); err != nil {
		return err
	}
	relatedIDs = utils.DeduplicateStrings(relatedIDs)
	for _, relatedID := range relatedIDs {
		if relatedID == article.ID {
			continue
		}
		if err := qtx.AddRelatedArticle(ctx, sqlc.AddRelatedArticleParams{
			ArticleID:        article.ID,
			RelatedArticleID: relatedID,
			CreatedAt:        now,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *sqlArticleRepository) Delete(ctx context.Context, id string) error {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	baseSlug := existing.Slug
	if len(baseSlug) > 220 {
		baseSlug = baseSlug[:220]
	}
	deletedSlug := fmt.Sprintf("%s-deleted-%s", baseSlug, strings.ToLower(utils.NewULID()))

	if r.db != nil {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		qtx := r.q.WithTx(tx)
		// Detach article_media when article is soft-deleted so media can be safely managed/deleted
		if err := qtx.DeleteArticleMedia(ctx, id); err != nil {
			return err
		}
		if err := qtx.SoftDeleteArticle(ctx, sqlc.SoftDeleteArticleParams{
			ID:        id,
			DeletedAt: sql.NullTime{Time: now, Valid: true},
			Slug:      deletedSlug,
		}); err != nil {
			return err
		}
		return tx.Commit()
	}

	return r.q.SoftDeleteArticle(ctx, sqlc.SoftDeleteArticleParams{
		ID:        id,
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		Slug:      deletedSlug,
	})
}

func (r *sqlArticleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {
	row, err := r.q.GetArticleByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	article := r.mapArticleRow(row)
	r.populateRelations(ctx, article)
	return article, nil
}

func (r *sqlArticleRepository) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	row, err := r.q.GetArticleBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	article := r.mapArticleRow(sqlc.GetArticleByIDRow(row))
	r.populateRelations(ctx, article)
	return article, nil
}

func (r *sqlArticleRepository) GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error) {
	row, err := r.q.GetPublishedArticleBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	article := r.mapArticleRow(sqlc.GetArticleByIDRow(row))
	r.populateRelations(ctx, article)
	return article, nil
}

func (r *sqlArticleRepository) PublishScheduled(ctx context.Context) error {
	return r.q.PublishScheduledArticles(ctx)
}

func (r *sqlArticleRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountArticles(ctx)
}

func (r *sqlArticleRepository) ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error) {
	rows, err := r.q.ListArticleRevisions(ctx, articleID)
	if err != nil {
		return nil, err
	}
	revisions := make([]models.ArticleRevision, len(rows))
	for i, row := range rows {
		revisions[i] = models.ArticleRevision{
			ID:        row.ID,
			ArticleID: row.ArticleID,
			UserID:    row.UserID,
			Username:  row.Username,
			Email:     row.Email,
			Snapshot:  row.Snapshot,
			CreatedAt: row.CreatedAt,
		}
	}
	return revisions, nil
}

func (r *sqlArticleRepository) CheckArticlePublishedExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM articles WHERE id = ? AND status = 'published' AND deleted_at IS NULL)"
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (r *sqlArticleRepository) ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	whereClauses := []string{"a.status = 'published'", "a.deleted_at IS NULL"}
	var args []interface{}

	if filter.CategorySlug != "" {
		whereClauses = append(whereClauses, "c.slug = ?")
		args = append(args, filter.CategorySlug)
	} else if filter.CategoryID != "" {
		whereClauses = append(whereClauses, "a.category_id = ?")
		args = append(args, filter.CategoryID)
	}

	if filter.IsFeatured != nil {
		whereClauses = append(whereClauses, "a.is_featured = ?")
		args = append(args, *filter.IsFeatured)
	}

	if filter.TagSlug != "" {
		whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM article_tags at INNER JOIN tags t ON at.tag_id = t.id WHERE at.article_id = a.id AND t.slug = ?)")
		args = append(args, filter.TagSlug)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, "(a.title LIKE ? OR a.excerpt LIKE ? OR a.content LIKE ?)")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM articles a
		LEFT JOIN categories c ON a.category_id = c.id
		WHERE %s`, whereSQL)

	var totalItems int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.author_id, a.category_id, a.featured_image_id, a.og_image_id,
			a.title, a.slug, a.excerpt, '' AS content, a.status, a.is_featured,
			a.scheduled_at, a.published_at, a.meta_title, a.meta_description,
			a.canonical_url, a.robots_index, a.robots_follow, a.meta,
			a.created_at, a.updated_at, a.deleted_at,
			u.username AS author_username,
			c.name AS category_name, c.slug AS category_slug,
			mf.url AS featured_image_url,
			mo.url AS og_image_url,
			COALESCE((SELECT SUM(total_views) FROM article_daily_analytics WHERE article_id = a.id), 0) AS total_views
		FROM articles a
		LEFT JOIN users u ON a.author_id = u.id
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN media mf ON a.featured_image_id = mf.id
		LEFT JOIN media mo ON a.og_image_id = mo.id
		WHERE %s
		ORDER BY a.published_at DESC, a.id DESC
		LIMIT ? OFFSET ?`, whereSQL)

	queryArgs := append(args, perPage, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	articles, err := r.scanArticles(rows)
	if err != nil {
		return nil, 0, err
	}

	r.populateBatchTags(ctx, articles)

	return articles, totalItems, nil
}

func (r *sqlArticleRepository) ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	whereClauses := []string{"a.deleted_at IS NULL"}
	var args []interface{}

	if filter.Status != "" {
		whereClauses = append(whereClauses, "a.status = ?")
		args = append(args, filter.Status)
	}

	if filter.CategoryID != "" {
		whereClauses = append(whereClauses, "a.category_id = ?")
		args = append(args, filter.CategoryID)
	}

	if filter.AuthorID != "" {
		whereClauses = append(whereClauses, "a.author_id = ?")
		args = append(args, filter.AuthorID)
	}

	if filter.IsFeatured != nil {
		whereClauses = append(whereClauses, "a.is_featured = ?")
		args = append(args, *filter.IsFeatured)
	}

	if filter.TagSlug != "" {
		whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM article_tags at INNER JOIN tags t ON at.tag_id = t.id WHERE at.article_id = a.id AND t.slug = ?)")
		args = append(args, filter.TagSlug)
	}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, "(a.title LIKE ? OR a.excerpt LIKE ? OR a.slug LIKE ?)")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM articles a
		LEFT JOIN categories c ON a.category_id = c.id
		WHERE %s`, whereSQL)

	var totalItems int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalItems); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.author_id, a.category_id, a.featured_image_id, a.og_image_id,
			a.title, a.slug, a.excerpt, '' AS content, a.status, a.is_featured,
			a.scheduled_at, a.published_at, a.meta_title, a.meta_description,
			a.canonical_url, a.robots_index, a.robots_follow, a.meta,
			a.created_at, a.updated_at, a.deleted_at,
			u.username AS author_username,
			c.name AS category_name, c.slug AS category_slug,
			mf.url AS featured_image_url,
			mo.url AS og_image_url,
			COALESCE((SELECT SUM(total_views) FROM article_daily_analytics WHERE article_id = a.id), 0) AS total_views
		FROM articles a
		LEFT JOIN users u ON a.author_id = u.id
		LEFT JOIN categories c ON a.category_id = c.id
		LEFT JOIN media mf ON a.featured_image_id = mf.id
		LEFT JOIN media mo ON a.og_image_id = mo.id
		WHERE %s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT ? OFFSET ?`, whereSQL)

	queryArgs := append(args, perPage, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	articles, err := r.scanArticles(rows)
	if err != nil {
		return nil, 0, err
	}

	r.populateBatchTags(ctx, articles)

	return articles, totalItems, nil
}

func (r *sqlArticleRepository) populateBatchTags(ctx context.Context, articles []models.Article) {
	if len(articles) == 0 {
		return
	}

	articleMap := make(map[string]*models.Article, len(articles))
	placeholders := make([]string, len(articles))
	args := make([]interface{}, len(articles))
	for i := range articles {
		articleMap[articles[i].ID] = &articles[i]
		placeholders[i] = "?"
		args[i] = articles[i].ID
	}

	query := fmt.Sprintf(`
		SELECT at.article_id, t.id, t.name, t.slug, t.created_at, t.updated_at
		FROM article_tags at
		INNER JOIN tags t ON at.tag_id = t.id
		WHERE at.article_id IN (%s)
		ORDER BY t.name ASC`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var (
			articleID string
			t         models.Tag
		)
		if err := rows.Scan(&articleID, &t.ID, &t.Name, &t.Slug, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		if a, ok := articleMap[articleID]; ok {
			a.Tags = append(a.Tags, t)
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "failed iterating article tags rows", slog.Any("error", err))
	}
}

func (r *sqlArticleRepository) populateRelations(ctx context.Context, a *models.Article) {
	// Tags
	tagRows, err := r.q.GetTagsByArticleID(ctx, a.ID)
	if err == nil {
		tags := make([]models.Tag, len(tagRows))
		for i, tr := range tagRows {
			tags[i] = models.Tag{
				ID:        tr.ID,
				Name:      tr.Name,
				Slug:      tr.Slug,
				CreatedAt: tr.CreatedAt,
				UpdatedAt: tr.UpdatedAt,
			}
		}
		a.Tags = tags
	}

	// Media
	mediaRows, err := r.q.GetMediaByArticleID(ctx, a.ID)
	if err == nil {
		medias := make([]models.Media, len(mediaRows))
		for i, mr := range mediaRows {
			m := models.Media{
				ID:          mr.ID,
				UploadedBy:  mr.UploadedBy,
				Filename:    mr.Filename,
				StorageDisk: mr.StorageDisk,
				StorageKey:  mr.StorageKey,
				URL:         mr.Url,
				MimeType:    mr.MimeType,
				Size:        mr.Size,
				CreatedAt:   mr.CreatedAt,
				UpdatedAt:   mr.UpdatedAt,
			}
			if mr.Width.Valid {
				w := mr.Width.Int32
				m.Width = &w
			}
			if mr.Height.Valid {
				h := mr.Height.Int32
				m.Height = &h
			}
			medias[i] = m
		}
		a.Media = medias
	}

	// Related articles
	relRows, err := r.q.GetRelatedArticles(ctx, a.ID)
	if err == nil {
		related := make([]models.ArticleSummary, len(relRows))
		for i, rr := range relRows {
			summary := models.ArticleSummary{
				ID:        rr.ID,
				Title:     rr.Title,
				Slug:      rr.Slug,
				Status:    string(rr.Status),
				CreatedAt: rr.CreatedAt,
			}
			if rr.Excerpt.Valid {
				summary.Excerpt = &rr.Excerpt.String
			}
			if rr.FeaturedImageUrl.Valid {
				summary.FeaturedImageURL = &rr.FeaturedImageUrl.String
			}
			if rr.PublishedAt.Valid {
				summary.PublishedAt = &rr.PublishedAt.Time
			}
			related[i] = summary
		}
		a.RelatedArticles = related
	}

	a.ComputeSEOMetadata()
}

func parseViews(val interface{}) uint64 {
	switch v := val.(type) {
	case int64:
		return uint64(v)
	case int32:
		return uint64(v)
	case int:
		return uint64(v)
	case uint64:
		return v
	case []byte:
		n, _ := strconv.ParseUint(string(v), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return n
	default:
		return 0
	}
}

func (r *sqlArticleRepository) mapArticleRow(row sqlc.GetArticleByIDRow) *models.Article {
	a := &models.Article{
		ID:           row.ID,
		AuthorID:     row.AuthorID,
		Title:        row.Title,
		Slug:         row.Slug,
		Content:      row.Content,
		Status:       string(row.Status),
		IsFeatured:   row.IsFeatured,
		RobotsIndex:  row.RobotsIndex,
		RobotsFollow: row.RobotsFollow,
		Views:        parseViews(row.TotalViews),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if row.AuthorUsername.Valid {
		a.AuthorUsername = row.AuthorUsername.String
	}
	if row.CategoryID.Valid {
		a.CategoryID = &row.CategoryID.String
	}
	if row.CategoryName.Valid {
		a.CategoryName = &row.CategoryName.String
	}
	if row.CategorySlug.Valid {
		a.CategorySlug = &row.CategorySlug.String
	}
	if row.FeaturedImageID.Valid {
		a.FeaturedImageID = &row.FeaturedImageID.String
	}
	if row.FeaturedImageUrl.Valid {
		a.FeaturedImageURL = &row.FeaturedImageUrl.String
	}
	if row.OgImageID.Valid {
		a.OgImageID = &row.OgImageID.String
	}
	if row.OgImageUrl.Valid {
		a.OgImageURL = &row.OgImageUrl.String
	}
	if row.Excerpt.Valid {
		a.Excerpt = &row.Excerpt.String
	}
	if row.ScheduledAt.Valid {
		a.ScheduledAt = &row.ScheduledAt.Time
	}
	if row.PublishedAt.Valid {
		a.PublishedAt = &row.PublishedAt.Time
	}
	if row.MetaTitle.Valid {
		a.MetaTitle = &row.MetaTitle.String
	}
	if row.MetaDescription.Valid {
		a.MetaDescription = &row.MetaDescription.String
	}
	if row.CanonicalUrl.Valid {
		a.CanonicalURL = &row.CanonicalUrl.String
	}
	if len(row.Meta) > 0 {
		a.Meta = row.Meta
	}
	if row.DeletedAt.Valid {
		a.DeletedAt = &row.DeletedAt.Time
	}

	return a
}

func (r *sqlArticleRepository) scanArticles(rows *sql.Rows) ([]models.Article, error) {
	var articles []models.Article

	for rows.Next() {
		var (
			a                models.Article
			statusStr        string
			authorUsername   sql.NullString
			categoryID       sql.NullString
			categoryName     sql.NullString
			categorySlug     sql.NullString
			featuredImageID  sql.NullString
			featuredImageURL sql.NullString
			ogImageID        sql.NullString
			ogImageURL       sql.NullString
			excerpt          sql.NullString
			scheduledAt      sql.NullTime
			publishedAt      sql.NullTime
			metaTitle        sql.NullString
			metaDescription  sql.NullString
			canonicalURL     sql.NullString
			deletedAt        sql.NullTime
			rawMeta          []byte
			totalViews       int64
		)

		err := rows.Scan(
			&a.ID,
			&a.AuthorID,
			&categoryID,
			&featuredImageID,
			&ogImageID,
			&a.Title,
			&a.Slug,
			&excerpt,
			&a.Content,
			&statusStr,
			&a.IsFeatured,
			&scheduledAt,
			&publishedAt,
			&metaTitle,
			&metaDescription,
			&canonicalURL,
			&a.RobotsIndex,
			&a.RobotsFollow,
			&rawMeta,
			&a.CreatedAt,
			&a.UpdatedAt,
			&deletedAt,
			&authorUsername,
			&categoryName,
			&categorySlug,
			&featuredImageURL,
			&ogImageURL,
			&totalViews,
		)
		if err != nil {
			return nil, err
		}

		a.Status = statusStr
		a.Views = uint64(totalViews)

		if authorUsername.Valid {
			a.AuthorUsername = authorUsername.String
		}
		if categoryID.Valid {
			a.CategoryID = &categoryID.String
		}
		if categoryName.Valid {
			a.CategoryName = &categoryName.String
		}
		if categorySlug.Valid {
			a.CategorySlug = &categorySlug.String
		}
		if featuredImageID.Valid {
			a.FeaturedImageID = &featuredImageID.String
		}
		if featuredImageURL.Valid {
			a.FeaturedImageURL = &featuredImageURL.String
		}
		if ogImageID.Valid {
			a.OgImageID = &ogImageID.String
		}
		if ogImageURL.Valid {
			a.OgImageURL = &ogImageURL.String
		}
		if excerpt.Valid {
			a.Excerpt = &excerpt.String
		}
		if scheduledAt.Valid {
			a.ScheduledAt = &scheduledAt.Time
		}
		if publishedAt.Valid {
			a.PublishedAt = &publishedAt.Time
		}
		if metaTitle.Valid {
			a.MetaTitle = &metaTitle.String
		}
		if metaDescription.Valid {
			a.MetaDescription = &metaDescription.String
		}
		if canonicalURL.Valid {
			a.CanonicalURL = &canonicalURL.String
		}
		if len(rawMeta) > 0 {
			a.Meta = rawMeta
		}
		if deletedAt.Valid {
			a.DeletedAt = &deletedAt.Time
		}

		a.ComputeSEOMetadata()
		articles = append(articles, a)
	}

	return articles, nil
}
