package invoicer

import (
	"time"

	"github.com/shopspring/decimal"
	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
)

func validatePayment(ctx types.Context) abci.Result {
	//Validate Tx
	switch {
	case len(ctx.Sender) == 0:
		return abci.ErrInternalError.AppendLog("Invoice must have a sender")
	case len(ctx.Receiver) == 0:
		return abci.ErrInternalError.AppendLog("Invoice must have a receiver")
	case len(ctx.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("Invoice must have an accepted currency")
	case ctx.Amount == nil:
		return abci.ErrInternalError.AppendLog("Invoice amount is nil")
	case ctx.Due.Before(time.Now()):
		return abci.ErrInternalError.AppendLog("Cannot issue overdue invoice")
	default:
		return abci.OK
	}
}

func runTxPayment(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var payment = new(types.Payment)
	err := wire.ReadBinaryBytes(txBytes, payment)
	if err != nil {
		return abciErrDecodingTX(err)
	}

	//Validate Tx
	switch {
	case len(close.IDs) == 0:
		return abci.ErrInternalError.AppendLog("Closer doesn't contain any IDs to close!")
	case len(close.TransactionID) == 0:
		return abci.ErrInternalError.AppendLog("Closer must include a transaction ID")
	}

	//Get all invoices
	var invoices []*types.Invoice
	for _, invoiceID := range payment.InvoiceIDs {
		invoice, err := getInvoice(store, payment.ID)
		if err != nil {
			return abciErrInvoiceMissing
		}
		invoices = append(invoice, invoices...)
	}

	//Make sure that the invoice is not paying too much!
	var totalCost decimal.Decimal
	for _, invoice := range invoices {
		totalCost = totalCost.Add(invoice.Unpaid)
	}
	if payment.PaymentCurTime.GT(totalCost) {
		return abciErrOverPayment
	}

	//calculate and write changes to the set of all invoices
	bal := payment.PaymentCurTime
	for i, invoiceID := range payment.InvoiceIDs {

		//get the invoice
		invoice, err := getInvoice(store, close.ID)
		if err != nil {
			return abciErrInvoiceMissing
		}

		//pay the funds to the invoice, reduce funds from bal
		invoice.GetCtx().Pay(bal) //TODO write test case here!

		store.Set(InvoiceKey(invoice.GetID()), wire.BinaryBytes(invoice))
	}

	//add the payment object to the store
	store.Set(InvoiceKey(payment.ID), wire.BinaryBytes(payment))
	payments, err := getListBytes(store, ListPaymentsKey())
	if err != nil {
		return abciErrGetPayments
	}
	payments = append(payments, payment.ID)
	store.Set(ListPaymentsKey(), wire.BinaryBytes(payments))

	return abci.OK
}
