package nues

type ConfigNues struct {
	Id             string `json:"id" bson:"_id"`
	Reset          bool   `json:"reset" bson:"reset"`
	AdminToken     string `json:"admin_token" bson:"admin_token"`
	ColCommands    string `json:"col_commands" bson:"col_commands"`
	ColEvents      string `json:"col_events" bson:"col_events"`
	ColWatchers    string `json:"col_watchers" bson:"col_watchers"`
	ColSessions    string `json:"col_sessions" bson:"col_sessions"`
	ColIdentity    string `json:"col_identity" bson:"col_identity"`
	ColProjections string `json:"col_projections" bson:"col_projections"`
	DbPrefix       string `json:"db_prefix" bson:"db_prefix"`
}

type TransferredOperation int

const (
	SendOp TransferredOperation = iota
	PayOp
	CommissionOp
	FeeOp
	VoucherOp
	CashinOp
	CashoutOp
)

type UserLevel int

const (
	UserRegular UserLevel = iota
	UserAgent
	UserMerchant
	UserRelay
)
