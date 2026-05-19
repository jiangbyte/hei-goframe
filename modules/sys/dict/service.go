package dict

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"hei-gin/core/constants"
	"hei-gin/core/db"
	"hei-gin/core/exception"
	"hei-gin/core/result"
	"hei-gin/core/utils"
	ent "hei-gin/ent/gen"
	"hei-gin/ent/gen/sysdict"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Dict CRUD
// ---------------------------------------------------------------------------

// DictPage returns a paginated list of dictionary records.
func DictPage(c *gin.Context, param *DictPageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	query := db.Client.SysDict.Query()

	if param.ParentID != "" {
		query = query.Where(sysdict.Or(sysdict.ParentID(param.ParentID), sysdict.ID(param.ParentID)))
	}
	if param.Category != "" {
		query = query.Where(sysdict.CategoryEQ(param.Category))
	}
	if param.Keyword != "" {
		query = query.Where(sysdict.LabelContains(param.Keyword))
	}

	total, err := query.Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典分页失败: "+err.Error(), 500))
	}

	records, err := query.Clone().
		Order(sysdict.BySortCode()).
		Limit(param.Size).
		Offset((param.Current - 1) * param.Size).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典分页失败: "+err.Error(), 500))
	}

	vos := make([]*DictVO, len(records))
	for i, r := range records {
		vos[i] = entToVO(r)
	}

	return result.PageDataResult(c, vos, total, param.Current, param.Size)
}

// DictList returns all dictionary records matching the given filters.
func DictList(c *gin.Context, param *DictListParam) []*DictVO {
	ctx := context.Background()
	query := db.Client.SysDict.Query().Order(sysdict.BySortCode())

	if param.ParentID != "" {
		query = query.Where(sysdict.ParentID(param.ParentID))
	}
	if param.Category != "" {
		query = query.Where(sysdict.CategoryEQ(param.Category))
	}

	records, err := query.All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典列表失败: "+err.Error(), 500))
	}

	vos := make([]*DictVO, len(records))
	for i, r := range records {
		vos[i] = entToVO(r)
	}
	return vos
}

// DictTree returns the dictionary tree structure.
func DictTree(c *gin.Context, param *DictTreeParam) []map[string]interface{} {
	ctx := context.Background()
	query := db.Client.SysDict.Query().Order(sysdict.BySortCode())
	if param.Category != "" {
		query = query.Where(sysdict.CategoryEQ(param.Category))
	}

	all, err := query.All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典树失败: "+err.Error(), 500))
	}

	childrenMap := make(map[string][]map[string]interface{})
	for _, e := range all {
		node := entityToNode(e)
		pid := getParentIDKey(e.ParentID)
		childrenMap[pid] = append(childrenMap[pid], node)
	}

	// Root nodes have parent_id = "0" or nil
	roots := make([]map[string]interface{}, 0)
	for _, pid := range []string{"0", ""} {
		roots = append(roots, childrenMap[pid]...)
	}

	var buildTree func(pid string) []map[string]interface{}
	buildTree = func(pid string) []map[string]interface{} {
		nodes := make([]map[string]interface{}, 0)
		for _, n := range childrenMap[pid] {
			id, _ := n["id"].(string)
			n["children"] = buildTree(id)
			nodes = append(nodes, n)
		}
		sortTreeNodes(nodes)
		return nodes
	}

	result := buildTree("0")
	result = append(result, buildTree("")...)
	sortTreeNodes(result)
	return result
}

