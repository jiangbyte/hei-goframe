package role

import (
	"context"
	"encoding/json"
	"time"

	"hei-gin/core/db"
	"hei-gin/core/exception"
	"hei-gin/core/result"
	"hei-gin/core/utils"
	ent "hei-gin/ent/gen"
	"hei-gin/ent/gen/relrolepermission"
	"hei-gin/ent/gen/relroleresource"
	"hei-gin/ent/gen/reluserrole"
	"hei-gin/ent/gen/sysresource"
	"hei-gin/ent/gen/sysrole"

	"entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
)

// entToVO converts an ent SysRole entity to a RoleVO.
func entToVO(entity *ent.SysRole) *RoleVO {
	if entity == nil {
		return nil
	}
	return &RoleVO{
		ID:          entity.ID,
		Code:        entity.Code,
		Name:        entity.Name,
		Category:    entity.Category,
		Description: entity.Description,
		Status:      entity.Status,
		SortCode:    entity.SortCode,
		Extra:       entity.Extra,
		CreatedAt:   formatTime(entity.CreatedAt),
		CreatedBy:   entity.CreatedBy,
		UpdatedAt:   formatTime(entity.UpdatedAt),
		UpdatedBy:   entity.UpdatedBy,
	}
}

// formatTime formats a *time.Time to a string in the "2006-01-02 15:04:05" layout.
// Returns an empty string if t is nil.
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// RolePage returns a paginated list of roles.
func RolePage(c *gin.Context, param *RolePageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	offset := (param.Current - 1) * param.Size

	total, err := db.Client.SysRole.Query().Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色列表失败: "+err.Error(), 500))
	}

	records, err := db.Client.SysRole.Query().
		Order(sysrole.ByCreatedAt(sql.OrderDesc())).
		Limit(param.Size).
		Offset(offset).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色列表失败: "+err.Error(), 500))
	}

	vos := make([]*RoleVO, 0, len(records))
	for _, r := range records {
		vos = append(vos, entToVO(r))
	}

	return result.PageDataResult(c, vos, total, param.Current, param.Size)
}

// RoleCreate creates a new role.
func RoleCreate(c *gin.Context, vo *RoleVO, userID string) {
	ctx := context.Background()
	now := time.Now()

	builder := db.Client.SysRole.Create().
		SetID(utils.GenerateID()).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetSortCode(vo.SortCode).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if vo.Description != nil {
		builder.SetNillableDescription(vo.Description)
	}
	if vo.Status != "" {
		builder.SetStatus(vo.Status)
	}
	if vo.Extra != nil {
		builder.SetNillableExtra(vo.Extra)
	}
	if userID != "" {
		builder.SetCreatedBy(userID).SetUpdatedBy(userID)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("添加角色失败: "+err.Error(), 500))
	}
}

// RoleModify updates an existing role.
func RoleModify(c *gin.Context, vo *RoleVO, userID string) {
	ctx := context.Background()
	if vo.ID == "" {
		panic(exception.NewBusinessError("ID不能为空", 400))
	}

	// Verify the role exists
	_, err := db.Client.SysRole.Get(ctx, vo.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			panic(exception.NewBusinessError("数据不存在", 400))
		}
		panic(exception.NewBusinessError("查询角色失败: "+err.Error(), 500))
	}

	now := time.Now()
	builder := db.Client.SysRole.UpdateOneID(vo.ID).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetSortCode(vo.SortCode).
		SetUpdatedAt(now)

	if vo.Description != nil {
		builder.SetNillableDescription(vo.Description)
	}
	if vo.Status != "" {
		builder.SetStatus(vo.Status)
	}
	if vo.Extra != nil {
		builder.SetNillableExtra(vo.Extra)
	}
	if userID != "" {
		builder.SetUpdatedBy(userID)
	}

	_, err = builder.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑角色失败: "+err.Error(), 500))
	}
}

// RoleRemove deletes roles by IDs.
func RoleRemove(c *gin.Context, ids []string) {
	ctx := context.Background()
	if len(ids) == 0 {
		return
	}

	// Check for associated users
	count, err := db.Client.RelUserRole.Query().
		Where(reluserrole.RoleIDIn(ids...)).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色关联用户失败: "+err.Error(), 500))
	}
	if count > 0 {
		panic(exception.NewBusinessError("角色存在关联用户，无法删除", 400))
	}

	// Delete RelRolePermission
	_, err = db.Client.RelRolePermission.Delete().
		Where(relrolepermission.RoleIDIn(ids...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除角色权限失败: "+err.Error(), 500))
	}

	// Delete RelRoleResource
	_, err = db.Client.RelRoleResource.Delete().
		Where(relroleresource.RoleIDIn(ids...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除角色资源失败: "+err.Error(), 500))
	}

	// Delete SysRole
	_, err = db.Client.SysRole.Delete().
		Where(sysrole.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除角色失败: "+err.Error(), 500))
	}
}

// RoleDetail returns a single role by ID.
func RoleDetail(c *gin.Context, id string) *RoleVO {
	ctx := context.Background()
	if id == "" {
		return nil
	}

	entity, err := db.Client.SysRole.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询角色详情失败: "+err.Error(), 500))
	}

	return entToVO(entity)
}

