package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	domain "workout-app/internal/domain/user"
	repo "workout-app/internal/repository/interfaces"
	authuc "workout-app/internal/usecase/auth"
	jwtsvc "workout-app/pkg/jwt"
)

// ==== Fakes for repositories and services ====

type fakeUserRepo struct {
	usersByEmail map[string]*domain.User
}

func (r *fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, repo.ErrNotFound
}
func (r *fakeUserRepo) GetByUsername(context.Context, string) (*domain.User, error) {
	return nil, repo.ErrNotFound
}
func (r *fakeUserRepo) Update(context.Context, *domain.User) error   { return nil }
func (r *fakeUserRepo) SoftDelete(context.Context, uuid.UUID) error  { return nil }
func (r *fakeUserRepo) List(context.Context) ([]*domain.User, error) { return nil, nil }
func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.usersByEmail[email]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return u, nil
}

type fakeEmailVerifRepo struct {
	deletedForUser uuid.UUID
	created        *domain.EmailVerification
}

func (r *fakeEmailVerifRepo) Create(_ context.Context, v *domain.EmailVerification) error {
	r.created = v
	return nil
}
func (r *fakeEmailVerifRepo) GetActiveByUserID(context.Context, uuid.UUID) (*domain.EmailVerification, error) {
	return nil, repo.ErrNotFound
}
func (r *fakeEmailVerifRepo) IncrementAttempts(context.Context, int64) error { return nil }
func (r *fakeEmailVerifRepo) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	r.deletedForUser = userID
	return nil
}

type fakeEmailSender struct {
	sentTo string
	code   string
}

func (s *fakeEmailSender) SendEmailVerificationCode(_ context.Context, email, code string) error {
	s.sentTo = email
	s.code = code
	return nil
}

// fakeJWT реализует jwtsvc.Service, но для этих тестов не используется.
type fakeJWT struct{}

func (f *fakeJWT) GenerateAccessToken(*domain.User) (string, error)          { return "", nil }
func (f *fakeJWT) GenerateRefreshToken(*domain.User) (string, string, error) { return "", "", nil }
func (f *fakeJWT) ParseAccessToken(string) (*jwtsvc.Claims, error)           { return &jwtsvc.Claims{}, nil }
func (f *fakeJWT) ParseRefreshToken(string) (*jwtsvc.Claims, error)          { return &jwtsvc.Claims{}, nil }

// ==== Tests for ResendVerificationCode ====

func TestResendVerificationCode_NoUser_SilentSuccess(t *testing.T) {
	userRepo := &fakeUserRepo{usersByEmail: map[string]*domain.User{}}
	verifRepo := &fakeEmailVerifRepo{}
	sender := &fakeEmailSender{}

	svc := authuc.NewService(userRepo, verifRepo, &fakeJWT{}, sender, 15*time.Minute, 5, 6)

	err := svc.ResendVerificationCode(context.Background(), "nouser@example.com")
	require.NoError(t, err)
	require.Empty(t, sender.sentTo)
	require.Nil(t, verifRepo.created)
}

func TestResendVerificationCode_AlreadyVerified(t *testing.T) {
	u := &domain.User{
		ID:              uuid.New(),
		Email:           "verified@example.com",
		IsEmailVerified: true,
	}
	userRepo := &fakeUserRepo{usersByEmail: map[string]*domain.User{
		u.Email: u,
	}}
	verifRepo := &fakeEmailVerifRepo{}
	sender := &fakeEmailSender{}

	svc := authuc.NewService(userRepo, verifRepo, &fakeJWT{}, sender, 15*time.Minute, 5, 6)

	err := svc.ResendVerificationCode(context.Background(), u.Email)
	require.Error(t, err)
	require.ErrorIs(t, err, authuc.ErrEmailAlreadyVerified)
	require.Empty(t, sender.sentTo)
	require.Nil(t, verifRepo.created)
}

func TestResendVerificationCode_Unverified_CreatesNewCodeAndDeletesOld(t *testing.T) {
	u := &domain.User{
		ID:              uuid.New(),
		Email:           "unverified@example.com",
		IsEmailVerified: false,
	}
	userRepo := &fakeUserRepo{usersByEmail: map[string]*domain.User{
		u.Email: u,
	}}
	verifRepo := &fakeEmailVerifRepo{}
	sender := &fakeEmailSender{}

	svc := authuc.NewService(userRepo, verifRepo, &fakeJWT{}, sender, 15*time.Minute, 5, 6)

	err := svc.ResendVerificationCode(context.Background(), u.Email)
	require.NoError(t, err)

	require.Equal(t, u.ID, verifRepo.deletedForUser)
	require.NotNil(t, verifRepo.created)
	require.Equal(t, u.ID, verifRepo.created.UserID)
	require.Equal(t, u.Email, sender.sentTo)
	require.NotEmpty(t, sender.code)
}
