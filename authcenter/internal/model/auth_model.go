package model

import (
	"net/netip"
)

type LoginResponseModel struct {
	AccessToken  string
	RefreshToken string
	User         UserModel
}

type GetSessionByRequestInfoModel struct {
	UserID     int32
	IpAddress  netip.Addr
	DeviceInfo string
	Region     *string
	UserAgent  *string
}
