package invoicer

import (
	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func ProfileKey() []byte {
	return []byte(cmn.Fmt("%v,Profiles", Name))
}

func InvoiceKey() []byte {
	return []byte(cmn.Fmt("%v,Invoices", Name))
}

//Get objects directly from the store

func getProfiles(store btypes.KVStore) (profile map[string]types.Profile, err error) {
	bytes := store.Get(ProfileKey())
	return GetProfileFromWire(bytes)
}

func getInvoices(store btypes.KVStore) (invoice map[string]types.Invoice, err error) {
	bytes := store.Get(InvoiceKey())
	return GetInvoiceFromWire(bytes)
}

//Get objects from query bytes

func GetProfilesFromWire(bytes []byte) (profile map[string]types.Profile, err error) {
	out, err := getFromWire(bytes, profile)
	return out.(map[string]types.Profile), err
}

func GetInvoicesFromWire(bytes []byte) (invoice map[string]types.Invoice, err error) {
	out, err := getFromWire(bytes, invoice)
	return out.(map[string]types.Invoice), err
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
