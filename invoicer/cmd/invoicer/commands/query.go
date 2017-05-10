package commands

import (
	"encoding/hex"
	"fmt"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	bcmd "github.com/tendermint/basecoin/cmd/commands"
	"github.com/tendermint/go-wire"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	"github.com/tendermint/basecoin-examples/invoicer/types"
)

var (
	//commands
	QueryInvoiceCmd = &cobra.Command{
		Use:   "invoice [hexID]",
		Short: "Query an invoice by ID",
		RunE:  queryInvoiceCmd,
	}

	QueryInvoicesCmd = &cobra.Command{
		Use:   "invoices",
		Short: "Query all invoice",
		RunE:  queryInvoicesCmd,
	}

	QueryProfilesCmd = &cobra.Command{
		Use:   "profiles",
		Short: "List all open profiles",
		RunE:  queryInvoicesCmd,
	}
)

func init() {
	//register flags
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagDownloadExp, "", "download expenses pdfs to the relative path specified")

	QueryInvoiceCmd.Flags().AddFlagSet(fs)

	fs.Int(FlagNum, 0, "number of results to display, use 0 for no limit")
	fs.Bool(FlagShort, false, "output fields: paid, amount, date, sender, receiver")
	fs.String(FlagType, "",
		"limit the scope by using any of the following modifiers with commas: invoice,expense,paid,unpaid")
	fs.String(FlagDate, "",
		"Query within the date range start:end, where start/end are in the format YYYY-MM-DD, or empty. ex. --date 1991-10-21:")
	fs.String(FlagFrom, "", "only query for invoices from these addresses in the format <ADDR1>,<ADDR2>, etc.")
	fs.String(FlagTo, "", "only query for invoices to these addresses in the format <ADDR1>,<ADDR2>, etc.")

	QueryInvoicesCmd.Flags().AddFlagSet(fs)

	//register commands
	bcmd.RegisterQuerySubcommand(QueryInvoicesCmd)
	bcmd.RegisterQuerySubcommand(QueryInvoiceCmd)
}

func queryInvoiceCmd(cmd *cobra.Command, args []string) error {

	if len(args) != 1 {
		return fmt.Errorf("query command requires an argument ([hexID])") //never stack trace
	}
	if !isHex(args[0]) {
		return fmt.Errorf("HexID is not formatted correctly") //never stack trace
	}
	id, err := hex.DecodeString(StripHex(args[0]))
	if err != nil {
		return err
	}

	//get the invoicer object and print it
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	invoice, err := queryInvoice(tmAddr, id) //TODO should also be querying for expense here :(
	if err != nil {
		return err
	}

	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(invoice))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(invoice)))
	}

	expense, ok := invoice.(types.Expense)
	if ok {
		savePath := viper.GetString(FlagDownloadExp)
		if len(savePath) > 0 {
			path.Join(savePath, expense.DocFileName)
		}
	}

	return nil
}

func queryInvoicesCmd(cmd *cobra.Command, args []string) error {
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	listInvoices, err := queryListInvoice(tmAddr)
	if err != nil {
		return err
	}

	_ = listInvoices
	//TODO actually get invoices/expenses and filter by flags

	//switch viper.GetString("output") {
	//case "text":
	//fmt.Println(string(wire.JSONBytes(listInvoices))) //TODO Actually make text
	//fmt.Println(string(wire.JSONBytes(listExpenses)))
	//case "json":
	//fmt.Println(string(wire.JSONBytes(listInvoices)))
	//fmt.Println(string(wire.JSONBytes(listExpenses)))
	//}
	return nil
}

func queryProfilesCmd(cmd *cobra.Command, args []string) error {
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	listProfiles, err := queryListProfile(tmAddr)
	if err != nil {
		return err
	}
	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(listProfiles))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(listProfiles)))
	}
	return nil
}

///////////////////////////////////////////////////////////////////

func queryProfile(tmAddr, name string) (invoice types.Profile, err error) {
	key := invoicer.ProfileKey(name)
	res, err := query(tmAddr, key)
	if err != nil {
		return invoice, err
	}
	return invoicer.GetProfileFromWire(res)
}

func queryInvoice(tmAddr string, id []byte) (invoice interface{}, err error) {

	if len(id) == 0 {
		return invoice, errors.New("invalid invoice query id")
	}

	key := invoicer.InvoiceKey(id)
	res, err := query(tmAddr, key)
	if err != nil {
		return invoice, err
	}

	switch id[0] {
	case types.TBIDExpense:
		return invoicer.GetExpenseFromWire(res)
	case types.TBIDInvoice:
		return invoicer.GetInvoiceFromWire(res)
	default: //should never occur if keys correctly written into the store
		return invoice, errors.New("invalid query id type")
	}
}

func queryListProfile(tmAddr string) (profile []string, err error) {
	key := invoicer.ListProfileKey()
	res, err := query(tmAddr, key)
	if err != nil {
		return profile, err
	}
	return invoicer.GetListProfileFromWire(res)
}

func queryListInvoice(tmAddr string) (invoice [][]byte, err error) {
	key := invoicer.ListInvoiceKey()
	res, err := query(tmAddr, key)
	if err != nil {
		return invoice, err
	}
	return invoicer.GetListInvoiceFromWire(res)
}

//Wrap the basecoin query function with a response code check
func query(tmAddr string, key []byte) ([]byte, error) {
	resp, err := bcmd.Query(tmAddr, key)
	if err != nil {
		return nil, err
	}
	if !resp.Code.IsOK() {
		return nil, errors.Errorf("Query for key (%v) returned non-zero code (%v): %v",
			string(key), resp.Code, resp.Log)
	}
	return resp.Value, nil
}
