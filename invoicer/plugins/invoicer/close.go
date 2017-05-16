package invoicer

import (
	"time"

	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func validateClose(ctx types.Context) abci.Result {
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

func runTxCloseInvoices(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var close = new(types.CloseInvoices)
	err := wire.ReadBinaryBytes(txBytes, close)
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

	//actually write the changes
	invoice, err := getInvoice(store, close.ID)
	if err != nil {
		return abciErrInvoiceMissing
	}
	invoice.Close(close)

	store.Set(InvoiceKey(invoice.GetID()), wire.BinaryBytes(invoice))

	return abci.OK
}
