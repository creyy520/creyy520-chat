package database

import (
	"context"
	"github.com/OpenIMSDK/chat/pkg/common/db/model/organization"
	table "github.com/OpenIMSDK/chat/pkg/common/db/table/organization"
	"github.com/OpenIMSDK/tools/tx"
	"gorm.io/gorm"
)

type OrganizationDatabaseInterface interface {
	//department
	GetDepartmentByID(ctx context.Context, departmentID string) (*table.Department, error)
	CreateDepartment(ctx context.Context, department *table.Department) error
	UpdateDepartment(ctx context.Context, departmentID string, update map[string]any) error
	GetParentDepartment(ctx context.Context, parentID string) ([]*table.Department, error)
	GetDepartment(ctx context.Context, departmentID string) (*table.Department, error)
	GetDepartmentList(ctx context.Context, departmentIDList []string) ([]*table.Department, error)
	DeleteDepartment(ctx context.Context, departmentIDList []string) error
	UpdateParentID(ctx context.Context, oldParentID, newParentID string) error
	GetMaxDepartmentOrder(ctx context.Context, parentID string) (int32, error)
	GetDepartmentByName(ctx context.Context, name, parentID string) (*table.Department, error)
	SearchDepartment(ctx context.Context, keyword string) ([]string, error)
	IncrDepartmentOrder(ctx context.Context, parentDepartmentID string, order int32) error
	//departmentMember
	FindDepartmentMember(ctx context.Context, departmentIDList []string) ([]*table.DepartmentMember, error)
	GetDepartmentMemberInUserID(ctx context.Context, userIDs []string) ([]*table.DepartmentMember, error)
	CreateDepartmentMember(ctx context.Context, DepartmentMember *table.DepartmentMember) error
	CreateDepartmentMembers(ctx context.Context, departmentMembers []*table.DepartmentMember) error
	DeleteDepartmentIDList(ctx context.Context, departmentIDList []string) error
	DeleteDepartmentMemberByKey(ctx context.Context, userID string, departmentID string) error
	UpdateDepartmentMember(ctx context.Context, departmentID string, userID string, update map[string]any) error
	FindDepartmentMemberByUserID(ctx context.Context, userIDList []string) ([]*table.DepartmentMember, error)
	GetDepartmentMemberByDepartmentID(ctx context.Context, departmentID string) ([]*table.DepartmentMember, error)
	GetDepartmentMemberByKey(ctx context.Context, userID, departmentID string) (*table.DepartmentMember, error)
	MoveDepartmentMember(ctx context.Context, userID string, oldDepartmentID string, newDepartmentID string) error
	SearchDepartmentMember(ctx context.Context, keyword string, departmentIDs []string) ([]string, error)
	GetMaxDepartmentMemberOrder(ctx context.Context, departmentID string) (int32, error)
	IncrDepartmentMemberOrder(ctx context.Context, parentDepartmentID string, order int32) error
	GetMemberNum(ctx context.Context, departmentIDs []string) (int64, error)
	//organizaiton
	SetOrganization(ctx context.Context, update map[string]any) error
	GetOrganization(ctx context.Context) (*table.Organization, error)
}

func NewOrganizationDatabase(db *gorm.DB) OrganizationDatabaseInterface {
	return &OrganizationDatabase{
		tx:               tx.NewGorm(db),
		Department:       organization.NewDepartment(db),
		DepartmentMember: organization.NewDepartmentMember(db),
		Organization:     organization.NewOrganization(db),
	}
}

type OrganizationDatabase struct {
	tx               tx.Tx
	Department       table.DepartmentInterface
	DepartmentMember table.DepartmentMemberInterface
	Organization     table.OrganizationInterface
}

func (o *OrganizationDatabase) GetMemberNum(ctx context.Context, departmentIDs []string) (int64, error) {
	return o.DepartmentMember.GetNum(ctx, departmentIDs)
}

func (o *OrganizationDatabase) IncrDepartmentMemberOrder(ctx context.Context, parentDepartmentID string, order int32) error {
	return o.DepartmentMember.IncrOrder(ctx, parentDepartmentID, order)
}

func (o *OrganizationDatabase) GetMaxDepartmentMemberOrder(ctx context.Context, departmentID string) (int32, error) {
	return o.DepartmentMember.GetMaxOrder(ctx, departmentID)
}

func (o *OrganizationDatabase) IncrDepartmentOrder(ctx context.Context, parentDepartmentID string, order int32) error {
	return o.Department.IncrOrder(ctx, parentDepartmentID, order)
}

func (o *OrganizationDatabase) GetDepartmentMemberByKey(ctx context.Context, userID, departmentID string) (*table.DepartmentMember, error) {
	return o.DepartmentMember.GetByKey(ctx, userID, departmentID)
}

