package mysql

import (
	"context"

	"gorm.io/gorm"
)

// TxManager 事务管理器
// 教学要点:
// 1. 封装GORM的Transaction方法
// 2. 通过context传递事务DB(避免全局变量)
// 3. 支持嵌套事务(GORM自动使用Savepoint)
type TxManager struct {
	db *gorm.DB
}

// NewTxManager 创建事务管理器
func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// Transaction 执行事务
// 教学要点:
// 1. fn函数内的所有Repository操作都会在同一事务中执行
// 2. fn返回error时自动ROLLBACK,返回nil时自动COMMIT
// 3. 通过context.WithValue传递事务DB
//
// 使用示例:
//
//	err := txManager.Transaction(ctx, func(ctx context.Context) error {
//	    // 1. 锁定库存
//	    book, err := bookRepo.LockByID(ctx, bookID)
//	    if err != nil {
//	        return err
//	    }
//	    // 2. 创建订单
//	    err = orderRepo.Create(ctx, order)
//	    if err != nil {
//	        return err // 自动回滚
//	    }
//	    // 3. 扣减库存
//	    err = bookRepo.UpdateStock(ctx, bookID, -quantity)
//	    return err // nil则提交,非nil则回滚
//	})
func (m *TxManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将事务DB注入到Context中
		// Repository的getDB方法会从context提取事务DB
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx)
	})
}
