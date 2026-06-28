package store

type UserManager interface {
	Register(username, email, password string) (AuthUser, error)
	Authenticate(email, password string) (AuthUser, error)
	FindByID(id int) (AuthUser, bool)
	UpdateProfile(id int, username, bio, avatar string) (AuthUser, error)
	UpsertGitHubUser(githubID int64, username, email, avatar string) (AuthUser, error)
	Follow(followerID, followingID int) (FollowStatus, bool, error)
	Unfollow(followerID, followingID int) (FollowStatus, bool, error)
	FollowStatus(userID, viewerID int) (FollowStatus, error)
	ListFollowing(userID int) ([]PublicUser, error)
	ListFollowers(userID int) ([]PublicUser, error)
}

type PromptManager interface {
	Query(filter PromptFilter) ([]Prompt, error)
	FindByID(id int) (Prompt, bool, error)
	Create(input CreatePromptInput) (Prompt, error)
	Update(id int, userID int, input CreatePromptInput) (Prompt, error)
	Delete(id int, userID int) error
	Like(id int, userID int) (Prompt, bool, error)
	Favorite(id int, userID int) (Prompt, bool, error)
	ListUserFavorites(userID int) ([]Prompt, error)
	ListUserLikes(userID int) ([]Prompt, error)
}

type CommentManager interface {
	ListByTarget(filter CommentFilter) ([]Comment, error)
	Create(input CreateCommentInput) (Comment, error)
	Like(id int, userID int) (Comment, bool, error)
	Report(input ReportCommentInput) (Report, bool, error)
}
