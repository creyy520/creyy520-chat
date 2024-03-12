// Copyright © 2023 OpenIM open source community. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"github.com/OpenIMSDK/chat/example/callback"

	"github.com/OpenIMSDK/chat/pkg/common/config"
	"github.com/OpenIMSDK/tools/discoveryregistry"
	"github.com/gin-gonic/gin"
)

func NewChatRoute(router gin.IRouter, discov discoveryregistry.SvcDiscoveryRegistry) {
	chatConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImChatName)
	if err != nil {
		panic(err)
	}
	adminConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImAdminName)
	if err != nil {
		panic(err)
	}
	officeConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImOfficeName)
	if err != nil {
		panic(err)
	}
	orgConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImOrganizationName)
	if err != nil {
		panic(err)
	}
	mw := NewMW(adminConn)
	chat := NewChat(chatConn, adminConn)
	org := NewOrg(chatConn, adminConn, orgConn)

	account := router.Group("/account")
	//account.POST("/code/send", chat.SendVerifyCode)                      // Send verification code
	//account.POST("/code/verify", chat.VerifyCode)                        // Verify the verification code
	//account.POST("/register", mw.CheckAdminOrNil, chat.RegisterUser)     // Register
	account.POST("/login", chat.Login)                                   // Login
	account.POST("/password/reset", chat.ResetPassword)                  // Forgot password
	account.POST("/password/change", mw.CheckToken, chat.ChangePassword) // Change password

	user := router.Group("/user", mw.CheckToken)
	user.POST("/update", chat.UpdateUserInfo)             // Edit personal information
	user.POST("/find/public", org.FindUserPublicInfo)     // Get user's public information
	user.POST("/find/full", org.FindUserFullInfo)         // Get all information of the user
	user.POST("/search/full", org.SearchUserFullInfo)     // Search user's public information
	user.POST("/search/public", org.SearchUserPublicInfo) // Search all information of the user

	router.POST("/friend/search", mw.CheckToken, chat.SearchFriend)

	router.Group("/applet").POST("/find", mw.CheckToken, chat.FindApplet) // Applet list

	router.Group("/client_config").POST("/get", chat.GetClientConfig) // Get client initialization configuration

	router.Group("/callback").POST("/open_im", chat.OpenIMCallback) // Callback

	router.Group("/callbackExample").POST("/callbackAfterSendSingleMsgCommand", callback.CallbackExample)

	logs := router.Group("/logs", mw.CheckToken)
	logs.POST("/upload", chat.UploadLogs)
	logs.POST("/delete", chat.DeleteLogs)

	office := NewOffice(officeConn)
	officeRouter := router.Group("/office", mw.CheckToken)
	officeRouter.POST("/tag/add", office.CreateTag)
	officeRouter.POST("/tag/del", office.DeleteTag)
	officeRouter.POST("/tag/set", office.SetTag)
	officeRouter.POST("/tag/get", office.GetUserTagByID)
	officeRouter.POST("/tag/find/user", office.GetUserTags)
	officeRouter.POST("/tag/send", office.SendMsg2Tag)
	officeRouter.POST("/tag/send/log", office.GetTagSendLogs)
	officeRouter.POST("/tag/send/log/del", office.DelTagSendLogs)

	officeRouter.POST("/work_moment/add", office.CreateOneWorkMoment)
	officeRouter.POST("/work_moment/del", office.DeleteOneWorkMoment)
	officeRouter.POST("/work_moment/like", office.LikeOneWorkMoment)
	officeRouter.POST("/work_moment/comment/add", office.CommentOneWorkMoment)
	officeRouter.POST("/work_moment/comment/del", office.DeleteComment)
	officeRouter.POST("/work_moment/get", office.GetWorkMomentByID)
	officeRouter.POST("/work_moment/find/send", office.GetUserSendWorkMoments)
	officeRouter.POST("/work_moment/find/recv", office.GetUserRecvWorkMoments)

	officeRouter.POST("/work_moment/logs", office.GetUnreadWorkMoments)
	officeRouter.POST("/work_moment/unread/count", office.GetUnreadWorkMomentsCount)
	officeRouter.POST("/work_moment/unread/clear", office.ReadWorkMoments)

	organizationGroup := router.Group("/organization", mw.CheckToken)
	organizationGroup.POST("/info", org.GetOrganization)                     // 获取公司信息
	organizationGroup.POST("/department/all", org.GetOrganizationDepartment) // 获取组织部门
	organizationGroup.POST("/department/find", org.GetDepartment)            // 查询部门
	organizationGroup.POST("/user/department", org.GetUserInDepartment)      // 获取用户所在部门
	organizationGroup.POST("/department/child", org.GetSubDepartment)        // 获取部门的人和同级部门

	/*
		对应关系
		/organization/get_department -> /organization/department/find
		/organization/get_user_in_department -> /organization/user/department
		/organization/get_sub_department -> /organization/department/child
		/organization/get_search_department_user -> /user/search/full

		接口不变,多字段
		/user/find/public
		/user/find/full
		/user/search/full
		/user/search/public

		移除接口
		/account/code/send
		/account/code/verify
		/account/register

	*/

}

