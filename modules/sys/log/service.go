package log

import (
	"context"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"

	"hei-gin/core/db"
	"hei-gin/core/exception"
	"hei-gin/core/result"
	"hei-gin/core/utils"
	ent "hei-gin/ent/gen"
	syslog "hei-gin/ent/gen/syslog"
)

// Page returns a paginated list of SysLog records excluding large fields.
func Page(c *gin.Context, param *LogPageParam) gin.H {
	ctx := context.Background()
	if param.Current < 1 {
		param.Current = 1
	}
	if param.Size < 1 {
		param.Size = 10
	}

	// Build the base filtered query
	query := db.Client.SysLog.Query()
	if strings.TrimSpace(param.Keyword) != "" {
		query = query.Where(syslog.NameContains(strings.TrimSpace(param.Keyword)))
	}
	if param.Category != "" {
		query = query.Where(syslog.CategoryEQ(param.Category))
	}
	if param.ExeStatus != "" {
		query = query.Where(syslog.ExeStatusEQ(param.ExeStatus))
	}

	// Count total (clone so we can reuse query with Select)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询操作日志列表失败: "+err.Error(), 500))
	}

	// Select only non-large fields for the page list
	records, err := query.
		Select(
			syslog.FieldID,
			syslog.FieldCategory,
			syslog.FieldName,
			syslog.FieldExeStatus,
			syslog.FieldOpIP,
			syslog.FieldOpAddress,
			syslog.FieldOpBrowser,
			syslog.FieldOpOs,
			syslog.FieldClassName,
			syslog.FieldMethodName,
			syslog.FieldReqMethod,
			syslog.FieldReqURL,
			syslog.FieldOpTime,
			syslog.FieldTraceID,
			syslog.FieldOpUser,
			syslog.FieldCreatedAt,
			syslog.FieldCreatedBy,
			syslog.FieldUpdatedAt,
			syslog.FieldUpdatedBy,
		).
		Order(syslog.ByOpTime(entsql.OrderDesc())).
		Limit(param.Size).
		Offset((param.Current - 1) * param.Size).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询操作日志列表失败: "+err.Error(), 500))
	}

	recordsVO := make([]*LogVO, len(records))
	for i, r := range records {
		recordsVO[i] = entToVO(r)
	}

	return result.PageDataResult(c, recordsVO, total, param.Current, param.Size)
}

// Detail returns a single SysLog record by ID including all fields.
func Detail(c *gin.Context, id string) *LogVO {
	ctx := context.Background()
	entity, err := db.Client.SysLog.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		panic(exception.NewBusinessError("查询操作日志详情失败: "+err.Error(), 500))
	}
	return entToVO(entity)
}

// Create inserts a new SysLog record.
func Create(c *gin.Context, vo *LogVO, userID string) {
	ctx := context.Background()
	now := time.Now()

	create := db.Client.SysLog.Create().
		SetID(utils.GenerateID()).
		SetCategory(vo.Category).
		SetName(vo.Name).
		SetExeStatus(vo.ExeStatus).
		SetOpIP(vo.OpIP).
		SetOpAddress(vo.OpAddress).
		SetOpBrowser(vo.OpBrowser).
		SetOpOs(vo.OpOs).
		SetClassName(vo.ClassName).
		SetMethodName(vo.MethodName).
		SetReqMethod(vo.ReqMethod).
		SetReqURL(vo.ReqURL).
		SetTraceID(vo.TraceID).
		SetOpUser(vo.OpUser).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	// Set potentially large fields via SetNillable
	exeMsg := vo.ExeMessage
	create.SetNillableExeMessage(&exeMsg)
	paramsJSON := vo.ParamJSON
	create.SetNillableParamJSON(&paramsJSON)
	resultJSON := vo.ResultJSON
	create.SetNillableResultJSON(&resultJSON)
	signData := vo.SignData
	create.SetNillableSignData(&signData)

	// Parse OpTime if provided, otherwise use now
	if vo.OpTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", vo.OpTime); err == nil {
			create.SetOpTime(t)
		} else {
			create.SetOpTime(now)
		}
	} else {
		create.SetOpTime(now)
	}

	if userID != "" {
		create.SetCreatedBy(userID).SetUpdatedBy(userID)
	}

	_, err := create.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("添加操作日志失败: "+err.Error(), 500))
	}
}

