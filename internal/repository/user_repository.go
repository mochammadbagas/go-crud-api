package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mochammadbagas/go-crud-api/internal/models"
	"github.com/mochammadbagas/go-crud-api/internal/repository/sqlc"
	"github.com/mochammadbagas/go-crud-api/internal/utils"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate entry")
)

// IsDuplicateKeyError checks if the given error is a MySQL duplicate entry error (code 1062).
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	CreateWithProfileAndRole(ctx context.Context, user *models.User, profile *models.UserProfile, roleID string) error
	UpdatePassword(ctx context.Context, id string, passwordHash string) error
	UpdateUsername(ctx context.Context, id string, username string) error
	UpdateStatus(ctx context.Context, id string, status string) error
	VerifyEmail(ctx context.Context, id string) error
	UpdateEmail(ctx context.Context, id string, newEmail string) error
	Count(ctx context.Context) (int64, error)

	// Admin and RBAC extensions
	Delete(ctx context.Context, id string) error
	UpdateByAdmin(ctx context.Context, id string, username, email string) error
	ListAdmin(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error)
	GetUserRoles(ctx context.Context, userID string) ([]*models.Role, error)
	GetUserRoleNames(ctx context.Context, userID string) ([]string, error)
	AssignRole(ctx context.Context, userID, roleID string) error
	RemoveRole(ctx context.Context, userID, roleID string) error
	SetRoles(ctx context.Context, userID string, roleIDs []string) error
	CountActiveSuperAdmins(ctx context.Context) (int64, error)
	CountActiveSuperAdminsExcluding(ctx context.Context, userID string) (int64, error)
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
	CheckUserHasRole(ctx context.Context, userID, roleName string) (bool, error)
	HasAssociatedData(ctx context.Context, userID string) (bool, error)
}

type sqlUserRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

func NewUserRepository(sqlDB *sql.DB) UserRepository {
	return &sqlUserRepository{
		db: sqlDB,
		q:  sqlc.New(sqlDB),
	}
}

func (r *sqlUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var verifiedAt *time.Time
	if row.EmailVerifiedAt.Valid {
		t := row.EmailVerifiedAt.Time
		verifiedAt = &t
	}

	u := &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	// Attach roles
	roles, err := r.GetUserRoleNames(ctx, u.ID)
	if err == nil && len(roles) > 0 {
		u.Roles = roles
	} else {
		u.Roles = []string{u.Role}
	}

	return u, nil
}

func (r *sqlUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var verifiedAt *time.Time
	if row.EmailVerifiedAt.Valid {
		t := row.EmailVerifiedAt.Time
		verifiedAt = &t
	}

	u := &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	roles, err := r.GetUserRoleNames(ctx, u.ID)
	if err == nil && len(roles) > 0 {
		u.Roles = roles
	} else {
		u.Roles = []string{u.Role}
	}

	return u, nil
}

