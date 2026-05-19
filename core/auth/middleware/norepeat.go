package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"hash/fnv"
	"io"
	"strconv"
	"time"

	"hei-gin/core/auth"
	"hei-gin/core/constants"
	"hei-gin/core/db"
	"hei-gin/core/utils"

	"github.com/gin-gonic/gin"
)

// NoRepeat returns a middleware that prevents duplicate submissions within the given interval (in milliseconds).
// It uses Redis to store a hash of the request params keyed by userID + IP + URL path.
func NoRepeat(interval int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID (try CONSUMER first, fallback to BUSINESS)
		clientAuth := auth.NewHeiClientAuthTool()
		userID := clientAuth.GetLoginIDDefaultNull(c)
		if userID == "" {
			userID = auth.GetLoginIDDefaultNull(c)
		}

		// Get client IP
		ip := utils.GetClientIP(c)

		// Build cache key
		cacheKey := constants.NO_REPEAT_PREFIX + ip + ":" + userID + ":" + c.Request.URL.Path

		// Hash request params
		phash := paramsHash(c)

		// Check Redis
		redisClient := db.Redis
		if redisClient != nil {
			ctx := context.Background()
			cached, err := redisClient.Get(ctx, cacheKey).Result()
			if err == nil {
				var data struct {
					Hash string `json:"hash"`
					Time int64  `json:"time"`
				}
				if err := json.Unmarshal([]byte(cached), &data); err == nil {
					if data.Hash == phash {
						elapsed := time.Now().UnixMilli() - data.Time
						if elapsed < int64(interval) {
							remaining := (int64(interval) - elapsed) / 1000
							if remaining < 1 {
								remaining = 1
							}
							c.Abort()
							c.JSON(200, gin.H{
								"code":    429,
								"message": "请求过于频繁，请" + strconv.FormatInt(remaining, 10) + "秒后再试",
								"success": false,
							})
							return
						}
					}
				}
			}

			// Store new request info in Redis with 3600s TTL
			nowMS := time.Now().UnixMilli()
			storeData, marshalErr := json.Marshal(map[string]interface{}{
				"hash": phash,
				"time": nowMS,
			})
			if marshalErr == nil {
				cacheTTL := interval / 1000 // convert ms to seconds
				if cacheTTL < 60 {
					cacheTTL = 60
				} else if cacheTTL > 3600 {
					cacheTTL = 3600
				}
				redisClient.SetEx(ctx, cacheKey, string(storeData), time.Duration(cacheTTL)*time.Second)
			}
		}

		c.Next()
	}
}

// paramsHash generates a deterministic hash from the request's query, form, and body parameters.
func paramsHash(c *gin.Context) string {
	params := make(map[string]interface{})

	// Collect query parameters
	for k, v := range c.Request.URL.Query() {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}

	// Collect form parameters (for POST/PUT/PATCH)
	if c.Request.Method != "GET" {
		_ = c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) == 1 {
				params[k] = v[0]
			} else {
				params[k] = v
			}
		}
	}

	// Read request body and restore it for downstream handlers (Gin v1.12.0 GetRawData does not restore Body)
	if body, err := c.GetRawData(); err == nil {
		if len(body) > 0 {
			params["_body"] = string(body)
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	// Marshal to JSON and hash
	jsonBytes, _ := json.Marshal(params)
	h := fnv.New64a()
	_, _ = h.Write(jsonBytes)
	return strconv.FormatUint(h.Sum64(), 16)
}