// Modify updates an existing SysLog record.
func Modify(c *gin.Context, vo *LogVO, userID string) {
	ctx := context.Background()
	// Check existence
	_, err := db.Client.SysLog.Get(ctx, vo.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			panic(exception.NewBusinessError("数据不存在", 400))
		}
		panic(exception.NewBusinessError("查询操作日志失败: "+err.Error(), 500))
	}

	now := time.Now()
	update := db.Client.SysLog.UpdateOneID(vo.ID).
		SetCategory(vo.Category).
		SetName(vo.Name).
		SetExeStatus(vo.ExeStatus).
		SetOpIP(vo.OpIP).
		SetOpAddress(vo.OpAddress).
		SetOpBrowser(vo.OpBrowser).
		SetOpOs(vo.OpOs).
		SetClassName(vo.ClassName).
		SetMethodName(vo.MethodName).
		SetReqMethod(vo.ReqMethod).
		SetReqURL(vo.ReqURL).
		SetTraceID(vo.TraceID).
		SetOpUser(vo.OpUser).
		SetUpdatedAt(now)

	// Set potentially large fields via SetNillable
	exeMsg := vo.ExeMessage
	update.SetNillableExeMessage(&exeMsg)
	paramsJSON := vo.ParamJSON
	update.SetNillableParamJSON(&paramsJSON)
	resultJSON := vo.ResultJSON
	update.SetNillableResultJSON(&resultJSON)
	signData := vo.SignData
	update.SetNillableSignData(&signData)

	// Parse OpTime if provided
	if vo.OpTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", vo.OpTime); err == nil {
			update.SetOpTime(t)
		}
	}

	if userID != "" {
		update.SetUpdatedBy(userID)
	}

	_, err = update.Save(ctx)
	if err != nil {
		panic(exception.NewBusinessError("编辑操作日志失败: "+err.Error(), 500))
	}
}

// Remove deletes SysLog records by the given ids.
func Remove(c *gin.Context, ids []string) {
	ctx := context.Background()
	_, err := db.Client.SysLog.Delete().
		Where(syslog.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("删除操作日志失败: "+err.Error(), 500))
	}
}

// DeleteByCategory deletes all SysLog records matching the given category.
func DeleteByCategory(c *gin.Context, param *LogDeleteByCategoryParam) {
	ctx := context.Background()
	_, err := db.Client.SysLog.Delete().
		Where(syslog.CategoryEQ(param.Category)).
		Exec(ctx)
	if err != nil {
		panic(exception.NewBusinessError("按分类删除操作日志失败: "+err.Error(), 500))
	}
}

// VisLineChart returns daily LOGIN/LOGOUT counts for the last 7 days.
func VisLineChart(c *gin.Context) *BarChartData {
	ctx := context.Background()
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)

	records, err := db.Client.SysLog.Query().
		Where(syslog.CategoryIn("LOGIN", "LOGOUT")).
		Where(syslog.OpTimeGTE(since)).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询访问量折线图失败: "+err.Error(), 500))
	}

	days := make([]string, 7)
	for i := 0; i < 7; i++ {
		days[i] = since.AddDate(0, 0, i).Format("2006-01-02")
	}

	loginMap := make(map[string]int)
	logoutMap := make(map[string]int)
	for _, r := range records {
		if r.OpTime != nil && r.Category != nil {
			dayStr := r.OpTime.Format("2006-01-02")
			switch *r.Category {
			case "LOGIN":
				loginMap[dayStr]++
			case "LOGOUT":
				logoutMap[dayStr]++
			}
		}
	}

	loginData := make([]int, 7)
	logoutData := make([]int, 7)
	for i, d := range days {
		loginData[i] = loginMap[d]
		logoutData[i] = logoutMap[d]
	}

	return &BarChartData{
		Days: days,
		Series: []CategorySeries{
			{Name: "登录", Data: loginData},
			{Name: "登出", Data: logoutData},
		},
	}
}

// VisPieChart returns total LOGIN and LOGOUT counts.
func VisPieChart(c *gin.Context) *PieChartData {
	ctx := context.Background()
	loginTotal, err := db.Client.SysLog.Query().
		Where(syslog.CategoryEQ("LOGIN")).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询访问量饼图失败: "+err.Error(), 500))
	}

	logoutTotal, err := db.Client.SysLog.Query().
		Where(syslog.CategoryEQ("LOGOUT")).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询访问量饼图失败: "+err.Error(), 500))
	}

	return &PieChartData{
		Data: []CategoryTotal{
			{Category: "登录", Total: loginTotal},
			{Category: "登出", Total: logoutTotal},
		},
	}
}

