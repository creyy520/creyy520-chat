package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/OpenIMSDK/chat/pkg/common/apicall"
	"github.com/OpenIMSDK/chat/pkg/common/apistruct"
	"github.com/OpenIMSDK/chat/pkg/common/config"
	"github.com/OpenIMSDK/chat/pkg/common/mctx"
	"github.com/OpenIMSDK/chat/pkg/common/xlsx"
	"github.com/OpenIMSDK/chat/pkg/common/xlsx/model"
	"github.com/OpenIMSDK/chat/pkg/proto/admin"
	"github.com/OpenIMSDK/chat/pkg/proto/chat"
	"github.com/OpenIMSDK/chat/pkg/proto/common"
	"github.com/OpenIMSDK/chat/pkg/proto/organization"
	constant2 "github.com/OpenIMSDK/protocol/constant"
	"github.com/OpenIMSDK/protocol/sdkws"
	"github.com/OpenIMSDK/protocol/wrapperspb"
	"github.com/OpenIMSDK/tools/a2r"
	"github.com/OpenIMSDK/tools/apiresp"
	"github.com/OpenIMSDK/tools/checker"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/utils"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewOrg(chatConn, adminConn, orgConn grpc.ClientConnInterface) *Org {
	return &Org{
		organizationClient: organization.NewOrganizationClient(orgConn),
		chatClient:         chat.NewChatClient(chatConn),
		adminClient:        admin.NewAdminClient(adminConn),
		imApiCaller:        apicall.NewCallerInterface(),
	}
}

type Org struct {
	organizationClient organization.OrganizationClient
	chatClient         chat.ChatClient
	adminClient        admin.AdminClient
	imApiCaller        apicall.CallerInterface
}

type registerUserDepartment struct {
	DepartmentID    string `json:"departmentID"`
	Position        string `json:"position"`
	EntryTime       int64  `json:"entryTime"`
	TerminationTime int64  `json:"terminationTime"`
}

type registerUserReq struct {
	Departments []registerUserDepartment `json:"departments"`
	User        chat.RegisterUserInfo    `json:"user"`
}

func (o *Org) RegisterUser(c *gin.Context) {
	var req registerUserReq
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	ip, err := o.getClientIP(c)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	userID, err := o.registerUser(c, &req, ip)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, &struct {
		UserID string `json:"userID"`
	}{UserID: userID})
}

