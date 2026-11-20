package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/config"
	"github.com/mochammadbagas/go-crud-api/internal/handlers"
	"github.com/mochammadbagas/go-crud-api/internal/mailer"
	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository"
	"github.com/mochammadbagas/go-crud-api/internal/routes"
	"github.com/mochammadbagas/go-crud-api/internal/service"
	"github.com/mochammadbagas/go-crud-api/internal/storage"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/alexedwards/argon2id"
)

type testUserRepo struct {
	usersByID       map[string]*models.User
	usersByEmail    map[string]*models.User
	usersByUsername map[string]*models.User
	userRoles       map[string][]string
	userRoleIDs     map[string][]string
	userPerms       map[string][]string
}

func newTestUserRepo() *testUserRepo {
	return &testUserRepo{
		usersByID:       make(map[string]*models.User),
		usersByEmail:    make(map[string]*models.User),
		usersByUsername: make(map[string]*models.User),
		userRoles:       make(map[string][]string),
		userRoleIDs:     make(map[string][]string),
		userPerms:       make(map[string][]string),
	}
}

func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if u, ok := r.usersByEmail[email]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if u, ok := r.usersByUsername[username]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	if u, ok := r.usersByID[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testUserRepo) Create(ctx context.Context, user *models.User) error {
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByUsername[user.Username] = user
	return nil
}
func (r *testUserRepo) CreateWithProfileAndRole(ctx context.Context, user *models.User, profile *models.UserProfile, roleID string) error {
	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	r.usersByUsername[user.Username] = user
	if roleID != "" {
		r.userRoleIDs[user.ID] = append(r.userRoleIDs[user.ID], roleID)
	}
	return nil
}
func (r *testUserRepo) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	if u, ok := r.usersByID[id]; ok {
		u.PasswordHash = passwordHash
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateUsername(ctx context.Context, id string, username string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByUsername, u.Username)
		u.Username = username
		r.usersByUsername[username] = u
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if u, ok := r.usersByID[id]; ok {
		u.Status = status
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) VerifyEmail(ctx context.Context, id string) error {
	if u, ok := r.usersByID[id]; ok {
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
		u.Status = models.StatusActive
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) UpdateEmail(ctx context.Context, id string, newEmail string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByEmail, u.Email)
		u.Email = newEmail
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
		r.usersByEmail[newEmail] = u
		return nil
	}
	return repository.ErrNotFound
}
func (r *testUserRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.usersByID)), nil
}

func (r *testUserRepo) Delete(ctx context.Context, id string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByID, id)
		delete(r.usersByEmail, u.Email)
		delete(r.usersByUsername, u.Username)
		return nil
	}
	return repository.ErrNotFound
}

func (r *testUserRepo) UpdateByAdmin(ctx context.Context, id string, username, email string) error {
	if u, ok := r.usersByID[id]; ok {
		delete(r.usersByUsername, u.Username)
		delete(r.usersByEmail, u.Email)
		u.Username = username
		u.Email = email
		r.usersByUsername[username] = u
		r.usersByEmail[email] = u
		return nil
	}
	return repository.ErrNotFound
}

func (r *testUserRepo) ListAdmin(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error) {
	var list []*models.UserWithProfile
	for _, u := range r.usersByID {
		roles := r.userRoles[u.ID]
		if len(roles) == 0 {
			roles = []string{u.Role}
		}
		list = append(list, &models.UserWithProfile{
			ID:              u.ID,
			Username:        u.Username,
			Email:           u.Email,
			Role:            u.Role,
			Roles:           roles,
			Status:          u.Status,
			EmailVerifiedAt: u.EmailVerifiedAt,
			CreatedAt:       u.CreatedAt,
			UpdatedAt:       u.UpdatedAt,
		})
	}
	return list, int64(len(list)), nil
}

func (r *testUserRepo) GetUserRoles(ctx context.Context, userID string) ([]*models.Role, error) {
	roleNames := r.userRoles[userID]
	var res []*models.Role
	for _, rn := range roleNames {
		res = append(res, &models.Role{ID: rn, Name: rn, DisplayName: rn})
	}
	return res, nil
}

func (r *testUserRepo) GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	if roles, ok := r.userRoles[userID]; ok && len(roles) > 0 {
		return roles, nil
	}
	if u, ok := r.usersByID[userID]; ok {
		return []string{u.Role}, nil
	}
	return []string{}, nil
}

func (r *testUserRepo) AssignRole(ctx context.Context, userID, roleID string) error {
	r.userRoleIDs[userID] = append(r.userRoleIDs[userID], roleID)
	r.userRoles[userID] = append(r.userRoles[userID], roleID)
	return nil
}

func (r *testUserRepo) RemoveRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (r *testUserRepo) SetRoles(ctx context.Context, userID string, roleIDs []string) error {
	r.userRoleIDs[userID] = roleIDs
	r.userRoles[userID] = roleIDs
	return nil
}

func (r *testUserRepo) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	for _, u := range r.usersByID {
		if u.Status == models.StatusActive {
			roles := r.userRoles[u.ID]
			isSuper := (u.Role == models.RoleSuperAdmin)
			for _, r := range roles {
				if r == models.RoleSuperAdmin {
					isSuper = true
				}
			}
			if isSuper {
				count++
			}
		}
	}
	return count, nil
}

func (r *testUserRepo) CountActiveSuperAdminsExcluding(ctx context.Context, userID string) (int64, error) {
	var count int64
	for _, u := range r.usersByID {
		if u.ID != userID && u.Status == models.StatusActive {
			roles := r.userRoles[u.ID]
			isSuper := (u.Role == models.RoleSuperAdmin)
			for _, r := range roles {
				if r == models.RoleSuperAdmin {
					isSuper = true
				}
			}
			if isSuper {
				count++
			}
		}
	}
	return count, nil
}

func (r *testUserRepo) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	if perms, ok := r.userPerms[userID]; ok {
		return perms, nil
	}
	// For testing admin user, return all permissions
	return []string{
		"users.read", "users.create", "users.update", "users.delete",
		"roles.read", "roles.create", "roles.update", "roles.delete",
		"permissions.read", "permissions.create", "permissions.update", "permissions.delete",
		"posts.read", "posts.create", "posts.update", "posts.delete",
		"categories.manage",
	}, nil
}

