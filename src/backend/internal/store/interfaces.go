package store

import "time"

type UserManager interface {
	Register(username, email, password string) (AuthUser, error)
	Authenticate(email, password string) (AuthUser, error)
	ResetPassword(email, password string) error
	FindByID(id int) (AuthUser, bool)
	UpdateProfile(id int, username, bio, avatar string) (AuthUser, error)
	UpsertGitHubUser(githubID int64, username, email, avatar string) (AuthUser, error)
	Follow(followerID, followingID int) (FollowStatus, bool, error)
	Unfollow(followerID, followingID int) (FollowStatus, bool, error)
	FollowStatus(userID, viewerID int) (FollowStatus, error)
	ListFollowing(userID int) ([]PublicUser, error)
	ListFollowers(userID int) ([]PublicUser, error)
	// BumpSessionVersion increments the user's session version so all previously
	// issued tokens are rejected (used after a password reset).
	BumpSessionVersion(email string) error
	// DeleteAccount disables the account, removes authentication/binding data,
	// anonymizes direct identifiers, and revokes all existing sessions.
	DeleteAccount(id int) error
}

type PromptManager interface {
	Query(filter PromptFilter) ([]Prompt, error)
	// QueryPage returns one page of results and the total count, pushing
	// pagination down to the database layer.
	QueryPage(filter PromptFilter, page, pageSize int) ([]Prompt, int, error)
	// HomeSummary returns real aggregates for the home page.
	HomeSummary() (HomeSummary, error)
	// ListCategories returns categories from the database.
	ListCategories() ([]Category, error)
	// CategoryExists reports whether a prompt-type category exists.
	CategoryExists(id int) (bool, error)
	FindByID(id int) (Prompt, bool, error)
	FindOwnedByID(id int, userID int) (Prompt, bool, error)
	Create(input CreatePromptInput) (Prompt, error)
	Update(id int, userID int, input CreatePromptInput) (Prompt, error)
	Delete(id int, userID int) error
	Like(id int, userID int) (Prompt, bool, error)
	Unlike(id int, userID int) (Prompt, bool, error)
	Favorite(id int, userID int) (Prompt, bool, error)
	Unfavorite(id int, userID int) (Prompt, bool, error)
	RecordView(id int, userID int) (Prompt, bool, error)
	Report(id int, userID int, reason string, detail string) (Report, bool, error)
	// GetInteractionStatus reports whether the given user has liked/favorited a
	// prompt. It is used by the frontend to render toggle state without guessing
	// from the denormalized counters. A missing or soft-deleted prompt yields
	// ErrPromptNotFound so callers cannot infer existence.
	GetInteractionStatus(id int, userID int) (InteractionStatus, error)
	ListUserFavorites(userID int) ([]Prompt, error)
	ListUserLikes(userID int) ([]Prompt, error)
	ListUserHistory(userID int) ([]Prompt, error)
	// ListUserHistoryPage returns a page of the requesting user's browsing
	// history plus the total, always excluding soft-deleted prompts. Pagination
	// is pushed down to the store for both MySQL and in-memory backends so the
	// history list stays cheap as the table grows.
	ListUserHistoryPage(userID int, page, pageSize int) ([]Prompt, int, error)
	ListUserDrafts(userID int) ([]Prompt, error)
	// ListUserPrompts returns the caller's non-deleted prompts, including drafts,
	// for the personal data export flow.
	ListUserPrompts(userID int) ([]Prompt, error)
	// ClearUserHistory permanently removes only the caller's browsing history.
	ClearUserHistory(userID int) error
}

// HomeSummary carries live community aggregates for the home page.
type HomeSummary struct {
	PromptCount   int      `json:"promptCount"`
	CreatorCount  int      `json:"creatorCount"`
	TotalViews    int64    `json:"totalViews"`
	HotTags       []string `json:"hotTags"`
	HotCategories []string `json:"hotCategories"`
}

type CommentManager interface {
	ListByTarget(filter CommentFilter) ([]Comment, error)
	// ListByTargetPage returns one page of root comments plus the total.
	ListByTargetPage(filter CommentFilter, page, pageSize int) ([]Comment, int, error)
	Create(input CreateCommentInput) (Comment, error)
	Like(id int, userID int) (Comment, bool, error)
	Report(input ReportCommentInput) (Report, bool, error)
}

// InteractionStatus is the per-user like/favorite state of a prompt, returned by
// GetInteractionStatus so the frontend can render toggle buttons accurately.
type InteractionStatus struct {
	Liked     bool `json:"liked"`
	Favorited bool `json:"favorited"`
}

// Upload purpose discriminates why an object was uploaded, which scopes its
// ownership and validation rules.
type UploadPurpose string

const (
	// UploadPurposePromptImage is an uploaded image attached to a prompt cover or gallery.
	UploadPurposePromptImage UploadPurpose = "prompt_image"
	// UploadPurposeAvatar is an uploaded user avatar.
	UploadPurposeAvatar UploadPurpose = "avatar"
)

// UploadStatus describes the lifecycle of an uploaded object.
type UploadStatus string

const (
	// UploadStatusPending means the object was written but not yet referenced.
	UploadStatusPending UploadStatus = "pending"
	// UploadStatusReferenced means a prompt/avatar now references the object.
	UploadStatusReferenced UploadStatus = "referenced"
	// UploadStatusTrashed means the object was soft-deleted and scheduled for removal.
	UploadStatusTrashed UploadStatus = "trashed"
)

// UploadRecord is a persisted row in the uploads table. It lets the backend
// prove ownership of a private temporary upload and garbage-collect unreferenced
// objects instead of leaking them in the bucket.
type UploadRecord struct {
	ID          int64         `json:"id"`
	OwnerID     int           `json:"ownerId"`
	Provider    string        `json:"provider"`
	Purpose     UploadPurpose `json:"purpose"`
	ObjectKey   string        `json:"objectKey"`
	ContentType string        `json:"contentType"`
	Size        int64         `json:"size"`
	Status      UploadStatus  `json:"status"`
	CreatedAt   string        `json:"createdAt"`
}

// UploadManager persists upload metadata and validates ownership. It is
// independent of the storage backend so both local and R2 implementations can
// be audited for dangling objects.
type UploadManager interface {
	// RecordUpload writes a metadata row for an object that was just stored.
	RecordUpload(rec UploadRecord) (UploadRecord, error)
	// MarkUploadsReferenced flips pending uploads to referenced once a prompt or
	// avatar references their object keys, protecting them from cleanup.
	MarkUploadsReferenced(objectKeys []string, ownerID int) error
	// FindUpload returns the metadata for an object key.
	FindUpload(objectKey string) (UploadRecord, bool, error)
	// SoftDeleteUpload marks an upload as trashed (no hard delete here).
	SoftDeleteUpload(objectKey string, ownerID int) error
	// ListUnreferencedUploads returns pending uploads older than the given age,
	// which a cleanup job can safely remove.
	ListUnreferencedUploads(olderThan time.Time) ([]UploadRecord, error)
	// TrashUnreferenced flips unreferenced uploads older than olderThan to
	// trashed in one call, returning the keys that were transitioned.
	TrashUnreferenced(olderThan time.Time) ([]string, error)
	// ActiveUploadBytes returns the persisted bytes for uploads that have not
	// been trashed. It is used to enforce the configured storage capacity.
	ActiveUploadBytes() (int64, error)
}

// Clock abstracts time.Now for testability of time-based logic.
type Clock func() time.Time