func (o *OrganizationDatabase) GetDepartmentByName(ctx context.Context, name, parentID string) (*table.Department, error) {
	return o.Department.GetByName(ctx, name, parentID)
}

func (o *OrganizationDatabase) GetMaxDepartmentOrder(ctx context.Context, parentID string) (int32, error) {
	return o.Department.GetMaxOrder(ctx, parentID)
}

func (o *OrganizationDatabase) SearchDepartment(ctx context.Context, keyword string) ([]string, error) {
	return o.Department.Search(ctx, keyword)
}

func (o *OrganizationDatabase) SearchDepartmentMember(ctx context.Context, keyword string, departmentIDs []string) ([]string, error) {
	return o.DepartmentMember.Search(ctx, keyword, departmentIDs)
}

func (o *OrganizationDatabase) GetDepartmentMemberByDepartmentID(ctx context.Context, departmentID string) ([]*table.DepartmentMember, error) {
	return o.DepartmentMember.GetByDepartmentID(ctx, departmentID)
}

func (o *OrganizationDatabase) GetOrganization(ctx context.Context) (*table.Organization, error) {
	return o.Organization.Get(ctx)
}

func (o *OrganizationDatabase) SetOrganization(ctx context.Context, update map[string]any) error {
	return o.Organization.Set(ctx, update)
}

func (o *OrganizationDatabase) FindDepartmentMemberByUserID(ctx context.Context, userIDList []string) ([]*table.DepartmentMember, error) {
	return o.DepartmentMember.FindByUserID(ctx, userIDList)
}

func (o *OrganizationDatabase) MoveDepartmentMember(ctx context.Context, userID string, oldDepartmentID string, newDepartmentID string) error {
	return o.DepartmentMember.Move(ctx, userID, oldDepartmentID, newDepartmentID)
}

func (o *OrganizationDatabase) GetDepartmentMemberInUserID(ctx context.Context, userIDs []string) ([]*table.DepartmentMember, error) {
	return o.DepartmentMember.FindByUserID(ctx, userIDs)
}

func (o *OrganizationDatabase) CreateDepartmentMember(ctx context.Context, DepartmentMember *table.DepartmentMember) error {
	return o.DepartmentMember.Create(ctx, DepartmentMember)
}

func (o *OrganizationDatabase) CreateDepartmentMembers(ctx context.Context, departmentMembers []*table.DepartmentMember) error {
	return o.DepartmentMember.Creates(ctx, departmentMembers)
}

func (o *OrganizationDatabase) DeleteDepartmentMemberByKey(ctx context.Context, userID string, departmentID string) error {
	return o.DepartmentMember.DeleteByKey(ctx, userID, departmentID)
}

func (o *OrganizationDatabase) UpdateDepartmentMember(ctx context.Context, departmentID string, userID string, update map[string]any) error {
	return o.DepartmentMember.Update(ctx, departmentID, userID, update)
}

func (o *OrganizationDatabase) DeleteDepartmentIDList(ctx context.Context, departmentIDList []string) error {
	return o.DepartmentMember.DeleteDepartmentIDList(ctx, departmentIDList)
}

func (o *OrganizationDatabase) DeleteDepartment(ctx context.Context, departmentIDList []string) error {
	return o.Department.Delete(ctx, departmentIDList)
}

func (o *OrganizationDatabase) UpdateParentID(ctx context.Context, oldParentID, newParentID string) error {
	return o.Department.UpdateParentID(ctx, oldParentID, newParentID)
}

func (o *OrganizationDatabase) GetDepartmentList(ctx context.Context, departmentIDList []string) ([]*table.Department, error) {
	return o.Department.GetList(ctx, departmentIDList)
}

func (o *OrganizationDatabase) FindDepartmentMember(ctx context.Context, departmentIDList []string) ([]*table.DepartmentMember, error) {
	return o.DepartmentMember.FindByDepartmentID(ctx, departmentIDList)
}

func (o *OrganizationDatabase) GetParentDepartment(ctx context.Context, parentID string) ([]*table.Department, error) {
	return o.Department.GetParent(ctx, parentID)
}

func (o *OrganizationDatabase) UpdateDepartment(ctx context.Context, departmentID string, data map[string]any) error {
	return o.Department.Update(ctx, departmentID, data)
}

func (o *OrganizationDatabase) GetDepartmentByID(ctx context.Context, departmentID string) (*table.Department, error) {
	return o.Department.FindOne(ctx, departmentID)
}

func (o *OrganizationDatabase) CreateDepartment(ctx context.Context, department *table.Department) error {
	return o.Department.Create(ctx, department)
}

func (o *OrganizationDatabase) GetDepartment(ctx context.Context, departmentID string) (*table.Department, error) {
	return o.Department.GetDepartment(ctx, departmentID)
}