func (r *testUserRepo) CheckUserHasRole(ctx context.Context, userID, roleName string) (bool, error) {
	roles := r.userRoles[userID]
	for _, r := range roles {
		if r == roleName {
			return true, nil
		}
	}
	if u, ok := r.usersByID[userID]; ok && u.Role == roleName {
		return true, nil
	}
	return false, nil
}

func (r *testUserRepo) HasAssociatedData(ctx context.Context, userID string) (bool, error) {
	return false, nil
}

type testRoleRepo struct {
	roles   map[string]*models.Role
	byName  map[string]*models.Role
	permIDs map[string][]string
}

func newTestRoleRepo() *testRoleRepo {
	return &testRoleRepo{
		roles:   make(map[string]*models.Role),
		byName:  make(map[string]*models.Role),
		permIDs: make(map[string][]string),
	}
}

func (r *testRoleRepo) Create(ctx context.Context, role *models.Role) error {
	if role.ID == "" {
		role.ID = utils.NewULID()
	}
	r.roles[role.ID] = role
	r.byName[role.Name] = role
	return nil
}

func (r *testRoleRepo) GetByID(ctx context.Context, id string) (*models.Role, error) {
	if role, ok := r.roles[id]; ok {
		return role, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testRoleRepo) GetByName(ctx context.Context, name string) (*models.Role, error) {
	if role, ok := r.byName[name]; ok {
		return role, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testRoleRepo) List(ctx context.Context) ([]*models.Role, error) {
	var list []*models.Role
	for _, role := range r.roles {
		list = append(list, role)
	}
	return list, nil
}

func (r *testRoleRepo) Update(ctx context.Context, id string, displayName string, description *string) error {
	if role, ok := r.roles[id]; ok {
		role.DisplayName = displayName
		role.Description = description
		return nil
	}
	return repository.ErrNotFound
}

func (r *testRoleRepo) Delete(ctx context.Context, id string) error {
	if role, ok := r.roles[id]; ok {
		delete(r.byName, role.Name)
		delete(r.roles, id)
		return nil
	}
	return repository.ErrNotFound
}

func (r *testRoleRepo) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	r.permIDs[roleID] = permissionIDs
	return nil
}

func (r *testRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*models.Permission, error) {
	return []*models.Permission{}, nil
}

func (r *testRoleRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.roles)), nil
}

type testPermRepo struct {
	perms  map[string]*models.Permission
	byName map[string]*models.Permission
}

func newTestPermRepo() *testPermRepo {
	return &testPermRepo{
		perms:  make(map[string]*models.Permission),
		byName: make(map[string]*models.Permission),
	}
}

func (r *testPermRepo) Create(ctx context.Context, perm *models.Permission) error {
	if perm.ID == "" {
		perm.ID = utils.NewULID()
	}
	r.perms[perm.ID] = perm
	r.byName[perm.Name] = perm
	return nil
}

func (r *testPermRepo) GetByID(ctx context.Context, id string) (*models.Permission, error) {
	if p, ok := r.perms[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testPermRepo) GetByName(ctx context.Context, name string) (*models.Permission, error) {
	if p, ok := r.byName[name]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testPermRepo) List(ctx context.Context) ([]*models.Permission, error) {
	var list []*models.Permission
	for _, p := range r.perms {
		list = append(list, p)
	}
	return list, nil
}

func (r *testPermRepo) Update(ctx context.Context, id string, description *string) error {
	if p, ok := r.perms[id]; ok {
		p.Description = description
		return nil
	}
	return repository.ErrNotFound
}

func (r *testPermRepo) Delete(ctx context.Context, id string) error {
	if p, ok := r.perms[id]; ok {
		delete(r.byName, p.Name)
		delete(r.perms, id)
		return nil
	}
	return repository.ErrNotFound
}

func (r *testPermRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.perms)), nil
}

type testAuthTokenRepo struct {
	tokens map[string]*models.AuthToken
}

func newTestAuthTokenRepo() *testAuthTokenRepo {
	return &testAuthTokenRepo{tokens: make(map[string]*models.AuthToken)}
}

func (r *testAuthTokenRepo) Create(ctx context.Context, token *models.AuthToken) error {
	r.tokens[token.TokenHash+":"+token.TokenType] = token
	return nil
}

func (r *testAuthTokenRepo) GetByHashAndType(ctx context.Context, tokenHash, tokenType string) (*models.AuthToken, error) {
	if t, ok := r.tokens[tokenHash+":"+tokenType]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testAuthTokenRepo) MarkAsUsed(ctx context.Context, id string) error {
	for _, t := range r.tokens {
		if t.ID == id {
			t.IsUsed = true
			return nil
		}
	}
	return nil
}

func (r *testAuthTokenRepo) MarkAsUsedAtomic(ctx context.Context, id string) (bool, error) {
	for _, t := range r.tokens {
		if t.ID == id {
			if t.IsUsed {
				return false, nil
			}
			t.IsUsed = true
			return true, nil
		}
	}
	return false, nil
}

func (r *testAuthTokenRepo) InvalidateUserTokensByType(ctx context.Context, userID, tokenType string) error {
	for _, t := range r.tokens {
		if t.UserID == userID && t.TokenType == tokenType {
			t.IsUsed = true
		}
	}
	return nil
}

func (r *testAuthTokenRepo) DeleteExpiredOrUsed(ctx context.Context) error {
	return nil
}

type testProfileRepo struct {
	profiles map[string]*models.UserProfile
}

func newTestProfileRepo() *testProfileRepo {
	return &testProfileRepo{profiles: make(map[string]*models.UserProfile)}
}

