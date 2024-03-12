package organization

import (
	"context"
	"github.com/OpenIMSDK/chat/pkg/common/config"
	"github.com/OpenIMSDK/chat/pkg/common/constant"
	"github.com/OpenIMSDK/chat/pkg/common/db/database"
	organization2 "github.com/OpenIMSDK/chat/pkg/common/db/model/organization"
	table "github.com/OpenIMSDK/chat/pkg/common/db/table/organization"
	"github.com/OpenIMSDK/chat/pkg/common/dbconn"
	chat2 "github.com/OpenIMSDK/chat/pkg/proto/chat"
	"github.com/OpenIMSDK/chat/pkg/proto/organization"
	"github.com/OpenIMSDK/chat/pkg/rpclient/chat"
	"github.com/OpenIMSDK/protocol/wrapperspb"
	"github.com/OpenIMSDK/tools/discoveryregistry"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/utils"
	"google.golang.org/grpc"
	"time"
)

func Start(discov discoveryregistry.SvcDiscoveryRegistry, server *grpc.Server) error {
	db, err := dbconn.NewGormDB()
	if err != nil {
		return err
	}
	tables := []any{
		table.Department{},
		table.DepartmentMember{},
		table.Organization{},
	}
	if err := db.AutoMigrate(tables...); err != nil {
		return err
	}
	if err := organization2.NewDepartment(db).InitUngroupedName(context.Background(), constant.UngroupedID, config.Config.UngroupedName); err != nil {
		return err
	}
	if err := organization2.NewOrganization(db).Init(context.Background()); err != nil {
		return err
	}
	organization.RegisterOrganizationServer(server, &organizationSvr{
		Database: database.NewOrganizationDatabase(db),
		Chat:     chat.NewChatClient(discov),
		Admin:    chat.NewAdminClient(discov),
	})
	return nil
}

type organizationSvr struct {
	Database database.OrganizationDatabaseInterface
	Chat     *chat.ChatClient
	Admin    *chat.AdminClient
}

func (o *organizationSvr) GetDepartmentParents(ctx context.Context, req *organization.GetDepartmentParentsReq) (*organization.GetDepartmentParentsResp, error) {
	rootDepartment, err := o.Database.GetDepartment(ctx, req.DepartmentID)
	if IsNotFound(err) {
		return &organization.GetDepartmentParentsResp{}, nil
	} else if err != nil {
		return nil, err
	}
	var departments []*table.Department
	departments = append(departments, rootDepartment)
	ringDetection := make(map[string]struct{})
	ringDetection[rootDepartment.DepartmentID] = struct{}{}
	for {
		departmentID := departments[len(departments)-1].ParentDepartmentID
		if departmentID == "" {
			break
		}
		department, err := o.Database.GetDepartmentByID(ctx, departmentID)
		if err == nil {
			departments = append(departments, department)
			if _, ok := ringDetection[department.DepartmentID]; ok {
				return nil, errs.ErrInternalServer.Wrap("department ring detection")
			}
			ringDetection[department.DepartmentID] = struct{}{}
		} else if IsNotFound(err) {
			break
		} else {
			return nil, err
		}
	}
	return &organization.GetDepartmentParentsResp{Departments: utils.Batch(DepartmentDb2Pb, departments)}, nil
}

func (o *organizationSvr) GetDepartmentByName(ctx context.Context, req *organization.GetDepartmentByNameReq) (*organization.GetDepartmentByNameResp, error) {
	if len(req.Names) == 0 {
		return nil, errs.ErrArgs.Wrap("req.Names is empty")
	}
	var departments []*table.Department
	var parentDepartmentID string
	for _, name := range req.Names {
		department, err := o.Database.GetDepartmentByName(ctx, name, parentDepartmentID)
		if err == nil {
			parentDepartmentID = department.DepartmentID
			departments = append(departments, department)
		} else if IsNotFound(err) {
			break
		} else {
			return nil, err
		}
	}
	return &organization.GetDepartmentByNameResp{Departments: utils.Batch(DepartmentDb2Pb, departments)}, nil
}

