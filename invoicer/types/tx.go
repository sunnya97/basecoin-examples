package types

import (
	"time"

	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmlibs/merkle"
)

const (
	TBIDExpense = iota
	TBIDWage

	TBTxProfileOpen
	TBTxProfileEdit
	TBTxProfileClose

	TBTxWageOpen
	TBTxWageEdit

	TBTxExpenseOpen
	TBTxExpenseEdit

	TBTxCloseInvoice
	TBTxBulkImport
)

func txBytes(object interface{}, tb byte) []byte {
	data := wire.BinaryBytes(object)
	return append([]byte{tb}, data...)
}

type Profile struct {
	Name            string        //identifier for querying
	AcceptedCur     string        //currency you will accept payment in
	DepositInfo     string        //default deposit information (mostly for fiat)
	DueDurationDays int           //default duration until a sent invoice due date
	Timezone        time.Location //default duration until a sent invoice due date
}

func NewProfile(Name string, AcceptedCur string, DepositInfo string,
	DueDurationDays int, Timezone time.Location) Profile {
	return Profile{
		Name:            Name,
		AcceptedCur:     AcceptedCur,
		DepositInfo:     DepositInfo,
		DueDurationDays: DueDurationDays,
		Timezone:        Timezone,
	}
}

func (p Profile) TxBytesOpen() []byte {
	return txBytes(p, TBTxProfileOpen)
}

func (p Profile) TxBytesEdit() []byte {
	return txBytes(p, TBTxProfileEdit)
}

func (p Profile) TxBytesClose() []byte {
	return txBytes(p, TBTxProfileClose)
}

type Invoice interface {
	SetID()
	TxBytesOpen()
	TxBytesEdit()
}

type Wage struct {
	Ctx            Context
	ID             []byte
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurTime //currency used to pay invoice, empty when unpaid
}

//struct used for hash to determine ID
type Context struct {
	Sender      string
	Receiver    string
	DepositInfo string
	Notes       string
	Amount      *AmtCurTime
	AcceptedCur string
	Due         time.Time
}

func NewWage(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurTime, AcceptedCur string, Due time.Time) Wage {

	return Wage{
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

func (w *Wage) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(w.Ctx)
	w.ID = append([]byte{TBIDWage}, hashBytes...)
}

func (w *Wage) TxBytesOpen() []byte {
	return txBytes(w, TBTxWageOpen)
}

func (w *Wage) TxBytesEdit() []byte {
	return txBytes(w, TBTxWageEdit)
}

type Expense struct {
	Ctx            Context
	ID             []byte
	Document       []byte
	DocFileName    string
	ExpenseTaxes   *AmtCurTime
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurTime //currency used to pay invoice, empty when unpaid
}

func NewExpense(Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurTime, AcceptedCur string, Due time.Time,
	Document []byte, DocFileName string, ExpenseTaxes *AmtCurTime) Expense {

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

func (e *Expense) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(e.Ctx)
	e.ID = append([]byte{TBIDExpense}, hashBytes...)
}

func (e *Expense) TxBytesOpen() []byte {
	return txBytes(e, TBTxExpenseOpen)
}

func (e *Expense) TxBytesEdit() []byte {
	return txBytes(e, TBTxExpenseEdit)
}

type CloseInvoice struct {
	ID             []byte
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurTime //currency used to pay invoice, empty when unpaid
}

func NewClose(ID []byte, TransactionID string, PaymentCurTime *AmtCurTime) CloseInvoice {
	return CloseInvoice{
		ID:             ID,
		TransactionID:  TransactionID,
		PaymentCurTime: PaymentCurTime,
	}

}

func (c *CloseInvoice) TxBytes() []byte {
	return txBytes(c, TBTxCloseInvoice)
}
