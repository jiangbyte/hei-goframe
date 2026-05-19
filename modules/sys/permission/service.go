package permission

import (
	"context"
	"encoding/json"
	"sort"

	"hei-gin/core/constants"
	"hei-gin/core/db"

	"github.com/gin-gonic/gin"
)

// ListModules returns sorted permission module names from Redis.
func ListModules(c *gin.Context) []string {
	ctx := context.Background()
	data, err := db.Redis.Get(ctx, constants.PERMISSION_CACHE_KEY).Result()
	if err != nil {
		return []string{}
	}
	var tree map[string]interface{}
	if err := json.Unmarshal([]byte(data), &tree); err != nil {
		return []string{}
	}
	modules := make([]string, 0, len(tree))
	for k := range tree {
		modules = append(modules, k)
	}
	sort.Strings(modules)
	return modules
}

// ListByModule returns permission list for a specific module from Redis.
func ListByModule(c *gin.Context, module string) []interface{} {
	ctx := context.Background()
	data, err := db.Redis.Get(ctx, constants.PERMISSION_CACHE_KEY).Result()
	if err != nil {
		return []interface{}{}
	}
	var tree map[string]interface{}
	if err := json.Unmarshal([]byte(data), &tree); err != nil {
		return []interface{}{}
	}
	modulePerms, ok := tree[module].(map[string]interface{})
	if !ok {
		return []interface{}{}
	}
	perms := make([]interface{}, 0, len(modulePerms))
	for _, v := range modulePerms {
		perms = append(perms, v)
	}
	return perms
}