func (o *organizationSvr) CreateDepartment(ctx context.Context, req *organization.CreateDepartmentReq) (*organization.CreateDepartmentResp, error) {
	if req.Name == "" {
		return nil, errs.ErrArgs.Wrap("name is empty")
	}
	if req.DepartmentID == "" {
		req.DepartmentID = genDepartmentID()
	}
	if req.Order == nil {
		req.Order = wrapperspb.Int32(constant.DefaultDepartmentOrder)
	}
	if req.DepartmentID == constant.UngroupedID {
		return nil, errs.ErrArgs.Wrap("departmentID is ungroupedID")
	}
	department := table.Department{
		DepartmentID:       req.DepartmentID,
		FaceURL:            req.FaceURL,
		Name:               req.Name,
		Order:              req.Order.Value,
		ParentDepartmentID: req.ParentDepartmentID,
		CreateTime:         time.Now(),
	}
	departmentIDs := []string{department.DepartmentID}
	if department.ParentDepartmentID != "" {
		if department.DepartmentID == department.ParentDepartmentID {
			return nil, errs.ErrArgs.Wrap("departmentID is equal parentDepartmentID")
		}
		departmentIDs = append(departmentIDs, department.ParentDepartmentID)
	}
	departments, err := o.Database.GetDepartmentList(ctx, departmentIDs)
	if err != nil {
		return nil, err
	}
	var parent bool
	for _, d := range departments {
		if d.DepartmentID == department.DepartmentID {
			return nil, errs.ErrArgs.Wrap("departmentID is exist")
		}
		if d.DepartmentID == department.ParentDepartmentID {
			parent = true
		}
	}
	if department.ParentDepartmentID != "" && (!parent) {
		return nil, errs.ErrRecordNotFound.Wrap("parent department not found")
	}
	if err := o.Database.CreateDepartment(ctx, &department); err != nil {
		return nil, err
	}
	return &organization.CreateDepartmentResp{DepartmentID: department.DepartmentID}, nil
}

func (o *organizationSvr) UpdateDepartment(ctx context.Context, req *organization.UpdateDepartmentReq) (*organization.UpdateDepartmentResp, error) {
	update, err := UpdateDepartment(req)
	if err != nil {
		return nil, err
	}
	departmentIDs := []string{req.DepartmentID}
	if req.ParentDepartmentID != nil && req.ParentDepartmentID.Value != "" {
		if req.DepartmentID == req.ParentDepartmentID.Value {
			return nil, errs.ErrArgs.Wrap("departmentID is equal parentDepartmentID")
		}
		departmentIDs = append(departmentIDs, req.ParentDepartmentID.Value)
	}
	departments, err := o.Database.GetDepartmentList(ctx, departmentIDs)
	if err != nil {
		return nil, err
	}
	if len(departments) != len(departmentIDs) {
		return nil, errs.ErrRecordNotFound.Wrap("department not found")
	}
	//departmentMap := utils.SliceToMap(departments, func(e *table.Department) string {
	//	return e.DepartmentID
	//})
	//if departmentMap[req.DepartmentID] == nil {
	//	return nil, errs.ErrRecordNotFound.Wrap("department not found")
	//}
	//if req.ParentDepartmentID != nil && req.ParentDepartmentID.Value != "" && departmentMap[req.ParentDepartmentID.Value] == nil {
	//	return nil, errs.ErrRecordNotFound.Wrap("parent department not found")
	//}
	if err := o.Database.UpdateDepartment(ctx, req.DepartmentID, update); err != nil {
		return nil, err
	}
	return &organization.UpdateDepartmentResp{}, nil
}

