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
	EvName  string
	Command interface{}
}

var EvSignedupName string = "EvSignedup"

type EvSignedup struct {
	Username string
	Phone    string
	Pin      string
	FcmToken string
}

var EvLoggedinName string = "EvLoggedin"

type EvLoggedin struct {
	UserId string
	Phone  string
	Pin    string
	Token  string
}

var EvLoggedOutName string = "EvLoggedOut"

type EvLoggedOut struct {
	UserId string
}

var EvProfileUpdatedName string = "EvProfileUpdated"

type EvProfileUpdated struct {
	UserId   string
	Username string
	Email    string
	Gender   string
	Birthday time.Time
}

var EvPhoneValidatedName string = "EvPhoneValidated"

type EvPhoneValidated struct {
	UserId string
	Phone  string
}

var EvOtpSentName string = "EvOtpSent"

type EvOtpSent struct {
	Phone string
}

var EvAccountDeletedName string = "EvAccountDeleted"

type EvAccountDeleted struct {
	UserId string
}

var EvUpgradedName string = "EvUpgraded"

type EvUpgraded struct {
	UserId   string
	Username string
	Type     string
	Phone2   string
	Phone3   string
	Wilaya   string
	Long     float64
	Lat      float64
	Extras   map[string]interface{}
}

var EvDowngradedName string = "EvDowngraded"

type EvDowngraded struct {
	UserId string
}

var EvPinResetName string = "EvPinReset"

type EvPinReset struct {
	Phone  string
	UserId string
	Pin    string
}

var EvBalanceLockedName string = "EvBalanceLocked"

type EvBalanceLocked struct {
	UserId string
	Amount float64
	Reason string
}

var EvBalanceUnlockedName string = "EvBalanceUnlocked"

type EvBalanceUnlocked struct {
	EvBalanceLockedId string
	UserId            string
	Amount            float64
}

var EvSendRequestIssuedName string = "EvSendRequestIssued"

type EvSendRequestIssued struct {
	Reference string
	UserId    string
	Amount    float64
	Note      string
}

var EvVoucherCreatedName string = "EvVoucherCreated"

type EvVoucherCreated struct {
	Code    string
	Barcode string
	Amount  float64
	Date    time.Time
	Expires time.Time
}

var EvVoucherRedeemedName string = "EvVoucherRedeemed"

type EvVoucherRedeemed struct {
	Code   string
	UserId string
	Amount float64
}

var EvVoucherVerifiedName string = "EvVoucherVerified"

type EvVoucherVerified struct {
	Barcode string
	UserId  string
}

var EvVoucherExpiredName string = "EvVoucherExpired"

type EvVoucherExpired struct {
	Code string
	Date time.Time
}

var EvSentName string = "EvSent"

type EvSent struct {
	Reference string
	UserId    string
	Recipient string
	Comments  string
	Purpose   string
	RequestId string
	Amount    float64
}

var EvCashTopupName string = "EvCashTopup"

type EvCashTopup struct {
	UserId string
	Amount float64
	Paid   float64
}

var EvPayseraTopupName string = "EvPayseraTopup"

type EvPayseraTopup struct {
	UserId string
	Amount float64
	Paid   float64
}

var EvCibTopupName string = "EvCibTopup"

type EvCibTopup struct {
	UserId string
	Amount float64
	Paid   float64
}

var EvCommissionName string = "EvCommission"

type EvCommission struct {
	UserId    string
	Amount    float64
	Reference string
}

var EvFeeName string = "EvFee"

type EvFee struct {
	UserId    string
	Amount    float64
	Reference string
}

var EvProductStockName string = "EvProductStock"

type EvProductStock struct {
	Details   string
	ProductId string
	Price     float64
	Qty       float64
}

var EvPurchaseName string = "EvPurchase"

type EvPurchase struct {
	Reference            string
	UserId               string
	ProductId            string
	ProductName          string
	Price                float64
	CalculatedFee        float64
	CalculatedCommission float64
	Qty                  float64
	Total                float64
	Form                 map[string]interface{}
}

var EvDeliveredName string = "EvDelivered"

type EvDelivered struct {
	Reference   string
	UserId      string
	ProductId   string
	ProductName string
}

var EvProductRatesUpdatedName string = "EvProductRatesUpdated"

type EvProductRatesUpdated struct {
	ProductId                     string
	ProductPrice                  float64
	ProductAgentCommissionPercent float64
	ProductAgentFeePercent        float64
	ProductAgentCommissionFlat    float64
	ProductAgentFeeFlat           float64
	ProductUserCommissionPercent  float64
	ProductUserFeePercent         float64
	ProductUserCommissionFlat     float64
	ProductUserFeeFlat            float64
}

var EvSendRatesUpdatedName string = "EvSendRatesUpdated"

type EvSendRatesUpdated struct {
	SendUserFeeFlat     float64
	SendUserFeePercent  float64
	SendAgentFeeFlat    float64
	SendAgentFeePercent float64
}

var evMutex sync.Mutex

type Event struct {
	Id        string      `bson:"_id"`
	Name      string      `bson:"name"`
	Sequence  int64       `bson:"sequence"`
	Timestamp time.Time   `bson:"timestamp"`
	Data      interface{} `bson:"data"`
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
