package invoicer

import (
	"bytes"
	"time"

	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func runTxWageOpen(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var wage types.Wage
	err := wire.ReadBinaryBytes(txBytes, &wage)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate
	res = validateInvoiceCtx(wage.Ctx)
	if res.IsErr() {
		return res
	}

	//Set the id, then validate a bit more
	(&wage).SetID()

	if _, err := getProfile(store, wage.Ctx.Sender); err != nil {
		return abci.ErrInternalError.AppendLog("Senders Profile doesn't exist")
	}
	if _, err := getProfile(store, wage.Ctx.Receiver); err != nil {
		return abci.ErrInternalError.AppendLog("Receiver profile doesn't exist")
	}

	//Check if invoice already exists
	invoices, err := getListInvoice(store)
	for _, in := range invoices {
		if bytes.Compare(in, wage.ID) == 0 {
			return abci.ErrInternalError.AppendLog("Duplicate Invoice, edit the invoice notes to make them unique")
		}
	}

	//Store invoice
	store.Set(InvoiceKey(wage.ID), wire.BinaryBytes(wage))

	//also add it to the list of open invoices
	invoices = append(invoices, wage.ID)
	store.Set(ListInvoiceKey(), wire.BinaryBytes(invoices))
	return abci.OK
}

func validateInvoiceCtx(ctx types.Context) abci.Result {
	//Validate Tx
	switch {
	case len(ctx.Sender) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a sender")
	case len(ctx.Receiver) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have a receiver")
	case len(ctx.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("invoice must have an accepted currency")
	case ctx.Amount == nil:
		return abci.ErrInternalError.AppendLog("invoice amount is nil")
	case ctx.Due.Before(time.Now()):
		return abci.ErrInternalError.AppendLog("cannot issue overdue invoice")
	default:
		return abci.OK
	}
}

func runTxWageEdit(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {
	return abci.OK //TODO add functionality
}

func runTxExpenseOpen(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var expense types.Expense
	err := wire.ReadBinaryBytes(txBytes, &expense)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate
	res = validateInvoiceCtx(expense.Ctx)
	if res.IsErr() {
		return res
	}

	//Set the id, then validate a bit more
	(&expense).SetID()

	if _, err := getProfile(store, expense.Ctx.Sender); err != nil {
		return abci.ErrInternalError.AppendLog("Senders Profile doesn't exist")
	}

	if _, err := getProfile(store, expense.Ctx.Receiver); err != nil {
		return abci.ErrInternalError.AppendLog("Receiver profile doesn't exist")
	}

	//Return if the invoice already exists, aka no error was thrown
	if _, err := getInvoice(store, expense.ID); err == nil {
		return abci.ErrInternalError.AppendLog("Duplicate Invoice, edit the invoice notes to make them unique")
	}

	//Store profile
	store.Set(InvoiceKey(expense.ID), wire.BinaryBytes(expense))
	return abci.OK
}

func runTxExpenseEdit(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {
	return abci.OK //TODO add functionality
}

func runTxCloseInvoice(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var close types.CloseInvoice
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
		expense, err := getInvoice(store, close.ID)
		if err != nil {
			return abci.ErrInternalError.AppendLog("Expense ID is missing from existing expense")
		}
		store.Set(InvoiceKey(close.ID), wire.BinaryBytes(expense))
	case types.TBIDWage:
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
