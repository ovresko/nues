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

var EvPhoneValidatedName string = "EvPhoneValidated"

type EvPhoneValidated struct {
	UserId string `json:"user_id"`
	Phone  string `json:"phone"`
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
	UserId   string                 `json:"user_id"`
	Username string                 `json:"username"`
	Type     string                 `json:"type"`
	Phone2   string                 `json:"phone_2"`
	Phone3   string                 `json:"phone_3"`
	Wilaya   string                 `json:"wilaya"`
	Long     float64                `json:"long"`
	Lat      float64                `json:"lat"`
	Extras   map[string]interface{} `json:"extras"`
}

var EvDowngradedName string = "EvDowngraded"

type EvDowngraded struct {
	UserId string `json:"user_id"`
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
	Reference string  `json:"reference"`
	UserId    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Note      string  `json:"note"`
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
	Code   string  `json:"code"`
	UserId string  `json:"user_id"`
	Amount float64 `json:"amount"`
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
	Reference string  `json:"reference"`
	UserId    string  `json:"user_id"`
	Recipient string  `json:"recipient"`
	Comments  string  `json:"comments"`
	Purpose   string  `json:"purpose"`
	RequestId string  `json:"request_id"`
	Amount    float64 `json:"amount"`
}

var EvCashTopupName string = "EvCashTopup"

type EvCashTopup struct {
	UserId string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Paid   float64 `json:"paid"`
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
	Details   string  `json:"details"`
	ProductId string  `json:"product_id"`
	Price     float64 `json:"price"`
	Qty       float64 `json:"qty"`
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
	ProductId                     string  `json:"product_id"`
	ProductPrice                  float64 `json:"product_price"`
	ProductAgentCommissionPercent float64 `json:"product_agent_commission_percent"`
	ProductAgentFeePercent        float64 `json:"product_agent_fee_percent"`
	ProductAgentCommissionFlat    float64 `json:"product_agent_commission_flat"`
	ProductAgentFeeFlat           float64 `json:"product_agent_fee_flat"`
	ProductUserCommissionPercent  float64 `json:"product_user_commission_percent"`
	ProductUserFeePercent         float64 `json:"product_user_fee_percent"`
	ProductUserCommissionFlat     float64 `json:"product_user_commission_flat"`
	ProductUserFeeFlat            float64 `json:"product_user_fee_flat"`
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