// DictCreate creates a new dictionary record.
func DictCreate(c *gin.Context, vo *DictVO, userID string) {
	ctx := context.Background()
	now := time.Now()

	parentID := "0"
	if vo.ParentID != nil && *vo.ParentID != "" {
		parentID = *vo.ParentID
	}

	dictCheckDuplicate(parentID, vo.Label, vo.Value, "")

	builder := db.Client.SysDict.Create().
		SetID(utils.GenerateID()).
		SetCode(vo.Code).
		SetSortCode(vo.SortCode).
		SetParentID(parentID).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if vo.Label != nil {
		builder.SetNillableLabel(vo.Label)
	}
	if vo.Value != nil {
		builder.SetNillableValue(vo.Value)
	}
	if vo.Color != nil {
		builder.SetNillableColor(vo.Color)
	}
	if vo.Category != nil {
		builder.SetNillableCategory(vo.Category)
	}
	if vo.Status != "" {
		builder.SetStatus(vo.Status)
	}
	if userID != "" {
		builder.SetCreatedBy(userID).SetUpdatedBy(userID)
	}

	_, err := builder.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("添加字典失败: "+err.Error(), 500))
	}

	syncDictCache()
}

// DictModify updates an existing dictionary record.
func DictModify(c *gin.Context, vo *DictVO, userID string) {
	ctx := context.Background()
	if vo.ID == "" {
		panic(exception.NewBusinessError("ID不能为空", 400))
	}

	entity, err := db.Client.SysDict.Get(ctx, vo.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			panic(exception.NewBusinessError("数据不存在", 404))
		}
		panic(exception.NewBusinessError("查询字典失败: "+err.Error(), 500))
	}

	parentID := "0"
	if vo.ParentID != nil && *vo.ParentID != "" {
		parentID = *vo.ParentID
	}

	dictCheckDuplicate(parentID, vo.Label, vo.Value, vo.ID)

	// Check circular reference if parent_id changed
	oldParentID := ""
	if entity.ParentID != nil {
		oldParentID = *entity.ParentID
	}
	if oldParentID != parentID {
		dictCheckCircularParent(vo.ID, parentID)
	}

	now := time.Now()
	builder := db.Client.SysDict.UpdateOneID(vo.ID).
		SetCode(vo.Code).
		SetSortCode(vo.SortCode).
		SetParentID(parentID).
		SetUpdatedAt(now)

	if vo.Label != nil {
		builder.SetLabel(*vo.Label)
	} else {
		builder.ClearLabel()
	}
	if vo.Value != nil {
		builder.SetValue(*vo.Value)
	} else {
		builder.ClearValue()
	}
	if vo.Color != nil {
		builder.SetColor(*vo.Color)
	} else {
		builder.ClearColor()
	}
	if vo.Category != nil {
		builder.SetCategory(*vo.Category)
	} else {
		builder.ClearCategory()
	}
	if vo.Status != "" {
		builder.SetStatus(vo.Status)
	} else {
		builder.ClearStatus()
	}
	if userID != "" {
		builder.SetUpdatedBy(userID)
	}

	_, err = builder.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑字典失败: "+err.Error(), 500))
	}

	syncDictCache()
}

// DictRemove deletes dictionary records by IDs, including all descendants.
func DictRemove(c *gin.Context, ids []string) {
	ctx := context.Background()
	if len(ids) == 0 {
		return
	}

	allIDs := dictCollectDescendantIDs(ids)

	_, err := db.Client.SysDict.Delete().Where(sysdict.IDIn(allIDs...)).Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除字典失败: "+err.Error(), 500))
	}

	syncDictCache()
}

// DictDetail returns a single dictionary record by ID.
func DictDetail(c *gin.Context, id string) *DictVO {
	ctx := context.Background()
	if id == "" {
		return nil
	}

	entity, err := db.Client.SysDict.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询字典详情失败: "+err.Error(), 500))
	}

	return entToVO(entity)
}

// DictGetLabel looks up a dictionary label by typeCode and value.
func DictGetLabel(c *gin.Context, typeCode, value string) gin.H {
	ctx := context.Background()
	root, err := db.Client.SysDict.Query().
		Where(sysdict.CodeEQ(typeCode)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return result.Success(c, gin.H{
				"type_code": typeCode,
				"value":     value,
				"label":     nil,
			})
		}
		panic(exception.NewBusinessError("查询字典失败: "+err.Error(), 500))
	}

	children, err := db.Client.SysDict.Query().
		Where(sysdict.ParentID(root.ID)).
		Where(sysdict.ValueEQ(value)).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典子项失败: "+err.Error(), 500))
	}

	var label *string
	if len(children) > 0 && children[0].Label != nil {
		label = children[0].Label
	}

	return result.Success(c, gin.H{
		"type_code": typeCode,
		"value":     value,
		"label":     label,
	})
}

