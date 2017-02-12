package stake

import (
	"fmt"
	"strconv"

	abci "github.com/tendermint/abci/types"
	bcs "github.com/tendermint/basecoin/state"
	"github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire"
)

const denomATOM = "atom"

// Plugin is a proof-of-stake plugin for Basecoin
type Plugin struct {
	unbondingPeriod uint64 // how long unbonding takes (measured in blocks)
	height          uint64 // current block height
}

// Name returns the name of the stake plugin
func (sp Plugin) Name() string {
	return "stake"
}

// SetOption from ABCI
func (sp Plugin) SetOption(store types.KVStore, key string, value string) (log string) {
	if key == "unbondingPeriod" {
		var err error
		sp.unbondingPeriod, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			panic(fmt.Errorf("Could not parse int: '%s'", value))
		}
	}
	panic(fmt.Errorf("Unknown option key '%s'", key))
}

// RunTx from ABCI
func (sp Plugin) RunTx(store types.KVStore, ctx types.CallContext, txBytes []byte) (res abci.Result) {
	var tx Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return abci.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}

	_, isBondTx := tx.(BondTx)
	if isBondTx {
		return sp.runBondTx(tx.(BondTx), store, ctx)
	} else {
		return sp.runUnbondTx(tx.(UnbondTx), store, ctx)
	}
}

func (sp Plugin) runBondTx(tx BondTx, store types.KVStore, ctx types.CallContext) (res abci.Result) {
	// make sure collateral was paid with "atom" denomination
	if len(ctx.Coins) != 1 || ctx.Coins[0].Denom != denomATOM {
		return abci.ErrInternalError.AppendLog("Invalid coins or denomination")
	}

	// get amount paid
	amount := ctx.Coins[0].Amount
	if amount <= 0 {
		return abci.ErrInternalError.AppendLog("Amount must be > 0")
	}

	// add newly-created collateral to list
	state := loadState(store)
	state.Collateral = state.Collateral.Add(Collateral{
		ValidatorPubKey: tx.ValidatorPubKey,
		Address:         ctx.CallerAddress,
		Amount:          uint64(amount),
	})
	saveState(store, state)

	return abci.OK
}

func (sp Plugin) runUnbondTx(tx UnbondTx, store types.KVStore, ctx types.CallContext) (res abci.Result) {
	if tx.Amount <= 0 {
		return abci.ErrInternalError.AppendLog("Unbond amount must be > 0")
	}

	state := loadState(store)
	coll, i := state.Collateral.Get(ctx.CallerAddress, tx.ValidatorPubKey)

	// fail if no collateral w/ this address+validator, or not enough bonded coins
	if coll == nil || coll.Amount < tx.Amount {
		return abci.ErrBaseInsufficientFunds.AppendLog("Could not unbond")
	}

	// subtract coins from collateral
	state.Collateral[i].Amount -= tx.Amount

	// create new unbond record
	state.Unbonding = append(state.Unbonding, Unbond{
		ValidatorPubKey: tx.ValidatorPubKey,
		Amount:          tx.Amount,
		Address:         ctx.CallerAddress,
		Height:          sp.height, // unbonds at `height + unbondingPeriod`
	})

	saveState(store, state)

	return abci.OK
}

// InitChain from ABCI
func (sp Plugin) InitChain(store types.KVStore, vals []*abci.Validator) {
	// create collateral for initial validators
	state := loadState(store)
	for _, v := range vals {
		state.Collateral.Add(Collateral{
			ValidatorPubKey: v.PubKey,
			Address:         crypto.Ripemd160(v.PubKey),
			Amount:          v.Power,
		})
	}
	saveState(store, state)
}

// BeginBlock from ABCI
func (sp *Plugin) BeginBlock(store types.KVStore, height uint64) {
	sp.height = height
	state := loadState(store)

	// Once collateral is done unbonding, pay out coins into
	// basecoin accounts
	unbonding := state.Unbonding
	for len(unbonding) > 0 && height-unbonding[0].Height < sp.unbondingPeriod {
		// remove first record from list
		unbond := unbonding[0]
		unbonding = unbonding[1:]

		// add unbonded coins to basecoin account
		account := bcs.GetAccount(store, unbond.Address)
		account.Balance = account.Balance.Plus(makeAtoms(unbond.Amount))
		bcs.SetAccount(store, unbond.Address, account)
	}

	saveState(store, state)
}

// EndBlock from ABCI
func (sp *Plugin) EndBlock(store types.KVStore, height uint64) abci.Validators {
	sp.height = height + 1
	return loadState(store).Collateral.Validators()
}

func loadState(store types.KVStore) *State {
	bytes := store.Get([]byte("state"))
	if len(bytes) == 0 {
		return &State{}
	}
	var state *State
	err := wire.ReadBinaryBytes(bytes, &state)
	if err != nil {
		panic(err)
	}
	return state
}

func saveState(store types.KVStore, state *State) {
	bytes := wire.BinaryBytes(state)
	store.Set([]byte("state"), bytes)
}

// returns a []Coin with a single coin of type ATOM
func makeAtoms(amount uint64) types.Coins {
	return types.Coins{
		types.Coin{
			Denom:  denomATOM,
			Amount: int64(amount),
		},
	}
}
