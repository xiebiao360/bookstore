package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/xiebiao/bookstore/proto/catalogv1"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/domain/book"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/infrastructure/persistence/redis"
)

// CatalogServiceServer 图书目录服务gRPC实现
//
// 教学要点：
// 1. gRPC Handler的职责
//
//   - 协议转换（Protobuf ↔ 领域实体）
//
//   - 参数验证
//
//   - 错误处理（领域错误 → gRPC错误码）
//
//   - 不包含业务逻辑（业务逻辑在领域层）
//
//     2. Phase 1 vs Phase 2 对比
//     Phase 1: HTTP Handler → UseCase → Domain Service → Repository
//     Phase 2: gRPC Handler → Repository（简化，因为CRUD不需要UseCase）
//
// 3. 缓存策略（Cache-Aside模式）
//   - 查询：先查缓存，未命中再查数据库，结果写入缓存
//   - 更新：更新数据库后删除缓存
//   - 删除：删除数据库后删除缓存
type CatalogServiceServer struct {
	catalogv1.UnimplementedCatalogServiceServer
	repo  book.Repository
	cache *redis.CacheStore
}

// NewCatalogServiceServer 创建gRPC服务实例
func NewCatalogServiceServer(repo book.Repository, cache *redis.CacheStore) *CatalogServiceServer {
	return &CatalogServiceServer{
		repo:  repo,
		cache: cache,
	}
}

// GetBook 获取图书详情
//
// 教学要点：缓存策略（Cache-Aside）
// 1. 先查Redis缓存
// 2. 缓存命中：直接返回
// 3. 缓存未命中：查MySQL，结果写入Redis
func (s *CatalogServiceServer) GetBook(ctx context.Context, req *catalogv1.GetBookRequest) (*catalogv1.GetBookResponse, error) {
	// 步骤1：参数验证
	if req.BookId == 0 {
		return &catalogv1.GetBookResponse{
			Code:    40001,
			Message: "图书ID不能为空",
		}, nil
	}

	bookID := uint(req.BookId)

	// 步骤2：先查缓存
	cachedBook, err := s.cache.GetBookDetail(ctx, bookID)
	if err != nil {
		// 缓存查询失败不影响主流程，继续查数据库
		// 但需要记录日志（生产环境应该接入日志系统）
		// logger.Error("failed to get book from cache", zap.Error(err))
	}

	if cachedBook != nil {
		// 缓存命中，直接返回
		return &catalogv1.GetBookResponse{
			Code:    0,
			Message: "success",
			Book:    s.toProtoBook(cachedBook),
		}, nil
	}

	// 步骤3：缓存未命中，查询数据库
	b, err := s.repo.FindByID(ctx, bookID)
	if err != nil {
		if errors.Is(err, book.ErrBookNotFound) {
			return &catalogv1.GetBookResponse{
				Code:    40401,
				Message: "图书不存在",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "查询图书失败: %v", err)
	}

	// 步骤4：写入缓存（异步，失败不影响主流程）
	go func() {
		if err := s.cache.SetBookDetail(context.Background(), b); err != nil {
			// logger.Error("failed to set book cache", zap.Error(err))
		}
	}()

	// 步骤5：返回结果
	return &catalogv1.GetBookResponse{
		Code:    0,
		Message: "success",
		Book:    s.toProtoBook(b),
	}, nil
}

// ListBooks 图书列表（分页、排序）
//
// 教学要点：
// 1. 分页参数默认值处理
// 2. 排序参数验证（白名单）
// 3. 列表缓存策略
func (s *CatalogServiceServer) ListBooks(ctx context.Context, req *catalogv1.ListBooksRequest) (*catalogv1.ListBooksResponse, error) {
	// 步骤1：参数验证和默认值
	page := int(req.Page)
	if page < 1 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 // 限制最大每页数量
	}

	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	order := req.Order
	if order == "" {
		order = "desc"
	}

	// 步骤2：先查缓存
	cachedBooks, cachedTotal, err := s.cache.GetBookList(ctx, page, pageSize, sortBy, order)
	if err != nil {
		// 缓存失败不影响主流程
	}

	if cachedBooks != nil {
		// 缓存命中
		return &catalogv1.ListBooksResponse{
			Code:     0,
			Message:  "success",
			Books:    s.toProtoBooks(cachedBooks),
			Total:    uint32(cachedTotal),
			Page:     uint32(page),
			PageSize: uint32(pageSize),
		}, nil
	}

	// 步骤3：查询数据库
	books, total, err := s.repo.List(ctx, page, pageSize, sortBy, order)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询图书列表失败: %v", err)
	}

	// 步骤4：写入缓存
	go func() {
		if err := s.cache.SetBookList(context.Background(), books, total, page, pageSize, sortBy, order); err != nil {
			// logger.Error("failed to set book list cache", zap.Error(err))
		}
	}()

	// 步骤5：返回结果
	return &catalogv1.ListBooksResponse{
		Code:     0,
		Message:  "success",
		Books:    s.toProtoBooks(books),
		Total:    uint32(total),
		Page:     uint32(page),
		PageSize: uint32(pageSize),
	}, nil
}

