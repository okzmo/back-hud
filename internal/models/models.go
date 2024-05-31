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

type Server struct {
	ID         string     `json:"id,omitempty"`
	Name       string     `json:"name"`
	Icon       string     `json:"icon,omitempty"`
	Banner     string     `json:"banner,omitempty"`
	Categories []Category `json:"categories,omitempty"`
	Roles      []string   `json:"roles,omitempty"`
	CreatedAt  string     `json:"created_at,omitempty"`
}

type Category struct {
	Name     string    `json:"name"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Private   bool   `json:"private"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Message struct {
	ID        string `json:"id,omitempty"`
	Author    User   `json:"author"`
	ChannelId string `json:"channel_id"`
	Content   any    `json:"content"`
	Edited    bool   `json:"edited"`
	UpdatedAt string `json:"updated_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type WSMessage struct {
	Type    string `json:"type"`
	Content any    `json:"content"`
}
