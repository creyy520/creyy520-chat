package organization

import (
	table "github.com/OpenIMSDK/chat/pkg/common/db/table/organization"
	"github.com/OpenIMSDK/chat/pkg/proto/organization"
	"github.com/OpenIMSDK/tools/errs"
	"gorm.io/gorm"
	"math"
)

func DepartmentDb2Pb(d *table.Department) *organization.Department {
	if d == nil {
		return nil
	}
	return &organization.Department{
		DepartmentID:       d.DepartmentID,
		FaceURL:            d.FaceURL,
		Name:               d.Name,
		ParentDepartmentID: d.ParentDepartmentID,
		Order:              d.Order,
		CreateTime:         d.CreateTime.UnixMilli(),
	}
}

func DepartmentNumDb2Pb(d *table.Department, num uint32) *organization.DepartmentNum {
	if d == nil {
		return nil
	}
	return &organization.DepartmentNum{
		DepartmentID:       d.DepartmentID,
		FaceURL:            d.FaceURL,
		Name:               d.Name,
		ParentDepartmentID: d.ParentDepartmentID,
		Order:              d.Order,
		MemberNum:          num,
		CreateTime:         d.CreateTime.UnixMilli(),
	}
}

func DepartmentNumPb2Pb(d *organization.Department, num uint32) *organization.DepartmentNum {
	if d == nil {
		return nil
	}
	return &organization.DepartmentNum{
		DepartmentID:       d.DepartmentID,
		FaceURL:            d.FaceURL,
		Name:               d.Name,
		ParentDepartmentID: d.ParentDepartmentID,
		Order:              d.Order,
		MemberNum:          num,
		CreateTime:         d.CreateTime,
	}
}

func Organization2DepartmentNum(org *table.Organization, num uint32) *organization.DepartmentNum {
	return &organization.DepartmentNum{
		DepartmentID: "",
		FaceURL:      org.LogoURL,
		Name:         org.Name,
		Order:        math.MinInt32,
		CreateTime:   org.CreateTime.UnixMilli(),
		MemberNum:    num,
	}
}

func Organization2Department(org *table.Organization) *organization.Department {
	return &organization.Department{
		DepartmentID: "",
		FaceURL:      org.LogoURL,
		Name:         org.Name,
		Order:        math.MinInt32,
		CreateTime:   org.CreateTime.UnixMilli(),
	}
}

func DepartmentMemberDb2Pb(member *table.DepartmentMember) *organization.DepartmentMember {
	var terminationTime int64
	if member.TerminationTime != nil {
		terminationTime = member.TerminationTime.UnixMilli()
	}
	return &organization.DepartmentMember{
		UserID:          member.UserID,
		DepartmentID:    member.DepartmentID,
		Position:        member.Position,
		Station:         member.Station,
		Order:           member.Order,
		EntryTime:       member.EntryTime.UnixMilli(),
		TerminationTime: terminationTime,
		CreateTime:      member.CreateTime.UnixMilli(),
	}
}

func MemberDepartmentDb2Pb(member *table.DepartmentMember, department *organization.DepartmentNum) *organization.MemberDepartment {
	var terminationTime int64
	if member.TerminationTime != nil {
		terminationTime = member.TerminationTime.UnixMilli()
	}
	return &organization.MemberDepartment{
		UserID:          member.UserID,
		DepartmentID:    member.DepartmentID,
		Position:        member.Position,
		Station:         member.Station,
		Order:           member.Order,
		EntryTime:       member.EntryTime.UnixMilli(),
		TerminationTime: terminationTime,
		CreateTime:      member.CreateTime.UnixMilli(),
		Department:      department,
	}
}

func IsNotFound(err error) bool {
	return errs.Unwrap(err) == gorm.ErrRecordNotFound
}
