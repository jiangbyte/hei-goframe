package resource

import (
	"context"
	"encoding/json"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"

	"hei-gin/core/db"
	"hei-gin/core/exception"
	"hei-gin/core/result"
	"hei-gin/core/utils"
	gen "hei-gin/ent/gen"
	"hei-gin/ent/gen/relrolepermission"
	"hei-gin/ent/gen/relroleresource"
	"hei-gin/ent/gen/sysmodule"
	"hei-gin/ent/gen/sysresource"
)

// ---------------------------------------------------------------------------
// Module
// ---------------------------------------------------------------------------

// ModulePage returns a paginated list of modules.
func ModulePage(c *gin.Context, param *ModulePageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	total, err := db.Client.SysModule.Query().Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询模块列表失败: "+err.Error(), 500))
	}

	records, err := db.Client.SysModule.Query().
		Order(sysmodule.ByCreatedAt(entsql.OrderDesc())).
		Limit(param.Size).
		Offset((param.Current - 1) * param.Size).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询模块列表失败: "+err.Error(), 500))
	}

	return result.PageDataResult(c, records, total, param.Current, param.Size)
}

// ModuleDetail returns a single module by ID.
func ModuleDetail(c *gin.Context, id string) *gen.SysModule {
	ctx := context.Background()
	entity, err := db.Client.SysModule.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询模块详情失败: "+err.Error(), 500))
	}
	return entity
}

// ModuleCreate creates a new module.
func ModuleCreate(c *gin.Context, vo *ModuleVO, userID string) {
	ctx := context.Background()
	id := utils.GenerateID()
	now := time.Now()

	create := db.Client.SysModule.Create().
		SetID(id).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if userID != "" {
		create.SetCreatedBy(userID).SetUpdatedBy(userID)
	}
	if vo.Icon != nil {
		create.SetIcon(*vo.Icon)
	}
	if vo.Color != nil {
		create.SetColor(*vo.Color)
	}
	if vo.Description != nil {
		create.SetDescription(*vo.Description)
	}
	if vo.SortCode != 0 {
		create.SetSortCode(vo.SortCode)
	}
	if vo.IsVisible != "" {
		create.SetIsVisible(vo.IsVisible)
	}
	if vo.Status != "" {
		create.SetStatus(vo.Status)
	}

	_, err := create.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("添加模块失败: "+err.Error(), 500))
	}
}

// ModuleModify updates an existing module.
func ModuleModify(c *gin.Context, vo *ModuleVO, userID string) {
	ctx := context.Background()
	_, err := db.Client.SysModule.Get(ctx, vo.ID)
	if err != nil {
		if gen.IsNotFound(err) {
			panic(exception.NewBusinessError("模块不存在", 400))
		}
		panic(exception.NewBusinessError("查询模块失败: "+err.Error(), 500))
	}

	now := time.Now()
	update := db.Client.SysModule.UpdateOneID(vo.ID).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetUpdatedAt(now)

	if userID != "" {
		update.SetUpdatedBy(userID)
	}
	if vo.SortCode != 0 {
		update.SetSortCode(vo.SortCode)
	} else {
		update.SetSortCode(0)
	}
	if vo.Icon != nil {
		update.SetIcon(*vo.Icon)
	} else {
		update.ClearIcon()
	}
	if vo.Color != nil {
		update.SetColor(*vo.Color)
	} else {
		update.ClearColor()
	}
	if vo.Description != nil {
		update.SetDescription(*vo.Description)
	} else {
		update.ClearDescription()
	}
	if vo.IsVisible != "" {
		update.SetIsVisible(vo.IsVisible)
	}
	if vo.Status != "" {
		update.SetStatus(vo.Status)
	}

	_, err = update.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑模块失败: "+err.Error(), 500))
	}
}

