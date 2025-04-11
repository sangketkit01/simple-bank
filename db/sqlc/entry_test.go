package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sangketkit01/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T) Entry {
	arg := CreateEntryParams{
		AccountID: sql.NullInt64{Int64: util.RandomInt(1, 30), Valid: true},
		Amount:    util.RandomMoney(),
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.Equal(t, arg.AccountID, entry.AccountID)
	require.Equal(t, arg.Amount, entry.Amount)
	require.NotZero(t, entry.ID)
	require.NotZero(t, entry.CreatedAt)
	require.NotZero(t, entry.Amount)


	return entry
}

func TestCreateEntry(t *testing.T) {
   createRandomEntry(t)
}

func TestGetEntry(t *testing.T){
	entry := createRandomEntry(t)

	fetchedEntry, err := testQueries.GetEntry(context.Background(), entry.ID)
	require.NoError(t, err)
	require.NotEmpty(t, fetchedEntry)

	require.Equal(t, entry.ID, fetchedEntry.ID)
    require.Equal(t, entry.AccountID, fetchedEntry.AccountID)
    require.Equal(t, entry.Amount, fetchedEntry.Amount)
	require.WithinDuration(t, entry.CreatedAt, fetchedEntry.CreatedAt, time.Second)
}

func TestListsEntry(t *testing.T){
	for i := 0 ; i < 5 ; i++{
		createRandomEntry(t)
	}

	arg := ListEntriesParams{
		Limit: 5,
		Offset: 5,
	}

	entries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries{
		require.NotEmpty(t, entry)
	}
}