func (r *sqlUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var verifiedAt *time.Time
	if row.EmailVerifiedAt.Valid {
		t := row.EmailVerifiedAt.Time
		verifiedAt = &t
	}

	u := &models.User{
		ID:              row.ID,
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		Role:            string(row.Role),
		Status:          string(row.Status),
		EmailVerifiedAt: verifiedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	roles, err := r.GetUserRoleNames(ctx, u.ID)
	if err == nil && len(roles) > 0 {
		u.Roles = roles
	} else {
		u.Roles = []string{u.Role}
	}

	return u, nil
}

func (r *sqlUserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = utils.NewULID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	if user.Role == "" {
		user.Role = models.RoleUser
	}
	if user.Status == "" {
		user.Status = models.StatusActive
	}

	var emailVerifiedAt sql.NullTime
	if user.EmailVerifiedAt != nil {
		emailVerifiedAt = sql.NullTime{Time: *user.EmailVerifiedAt, Valid: true}
	}

	err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		PasswordHash:    user.PasswordHash,
		Role:            sqlc.UsersRole(user.Role),
		Status:          sqlc.UsersStatus(user.Status),
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) CreateWithProfileAndRole(ctx context.Context, user *models.User, profile *models.UserProfile, roleID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := r.q.WithTx(tx)

	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = utils.NewULID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	if user.Role == "" {
		user.Role = models.RoleUser
	}
	if user.Status == "" {
		user.Status = models.StatusActive
	}

	var emailVerifiedAt sql.NullTime
	if user.EmailVerifiedAt != nil {
		emailVerifiedAt = sql.NullTime{Time: *user.EmailVerifiedAt, Valid: true}
	}

	// 1. Create User
	err = qtx.CreateUser(ctx, sqlc.CreateUserParams{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		PasswordHash:    user.PasswordHash,
		Role:            sqlc.UsersRole(user.Role),
		Status:          sqlc.UsersStatus(user.Status),
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	// 2. Create User Profile
	if profile != nil {
		if profile.UserID == "" {
			profile.UserID = user.ID
		}
		if profile.CreatedAt.IsZero() {
			profile.CreatedAt = user.CreatedAt
		}
		if profile.UpdatedAt.IsZero() {
			profile.UpdatedAt = user.UpdatedAt
		}

		var bio, phone, avatar sql.NullString
		if profile.Bio != nil {
			bio = sql.NullString{String: *profile.Bio, Valid: true}
		}
		if profile.PhoneNumber != nil {
			phone = sql.NullString{String: *profile.PhoneNumber, Valid: true}
		}
		if profile.AvatarURL != nil {
			avatar = sql.NullString{String: *profile.AvatarURL, Valid: true}
		}

		err = qtx.CreateUserProfile(ctx, sqlc.CreateUserProfileParams{
			UserID:      profile.UserID,
			FullName:    profile.FullName,
			Bio:         bio,
			PhoneNumber: phone,
			AvatarUrl:   avatar,
			CreatedAt:   profile.CreatedAt,
			UpdatedAt:   profile.UpdatedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to insert user profile: %w", err)
		}
	}

	// 3. Assign Role (if roleID is provided)
	if roleID != "" {
		err = qtx.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{
			UserID:    user.ID,
			RoleID:    roleID,
			CreatedAt: user.CreatedAt,
		})
		if err != nil {
			return fmt.Errorf("failed to assign user role: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *sqlUserRepository) UpdatePassword(ctx context.Context, id string, passwordHash string) error {
	return r.q.UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    time.Now().UTC(),
		ID:           id,
	})
}

func (r *sqlUserRepository) UpdateUsername(ctx context.Context, id string, username string) error {
	err := r.q.UpdateUsername(ctx, sqlc.UpdateUsernameParams{
		Username:  username,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.q.UpdateStatus(ctx, sqlc.UpdateStatusParams{
		Status:    sqlc.UsersStatus(status),
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
}

func (r *sqlUserRepository) VerifyEmail(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.q.VerifyUserEmail(ctx, sqlc.VerifyUserEmailParams{
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:       now,
		ID:              id,
	})
}

func (r *sqlUserRepository) UpdateEmail(ctx context.Context, id string, newEmail string) error {
	now := time.Now().UTC()
	err := r.q.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{
		Email:           newEmail,
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:       now,
		ID:              id,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountUsers(ctx)
}

func (r *sqlUserRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteUser(ctx, id)
}

func (r *sqlUserRepository) UpdateByAdmin(ctx context.Context, id string, username, email string) error {
	err := r.q.UpdateUserByAdmin(ctx, sqlc.UpdateUserByAdminParams{
		Username:  username,
		Email:     email,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) ListAdmin(ctx context.Context, filter models.AdminUserFilter) ([]*models.UserWithProfile, int64, error) {
	baseQuery := `
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
	`
	var whereClauses []string
	var args []interface{}

	if strings.TrimSpace(filter.Search) != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, "(u.username LIKE ? OR u.email LIKE ? OR p.full_name LIKE ?)")
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if strings.TrimSpace(filter.Status) != "" {
		whereClauses = append(whereClauses, "u.status = ?")
		args = append(args, strings.TrimSpace(filter.Status))
	}

	if strings.TrimSpace(filter.Role) != "" {
		roleName := strings.TrimSpace(filter.Role)
		whereClauses = append(whereClauses, "(u.role = ? OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = u.id AND r.name = ?))")
		args = append(args, roleName, roleName)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Count total records
	countSQL := "SELECT COUNT(DISTINCT u.id) " + baseQuery + whereSQL
	var total int64
	err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.UserWithProfile{}, 0, nil
	}

	// 2. Fetch paginated records
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	selectSQL := `
		SELECT 
			u.id, u.username, u.email, u.role, u.status, u.email_verified_at, u.created_at, u.updated_at,
			COALESCE(p.full_name, '') as full_name, p.bio, p.phone_number, p.avatar_url
	` + baseQuery + whereSQL + " ORDER BY u.created_at DESC LIMIT ? OFFSET ?"

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectSQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.UserWithProfile
	var userIDs []interface{}
	userMap := make(map[string]*models.UserWithProfile)

	for rows.Next() {
		var (
			u          models.UserWithProfile
			verifiedAt sql.NullTime
			bio        sql.NullString
			phone      sql.NullString
			avatar     sql.NullString
		)

		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.Role,
			&u.Status,
			&verifiedAt,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.FullName,
			&bio,
			&phone,
			&avatar,
		)
		if err != nil {
			return nil, 0, err
		}

		if verifiedAt.Valid {
			t := verifiedAt.Time
			u.EmailVerifiedAt = &t
		}
		if bio.Valid {
			u.Bio = &bio.String
		}
		if phone.Valid {
			u.PhoneNumber = &phone.String
		}
		if avatar.Valid {
			u.AvatarURL = &avatar.String
		}
		u.Roles = []string{}

		users = append(users, &u)
		userIDs = append(userIDs, u.ID)
		userMap[u.ID] = &u
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// 3. Batch load roles for fetched users
	if len(userIDs) > 0 {
		placeholders := make([]string, len(userIDs))
		for i := range userIDs {
			placeholders[i] = "?"
		}
		rolesSQL := fmt.Sprintf(`
			SELECT ur.user_id, r.name
			FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id IN (%s)
			ORDER BY r.name ASC
		`, strings.Join(placeholders, ","))

		roleRows, err := r.db.QueryContext(ctx, rolesSQL, userIDs...)
		if err == nil {
			defer roleRows.Close()
			for roleRows.Next() {
				var uid, rname string
				if err := roleRows.Scan(&uid, &rname); err == nil {
					if user, ok := userMap[uid]; ok {
						user.Roles = append(user.Roles, rname)
					}
				}
			}
		}

		// If user has no roles in user_roles table, fallback to legacy u.Role
		for _, u := range users {
			if len(u.Roles) == 0 && u.Role != "" {
				u.Roles = []string{u.Role}
			}
		}
	}

	return users, total, nil
}

func (r *sqlUserRepository) GetUserRoles(ctx context.Context, userID string) ([]*models.Role, error) {
	rows, err := r.q.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*models.Role, len(rows))
	for i, row := range rows {
		role := &models.Role{
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: row.DisplayName,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
		if row.Description.Valid {
			role.Description = &row.Description.String
		}
		result[i] = role
	}
	return result, nil
}

func (r *sqlUserRepository) GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	return r.q.GetUserRoleNames(ctx, userID)
}

func (r *sqlUserRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	err := r.q.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if IsDuplicateKeyError(err) {
			return nil // idempotent
		}
		return err
	}
	return nil
}

func (r *sqlUserRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	return r.q.RemoveRoleFromUser(ctx, sqlc.RemoveRoleFromUserParams{
		UserID: userID,
		RoleID: roleID,
	})
}

func (r *sqlUserRepository) SetRoles(ctx context.Context, userID string, roleIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	// 1. Remove existing roles
	if err := qtx.RemoveAllRolesFromUser(ctx, userID); err != nil {
		return err
	}

	// 2. Assign new roles
	now := time.Now().UTC()
	for _, roleID := range roleIDs {
		if err := qtx.AssignRoleToUser(ctx, sqlc.AssignRoleToUserParams{
			UserID:    userID,
			RoleID:    roleID,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}

	// 3. Synchronize legacy users.role column with highest assigned role (SEC-07)
	primaryRole := sqlc.UsersRoleUser
	for _, roleID := range roleIDs {
		roleRow, err := qtx.GetRoleByID(ctx, roleID)
		if err == nil {
			if roleRow.Name == models.RoleSuperAdmin {
				primaryRole = sqlc.UsersRoleSuperAdmin
				break
			}
			if roleRow.Name == models.RoleAdmin && primaryRole != sqlc.UsersRoleSuperAdmin {
				primaryRole = sqlc.UsersRoleAdmin
			}
		}
	}

	if err := qtx.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{
		ID:        userID,
		Role:      primaryRole,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqlUserRepository) CountActiveSuperAdmins(ctx context.Context) (int64, error) {
	return r.q.CountActiveSuperAdmins(ctx)
}

func (r *sqlUserRepository) CountActiveSuperAdminsExcluding(ctx context.Context, userID string) (int64, error) {
	return r.q.CountActiveSuperAdminsExcludingUser(ctx, userID)
}

func (r *sqlUserRepository) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	return r.q.GetUserPermissionsByUserID(ctx, userID)
}

func (r *sqlUserRepository) CheckUserHasRole(ctx context.Context, userID, roleName string) (bool, error) {
	count, err := r.q.CheckUserHasRole(ctx, sqlc.CheckUserHasRoleParams{
		UserID: userID,
		Name:   roleName,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sqlUserRepository) HasAssociatedData(ctx context.Context, userID string) (bool, error) {
	query := `
		SELECT (
			(SELECT COUNT(*) FROM articles WHERE author_id = ?) +
			(SELECT COUNT(*) FROM media WHERE uploaded_by = ?) +
			(SELECT COUNT(*) FROM article_revisions WHERE user_id = ?)
		)
	`
	var total int64
	if err := r.db.QueryRowContext(ctx, query, userID, userID, userID).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
}