func (o *Org) registerUser(ctx context.Context, req *registerUserReq, ip string) (string, error) {
	if err := checker.Validate(&req); err != nil {
		return "", err
	}
	if len(req.Departments) > 0 {
		departmentIDs := make(map[string]struct{})
		for _, department := range req.Departments {
			departmentIDs[department.DepartmentID] = struct{}{}
		}
		if len(departmentIDs) != len(req.Departments) {
			return "", errs.ErrArgs.Wrap("departmentID repeat")
		}
		resp, err := o.organizationClient.GetDepartment(ctx, &organization.GetDepartmentReq{DepartmentIDs: utils.Keys(departmentIDs)})
		if err != nil {
			return "", err
		}
		if len(resp.Departments) != len(departmentIDs) {
			return "", errs.ErrArgs.Wrap("departmentID not exist")
		}
	}
	respRegisterUser, err := o.chatClient.RegisterUser(ctx, &chat.RegisterUserReq{
		Ip:        ip,
		Platform:  constant2.AdminPlatformID,
		AutoLogin: false,
		User:      &req.User,
	})
	if err != nil {
		return "", err
	}
	var tag int
	if len(req.Departments) > 0 {
		for _, department := range req.Departments {
			_, err := o.organizationClient.CreateDepartmentMember(ctx, &organization.CreateDepartmentMemberReq{
				UserID:          respRegisterUser.UserID,
				DepartmentID:    department.DepartmentID,
				Position:        department.Position,
				EntryTime:       department.EntryTime,
				TerminationTime: department.TerminationTime,
			})
			if err == nil {
				tag++
			} else {
				log.ZError(ctx, "CreateDepartmentMember err", err, "userID", respRegisterUser.UserID, "departmentID", department.DepartmentID)
			}
		}
	}
	if tag == 0 {
		_, err := o.organizationClient.AddUserToUngrouped(ctx, &organization.AddUserToUngroupedReq{UserID: respRegisterUser.UserID})
		if err != nil {
			log.ZError(ctx, "AddUserToUngrouped err", err, "userID", respRegisterUser.UserID)
		}
	}
	userInfo := &sdkws.UserInfo{
		UserID:     respRegisterUser.UserID,
		Nickname:   req.User.Nickname,
		FaceURL:    req.User.FaceURL,
		CreateTime: time.Now().UnixMilli(),
	}
	err = o.imApiCaller.RegisterUser(ctx, []*sdkws.UserInfo{userInfo})
	if err != nil {
		return "", err
	}
	func() {
		imToken, err := o.imApiCaller.ImAdminTokenWithDefaultAdmin(ctx)
		if err != nil {
			log.ZError(ctx, "ImAdminTokenWithDefaultAdmin err", err)
			return
		}
		apiCtx := mctx.WithApiToken(ctx, imToken)
		rpcCtx := mctx.WithAdminUser(ctx)
		if resp, err := o.adminClient.FindDefaultFriend(rpcCtx, &admin.FindDefaultFriendReq{}); err == nil {
			if len(resp.UserIDs) > 0 {
				if err := o.imApiCaller.ImportFriend(apiCtx, respRegisterUser.UserID, resp.UserIDs); err != nil {
					log.ZError(ctx, "ImportFriend err", err, "userID", respRegisterUser.UserID, "friendIDs", resp.UserIDs)
				}
			}
		} else {
			log.ZError(ctx, "FindDefaultFriend err", err, "userID", respRegisterUser.UserID)
		}
		if resp, err := o.adminClient.FindDefaultGroup(rpcCtx, &admin.FindDefaultGroupReq{}); err == nil {
			if len(resp.GroupIDs) > 0 {
				if err := o.imApiCaller.InviteToGroup(apiCtx, respRegisterUser.UserID, resp.GroupIDs); err != nil {
					log.ZError(ctx, "InviteToGroup err", err, "userID", respRegisterUser.UserID, "groupIDs", resp.GroupIDs)
				}
			}
		} else {
			log.ZError(ctx, "FindDefaultGroup err", err, "userID", respRegisterUser.UserID)
		}
	}()
	return userInfo.UserID, nil
}

func (o *Org) CreateDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.CreateDepartment, o.organizationClient, c)
}

func (o *Org) UpdateDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.UpdateDepartment, o.organizationClient, c)
}

func (o *Org) DeleteDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.DeleteDepartment, o.organizationClient, c)
}

func (o *Org) GetDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetDepartment, o.organizationClient, c)
}

func (o *Org) UpdateOrganizationUser(c *gin.Context) {
	var (
		req  chat.UpdateUserInfoReq
		resp apistruct.UpdateUserInfoResp
	)
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	log.ZInfo(c, "updateUserInfo", "req", &req)
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err) // 参数校验失败
		return
	}
	respUpdate, err := o.chatClient.UpdateUserInfo(c, &req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	imToken, err := o.imApiCaller.UserToken(c, config.GetIMAdmin(mctx.GetOpUserID(c)), constant2.AdminPlatformID)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	var (
		nickName string
		faceURL  string
	)
	if req.Nickname != nil {
		nickName = req.Nickname.Value
	} else {
		nickName = respUpdate.NickName
	}
	if req.FaceURL != nil {
		faceURL = req.FaceURL.Value
	} else {
		faceURL = respUpdate.FaceUrl
	}
	err = o.imApiCaller.UpdateUserInfo(mctx.WithApiToken(c, imToken), req.UserID, nickName, faceURL)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, resp)
}

func (o *Org) CreateDepartmentMember(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.CreateDepartmentMember, o.organizationClient, c)
}

