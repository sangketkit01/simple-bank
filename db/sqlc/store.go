package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// Store provides all functions to execute db queries and transactions
type Store interface{
	Querier
	TransferTx(ctx context.Context, arg TransferParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
}

type SQLStore struct{
	*Queries
	db *sql.DB
}

// NewStore creates a new Strore
func NewStore(db *sql.DB) Store{
	db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)
	return &SQLStore{
		db: db,
		Queries: New(db),
	}
}

// exexTx executes a function within a database transaction
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error{
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil{
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil{
		if rbErr := tx.Rollback() ; rbErr != nil{
			return fmt.Errorf("tx err: %v, rb err: %v",err,rbErr)
		}

		return err
	}

	return tx.Commit() 
}

// TransferParams contains the input parameters of the transfer transaction
type TransferParams struct{
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID int64 `json:"to_account_id"`
	Amount int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct{
	Transfer Transfer `json:"transfer"`
	FromAccount Account `json:"from_account"`
	ToAccount Account `json:"to_account"`
	FromEntry Entry `json:"from_entry"`
	ToEntry Entry `json:"to_entry"`
}


// TransferTx performs a money transfer from one account to other.
// It creates a transfer record, add account entries, and update account's balance within a single database transaction
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferParams) (TransferTxResult, error){
	var result TransferTxResult

	err := store.execTx(ctx, func (q *Queries) error  {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: sql.NullInt64{Int64: arg.FromAccountID, Valid: true},
			ToAccountID: sql.NullInt64{Int64: arg.ToAccountID, Valid: true},
			Amount: arg.Amount,
		})
		if err != nil{
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: sql.NullInt64{Int64: arg.FromAccountID, Valid: true},
			Amount: -arg.Amount,
		})	

		if err != nil{
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: sql.NullInt64{Int64: arg.ToAccountID, Valid: true},
			Amount: arg.Amount,
		})	

		if err != nil{
			return err
		}

		// get account -> update its balance
		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount,err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		}else{
			result.ToAccount, result.FromAccount,err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}

		if err != nil{
			return err
		}

		return nil
	})

	return result, err
}

func addMoney(
    ctx context.Context,
    q *Queries,
    accountID1 int64,
    amount1 int64,
    accountID2 int64,
    amount2 int64,
) (account1 Account, account2 Account, err error) {
	context, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()


    account1, err = q.AddAccountBalance(context, AddAccountBalanceParams{
        ID:     accountID1,
        Amount: amount1,
    })

    if err != nil {
        return
    }
    
    account2, err = q.AddAccountBalance(context, AddAccountBalanceParams{
        ID:     accountID2,
        Amount: amount2,
    })
    if err != nil {
        return
    }
    
    return
}

func addMoneyWithRetry(ctx context.Context, q *Queries, accountID1, amount1, accountID2, amount2 int64) (Account, Account, error) {
    var account1, account2 Account
    var err error
    
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        account1, account2, err = addMoney(ctx, q, accountID1, amount1, accountID2, amount2)
        if err == nil {
            return account1, account2, nil
        }
        
        // ตรวจสอบว่าเป็น error ที่ควร retry หรือไม่
        if strings.Contains(err.Error(), "bad connection") || strings.Contains(err.Error(), "canceling statement") {
            log.Printf("Retrying operation (%d/%d) after error: %v", i+1, maxRetries, err)
            time.Sleep(time.Duration(100*(i+1)) * time.Millisecond) // Exponential backoff
            continue
        }
        
        // ถ้าเป็น error ประเภทอื่น ให้ return เลย
        return account1, account2, err
    }
    
    return account1, account2, fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}