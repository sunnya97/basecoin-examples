package invoicer

import (
	"github.com/shopspring/decimal"

	"github.com/tendermint/basecoin-examples/invoicer/types"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/basecoin/state"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
)

const invoicerName = "invoicer"

type Invoicer struct {
	name string
}

func New() *Invoicer {
	return &Invoicer{
		name: invoicerName,
	}
}

///////////////////////////////////////////////////

func newP2VIssue(issue string, feePerVote btypes.Coins) P2VIssue {
	return P2VIssue{
		Issue:        issue,
		FeePerVote:   feePerVote,
		VotesFor:     0,
		VotesAgainst: 0,
	}
}

func ProfileKey(name string) []byte {
	return []byte(cmn.Fmt("%v,Profile=%v", invoicerName, name))
}

func InvoicerKey(ID []byte) []byte {
	return []byte(cmn.Fmt("%v,ID=%x", invoicerName, ID))
}

//get an Invoice from store bytes
func GetProfileFromWire(profileBytes []byte) (profile Profile, err error) {
	out, err := getFromWire(profileBytes, profile)
	return out.(Profile), err
}

func GetInvoiceFromWire(invoiceBytes []byte) (invoice Invoice, err error) {
	out, err := getFromWire(invoiceBytes, invoice)
	return out.(Invoice), err
}

func GetExpenseFromWire(expenseBytes []byte) (expense Expense, err error) {
	out, err := getFromWire(expenseBytes, expense)
	return out.(Expense), err
}

func getFromWire(bytes []byte, destination interface{}) (interface{}, error) {
	var err error

	//Determine if the issue already exists and load
	if len(profileBytes) > 0 { //is there a record of the issue existing?
		err = wire.ReadBinaryBytes(profileBytes, &destination)
		if err != nil {
			err = abci.ErrInternalError.AppendLog("Error decoding state: " + err.Error())
		}
	} else {
		err = abci.ErrInternalError.AppendLog("state not found")
	}
	return in, err
}

func getProfile(store btypes.KVStore, name string) (profile Profile, err error) {
	bytes := store.Get(ProfileKey(name))
	return GetProfileFromWire(bytes)
}

func getInvoice(store btypes.KVStore, ID []byte) (invoice Invoice, err error) {
	bytes := store.Get(InvoicerKey(ID))
	return GetInvoiceFromWire(bytes)
}

func getExpense(store btypes.KVStore, ID []byte) (expense Expense, err error) {
	bytes := store.Get(ExpenseKey(ID))
	return GetExpenseFromWire(bytes)
}

///////////////////////////////////////////////////

func (inv *Invoicer) Name() string {
	return inv.name
}

func (inv *Invoicer) SetOption(store btypes.KVStore, key string, value string) (log string) {
	return ""
}

func (inv *Invoicer) RunTx(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	defer func() {
		//Return the ctx coins to the wallet if there is an error
		if res.IsErr() {
			acc := ctx.CallerAccount
			acc.Balance = acc.Balance.Plus(ctx.Coins)       // add the context transaction coins
			state.SetAccount(store, ctx.CallerAddress, acc) // save the new balance
		}
	}()

	//Determine the transaction type and then send to the appropriate transaction function
	if len(txBytes) < 1 {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: no tx bytes")
	}

	//Note that the zero position of txBytes contains the type-byte for the tx type
	//TODO add options for editing existing profile
	switch txBytes[0] {
	case types.TBTxNewProfile:
		return inv.runTxNewProfile(store, ctx, txBytes[1:])
	case types.TBTxOpenInvoice:
		return inv.runTxOpenInvoice(store, ctx, txBytes[1:])
	case types.TBTxOpenExpense:
		return inv.runTxOpenExpense(store, ctx, txBytes[1:])
	case types.TBTxClose:
		return inv.runTxClose(store, ctx, txBytes[1:])
	default:
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: bad prepended bytes")
	}
}