func (o *organizationSvr) GetOrganizationDepartment(ctx context.Context, req *organization.GetOrganizationDepartmentReq) (*organization.GetOrganizationDepartmentResp, error) {
	resp := &organization.GetOrganizationDepartmentResp{Departments: []*organization.DepartmentInfo{}}
	var getSubDepartmentList func(departmentId string, list *[]*organization.DepartmentInfo) error
	getSubDepartmentList = func(departmentId string, list *[]*organization.DepartmentInfo) error {
		departments, err := o.Database.GetParentDepartment(ctx, departmentId)
		if err != nil {
			return err
		}
		for _, department := range departments {
			subs := make([]*organization.DepartmentInfo, 0)
			err = getSubDepartmentList(department.DepartmentID, &subs)
			if err != nil {
				return err
			}
			num, err := o.GetDepartmentMemberNum(ctx, department.DepartmentID)
			if err != nil {
				return err
			}
			*list = append(*list, &organization.DepartmentInfo{
				Department:     DepartmentNumDb2Pb(department, uint32(num)),
				Subdepartments: subs,
			})
		}
		return nil
	}

	if err := getSubDepartmentList("", &resp.Departments); err != nil {
		return nil, err
	}
	return resp, nil
}

func (o *organizationSvr) DeleteDepartment(ctx context.Context, req *organization.DeleteDepartmentReq) (*organization.DeleteDepartmentResp, error) {
	if len(req.DepartmentIDs) == 0 {
		return nil, errs.ErrArgs.Wrap("req.DepartmentIDs is empty")
	}
	if utils.Contain(constant.UngroupedID, req.DepartmentIDs...) {
		return nil, errs.ErrArgs.Wrap("can not delete ungrouped department")
	}
	departmentList, err := o.Database.GetDepartmentList(ctx, req.DepartmentIDs)
	if err != nil {
		return nil, err
	}
	if len(departmentList) == 0 {
		return nil, errs.ErrArgs.Wrap("department not found")
	}
	departmentMembers, err := o.Database.FindDepartmentMember(ctx, req.DepartmentIDs)
	if err != nil {
		return nil, err
	}
	if len(departmentMembers) > 0 {
		defer func() {
			for _, member := range departmentMembers {
				_, err := o.AddUserToUngrouped(ctx, &organization.AddUserToUngroupedReq{UserID: member.UserID})
				if err != nil {
					log.ZError(ctx, "AddUserToUngrouped", err, "userID", member.UserID)
				}
			}
		}()
	}
	// 修改删除的子部门的父部门为删除的上级
	for _, department := range departmentList {
		err := o.Database.UpdateParentID(ctx, department.DepartmentID, department.ParentDepartmentID)
		if err != nil {
			return nil, err
		}
	}
	// 删除职位信息
	if err := o.Database.DeleteDepartmentIDList(ctx, req.DepartmentIDs); err != nil {
		return nil, err
	}
	// 删除部门
	if err := o.Database.DeleteDepartment(ctx, req.DepartmentIDs); err != nil {
		return nil, err
	}
	return &organization.DeleteDepartmentResp{}, nil
}

func (o *organizationSvr) GetDepartment(ctx context.Context, req *organization.GetDepartmentReq) (*organization.GetDepartmentResp, error) {
	resp := &organization.GetDepartmentResp{}
	if len(req.DepartmentIDs) == 0 {
		org, err := o.Database.GetOrganization(ctx)
		if err != nil {
			return nil, err
		}
		return &organization.GetDepartmentResp{
			Departments: []*organization.Department{Organization2Department(org)},
		}, nil
	}
	departments, err := o.Database.GetDepartmentList(ctx, req.DepartmentIDs)
	if err != nil {
		return nil, err
	}
	for _, department := range departments {
		resp.Departments = append(resp.Departments, DepartmentDb2Pb(department))
	}
	return resp, nil
}

