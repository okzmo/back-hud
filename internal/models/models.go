package models

type User struct {
	ID          string `json:"id,omitempty"`
	Email       string `json:"email"`
	Password    string `json:"password,omitempty"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
	Banner      string `json:"banner"`
	Status      string `json:"status"`
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
