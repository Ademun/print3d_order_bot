package order

import (
	"context"
	"fmt"
	"print3d-order-bot/internal/pkg"
	"print3d-order-bot/internal/pkg/model"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo interface {
	NewOrderOpenTx(ctx context.Context, order DBOrder, files []model.TGOrderFile) (string, pgx.Tx, error)
	NewOrderCloseTX(ctx context.Context, tx pgx.Tx) error
	NewOrderRollbackTX(ctx context.Context, tx pgx.Tx) error
	NewOrderFiles(ctx context.Context, orderID int, files []model.TGOrderFile) error
	GetOrders(ctx context.Context, getActive bool) ([]DBOrder, error)
	GetOrdersIDs(ctx context.Context, getActive bool) ([]int, error)
	GetOrderByID(ctx context.Context, orderID int) (*DBOrder, error)
	GetOrderFiles(ctx context.Context, orderID int) ([]DBOrderFile, error)
	UpdateOrderStatus(ctx context.Context, orderID int, status model.OrderStatus) error
	DeleteOrder(ctx context.Context, orderID int) error
	DeleteOrderFiles(ctx context.Context, orderID int, filenames []string) error
}

type DefaultRepo struct {
	pool    *pgxpool.Pool
	builder squirrel.StatementBuilderType
}

func NewDefaultRepo(pool *pgxpool.Pool) Repo {
	return &DefaultRepo{
		pool:    pool,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (d *DefaultRepo) NewOrderOpenTx(ctx context.Context, order DBOrder, files []model.TGOrderFile) (string, pgx.Tx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return "", nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to begin transaction",
				Info:  "NewOrderOpenTx",
				Err:   err,
			}
	}

	orderID, err := d.insertOrder(ctx, order, tx)
	if err != nil {
		return "", nil, err
	}

	path := createFolderPath(order.ClientName, order.CreatedAt, orderID)
	stmt := d.builder.Update("orders").
		Set("folder_path", path).
		Where(squirrel.Eq{"order_id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return "", nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "NewOrderOpenTx",
			Err:   err,
		}
	}

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return "", nil,
				&pkg.ErrDBProcedure{
					Cause: "failed to rollback transaction",
					Info:  "NewOrderOpenTx",
					Err:   err,
				}
		}
		return "", nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to execute query",
				Info:  fmt.Sprintf("NewOrderOpenTx; query: %s", query),
				Err:   err,
			}
	}

	if len(files) == 0 {
		return path, tx, nil
	}

	builder := d.builder.Insert("order_files").
		Columns("file_name", "tg_file_id", "order_id")
	for _, file := range files {
		builder = builder.Values(file.FileName, file.TGFileID, orderID)
	}
	query, args, err = builder.ToSql()
	if err != nil {
		return "", nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "NewOrderOpenTx",
			Err:   err,
		}
	}

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return "", nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to insert file data",
				Info:  fmt.Sprintf("NewOrderOpenTx; query: %s", query),
				Err:   err,
			}
	}

	return path, tx, nil
}

func (d *DefaultRepo) NewOrderCloseTX(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return &pkg.ErrDBProcedure{
				Cause: "failed to rollback transaction",
				Info:  "NewOrderCloseTx",
				Err:   err,
			}
		}
		return &pkg.ErrDBProcedure{
			Cause: "failed to commit transaction",
			Info:  "NewOrderCloseTx",
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) NewOrderRollbackTX(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(ctx); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to rollback transaction",
			Info:  "NewOrderRollbackTx",
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) NewOrderFiles(ctx context.Context, orderID int, files []model.TGOrderFile) error {
	builder := d.builder.Insert("order_files").
		Columns("file_name", "tg_file_id", "order_id")
	for _, file := range files {
		builder = builder.Values(file.FileName, file.TGFileID, orderID)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "NewOrderFiles",
			Err:   err,
		}
	}

	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to execute query",
			Info:  fmt.Sprintf("NewOrderFiles; query: %s", query),
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) GetOrders(ctx context.Context, getActive bool) ([]DBOrder, error) {
	stmt := d.builder.Select("*").From("orders").OrderBy("created_at")
	if getActive {
		stmt = stmt.Where(squirrel.Or{
			squirrel.Eq{"order_status": model.StatusActive},
			squirrel.And{
				squirrel.NotEq{"closed_at": nil},
				squirrel.Expr("closed_at >= NOW() - INTERVAL '1 second'"),
			}})
	}
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "GetOrders",
			Err:   err,
		}
	}

	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to select orders",
				Info:  fmt.Sprintf("GetOrders; query: %s", query),
				Err:   err,
			}
	}
	defer rows.Close()

	var orders []DBOrder
	for rows.Next() {
		var order DBOrder
		if err := rows.Scan(&order.OrderID, &order.OrderStatus, &order.ClientName, &order.Cost, &order.Comments, &order.Contacts, &order.Links, &order.CreatedAt, &order.ClosedAt, &order.FolderPath); err != nil {
			return nil, &pkg.ErrDBProcedure{
				Cause: "failed to scan row",
				Info:  fmt.Sprintf("GetOrders; query: %s", query),
				Err:   err,
			}
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (d *DefaultRepo) GetOrdersIDs(ctx context.Context, getActive bool) ([]int, error) {
	stmt := d.builder.Select("order_id").From("orders").OrderBy("created_at")
	if getActive {
		stmt = stmt.Where(squirrel.Or{
			squirrel.Eq{"order_status": model.StatusActive},
			squirrel.And{
				squirrel.NotEq{"closed_at": nil},
				squirrel.Expr("closed_at >= NOW() - INTERVAL '1 second'"),
			}})
	}
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "GetOrdersIDs",
			Err:   err,
		}
	}

	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil,
			&pkg.ErrDBProcedure{
				Cause: "failed to select orders",
				Info:  fmt.Sprintf("GetOrdersIDs; query: %s", query),
				Err:   err,
			}
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, &pkg.ErrDBProcedure{
				Cause: "failed to scan row",
				Info:  fmt.Sprintf("GetOrdersIDs; query: %s", query),
				Err:   err,
			}
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (d *DefaultRepo) GetOrderByID(ctx context.Context, orderID int) (*DBOrder, error) {
	stmt := d.builder.Select("*").From("orders").Where(squirrel.Eq{"order_id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "GetOrderByID",
			Err:   err,
		}
	}

	var order DBOrder
	if err := d.pool.QueryRow(ctx, query, args...).Scan(&order.OrderID, &order.OrderStatus, &order.ClientName, &order.Cost, &order.Comments, &order.Contacts, &order.Links, &order.CreatedAt, &order.ClosedAt, &order.FolderPath); err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to select order",
			Info:  fmt.Sprintf("GetOrderByID; query: %s", query),
			Err:   err,
		}
	}

	return &order, nil
}

