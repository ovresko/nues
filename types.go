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

func (c ConfigNues) Name() string {
	return "nues"
}

type TransferredOperation int

const (
	OpSend TransferredOperation = iota
	OpPay
	OpCommission
	OpFee
	OpVoucher
	OpCashin
	OpCashout
)

type UserLevel int

const (
	UserRegular UserLevel = iota
	UserAgent
	UserMerchant
	UserRelay
)

type OpSetting struct {
	FeeFlat           float64 `json:"fee_flat" bson:"fee_flat"`
	FeePercent        float64 `json:"fee_percent" bson:"fee_percent"`
	FeeRecipient      string  `json:"fee_recipient" bson:"fee_recipient"`
	CommissionFlat    float64 `json:"commission_flat" bson:"commission_flat"`
	CommissionPercent float64 `json:"commission_percent" bson:"commission_percent"`
	CommissionSender  string  `json:"commission_sender" bson:"commission_sender"`
	CashinSender      string  `json:"cashin_sender" bson:"cashin_sender"`
	VoucherSender     string  `json:"voucher_sender" bson:"voucher_sender"`
	CashoutRecipient  string  `json:"cashout_recipient" bson:"cashout_recipient"`
	Limit             float64 `json:"limit" bson:"limit"`
}

type ProductRateConfig struct {
	Level            UserLevel `json:"level" bson:"level"`
	FeeFlat          float64   `json:"fee_flat" bson:"fee_flat"`
	FeePercent       float64   `json:"fee_percent" bson:"fee_percent"`
	CommissionFlat   float64   `json:"commission_flat" bson:"commission_flat"`
	CommissionPercet float64   `json:"commission_percet" bson:"commission_percet"`
}

type ConfigService interface {
	Name() string
}
