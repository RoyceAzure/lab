package model

type PermissionModel struct {
	ID          int32
	Name        string
	Description *string
	Resource    string
	Actions     string
}
