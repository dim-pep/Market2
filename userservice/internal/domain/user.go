package domain

type UserRole int

const (
	UserRoleUndefined UserRole = iota
	UserRoleBasic
	UserRolePremium
	UserRoleAdmin
)

type User struct {
	id       string
	username string
	fullName string
	userRole UserRole
}

func NewUser(id, name string, fullName string, userRole UserRole) *User {
	return &User{id: id, username: name, fullName: fullName, userRole: userRole}
}
