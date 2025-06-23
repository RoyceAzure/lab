package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDbConn(t *testing.T) {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(t, err)
	require.NotNil(t, db)
}