func (d *DefaultRepo) GetOrderFiles(ctx context.Context, orderID int) ([]DBOrderFile, error) {
	stmt := d.builder.Select("*").From("order_files").Where(squirrel.Eq{"order_id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "GetOrderFiles",
			Err:   err,
		}
	}
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to select order files",
			Info:  fmt.Sprintf("GetOrderFiles; query: %s", query),
			Err:   err,
		}
	}
	defer rows.Close()

	var orderFiles []DBOrderFile
	for rows.Next() {
		var file DBOrderFile
		if err := rows.Scan(&file.FileName, &file.TgFileID, &file.OrderID); err != nil {
			return nil, &pkg.ErrDBProcedure{
				Cause: "failed to scan row",
				Info:  fmt.Sprintf("GetOrderFiles; query: %s", query),
				Err:   err,
			}
		}
		orderFiles = append(orderFiles, file)
	}

	return orderFiles, nil
}

func (d *DefaultRepo) UpdateOrderStatus(ctx context.Context, orderID int, status model.OrderStatus) error {
	stmt := d.builder.Update("orders").Set("order_status", status)
	switch status {
	case model.StatusClosed:
		stmt = stmt.Set("closed_at", time.Now())
	case model.StatusActive:
		stmt = stmt.Set("closed_at", nil)
	}
	stmt = stmt.Where(squirrel.Eq{"order_id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "UpdateOrderStatus",
			Err:   err,
		}
	}

	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to execute query",
			Info:  fmt.Sprintf("UpdateOrderStatus; query: %s", query),
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) DeleteOrder(ctx context.Context, orderID int) error {
	stmt := d.builder.Delete("orders").Where(squirrel.Eq{"order_id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "DeleteOrder",
			Err:   err,
		}
	}
	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to delete order",
			Info:  fmt.Sprintf("DeleteOrder; orderID: %d", orderID),
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) DeleteOrderFiles(ctx context.Context, orderID int, filenames []string) error {
	stmt := d.builder.Delete("order_files").
		Where(squirrel.And{
			squirrel.Eq{"order_id": orderID},
			squirrel.Eq{"file_name": filenames},
		})
	query, args, err := stmt.ToSql()
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "DeleteOrderFiles",
			Err:   err,
		}
	}

	if _, err := d.pool.Exec(ctx, query, args...); err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to execute query",
			Info:  fmt.Sprintf("DeleteOrderFiles; query: %s", query),
			Err:   err,
		}
	}
	return nil
}

func (d *DefaultRepo) insertOrder(ctx context.Context, order DBOrder, tx pgx.Tx) (int, error) {
	stmt := d.builder.Insert("orders").
		Columns("order_status, client_name, cost, comments, contacts, links, created_at, folder_path").
		Values(order.OrderStatus, order.ClientName, order.Cost, order.Comments, order.Contacts, order.Links, order.CreatedAt, order.FolderPath).
		Suffix("returning order_id")
	query, args, err := stmt.ToSql()
	if err != nil {
		return 0, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "InsertOrder",
			Err:   err,
		}
	}

	var orderID int
	if err := tx.QueryRow(ctx, query, args...).Scan(&orderID); err != nil {
		return 0, &pkg.ErrDBProcedure{
			Cause: "failed to insert order",
			Info:  fmt.Sprintf("InsertOrder; query: %s", query),
			Err:   err,
		}
	}

	return orderID, nil
}
