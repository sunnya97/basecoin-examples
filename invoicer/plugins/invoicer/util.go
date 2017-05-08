package invoicer

import (
	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func ProfileKey(name string) []byte {
	return []byte(cmn.Fmt("%v,Profile=%v", InvoicerName, name))
}

func InvoicerKey(ID []byte) []byte {
	return []byte(cmn.Fmt("%v,ID=%x", InvoicerName, ID))
}

func ListProfilesKey() []byte {
	return []byte(cmn.Fmt("%v,Profiles", InvoicerName))
}

func ListInvoicesKey() []byte {
	return []byte(cmn.Fmt("%v,Invoices", InvoicerName))
}

func ListExpensesKey() []byte {
	return []byte(cmn.Fmt("%v,Expenses", InvoicerName))
}

//Get objects from query bytes

func GetProfileFromWire(bytes []byte) (profile types.Profile, err error) {
	out, err := getFromWire(bytes, profile)
	return out.(types.Profile), err
}

func GetInvoiceFromWire(bytes []byte) (invoice types.Invoice, err error) {
	out, err := getFromWire(bytes, invoice)
	return out.(types.Invoice), err
}

func GetExpenseFromWire(bytes []byte) (expense types.Expense, err error) {
	out, err := getFromWire(bytes, expense)
	return out.(types.Expense), err
}

func GetListProfilesFromWire(bytes []byte) (profiles []types.Profile, err error) {
	out, err := getFromWire(bytes, profiles)
	return out.([]types.Profile), err
}

func GetListInvoicesFromWire(bytes []byte) (invoices []types.Invoice, err error) {
	out, err := getFromWire(bytes, invoices)
	return out.([]types.Invoice), err
}

func GetListExpensesFromWire(bytes []byte) (expenses []types.Expense, err error) {
	out, err := getFromWire(bytes, expenses)
	return out.([]types.Expense), err
}

func getFromWire(bytes []byte, destination interface{}) (interface{}, error) {
	var err error

	//Determine if the object already exists and load
	if len(bytes) > 0 { //is there a record of the object existing?
		err = wire.ReadBinaryBytes(bytes, &destination)
		if err != nil {
			err = abci.ErrInternalError.AppendLog("Error decoding state: " + err.Error())
		}
	} else {
		err = abci.ErrInternalError.AppendLog("state not found")
	}
	return destination, err
}

//Get objects directly from the store

func getProfile(store btypes.KVStore, name string) (profile types.Profile, err error) {
	bytes := store.Get(ProfileKey(name))
	return GetProfileFromWire(bytes)
}

func getInvoice(store btypes.KVStore, ID []byte) (invoice types.Invoice, err error) {
	bytes := store.Get(InvoicerKey(ID))
	return GetInvoiceFromWire(bytes)
}

func getExpense(store btypes.KVStore, ID []byte) (expense types.Expense, err error) {
	bytes := store.Get(InvoicerKey(ID))
	return GetExpenseFromWire(bytes)
}

func getListProfiles(store btypes.KVStore) (profiles []types.Profile, err error) {
	bytes := store.Get(ListProfilesKey())
	return GetListProfilesFromWire(bytes)
}
