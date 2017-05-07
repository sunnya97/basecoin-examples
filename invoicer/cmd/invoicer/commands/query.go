package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	bcmd "github.com/tendermint/basecoin/cmd/commands"
	"github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
)

var (
	//commands
	QueryInvoiceCmd = &cobra.Command{
		Use:   "invoice [hexID]",
		Short: "Query an invoice by invoice ID",
		RunE:  queryInvoiceCmd,
	}

	QueryInvoicesCmd = &cobra.Command{
		Use:   "invoices",
		Short: "Query all invoice",
		RunE:  queryInvoicesCmd,
	}
)

func init() {
	//register flags
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.Bool(downloadExp, false, "download expenses pdfs to the relative path specified")

	QueryInvoiceCmd.AddFlagSet(fs)

	fs.String(FlagTo, "", "Destination address for the bits")
	fs.Int(FlagNum, 0, "number of results to display, use 0 for no limit")
	fs.Bool(FlagShort, false, "output fields: paid, amount, date, sender, receiver")
	fs.String(FlagType, "",
		"limit the scope by using any of the following modifiers with commas: invoice,expense,paid,unpaid")
	fs.String(FlagDate, "",
		"Query within the date range start:end, where start/end are in the format YYYY-MM-DD, or empty. ex. --date 1991-10-21:")
	fs.String(FlagFrom, "", "only query for invoices from these addresses in the format <ADDR1>,<ADDR2>, etc.")
	fs.String(FlagTo, "", "only query for invoices to these addresses in the format <ADDR1>,<ADDR2>, etc.")

	QueryInvoicesCmd.AddFlagSet(fs)

	//register commands
	bcmd.RegisterQuerySubcommand(QueryInvoicesCmd)
	bcmd.RegisterQuerySubcommand(QueryInvoiceCmd)
}

func queryInvoiceCmd(cmd *cobra.Command, args []string) error {

	//get the issue, generate issue key
	if len(args) != 1 {
		return fmt.Errorf("query command requires an argument ([hexID])") //never stack trace
	}
	if !isHex(args[0]) {
		return fmt.Errorf("HexID is not formatted correctly") //never stack trace
	}
	id := StripHex(args[0])
	key := invoicer.InvoiceKey(id)

	//perform the query, get response
	resp, err := bcmd.Query(cmd.Parent().Flag("node").Value.String(), key) //TODO Upgrade to viper once basecoin viper upgrade complete
	if err != nil {
		return err
	}
	if !resp.Code.IsOK() {
		return errors.Errorf("Query for invoice key (%v) returned non-zero code (%v): %v",
			string(key), resp.Code, resp.Log)
	}

	//get the invoicer issue object and print it
	invoice, err := invoicer.GetInvoiceFromWire(resp.Value)
	if err != nil {
		return err
	}
	fmt.Println(string(wire.JSONBytes(invoice)))
	return nil
}

func queryInvoicesCmd(cmd *cobra.Command, args []string) error {
	return nil
}