// ModuleRemove deletes modules by the given ids.
func ModuleRemove(c *gin.Context, ids []string) {
	ctx := context.Background()
	_, err := db.Client.SysModule.Delete().
		Where(sysmodule.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除模块失败: "+err.Error(), 500))
	}
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

// ResourcePage returns a paginated list of resources.
func ResourcePage(c *gin.Context, param *ResourcePageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	total, err := db.Client.SysResource.Query().Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询资源列表失败: "+err.Error(), 500))
	}

	records, err := db.Client.SysResource.Query().
		Order(sysresource.ByCreatedAt(entsql.OrderDesc())).
		Limit(param.Size).
		Offset((param.Current - 1) * param.Size).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询资源列表失败: "+err.Error(), 500))
	}

	return result.PageDataResult(c, records, total, param.Current, param.Size)
}

// ResourceDetail returns a single resource by ID.
func ResourceDetail(c *gin.Context, id string) *gen.SysResource {
	ctx := context.Background()
	entity, err := db.Client.SysResource.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询资源详情失败: "+err.Error(), 500))
	}
	return entity
}

// ResourceCreate creates a new resource.
func ResourceCreate(c *gin.Context, vo *ResourceVO, userID string) {
	ctx := context.Background()
	id := utils.GenerateID()
	now := time.Now()

	create := db.Client.SysResource.Create().
		SetID(id).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetType(vo.Type).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if userID != "" {
		create.SetCreatedBy(userID).SetUpdatedBy(userID)
	}
	if vo.SortCode != 0 {
		create.SetSortCode(vo.SortCode)
	}
	if vo.Description != nil {
		create.SetDescription(*vo.Description)
	}
	if vo.ParentID != nil {
		create.SetParentID(*vo.ParentID)
	}
	if vo.RoutePath != nil {
		create.SetRoutePath(*vo.RoutePath)
	}
	if vo.ComponentPath != nil {
		create.SetComponentPath(*vo.ComponentPath)
	}
	if vo.RedirectPath != nil {
		create.SetRedirectPath(*vo.RedirectPath)
	}
	if vo.Icon != nil {
		create.SetIcon(*vo.Icon)
	}
	if vo.Color != nil {
		create.SetColor(*vo.Color)
	}
	if vo.IsVisible != "" {
		create.SetIsVisible(vo.IsVisible)
	}
	if vo.IsCache != "" {
		create.SetIsCache(vo.IsCache)
	}
	if vo.IsAffix != "" {
		create.SetIsAffix(vo.IsAffix)
	}
	if vo.IsBreadcrumb != "" {
		create.SetIsBreadcrumb(vo.IsBreadcrumb)
	}
	if vo.ExternalURL != nil {
		create.SetExternalURL(*vo.ExternalURL)
	}
	if vo.Extra != nil {
		create.SetExtra(*vo.Extra)
	}
	if vo.Status != "" {
		create.SetStatus(vo.Status)
	}

	_, err := create.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("添加资源失败: "+err.Error(), 500))
	}
}

