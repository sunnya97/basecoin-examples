package invoicer

import (
	abci "github.com/tendermint/abci/types"
	btypes "github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/basecoin-examples/invoicer/types"
)

func validateProfile(profile types.Profile) abci.Result {
	switch {
	case len(profile.Name) == 0:
		return abci.ErrInternalError.AppendLog("new profile must have a name")
	case len(profile.AcceptedCur) == 0:
		return abci.ErrInternalError.AppendLog("new profile must have an accepted currency")
	case profile.DueDurationDays < 0:
		return abci.ErrInternalError.AppendLog("new profile due duration must be non-negative")
	default:
		return abci.OK
	}
}

func writeProfile(store btypes.KVStore, active []string, profile types.Profile) {

	store.Set(ProfileKey(profile.Name), wire.BinaryBytes(profile))

	//also add it to the list of open profiles
	active = append(active, profile.Name)
	store.Set(ListProfileKey(), wire.BinaryBytes(active))
}

//TODO remove this once replaced KVStore functionality
func profileIsActive(active []string, name string) bool {
	for _, p := range active {
		if p == name {
			return true
		}
	}
	return false
}

func removeProfile(store btypes.KVStore, active []string, name string) {

	//remove profile XXX can't delete store entry on current KVstore implementation
	store.Set(ProfileKey(name), nil)

	//remove from the active profile list
	for i, v := range active {
		if v == name {
			active = append(active[:i], active[i+1:]...)
			break
		}
	}
	store.Set(ListProfileKey(), wire.BinaryBytes(active))
}

func runTxProfileOpen(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var profile types.Profile
	err := wire.ReadBinaryBytes(txBytes, &profile)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Validate Tx
	res = validateProfile(profile)
	if res.IsErr() {
		return res
	}

	//getprofile list
	active, err := getListProfile(store)
	if err != nil {
		return abci.ErrInternalError.AppendLog("error retrieving active profiles")
	}

	if profileIsActive(active, profile.Name) {
		return abci.ErrInternalError.AppendLog("Cannot create an already existing Profile")
	}

	writeProfile(store, active, profile)
	return abci.OK
}

func runTxProfileEdit(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var profile types.Profile
	err := wire.ReadBinaryBytes(txBytes, &profile)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Check if profile is active
	active, err := getListProfile(store)
	if err != nil {
		return abci.ErrInternalError.AppendLog("error retrieving active profile list")
	}
	if !profileIsActive(active, profile.Name) {
		return abci.ErrInternalError.AppendLog("Cannot edit a non-existing Profile")
	}

	//Validate Tx
	res = validateProfile(profile)
	if res.IsErr() {
		return res
	}

	writeProfile(store, active, profile)
	return abci.OK
}

func runTxProfileClose(store btypes.KVStore, ctx btypes.CallContext, txBytes []byte) (res abci.Result) {

	// Decode tx
	var name string
	err := wire.ReadBinaryBytes(txBytes, &name)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	//Check if profile is active
	profiles, err := getListProfile(store)
	if err != nil {
		return abci.ErrInternalError.AppendLog("error retrieving active profile list")
	}
	if !profileIsActive(profiles, name) {
		return abci.ErrInternalError.AppendLog("Cannot edit a non-existing Profile")
	}

	return abci.OK
}
