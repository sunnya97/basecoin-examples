package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type CurrencyTime struct {
	Cur  string
	Date time.Time
}

type AmtCurTime struct {
	CurTime CurrencyTime
	Amount  string //Decimal Number
}

func (a *AmtCurTime) Add(a2 *AmtCurTime) (*AmtCurTime, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return &AmtCurTime{CurrencyTime{a.CurTime.Cur, a.CurTime.Date}, amt1.Add(amt2)}, nil
}

func (a *AmtCurTime) Minus(a2 *AmtCurTime) (*AmtCurTime, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return &AmtCurTime{CurrencyTime{a.CurTime.Cur, a.CurTime.Date}, amt1.Sub(amt2)}, nil
}

func (a *AmtCurTime) GT(a2 *AmtCurTime) (bool, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return amt1.GreaterThan(amt2), nil
}

func (a *AmtCurTime) GTE(a2 *AmtCurTime) (bool, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return amt1.GreaterThanOrEqual(amt2), nil
}

func (a *AmtCurTime) LT(a2 *AmtCurTime) (bool, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return amt1.LessThan(amt2), nil
}

func (a *AmtCurTime) LTE(a2 *AmtCurTime) (bool, error) {
	amt1, amt2, err := getDecimals(a, a2)
	if err != nil {
		return false, err
	}
	return amt1.LessThanOrEaual(amt2), nil
}

func (a *AmtCurTime) validateOperation(a2 *AmtCurTime) error {
	switch {
	case a.Cur != a2.Cur:
		return errors.New("Can't operate on two different currencies")
	case a.Date != a2.Date:
		return errors.New("Can't operate on two different dates")
	}
}

func getDecimals(a1 *AmtCurTime, a2 *AmtCurTime) (amt1 decimal.Decimal, amt2 decimal.Decimal, err error) {
	amt1, err := decimal.NewFromString(a.Amount)
	if err != nil {
		return false, err
	}
	amt2, err := decimal.NewFromString(a2.Amount)
	if err != nil {
		return false, err
	}
	err = a.ValidateOperation(a2)
	if err != nil {
		return false, err
	}
}

///////////////////////////////////////////////////////////////////////////

func ParseAmtCurTime(amtCur string, date time.Time) (*AmtCurTime, error) {

	if len(amtCur) == 0 {
		return nil, errors.New("not enought information to parse AmtCurTime")
	}

	var reAmt = regexp.MustCompile("([\\d\\.]+)")
	var reCur = regexp.MustCompile("([^\\d\\W]+)")
	amt := reAmt.FindString(amtCur)
	cur := reCur.FindString(amtCur)

	return &AmtCurTime{CurrencyTime{cur, date}, amt}, nil
}

func ParseDate(date string, timezone string) (t time.Time, err error) {

	//get the time of invoice
	t = time.Now()
	if len(timezone) > 0 {

		tz := time.UTC
		if len(timezone) > 0 {
			tz, err = time.LoadLocation(timezone)
			if err != nil {
				return t, fmt.Errorf("error loading timezone, error: ", err) //never stack trace
			}
		}

		str := strings.Split(date, "-")
		var ymd = []int{}
		for _, i := range str {
			j, err := strconv.Atoi(i)
			if err != nil {
				return t, err
			}
			ymd = append(ymd, j)
		}
		if len(ymd) != 3 {
			return t, fmt.Errorf("bad date parsing, not 3 segments") //never stack trace
		}

		t = time.Date(ymd[0], time.Month(ymd[1]), ymd[2], 0, 0, 0, 0, tz)

	}

	return t, nil
}
