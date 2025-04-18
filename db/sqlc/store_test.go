package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func convertNullInt64ToInt64(data sql.NullInt64) int64{
	return data.Int64
}

func TestTransferTxDeadlock(t *testing.T){
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:",account1.Balance, account2.Balance)

	// run n concurrent transfer transactions
	n := 10
	amount := int64(10)

	errs := make(chan error)


	for i := 0 ; i < n ; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i % 2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}
		go func ()  {
			ctx := context.Background()
			_, err := store.TransferTx(ctx, TransferParams{
				FromAccountID: fromAccountID,
				ToAccountID: toAccountID,
				Amount: amount,
			})	

			errs <- err
		}()
	}

	// check results

	for i := 0 ; i < n ; i++{
		err := <- errs
		require.NoError(t, err)

		
	}

	// check the final updated balances
	updateAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:",updateAccount1.Balance,updateAccount2.Balance)
	require.Equal(t, account1.Balance, updateAccount1.Balance)
	require.Equal(t, account2.Balance, updateAccount2.Balance)
}

func TestTransferTx(t *testing.T){
	store := NewStore(testDB)

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:",account1.Balance,account2.Balance)

	// run n concurrent transfer transactions
	n := 2
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResult)


	for i := 0 ; i < n ; i++ {
		go func ()  {
			ctx := context.Background()
			result, err := store.TransferTx(ctx, TransferParams{
				FromAccountID: account1.ID,
				ToAccountID: account2.ID,
				Amount: amount,
			})	

			errs <- err
			results <- result
		}()
	}

	// check results
	existed := make(map[int]bool)
	for i := 0 ; i < n ; i++{
		err := <- errs
		require.NoError(t, err)

		result := <- results
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, convertNullInt64ToInt64(transfer.FromAccountID))
		require.Equal(t, account2.ID, convertNullInt64ToInt64(transfer.ToAccountID))
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = store.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, convertNullInt64ToInt64(fromEntry.AccountID))
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, convertNullInt64ToInt64(toEntry.AccountID))
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAcount := result.FromAccount
		require.NotEmpty(t, fromAcount)
		require.Equal(t, account1.ID, fromAcount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)
		
		// check account's balance
		fmt.Println(">> tx:",fromAcount.Balance,toAccount.Balance)
		diff1 := account1.Balance - fromAcount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1 % amount == 0) 

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final updated balances
	updateAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updateAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:",updateAccount1.Balance,updateAccount2.Balance)
	require.Equal(t, account1.Balance - int64(n) * amount, updateAccount1.Balance)
	require.Equal(t, account2.Balance + int64(n) * amount, updateAccount2.Balance)
}