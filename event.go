package nues

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var EvAttemptName string = "EvAttempt"

type EvAttempt struct {
	EvName  string      `json:"ev_name"`
	Command interface{} `json:"command"`
}

var EvProductDelistedName string = "EvProductDelisted"

type EvProductDelisted struct {
	Id string `validate:"required" bson:"_id" json:"id"`
}

var EvProductEnlistedName string = "EvProductEnlisted"

type EvProductEnlisted struct {
	Id          string         `validate:"required" bson:"_id" json:"id"`
	MerchantId  string         `validate:"required,identity" json:"merchant_id" bson:"merchant_id"`
	ProductName string         `validate:"required" json:"product_name" bson:"product_name"`
	Active      bool           `json:"active" bson:"active"`
	Banner      string         `json:"banner" bson:"banner"`
	Category    string         `validate:"required" json:"category" bson:"category"`
	Description string         `validate:"required" json:"description" bson:"description"`
	Featured    bool           `json:"featured" bson:"featured"`
	Fields      []ProductField `validate:"dive" json:"fields" bson:"fields"`
	Image       string         `validate:"required" json:"image" bson:"image"`
	ProductUrl  string         `json:"product_url" bson:"product_url"`
}

type ProductField struct {
	Id         string  `validate:"required" bson:"_id" json:"id"`
	Name       string  `validate:"required" json:"name" bson:"name"`
	Label      string  `validate:"required" json:"label" bson:"label"`
	Type       string  `validate:"required" json:"type" bson:"type"`
	Required   bool    `json:"required" bson:"required"`
	IsQty      bool    `bson:"is_qty" json:"is_qty"`
	Min        float64 `json:"min" bson:"min"`
	Max        float64 `json:"max" bson:"max"`
	Choices    string  `json:"choices" bson:"choices"`
	Validation string  `json:"validation" bson:"validation"`
}

var EvAppConfigUpdatedName string = "EvAppConfigUpdated"

type EvAppConfigUpdated struct {
	AppName        string
	SupportContact string
	SupportWebsite string
	Banners        []map[string]interface{}
}

var EvSignedupName string = "EvSignedup"

type EvSignedup struct {
	UserId   string `validate:"required" json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Pin      string `json:"pin"`
	FcmToken string `json:"fcm_token"`
}

var EvUserBlockedName string = "EvUserBlocked"

type EvUserBlocked struct {
	UserId string `json:"id"`
}

var EvUserUnblockedName string = "EvUserUnblocked"

type EvUserUnblocked struct {
	UserId string `json:"id"`
}

var EvLoggedinName string = "EvLoggedin"

type EvLoggedin struct {
	UserId string `json:"user_id"`
	Phone  string `json:"phone"`
	Pin    string `json:"pin"`
	Token  string `json:"token"`
}

var EvLoggedOutName string = "EvLoggedOut"

type EvLoggedOut struct {
	UserId string `json:"user_id"`
}

var EvProfileUpdatedName string = "EvProfileUpdated"

type EvProfileUpdated struct {
	UserId   string    `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Gender   string    `json:"gender"`
	Birthday time.Time `json:"birthday"`
	Wilaya   string    `json:"wilaya"`
}

var EvOtpSentName string = "EvOtpSent"

type EvOtpSent struct {
	Phone  string `json:"phone"`
	Token  string `json:"token"`
	Caller string `json:"caller"`
}

var EvAccountDeletedName string = "EvAccountDeleted"

type EvAccountDeleted struct {
	UserId string `json:"user_id"`
}

var EvUpgradedName string = "EvUpgraded"

type EvUpgraded struct {
	UserId         string                 `validate:"required,identity" json:"user_id" bson:"user_id"`
	Username       string                 `validate:"required" json:"username" bson:"username"`
	Levels         []UserLevel            `json:"levels" bson:"levels"`
	Phone2         string                 `json:"phone_2" bson:"phone_2"`
	Phone3         string                 `json:"phone_3" bson:"phone_3"`
	Wilaya         string                 `json:"wilaya" bson:"wilaya"`
	SendLimit      float64                `validate:"required,ne=0" json:"send_limit" bson:"send_limit"`
	Long           float64                `json:"long" bson:"long"`
	Lat            float64                `json:"lat" bson:"lat"`
	Extras         map[string]interface{} `json:"extras" bson:"extras"`
	CustomSettings map[TransferredOperation]OpSetting
}