// ResourceModify updates an existing resource, checking for circular parent
// references and syncing RelRolePermission when extra.permission_code changes.
func ResourceModify(c *gin.Context, vo *ResourceVO, userID string) {
	ctx := context.Background()
	entity, err := db.Client.SysResource.Get(ctx, vo.ID)
	if err != nil {
		if gen.IsNotFound(err) {
			panic(exception.NewBusinessError("资源不存在", 400))
		}
		panic(exception.NewBusinessError("查询资源失败: "+err.Error(), 500))
	}

	if vo.ParentID != nil && *vo.ParentID != "" && *vo.ParentID != "0" {
		if checkCircularParent(vo.ID, *vo.ParentID) {
			panic(exception.NewBusinessError("父资源不能选择自身或下级资源", 400))
		}
	}

	oldExtra := entity.Extra

	now := time.Now()
	update := db.Client.SysResource.UpdateOneID(vo.ID).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetType(vo.Type).
		SetUpdatedAt(now)

	if userID != "" {
		update.SetUpdatedBy(userID)
	}
	if vo.SortCode != 0 {
		update.SetSortCode(vo.SortCode)
	} else {
		update.SetSortCode(0)
	}
	if vo.Description != nil {
		update.SetDescription(*vo.Description)
	} else {
		update.ClearDescription()
	}
	if vo.ParentID != nil {
		update.SetParentID(*vo.ParentID)
	} else {
		update.ClearParentID()
	}
	if vo.RoutePath != nil {
		update.SetRoutePath(*vo.RoutePath)
	} else {
		update.ClearRoutePath()
	}
	if vo.ComponentPath != nil {
		update.SetComponentPath(*vo.ComponentPath)
	} else {
		update.ClearComponentPath()
	}
	if vo.RedirectPath != nil {
		update.SetRedirectPath(*vo.RedirectPath)
	} else {
		update.ClearRedirectPath()
	}
	if vo.Icon != nil {
		update.SetIcon(*vo.Icon)
	} else {
		update.ClearIcon()
	}
	if vo.Color != nil {
		update.SetColor(*vo.Color)
	} else {
		update.ClearColor()
	}
	if vo.IsVisible != "" {
		update.SetIsVisible(vo.IsVisible)
	}
	if vo.IsCache != "" {
		update.SetIsCache(vo.IsCache)
	}
	if vo.IsAffix != "" {
		update.SetIsAffix(vo.IsAffix)
	}
	if vo.IsBreadcrumb != "" {
		update.SetIsBreadcrumb(vo.IsBreadcrumb)
	}
	if vo.ExternalURL != nil {
		update.SetExternalURL(*vo.ExternalURL)
	} else {
		update.ClearExternalURL()
	}
	if vo.Extra != nil {
		update.SetExtra(*vo.Extra)
	} else {
		update.ClearExtra()
	}
	if vo.Status != "" {
		update.SetStatus(vo.Status)
	}

	_, err = update.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑资源失败: "+err.Error(), 500))
	}

	syncPermission(vo.ID, oldExtra, vo.Extra)
}