// OpBarChart returns daily OPERATE/EXCEPTION counts for the last 7 days.
func OpBarChart(c *gin.Context) *BarChartData {
	ctx := context.Background()
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)

	records, err := db.Client.SysLog.Query().
		Where(syslog.CategoryIn("OPERATE", "EXCEPTION")).
		Where(syslog.OpTimeGTE(since)).
		All(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询操作量柱状图失败: "+err.Error(), 500))
	}

	days := make([]string, 7)
	for i := 0; i < 7; i++ {
		days[i] = since.AddDate(0, 0, i).Format("2006-01-02")
	}

	operateMap := make(map[string]int)
	exceptionMap := make(map[string]int)
	for _, r := range records {
		if r.OpTime != nil && r.Category != nil {
			dayStr := r.OpTime.Format("2006-01-02")
			switch *r.Category {
			case "OPERATE":
				operateMap[dayStr]++
			case "EXCEPTION":
				exceptionMap[dayStr]++
			}
		}
	}

	operateData := make([]int, 7)
	exceptionData := make([]int, 7)
	for i, d := range days {
		operateData[i] = operateMap[d]
		exceptionData[i] = exceptionMap[d]
	}

	return &BarChartData{
		Days: days,
		Series: []CategorySeries{
			{Name: "操作", Data: operateData},
			{Name: "异常", Data: exceptionData},
		},
	}
}

// OpPieChart returns total OPERATE and EXCEPTION counts.
func OpPieChart(c *gin.Context) *PieChartData {
	ctx := context.Background()
	operateTotal, err := db.Client.SysLog.Query().
		Where(syslog.CategoryEQ("OPERATE")).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询操作量饼图失败: "+err.Error(), 500))
	}

	exceptionTotal, err := db.Client.SysLog.Query().
		Where(syslog.CategoryEQ("EXCEPTION")).
		Count(ctx)
	if err != nil {
		panic(exception.NewBusinessError("查询操作量饼图失败: "+err.Error(), 500))
	}

	return &PieChartData{
		Data: []CategoryTotal{
			{Category: "操作", Total: operateTotal},
			{Category: "异常", Total: exceptionTotal},
		},
	}
}

// entToVO converts an ent SysLog entity to a LogVO.
// All *string pointers are dereferenced to empty string when nil.
// *time.Time pointers are formatted as "2006-01-02 15:04:05".
func entToVO(entity *ent.SysLog) *LogVO {
	if entity == nil {
		return nil
	}

	vo := &LogVO{
		ID: entity.ID,
	}

	if entity.Category != nil {
		vo.Category = *entity.Category
	}
	if entity.Name != nil {
		vo.Name = *entity.Name
	}
	if entity.ExeStatus != nil {
		vo.ExeStatus = *entity.ExeStatus
	}
	if entity.ExeMessage != nil {
		vo.ExeMessage = *entity.ExeMessage
	}
	if entity.OpIP != nil {
		vo.OpIP = *entity.OpIP
	}
	if entity.OpAddress != nil {
		vo.OpAddress = *entity.OpAddress
	}
	if entity.OpBrowser != nil {
		vo.OpBrowser = *entity.OpBrowser
	}
	if entity.OpOs != nil {
		vo.OpOs = *entity.OpOs
	}
	if entity.ClassName != nil {
		vo.ClassName = *entity.ClassName
	}
	if entity.MethodName != nil {
		vo.MethodName = *entity.MethodName
	}
	if entity.ReqMethod != nil {
		vo.ReqMethod = *entity.ReqMethod
	}
	if entity.ReqURL != nil {
		vo.ReqURL = *entity.ReqURL
	}
	if entity.ParamJSON != nil {
		vo.ParamJSON = *entity.ParamJSON
	}
	if entity.ResultJSON != nil {
		vo.ResultJSON = *entity.ResultJSON
	}
	if entity.TraceID != nil {
		vo.TraceID = *entity.TraceID
	}
	if entity.OpUser != nil {
		vo.OpUser = *entity.OpUser
	}
	if entity.SignData != nil {
		vo.SignData = *entity.SignData
	}
	if entity.CreatedBy != nil {
		vo.CreatedBy = *entity.CreatedBy
	}
	if entity.UpdatedBy != nil {
		vo.UpdatedBy = *entity.UpdatedBy
	}
	if entity.OpTime != nil {
		vo.OpTime = entity.OpTime.Format("2006-01-02 15:04:05")
	}
	if entity.CreatedAt != nil {
		vo.CreatedAt = entity.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if entity.UpdatedAt != nil {
		vo.UpdatedAt = entity.UpdatedAt.Format("2006-01-02 15:04:05")
	}

	return vo
}
