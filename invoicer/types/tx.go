package types

import (
	"time"

	bcmd "github.com/tendermint/basecoin/cmd/commands"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmlibs/merkle"
)

const (
	TBIDExpense = iota
	TBIDWage

	TBTxProfileOpen
	TBTxProfileEdit
	TBTxProfileDeactivate

	TBTxWageOpen
	TBTxWageEdit

	TBTxExpenseOpen
	TBTxExpenseEdit

	TBTxCloseInvoices
)

func TxBytes(object interface{}, tb byte) []byte {
	data := wire.BinaryBytes(object)
	return append([]byte{tb}, data...)
}

type Profile struct {
	Address         bcmd.Address  //identifier for querying
	Name            string        //identifier for querying
	AcceptedCur     string        //currency you will accept payment in
	DepositInfo     string        //default deposit information (mostly for fiat)
	DueDurationDays int           //default duration until a sent invoice due date
	Timezone        time.Location //default duration until a sent invoice due date
}

func NewProfile(Address bcmd.Address, Name, AcceptedCur, DepositInfo string,
	DueDurationDays int, Timezone time.Location) *Profile {
	return &Profile{
		Address:         Address,
		Name:            Name,
		AcceptedCur:     AcceptedCur,
		DepositInfo:     DepositInfo,
		DueDurationDays: DueDurationDays,
		Timezone:        Timezone,
	}
}

//////////////////////////////////////////////////////////////////////

// +gen holder:"Invoice,Impl[*Wage,*Expense]"
type InvoiceInner interface {
	SetID()
	GetID() []byte
	GetCtx() Context
	Close(close *CloseInvoice)
}

//for checking errors at compile time
var _ InvoiceInner = new(Wage)
var _ InvoiceInner = new(Expense)

type Wage struct {
	ID  []byte
	Ctx Context
}

//struct used for hash to determine ID
type Context struct {
	Open        bool
	Sender      string
	Receiver    string
	DepositInfo string
	Notes       string
	Amount      *AmtCurTime
	AcceptedCur string
	Due         time.Time
}

func NewWage(ID []byte, Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurTime, AcceptedCur string, Due time.Time) *Wage {

	return &Wage{
		ID: ID,
		Ctx: Context{
			Open:        true,
			Sender:      Sender,
			Receiver:    Receiver,
			DepositInfo: DepositInfo,
			Notes:       Notes,
			Amount:      Amount,
			AcceptedCur: AcceptedCur,
			Due:         Due,
		},
	}
}

func (w *Wage) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(w.Ctx)
	w.ID = append([]byte{TBIDWage}, hashBytes...)
}

func (w *Wage) GetID() []byte {
	return w.ID
}

func (w *Wage) GetCtx() Context {
	return w.Ctx
}

func (w *Wage) Close(close *CloseInvoice) {
	w.TransactionID = close.TransactionID
	w.PaymentCurTime = close.PaymentCurTime
}

type Expense struct {
	ID           []byte
	Ctx          Context
	Document     []byte
	DocFileName  string
	ExpenseTaxes *AmtCurTime
}

func NewExpense(ID []byte, Sender, Receiver, DepositInfo, Notes string,
	Amount *AmtCurTime, AcceptedCur string, Due time.Time,
	Document []byte, DocFileName string, ExpenseTaxes *AmtCurTime) *Expense {

	return &Expense{
		ID: ID,
		Ctx: Context{
			Open:        true,
			Sender:      Sender,
			Receiver:    Receiver,
			DepositInfo: DepositInfo,
			Notes:       Notes,
			Amount:      Amount,
			AcceptedCur: AcceptedCur,
			Due:         Due,
		},
		Document:     Document,
		DocFileName:  DocFileName,
		ExpenseTaxes: ExpenseTaxes,
	}
}

func (e *Expense) SetID() {
	hashBytes := merkle.SimpleHashFromBinary(e.Ctx)
	e.ID = append([]byte{TBIDExpense}, hashBytes...)
}

func (e *Expense) GetID() []byte {
	return e.ID
}

func (e *Expense) GetCtx() Context {
	return e.Ctx
}

func (e *Expense) Close(close *CloseInvoice) {
	e.TransactionID = close.TransactionID
	e.PaymentCurTime = close.PaymentCurTime
}

/////////////////////////////////////////////////////////////////////////

type CloseInvoices struct {
	IDs            [][]byte    //list of ID's to close with transaction
	TransactionID  string      //empty when unpaid
	PaymentCurTime *AmtCurTime //currency used to pay invoice, empty when unpaid
}

func NewCloseInvoices(IDs []byte, TransactionID string, PaymentCurTime *AmtCurTime) *CloseInvoices {
	return &CloseInvoices{
		ID:             ID,
		TransactionID:  TransactionID,
		PaymentCurTime: PaymentCurTime,
	}
}
