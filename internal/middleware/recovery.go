package middleware

import (
    "github.com/gin-gonic/gin"
    log "recharge-go/pkg/log"
)

func Recovery() gin.HandlerFunc {
    return log.GinRecovery()
}
