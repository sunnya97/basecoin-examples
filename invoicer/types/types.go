package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmlibs/merkle"
)

const (
	TBIDExpense byte = 0x01
	TBIDInvoice byte = 0x02

	TBTxNewProfile  byte = 0x01
	TBTxOpenInvoice byte = 0x02
	TBTxOpenExpense byte = 0x03
	TBTxClose       byte = 0x04
)

//////////////////////////////

type CurTime struct {
	cur  string
	date time.Time
}

type AmtCurDate struct {
	cur    CurTime
	amount decimal.Decimal
}

func ParseAmtCurDate(amtCur string, date time.Time) (*AmtCurDate, error) {

	if len(amtCur) == 0 {
		return nil, errors.New("not enought information to parse AmtCurDate")
	}

	var reAmt = regexp.MustCompile("(\\d+)")
	amt, err := decimal.NewFromString(reAmt.FindString(amtCur))
	if err != nil {
		return nil, err
	}

	var reCur = regexp.MustCompile("([^\\d\\W]+)")
	cur := reCur.FindString(amtCur)

	return &AmtCurDate{CurTime{cur, date}, amt}, nil
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

///////////////////////////////

type Profile struct {
	Name            string        //identifier for querying
	AcceptedCur     string        //currency you will accept payment in
	DepositInfo     string        //default deposit information (mostly for fiat)
	DueDurationDays int           //default duration until a sent invoice due date
	Timezone        time.Location //default duration until a sent invoice due date
}

func NewProfile(Name string, AcceptedCur string,
	DepositInfo string, DueDurationDays int) Profile {
	return Profile{
		Name:            Name,
		AcceptedCur:     AcceptedCur,
		DepositInfo:     DepositInfo,
		DueDurationDays: DueDurationDays,
	}
}

func NewTxBytesNewProfile(Name string, AcceptedCur string,
	DepositInfo string, DueDurationDays int) []byte {

	data := wire.BinaryBytes(NewProfile(Name, AcceptedCur,
		DepositInfo, DueDurationDays))
	data = append([]byte{TBTxNewProfile}, data...)
	return data
}

type Invoice struct {
	Ctx            Context
	ID             []byte
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurDate //currency used to pay invoice, empty when unpaid
}

//struct used for hash to determine ID
type Context struct {
	Sender      string
	Receiver    string
	DepositInfo string
	Notes       string
	Amount      *AmtCurDate
	AcceptedCur string
	Due         time.Time
}

func (i Invoice) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(i.Ctx)
	i.ID = append([]byte{TBIDInvoice}, hashBytes...)
}

func NewInvoice(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurDate, AcceptedCur string, Due time.Time) Invoice {

	return Invoice{
		Ctx: Context{
			Sender:      Sender,
			Receiver:    Receiver,
			DepositInfo: DepositInfo,
			Notes:       Notes,
			Amount:      Amount,
			AcceptedCur: AcceptedCur,
			Due:         Due,
		},
		ID:             nil,
		TransactionID:  "",
		PaymentCurTime: nil,
	}
}

func NewTxBytesOpenInvoice(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurDate, AcceptedCur string, Due time.Time) []byte {

	data := wire.BinaryBytes(NewInvoice(Sender, Receiver, DepositInfo, Notes,
		Amount, AcceptedCur, Due))
	data = append([]byte{TBTxOpenInvoice}, data...)
	return data
}

type Expense struct {
	Ctx            Context
	ID             []byte
	Document       []byte
	DocFileName    string
	ExpenseTaxes   *AmtCurDate
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurDate //currency used to pay invoice, empty when unpaid
}

func (e *Expense) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(e.Ctx)
	e.ID = append([]byte{TBIDExpense}, hashBytes...)
}

func NewExpense(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurDate, AcceptedCur string, Due time.Time,
	Document []byte, DocFileName string, ExpenseTaxes *AmtCurDate) Expense {

	return Expense{
		Ctx: Context{
			Sender:      Sender,
			Receiver:    Receiver,
			DepositInfo: DepositInfo,
			Notes:       Notes,
			Amount:      Amount,
			AcceptedCur: AcceptedCur,
			Due:         Due,
		},
		ID:             nil,
		Document:       Document,
		DocFileName:    DocFileName,
		ExpenseTaxes:   ExpenseTaxes,
		TransactionID:  "",
		PaymentCurTime: nil,
	}
}

func NewTxBytesOpenExpense(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurDate, AcceptedCur string, Due time.Time,
	Document []byte, DocFileName string, TaxesPaid *AmtCurDate) []byte {

	data := wire.BinaryBytes(NewExpense(Sender, Receiver, DepositInfo, Notes,
		Amount, AcceptedCur, Due, Document, DocFileName, TaxesPaid))

	data = append([]byte{TBTxOpenExpense}, data...)
	return data
}

type Close struct {
	ID             []byte
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurDate //currency used to pay invoice, empty when unpaid
}

func NewClose(ID []byte, TransactionID string, PaymentCurTime *AmtCurDate) Close {
	return Close{
		ID:             ID,
		TransactionID:  TransactionID,
		PaymentCurTime: PaymentCurTime,
	}

}

func NewTxBytesClose(ID []byte, TransactionID string, PaymentCurTime *AmtCurDate) []byte {
	data := wire.BinaryBytes(NewClose(ID, TransactionID, PaymentCurTime))
	data = append([]byte{TBTxClose}, data...)
	return data
}
