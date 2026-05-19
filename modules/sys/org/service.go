package org

import (
	"context"
	"sort"
	"time"

	"hei-gin/core/db"
	"hei-gin/core/exception"
	"hei-gin/core/result"
	"hei-gin/core/utils"
	ent "hei-gin/ent/gen"
	"hei-gin/ent/gen/sysgroup"
	"hei-gin/ent/gen/sysorg"
	"hei-gin/ent/gen/sysposition"
	"hei-gin/ent/gen/sysuser"

	"github.com/gin-gonic/gin"
)

// formatTime formats a *time.Time to a string in the "2006-01-02 15:04:05" layout.
// Returns an empty string if t is nil.
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// entToVO converts an ent SysOrg entity to an OrgVO.
func entToVO(entity *ent.SysOrg) *OrgVO {
	if entity == nil {
		return nil
	}
	return &OrgVO{
		ID:          entity.ID,
		Code:        entity.Code,
		Name:        entity.Name,
		Category:    entity.Category,
		ParentID:    entity.ParentID,
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

// orgToVOMap converts an ent SysOrg entity to a map for tree building.
func orgToVOMap(entity *ent.SysOrg) map[string]interface{} {
	node := map[string]interface{}{
		"id":        entity.ID,
		"code":      entity.Code,
		"name":      entity.Name,
		"category":  entity.Category,
		"status":    entity.Status,
		"sort_code": entity.SortCode,
		"children":  make([]map[string]interface{}, 0),
	}
	if entity.ParentID != nil {
		node["parent_id"] = *entity.ParentID
	}
	if entity.Description != nil {
		node["description"] = *entity.Description
	}
	if entity.Extra != nil {
		node["extra"] = *entity.Extra
	}
	if entity.CreatedAt != nil {
		node["created_at"] = entity.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if entity.CreatedBy != nil {
		node["created_by"] = *entity.CreatedBy
	}
	if entity.UpdatedAt != nil {
		node["updated_at"] = entity.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	if entity.UpdatedBy != nil {
		node["updated_by"] = *entity.UpdatedBy
	}
	return node
}

// sortTreeNodes sorts a slice of tree node maps by sort_code, recursing into children.
func sortTreeNodes(nodes []map[string]interface{}) {
	sort.Slice(nodes, func(i, j int) bool {
		si, _ := nodes[i]["sort_code"].(int)
		sj, _ := nodes[j]["sort_code"].(int)
		return si < sj
	})
	for _, n := range nodes {
		if children, ok := n["children"].([]map[string]interface{}); ok {
			sortTreeNodes(children)
		}
	}
}

// getParentIDKey returns the parent ID string for grouping. Empty string for nil/empty parents.
func getParentIDKey(parentID *string) string {
	if parentID == nil || *parentID == "" || *parentID == "0" {
		return ""
	}
	return *parentID
}

// Page returns a paginated list of organizations.
func Page(c *gin.Context, param *OrgPageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	query := db.Client.SysOrg.Query()

	if param.ParentID != "" {
		if param.ParentID == "0" {
			query = query.Where(sysorg.Or(sysorg.ParentIDIsNil(), sysorg.ParentID(""), sysorg.ID(param.ParentID)))
		} else {
			query = query.Where(sysorg.Or(sysorg.ParentID(param.ParentID), sysorg.ID(param.ParentID)))
		}
	}
	if param.Keyword != "" {
		query = query.Where(sysorg.NameContains(param.Keyword))
	}

	total, err := query.Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询组织分页失败: "+err.Error(), 500))
	}

	records, err := query.Clone().
		Order(sysorg.BySortCode()).
		Limit(param.Size).
		Offset((param.Current - 1) * param.Size).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询组织分页失败: "+err.Error(), 500))
	}

	vos := make([]*OrgVO, len(records))
	for i, r := range records {
		vos[i] = entToVO(r)
	}

	return result.PageDataResult(c, vos, total, param.Current, param.Size)
}

// Tree returns the organization tree structure.
func Tree(c *gin.Context, param *OrgTreeParam) []map[string]interface{} {
	ctx := context.Background()
	query := db.Client.SysOrg.Query().Order(sysorg.BySortCode())
	if param.Category != "" {
		query = query.Where(sysorg.CategoryEQ(param.Category))
	}

	all, err := query.All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询组织树失败: "+err.Error(), 500))
	}

	if len(all) == 0 {
		return make([]map[string]interface{}, 0)
	}

	// Build node map: id -> node
	nodeMap := make(map[string]map[string]interface{}, len(all))
	for _, e := range all {
		nodeMap[e.ID] = orgToVOMap(e)
	}

	// Build tree: attach children to parents
	roots := make([]map[string]interface{}, 0)
	for _, e := range all {
		node := nodeMap[e.ID]
		pid := getParentIDKey(e.ParentID)
		if pid == "" {
			roots = append(roots, node)
		} else {
			parent, ok := nodeMap[pid]
			if ok {
				children, _ := parent["children"].([]map[string]interface{})
				children = append(children, node)
				parent["children"] = children
			} else {
				// Parent not found in filtered set, treat as root
				roots = append(roots, node)
			}
		}
	}

	sortTreeNodes(roots)
	return roots
}

// Create creates a new organization.
func Create(c *gin.Context, vo *OrgVO, userID string) {
	ctx := context.Background()
	now := time.Now()

	builder := db.Client.SysOrg.Create().
		SetID(utils.GenerateID()).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetSortCode(vo.SortCode).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if vo.ParentID != nil {
		builder.SetNillableParentID(vo.ParentID)
	}
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
		panic(exception.NewBusinessError("添加组织失败: "+err.Error(), 500))
	}
}

