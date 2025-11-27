package middleware

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "math/rand"
    "time"

    "github.com/gin-gonic/gin"
    log "recharge-go/pkg/log"
    "go.uber.org/zap"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery

        reqID := time.Now().UnixNano() + int64(rand.Intn(1000))
        rid := fmt.Sprintf("%d", reqID)
        c.Set("request_id", rid)
        ctx := context.WithValue(c.Request.Context(), "request_id", rid)
        c.Request = c.Request.WithContext(ctx)

        var body string
        if c.Request.Body != nil {
            data, _ := io.ReadAll(c.Request.Body)
            body = string(data)
            c.Request.Body = io.NopCloser(bytes.NewBuffer(data))
        }

        c.Next()

        cost := time.Since(start)
        l := log.WithContext(c.Request.Context())
        fields := []zap.Field{
            log.Int("status", c.Writer.Status()),
            log.String("method", c.Request.Method),
            log.String("path", path),
            log.String("query", query),
            log.String("ip", c.ClientIP()),
            log.String("user-agent", c.Request.UserAgent()),
            log.String("content-type", c.ContentType()),
            log.String("raw_body", body),
            log.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
            log.Duration("cost", cost),
        }
        if c.Writer.Status() >= 500 {
            l.Error("http_request_failed", fields...)
        } else if c.Writer.Status() >= 400 {
            l.Warn("http_request_error", fields...)
        } else {
            l.Info("http_request", fields...)
        }
    }
}
