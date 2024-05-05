package models

type User struct {
	ID          string `json:"id,omitempty"`
	Email       string `json:"email,omitempty"`
	Password    string `json:"password,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	Banner      string `json:"banner,omitempty"`
	Status      string `json:"status,omitempty"`
	AboutMe     string `json:"about_me"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type Session struct {
	ID         string `json:"id,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ExpiresdAt string `json:"expires_at,omitempty"`
	IpAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent"`
	UserId     string `json:"user_id"`
}