func NewAdminRoute(router gin.IRouter, discov discoveryregistry.SvcDiscoveryRegistry) {
	adminConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImAdminName)
	if err != nil {
		panic(err)
	}
	chatConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImChatName)
	if err != nil {
		panic(err)
	}
	orgConn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImOrganizationName)
	if err != nil {
		panic(err)
	}
	rtcConn, err := discov.GetConn(context.Background(), *config.Config.RpcRegisterName.OpenImRtcName)
	if err != nil {
		panic(err)
	}
	mw := NewMW(adminConn)
	org := NewOrg(chatConn, adminConn, orgConn)

	admin := NewAdmin(chatConn, adminConn, orgConn, rtcConn)
	adminRouterGroup := router.Group("/account")
	adminRouterGroup.POST("/login", admin.AdminLogin)                                   // Login
	adminRouterGroup.POST("/update", mw.CheckAdmin, admin.AdminUpdateInfo)              // Modify information
	adminRouterGroup.POST("/info", mw.CheckAdmin, admin.AdminInfo)                      // Get information
	adminRouterGroup.POST("/change_password", mw.CheckAdmin, admin.ChangeAdminPassword) // Change admin account's password
	adminRouterGroup.POST("/add_admin", mw.CheckAdmin, admin.AddAdminAccount)           // Add admin account
	adminRouterGroup.POST("/add_user", mw.CheckAdmin, admin.AddUserAccount)             // Add user account
	adminRouterGroup.POST("/del_admin", mw.CheckAdmin, admin.DelAdminAccount)           // Delete admin
	adminRouterGroup.POST("/search", mw.CheckAdmin, admin.SearchAdminAccount)           // Get admin list
	//account.POST("/add_notification_account")

	importGroup := router.Group("/user/import")
	importGroup.POST("/json", mw.CheckAdminOrNil, admin.ImportUserByJson)
	importGroup.POST("/xlsx", mw.CheckAdminOrNil, admin.ImportUserByXlsx)
	importGroup.GET("/xlsx", admin.BatchImportTemplate)

	defaultRouter := router.Group("/default", mw.CheckAdmin)
	defaultUserRouter := defaultRouter.Group("/user")
	defaultUserRouter.POST("/add", admin.AddDefaultFriend)       // Add default friend at registration
	defaultUserRouter.POST("/del", admin.DelDefaultFriend)       // Delete default friend at registration
	defaultUserRouter.POST("/find", admin.FindDefaultFriend)     // Default friend list
	defaultUserRouter.POST("/search", admin.SearchDefaultFriend) // Search default friend list at registration
	defaultGroupRouter := defaultRouter.Group("/group")
	defaultGroupRouter.POST("/add", admin.AddDefaultGroup)       // Add default group at registration
	defaultGroupRouter.POST("/del", admin.DelDefaultGroup)       // Delete default group at registration
	defaultGroupRouter.POST("/find", admin.FindDefaultGroup)     // Get default group list at registration
	defaultGroupRouter.POST("/search", admin.SearchDefaultGroup) // Search default group list at registration

	invitationCodeRouter := router.Group("/invitation_code", mw.CheckAdmin)
	invitationCodeRouter.POST("/add", admin.AddInvitationCode)       // Add invitation code
	invitationCodeRouter.POST("/gen", admin.GenInvitationCode)       // Generate invitation code
	invitationCodeRouter.POST("/del", admin.DelInvitationCode)       // Delete invitation code
	invitationCodeRouter.POST("/search", admin.SearchInvitationCode) // Search invitation code

	forbiddenRouter := router.Group("/forbidden", mw.CheckAdmin)
	ipForbiddenRouter := forbiddenRouter.Group("/ip")
	ipForbiddenRouter.POST("/add", admin.AddIPForbidden)       // Add forbidden IP for registration/login
	ipForbiddenRouter.POST("/del", admin.DelIPForbidden)       // Delete forbidden IP for registration/login
	ipForbiddenRouter.POST("/search", admin.SearchIPForbidden) // Search forbidden IPs for registration/login
	userForbiddenRouter := forbiddenRouter.Group("/user")
	userForbiddenRouter.POST("/add", admin.AddUserIPLimitLogin)       // Add limit for user login on specific IP
	userForbiddenRouter.POST("/del", admin.DelUserIPLimitLogin)       // Delete user limit on specific IP for login
	userForbiddenRouter.POST("/search", admin.SearchUserIPLimitLogin) // Search limit for user login on specific IP

	appletRouterGroup := router.Group("/applet", mw.CheckAdmin)
	appletRouterGroup.POST("/add", admin.AddApplet)       // Add applet
	appletRouterGroup.POST("/del", admin.DelApplet)       // Delete applet
	appletRouterGroup.POST("/update", admin.UpdateApplet) // Modify applet
	appletRouterGroup.POST("/search", admin.SearchApplet) // Search applet

	blockRouter := router.Group("/block", mw.CheckAdmin)
	blockRouter.POST("/add", admin.BlockUser)          // Block user
	blockRouter.POST("/del", admin.UnblockUser)        // Unblock user
	blockRouter.POST("/search", admin.SearchBlockUser) // Search blocked users

	userRouter := router.Group("/user", mw.CheckAdmin)
	userRouter.POST("/password/reset", admin.ResetUserPassword) // Reset user password
	userRouter.POST("/add", org.RegisterUser)                   // 添加新用户

	initGroup := router.Group("/client_config", mw.CheckAdmin)
	initGroup.POST("/get", admin.GetClientConfig) // Get client initialization configuration
	initGroup.POST("/set", admin.SetClientConfig) // Set client initialization configuration
	initGroup.POST("/del", admin.DelClientConfig) // Delete client initialization configuration

	statistic := router.Group("/statistic", mw.CheckAdmin)
	statistic.POST("/new_user_count", admin.NewUserCount)
	statistic.POST("/login_user_count", admin.LoginUserCount)

	rtc := router.Group("/rtc", mw.CheckAdmin)
	rtc.POST("/get_signal_invitation_records", admin.GetSignalInvitationRecords)
	rtc.POST("/delete_signal_records", admin.DeleteSignalRecords)
	rtc.POST("/get_meeting_records", admin.GetMeetingRecords)
	rtc.POST("/delete_meeting_records", admin.DeleteMeetingRecords)

	logs := router.Group("/logs", mw.CheckAdmin)
	logs.POST("/search", admin.SearchLogs)

	organizationGroup := router.Group("/organization", mw.CheckAdmin)
	router.GET("/organization/import/template", org.BatchImportTemplate) // 批量导入模板
	organizationGroup.POST("/import", org.BatchImport)                   // 批量导入
	//部门  增删改查
	organizationGroup.POST("/department/add", org.CreateDepartment)          // 创建部门
	organizationGroup.POST("/department/update", org.UpdateDepartment)       // 修改部门
	organizationGroup.POST("/department/del", org.DeleteDepartment)          // 删除部门
	organizationGroup.POST("/department/find", org.GetDepartment)            // 获取部门
	organizationGroup.POST("/department/all", org.GetOrganizationDepartment) // 获取部门
	organizationGroup.POST("/department/expand", org.GetSubDepartment)       // 获取部门的人和同级部门
	organizationGroup.POST("/department/user", org.GetUserInDepartment)      // 用户所在部门
	organizationGroup.POST("/department/sort", org.SortDepartmentList)       // 部门排序

	organizationGroup.POST("/department/member/add", org.CreateDepartmentMember)    // 修改用户部门
	organizationGroup.POST("/department/member/update", org.UpdateUserInDepartment) // 修改用户部门
	organizationGroup.POST("/department/member/move", org.MoveUserDepartment)       // 移动用户部门
	organizationGroup.POST("/department/member/del", org.DeleteUserInDepartment)    // 删除部门成员
	organizationGroup.POST("/department/member/sort", org.SortOrganizationUserList) // 部门成员排序

	organizationGroup.POST("/user/add", org.RegisterUser)              // 添加用户
	organizationGroup.POST("/user/update", org.UpdateOrganizationUser) // 修改用户信息
	organizationGroup.POST("/user/search", org.GetSearchUserList)      // 搜索用户

	organizationGroup.POST("/get", org.GetOrganization) // 获取公司信息
	organizationGroup.POST("/set", org.SetOrganization) // 设置公司信息

	logs.POST("/delete", admin.DeleteLogs)
}
