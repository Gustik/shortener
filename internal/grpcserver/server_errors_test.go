package grpcserver_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/Gustik/shortener/internal/grpcserver"
	"github.com/Gustik/shortener/internal/model"
	pb "github.com/Gustik/shortener/pkg/shortener/v1"
	"github.com/Gustik/shortener/internal/zaplog"
)

// errSvc — стаб, всегда возвращающий неожиданную ошибку.
type errSvc struct{}

var errInternal = errors.New("internal error")

func (errSvc) ShortenURL(_ context.Context, _, _ string) (string, error) {
	return "", errInternal
}
func (errSvc) ShortenURLBatch(_ context.Context, _ []model.BatchRequest, _ string) ([]model.BatchResponse, error) {
	return nil, errInternal
}
func (errSvc) GetOriginalURL(_ context.Context, _ string) (string, error) {
	return "", errInternal
}
func (errSvc) GetUserURLs(_ context.Context, _ string) ([]model.UserURLResponse, error) {
	return nil, errInternal
}
func (errSvc) DeleteURLs(_ context.Context, _ string, _ []string) {}
func (errSvc) Stats(_ context.Context) (int, int, error)           { return 0, 0, errInternal }
func (errSvc) Ping(_ context.Context) error                        { return errInternal }

func newErrServer(t *testing.T) (pb.ShortenerServiceClient, func()) {
	t.Helper()
	logger := zaplog.NewNoop()
	interceptor := grpcserver.NewAuthInterceptor(testJWTSecret, logger)
	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	pb.RegisterShortenerServiceServer(srv, grpcserver.NewShortenerServer(errSvc{}, logger))

	lis := bufconn.Listen(1024 * 1024)
	go srv.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	return pb.NewShortenerServiceClient(conn), func() { conn.Close(); srv.Stop() }
}

func TestShortenURL_InternalError(t *testing.T) {
	client, cleanup := newErrServer(t)
	defer cleanup()

	_, err := client.ShortenURL(ctxWithToken("user1"), &pb.URLShortenRequest{Url: "https://ya.ru"})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestExpandURL_InternalError(t *testing.T) {
	client, cleanup := newErrServer(t)
	defer cleanup()

	_, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc12345"})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestListUserURLs_InternalError(t *testing.T) {
	client, cleanup := newErrServer(t)
	defer cleanup()

	_, err := client.ListUserURLs(ctxWithToken("user1"), &emptypb.Empty{})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