func (o *Org) GetUserInDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetUserInDepartment, o.organizationClient, c)
	//var req organization.GetUserInDepartmentReq
	//if err := c.BindJSON(&req); err != nil {
	//	apiresp.GinError(c, err)
	//	return
	//}
	//if err := checker.Validate(&req); err != nil {
	//	apiresp.GinError(c, err)
	//	return
	//}
	//
	//respDepartment, err := o.organizationClient.GetUserInDepartment(c, &req)
	//if err != nil {
	//	apiresp.GinError(c, err)
	//	return
	//}
	//type UserFullInfo struct {
	//	*common.UserFullInfo
	//	Departments []*organization.DepartmentMember `json:"departments"`
	//}
	//type FindUserFullInfoResp struct {
	//	Users []*UserFullInfo `json:"users"`
	//}
	//resp := FindUserFullInfoResp{
	//	Users: make([]*UserFullInfo, 0),
	//}
	//if len(respDepartment.Departments) > 0 {
	//	userIDMap := make(map[string][]*organization.DepartmentMember)
	//	for _, department := range respDepartment.Departments {
	//		userIDMap[department.UserID] = append(userIDMap[department.UserID], append(userIDMap[department.UserID], department)...)
	//	}
	//	respUsers, err := o.chatClient.FindUserFullInfo(c, &chat.FindUserFullInfoReq{UserIDs: utils.Keys(userIDMap)})
	//	if err != nil {
	//		apiresp.GinError(c, err)
	//		return
	//	}
	//	for _, user := range respUsers.Users {
	//		resp.Users = append(resp.Users, &UserFullInfo{
	//			UserFullInfo: user,
	//			Departments:  userIDMap[user.UserID],
	//		})
	//	}
	//}
	//apiresp.GinSuccess(c, resp)
}

func (o *Org) UpdateUserInDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.UpdateUserInDepartment, o.organizationClient, c)
}

func (o *Org) DeleteUserInDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.DeleteUserInDepartment, o.organizationClient, c)
}

func (o *Org) GetSearchUserList(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetSearchDepartmentUser, o.organizationClient, c)
}

func (o *Org) SetOrganization(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.SetOrganization, o.organizationClient, c)
}

func (o *Org) GetOrganization(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetOrganization, o.organizationClient, c)
}

func (o *Org) MoveUserDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.MoveUserDepartment, o.organizationClient, c)
}

func (o *Org) GetSubDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetSubDepartment, o.organizationClient, c)
}

func (o *Org) GetSearchDepartmentUser(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetSearchDepartmentUser, o.organizationClient, c)
}

func (o *Org) FindUserPublicInfo(c *gin.Context) {
	var req organization.GetUserInDepartmentReq
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	respDepartment, err := o.organizationClient.GetUserInDepartment(c, &req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	type User struct {
		*common.UserPublicInfo
		Members []*organization.MemberDepartment `json:"members"`
	}
	users := make([]*User, 0, len(respDepartment.Users))
	for _, member := range respDepartment.Users {
		members := member.Members
		utils.InitSlice(&members)
		users = append(users, &User{
			UserPublicInfo: o.userFullToPublic(member.User),
			Members:        members,
		})
	}
	apiresp.GinSuccess(c, gin.H{"users": users})
}

func (o *Org) userFullToPublic(userFull *common.UserFullInfo) *common.UserPublicInfo {
	return &common.UserPublicInfo{
		UserID:   userFull.UserID,
		Account:  userFull.Account,
		Email:    userFull.Email,
		Nickname: userFull.Nickname,
		FaceURL:  userFull.FaceURL,
		Gender:   userFull.Gender,
		Level:    userFull.Level,
	}
}

func (o *Org) FindUserFullInfo(c *gin.Context) {
	var req organization.GetUserInDepartmentReq
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	respDepartment, err := o.organizationClient.GetUserInDepartment(c, &req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	type User struct {
		*common.UserFullInfo
		Members []*organization.MemberDepartment `json:"members"`
	}
	users := make([]*User, 0, len(respDepartment.Users))
	for _, member := range respDepartment.Users {
		members := member.Members
		utils.InitSlice(&members)
		users = append(users, &User{
			UserFullInfo: member.User,
			Members:      members,
		})
	}
	apiresp.GinSuccess(c, gin.H{"users": users})
}

func (o *Org) SearchUserFullInfo(c *gin.Context) {
	var req organization.GetSearchDepartmentUserReq
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	respSearch, err := o.organizationClient.GetSearchDepartmentUser(c, &req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	type User struct {
		*common.UserFullInfo
		Members []*organization.MemberDepartment `json:"members"`
	}
	users := make([]*User, 0, len(respSearch.Users))
	for _, member := range respSearch.Users {
		members := member.Members
		utils.InitSlice(&members)
		users = append(users, &User{
			UserFullInfo: member.User,
			Members:      members,
		})
	}
	apiresp.GinSuccess(c, gin.H{"total": respSearch.Total, "users": users})
}

func (o *Org) SearchUserPublicInfo(c *gin.Context) {
	var req organization.GetSearchDepartmentUserReq
	if err := c.BindJSON(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err)
		return
	}
	respSearch, err := o.organizationClient.GetSearchDepartmentUser(c, &req)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	type User struct {
		*common.UserPublicInfo
		Members []*organization.MemberDepartment `json:"members"`
	}
	users := make([]*User, 0, len(respSearch.Users))
	for _, member := range respSearch.Users {
		members := member.Members
		utils.InitSlice(&members)
		users = append(users, &User{
			UserPublicInfo: o.userFullToPublic(member.User),
			Members:        members,
		})
	}
	apiresp.GinSuccess(c, gin.H{"total": respSearch.Total, "users": users})
}

func (o *Org) GetOrganizationDepartment(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.GetOrganizationDepartment, o.organizationClient, c)
}

func (o *Org) SortDepartmentList(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.SortDepartmentList, o.organizationClient, c)
}

