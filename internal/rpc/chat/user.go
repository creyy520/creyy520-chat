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

package chat

import (
	"context"
	"github.com/OpenIMSDK/chat/pkg/common/db/dbutil"
	chat2 "github.com/OpenIMSDK/chat/pkg/common/db/table/chat"
	constant2 "github.com/OpenIMSDK/protocol/constant"
	"github.com/OpenIMSDK/tools/mcontext"
	"time"

	"github.com/OpenIMSDK/chat/pkg/common/constant"
	"github.com/OpenIMSDK/chat/pkg/common/mctx"
	"github.com/OpenIMSDK/chat/pkg/eerrs"
	"github.com/OpenIMSDK/chat/pkg/proto/chat"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/log"
)

func (o *chatSvr) UpdateUserInfo(ctx context.Context, req *chat.UpdateUserInfoReq) (*chat.UpdateUserInfoResp, error) {
	defer log.ZDebug(ctx, "return")
	resp := &chat.UpdateUserInfoResp{}
	opUserID, userType, err := mctx.Check(ctx)
	if err != nil {
		return nil, err
	}
	if req.UserID == "" {
		return nil, errs.ErrArgs.Wrap("user id is empty")
	}
	switch userType {
	case constant.NormalUser:
		//if req.UserID == "" {
		//	req.UserID = opUserID
		//}
		if req.UserID != opUserID {
			return nil, errs.ErrNoPermission.Wrap("only admin can update other user info")
		}
		if req.AreaCode != nil {
			return nil, errs.ErrNoPermission.Wrap("areaCode can not be updated")
		}
		if req.PhoneNumber != nil {
			return nil, errs.ErrNoPermission.Wrap("phoneNumber can not be updated")
		}
		if req.Account != nil {
			return nil, errs.ErrNoPermission.Wrap("account can not be updated")
		}
		if req.Level != nil {
			return nil, errs.ErrNoPermission.Wrap("level can not be updated")
		}
	case constant.AdminUser:
	default:
		return nil, errs.ErrNoPermission.Wrap("user type error")
	}
	update, err := ToDBAttributeUpdate(req)
	if err != nil {
		return nil, err
	}
	attribute, err := o.Database.TakeAttributeByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if req.Account != nil && req.Account.Value != attribute.Account {
		_, err := o.Database.TakeAttributeByAccount(ctx, req.Account.Value)
		if err == nil {
			return nil, eerrs.ErrAccountAlreadyRegister.Wrap()
		} else if !o.Database.IsNotFound(err) {
			return nil, err
		}
	}
	if req.AreaCode != nil || req.PhoneNumber != nil {
		areaCode := attribute.AreaCode
		phoneNumber := attribute.PhoneNumber
		if req.AreaCode != nil {
			areaCode = req.AreaCode.Value
		}
		if req.PhoneNumber != nil {
			phoneNumber = req.PhoneNumber.Value
		}
		if attribute.AreaCode != areaCode || attribute.PhoneNumber != phoneNumber {
			_, err := o.Database.TakeAttributeByPhone(ctx, areaCode, phoneNumber)
			if err == nil {
				return nil, eerrs.ErrAccountAlreadyRegister.Wrap()
			} else if !o.Database.IsNotFound(err) {
				return nil, err
			}
		}
	}
	if err := o.Database.UpdateUseInfo(ctx, req.UserID, update); err != nil {
		return nil, err
	}
	return resp, nil
}

func (o *chatSvr) FindUserPublicInfo(ctx context.Context, req *chat.FindUserPublicInfoReq) (*chat.FindUserPublicInfoResp, error) {
	defer log.ZDebug(ctx, "return")
	if len(req.UserIDs) == 0 {
		return nil, errs.ErrArgs.Wrap("UserIDs is empty")
	}
	attributes, err := o.Database.FindAttribute(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	return &chat.FindUserPublicInfoResp{
		Users: DbToPbAttributes(attributes),
	}, nil
}

func (o *chatSvr) AddUserAccount(ctx context.Context, req *chat.AddUserAccountReq) (*chat.AddUserAccountResp, error) {
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}

	if err := o.checkTheUniqueness(ctx, req); err != nil {
		return nil, err
	}

	if req.User.UserID == "" {
		for i := 0; i < 20; i++ {
			userID := o.genUserID()
			_, err := o.Database.GetUser(ctx, userID)
			if err == nil {
				continue
			} else if dbutil.IsGormNotFound(err) {
				req.User.UserID = userID
				break
			} else {
				return nil, err
			}
		}
		if req.User.UserID == "" {
			return nil, errs.ErrInternalServer.Wrap("gen user id failed")
		}
	}

	register := &chat2.Register{
		UserID:      req.User.UserID,
		DeviceID:    req.DeviceID,
		IP:          req.Ip,
		Platform:    constant2.PlatformID2Name[int(req.Platform)],
		AccountType: "",
		Mode:        constant.UserMode,
		CreateTime:  time.Now(),
	}
	account := &chat2.Account{
		UserID:         req.User.UserID,
		Password:       req.User.Password,
		OperatorUserID: mcontext.GetOpUserID(ctx),
		ChangeTime:     register.CreateTime,
		CreateTime:     register.CreateTime,
	}
	attribute := &chat2.Attribute{
		UserID:         req.User.UserID,
		Account:        req.User.Account,
		PhoneNumber:    req.User.PhoneNumber,
		AreaCode:       req.User.AreaCode,
		Email:          req.User.Email,
		Nickname:       req.User.Nickname,
		FaceURL:        req.User.FaceURL,
		Gender:         req.User.Gender,
		BirthTime:      time.UnixMilli(req.User.Birth),
		ChangeTime:     register.CreateTime,
		CreateTime:     register.CreateTime,
		AllowVibration: constant.DefaultAllowVibration,
		AllowBeep:      constant.DefaultAllowBeep,
		AllowAddFriend: constant.DefaultAllowAddFriend,
	}

	if err := o.Database.RegisterUser(ctx, register, account, attribute); err != nil {
		return nil, err
	}

	return &chat.AddUserAccountResp{}, nil
}