func (o *organizationSvr) CreateDepartmentMember(ctx context.Context, req *organization.CreateDepartmentMemberReq) (*organization.CreateDepartmentMemberResp, error) {
	if req.UserID == "" {
		return nil, errs.ErrArgs.Wrap("userID is empty")
	}
	if req.DepartmentID == constant.UngroupedID {
		return nil, errs.ErrArgs.Wrap("can not add user to ungrouped department")
	}
	if _, err := o.Chat.GetUserPublicInfo(ctx, req.UserID); err != nil {
		return nil, err
	}
	if _, err := o.Database.GetDepartmentByID(ctx, req.DepartmentID); err != nil {
		return nil, err
	}
	var terminationTime *time.Time
	if req.TerminationTime > req.EntryTime {
		t := time.UnixMilli(req.EntryTime)
		terminationTime = &t
	}
	err := o.Database.CreateDepartmentMember(ctx, &table.DepartmentMember{
		UserID:          req.UserID,
		DepartmentID:    req.DepartmentID,
		Position:        req.Position,
		Station:         req.Station,
		Order:           req.Order,
		EntryTime:       time.UnixMilli(req.EntryTime),
		TerminationTime: terminationTime,
		CreateTime:      time.Now(),
	})
	if err != nil {
		return nil, err
	}
	if _, err := o.AddUserToUngrouped(ctx, &organization.AddUserToUngroupedReq{UserID: req.UserID}); err != nil {
		return nil, err
	}
	return &organization.CreateDepartmentMemberResp{}, nil
}

