package order

import (
	"context"
	"fmt"
	"print3d-order-bot/internal/pkg"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Repo interface {
	NewOrderOpenTx(ctx context.Context, order DBOrder, files []TGOrderFile) (*sqlx.Tx, error)
	NewOrderCloseTX(tx *sqlx.Tx) error
	NewOrderRollbackTX(tx *sqlx.Tx) error
	GetOrders(ctx context.Context, getActive bool) ([]DBOrder, error)
	DeleteOrder(ctx context.Context, orderID int) error
}

type DefaultRepo struct {
	db *sqlx.DB
}

func NewDefaultRepo(db *sqlx.DB) Repo {
	return &DefaultRepo{db: db}
}

func (d *DefaultRepo) NewOrderOpenTx(ctx context.Context, order DBOrder, files []TGOrderFile) (*sqlx.Tx, error) {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to begin transaction",
				Err:   err,
			}
	}

	orderID, err := d.insertOrders(ctx, order, err, tx)
	if err != nil {
		return nil, err
	}

	query := `update orders set folder_path = ? where order_id = ?`
	path := createFolderPath(order.ClientName, order.CreatedAt, int(orderID))
	if _, err := tx.ExecContext(ctx, query, &path); err != nil {
		if err := tx.Rollback(); err != nil {
			return nil,
				&pkg.ErrDBProcedure{
					Cause: "failed to rollback transaction",
					Err:   err,
				}
		}
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to execute query",
				Info:  fmt.Sprintf("query: %s", query),
				Err:   err,
			}
	}

	dbFiles := make([]DBOrderFile, len(files))
	for i, file := range files {
		dbFiles[i] = DBOrderFile{
			FileName: file.FileName,
			TgFileID: file.FileID,
			OrderID:  int(orderID),
		}
	}

	query = `insert into order_files (file_name, tg_file_id, order_id) values (:file_name, :tg_file_id, :order_id)`
	if _, err := tx.NamedExecContext(ctx, query, &dbFiles); err != nil {
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to insert file data",
				Info:  fmt.Sprintf("query: %s", query),
				Err:   err,
			}
	}

	return tx, nil
}

func (d *DefaultRepo) NewOrderCloseTX(tx *sqlx.Tx) error {
	if err := tx.Commit(); err != nil {
		if err := tx.Rollback(); err != nil {
			return &pkg.ErrDBProcedure{
				Cause: "failed to rollback transaction",
				Err:   err,
			}
		}
		return &pkg.ErrDBProcedure{
			Cause: "failed to commit transaction",
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) NewOrderRollbackTX(tx *sqlx.Tx) error {
	if err := tx.Rollback(); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to rollback transaction",
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) GetOrders(ctx context.Context, getActive bool) ([]DBOrder, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`select * from orders`)

	if getActive {
		queryBuilder.WriteString(` WHERE order_status = 0`)
	}
	query := queryBuilder.String()

	var orders []DBOrder
	if err := d.db.SelectContext(ctx, &orders, query); err != nil {
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to select orders",
				Info:  fmt.Sprintf("query: %s", query),
				Err:   err,
			}
	}
	return orders, nil
}

func (d *DefaultRepo) DeleteOrder(ctx context.Context, orderID int) error {
	query := `delete from orders where order_id = ?`
	if _, err := d.db.ExecContext(ctx, query, orderID); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to delete order",
			Info:  fmt.Sprintf("orderID: %d", orderID),
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) insertOrders(ctx context.Context, order DBOrder, err error, tx *sqlx.Tx) (int, error) {
	query := `insert into orders (order_status, client_name, created_at, folder_path) values  (:order_status, :client_name, :created_at, :folder_path)`
	result, err := tx.NamedExecContext(ctx, query, &order)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return -1,
				&pkg.ErrDBProcedure{
					Cause: "failed to rollback transaction",
					Err:   err,
				}
		}
		return -1,
			&pkg.ErrDBProcedure{
				Cause: "failed to execute query",
				Info:  fmt.Sprintf("query: %s", query),
				Err:   err,
			}
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return -1,
				&pkg.ErrDBProcedure{
					Cause: "failed to rollback transaction",
					Err:   err,
				}
		}
		return -1,
			&pkg.ErrDBProcedure{
				Cause: "failed to retrieve last insert ID",
				Err:   err,
			}
	}

	return int(orderID), nil
}
