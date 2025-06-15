package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDbDao_InitMigrate(t *testing.T) {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(t, err)
	require.NotNil(t, db)

	dao := NewDbDao(db)
	err = dao.InitMigrate()
	require.NoError(t, err)

}