// ResourceRemove deletes resources and all their descendants recursively,
// cleaning up role-resource links.
func ResourceRemove(c *gin.Context, ids []string) {
	ctx := context.Background()
	allIDs := collectDescendantIDs(ids)

	_, err := db.Client.RelRoleResource.Delete().
		Where(relroleresource.ResourceIDIn(allIDs...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除资源关联失败: "+err.Error(), 500))
	}

	_, err = db.Client.SysResource.Delete().
		Where(sysresource.IDIn(allIDs...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除资源失败: "+err.Error(), 500))
	}
}

// ResourceTree returns the full resource tree starting from root nodes.
func ResourceTree(c *gin.Context) []*ResourceVO {
	ctx := context.Background()
	all, err := db.Client.SysResource.Query().
		Order(sysresource.BySortCode()).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询资源树失败", 500))
	}

	nodes := make([]*ResourceVO, len(all))
	for i, r := range all {
		nodes[i] = resourceFromEnt(r)
	}

	childrenMap := make(map[string][]*ResourceVO)
	for _, n := range nodes {
		pid := ""
		if n.ParentID != nil && *n.ParentID != "0" {
			pid = *n.ParentID
		}
		childrenMap[pid] = append(childrenMap[pid], n)
	}

	var build func(pid string) []*ResourceVO
	build = func(pid string) []*ResourceVO {
		result := make([]*ResourceVO, 0)
		for _, n := range childrenMap[pid] {
			n.Children = build(n.ID)
			result = append(result, n)
		}
		return result
	}

	roots := build("")
	return roots
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// checkCircularParent checks if setting newParentID as the parent of entityID
// would create a circular reference.
func checkCircularParent(entityID, newParentID string) bool {
	ctx := context.Background()
	if newParentID == "" || newParentID == "0" || entityID == "" {
		return false
	}

	current := newParentID
	for current != "" {
		if current == entityID {
			return true
		}
		entity, err := db.Client.SysResource.Get(ctx, current)
		if err != nil {
			return false
		}
		if entity.ParentID == nil || *entity.ParentID == "" || *entity.ParentID == "0" {
			break
		}
		current = *entity.ParentID
	}
	return false
}

// collectDescendantIDs gathers all resource IDs that are the given IDs or
// any of their descendants (recursive).
func collectDescendantIDs(ids []string) []string {
	ctx := context.Background()
	if len(ids) == 0 {
		return ids
	}

	allIDs := make(map[string]bool)
	for _, id := range ids {
		allIDs[id] = true
	}

	queue := make([]string, len(ids))
	copy(queue, ids)

	for len(queue) > 0 {
		parentID := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		children, err := db.Client.SysResource.Query().
			Where(sysresource.ParentID(parentID)).
			All(ctx)
		if err != nil {
			continue
		}

		for _, child := range children {
			if !allIDs[child.ID] {
				allIDs[child.ID] = true
				queue = append(queue, child.ID)
			}
		}
	}

	result := make([]string, 0, len(allIDs))
	for id := range allIDs {
		result = append(result, id)
	}
	return result
}

// syncPermission compares old and new extra JSON for changes to
// permission_code and updates RelRolePermission accordingly.
func syncPermission(resourceID string, oldExtra, newExtra *string) {
	ctx := context.Background()
	oldCode := extractPermissionCode(oldExtra)
	newCode := extractPermissionCode(newExtra)
	if oldCode == newCode {
		return
	}

	roleRelations, err := db.Client.RelRoleResource.Query().
		Where(relroleresource.ResourceID(resourceID)).
		All(ctx)
	if err != nil || len(roleRelations) == 0 {
		return
	}

	roleIDs := make([]string, 0, len(roleRelations))
	for _, rr := range roleRelations {
		roleIDs = append(roleIDs, rr.RoleID)
	}

	if oldCode != "" {
		_, _ = db.Client.RelRolePermission.Delete().
			Where(relrolepermission.RoleIDIn(roleIDs...)).
			Where(relrolepermission.PermissionCode(oldCode)).
			Exec(ctx)
	}

	if newCode != "" {
		for _, roleID := range roleIDs {
			exists, _ := db.Client.RelRolePermission.Query().
				Where(relrolepermission.RoleID(roleID)).
				Where(relrolepermission.PermissionCode(newCode)).
				Exist(ctx)
			if !exists {
				_ = db.Client.RelRolePermission.Create().
					SetID(utils.GenerateID()).
					SetRoleID(roleID).
					SetPermissionCode(newCode).
					Exec(ctx)
			}
		}
	}
}

// extractPermissionCode parses the extra JSON string and returns the
// permission_code value.
func extractPermissionCode(extra *string) string {
	if extra == nil || *extra == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(*extra), &m); err != nil {
		return ""
	}
	code, _ := m["permission_code"].(string)
	return code
}

// resourceFromEnt converts a gen.SysResource entity to a ResourceVO.
func resourceFromEnt(r *gen.SysResource) *ResourceVO {
	return &ResourceVO{
		ID:            r.ID,
		Code:          r.Code,
		Name:          r.Name,
		Category:      r.Category,
		Type:          r.Type,
		Description:   r.Description,
		ParentID:      r.ParentID,
		RoutePath:     r.RoutePath,
		ComponentPath: r.ComponentPath,
		RedirectPath:  r.RedirectPath,
		Icon:          r.Icon,
		Color:         r.Color,
		IsVisible:     r.IsVisible,
		IsCache:       r.IsCache,
		IsAffix:       r.IsAffix,
		IsBreadcrumb:  r.IsBreadcrumb,
		ExternalURL:   r.ExternalURL,
		Extra:         r.Extra,
		Status:        r.Status,
		SortCode:      r.SortCode,
		CreatedAt:     r.CreatedAt,
		CreatedBy:     r.CreatedBy,
		UpdatedAt:     r.UpdatedAt,
		UpdatedBy:     r.UpdatedBy,
	}
}
