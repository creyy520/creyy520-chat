package chat

//func (o *chatSvr) CreateUser(ctx context.Context, req *chat.CreateUserReq) (*chat.CreateUserResp, error) {
//	if _, err := mctx.CheckAdmin(ctx); err != nil {
//		return nil, err
//	}
//	if req.User.UserID == "" {
//		for i := 0; i < 10; i++ {
//			userID := o.genUserID()
//			if _, err := o.Database.GetUser(ctx, userID); err == nil {
//				continue
//			} else if dbutil.IsGormNotFound(err) {
//				req.User.UserID = userID
//			} else {
//				return nil, err
//			}
//		}
//		if req.User.UserID == "" {
//			return nil, errs.ErrInternalServer.Wrap("gen user id failed")
//		}
//	} else {
//		if _, err := o.Database.GetUser(ctx, req.User.UserID); err == nil {
//			return nil, errs.ErrDuplicateKey.Wrap("user id already exists")
//		} else if !dbutil.IsGormNotFound(err) {
//			return nil, err
//		}
//	}
//	register := &chat2.Register{
//		UserID:      req.User.UserID,
//		DeviceID:    "",
//		IP:          "",
//		Platform:    "",
//		AccountType: "",
//		Mode:        constant.UserMode,
//		CreateTime:  time.Now(),
//	}
//	account := &chat2.Account{
//		UserID:         req.User.UserID,
//		Password:       req.User.Password,
//		OperatorUserID: mcontext.GetOpUserID(ctx),
//		ChangeTime:     register.CreateTime,
//		CreateTime:     register.CreateTime,
//	}
//	attribute := &chat2.Attribute{
//		UserID:         req.User.UserID,
//		Account:        req.User.Account,
//		PhoneNumber:    req.User.PhoneNumber,
//		AreaCode:       req.User.AreaCode,
//		Email:          req.User.Email,
//		Nickname:       req.User.Nickname,
//		FaceURL:        req.User.FaceURL,
//		Gender:         req.User.Gender,
//		BirthTime:      time.UnixMilli(req.User.Birth),
//		ChangeTime:     register.CreateTime,
//		CreateTime:     register.CreateTime,
//		AllowVibration: constant.DefaultAllowVibration,
//		AllowBeep:      constant.DefaultAllowBeep,
//		AllowAddFriend: constant.DefaultAllowAddFriend,
//
//		EnglishName: req.User.EnglishName,
//		Station:     req.User.Station,
//		Telephone:   req.User.Telephone,
//		Status:      req.User.Status,
//	}
//	openIMRegister := func() error {
//		return o.OpenIM.UserRegister(ctx, &sdkws.UserInfo{
//			UserID:     req.User.UserID,
//			Nickname:   req.User.Nickname,
//			FaceURL:    req.User.FaceURL,
//			CreateTime: register.CreateTime.UnixMilli(),
//		})
//	}
//	if err := o.Database.RegisterUser(ctx, register, account, attribute, openIMRegister); err != nil {
//		return nil, err
//	}
//	if userIDs, err := o.Admin.GetDefaultFriendUserID(ctx); err != nil {
//		log.ZError(ctx, "GetDefaultFriendUserID Failed", err, "userID", req.User.UserID)
//	} else if len(userIDs) > 0 {
//		if err := o.OpenIM.AddDefaultFriend(ctx, req.User.UserID, userIDs); err != nil {
//			log.ZError(ctx, "AddDefaultFriend Failed", err, "userID", req.User.UserID, "userIDs", userIDs)
//		}
//	}
//	if groupIDs, err := o.Admin.GetDefaultGroupID(ctx); err != nil {
//		log.ZError(ctx, "GetDefaultGroupID Failed", err, "userID", req.User.UserID)
//	} else if len(groupIDs) > 0 {
//		for _, groupID := range groupIDs {
//			if err := o.OpenIM.AddDefaultGroup(ctx, req.User.UserID, groupID); err != nil {
//				log.ZError(ctx, "GetDefaultGroupID Failed", err, "userID", req.User.UserID, "groupID", groupID)
//			}
//		}
//	}
//	return &chat.CreateUserResp{
//		UserID: req.User.UserID,
//	}, nil
//}