// SearchBooks 搜索图书
func (s *CatalogServiceServer) SearchBooks(ctx context.Context, req *catalogv1.SearchBooksRequest) (*catalogv1.SearchBooksResponse, error) {
	// 参数验证
	keyword := req.Keyword
	if keyword == "" {
		return &catalogv1.SearchBooksResponse{
			Code:    40001,
			Message: "搜索关键词不能为空",
		}, nil
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 先查缓存
	cachedBooks, cachedTotal, err := s.cache.GetSearchResult(ctx, keyword, page, pageSize)
	if err == nil && cachedBooks != nil {
		return &catalogv1.SearchBooksResponse{
			Code:    0,
			Message: "success",
			Books:   s.toProtoBooks(cachedBooks),
			Total:   uint32(cachedTotal),
		}, nil
	}

	// 查询数据库
	books, total, err := s.repo.Search(ctx, keyword, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "搜索图书失败: %v", err)
	}

	// 写入缓存
	go func() {
		if err := s.cache.SetSearchResult(context.Background(), books, total, keyword, page, pageSize); err != nil {
			// logger.Error("failed to set search result cache", zap.Error(err))
		}
	}()

	return &catalogv1.SearchBooksResponse{
		Code:    0,
		Message: "success",
		Books:   s.toProtoBooks(books),
		Total:   uint32(total),
	}, nil
}

// PublishBook 发布图书
//
// 教学要点：
// 1. 写操作需要删除缓存（保持数据一致性）
// 2. 删除所有列表缓存（因为新图书会影响所有列表查询）
func (s *CatalogServiceServer) PublishBook(ctx context.Context, req *catalogv1.PublishBookRequest) (*catalogv1.PublishBookResponse, error) {
	// 步骤1：Protobuf → 领域实体
	b := &book.Book{
		ISBN:        req.Isbn,
		Title:       req.Title,
		Author:      req.Author,
		Publisher:   req.Publisher,
		Price:       req.Price,
		CoverURL:    req.CoverUrl,
		Description: req.Description,
		PublisherID: uint(req.PublisherId),
	}

	// 步骤2：领域验证
	if err := b.Validate(); err != nil {
		// 将领域错误转换为业务错误码
		return &catalogv1.PublishBookResponse{
			Code:    40001,
			Message: err.Error(),
		}, nil
	}

	// 步骤3：检查ISBN是否重复
	existingBook, err := s.repo.FindByISBN(ctx, b.ISBN)
	if err != nil && !errors.Is(err, book.ErrBookNotFound) {
		return nil, status.Errorf(codes.Internal, "查询图书失败: %v", err)
	}

	if existingBook != nil {
		return &catalogv1.PublishBookResponse{
			Code:    40901,
			Message: "ISBN已存在",
		}, nil
	}

	// 步骤4：创建图书
	if err := s.repo.Create(ctx, b); err != nil {
		if errors.Is(err, book.ErrISBNDup) {
			return &catalogv1.PublishBookResponse{
				Code:    40901,
				Message: "ISBN已存在",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "创建图书失败: %v", err)
	}

	// 步骤5：删除所有列表缓存（因为新图书会出现在列表中）
	go func() {
		if err := s.cache.DeleteBookListCache(context.Background()); err != nil {
			// logger.Error("failed to delete book list cache", zap.Error(err))
		}
	}()

	// 步骤6：返回结果
	return &catalogv1.PublishBookResponse{
		Code:    0,
		Message: "发布成功",
		BookId:  uint64(b.ID),
	}, nil
}

// BatchGetBooks 批量获取图书（供order-service调用）
//
// 教学要点：
// 1. 避免N+1查询问题
// 2. 批量接口的缓存策略（单个缓存，批量查询）
func (s *CatalogServiceServer) BatchGetBooks(ctx context.Context, req *catalogv1.BatchGetBooksRequest) (*catalogv1.BatchGetBooksResponse, error) {
	// 参数验证
	if len(req.BookIds) == 0 {
		return &catalogv1.BatchGetBooksResponse{
			Code:    0,
			Message: "success",
			Books:   []*catalogv1.Book{},
		}, nil
	}

	// 转换ID类型
	ids := make([]uint, len(req.BookIds))
	for i, id := range req.BookIds {
		ids[i] = uint(id)
	}

	// 批量查询
	bookMap, err := s.repo.BatchFindByIDs(ctx, ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "批量查询图书失败: %v", err)
	}

	// 转换为数组（保持请求顺序）
	books := make([]*book.Book, 0, len(req.BookIds))
	for _, id := range ids {
		if b, ok := bookMap[id]; ok {
			books = append(books, b)
		}
	}

	return &catalogv1.BatchGetBooksResponse{
		Code:    0,
		Message: "success",
		Books:   s.toProtoBooks(books),
	}, nil
}

// toProtoBook 领域实体 → Protobuf消息
func (s *CatalogServiceServer) toProtoBook(b *book.Book) *catalogv1.Book {
	if b == nil {
		return nil
	}

	return &catalogv1.Book{
		Id:          uint64(b.ID),
		Isbn:        b.ISBN,
		Title:       b.Title,
		Author:      b.Author,
		Publisher:   b.Publisher,
		Price:       b.Price,
		CoverUrl:    b.CoverURL,
		Description: b.Description,
		PublisherId: uint64(b.PublisherID),
		CreatedAt:   b.CreatedAt.Unix(),
		UpdatedAt:   b.UpdatedAt.Unix(),
	}
}

// toProtoBooks 批量转换
func (s *CatalogServiceServer) toProtoBooks(books []*book.Book) []*catalogv1.Book {
	result := make([]*catalogv1.Book, len(books))
	for i, b := range books {
		result[i] = s.toProtoBook(b)
	}
	return result
}