var EvPinResetName string = "EvPinReset"

type EvPinReset struct {
	Phone  string `json:"phone"`
	UserId string `json:"user_id"`
	Pin    string `json:"pin"`
}

var EvBalanceLockedName string = "EvBalanceLocked"

type EvBalanceLocked struct {
	UserId string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

var EvBalanceUnlockedName string = "EvBalanceUnlocked"

type EvBalanceUnlocked struct {
	EvBalanceLockedId string  `json:"ev_balance_locked_id"`
	UserId            string  `json:"user_id"`
	Amount            float64 `json:"amount"`
}

var EvSendRequestIssuedName string = "EvSendRequestIssued"

type EvSendRequestIssued struct {
	Reference string  `validate:"" json:"reference" bson:"reference"`
	UserId    string  `validate:"required" json:"user_id" bson:"user_id"`
	Amount    float64 `validate:"required,gt=0" json:"amount" bson:"amount"`
	Note      string  `validate:"required" json:"note" bson:"note"`
}

var EvVoucherCreatedName string = "EvVoucherCreated"

type EvVoucherCreated struct {
	Code    string    `json:"code"`
	Barcode string    `json:"barcode"`
	Amount  float64   `json:"amount"`
	Date    time.Time `json:"date"`
	Expires time.Time `json:"expires"`
}

var EvVoucherRedeemedName string = "EvVoucherRedeemed"

type EvVoucherRedeemed struct {
	Reference string  `json:"reference" bson:"reference"`
	Code      string  `json:"code" bson:"code"`
	UserId    string  `json:"user_id" bson:"user_id"`
	Amount    float64 `json:"amount" bson:"amount"`
}

var EvVoucherVerifiedName string = "EvVoucherVerified"

type EvVoucherVerified struct {
	Barcode string `json:"barcode"`
	UserId  string `json:"user_id"`
}

var EvVoucherExpiredName string = "EvVoucherExpired"

type EvVoucherExpired struct {
	Code string    `json:"code"`
	Date time.Time `json:"date"`
}

var EvSentName string = "EvSent"

type EvSent struct {
	Reference   string  `json:"reference" bson:"reference"`
	SenderId    string  `json:"sender_id" bson:"sender_id"`
	RecipientId string  `json:"recipient_id" bson:"recipient_id"`
	Comments    string  `json:"comments" bson:"comments"`
	Purpose     string  `json:"purpose" bson:"purpose"`
	RequestId   string  `json:"request_id" bson:"request_id"`
	Amount      float64 `json:"amount" bson:"amount"`
}

var EvTransferredName string = "EvTransferred"

type EvTransferred struct {
	Reference   string               `validate:"required" json:"reference" bson:"reference"`
	SenderId    string               `validate:"required,identity" json:"sender_id" bson:"sender_id"`
	RecipientId string               `validate:"required,identity" json:"recipient_id" bson:"recipient_id"`
	Amount      float64              `validate:"required,gt=0" json:"amount" bson:"amount"`
	Operation   TransferredOperation `validate:"required" json:"operation" bson:"operation"`
}

var EvCashTopupName string = "EvCashTopup"

type EvCashTopup struct {
	Reference string  `json:"reference" bson:"reference"`
	UserId    string  `json:"user_id" bson:"user_id"`
	Amount    float64 `json:"amount" bson:"amount"`
	Paid      float64 `json:"paid" bson:"paid"`
}

var EvPayseraTopupName string = "EvPayseraTopup"

type EvPayseraTopup struct {
	UserId string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Paid   float64 `json:"paid"`
}

var EvCibTopupName string = "EvCibTopup"

type EvCibTopup struct {
	UserId string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Paid   float64 `json:"paid"`
}

var EvCommissionName string = "EvCommission"

type EvCommission struct {
	UserId    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

var EvFeeName string = "EvFee"

type EvFee struct {
	UserId    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

var EvProductStockName string = "EvProductStock"

type EvProductStock struct {
	Details   string  `validate:"required" json:"details" bson:"details"`
	ProductId string  `validate:"required" json:"product_id" bson:"product_id"`
	Price     float64 `validate:"required,gt=0" json:"price" bson:"price"`
	Qty       float64 `validate:"required,ne=0" json:"qty" bson:"qty"`
}

var EvPurchaseName string = "EvPurchase"

type EvPurchase struct {
	Reference            string                 `json:"reference"`
	UserId               string                 `json:"user_id"`
	ProductId            string                 `json:"product_id"`
	ProductName          string                 `json:"product_name"`
	Price                float64                `json:"price"`
	CalculatedFee        float64                `json:"calculated_fee"`
	CalculatedCommission float64                `json:"calculated_commission"`
	Qty                  float64                `json:"qty"`
	Total                float64                `json:"total"`
	Form                 map[string]interface{} `json:"form"`
}

var EvDeliveredName string = "EvDelivered"

type EvDelivered struct {
	Reference   string `json:"reference"`
	UserId      string `json:"user_id"`
	ProductId   string `json:"product_id"`
	ProductName string `json:"product_name"`
}

var EvProductRatesUpdatedName string = "EvProductRatesUpdated"

type EvProductRatesUpdated struct {
	ProductId    string              `validate:"required" json:"product_id" bson:"product_id"`
	ProductPrice float64             `validate:"required,gt=0" json:"product_price" bson:"product_price"`
	UserRates    []ProductRateConfig `json:"user_rates" bson:"user_rates"`
}

var EvSendRatesUpdatedName string = "EvSendRatesUpdated"

type EvSendRatesUpdated struct {
	SendUserFeeFlat     float64 `json:"send_user_fee_flat"`
	SendUserFeePercent  float64 `json:"send_user_fee_percent"`
	SendAgentFeeFlat    float64 `json:"send_agent_fee_flat"`
	SendAgentFeePercent float64 `json:"send_agent_fee_percent"`
}

var evMutex sync.Mutex

type Event struct {
	Id        string      `bson:"_id" json:"id"`
	Name      string      `bson:"name" json:"name"`
	Sequence  int64       `bson:"sequence" json:"sequence"`
	Timestamp time.Time   `bson:"timestamp" json:"timestamp"`
	Data      interface{} `bson:"data" json:"data"`
}

func (e *Event) save(ctx context.Context) error {
	defer evMutex.Unlock()
	evMutex.Lock()

	if len(e.Id) == 0 {
		return fmt.Errorf("event id is empty")
	}
	if len(e.Name) == 0 {
		return fmt.Errorf("event name is empty")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("event tiemstamp is not valid")
	}

	_, err := DB.GetCollection(nues.colEvents).InsertOne(ctx, e)

	return err
}

func RegisterEvents(ctx context.Context, evs ...interface{}) error {

	last := GetLastSequence()
	for _, ev := range evs {
		last = last + 1
		evName := reflect.TypeOf(ev).Name()
		if len(evName) == 0 {
			panic("unknown event name")
		}
		e := &Event{
			Id:        GenerateId(),
			Name:      evName,
			Sequence:  last,
			Timestamp: time.Now(),
			Data:      ev,
		}
		if err := e.save(ctx); err != nil {
			slog.Error("event save failed", err)
			return ErrSystemInternal
		}
	}
	return nil

}

func GetLastSequence() int64 {
	defer evMutex.Unlock()
	evMutex.Lock()

	var res bson.M
	err := DB.GetCollection(nues.colEvents).FindOne(context.TODO(), bson.D{}, options.FindOne().SetProjection(bson.D{{"sequence", 1}}).SetSort(bson.D{{"sequence", -1}})).Decode(&res)
	if err != nil {
		switch err {
		case mongo.ErrNoDocuments:
			return 0
		default:
			panic(err)
		}
	}
	return res["sequence"].(int64)
}
