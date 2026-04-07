package grpcserver_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/Gustik/shortener/internal/grpcserver"
	"github.com/Gustik/shortener/internal/handler/middleware"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
	pb "github.com/Gustik/shortener/pkg/shortener/v1"
)

const (
	testBaseURL   = "http://localhost:8080"
	testJWTSecret = "test-secret"
)

// newTestServer поднимает gRPC-сервер через bufconn и возвращает клиента и cleanup.
func newTestServer(t *testing.T) (pb.ShortenerServiceClient, *repository.InMemoryURLRepository, func()) {
	t.Helper()

	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, testBaseURL, zaplog.NewNoop())
	logger := zaplog.NewNoop()

	interceptor := grpcserver.NewAuthInterceptor(testJWTSecret, logger)
	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	pb.RegisterShortenerServiceServer(srv, grpcserver.NewShortenerServer(svc, logger))

	lis := bufconn.Listen(1024 * 1024)
	go srv.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return pb.NewShortenerServiceClient(conn), repo, cleanup
}

// makeToken создаёт валидный JWT для указанного userID.
func makeToken(userID string) string {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(testJWTSecret))
	return s
}

// ctxWithToken добавляет JWT в исходящий gRPC metadata.
func ctxWithToken(userID string) context.Context {
	md := metadata.Pairs("authorization", makeToken(userID))
	return metadata.NewOutgoingContext(context.Background(), md)
}

// --- ShortenURL ---

func TestShortenURL_Success(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	resp, err := client.ShortenURL(ctxWithToken("user1"), &pb.URLShortenRequest{Url: "https://ya.ru"})

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(resp.Result, testBaseURL), "результат должен начинаться с baseURL")
}

func TestShortenURL_EmptyURL(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	_, err := client.ShortenURL(ctxWithToken("user1"), &pb.URLShortenRequest{Url: ""})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestShortenURL_Conflict(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	ctx := ctxWithToken("user1")
	req := &pb.URLShortenRequest{Url: "https://ya.ru"}

	resp1, err := client.ShortenURL(ctx, req)
	require.NoError(t, err)

	// Повторный запрос с тем же URL — ErrURLExists, но сервер всё равно возвращает OK со старым shortURL.
	resp2, err := client.ShortenURL(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, resp1.Result, resp2.Result)
}

func TestShortenURL_NoToken(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	// Без токена интерцептор генерирует новый userID — запрос всё равно проходит.
	resp, err := client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "https://example.com"})

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(resp.Result, testBaseURL))
}

// --- ExpandURL ---

func TestExpandURL_Success(t *testing.T) {
	client, repo, cleanup := newTestServer(t)
	defer cleanup()

	_, err := repo.Save(context.Background(), "abc12345", "https://ya.ru", "user1")
	require.NoError(t, err)

	resp, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc12345"})

	require.NoError(t, err)
	assert.Equal(t, "https://ya.ru", resp.Result)
}

func TestExpandURL_NotFound(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	_, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "notexists"})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExpandURL_Deleted(t *testing.T) {
	client, repo, cleanup := newTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Save(ctx, "deleted1", "https://ya.ru", "user1")
	require.NoError(t, err)
	err = repo.DeleteURLs(ctx, []string{"deleted1"}, "user1")
	require.NoError(t, err)

	_, err = client.ExpandURL(ctx, &pb.URLExpandRequest{Id: "deleted1"})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// --- ListUserURLs ---

func TestListUserURLs_Success(t *testing.T) {
	client, repo, cleanup := newTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Save(ctx, "abc11111", "https://ya.ru", "user1")
	require.NoError(t, err)
	_, err = repo.Save(ctx, "abc22222", "https://google.com", "user1")
	require.NoError(t, err)
	_, err = repo.Save(ctx, "abc33333", "https://other.com", "user2")
	require.NoError(t, err)

	resp, err := client.ListUserURLs(ctxWithToken("user1"), &emptypb.Empty{})

	require.NoError(t, err)
	assert.Len(t, resp.Url, 2)

	originals := make([]string, 0, len(resp.Url))
	for _, u := range resp.Url {
		originals = append(originals, u.OriginalUrl)
	}
	assert.Contains(t, originals, "https://ya.ru")
	assert.Contains(t, originals, "https://google.com")
}

func TestListUserURLs_Empty(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	resp, err := client.ListUserURLs(ctxWithToken("user_with_no_urls"), &emptypb.Empty{})

	require.NoError(t, err)
	assert.Empty(t, resp.Url)
}

// --- Auth interceptor ---

func TestAuth_InvalidToken(t *testing.T) {
	client, _, cleanup := newTestServer(t)
	defer cleanup()

	// Невалидный токен — интерцептор генерирует новый userID, запрос проходит.
	md := metadata.Pairs("authorization", "not.a.valid.jwt")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://example.com"})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Result)
}

func TestAuth_UserIsolation(t *testing.T) {
	client, repo, cleanup := newTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Save(ctx, "u1url1", "https://user1.com", "user1")
	require.NoError(t, err)
	_, err = repo.Save(ctx, "u2url1", "https://user2.com", "user2")
	require.NoError(t, err)

	resp, err := client.ListUserURLs(ctxWithToken("user1"), &emptypb.Empty{})

	require.NoError(t, err)
	require.Len(t, resp.Url, 1)
	assert.Equal(t, "https://user1.com", resp.Url[0].OriginalUrl)
}
