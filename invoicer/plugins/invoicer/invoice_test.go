package invoicer

import (
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/basecoin-examples/invoicer/types"
	wire "github.com/tendermint/go-wire"
)

func TestRunInvoice(t *testing.T) {
	require := require.New(t)

	amt, err := types.ParseAmtCurTime("100BTC", time.Now())
	require.Nil(err)

	var invoice types.Invoice

	invoice = types.NewContract(
		nil,
		"foo", //from
		"bar", //to
		"deposit info",
		"notes",
		amt,
		"btc",
		time.Now().Add(time.Hour*100),
	).Wrap()

	//txBytes := types.TxBytes(invoice, 0x01)
	txBytes := types.TxBytes(invoice, 0x01)
	//txBytes := wire.BinaryBytes(struct{ types.Invoice }{invoice})

	var invoiceRead = new(types.Invoice)

	//err = wire.ReadBinaryBytes(txBytes, invoiceRead)
	err = wire.ReadBinaryBytes(txBytes[1:], invoiceRead)
	require.Nil(err)
	require.False(invoiceRead.Empty())
	_, ok := invoiceRead.Unwrap().(*types.Contract)
	require.True(ok)
}

func TestReadWrite(t *testing.T) {
	require := require.New(t)

	imgPath := "/home/riger/Desktop/test.png"
	_, filename := path.Split(imgPath)
	savePath := path.Join("/home/riger/Desktop/rec", filename)

	docBytes, err := ioutil.ReadFile(imgPath)
	require.Nil(err)

	err = ioutil.WriteFile(savePath, docBytes, 0644)
	require.Nil(err)
}
