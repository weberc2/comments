package comments

import "time"

type PostID string
type CommentID string
type UserID string

type Comment struct {
	ID       CommentID `json:"id"`
	Post     PostID    `json:"post"`
	Parent   CommentID `json:"parent"`
	Author   UserID    `json:"author"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Body     string    `json:"body"`
}
