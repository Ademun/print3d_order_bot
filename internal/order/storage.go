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
	NewOrder(ctx context.Context, order DBOrder, files []DBFile) error
	AddFilesToOrder(ctx context.Context, orderID int, files []DBFile) error
	GetOrdersIDs(ctx context.Context, getActive bool) ([]int, error)
	GetOrderByID(ctx context.Context, orderID int) (*DBOrder, error)
	UpdateOrderStatus(ctx context.Context, orderID int, status model.OrderStatus) error
	DeleteOrder(ctx context.Context, orderID int) error
	GetOrderFiles(ctx context.Context, orderID int) ([]DBFile, error)
	GetOrderFilenames(ctx context.Context, orderID int) ([]string, error)
	DeleteOrderFiles(ctx context.Context, orderID int, filenames []string) error
	UpdateOrderFiles(ctx context.Context, orderID int, files []DBFile) error
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

func (d *DefaultRepo) NewOrder(ctx context.Context, order DBOrder, files []DBFile) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to begin transaction",
			Info:  "NewOrder",
			Err:   err,
		}
	}

	orderID, err := d.insertOrder(ctx, order, tx)
	if err != nil {
		return err
	}

	builder := d.builder.Insert("order_files").
		Columns("name", "checksum", "tg_file_id", "order_id")
	for _, file := range files {
		builder = builder.Values(file.Name, file.Checksum, file.TgFileID, orderID)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		tx.Rollback(ctx)
		return &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "NewOrder",
			Err:   err,
		}
	}

	if _, err := tx.Exec(ctx, query, args...); err != nil {
		tx.Rollback(ctx)
		return &pkg.ErrDBProcedure{
			Cause: "failed to insert file data",
			Info:  fmt.Sprintf("NewOrderOpenTx; query: %s", query),
			Err:   err,
		}
	}

	tx.Commit(ctx)
	return nil
}

func (d *DefaultRepo) insertOrder(ctx context.Context, order DBOrder, tx pgx.Tx) (int, error) {
	stmt := d.builder.Insert("orders").
		Columns("status, client_name, cost, comments, contacts, links, created_at, folder_path").
		Values(order.Status, order.ClientName, order.Cost, order.Comments, order.Contacts, order.Links, order.CreatedAt, order.FolderPath).
		Suffix("returning id")
	query, args, err := stmt.ToSql()
	if err != nil {
		tx.Rollback(ctx)
		return 0, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "InsertOrder",
			Err:   err,
		}
	}

	var orderID int
	if err := tx.QueryRow(ctx, query, args...).Scan(&orderID); err != nil {
		tx.Rollback(ctx)
		return 0, &pkg.ErrDBProcedure{
			Cause: "failed to insert order",
			Info:  fmt.Sprintf("InsertOrder; query: %s", query),
			Err:   err,
		}
	}

	return orderID, nil
}

func (d *DefaultRepo) AddFilesToOrder(ctx context.Context, orderID int, files []DBFile) error {
	builder := d.builder.Insert("order_files").
		Columns("name", "checksum", "tg_file_id", "order_id")
	for _, file := range files {
		builder = builder.Values(file.Name, file.Checksum, file.TgFileID, orderID)
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

func (d *DefaultRepo) GetOrdersIDs(ctx context.Context, getActive bool) ([]int, error) {
	stmt := d.builder.Select("id").From("orders").OrderBy("created_at")
	if getActive {
		stmt = stmt.Where(squirrel.Or{
			squirrel.Eq{"status": model.StatusActive},
			squirrel.And{
				squirrel.NotEq{"closed_at": nil},
				squirrel.Expr("closed_at >= NOW() - INTERVAL '1 day'"),
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
	stmt := d.builder.Select("*").From("orders").Where(squirrel.Eq{"id": orderID})
	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to build query",
			Info:  "GetOrderByID",
			Err:   err,
		}
	}

	var order DBOrder
	if err := d.pool.QueryRow(ctx, query, args...).Scan(&order.ID, &order.Status, &order.ClientName, &order.Cost, &order.Comments, &order.Contacts, &order.Links, &order.CreatedAt, &order.ClosedAt, &order.FolderPath); err != nil {
		return nil, &pkg.ErrDBProcedure{
			Cause: "failed to select order",
			Info:  fmt.Sprintf("GetOrderByID; query: %s", query),
			Err:   err,
		}
	}

	return &order, nil
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

func (d *DefaultRepo) GetOrderFiles(ctx context.Context, orderID int) ([]DBFile, error) {
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

	var orderFiles []DBFile
	for rows.Next() {
		var file DBFile
		if err := rows.Scan(&file.Name, &file.Checksum, &file.TgFileID, &file.OrderID); err != nil {
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

func (d *DefaultRepo) GetOrderFilenames(ctx context.Context, orderID int) ([]string, error) {
	stmt := d.builder.Select("name").From("order_files").Where(squirrel.Eq{"order_id": orderID})
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

	var orderFilenames []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, &pkg.ErrDBProcedure{
				Cause: "failed to scan row",
				Info:  fmt.Sprintf("GetOrderFiles; query: %s", query),
				Err:   err,
			}
		}
		orderFilenames = append(orderFilenames, filename)
	}

	return orderFilenames, nil
}

func (d *DefaultRepo) DeleteOrderFiles(ctx context.Context, orderID int, filenames []string) error {
	stmt := d.builder.Delete("order_files").
		Where(squirrel.And{
			squirrel.Eq{"order_id": orderID},
			squirrel.Eq{"name": filenames},
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

func (d *DefaultRepo) UpdateOrderFiles(ctx context.Context, orderID int, files []DBFile) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return &pkg.ErrDBProcedure{
			Cause: "failed to start transaction",
			Info:  "UpdateOrderFiles",
			Err:   err,
		}
	}

	for _, file := range files {
		stmt := d.builder.Update("order_files").
			Where(squirrel.And{
				squirrel.Eq{"order_id": orderID},
				squirrel.Eq{"name": file.Name},
			}).
			Set("name", file.Name).
			Set("checksum", file.Checksum).
			Set("tg_file_id", file.TgFileID)
		query, args, err := stmt.ToSql()
		if err != nil {
			tx.Rollback(ctx)
			return &pkg.ErrDBProcedure{
				Cause: "failed to build query",
				Info:  "UpdateOrderFiles",
				Err:   err,
			}
		}

		if _, err := tx.Exec(ctx, query, args...); err != nil {
			tx.Rollback(ctx)
			return &pkg.ErrDBProcedure{
				Cause: "failed to execute query",
				Info:  "UpdateOrderFiles",
				Err:   err,
			}
		}
	}

	tx.Commit(ctx)
	return nil
}
