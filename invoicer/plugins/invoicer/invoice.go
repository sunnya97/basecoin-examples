package invoicer

import (
	"bytes"
	"time"

	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func runTxOpenInvoice(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var invoice types.Invoice
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
	case invoice.Ctx.Due.Before(time.Now()):
		return abci.ErrInternalError.AppendLog("cannot issue overdue invoice")
	}

	(&invoice).SetID()

	if _, err := getProfile(store, invoice.Ctx.Sender); err != nil {
		return abci.ErrInternalError.AppendLog("Senders Profile doesn't exist")
	}
	if _, err := getProfile(store, invoice.Ctx.Receiver); err != nil {
		return abci.ErrInternalError.AppendLog("Receiver profile doesn't exist")
	}

	//Check if invoice already exists
	invoices, err := getListInvoice(store)
	for _, in := range invoices {
		if bytes.Compare(in, invoice.ID) == 0 {
			return abci.ErrInternalError.AppendLog("Duplicate Invoice, edit the invoice notes to make them unique")
		}
	}

	//Store invoice
	store.Set(InvoiceKey(invoice.ID), wire.BinaryBytes(invoice))

	//also add it to the list of open invoices
	invoices = append(invoices, invoice.ID)
	store.Set(ListInvoiceKey(), wire.BinaryBytes(invoices))
	return abci.OK
}

func runTxEditInvoice(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {
	return abci.OK //TODO add functionality
}

func runTxOpenExpense(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var expense types.Expense
	err := wire.ReadBinaryBytes(txBytes, &expense)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	switch {
	case len(expense.Ctx.Sender) == 0:
		return abci.ErrInternalError.AppendLog("expense must have a sender")
	case len(expense.Ctx.Receiver) == 0:
		return abci.ErrInternalError.AppendLog("expense must have a receiver")
	case len(expense.Ctx.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("expense must have an accepted currency")
	case expense.Ctx.Amount == nil:
		return abci.ErrInternalError.AppendLog("expense amount is nil")
	case expense.Ctx.Due.Before(time.Now()):
		return abci.ErrInternalError.AppendLog("cannot issue overdue expense")
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
	store.Set(InvoiceKey(expense.ID), wire.BinaryBytes(expense))
	return abci.OK
}

func runTxEditExpense(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {
	return abci.OK //TODO add functionality
}

func runTxClose(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var close types.Close
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
	case types.TBIDExpense:
		expense, err := getExpense(store, close.ID)
		if err != nil {
			return abci.ErrInternalError.AppendLog("Expense ID is missing from existing expense")
		}
		store.Set(InvoiceKey(close.ID), wire.BinaryBytes(expense))
	case types.TBIDInvoice:
		invoice, err := getInvoice(store, close.ID)
		if err != nil {
			return abci.ErrInternalError.AppendLog("Invoice ID is missing from existing invoices")
		}
		store.Set(InvoiceKey(close.ID), wire.BinaryBytes(invoice))
	default:
		return abci.ErrInternalError.AppendLog("ID Typebyte neither invoice nor expense")
	}

	return abci.OK
}

//TODO add JSON imports
func runTxBulkImport(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {
	return abci.OK //TODO add functionality
}