func (inv *Invoicer) runTxNewProfile(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var profile Profile
	err := wire.ReadBinaryBytes(txBytes, &profile)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	switch {
	case len(profile.Name) == 0:
		return abci.ErrInternalError.AppendLog("new profile must have a name")
	case len(profile.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("new profile must have an accepted currency")
	case DueDurationDays < 0:
		return abci.ErrInternalError.AppendLog("new profile due duration must be non-negative")
	}

	//Return if the issue already exists, aka no error was thrown
	if _, err := getProfile(store, profile.Name); err == nil {
		return abci.ErrInternalError.AppendLog("Cannot create an already existing Profile")
	}

	//Store profile
	store.Set(ProfileKey(profile.Name), wire.BinaryBytes(profile))
	return abci.OK
}

func (inv *Invoicer) runTxOpenInvoice(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var invoice Invoice
	err := wire.ReadBinaryBytes(txBytes, &invoice)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	switch {
	case len(invoice.Ctx.Sender) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a sender")
	case len(invoice.Ctx.Receiver) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a receiver")
	case len(invoice.Ctx.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have an accepted currency")
	case invoice.Ctx.Amount == nil:
		return abci.ErrInternalError.AppendLog("invoice amount is nil")
	case invoice.Ctx.Due.Before(time.Now):
		return abci.ErrInternalError.AppendLog("cannot issue overdue invoice")
	}

	(&invoice).SetID()

	if _, err := getProfile(store, invoice.Ctx.Sender); err != nil {
		return abci.ErrInternalError.AppendLog("Senders Profile doesn't exist")
	}

	if _, err := getProfile(store, invoice.Ctx.Receiver); err != nil {
		return abci.ErrInternalError.AppendLog("Receiver profile doesn't exist")
	}

	//Return if the invoice already exists, aka no error was thrown
	if _, err := getInvoice(store, invoice.ID); err == nil {
		return abci.ErrInternalError.AppendLog("Duplicate Invoice, edit the invoice notes to make them unique")
	}

	//Store profile
	store.Set(InvoicerKey(invoice.ID), wire.BinaryBytes(invoice))
	return abci.OK
}

func (inv *Invoicer) runTxOpenExpense(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var expense Expense
	err := wire.ReadBinaryBytes(txBytes, &expense)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	switch {
	case len(expense.Ctx.Sender) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a sender")
	case len(expense.Ctx.Receiver) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a receiver")
	case len(expense.Ctx.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have an accepted currency")
	case expense.Ctx.Amount == nil:
		return abci.ErrInternalError.AppendLog("invoice amount is nil")
	case expense.Ctx.Due.Before(time.Now):
		return abci.ErrInternalError.AppendLog("cannot issue overdue invoice")
	}

	(&expense).SetID()

	if _, err := getProfile(store, expense.Ctx.Sender); err != nil {
		return abci.ErrInternalError.AppendLog("Senders Profile doesn't exist")
	}

	if _, err := getProfile(store, expense.Ctx.Receiver); err != nil {
		return abci.ErrInternalError.AppendLog("Receiver profile doesn't exist")
	}

	//Return if the invoice already exists, aka no error was thrown
	if _, err := getExpense(store, expense.ID); err == nil {
		return abci.ErrInternalError.AppendLog("Duplicate Invoice, edit the invoice notes to make them unique")
	}

	//Store profile
	store.Set(InvoicerKey(expense.ID), wire.BinaryBytes(expense))
	return abci.OK
}

func (inv *Invoicer) runTxClose(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var close Close
	err := wire.ReadBinaryBytes(txBytes, &close)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	switch {
	case len(close.ID) == 0:
		return abci.ErrInternalError.AppendLog("closer doesn't have an ID")
	case len(close.TransactionID) == 0:
		return abci.ErrInternalError.AppendLog("closer must include a transaction ID")
	}

	//actually write the changes
	switch close.ID[0] {
	case TBIDExpense:
		invoice, err := getExpense(store, close.ID)
		if err != nil {
			return abci.ErrInternalError.AppendLog("Expense ID is missing from existing expense")
		}
		store.Set(InvoicerKey(close.ID), wire.BinaryBytes(expense))
	case TBIDInvoice:
		invoice, err := getInvoice(store, close.ID)
		if err != nil {
			return abci.ErrInternalError.AppendLog("Invoice ID is missing from existing invoices")
		}
		store.Set(InvoicerKey(close.ID), wire.BinaryBytes(invoice))
	default:
		return abci.ErrInternalError.AppendLog("ID Typebyte neither invoice nor expense")
	}

	return abci.OK
}

func (inv *Invoicer) InitChain(store btypes.KVStore, vals []*abci.Validator) {
}

func (inv *Invoicer) BeginBlock(store btypes.KVStore, hash []byte, header *abci.Header) {
}

func (inv *Invoicer) EndBlock(store btypes.KVStore, height uint64) (res abci.ResponseEndBlock) {
	return
}
