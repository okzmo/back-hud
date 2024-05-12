package models

type FriendRequest struct {
	CreatedAt   string `json:"created_at"`
	ID          string `json:"id"`
	InitiatorId string `json:"initiator_id"`
	Message     string `json:"message"`
	RequestId   string `json:"request_id"`
	Type        string `json:"type"`
	UserId      string `json:"user_id"`
}
