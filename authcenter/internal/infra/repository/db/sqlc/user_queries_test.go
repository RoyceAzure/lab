package sqlc

import (
	"context"
	"testing"
	"time"

	pgutil "github.com/RoyceAzure/rj/util/pg_util"
	"github.com/RoyceAzure/rj/util/random"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Helper function to create a random user for testing
// You would typically have more specific creation helpers or use a library for fixtures.
func createRandomUser(t *testing.T) (User, CreateUserParams) {
	t.Helper()

	arg := CreateUserParams{
		ID:        pgutil.UUIDToPgUUIDV5(uuid.New()),
		Email:     random.RandomEmail(),
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, arg.IsAdmin, user.IsAdmin)

	require.NotZero(t, user.CreatedAt)

	return user, arg
}

func TestCreateUser(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not configured, skipping TestCreateUser")
	}
	createRandomUser(t)
}

func TestGetUserByID(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not configured, skipping TestGetUserByID")
	}
	createdUser, _ := createRandomUser(t)
	t.Cleanup(func() {
		testQueries.DeleteUser(context.Background(), createdUser.ID)
	})

	retrievedUser, err := testQueries.GetUserByID(context.Background(), createdUser.ID)
	require.NoError(t, err)
	require.NotEmpty(t, retrievedUser)

	require.Equal(t, createdUser.ID, retrievedUser.ID)
	require.Equal(t, createdUser.Email, retrievedUser.Email)
	require.Equal(t, createdUser.Name, retrievedUser.Name)
	require.Equal(t, createdUser.PasswordHash, retrievedUser.PasswordHash)
	require.Equal(t, createdUser.IsAdmin, retrievedUser.IsAdmin)
	require.WithinDuration(t, createdUser.CreatedAt, retrievedUser.CreatedAt, time.Second)

}

func TestGetUserByEmail(t *testing.T) {
	if testQueries == nil {
		t.Skip("Database not configured, skipping TestGetUserByEmail")
	}
	createdUser, _ := createRandomUser(t)
	t.Cleanup(func() {
		testQueries.DeleteUser(context.Background(), createdUser.ID)
	})
	retrievedUser, err := testQueries.GetUserByEmail(context.Background(), createdUser.Email)
	require.NoError(t, err)
	require.NotEmpty(t, retrievedUser)

	require.Equal(t, createdUser.ID, retrievedUser.ID)
}
