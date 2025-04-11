package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sangketkit01/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomTransfer(t *testing.T) Transfer{
	arg := CreateTransferParams{
		FromAccountID: sql.NullInt64{Int64: util.RandomInt(10, 13), Valid: true},
		ToAccountID: sql.NullInt64{Int64: util.RandomInt(21, 23), Valid: true},
		Amount: util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.Equal(t, transfer.FromAccountID, arg.FromAccountID)
	require.Equal(t, transfer.ToAccountID, arg.ToAccountID)
	require.Equal(t, transfer.Amount, arg.Amount)

	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.Amount)

	return transfer
}

func TestCreateTransfer(t *testing.T){
	createRandomTransfer(t)
}

func TestGetTransfer(t *testing.T){
	transfers := createRandomTransfer(t)
	
	fetchedTransfer, err := testQueries.GetTransfer(context.Background(), transfers.ID)
	require.NoError(t, err)
	require.NotEmpty(t, fetchedTransfer)

	require.Equal(t, transfers.FromAccountID, fetchedTransfer.FromAccountID)
	require.Equal(t, transfers.ToAccountID, fetchedTransfer.ToAccountID)
	require.Equal(t, transfers.Amount, fetchedTransfer.Amount)
	require.WithinDuration(t, transfers.CreatedAt, fetchedTransfer.CreatedAt, time.Second)
}

func TestListTransfers(t *testing.T){
	for i := 0 ; i < 3 ; i++{
		createRandomTransfer(t)
	}

	arg := ListTransfersParams{
		Limit: 5,
		Offset: 5,
	}

	transfers, err := testQueries.ListTransfers(context.Background(),arg)
	require.NoError(t, err)
	require.Len(t, transfers, 5)
	
	for _, transfer := range transfers{
		require.NotEmpty(t, transfer)
	}
}
