package types

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

func convertAmtCurTime(denomOut string, in *AmtCurTime) (out *AmtCurTime, err error) {

	inDec, err := decimal.NewFromString(in.Amount)
	if err != nil {
		return out, err
	}

	outDec, err := convert(in.CurTime.Cur, denomOut, inDec, in.CurTime.Date)
	if err != nil {
		return out, err
	}

	return &AmtCurTime{CurrencyTime{denomOut, in.CurTime.Date}, outDec.String()}, nil
}

//XXX NON-DETERMINISTIC
func convert(denomIn, denomOut string, amt decimal.Decimal, date time.Time) (out decimal.Decimal, err error) {
	dateStr := date.Format("2006-01-02")
	urlFiat2USD := fmt.Sprintf("http://api.fixer.io/%v?base=USD", dateStr)
	urlUSD2BTC := fmt.Sprintf("http://api.coindesk.com/v1/bpi/historical/close.json?start=%v&end=%v", dateStr, dateStr)

	//calculate the conversion factor
	conv := decimal.New(1, 1)
	if denomIn != "USD" {
		multiplier, err := getConv(urlFiat2USD, "rates", denomIn)
		if err != nil {
			return out, err
		}
		conv = conv.Mul(multiplier)
	}
	if denomOut == "BTC" {
		multiplier, err := getConv(urlUSD2BTC, "bpi", dateStr)
		if err != nil {
			return out, err
		}
		conv = conv.Mul(multiplier)
	} else if denomOut != "USD" {
		divisor, err := getConv(urlFiat2USD, "rates", denomOut)
		if err != nil {
			return out, err
		}
		conv = conv.Div(divisor)
	}

	return amt.Div(conv), nil
}

//Get the conversion decimal from an http call
//XXX NON-DETERMINISTIC
func getConv(url, index1, index2 string) (out decimal.Decimal, err error) {
	var temp map[string]interface{}
	var client = &http.Client{Timeout: 10 * time.Second}
	r, err := client.Get(url)
	if err != nil {
		return out, err
	}
	err = json.NewDecoder(r.Body).Decode(&temp)
	r.Body.Close()
	if err != nil {
		return out, err
	}
	temp2 := temp[index1].(map[string]interface{})
	return decimal.NewFromFloat(temp2[index2].(float64)), nil
}