// Modify updates an existing organization.
func Modify(c *gin.Context, vo *OrgVO, userID string) {
	ctx := context.Background()
	if vo.ID == "" {
		panic(exception.NewBusinessError("ID不能为空", 400))
	}

	entity, err := db.Client.SysOrg.Get(ctx, vo.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			panic(exception.NewBusinessError("数据不存在", 400))
		}
		panic(exception.NewBusinessError("查询组织失败: "+err.Error(), 500))
	}

	// Check circular reference if parent_id changed
	if vo.ParentID != nil {
		oldParentID := ""
		if entity.ParentID != nil {
			oldParentID = *entity.ParentID
		}
		if *vo.ParentID != oldParentID {
			checkCircularParent(vo.ID, *vo.ParentID)
		}
	}

	now := time.Now()
	builder := db.Client.SysOrg.UpdateOneID(vo.ID).
		SetCode(vo.Code).
		SetName(vo.Name).
		SetCategory(vo.Category).
		SetSortCode(vo.SortCode).
		SetUpdatedAt(now)

	if vo.ParentID != nil {
		builder.SetNillableParentID(vo.ParentID)
	} else {
		builder.ClearParentID()
	}
	if vo.Description != nil {
		builder.SetNillableDescription(vo.Description)
	} else {
		builder.ClearDescription()
	}
	if vo.Status != "" {
		builder.SetStatus(vo.Status)
	}
	if vo.Extra != nil {
		builder.SetNillableExtra(vo.Extra)
	} else {
		builder.ClearExtra()
	}
	if userID != "" {
		builder.SetUpdatedBy(userID)
	}

	_, err = builder.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑组织失败: "+err.Error(), 500))
	}
}

// checkCircularParent panics if setting newParentID as the parent of entityID
// would create a circular reference.
func checkCircularParent(entityID, newParentID string) {
	ctx := context.Background()
	if newParentID == "" || newParentID == "0" || entityID == "" {
		return
	}

	all, err := db.Client.SysOrg.Query().All(ctx)
	if err != nil {
		return
	}

	parentMap := make(map[string]string, len(all))
	for _, e := range all {
		if e.ParentID != nil {
			parentMap[e.ID] = *e.ParentID
		}
	}

	current := newParentID
	for current != "" {
		if current == entityID {
			panic(exception.NewBusinessError("父级不能选择自身或子节点", 400))
		}
		current = parentMap[current]
	}
}

// collectDescendantOrgIDs gathers all IDs that are the given IDs or any of their
// descendants recursively using DFS.
func collectDescendantOrgIDs(ids []string) []string {
	ctx := context.Background()
	all, err := db.Client.SysOrg.Query().All(ctx)
	if err != nil {
		return ids
	}

	childrenMap := make(map[string][]string)
	for _, r := range all {
		pid := getParentIDKey(r.ParentID)
		childrenMap[pid] = append(childrenMap[pid], r.ID)
	}

	visited := make(map[string]bool, len(ids))
	for _, id := range ids {
		visited[id] = true
	}

	stack := make([]string, len(ids))
	copy(stack, ids)

	for len(stack) > 0 {
		parentID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, childID := range childrenMap[parentID] {
			if !visited[childID] {
				visited[childID] = true
				stack = append(stack, childID)
			}
		}
	}

	result := make([]string, 0, len(visited))
	for id := range visited {
		result = append(result, id)
	}
	return result
}

// Remove deletes organizations by IDs, including all descendants.
func Remove(c *gin.Context, ids []string) {
	ctx := context.Background()
	if len(ids) == 0 {
		return
	}

	allIDs := collectDescendantOrgIDs(ids)

	// Check SysUser association
	userCount, err := db.Client.SysUser.Query().
		Where(sysuser.OrgIDIn(allIDs...)).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询组织关联用户失败: "+err.Error(), 500))
	}
	if userCount > 0 {
		panic(exception.NewBusinessError("组织存在关联用户，无法删除", 400))
	}

	// Check SysGroup association
	groupCount, err := db.Client.SysGroup.Query().
		Where(sysgroup.OrgIDIn(allIDs...)).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询组织关联用户组失败: "+err.Error(), 500))
	}
	if groupCount > 0 {
		panic(exception.NewBusinessError("组织下存在用户组，无法删除", 400))
	}

	// Clear SysPosition.org_id references
	_, err = db.Client.SysPosition.Update().
		Where(sysposition.OrgIDIn(allIDs...)).
		ClearOrgID().
		Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("清除职位组织关联失败: "+err.Error(), 500))
	}

	// Delete SysOrg records
	_, err = db.Client.SysOrg.Delete().
		Where(sysorg.IDIn(allIDs...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除组织失败: "+err.Error(), 500))
	}
}

// Detail returns a single organization by ID.
func Detail(c *gin.Context, id string) *OrgVO {
	ctx := context.Background()
	if id == "" {
		return nil
	}

	entity, err := db.Client.SysOrg.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询组织详情失败: "+err.Error(), 500))
	}

	return entToVO(entity)
}