func (o *Org) SortOrganizationUserList(c *gin.Context) {
	a2r.Call(organization.OrganizationClient.SortOrganizationUserList, o.organizationClient, c)
}

func (o *Org) BatchImportTemplate(c *gin.Context) {
	md5Sum := md5.Sum(config.ImportTemplate)
	md5Val := hex.EncodeToString(md5Sum[:])
	if c.GetHeader("If-None-Match") == md5Val {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=template.xlsx")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Length", strconv.Itoa(len(config.ImportTemplate)))
	c.Header("ETag", md5Val)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", config.ImportTemplate)
}

func (o *Org) BatchImport(c *gin.Context) {
	if err := o.batchImport(c); err != nil {
		apiresp.GinError(c, err)
		return
	}
	apiresp.GinSuccess(c, nil)
}

func (o *Org) batchImport(c *gin.Context) error {
	const departmentNameSeparator = "/"
	ip, err := o.getClientIP(c)
	if err != nil {
		return err
	}
	formFile, err := c.FormFile("data")
	if err != nil {
		return err
	}
	file, err := formFile.Open()
	if err != nil {
		return err
	}
	defer file.Close()
	var (
		departments []model.Department
		users       []model.OrganizationUser
	)
	if err := xlsx.ParseAll(file, &departments, &users); err != nil {
		return err
	}
	for _, department := range departments {
		if department.Ignore {
			continue
		}
		if department.Name == "" {
			return errs.ErrArgs.Wrap("name is empty")
		}
		names := strings.Split(department.Name, departmentNameSeparator)
		for _, name := range names {
			if name == "" {
				return errs.ErrArgs.Wrap("name is empty")
			}
		}
	}
	for i, user := range users {
		if user.Ignore {
			continue
		}
		if user.Nickname == "" {
			return errs.ErrArgs.Wrap("nickname is empty")
		}
		if user.AreaCode == "" || user.PhoneNumber == "" {
			return errs.ErrArgs.Wrap("areaCode or phoneNumber is empty")
		}
		if user.Password == "" {
			return errs.ErrArgs.Wrap("password is empty")
		}
		if !strings.HasPrefix(user.AreaCode, "+") {
			return errs.ErrArgs.Wrap("areaCode format error")
		}
		if _, err := strconv.ParseUint(user.AreaCode[1:], 10, 16); err != nil {
			return errs.ErrArgs.Wrap("areaCode format error")
		}
		users[i].Password = utils.Md5(user.Password)
	}
	// 创建部门
	for _, department := range departments {
		if department.Ignore {
			continue
		}
		names := strings.Split(department.Name, departmentNameSeparator)
		if department.ID != "" {
			resp, err := o.organizationClient.GetDepartmentParents(c, &organization.GetDepartmentParentsReq{DepartmentID: department.ID})
			if err != nil {
				return err
			}
			if len(resp.Departments) > 0 {
				dbNames := make([]string, len(resp.Departments))
				for i, d := range resp.Departments {
					dbNames[len(dbNames)-1-i] = d.Name
				}
				if strings.Join(dbNames, departmentNameSeparator) != strings.Join(names, departmentNameSeparator) {
					return errs.ErrArgs.Wrap(fmt.Sprintf("department %s name not match", department.ID))
				}
				continue
			}
		}
		resp, err := o.organizationClient.GetDepartmentByName(c, &organization.GetDepartmentByNameReq{Names: names})
		if err != nil {
			return err
		}
		if len(resp.Departments) >= len(names) {
			continue
		}
		var parentDepartmentID string
		if len(parentDepartmentID) > 0 {
			parentDepartmentID = resp.Departments[len(resp.Departments)-1].DepartmentID
		}
		names = names[len(resp.Departments):]
		for i, name := range names {
			var departmentID string
			if len(names)-1 == i {
				departmentID = department.ID
			}
			var order *wrapperspb.Int32Value
			if val, err := strconv.Atoi(department.Order); err == nil {
				order = wrapperspb.Int32(int32(val))
			}
			respCreate, err := o.organizationClient.CreateDepartment(c, &organization.CreateDepartmentReq{
				DepartmentID:       departmentID,
				ParentDepartmentID: parentDepartmentID,
				FaceURL:            department.FaceURL,
				Name:               name,
				Order:              order,
			})
			if err != nil {
				return err
			}
			parentDepartmentID = respCreate.DepartmentID
		}
	}
	userReqs := make([]registerUserReq, len(users))
	for i, user := range users {
		if user.Ignore {
			continue
		}
		if user.Department != "" {
			// 开发/后端/Go:职位1;销售/后端/Go:职位2
			departmentIDSet := make(map[string]struct{})
			for _, d := range strings.Split(user.Department, ";") {
				var (
					names    []string
					position string
				)
				ds := strings.Split(d, ":")
				switch len(ds) {
				case 2:
					position = ds[1]
					fallthrough
				case 1:
					names = strings.Split(ds[0], departmentNameSeparator)
				default:
					return errs.ErrArgs.Wrap("department format error " + user.Department)
				}
				for _, name := range names {
					if name == "" {
						return errs.ErrArgs.Wrap("department name is empty " + user.Department)
					}
				}
				if len(names) > 0 {
					resp, err := o.organizationClient.GetDepartmentByName(c, &organization.GetDepartmentByNameReq{Names: names})
					if err != nil {
						return err
					}
					if len(resp.Departments) < len(names) {
						return errs.ErrArgs.Wrap("department not exist " + user.Department)
					}
					departmentID := resp.Departments[len(resp.Departments)-1].DepartmentID
					if _, ok := departmentIDSet[departmentID]; ok {
						return errs.ErrArgs.Wrap("department repeat " + user.Department)
					}
					userReqs[i].Departments = append(userReqs[i].Departments, registerUserDepartment{
						DepartmentID:    departmentID,
						Position:        position,
						EntryTime:       time.Now().UnixMilli(),
						TerminationTime: 0,
					})
				}
			}
		}
		parseBirth := func(s string) time.Time {
			if s == "" {
				return time.Now()
			}
			var separator byte
			for _, b := range []byte(s) {
				if b < '0' || b > '9' {
					separator = b
				}
			}
			arr := strings.Split(s, string([]byte{separator}))
			if len(arr) != 3 {
				return time.Now()
			}
			year, _ := strconv.Atoi(arr[0])
			month, _ := strconv.Atoi(arr[1])
			day, _ := strconv.Atoi(arr[2])
			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
			if t.Before(time.Date(1900, 0, 0, 0, 0, 0, 0, time.Local)) {
				return time.Now()
			}
			return t
		}
		gender, _ := strconv.Atoi(user.Gender)
		userReqs[i].User = chat.RegisterUserInfo{
			UserID:      user.UserID,
			Nickname:    user.Nickname,
			FaceURL:     user.FaceURL,
			Birth:       parseBirth(user.Birth).UnixMilli(),
			Gender:      int32(gender),
			AreaCode:    user.AreaCode,
			PhoneNumber: user.PhoneNumber,
			Email:       user.Email,
			Account:     user.Account,
			Password:    user.Password,

			EnglishName: user.EnglishName,
			Station:     user.Station,
			Telephone:   user.Telephone,
		}

	}
	//创建用户
	for i, user := range users {
		if user.Ignore {
			continue
		}
		_, err := o.registerUser(c, &userReqs[i], ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Org) getClientIP(c *gin.Context) (string, error) {
	if config.Config.ProxyHeader == "" {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		return ip, err
	}
	ip := c.Request.Header.Get(config.Config.ProxyHeader)
	if ip == "" {
		return "", errs.ErrInternalServer.Wrap()
	}
	if ip := net.ParseIP(ip); ip == nil {
		return "", errs.ErrInternalServer.Wrap(fmt.Sprintf("parse proxy ip header %s failed", ip))
	}
	return ip, nil
}