// DictGetChildren returns the direct children of a dictionary root identified by typeCode.
func DictGetChildren(c *gin.Context, typeCode string) []*DictVO {
	ctx := context.Background()
	root, err := db.Client.SysDict.Query().
		Where(sysdict.CodeEQ(typeCode)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return []*DictVO{}
		}
		panic(exception.NewBusinessError("查询字典失败: "+err.Error(), 500))
	}

	children, err := db.Client.SysDict.Query().
		Where(sysdict.ParentID(root.ID)).
		Order(sysdict.BySortCode()).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询字典子项失败: "+err.Error(), 500))
	}

	vos := make([]*DictVO, len(children))
	for i, c := range children {
		vos[i] = entToVO(c)
	}
	return vos
}

// ---------------------------------------------------------------------------
// Cache sync
// ---------------------------------------------------------------------------

// syncDictCache rebuilds the dictionary cache in Redis.
func syncDictCache() {
	ctx := context.Background()
	all, err := db.Client.SysDict.Query().Order(sysdict.BySortCode()).All(ctx)
	if err != nil {
		return
	}

	// Build children-by-parent map
	childrenByParent := make(map[string][]*ent.SysDict)
	for _, e := range all {
		pid := getParentIDKey(e.ParentID)
		childrenByParent[pid] = append(childrenByParent[pid], e)
	}

	// Flat cache: {code: [{label, value, color}, ...]}
	flatCache := make(map[string][]map[string]interface{})
	for _, pid := range []string{"0", ""} {
		for _, e := range childrenByParent[pid] {
			code := e.Code
			items := make([]map[string]interface{}, 0)
			for _, child := range childrenByParent[e.ID] {
				item := map[string]interface{}{}
				if child.Label != nil {
					item["label"] = *child.Label
				}
				if child.Value != nil {
					item["value"] = *child.Value
				}
				if child.Color != nil {
					item["color"] = *child.Color
				}
				items = append(items, item)
			}
			flatCache[code] = items
		}
	}

	flatJSON, _ := json.Marshal(flatCache)
	db.Redis.Set(ctx, constants.DICT_CACHE_KEY, string(flatJSON), 0)

	// Full tree cache
	var buildTree func(pid string) []map[string]interface{}
	buildTree = func(pid string) []map[string]interface{} {
		nodes := make([]map[string]interface{}, 0)
		for _, e := range childrenByParent[pid] {
			node := entityToNode(e)
			node["children"] = buildTree(e.ID)
			nodes = append(nodes, node)
		}
		sortTreeNodes(nodes)
		return nodes
	}

	roots := make([]map[string]interface{}, 0)
	for _, pid := range []string{"0", ""} {
		roots = append(roots, buildTree(pid)...)
	}
	sortTreeNodes(roots)

	treeJSON, _ := json.Marshal(roots)
	db.Redis.Set(ctx, constants.DICT_TREE_CACHE_KEY, string(treeJSON), 0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// dictCheckDuplicate checks for duplicate label or value under the same parent.
func dictCheckDuplicate(parentID string, label, value *string, excludeID string) {
	ctx := context.Background()
	if label != nil && *label != "" {
		cnt, err := db.Client.SysDict.Query().
			Where(sysdict.ParentID(parentID)).
			Where(sysdict.LabelEQ(*label)).
			Count(ctx)
		if err == nil {
			if excludeID != "" {
				// Exclude the current record (for modify)
				excludeCnt, _ := db.Client.SysDict.Query().
					Where(sysdict.ID(excludeID)).
					Where(sysdict.ParentID(parentID)).
					Where(sysdict.LabelEQ(*label)).
					Count(ctx)
				cnt -= excludeCnt
			}
			if cnt > 0 {
				panic(exception.NewBusinessError("同一父字典下已存在相同标签: "+*label, 400))
			}
		}
	}

	if value != nil && *value != "" {
		cnt, err := db.Client.SysDict.Query().
			Where(sysdict.ParentID(parentID)).
			Where(sysdict.ValueEQ(*value)).
			Count(ctx)
		if err == nil {
			if excludeID != "" {
				excludeCnt, _ := db.Client.SysDict.Query().
					Where(sysdict.ID(excludeID)).
					Where(sysdict.ParentID(parentID)).
					Where(sysdict.ValueEQ(*value)).
					Count(ctx)
				cnt -= excludeCnt
			}
			if cnt > 0 {
				panic(exception.NewBusinessError("同一父字典下已存在相同值: "+*value, 400))
			}
		}
	}
}

// dictCheckCircularParent panics if setting newParentID as the parent of entityID
// would create a circular reference.
func dictCheckCircularParent(entityID, newParentID string) {
	ctx := context.Background()
	if newParentID == "" || newParentID == "0" || entityID == "" {
		return
	}

	all, err := db.Client.SysDict.Query().All(ctx)
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

// dictCollectDescendantIDs gathers all IDs that are the given IDs or any of their
// descendants recursively.
func dictCollectDescendantIDs(ids []string) []string {
	ctx := context.Background()
	all, err := db.Client.SysDict.Query().All(ctx)
	if err != nil {
		return ids
	}

	childrenMap := make(map[string][]string)
	for _, r := range all {
		pid := getParentIDKey(r.ParentID)
		childrenMap[pid] = append(childrenMap[pid], r.ID)
	}

	allIDs := make(map[string]bool, len(ids))
	for _, id := range ids {
		allIDs[id] = true
	}

	stack := make([]string, len(ids))
	copy(stack, ids)

	for len(stack) > 0 {
		parentID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, childID := range childrenMap[parentID] {
			if !allIDs[childID] {
				allIDs[childID] = true
				stack = append(stack, childID)
			}
		}
	}

	result := make([]string, 0, len(allIDs))
	for id := range allIDs {
		result = append(result, id)
	}
	return result
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

// entToVO converts an ent SysDict entity to a DictVO.
func entToVO(entity *ent.SysDict) *DictVO {
	vo := &DictVO{
		ID:       entity.ID,
		Code:     entity.Code,
		Status:   entity.Status,
		SortCode: entity.SortCode,
	}

	if entity.Label != nil {
		vo.Label = entity.Label
	}
	if entity.Value != nil {
		vo.Value = entity.Value
	}
	if entity.Color != nil {
		vo.Color = entity.Color
	}
	if entity.Category != nil {
		vo.Category = entity.Category
	}
	if entity.ParentID != nil {
		vo.ParentID = entity.ParentID
	}
	if entity.CreatedAt != nil {
		vo.CreatedAt = entity.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if entity.CreatedBy != nil {
		vo.CreatedBy = entity.CreatedBy
	}
	if entity.UpdatedAt != nil {
		vo.UpdatedAt = entity.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	if entity.UpdatedBy != nil {
		vo.UpdatedBy = entity.UpdatedBy
	}

	return vo
}

// entityToNode converts an ent SysDict entity to a map for tree building.
func entityToNode(e *ent.SysDict) map[string]interface{} {
	node := map[string]interface{}{
		"id":        e.ID,
		"code":      e.Code,
		"status":    e.Status,
		"sort_code": e.SortCode,
	}
	if e.Label != nil {
		node["label"] = *e.Label
	}
	if e.Value != nil {
		node["value"] = *e.Value
	}
	if e.Color != nil {
		node["color"] = *e.Color
	}
	if e.Category != nil {
		node["category"] = *e.Category
	}
	if e.ParentID != nil {
		node["parent_id"] = *e.ParentID
	}
	return node
}

// getParentIDKey returns the key to use for grouping in the parent-child map.
// Nil and empty parent_id are treated as root indicators.
func getParentIDKey(parentID *string) string {
	if parentID == nil || *parentID == "" {
		return ""
	}
	return *parentID
}
