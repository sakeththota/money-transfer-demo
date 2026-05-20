package balances

import "database/sql"

func GetAllBalances(db *sql.DB) (map[string]float64, error) {
	rows, err := db.Query("SELECT name, balance FROM accounts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make(map[string]float64)
	for rows.Next() {
		var name string
		var balance float64
		if err := rows.Scan(&name, &balance); err != nil {
			return nil, err
		}
		balances[name] = balance
	}
	return balances, rows.Err()
}

func Debit(db *sql.DB, accountName string, amount float64) error {
	_, err := db.Exec(
		"UPDATE accounts SET balance = balance - ? WHERE name = ?",
		amount, accountName,
	)
	return err
}

func Credit(db *sql.DB, accountName string, amount float64) error {
	_, err := db.Exec(
		"UPDATE accounts SET balance = balance + ? WHERE name = ?",
		amount, accountName,
	)
	return err
}

func ResetBalances(db *sql.DB) error {
	for name, balance := range seedData {
		if _, err := db.Exec(
			"UPDATE accounts SET balance = ? WHERE name = ?",
			balance, name,
		); err != nil {
			return err
		}
	}
	return nil
}
