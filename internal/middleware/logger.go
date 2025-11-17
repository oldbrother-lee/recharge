package middleware

import (
    "bytes"
    "fmt"
    "io"
    "math/rand"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery

        reqID := time.Now().UnixNano() + int64(rand.Intn(1000))
        c.Set("request_id", fmt.Sprintf("%d", reqID))

        var body string
        if c.Request.Body != nil {
            data, _ := io.ReadAll(c.Request.Body)
            body = string(data)
            c.Request.Body = io.NopCloser(bytes.NewBuffer(data))
        }

        c.Next()

        cost := time.Since(start)
        zap.L().Info(path,
            zap.Int("status", c.Writer.Status()),
            zap.String("method", c.Request.Method),
            zap.String("path", path),
            zap.String("query", query),
            zap.String("ip", c.ClientIP()),
            zap.String("user-agent", c.Request.UserAgent()),
            zap.String("content-type", c.ContentType()),
            zap.String("request_id", fmt.Sprintf("%d", reqID)),
            zap.String("raw_body", body),
            zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
            zap.Duration("cost", cost),
        )
    }
}
