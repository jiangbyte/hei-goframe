package auth

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"

	"hei-gin/core/constants"
	"hei-gin/core/db"
)

// permissionCacheTTL is the TTL for the Redis permission cache.
// 0 means no expiration — cache is rebuilt only at server startup.
const permissionCacheTTL = 0

// PermissionEntry represents a scanned permission entry.
type PermissionEntry struct {
	Code   string `json:"code"`
	Module string `json:"module"`
	Name   string `json:"name"`
}

// permissionRegistry holds all registered permissions throughout the application.
var permissionRegistry []PermissionEntry

// RegisterPermission registers a permission entry for later scanning.
func RegisterPermission(entry PermissionEntry) {
	permissionRegistry = append(permissionRegistry, entry)
}

// RunPermissionScan groups registered permissions by module and stores them in Redis.
func RunPermissionScan() error {
	if db.Redis == nil {
		log.Println("[PermissionScan] Redis not available, skipping permission caching")
		return nil
	}

	tree := buildModuleTree(permissionRegistry)
	if len(tree) == 0 {
		log.Println("[PermissionScan] No permissions registered, nothing to cache")
		return nil
	}

	data, err := json.Marshal(tree)
	if err != nil {
		log.Printf("[PermissionScan] Failed to marshal permission tree: %v", err)
		return err
	}

	ctx := context.Background()
	if err := db.Redis.Set(ctx, constants.PERMISSION_CACHE_KEY, string(data), permissionCacheTTL).Err(); err != nil {
		log.Printf("[PermissionScan] Failed to store permission cache in Redis: %v", err)
		return err
	}

	total := 0
	for _, v := range tree {
		total += len(v)
	}
	log.Printf("[PermissionScan] Cached %d permissions in Redis across %d modules", total, len(tree))
	return nil
}

// GetModulesFromRedis reads distinct permission module prefixes from Redis.
func GetModulesFromRedis() ([]string, error) {
	if db.Redis == nil {
		return []string{}, nil
	}

	ctx := context.Background()
	data, err := db.Redis.Get(ctx, constants.PERMISSION_CACHE_KEY).Result()
	if err != nil {
		return []string{}, nil
	}
	if data == "" {
		return []string{}, nil
	}

	var tree map[string]map[string]PermissionEntry
	if err := json.Unmarshal([]byte(data), &tree); err != nil {
		return []string{}, nil
	}

	modules := make([]string, 0, len(tree))
	for module := range tree {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules, nil
}

// GetPermissionsByModuleFromRedis gets all permissions under a specific module from Redis.
func GetPermissionsByModuleFromRedis(module string) ([]PermissionEntry, error) {
	if db.Redis == nil {
		return []PermissionEntry{}, nil
	}

	ctx := context.Background()
	data, err := db.Redis.Get(ctx, constants.PERMISSION_CACHE_KEY).Result()
	if err != nil {
		return []PermissionEntry{}, nil
	}
	if data == "" {
		return []PermissionEntry{}, nil
	}

	var tree map[string]map[string]PermissionEntry
	if err := json.Unmarshal([]byte(data), &tree); err != nil {
		return []PermissionEntry{}, nil
	}

	modulePerms, ok := tree[module]
	if !ok {
		return []PermissionEntry{}, nil
	}

	result := make([]PermissionEntry, 0, len(modulePerms))
	for _, entry := range modulePerms {
		result = append(result, entry)
	}
	return result, nil
}

// getModuleFromCode extracts the module from a permission code.
// Example: "user:add" → "user", "sys:user:view" → "sys:user"
func getModuleFromCode(code string) string {
	parts := strings.Split(code, ":")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], ":")
	}
	return code
}

// buildModuleTree groups permissions by module.
func buildModuleTree(permissions []PermissionEntry) map[string]map[string]PermissionEntry {
	tree := make(map[string]map[string]PermissionEntry)
	for _, entry := range permissions {
		module := entry.Module
		if module == "" {
			module = getModuleFromCode(entry.Code)
		}
		if _, ok := tree[module]; !ok {
			tree[module] = make(map[string]PermissionEntry)
		}
		tree[module][entry.Code] = entry
	}
	return tree
}
