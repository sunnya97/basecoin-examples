package invoicer

import (
	"fmt"
	"time"

	abci "github.com/tendermint/abci/types"
	types "github.com/tendermint/basecoin-examples/invoicer/types"
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
	case ctx.Payable == nil:
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
	case len(payment.InvoiceIDs) == 0:
		return abci.ErrInternalError.AppendLog("Closer doesn't contain any IDs to close!")
	case len(payment.TransactionID) == 0:
		return abci.ErrInternalError.AppendLog("Closer must include a transaction ID")
	}

	//Get all invoices, verify the ID
	var invoices []*types.Invoice
	for _, invoiceID := range payment.InvoiceIDs {
		invoice, err := getInvoice(store, invoiceID)
		if err != nil {
			return abciErrInvoiceMissing
		}
		invoices = append([]*types.Invoice{&invoice}, invoices...)
		if invoice.GetCtx().Receiver != payment.Receiver {
			return abci.ErrInternalError.AppendLog(
				fmt.Sprintf("Invoice ID %x has receiver %v but the payment is to receiver %v!",
					invoice.GetID(),
					invoice.GetCtx().Receiver,
					payment.Receiver))
		}
	}

	//Make sure that the invoice is not paying too much!
	var totalCost *types.AmtCurTime
	for _, invoice := range invoices {
		unpaid, err := invoice.GetCtx().Unpaid()
		if err != nil {
			return abciErrDecimal(err)
		}
		totalCost, err = totalCost.Add(unpaid)
		if err != nil {
			return abciErrDecimal(err)
		}
	}
	gt, err := payment.PaymentCurTime.GT(totalCost)
	if err != nil {
		return abciErrDecimal(err)
	}
	if gt {
		return abciErrOverPayment
	}

	//calculate and write changes to the set of all invoices
	bal := payment.PaymentCurTime
	for _, invoice := range invoices {
		//pay the funds to the invoice, reduce funds from bal
		invoice.GetCtx().Pay(bal) //TODO write test case here!
		store.Set(InvoiceKey(invoice.GetID()), wire.BinaryBytes(invoice))
	}

	//add the payment object to the store
	store.Set(InvoiceKey(payment.ID), wire.BinaryBytes(payment))
	payments, err := getListBytes(store, ListPaymentKey())
	if err != nil {
		return abciErrGetPayments
	}
	payments = append(payments, payment.ID)
	store.Set(ListPaymentKey(), wire.BinaryBytes(payments))

	return abci.OK
}