// RoleGrantPermissions grants permissions to a role by replacing all existing permissions.
func RoleGrantPermissions(c *gin.Context, roleID string, permissions []PermissionItem, userID string) {
	ctx := context.Background()
	// Delete existing permissions
	_, err := db.Client.RelRolePermission.Delete().
		Where(relrolepermission.RoleID(roleID)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除角色权限失败: "+err.Error(), 500))
	}

	// Recreate permissions
	for _, item := range permissions {
		builder := db.Client.RelRolePermission.Create().
			SetID(utils.GenerateID()).
			SetRoleID(roleID).
			SetPermissionCode(item.PermissionCode).
			SetScope(item.Scope)

		if item.CustomScopeGroupIds != nil {
			builder.SetNillableCustomScopeGroupIds(item.CustomScopeGroupIds)
		}
		if item.CustomScopeOrgIds != nil {
			builder.SetNillableCustomScopeOrgIds(item.CustomScopeOrgIds)
		}

		_, err = builder.Save(ctx)
		if err != nil {
			panic(exception.NewBusinessError("分配角色权限失败: "+err.Error(), 500))
		}
	}
}

// RoleGrantResources grants resources to a role, and auto-adds missing button permissions.
func RoleGrantResources(c *gin.Context, roleID string, resourceIDs []string, permissions []ButtonPermissionScope) {
	ctx := context.Background()
	// Deduplicate resource IDs
	dedupMap := make(map[string]struct{})
	for _, id := range resourceIDs {
		dedupMap[id] = struct{}{}
	}
	uniqueIDs := make([]string, 0, len(dedupMap))
	for id := range dedupMap {
		uniqueIDs = append(uniqueIDs, id)
	}

	// Delete existing RelRoleResource
	_, err := db.Client.RelRoleResource.Delete().
		Where(relroleresource.RoleID(roleID)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除角色资源失败: "+err.Error(), 500))
	}

	// Recreate RelRoleResource
	for _, id := range uniqueIDs {
		_, err := db.Client.RelRoleResource.Create().
			SetID(utils.GenerateID()).
			SetRoleID(roleID).
			SetResourceID(id).
			Save(ctx)
		if err != nil {
			panic(exception.NewBusinessError("分配角色资源失败: "+err.Error(), 500))
		}
	}

	// Query SysResource with extra not null
	resources, err := db.Client.SysResource.Query().
		Where(
			sysresource.IDIn(uniqueIDs...),
			sysresource.ExtraNotNil(),
		).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询资源扩展信息失败: "+err.Error(), 500))
	}

	// Get existing permission codes for this role
	existingPerms, err := db.Client.RelRolePermission.Query().
		Where(relrolepermission.RoleID(roleID)).
		Select(relrolepermission.FieldPermissionCode).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色权限失败: "+err.Error(), 500))
	}
	existingPermMap := make(map[string]bool)
	for _, p := range existingPerms {
		existingPermMap[p.PermissionCode] = true
	}

	// Add missing button permissions extracted from resource extra JSON
	for _, resource := range resources {
		if resource.Extra == nil || *resource.Extra == "" {
			continue
		}
		var extraMap map[string]interface{}
		if err := json.Unmarshal([]byte(*resource.Extra), &extraMap); err != nil {
			continue
		}
		permCode, ok := extraMap["permission_code"].(string)
		if !ok || permCode == "" {
			continue
		}
		if existingPermMap[permCode] {
			continue
		}
		_, err := db.Client.RelRolePermission.Create().
			SetID(utils.GenerateID()).
			SetRoleID(roleID).
			SetPermissionCode(permCode).
			SetScope("ALL").
			Save(ctx)
		if err != nil {
			panic(exception.NewBusinessError("分配角色权限失败: "+err.Error(), 500))
		}
	}
}

// RoleOwnPermissionCodes returns the permission codes owned by a role.
func RoleOwnPermissionCodes(c *gin.Context, roleID string) []string {
	ctx := context.Background()
	perms, err := db.Client.RelRolePermission.Query().
		Where(relrolepermission.RoleID(roleID)).
		Select(relrolepermission.FieldPermissionCode).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色权限失败: "+err.Error(), 500))
	}

	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.PermissionCode)
	}
	return codes
}

// RoleOwnPermissionDetails returns detailed permission info (code, scope, custom fields) for a role.
func RoleOwnPermissionDetails(c *gin.Context, roleID string) []map[string]interface{} {
	ctx := context.Background()
	perms, err := db.Client.RelRolePermission.Query().
		Where(relrolepermission.RoleID(roleID)).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色权限详情失败: "+err.Error(), 500))
	}

	result := make([]map[string]interface{}, 0, len(perms))
	for _, p := range perms {
		item := map[string]interface{}{
			"permission_code":        p.PermissionCode,
			"scope":                  p.Scope,
			"custom_scope_group_ids": p.CustomScopeGroupIds,
			"custom_scope_org_ids":   p.CustomScopeOrgIds,
		}
		result = append(result, item)
	}
	return result
}

// RoleOwnResourceIDs returns the resource IDs owned by a role.
func RoleOwnResourceIDs(c *gin.Context, roleID string) []string {
	ctx := context.Background()
	resources, err := db.Client.RelRoleResource.Query().
		Where(relroleresource.RoleID(roleID)).
		Select(relroleresource.FieldResourceID).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询角色资源失败: "+err.Error(), 500))
	}

	ids := make([]string, 0, len(resources))
	for _, r := range resources {
		ids = append(ids, r.ResourceID)
	}
	return ids
}
