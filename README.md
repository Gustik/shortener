# Результаты оптимизации

```
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof 
File: shortener
Build ID: c97ccecb1ab1a5a1ee768f7e712dad2983d01297
Type: inuse_space
Time: 2026-02-18 08:26:10 +05
Showing nodes accounting for 2.48kB, 0.023% of 10598.35kB total
      flat  flat%   sum%        cum   cum%
-1536.05kB 14.49% 14.49%     0.03kB 0.00029%  github.com/Gustik/shortener/internal/handler.(*URLHandler).ShortenURL
    1026kB  9.68%  4.81%     1026kB  9.68%  runtime.allocm
-1024.05kB  9.66% 14.47% -1024.05kB  9.66%  github.com/google/uuid.UUID.String
 1024.03kB  9.66%  4.81%  1024.03kB  9.66%  bytes.(*Buffer).String (inline)
  512.50kB  4.84% 0.023%   512.50kB  4.84%  go.uber.org/zap/internal/bufferpool.init.NewPool.func1
  512.05kB  4.83%  4.85%   512.05kB  4.83%  internal/sync.runtime_SemacquireMutex
 -512.01kB  4.83% 0.023% -1536.02kB 14.49%  github.com/go-chi/chi/v5/middleware.RequestID.func1
         0     0% 0.023% -1024.02kB  9.66%  github.com/Gustik/shortener/internal/handler.SetupRoutes.AuthMiddleware.func3.1
         0     0% 0.023%     0.03kB 0.00029%  github.com/Gustik/shortener/internal/handler.SetupRoutes.ContentTypeMiddleware.func4.1
         0     0% 0.023% -1536.02kB 14.49%  github.com/Gustik/shortener/internal/handler.SetupRoutes.GzipMiddleware.func2.1
         0     0% 0.023% -1023.52kB  9.66%  github.com/Gustik/shortener/internal/handler.SetupRoutes.RequestLogger.func1.1
         0     0% 0.023%   512.05kB  4.83%  github.com/Gustik/shortener/internal/repository.(*InMemoryURLRepository).Save
         0     0% 0.023%   512.05kB  4.83%  github.com/Gustik/shortener/internal/service.(*urlService).ShortenURL
         0     0% 0.023%     0.03kB 0.00029%  github.com/go-chi/chi/v5.(*ChainHandler).ServeHTTP
         0     0% 0.023% -1023.52kB  9.66%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 0.023%     0.03kB 0.00029%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 0.023% -1024.02kB  9.66%  github.com/go-chi/chi/v5/middleware.RealIP.func1
         0     0% 0.023% -1536.02kB 14.49%  github.com/go-chi/chi/v5/middleware.Recoverer.func1
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap.(*SugaredLogger).Infoln
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap.(*SugaredLogger).logln
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/buffer.Pool.Get
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/internal/bufferpool.init.NewPool.New[go.shape.*uint8].func2
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/internal/pool.(*Pool[go.shape.*uint8]).Get (inline)
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/zapcore.(*CheckedEntry).Write
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/zapcore.(*ioCore).Write
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/zapcore.(*jsonEncoder).EncodeEntry
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/zapcore.EntryCaller.TrimmedPath
         0     0% 0.023%   512.50kB  4.84%  go.uber.org/zap/zapcore.ShortCallerEncoder
         0     0% 0.023%   512.05kB  4.83%  internal/sync.(*Mutex).Lock (inline)
         0     0% 0.023%   512.05kB  4.83%  internal/sync.(*Mutex).lockSlow
         0     0% 0.023% -1023.52kB  9.66%  net/http.(*conn).serve
         0     0% 0.023% -1023.52kB  9.66%  net/http.HandlerFunc.ServeHTTP
         0     0% 0.023% -1023.52kB  9.66%  net/http.serverHandler.ServeHTTP
         0     0% 0.023%     1026kB  9.68%  runtime.mstart
         0     0% 0.023%     1026kB  9.68%  runtime.mstart0
         0     0% 0.023%     1026kB  9.68%  runtime.mstart1
         0     0% 0.023%     1026kB  9.68%  runtime.newm
         0     0% 0.023%     1026kB  9.68%  runtime.resetspinning
         0     0% 0.023%     1026kB  9.68%  runtime.schedule
         0     0% 0.023%     1026kB  9.68%  runtime.startm
         0     0% 0.023%     1026kB  9.68%  runtime.wakep
         0     0% 0.023%   512.05kB  4.83%  sync.(*Mutex).Lock (inline)
         0     0% 0.023%   512.50kB  4.84%  sync.(*Pool).Get
```