func (o *chatSvr) SearchUserPublicInfo(ctx context.Context, req *chat.SearchUserPublicInfoReq) (*chat.SearchUserPublicInfoResp, error) {
	defer log.ZDebug(ctx, "return")
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}
	total, list, err := o.Database.Search(ctx, constant.FinDAllUser, req.Keyword, req.Genders, req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	return &chat.SearchUserPublicInfoResp{
		Total: total,
		Users: DbToPbAttributes(list),
	}, nil
}

func (o *chatSvr) SearchUserID(ctx context.Context, req *chat.SearchUserIDReq) (*chat.SearchUserIDResp, error) {
	defer log.ZDebug(ctx, "return")
	if req.Pagination == nil {
		return nil, errs.ErrArgs.Wrap("pagination is nil")
	}
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}
	total, userIDs, err := o.Database.SearchID(ctx, req.Keyword, req.OrUserIDs, req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	return &chat.SearchUserIDResp{
		Total:   total,
		UserIDs: userIDs,
	}, nil
}

func (o *chatSvr) FindUserFullInfo(ctx context.Context, req *chat.FindUserFullInfoReq) (*chat.FindUserFullInfoResp, error) {
	defer log.ZDebug(ctx, "return")
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}
	if len(req.UserIDs) == 0 {
		return nil, errs.ErrArgs.Wrap("UserIDs is empty")
	}
	attributes, err := o.Database.FindAttribute(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	return &chat.FindUserFullInfoResp{Users: DbToPbUserFullInfos(attributes)}, nil
}

func (o *chatSvr) SearchUserFullInfo(ctx context.Context, req *chat.SearchUserFullInfoReq) (*chat.SearchUserFullInfoResp, error) {
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}
	total, list, err := o.Database.Search(ctx, req.Normal, req.Keyword, req.Genders, req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	return &chat.SearchUserFullInfoResp{
		Total: total,
		Users: DbToPbUserFullInfos(list),
	}, nil
}

func (o *chatSvr) FindUserAccount(ctx context.Context, req *chat.FindUserAccountReq) (*chat.FindUserAccountResp, error) {
	defer log.ZDebug(ctx, "return")
	if len(req.UserIDs) == 0 {
		return nil, errs.ErrArgs.Wrap("user id list must be set")
	}
	if _, _, err := mctx.CheckAdminOrUser(ctx); err != nil {
		return nil, err
	}
	attributes, err := o.Database.FindAttribute(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	userAccountMap := make(map[string]string)
	for _, attribute := range attributes {
		userAccountMap[attribute.UserID] = attribute.Account
	}
	return &chat.FindUserAccountResp{UserAccountMap: userAccountMap}, nil
}

func (o *chatSvr) FindAccountUser(ctx context.Context, req *chat.FindAccountUserReq) (*chat.FindAccountUserResp, error) {
	defer log.ZDebug(ctx, "return")
	if len(req.Accounts) == 0 {
		return nil, errs.ErrArgs.Wrap("account list must be set")
	}
	if _, _, err := mctx.CheckAdminOrUser(ctx); err != nil {
		return nil, err
	}
	attributes, err := o.Database.FindAttribute(ctx, req.Accounts)
	if err != nil {
		return nil, err
	}
	accountUserMap := make(map[string]string)
	for _, attribute := range attributes {
		accountUserMap[attribute.Account] = attribute.UserID
	}
	return &chat.FindAccountUserResp{AccountUserMap: accountUserMap}, nil
}

func (o *chatSvr) SearchUserInfo(ctx context.Context, req *chat.SearchUserInfoReq) (*chat.SearchUserInfoResp, error) {
	if _, _, err := mctx.Check(ctx); err != nil {
		return nil, err
	}
	total, list, err := o.Database.SearchUser(ctx, req.Keyword, req.UserIDs, req.Genders, req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	return &chat.SearchUserInfoResp{
		Total: total,
		Users: DbToPbUserFullInfos(list),
	}, nil
}

func (o *chatSvr) checkTheUniqueness(ctx context.Context, req *chat.AddUserAccountReq) error {
	if req.User.PhoneNumber != "" {
		_, err := o.Database.TakeAttributeByPhone(ctx, req.User.AreaCode, req.User.PhoneNumber)
		if err == nil {
			return eerrs.ErrPhoneAlreadyRegister.Wrap()
		} else if !o.Database.IsNotFound(err) {
			return err
		}
	} else {
		_, err := o.Database.TakeAttributeByEmail(ctx, req.User.Email)
		if err == nil {
			return eerrs.ErrEmailAlreadyRegister.Wrap()
		} else if !o.Database.IsNotFound(err) {
			return err
		}
	}
	return nil
}