func (r *testProfileRepo) GetByUserID(ctx context.Context, userID string) (*models.UserProfile, error) {
	if p, ok := r.profiles[userID]; ok {
		copy := *p
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testProfileRepo) Create(ctx context.Context, profile *models.UserProfile) error {
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *testProfileRepo) Update(ctx context.Context, profile *models.UserProfile) error {
	r.profiles[profile.UserID] = profile
	return nil
}
func (r *testProfileRepo) UpdateAvatar(ctx context.Context, userID string, avatarURL string) error {
	if p, ok := r.profiles[userID]; ok {
		p.AvatarURL = &avatarURL
		return nil
	}
	return repository.ErrNotFound
}
func (r *testProfileRepo) DeleteAvatar(ctx context.Context, userID string) error {
	if p, ok := r.profiles[userID]; ok {
		p.AvatarURL = nil
		return nil
	}
	return repository.ErrNotFound
}
func (r *testProfileRepo) GetUserWithProfile(ctx context.Context, userID string) (*models.UserWithProfile, error) {
	p := r.profiles[userID]
	fullName := ""
	var bio, phone, avatar *string
	if p != nil {
		fullName = p.FullName
		bio = p.Bio
		phone = p.PhoneNumber
		avatar = p.AvatarURL
	}
	return &models.UserWithProfile{
		ID:          userID,
		Username:    "admin",
		Email:       "admin@test.com",
		Role:        models.RoleAdmin,
		Status:      models.StatusActive,
		FullName:    fullName,
		Bio:         bio,
		PhoneNumber: phone,
		AvatarURL:   avatar,
	}, nil
}

type testRefreshTokenRepo struct {
	tokensByHash map[string]*models.RefreshToken
	tokensByID   map[string]*models.RefreshToken
}

func newTestRefreshTokenRepo() *testRefreshTokenRepo {
	return &testRefreshTokenRepo{
		tokensByHash: make(map[string]*models.RefreshToken),
		tokensByID:   make(map[string]*models.RefreshToken),
	}
}

func (r *testRefreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	r.tokensByHash[token.TokenHash] = token
	r.tokensByID[token.ID] = token
	return nil
}
func (r *testRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	if t, ok := r.tokensByHash[tokenHash]; ok {
		copy := *t
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testRefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	if t, ok := r.tokensByID[id]; ok {
		t.IsRevoked = true
		return nil
	}
	return repository.ErrNotFound
}
func (r *testRefreshTokenRepo) RevokeAtomic(ctx context.Context, id string) (bool, error) {
	if t, ok := r.tokensByID[id]; ok {
		if t.IsRevoked {
			return false, nil
		}
		t.IsRevoked = true
		return true, nil
	}
	return false, repository.ErrNotFound
}
func (r *testRefreshTokenRepo) RevokeFamily(ctx context.Context, tokenFamily string) error {
	for _, t := range r.tokensByID {
		if t.TokenFamily == tokenFamily {
			t.IsRevoked = true
		}
	}
	return nil
}
func (r *testRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	for _, t := range r.tokensByID {
		if t.UserID == userID {
			t.IsRevoked = true
		}
	}
	return nil
}
func (r *testRefreshTokenRepo) DeleteExpired(ctx context.Context) error { return nil }

type testCategoryRepo struct {
	categories map[string]*models.Category
	bySlug     map[string]*models.Category
}

func newTestCategoryRepo() *testCategoryRepo {
	return &testCategoryRepo{
		categories: make(map[string]*models.Category),
		bySlug:     make(map[string]*models.Category),
	}
}

func (r *testCategoryRepo) List(ctx context.Context) ([]models.Category, error) {
	var list []models.Category
	for _, c := range r.categories {
		list = append(list, *c)
	}
	return list, nil
}
func (r *testCategoryRepo) GetByID(ctx context.Context, id string) (*models.Category, error) {
	if c, ok := r.categories[id]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testCategoryRepo) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	if c, ok := r.bySlug[slug]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (r *testCategoryRepo) Create(ctx context.Context, cat *models.Category) error {
	if cat.ID == "" {
		cat.ID = utils.NewULID()
	}
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Update(ctx context.Context, cat *models.Category) error {
	r.categories[cat.ID] = cat
	r.bySlug[cat.Slug] = cat
	return nil
}
func (r *testCategoryRepo) Delete(ctx context.Context, id string) error {
	if c, ok := r.categories[id]; ok {
		delete(r.bySlug, c.Slug)
		delete(r.categories, id)
		return nil
	}
	return repository.ErrNotFound
}
func (r *testCategoryRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.categories)), nil
}
func (r *testCategoryRepo) CountArticlesByCategoryID(ctx context.Context, categoryID string) (int64, error) {
	return 0, nil
}

type testTagRepo struct {
	tags   map[string]*models.Tag
	bySlug map[string]*models.Tag
}

func newTestTagRepo() *testTagRepo {
	return &testTagRepo{
		tags:   make(map[string]*models.Tag),
		bySlug: make(map[string]*models.Tag),
	}
}

func (r *testTagRepo) List(ctx context.Context) ([]models.Tag, error) {
	var list []models.Tag
	for _, t := range r.tags {
		list = append(list, *t)
	}
	return list, nil
}
func (r *testTagRepo) GetByID(ctx context.Context, id string) (*models.Tag, error) {
	if t, ok := r.tags[id]; ok {
		c := *t
		return &c, nil
	}
	return nil, repository.ErrTagNotFound
}
func (r *testTagRepo) GetBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	if t, ok := r.bySlug[slug]; ok {
		c := *t
		return &c, nil
	}
	return nil, repository.ErrTagNotFound
}
func (r *testTagRepo) GetByName(ctx context.Context, name string) (*models.Tag, error) {
	for _, t := range r.tags {
		if t.Name == name {
			c := *t
			return &c, nil
		}
	}
	return nil, repository.ErrTagNotFound
}
func (r *testTagRepo) Create(ctx context.Context, tag *models.Tag) error {
	if tag.ID == "" {
		tag.ID = utils.NewULID()
	}
	r.tags[tag.ID] = tag
	r.bySlug[tag.Slug] = tag
	return nil
}
func (r *testTagRepo) Update(ctx context.Context, tag *models.Tag) error {
	r.tags[tag.ID] = tag
	r.bySlug[tag.Slug] = tag
	return nil
}
func (r *testTagRepo) Delete(ctx context.Context, id string) error {
	if t, ok := r.tags[id]; ok {
		delete(r.bySlug, t.Slug)
		delete(r.tags, id)
		return nil
	}
	return repository.ErrTagNotFound
}
func (r *testTagRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.tags)), nil
}
func (r *testTagRepo) GetOrCreateTags(ctx context.Context, tagNames []string) ([]models.Tag, error) {
	var list []models.Tag
	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		slug := utils.GenerateSlug(name)
		if t, ok := r.bySlug[slug]; ok {
			list = append(list, *t)
		} else {
			tag := &models.Tag{ID: utils.NewULID(), Name: name, Slug: slug}
			_ = r.Create(ctx, tag)
			list = append(list, *tag)
		}
	}
	return list, nil
}

