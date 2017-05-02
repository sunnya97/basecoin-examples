package types

import (
	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/basecoin/state"
	"github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/shopspring/decimal"
)

const (
	TBTxNewProfile  byte = 0x01
	TBTxOpenInvoice byte = 0x02
	TBTxOpenExpense byte = 0x03
	TBTxClose       byte = 0x04
)

//////////////////////////////

type Currency string

type CurTime struct {
	cur  currency
	date time.Time
}

type AmtCurTime struct {
	cur    curTime
	amount decimal
}

///////////////////////////////

type Profile struct {
	Address            []byte //address
	Nickname           string //nickname for querying TODO check to make sure only one word
	LegalName          string
	AcceptedCur        []currency //currencies you will accept payment in
	DefaultDepositInfo string     //default deposit information (mostly for fiat)
	DueDurationDays    int        //default duration until a sent invoice due date
}

type Invoice struct {
	ID             int
	AccSender      []byte
	AccReceiver    []byte
	DepositInfo    string
	Amount         AmtCurTime
	AcceptedCur    []Currency
	TransactionID  string     //empty when unpaid
	PaymentCurTime AmtCurTime //currency used to pay invoice, empty when unpaid
}

type Expense struct {
	Invoice
	pdfReceipt []byte
	notes      string
	taxesPaid  AmtCurTime
}

type Close struct {
	ID int
}

func NewTxBytesNewProfile(Address []byte, Nickname string, LegalName string,
	AcceptedCur []currency, DefaultDepositInfo string, DueDurationDays int) []byte {

	data := wire.BinaryBytes(
		Proflie{
			Address:            Address,
			Nickname:           Nickname,
			LegalName:          LegalName,
			AcceptedCur:        AcceptedCur,
			DefaultDepositInfo: DefaultDepositInfo,
			DueDurationDays:    DueDurationDays,
		})
	data = append([]byte{TBTxNewProfile}, data...)
	return data
}

func NewTxBytesOpenInvoice(ID int, AccSender []byte, AccReceiver []byte, DepositInfo string,
	Amount AmtCurTime, AcceptedCur []Currency, TransactionID string, PaymentCurTime AmtCurTime) []byte {

	data := wire.BinaryBytes(
		Invoice{
			ID:             ID,
			AccSender:      AccSender,
			AccReceiver:    AccReceiver,
			DepositInfo:    DepositInfo,
			Amount:         Amount,
			AcceptedCur:    AcceptedCur,
			TransactionID:  TransactionID,
			PaymentCurTime: PaymentCurTime,
		})
	data = append([]byte{TBTxOpenInvoice}, data...)
	return data
}

func NewTxBytesOpenExpense(ID int, AccSender []byte, AccReceiver []byte, DepositInfo string,
	Amount AmtCurTime, AcceptedCur []Currency, TransactionID string, PaymentCurTime AmtCurTime,
	pdfReceipt []byte, notes string, taxesPaid AmtCurTime) []byte {

	data := wire.BinaryBytes(
		Expense{
			ID:             ID,
			AccSender:      AccSender,
			AccReceiver:    AccReceiver,
			DepositInfo:    DepositInfo,
			Amount:         Amount,
			AcceptedCur:    AcceptedCur,
			TransactionID:  TransactionID,
			PaymentCurTime: PaymentCurTime,
			pdfReceipt:     pdfReceipt,
			notes:          notes,
			taxesPaid:      taxesPaid,
		})
	data = append([]byte{TBTxOpenExpense}, data...)
	return data
}

func NewTxBytesClose(ID int) []byte {

	data := wire.BinaryBytes(
		Close{
			ID: ID,
		})
	data = append([]byte{TBTxClose}, data...)
	return data
}

func NewTxBytesNewProfile(Address []byte, Nickname string, LegalName string,
	AcceptedCur []currency, DefaultDepositInfo string, DueDurationDays int) []byte {

	data := wire.BinaryBytes(
		Proflie{
			Address:            Address,
			Nickname:           Nickname,
			LegalName:          LegalName,
			AcceptedCur:        AcceptedCur,
			DefaultDepositInfo: DefaultDepositInfo,
			DueDurationDays:    DueDurationDays,
		})
	data = append([]byte{TBTxNewProfile}, data...)
	return data
}
