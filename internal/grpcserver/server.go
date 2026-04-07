package grpcserver

import (
	"context"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/Gustik/shortener/internal/handler/middleware"
	pb "github.com/Gustik/shortener/pkg/shortener/v1"
	"github.com/Gustik/shortener/internal/service"
)

// ShortenerServer реализует gRPC-сервис ShortenerService.
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	service service.URLService
	logger  *zap.Logger
}

// NewShortenerServer создаёт новый gRPC-сервер.
func NewShortenerServer(svc service.URLService, logger *zap.Logger) *ShortenerServer {
	return &ShortenerServer{service: svc, logger: logger}
}

// ShortenURL реализует rpc ShortenURL — создаёт короткий URL.
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user id")
	}

	shortURL, err := s.service.ShortenURL(ctx, req.GetUrl(), userID)
	if errors.Is(err, service.ErrEmptyURL) {
		return nil, status.Error(codes.InvalidArgument, "url cannot be empty")
	}
	if err != nil && !errors.Is(err, service.ErrURLExists) {
		s.logger.Error("grpc: failed to shorten URL", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.URLShortenResponse{Result: shortURL}, nil
}

// ExpandURL реализует rpc ExpandURL — возвращает оригинальный URL по короткому id.
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	originalURL, err := s.service.GetOriginalURL(ctx, req.GetId())
	if errors.Is(err, service.ErrURLNotFound) {
		return nil, status.Error(codes.NotFound, "url not found")
	}
	if errors.Is(err, service.ErrURLDeleted) {
		return nil, status.Error(codes.NotFound, "url has been deleted")
	}
	if err != nil {
		s.logger.Error("grpc: failed to expand URL", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs реализует rpc ListUserURLs — возвращает все URL пользователя.
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user id")
	}

	urls, err := s.service.GetUserURLs(ctx, userID)
	if err != nil {
		s.logger.Error("grpc: failed to list user URLs", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	resp := &pb.UserURLsResponse{
		Url: make([]*pb.URLData, 0, len(urls)),
	}
	for _, u := range urls {
		resp.Url = append(resp.Url, &pb.URLData{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		})
	}

	return resp, nil
}

// NewAuthInterceptor возвращает unary-интерцептор, который читает JWT из
// metadata-заголовка "authorization", извлекает userID и кладёт его в контекст.
// Если токен отсутствует или невалиден, генерируется новый UUID (аналог HTTP middleware).
func NewAuthInterceptor(jwtSecret string, logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		userID := extractUserID(ctx, jwtSecret, logger)
		ctx = context.WithValue(ctx, middleware.UserIDContextKey, userID)
		return handler(ctx, req)
	}
}

func extractUserID(ctx context.Context, jwtSecret string, logger *zap.Logger) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get("authorization"); len(values) > 0 {
			tokenStr := strings.TrimPrefix(values[0], "Bearer ")
			token, err := jwt.ParseWithClaims(tokenStr, &middleware.Claims{}, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(*middleware.Claims); ok && claims.UserID != "" {
					logger.Debug("grpc: валидный JWT токен", zap.String("userID", claims.UserID))
					return claims.UserID
				}
			} else {
				logger.Debug("grpc: невалидный JWT токен", zap.Error(err))
			}
		}
	}

	userID := uuid.New().String()
	logger.Debug("grpc: генерация нового userID", zap.String("userID", userID))
	return userID
}