func (o *organizationSvr) GetUserInDepartment(ctx context.Context, req *organization.GetUserInDepartmentReq) (*organization.GetUserInDepartmentResp, error) {
	if len(req.UserIDs) == 0 {
		return nil, errs.ErrArgs.Wrap("req.UserIDs is nil")
	}
	if utils.Duplicate(req.UserIDs) {
		return nil, errs.ErrArgs.Wrap("req.UserIDs is duplicate")
	}
	dbMembers, err := o.Database.GetDepartmentMemberInUserID(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	memberMap := make(map[string][]*table.DepartmentMember)
	for _, member := range dbMembers {
		memberMap[member.UserID] = append(memberMap[member.UserID], member)
	}
	departmentIDList := make([]string, 0, len(dbMembers))
	for _, dm := range dbMembers {
		departmentIDList = append(departmentIDList, dm.DepartmentID)
	}
	departmentList, err := o.Database.GetDepartmentList(ctx, utils.Distinct(departmentIDList))
	if err != nil {
		return nil, err
	}
	departmentMap := make(map[string]*organization.DepartmentNum)
	for _, department := range departmentList {
		num, err := o.GetDepartmentMemberNum(ctx, department.DepartmentID)
		if err != nil {
			return nil, err
		}
		departmentMap[department.DepartmentID] = DepartmentNumDb2Pb(department, uint32(num))
	}
	userMap, err := o.Chat.MapUserFullInfo(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	resp := &organization.GetUserInDepartmentResp{Users: make([]*organization.DepartmentMemberUser, 0, len(dbMembers))}
	for _, userID := range req.UserIDs {
		members := memberMap[userID]
		user := userMap[userID]
		if members == nil || user == nil {
			continue
		}
		resp.Users = append(resp.Users, &organization.DepartmentMemberUser{
			User: user,
			Members: utils.Slice(members, func(e *table.DepartmentMember) *organization.MemberDepartment {
				return MemberDepartmentDb2Pb(e, departmentMap[e.DepartmentID])
			}),
		})
	}
	return resp, nil
}

func (o *organizationSvr) DeleteUserInDepartment(ctx context.Context, req *organization.DeleteUserInDepartmentReq) (*organization.DeleteUserInDepartmentResp, error) {
	if req.DepartmentID == constant.UngroupedID {
		return nil, errs.ErrArgs.Wrap("can not delete user in ungrouped department")
	}
	if err := o.Database.DeleteDepartmentMemberByKey(ctx, req.UserID, req.DepartmentID); err != nil {
		return nil, err
	}
	if _, err := o.AddUserToUngrouped(ctx, &organization.AddUserToUngroupedReq{UserID: req.UserID}); err != nil {
		return nil, err
	}
	return &organization.DeleteUserInDepartmentResp{}, nil
}

func (o *organizationSvr) UpdateUserInDepartment(ctx context.Context, req *organization.UpdateUserInDepartmentReq) (*organization.UpdateUserInDepartmentResp, error) {
	resp := &organization.UpdateUserInDepartmentResp{}
	update, err := UpdateDepartmentMember(req)
	if err != nil {
		return nil, err
	}
	if _, err := o.Database.GetDepartmentMemberByKey(ctx, req.UserID, req.DepartmentID); err != nil {
		return nil, err
	}
	if err := o.Database.UpdateDepartmentMember(ctx, req.DepartmentID, req.UserID, update); err != nil {
		return nil, err
	}
	return resp, nil
}

func (o *organizationSvr) SetOrganization(ctx context.Context, req *organization.SetOrganizationReq) (*organization.SetOrganizationResp, error) {
	update, err := UpdateOrganization(req)
	if err != nil {
		return nil, err
	}
	if err := o.Database.SetOrganization(ctx, update); err != nil {
		return nil, err
	}
	return &organization.SetOrganizationResp{}, nil
}

func (o *organizationSvr) GetOrganization(ctx context.Context, req *organization.GetOrganizationReq) (*organization.GetOrganizationResp, error) {
	org, err := o.Database.GetOrganization(ctx)
	if err != nil {
		return nil, err
	}
	return &organization.GetOrganizationResp{
		LogoURL:      org.LogoURL,
		Name:         org.Name,
		Homepage:     org.Homepage,
		Introduction: org.Introduction,
		CreateTime:   org.CreateTime.UnixMilli(),
	}, nil
}

func (o *organizationSvr) GetSubDepartment(ctx context.Context, req *organization.GetSubDepartmentReq) (*organization.GetSubDepartmentResp, error) {
	resp := &organization.GetSubDepartmentResp{}
	departmentList, err := o.Database.GetParentDepartment(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}
	for _, department := range departmentList {
		num, err := o.GetDepartmentMemberNum(ctx, department.DepartmentID)
		if err != nil {
			return nil, err
		}
		resp.Departments = append(resp.Departments, DepartmentNumDb2Pb(department, uint32(num)))
	}
	members, err := o.Database.GetDepartmentMemberByDepartmentID(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}
	userIDs := utils.Distinct(utils.Slice(members, func(e *table.DepartmentMember) string {
		return e.UserID
	}))
	userMap, err := o.Chat.MapUserFullInfo(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	blockMap, err := o.Admin.FindUserBlockInfo(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		user := userMap[member.UserID]
		if user == nil {
			continue
		}
		resp.Members = append(resp.Members, &organization.MemberUserInfo{
			Member:   DepartmentMemberDb2Pb(member),
			User:     user,
			Disabled: blockMap[member.UserID] != nil,
		})
	}
	departmentID := req.DepartmentID
	if departmentID != "" {
		department, err := o.Database.GetDepartmentByID(ctx, req.DepartmentID)
		if err != nil {
			return nil, err
		}
		departmentID = department.ParentDepartmentID
	}
	org, err := o.Database.GetOrganization(ctx)
	if err != nil {
		return nil, err
	}
	num, err := o.Database.GetMemberNum(ctx, nil)
	if err != nil {
		return nil, err
	}
	if req.DepartmentID == "" {
		resp.Current = Organization2DepartmentNum(org, uint32(num))
	} else {
		resp.Parents = append(resp.Parents, Organization2DepartmentNum(org, uint32(num)))
		parents, err := o.GetDepartmentParents(ctx, &organization.GetDepartmentParentsReq{DepartmentID: departmentID})
		if err != nil {
			return nil, err
		}
		for i := len(parents.Departments) - 1; i >= 0; i-- {
			d := parents.Departments[i]
			num, err := o.GetDepartmentMemberNum(ctx, d.DepartmentID)
			if err != nil {
				return nil, err
			}
			resp.Parents = append(resp.Parents, DepartmentNumPb2Pb(d, uint32(num)))
		}
		current, err := o.Database.GetDepartmentByID(ctx, req.DepartmentID)
		if err != nil {
			return nil, err
		}
		num, err := o.GetDepartmentMemberNum(ctx, current.DepartmentID)
		if err != nil {
			return nil, err
		}
		resp.Current = DepartmentNumDb2Pb(current, uint32(num))
	}
	return resp, nil
}

func (o *organizationSvr) GetSearchDepartmentUser(ctx context.Context, req *organization.GetSearchDepartmentUserReq) (*organization.GetSearchDepartmentUserResp, error) {
	defer log.ZDebug(ctx, "return")
	if req.Pagination == nil {
		return nil, errs.ErrArgs.Wrap("pagination is nil")
	}
	departmentIDs, err := o.Database.SearchDepartment(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}
	userIDs, err := o.Database.SearchDepartmentMember(ctx, req.Keyword, departmentIDs)
	if err != nil {
		return nil, err
	}
	searchResp, err := o.Chat.SearchUserID(ctx, &chat2.SearchUserIDReq{
		Keyword:    req.Keyword,
		OrUserIDs:  userIDs,
		Pagination: req.Pagination,
	})
	if err != nil {
		return nil, err
	}
	resp := &organization.GetSearchDepartmentUserResp{Total: searchResp.Total}
	if len(searchResp.UserIDs) > 0 {
		userResp, err := o.GetUserInDepartment(ctx, &organization.GetUserInDepartmentReq{UserIDs: searchResp.UserIDs})
		if err != nil {
			return nil, err
		}
		resp.Users = userResp.Users
	}
	return resp, nil
}

func (o *organizationSvr) SortDepartmentList(ctx context.Context, req *organization.SortDepartmentListReq) (*organization.SortDepartmentListResp, error) {
	if req.DepartmentID == req.NextDepartmentID {
		return nil, errs.ErrArgs.Wrap("department id equal")
	}
	department, err := o.Database.GetDepartmentByID(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}
	var setOrder int32
	if req.NextDepartmentID == "" {
		order, err := o.Database.GetMaxDepartmentOrder(ctx, department.ParentDepartmentID)
		if err != nil {
			return nil, err
		}
		setOrder = order + 1
	} else {
		nextDepartment, err := o.Database.GetDepartmentByID(ctx, req.NextDepartmentID)
		if err != nil {
			return nil, err
		}
		if department.ParentDepartmentID != nextDepartment.ParentDepartmentID {
			return nil, errs.ErrArgs.Wrap("department parent department id not equal")
		}
		if err := o.Database.IncrDepartmentOrder(ctx, nextDepartment.ParentDepartmentID, nextDepartment.Order); err != nil {
			return nil, err
		}
		setOrder = nextDepartment.Order
	}
	update, err := UpdateDepartment(&organization.UpdateDepartmentReq{
		DepartmentID: department.DepartmentID,
		Order:        wrapperspb.Int32(setOrder),
	})
	if err != nil {
		return nil, err
	}
	if err := o.Database.UpdateDepartment(ctx, department.DepartmentID, update); err != nil {
		return nil, err
	}
	return &organization.SortDepartmentListResp{Order: setOrder}, nil
}

func (o *organizationSvr) SortOrganizationUserList(ctx context.Context, req *organization.SortOrganizationUserListReq) (*organization.SortOrganizationUserListResp, error) {
	if req.DepartmentID == "" {
		return nil, errs.ErrArgs.Wrap("department id is empty")
	}
	if _, err := o.Database.GetDepartmentByID(ctx, req.DepartmentID); err != nil {
		return nil, err
	}
	if _, err := o.Database.GetDepartmentMemberByKey(ctx, req.UserID, req.DepartmentID); err != nil {
		return nil, err
	}
	var setOrder int32
	if req.NextUserID == "" {
		order, err := o.Database.GetMaxDepartmentMemberOrder(ctx, req.DepartmentID)
		if err != nil {
			return nil, err
		}
		setOrder = order + 1
	} else {
		member, err := o.Database.GetDepartmentMemberByKey(ctx, req.NextUserID, req.DepartmentID)
		if err != nil {
			return nil, err
		}
		if err := o.Database.IncrDepartmentMemberOrder(ctx, req.DepartmentID, member.Order); err != nil {
			return nil, err
		}
		setOrder = member.Order
	}
	update, err := UpdateDepartmentMember(&organization.UpdateUserInDepartmentReq{
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		Order:        wrapperspb.Int32(setOrder),
	})
	if err != nil {
		return nil, err
	}
	if err := o.Database.UpdateDepartmentMember(ctx, req.DepartmentID, req.UserID, update); err != nil {
		return nil, err
	}
	return &organization.SortOrganizationUserListResp{Order: setOrder}, nil
}

func (o *organizationSvr) MoveUserDepartment(ctx context.Context, req *organization.MoveUserDepartmentReq) (*organization.MoveUserDepartmentResp, error) {
	if len(req.Moves) == 0 {
		return nil, errs.ErrArgs.Wrap("move user department list is empty")
	}
	ungroupedUserIDSet := make(map[string]struct{})
	srcMap := make(map[[2]string]*table.DepartmentMember)
	for _, department := range req.Moves {
		if department.DepartmentID == constant.UngroupedID {
			return nil, errs.ErrArgs.Wrap("can not move user to ungrouped department")
		}
		if department.CurrentDepartmentID == constant.UngroupedID {
			ungroupedUserIDSet[department.UserID] = struct{}{}
		}
		if department.DepartmentID == department.CurrentDepartmentID {
			return nil, errs.ErrArgs.Wrap("department after equal")
		}
		key := [...]string{department.CurrentDepartmentID, department.UserID}
		if _, ok := srcMap[key]; ok {
			continue
		}
		member, err := o.Database.GetDepartmentMemberByKey(ctx, department.UserID, department.CurrentDepartmentID)
		if err != nil {
			return nil, err
		}
		srcMap[key] = member
	}
	members := make([]*table.DepartmentMember, 0, len(req.Moves))
	for _, department := range req.Moves {
		value := srcMap[[...]string{department.CurrentDepartmentID, department.UserID}]
		members = append(members, &table.DepartmentMember{
			UserID:          value.UserID,
			DepartmentID:    department.DepartmentID,
			Position:        value.Position,
			Station:         value.Station,
			Order:           value.Order,
			EntryTime:       value.EntryTime,
			TerminationTime: value.TerminationTime,
			CreateTime:      time.Now(),
		})
	}
	if err := o.Database.CreateDepartmentMembers(ctx, members); err != nil {
		return nil, err
	}
	for _, member := range srcMap {
		if err := o.Database.DeleteDepartmentMemberByKey(ctx, member.UserID, member.DepartmentID); err != nil {
			log.ZError(ctx, "DeleteDepartmentMemberByKey", err, "userID", member.UserID, "departmentID", member.DepartmentID)
		}
	}
	for userID := range ungroupedUserIDSet {
		if _, err := o.AddUserToUngrouped(ctx, &organization.AddUserToUngroupedReq{UserID: userID}); err != nil {
			log.ZError(ctx, "AddUserToUngrouped", err, "userID", userID)
		}
	}
	return &organization.MoveUserDepartmentResp{}, nil
}

func (o *organizationSvr) AddUserToUngrouped(ctx context.Context, req *organization.AddUserToUngroupedReq) (*organization.AddUserToUngroupedResp, error) {
	user, err := o.Chat.GetUserPublicInfo(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	members, err := o.Database.FindDepartmentMemberByUserID(ctx, []string{user.UserID})
	if err != nil {
		return nil, err
	}
	switch len(members) {
	case 0:
		err = o.Database.CreateDepartmentMember(ctx, &table.DepartmentMember{
			UserID:       user.UserID,
			DepartmentID: constant.UngroupedID,
			EntryTime:    time.Now(),
			CreateTime:   time.Now(),
		})
		if err != nil {
			return nil, err
		}
		return &organization.AddUserToUngroupedResp{Ungrouped: true}, nil
	case 1:
		return &organization.AddUserToUngroupedResp{Ungrouped: members[0].DepartmentID == constant.UngroupedID}, nil
	default:
		if err := o.Database.DeleteDepartmentMemberByKey(ctx, user.UserID, constant.UngroupedID); err != nil {
			return nil, err
		}
		return &organization.AddUserToUngroupedResp{Ungrouped: false}, nil
	}
}