type testMediaRepo struct {
	medias map[string]*models.Media
}

func newTestMediaRepo() *testMediaRepo {
	return &testMediaRepo{medias: make(map[string]*models.Media)}
}
func (r *testMediaRepo) Create(ctx context.Context, m *models.Media) error {
	if m.ID == "" {
		m.ID = utils.NewULID()
	}
	r.medias[m.ID] = m
	return nil
}
func (r *testMediaRepo) GetByID(ctx context.Context, id string) (*models.Media, error) {
	if m, ok := r.medias[id]; ok {
		c := *m
		return &c, nil
	}
	return nil, repository.ErrMediaNotFound
}
func (r *testMediaRepo) List(ctx context.Context, limit, offset int32) ([]models.Media, int64, error) {
	var list []models.Media
	for _, m := range r.medias {
		list = append(list, *m)
	}
	return list, int64(len(list)), nil
}
func (r *testMediaRepo) Delete(ctx context.Context, id string) error {
	if _, ok := r.medias[id]; ok {
		delete(r.medias, id)
		return nil
	}
	return repository.ErrMediaNotFound
}
func (r *testMediaRepo) IsInUse(ctx context.Context, id string) (bool, error) {
	return false, nil
}

type testAnalyticsRepo struct{}

func newTestAnalyticsRepo() *testAnalyticsRepo { return &testAnalyticsRepo{} }
func (r *testAnalyticsRepo) RecordEvent(ctx context.Context, articleID string, input models.AnalyticsEventInput, visitorID string) error {
	return nil
}
func (r *testAnalyticsRepo) GetArticleDailyAnalytics(ctx context.Context, articleID string, startDate, endDate time.Time) ([]models.ArticleDailyAnalytics, error) {
	return []models.ArticleDailyAnalytics{}, nil
}
func (r *testAnalyticsRepo) GetUniqueVisitorsByRange(ctx context.Context, articleID string, startDate, endDate time.Time) (int64, error) {
	return 1, nil
}
func (r *testAnalyticsRepo) AggregateDaily(ctx context.Context, date time.Time) error {
	return nil
}
func (r *testAnalyticsRepo) PruneRawEvents(ctx context.Context, before time.Time) error {
	return nil
}
func (r *testAnalyticsRepo) GetOverview(ctx context.Context) (*models.AnalyticsOverview, error) {
	return &models.AnalyticsOverview{GlobalViews: 10, GlobalUniqueVisitors: 5, GlobalShares: 2, GlobalReadingTime: 120}, nil
}

type testArticleRepo struct {
	articles map[string]*models.Article
	bySlug   map[string]*models.Article
}

func newTestArticleRepo() *testArticleRepo {
	return &testArticleRepo{
		articles: make(map[string]*models.Article),
		bySlug:   make(map[string]*models.Article),
	}
}

func (r *testArticleRepo) Create(ctx context.Context, a *models.Article, tagIDs, mediaIDs, relatedIDs []string) error {
	if a.ID == "" {
		a.ID = utils.NewULID()
	}
	r.articles[a.ID] = a
	r.bySlug[a.Slug] = a
	return nil
}
func (r *testArticleRepo) Update(ctx context.Context, a *models.Article, tagIDs, mediaIDs, relatedIDs []string, userID string) error {
	r.articles[a.ID] = a
	r.bySlug[a.Slug] = a
	return nil
}
func (r *testArticleRepo) Delete(ctx context.Context, id string) error {
	if a, ok := r.articles[id]; ok {
		delete(r.bySlug, a.Slug)
		delete(r.articles, id)
		return nil
	}
	return repository.ErrArticleNotFound
}
func (r *testArticleRepo) GetByID(ctx context.Context, id string) (*models.Article, error) {
	if a, ok := r.articles[id]; ok {
		c := *a
		return &c, nil
	}
	return nil, repository.ErrArticleNotFound
}
func (r *testArticleRepo) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	if a, ok := r.bySlug[slug]; ok {
		c := *a
		return &c, nil
	}
	return nil, repository.ErrArticleNotFound
}
func (r *testArticleRepo) GetPublishedBySlug(ctx context.Context, slug string) (*models.Article, error) {
	if a, ok := r.bySlug[slug]; ok && a.Status == models.StatusPublished {
		c := *a
		return &c, nil
	}
	return nil, repository.ErrArticleNotFound
}
func (r *testArticleRepo) ListPublic(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	var list []models.Article
	for _, a := range r.articles {
		if a.Status == models.StatusPublished {
			list = append(list, *a)
		}
	}
	return list, int64(len(list)), nil
}
func (r *testArticleRepo) ListAdmin(ctx context.Context, filter models.ArticleFilter) ([]models.Article, int64, error) {
	var list []models.Article
	for _, a := range r.articles {
		list = append(list, *a)
	}
	return list, int64(len(list)), nil
}
func (r *testArticleRepo) PublishScheduled(ctx context.Context) error {
	return nil
}
func (r *testArticleRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(r.articles)), nil
}
func (r *testArticleRepo) ListRevisions(ctx context.Context, articleID string) ([]models.ArticleRevision, error) {
	return []models.ArticleRevision{}, nil
}

func (r *testArticleRepo) CheckArticlePublishedExists(ctx context.Context, id string) (bool, error) {
	if a, ok := r.articles[id]; ok && a.Status == models.StatusPublished {
		return true, nil
	}
	return false, nil
}

