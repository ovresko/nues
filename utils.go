package nues

import (
	"crypto/md5"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"

	"io"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	phonePattern = `^0[5679]\d{8}$`
)

func IsValidPhoneNumber(phoneNumber string) bool {
	regex := regexp.MustCompile(phonePattern)
	return regex.MatchString(phoneNumber)
}

func CleanPhoneNumber(rawNumber string) (string, error) {

	if rawNumber == "" {
		return "", ErrPhoneBadFormat
	}

	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(rawNumber, "")
	cleaned = regexp.MustCompile(`^(2130)`).ReplaceAllString(cleaned, "0")

	if IsValidPhoneNumber(cleaned) {
		return cleaned, nil
	}
	return "", ErrPhoneBadFormat
}

func GenerateId() string {
	id := uuid.NewString()
	return id
}
func HashIt(val string) string {
	h := md5.New()
	io.WriteString(h, val)
	pinHash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return pinHash
}
func ParseM(val any) (bson.M, error) {
	data := bson.M{}

	b, err := bson.Marshal(val)
	if err != nil {
		return nil, err
	}
	err = bson.Unmarshal(b, data)
	return data, err
}

func ParseMany[T any](val []any) ([]T, error) {

	res := make([]T, 0)
	for i := 0; i < len(val); i++ {
		single, err := ParseSingle[T](val[i])
		if err != nil {
			return nil, err
		}
		res = append(res, single)
	}
	return res, nil

}
func ParseSingle[T any](val any) (T, error) {

	var t T
	if err := AssertNotEmpty(val, ErrSystemInternal); err != nil {
		return t, err
	}

	var d bson.D
	var m bson.M
	dok := false
	mok := false
	var bsonBytes []byte
	var err error

	d, dok = val.(bson.D)
	if !dok {
		m, mok = val.(bson.M)
		if !mok {
			return t, ErrSystemInternal
		}
	}
	if dok {
		bsonBytes, err = bson.Marshal(d)
		if err != nil {
			return t, err
		}
	} else if mok {
		bsonBytes, err = bson.Marshal(m)
		if err != nil {
			return t, err
		}
	} else {
		return t, ErrSystemInternal
	}

	err = bson.Unmarshal(bsonBytes, &t)
	if err != nil {
		return t, err
	}
	return t, nil
}

func AssertTrue(v bool, err error) error {

	if v != true {
		return err
	}
	return nil
}
func MustNotEmpty(value interface{}, err SysError) {
	if AssertNotEmpty(value, err) != nil {
		panic(err)
	}
}
func AssertNotEmpty(value interface{}, err SysError) error {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		if v == 0 {
			return err
		}
	case string:
		v = value.(string)
		if len(v) == 0 {
			return err

		}
	default:
		if v == nil {
			return err

		}
	}
	return nil
}

func ToFloat64(x interface{}) (float64, error) {

	var y float64
	switch v := x.(type) {
	case string:
		z, err := strconv.ParseFloat(v, 64)
		if err != nil {
			slog.Error("tofloat failed", err)
			return 0, err
		}
		y = z

	case int:
		y = float64(v)
	case int8:
		y = float64(v)
	case int16:
		y = float64(v)
	case int32:
		y = float64(v)
	case int64:
		y = float64(v)
	case uint:
		y = float64(v)
	case uint8:
		y = float64(v)
	case uint16:
		y = float64(v)
	case uint32:
		y = float64(v)
	case uint64:
		y = float64(v)
	case float32:
		y = float64(v)
	case float64:
		y = v
	default:
		slog.Error("", ErrParsingData)
		return 0, ErrParsingData
	}

	return y, nil
}

func GetClientIpAddr(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	ip, _, _ := net.SplitHostPort(IPAddress)
	return ip
}

func phoneValidator(fl validator.FieldLevel) bool {

	if !fl.Field().IsValid() {
		return false
	}

	phone := fl.Field().String()
	if phone == "" {
		return false
	}
	regex := regexp.MustCompile(phonePattern)

	return regex.MatchString(phone)
}
