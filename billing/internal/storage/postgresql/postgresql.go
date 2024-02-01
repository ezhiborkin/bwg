package postgresql

import (
	"billing/internal/lib/balance"
	"billing/internal/lib/transaction"
	"database/sql"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	db *sql.DB
}

func New(dataSourceName string) (*Storage, error) {
	const op = "storage.postgresql.New"

	// TODO
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	const op = "storage.postgresql.Close"

	// TODO
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) CreateWallet() (string, string, error) {
	const op = "storage.postgresql.CreateWallet"

	stmt, err := s.db.Prepare("INSERT INTO wallets (id, account_id) VALUES ($1, $2)")
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	id := gofakeit.UUID()
	account_id := gofakeit.Email()
	_, err = stmt.Exec(id, account_id)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return id, account_id, nil
}

func (s *Storage) GetBalance(walletID string) ([]balance.BalanceResponse, error) {
	const op = "storage.postgresql.GetBalance"

	var balances []balance.BalanceResponse

	stmt, err := s.db.Prepare("SELECT sub.wallet_id, sub.currency, sub.amount, sub.frozen_amount FROM subwallets sub JOIN wallets w ON sub.wallet_id = w.id WHERE w.id = $1")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(walletID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var b balance.BalanceResponse
		if err := rows.Scan(&b.WalletID, &b.Currency, &b.Amount, &b.FrozenAmount); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		balances = append(balances, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return balances, nil
}

func (s *Storage) GetTransaction(id int) (*transaction.Transaction, error) {
	const op = "storage.postgresql.GetWallet"

	var status transaction.Transaction

	stmt, err := s.db.Prepare("SELECT wallet_id, amount, type, date_created, status FROM transactions WHERE id = $1")
	if err != nil {
		return &transaction.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(id).Scan(&status.WalletID, &status.Amount, &status.Type, &status.DateCreated, &status.Status)
	if err != nil {
		return &transaction.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}
	return &status, nil
}

func (s *Storage) PerformInvoiceTransaction(walletID string, transactionType string, currency string, amount float64) (int, error) {
	const op = "storage.postgresql.PerformTransaction"

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		// If there's an error, roll back the transaction
		if err != nil {
			tx.Rollback()
			return
		}

		// If no error occurred, commit the transaction
		err = tx.Commit()
		if err != nil {
			fmt.Println(fmt.Errorf("error committing transaction -- %s: %w", op, err))
			return
			// Handle commit error, log or return it as needed
		}
	}()

	// Step 1: Top up frozen balance
	err = s.invoice(walletID, currency, amount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Step 2: Create transaction
	transactionID, err := s.createTransaction(walletID, amount, transactionType)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Step 3: Add frozen balance to regular balance
	err = s.addFBalanceInvoice(walletID, currency)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Step 4: Change transaction status
	err = s.editTransaction(transactionID, "Success")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return transactionID, nil
}

func (s *Storage) PerformWithdrawTransaction(walletID string, transactionType string, currency string, amount float64) (int, error) {
	const op = "storage.postgresql.PerformTransaction"

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		err = tx.Commit()
		if err != nil {
			fmt.Println(fmt.Errorf("error committing transaction -- %s: %w", op, err))
			return
			// Handle commit error, log or return it as needed
		}
	}()

	// Step 1: Top up frozen balance
	err = s.withdraw(walletID, currency, amount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Step 2: Create transaction
	transactionID, err := s.createTransaction(walletID, amount, transactionType)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Step 3: Add frozen balance to regular balance
	err = s.addFBalanceWithdraw(walletID, currency, transactionID, amount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return transactionID, nil
}

func (s *Storage) invoice(walletID string, currency string, amount float64) error {
	const op = "storage.postgresql.Invoice"

	// Check if a subwallet already exists for the given wallet_id and currency
	var existingAmount float64
	err := s.db.QueryRow("SELECT frozen_amount FROM subwallets WHERE wallet_id = $1 AND currency = $2", walletID, currency).Scan(&existingAmount)
	if err == nil {
		// Subwallet already exists, update the frozen_amount
		newAmount := existingAmount + amount
		_, err = s.db.Exec("UPDATE subwallets SET frozen_amount = $1 WHERE wallet_id = $2 AND currency = $3", newAmount, walletID, currency)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
	} else if err != sql.ErrNoRows {
		// An error occurred during the SELECT query
		return fmt.Errorf("%s: %w", op, err)
	}

	// Subwallet does not exist, insert a new one
	stmt, err := s.db.Prepare("INSERT INTO subwallets (wallet_id, currency, amount, frozen_amount) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(walletID, currency, 0.0, amount)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) withdraw(walletID string, currency string, amount float64) error {
	const op = "storage.postgresql.Withdraw"

	// Check if a subwallet already exists for the given wallet_id and currency
	var existingAmount float64
	err := s.db.QueryRow("SELECT frozen_amount FROM subwallets WHERE wallet_id = $1 AND currency = $2", walletID, currency).Scan(&existingAmount)
	if err == nil {
		// Subwallet already exists, check if there is sufficient balance for withdrawal

		// if existingAmount >= amount {
		newAmount := existingAmount - amount
		_, err := s.db.Exec("UPDATE subwallets SET frozen_amount = $1 WHERE wallet_id = $2 AND currency = $3", newAmount, walletID, currency)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		return nil
		// } else {
		// 	return fmt.Errorf("%s: insufficient balance for withdrawal", op)
		// }
	} else if err != sql.ErrNoRows {
		// An error occurred during the SELECT query
		return fmt.Errorf("%s: %w", op, err)
	}

	// Subwallet does not exist, cannot withdraw
	return fmt.Errorf("%s: subwallet does not exist for withdrawal", op)
}

func (s *Storage) createTransaction(walletID string, amount float64, typeO string) (int, error) {
	const op = "storage.postgresql.CreateWallet"

	var lastInsertId int

	stmt, err := s.db.Prepare("INSERT INTO transactions (wallet_id, amount, type, date_created, status) VALUES ($1, $2, $3, $4, $5) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(walletID, amount, typeO, time.Now(), "Created").Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) editTransaction(id int, status string) error {
	const op = "storage.postgresql.EditWallet"

	stmt, err := s.db.Prepare("UPDATE transactions SET status = $1 WHERE id = $2")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(status, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) addFBalanceInvoice(walletID string, currency string) error {
	const op = "storage.postgresql.MoveFBalance"

	stmt, err := s.db.Prepare("UPDATE subwallets SET amount = amount + frozen_amount, frozen_amount = 0 WHERE wallet_id = $1 AND currency = $2")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(walletID, currency)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) addFBalanceWithdraw(walletID string, currency string, transactionID int, frozen_amount float64) error {
	const op = "storage.postgresql.MoveFBalance"

	var amount float64
	err := s.db.QueryRow("SELECT amount FROM subwallets WHERE wallet_id = $1 AND currency = $2", walletID, currency).Scan(&amount)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if amount-frozen_amount >= 0 {
		stmt, err := s.db.Prepare("UPDATE subwallets SET amount = amount + frozen_amount, frozen_amount = frozen_amount + $1 WHERE wallet_id = $2 AND currency = $3")
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(frozen_amount, walletID, currency)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = s.editTransaction(transactionID, "Success")
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

	} else {
		stmt, err := s.db.Prepare("UPDATE subwallets SET frozen_amount = frozen_amount + $1 WHERE wallet_id = $2 AND currency = $3")
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		defer stmt.Close()

		_, err = stmt.Exec(frozen_amount, walletID, currency)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = s.editTransaction(transactionID, "Error")
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	return nil
}