func setupTestServer(t *testing.T) (http.Handler, string, string) {
	hashedPassword, err := argon2id.CreateHash("secret123", argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	adminID := utils.NewULID()
	adminUser := &models.User{
		ID:           adminID,
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: hashedPassword,
		Role:         models.RoleSuperAdmin,
		Status:       models.StatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	userRepo := newTestUserRepo()
	_ = userRepo.Create(context.Background(), adminUser)
	_ = userRepo.AssignRole(context.Background(), adminID, models.RoleSuperAdmin)

	profileRepo := newTestProfileRepo()
	_ = profileRepo.Create(context.Background(), &models.UserProfile{
		UserID:   adminID,
		FullName: "Administrator",
	})

	refreshTokenRepo := newTestRefreshTokenRepo()
	authTokenRepo := newTestAuthTokenRepo()
	catRepo := newTestCategoryRepo()
	tagRepo := newTestTagRepo()
	mediaRepo := newTestMediaRepo()
	articleRepo := newTestArticleRepo()
	analyticsRepo := newTestAnalyticsRepo()
	roleRepo := newTestRoleRepo()
	permRepo := newTestPermRepo()

	cfg := &config.Config{
		Port:           "8080",
		CookieSecure:   false,
		UploadDir:      "./uploads",
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTSecret:      "test-starter-kit-secret-key-12345",
		JWTAccessTTL:   15 * time.Minute,
		JWTRefreshTTL:  7 * 24 * time.Hour,
		FrontendURL:    "http://localhost:3000",
		MailerDriver:   "log",
	}

	mailSvc := mailer.NewLogMailer()
	authSvc := service.NewAuthService(userRepo, profileRepo, refreshTokenRepo, authTokenRepo, mailSvc, cfg, roleRepo)
	userSvc := service.NewUserService(userRepo, profileRepo, authTokenRepo, refreshTokenRepo, mailSvc, cfg)
	categorySvc := service.NewCategoryService(catRepo)
	tagSvc := service.NewTagService(tagRepo)
	articleSvc := service.NewArticleService(articleRepo, catRepo, tagRepo)

	tempUploadDir := t.TempDir()
	cfg.UploadDir = tempUploadDir
	testStorage := storage.NewLocalStorage(tempUploadDir)
	mediaSvc := service.NewMediaService(mediaRepo, testStorage, cfg)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo, articleRepo, cfg.JWTSecret)

	rbacSvc := service.NewRBACService(roleRepo, permRepo)
	adminUserSvc := service.NewAdminUserService(userRepo, profileRepo, roleRepo, refreshTokenRepo)

	now := time.Now().UTC()
	_ = articleRepo.Create(context.Background(), &models.Article{
		Title:       "Published Go Article",
		Slug:        "published-go-article",
		Content:     "<h1>Welcome</h1><p>This is a published article.</p>",
		Status:      models.StatusPublished,
		PublishedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil, nil, nil)

	_ = articleRepo.Create(context.Background(), &models.Article{
		Title:     "Draft Journal Entry",
		Slug:      "draft-journal-entry",
		Content:   "<p>Secret WIP article entry.</p>",
		Status:    models.StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil, nil, nil)

	authHandler := handlers.NewAuthHandler(authSvc, cfg)
	userHandler := handlers.NewUserHandler(userSvc, cfg, testStorage)
	adminArticleHandler := handlers.NewAdminArticleHandler(articleSvc)
	publicArticleHandler := handlers.NewPublicArticleHandler(articleSvc)
	adminCategoryHandler := handlers.NewAdminCategoryHandler(categorySvc)
	publicCategoryHandler := handlers.NewPublicCategoryHandler(categorySvc, articleSvc)
	tagHandler := handlers.NewTagHandler(tagSvc)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsSvc)
	roleHandler := handlers.NewRoleHandler(rbacSvc)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserSvc, testStorage)

	router := routes.Setup(
		cfg,
		authHandler,
		userHandler,
		adminArticleHandler,
		publicArticleHandler,
		adminCategoryHandler,
		publicCategoryHandler,
		tagHandler,
		mediaHandler,
		analyticsHandler,
		roleHandler,
		adminUserHandler,
		userRepo,
	)

	// Pre-generate an admin JWT token with super_admin role for testing
	adminToken, err := utils.GenerateAccessToken(cfg.JWTSecret, cfg.JWTAccessTTL, adminID, "admin", "admin@test.com", models.RoleSuperAdmin, models.RoleSuperAdmin, models.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	return router, adminToken, "secret123"
}

func TestPublicEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", rec.Code)
	}

	// 2. Public articles list
	req = httptest.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/v1/articles, got %d", rec.Code)
	}

	// 3. Public get published article by slug
	req = httptest.NewRequest(http.MethodGet, "/api/v1/articles/published-go-article", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for published article, got %d", rec.Code)
	}

	// 4. Draft article must return 404 on public route
	req = httptest.NewRequest(http.MethodGet, "/api/v1/articles/draft-journal-entry", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for draft article accessed publicly, got %d", rec.Code)
	}
}

func TestAuthEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Register new user
	regPayload := `{"username":"alice_user","email":"alice@test.com","password":"mypassword123","full_name":"Alice Wonderland"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for register, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 1b. Duplicate register should return 409 with EMAIL_ALREADY_EXISTS code
	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRec := httptest.NewRecorder()
	router.ServeHTTP(dupRec, dupReq)
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate register, got %d", dupRec.Code)
	}
	var dupResp utils.ErrorResponse
	if err := json.NewDecoder(dupRec.Body).Decode(&dupResp); err != nil {
		t.Fatalf("failed to parse duplicate error response: %v", err)
	}
	if dupResp.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected EMAIL_ALREADY_EXISTS code, got %s", dupResp.Code)
	}

	// 1c. Login with wrong password should return 401 with INVALID_CREDENTIALS code
	badLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"alice@test.com","password":"wrongpassword"}`))
	badLoginReq.Header.Set("Content-Type", "application/json")
	badLoginRec := httptest.NewRecorder()
	router.ServeHTTP(badLoginRec, badLoginReq)
	if badLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad login, got %d", badLoginRec.Code)
	}
	var badLoginResp utils.ErrorResponse
	if err := json.NewDecoder(badLoginRec.Body).Decode(&badLoginResp); err != nil {
		t.Fatalf("failed to parse bad login response: %v", err)
	}
	if badLoginResp.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS code, got %s", badLoginResp.Code)
	}

	// 1d. Accessing protected route without token should return 401 with TOKEN_REQUIRED code
	noAuthReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	noAuthRec := httptest.NewRecorder()
	router.ServeHTTP(noAuthRec, noAuthReq)
	if noAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for no token, got %d", noAuthRec.Code)
	}
	var noAuthResp utils.ErrorResponse
	if err := json.NewDecoder(noAuthRec.Body).Decode(&noAuthResp); err != nil {
		t.Fatalf("failed to parse no auth response: %v", err)
	}
	if noAuthResp.Code != "TOKEN_REQUIRED" {
		t.Errorf("expected TOKEN_REQUIRED code, got %s", noAuthResp.Code)
	}

	// 2. Login user
	loginPayload := `{"email":"alice@test.com","password":"mypassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for login, got %d (%s)", rec.Code, rec.Body.String())
	}

	var tokenResp models.TokenResponse
	if err := json.NewDecoder(rec.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens to be present")
	}

	// Verify Cookie was set
	cookies := rec.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == handlers.RefreshTokenCookieName {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatal("expected refresh_token cookie to be set")
	}

	// 3. User profile /api/v1/users/me using access token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /users/me, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 4. Refresh token rotation
	refreshPayload := `{"refresh_token":"` + tokenResp.RefreshToken + `"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for refresh token, got %d (%s)", rec.Code, rec.Body.String())
	}

	var rotatedResp models.TokenResponse
	_ = json.NewDecoder(rec.Body).Decode(&rotatedResp)
	if rotatedResp.AccessToken == "" || rotatedResp.RefreshToken == tokenResp.RefreshToken {
		t.Errorf("expected rotated tokens, got same or empty")
	}
}

func TestAdminEndpointsWithJWT(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. Create Category as Admin
	catPayload := `{"name":"Backend Architecture"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/categories", bytes.NewBufferString(catPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin create category, got %d (%s)", rec.Code, rec.Body.String())
	}

	var createdCat models.Category
	_ = json.NewDecoder(rec.Body).Decode(&createdCat)
	if createdCat.ID == "" || len(createdCat.ID) != 26 {
		t.Errorf("expected 26-char ULID for category ID, got %q", createdCat.ID)
	}

	// 2. Create Article as Admin with ULID Category
	articlePayload := `{
		"title": "Mastering ULID in Go",
		"content": "<p>Why ULID is better than auto increment.</p>",
		"category_id": "` + createdCat.ID + `",
		"status": "published"
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles", bytes.NewBufferString(articlePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin create article, got %d (%s)", rec.Code, rec.Body.String())
	}

	var createdArticle models.Article
	_ = json.NewDecoder(rec.Body).Decode(&createdArticle)
	if createdArticle.ID == "" || len(createdArticle.ID) != 26 {
		t.Errorf("expected 26-char ULID for article ID, got %q", createdArticle.ID)
	}

	// 2b. Get Category by ID as Admin
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/categories/"+createdCat.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin get category by ID, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2c. Get Category with invalid ID
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/categories/invalid-id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid category ID, got %d", rec.Code)
	}

	// 2d. Tags CRUD as Admin
	tagPayload := `{"name":"Golang"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/tags", bytes.NewBufferString(tagPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin create tag, got %d (%s)", rec.Code, rec.Body.String())
	}
	var createdTag models.Tag
	_ = json.NewDecoder(rec.Body).Decode(&createdTag)
	if createdTag.ID == "" || len(createdTag.ID) != 26 {
		t.Errorf("expected 26-char ULID for tag ID, got %q", createdTag.ID)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/tags/"+createdTag.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin get tag by ID, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for public get tags, got %d", rec.Code)
	}

	// 2e. Analytics & Revisions
	analyticsPayload := `{"event_id":"evt-test-1","event_type":"view"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/articles/"+createdArticle.ID+"/analytics", bytes.NewBufferString(analyticsPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202 for public record analytics event, got %d (%s)", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/analytics/overview", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin analytics overview, got %d (%s)", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/articles/"+createdArticle.ID+"/revisions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for admin get article revisions, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 3. Unauthorized access without token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/articles", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth header, got %d", rec.Code)
	}
}

func TestUploadEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// Create multipart image payload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	// Minimal valid PNG header magic bytes
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	_, _ = part.Write(pngHeader)
	_, _ = part.Write(make([]byte, 100))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/media", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for admin upload, got %d (%s)", rec.Code, rec.Body.String())
	}

	var createdMedia models.Media
	if err := json.Unmarshal(rec.Body.Bytes(), &createdMedia); err != nil {
		t.Fatalf("failed to decode upload response: %v", err)
	}
	if createdMedia.ID == "" {
		t.Fatalf("expected media id in response, got %+v", createdMedia)
	}
	mediaID := createdMedia.ID
	mediaURL := createdMedia.URL

	// Test user avatar upload with complete renaming
	avatarBody := &bytes.Buffer{}
	avatarWriter := multipart.NewWriter(avatarBody)
	avatarPart, _ := avatarWriter.CreateFormFile("avatar", "my-personal-avatar.png")
	_, _ = avatarPart.Write(pngHeader)
	_, _ = avatarPart.Write(make([]byte, 50))
	avatarWriter.Close()

	avatarReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", avatarBody)
	avatarReq.Header.Set("Content-Type", avatarWriter.FormDataContentType())
	avatarReq.Header.Set("Authorization", "Bearer "+adminToken)
	avatarRec := httptest.NewRecorder()
	router.ServeHTTP(avatarRec, avatarReq)
	if avatarRec.Code != http.StatusOK {
		t.Errorf("expected 200 for user avatar upload, got %d (%s)", avatarRec.Code, avatarRec.Body.String())
	}

	var avatarResp map[string]string
	_ = json.Unmarshal(avatarRec.Body.Bytes(), &avatarResp)
	avatar1URL := avatarResp["avatar_url"]
	if strings.Contains(avatar1URL, "personal") {
		t.Errorf("avatar url should be completely renamed without preserving original filename: %s", avatar1URL)
	}

	// 1. Verify avatar 1 is accessible
	checkReq1 := httptest.NewRequest(http.MethodGet, avatar1URL, nil)
	checkRec1 := httptest.NewRecorder()
	router.ServeHTTP(checkRec1, checkReq1)
	if checkRec1.Code != http.StatusOK {
		t.Fatalf("expected 200 for newly uploaded avatar1, got %d", checkRec1.Code)
	}

	// 2. Upload avatar 2 (replacing avatar 1)
	avatarBody2 := &bytes.Buffer{}
	avatarWriter2 := multipart.NewWriter(avatarBody2)
	avatarPart2, _ := avatarWriter2.CreateFormFile("avatar", "new-avatar.png")
	_, _ = avatarPart2.Write(pngHeader)
	_, _ = avatarPart2.Write(make([]byte, 60))
	avatarWriter2.Close()

	avatarReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/avatar", avatarBody2)
	avatarReq2.Header.Set("Content-Type", avatarWriter2.FormDataContentType())
	avatarReq2.Header.Set("Authorization", "Bearer "+adminToken)
	avatarRec2 := httptest.NewRecorder()
	router.ServeHTTP(avatarRec2, avatarReq2)
	if avatarRec2.Code != http.StatusOK {
		t.Fatalf("expected 200 for user avatar replacement, got %d", avatarRec2.Code)
	}

	var avatarResp2 map[string]string
	_ = json.Unmarshal(avatarRec2.Body.Bytes(), &avatarResp2)
	avatar2URL := avatarResp2["avatar_url"]

	// Auto-cleanup check: avatar 1 should now be deleted from storage (404)
	checkReq1Again := httptest.NewRequest(http.MethodGet, avatar1URL, nil)
	checkRec1Again := httptest.NewRecorder()
	router.ServeHTTP(checkRec1Again, checkReq1Again)
	if checkRec1Again.Code != http.StatusNotFound {
		t.Errorf("expected 404 for old avatar1 after replacement (auto-cleanup), got %d", checkRec1Again.Code)
	}

	// Avatar 2 should exist (200)
	checkReq2 := httptest.NewRequest(http.MethodGet, avatar2URL, nil)
	checkRec2 := httptest.NewRecorder()
	router.ServeHTTP(checkRec2, checkReq2)
	if checkRec2.Code != http.StatusOK {
		t.Errorf("expected 200 for avatar2, got %d", checkRec2.Code)
	}

	// 3. Delete avatar via DELETE /api/v1/users/avatar
	delAvatarReq := httptest.NewRequest(http.MethodDelete, "/api/v1/users/avatar", nil)
	delAvatarReq.Header.Set("Authorization", "Bearer "+adminToken)
	delAvatarRec := httptest.NewRecorder()
	router.ServeHTTP(delAvatarRec, delAvatarReq)
	if delAvatarRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for DELETE avatar, got %d", delAvatarRec.Code)
	}

	// Auto-cleanup check: avatar 2 should now be deleted from storage (404)
	checkReq2Again := httptest.NewRequest(http.MethodGet, avatar2URL, nil)
	checkRec2Again := httptest.NewRecorder()
	router.ServeHTTP(checkRec2Again, checkReq2Again)
	if checkRec2Again.Code != http.StatusNotFound {
		t.Errorf("expected 404 for avatar2 after DELETE (auto-cleanup), got %d", checkRec2Again.Code)
	}

	// 4. Test Article Creation with Featured Media
	createArticlePayload := fmt.Sprintf(`{
		"title": "Article With Featured Media",
		"slug": "article-with-featured-media",
		"content": "<h1>Article body content</h1>",
		"featured_image_id": "%s",
		"status": "published"
	}`, mediaID)
	artReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles", bytes.NewBufferString(createArticlePayload))
	artReq.Header.Set("Content-Type", "application/json")
	artReq.Header.Set("Authorization", "Bearer "+adminToken)
	artRec := httptest.NewRecorder()
	router.ServeHTTP(artRec, artReq)
	if artRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create article, got %d (%s)", artRec.Code, artRec.Body.String())
	}

	// Verify media is still accessible
	mediaCheckReq := httptest.NewRequest(http.MethodGet, mediaURL, nil)
	mediaCheckRec := httptest.NewRecorder()
	router.ServeHTTP(mediaCheckRec, mediaCheckReq)
	if mediaCheckRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for media, got %d", mediaCheckRec.Code)
	}
}

func TestForgotPasswordAndResetPasswordEndpoints(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// 1. Forgot password request (anti user-enumeration should return 200 for any valid email)
	payload := `{"email":"admin@mmdbconnect.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for forgot password, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2. Reset password with invalid token
	resetPayload := `{"token":"invalid-token","new_password":"newpassword123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(resetPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for reset password with invalid token, got %d", rec.Code)
	}
}

func TestEmailVerificationAndChangeEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. Verify email with empty/invalid token (POST required, GET returns 405)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify-email?token=invalid-token", nil)
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET verify email, got %d", recGet.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBufferString(`{"token":"invalid-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for verify email with invalid token, got %d", rec.Code)
	}

	// 2. Resend verification
	resendPayload := `{"email":"admin@mmdbconnect.com"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification", bytes.NewBufferString(resendPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Errorf("unexpected status code for resend verification: %d", rec.Code)
	}

	// 3. Request email change without auth -> 401
	changeReq := `{"new_email":"newadmin@example.com","password":"secret123"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/email/request-change", bytes.NewBufferString(changeReq))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth for request email change, got %d", rec.Code)
	}

	// 4. Request email change with valid auth
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/email/request-change", bytes.NewBufferString(changeReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for request email change with auth, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 5. Confirm email change with bogus token (POST required, GET returns 405)
	reqGet = httptest.NewRequest(http.MethodGet, "/api/v1/users/email/confirm-change?token=bogus", nil)
	recGet = httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET confirm email change, got %d", recGet.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/email/confirm-change", bytes.NewBufferString(`{"token":"bogus"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for confirm email change with bogus token, got %d", rec.Code)
	}
}

func TestRBACEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. GET /api/v1/admin/roles without auth -> 401
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rec.Code)
	}

	// 2. GET /api/v1/admin/roles with adminToken -> 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. POST /api/v1/admin/roles -> 201
	rolePayload := `{"name":"moderator","display_name":"Moderator Konten"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/roles", bytes.NewBufferString(rolePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for create role, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdRole models.Role
	_ = json.Unmarshal(rec.Body.Bytes(), &createdRole)

	// 4. GET /api/v1/admin/permissions -> 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for list permissions, got %d", rec.Code)
	}

	// 5. POST /api/v1/admin/permissions -> 201
	permPayload := `{"name":"audit.view","description":"Melihat log audit"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/permissions", bytes.NewBufferString(permPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 for create permission, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdPerm models.Permission
	_ = json.Unmarshal(rec.Body.Bytes(), &createdPerm)

	// 6. PUT /api/v1/admin/roles/{id}/permissions -> 200
	assignPayload := fmt.Sprintf(`{"permission_ids":["%s"]}`, createdPerm.ID)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/roles/"+createdRole.ID+"/permissions", bytes.NewBufferString(assignPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for assign permissions, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. GET /api/v1/admin/roles/invalid-id -> 400 Bad Request (VAL-05)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles/invalid-id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid role ULID, got %d", rec.Code)
	}

	// 8. GET /api/v1/admin/permissions/invalid-id -> 400 Bad Request (VAL-05)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/permissions/invalid-id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid permission ULID, got %d", rec.Code)
	}
}

func TestAdminUserEndpoints(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. GET /api/v1/admin/users without auth -> 401
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", rec.Code)
	}

	// 2. GET /api/v1/admin/users with adminToken -> 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for list admin users, got %d: %s", rec.Code, rec.Body.String())
	}

	var paginatedResp utils.PaginatedResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &paginatedResp)
	usersBytes, _ := json.Marshal(paginatedResp.Data)
	var usersList []*models.UserWithProfile
	_ = json.Unmarshal(usersBytes, &usersList)

	if len(usersList) == 0 {
		t.Fatal("expected at least one user in list")
	}

	adminUserID := usersList[0].ID

	// 3. GET /api/v1/admin/users/{id} -> 200
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/"+adminUserID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for user detail, got %d", rec.Code)
	}

	// 4. PUT /api/v1/admin/users/{id}/status to inactive on last active super admin -> 403 Forbidden!
	statusPayload := `{"status":"inactive"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+adminUserID+"/status", bytes.NewBufferString(statusPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for deactivating last super admin, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. DELETE /api/v1/admin/users/{id} on self -> 400 Bad Request (CANNOT_DELETE_SELF)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/users/"+adminUserID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for deleting self, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. GET /api/v1/admin/users/invalid-id -> 400 Bad Request (VAL-05)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/invalid-id", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid user ULID, got %d", rec.Code)
	}
}

func TestFieldLevelValidationResponse(t *testing.T) {
	router, adminToken, _ := setupTestServer(t)

	// 1. POST /api/v1/auth/register with empty payload
	regPayload := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty register, got %d", rec.Code)
	}

	var regResp utils.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&regResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if regResp.Error != "validation failed" {
		t.Errorf("expected error 'validation failed', got %q", regResp.Error)
	}
	if regResp.Code != "VALIDATION_FAILED" {
		t.Errorf("expected code 'VALIDATION_FAILED', got %q", regResp.Code)
	}
	if regResp.Errors == nil {
		t.Fatal("expected errors map to be non-nil")
	}
	if regResp.Errors["username"].Code == "" {
		t.Error("expected username error in errors map")
	}
	if regResp.Errors["email"].Code == "" {
		t.Error("expected email error in errors map")
	}
	if regResp.Errors["password"].Code == "" {
		t.Error("expected password error in errors map")
	}

	// 2. POST /api/v1/auth/login with empty fields
	loginPayload := `{"email":"","password":""}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty login, got %d", rec.Code)
	}

	var loginResp utils.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("failed to decode login error response: %v", err)
	}
	if loginResp.Error != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", loginResp.Error)
	}
	if loginResp.Errors["email"].Code == "" || loginResp.Errors["password"].Code == "" {
		t.Errorf("expected email and password validation errors, got %v", loginResp.Errors)
	}

	// 3. POST /api/v1/admin/articles with empty fields
	artPayload := `{}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles", bytes.NewBufferString(artPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty article creation, got %d", rec.Code)
	}

	var artResp utils.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&artResp); err != nil {
		t.Fatalf("failed to decode article error response: %v", err)
	}
	if artResp.Error != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", artResp.Error)
	}
	if artResp.Errors["title"].Code == "" || artResp.Errors["content"].Code == "" {
		t.Errorf("expected title and content validation errors, got %v", artResp.Errors)
	}
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	router, _, _ := setupTestServer(t)

	// Test 404 Not Found returns structured JSON
	req := httptest.NewRequest(http.MethodGet, "/api/v1/non-existent-endpoint", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var notFoundResp utils.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&notFoundResp); err != nil {
		t.Fatalf("failed to decode 404 response: %v", err)
	}
	if notFoundResp.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", notFoundResp.Code)
	}
	if notFoundResp.Error != "endpoint not found" {
		t.Errorf("expected error 'endpoint not found', got %q", notFoundResp.Error)
	}

	// Test 405 Method Not Allowed returns structured JSON
	req = httptest.NewRequest(http.MethodDelete, "/health", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	var methodResp utils.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&methodResp); err != nil {
		t.Fatalf("failed to decode 405 response: %v", err)
	}
	if methodResp.Code != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected code METHOD_NOT_ALLOWED, got %s", methodResp.Code)
	}
}
