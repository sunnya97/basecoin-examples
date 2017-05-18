package commands

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"time"

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
		Use:   "invoice [id]",
		Short: "Query an invoice by ID",
		RunE:  queryInvoiceCmd,
	}

	QueryInvoicesCmd = &cobra.Command{
		Use:   "invoices",
		Short: "Query all invoice",
		RunE:  queryInvoicesCmd,
	}

	QueryProfileCmd = &cobra.Command{
		Use:   "profile [name]",
		Short: "Query a profile",
		RunE:  queryProfileCmd,
	}

	QueryProfilesCmd = &cobra.Command{
		Use:   "profiles",
		Short: "List all open profiles",
		RunE:  queryProfilesCmd,
	}

	QueryPaymentCmd = &cobra.Command{
		Use:   "payment [id]",
		Short: "List historical payment",
		RunE:  queryPaymentCmd,
	}

	QueryPaymentsCmd = &cobra.Command{
		Use:   "payments",
		Short: "List historical payments",
		RunE:  queryPaymentsCmd,
	}
)

func init() {
	//register flags
	fsDownload := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagDownloadExp, "", "download expenses pdfs to the relative path specified")

	fsMultiple := func(obj, searchModifiers string) *flag.FlagSet {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		fs.Int(FlagNum, 0, "number of results to display, use 0 for no limit")
		fs.String(FlagType, "",
			"limit the scope by using any of the following modifiers with commas: invoice,expense,paid,unpaid")
		fs.String(FlagDateRange, "",
			"Query within the date range start:end, where start/end are in the format YYYY-MM-DD, or empty. ex. --date 1991-10-21:")
		fs.String(FlagFrom, "", "Only query for "+obj+" from these addresses in the format <ADDR1>,<ADDR2>, etc.")
		fs.String(FlagTo, "", "Only query for "+obj+" to these addresses in the format <ADDR1>,<ADDR2>, etc.")
		return fs
	}

	QueryInvoiceCmd.Flags().AddFlagSet(fsDownload)
	QueryInvoicesCmd.Flags().AddFlagSet(fsDownload)
	QueryInvoicesCmd.Flags().String(FlagSums, false, "Sum invoice values by sender") //TODO add functionality
	QueryInvoicesCmd.Flags().AddFlagSet(fsMultiple("invoices", "invoice,expense,paid,unpaid"))
	QueryPaymentsCmd.Flags().AddFlagSet(fsMultiple("payments", "invoice,expense"))

	//register commands
	bcmd.RegisterQuerySubcommand(QueryInvoicesCmd)
	bcmd.RegisterQuerySubcommand(QueryInvoiceCmd)
	bcmd.RegisterQuerySubcommand(QueryProfileCmd)
	bcmd.RegisterQuerySubcommand(QueryProfilesCmd)
	bcmd.RegisterQuerySubcommand(QueryPaymentsCmd)
}

func queryInvoiceCmd(cmd *cobra.Command, args []string) error {

	if len(args) != 1 {
		return errCmdReqArg("id")
	}
	if !isHex(args[0]) {
		return errBadHexID
	}
	id, err := hex.DecodeString(StripHex(args[0]))
	if err != nil {
		return err
	}

	//get the invoicer object and print it
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	invoice, err := queryInvoice(tmAddr, id)
	if err != nil {
		return err
	}

	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(invoice))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(invoice)))
	}

	expense, isExpense := invoice.Unwrap().(*types.Expense)
	if isExpense {
		err = downloadExp(expense)
		if err != nil {
			return errors.Errorf("Problem writing receipt file %v", err)
		}
	}

	return nil
}

func queryInvoicesCmd(cmd *cobra.Command, args []string) error {
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	listInvoices, err := queryListBytes(tmAddr)
	if err != nil {
		return err
	}

	if len(listInvoices) == 0 {
		return fmt.Errorf("No save invoices to return") //never stack trace
	}

	//init flag variables
	from := viper.GetString(FlagFrom)
	froms := strings.Split(from, ",")
	to := viper.GetString(FlagTo)
	toes := strings.Split(to, ",")

	ty := viper.GetString(FlagType)
	contractFilt, expenseFilt, paidFilt, unpaidFilt := true, true, true, true
	if len(ty) > 0 {
		contractFilt, expenseFilt, paidFilt, unpaidFilt = false, false, false, false
		if strings.Contains(ty, "contract") {
			contractFilt = true
		}
		if strings.Contains(ty, "expense") {
			expenseFilt = true
		}
		if strings.Contains(ty, "paid") {
			paidFilt = true
		}
		if strings.Contains(ty, "unpaid") {
			unpaidFilt = true
		}
	}

	//get the date range to query
	dates := strings.Split(flagDateRange)
	var startDate, endDate *time.Time = nil, nil
	parseDate := func(date string) (time.Time, error) {
		if len(date) == 0 {
			return nil, nil
		} else {
			return types.ParseDate(dates[0])
		}
	}
	var err error
	startDate, err = parseDate(dates[0])
	if err != nil {
		return err
	}
	endDate, err = parseDate(dates[1])
	if err != nil {
		return err
	}

	//actually loop through the invoices and query out the valid ones
	var invoices []types.Invoice
	for _, id := range listInvoices {

		invoice, err := queryInvoice(tmAddr, id)
		if err != nil {
			return errors.Errorf("Bad invoice in active invoice list %v", err)
		}

		contract, isContract := invoice.Unwrap().(*types.Contract)
		expense, isExpense := invoice.Unwrap().(*types.Expense)

		var ctx types.Context
		var transactionID string
		switch {
		case isContract:
			ctx = contract.Ctx
			transactionID = contract.TransactionID
		case isExpense:
			ctx = expense.Ctx
			transactionID = expense.TransactionID
		}

		//skip record if out of the date range
		d := invoice.GetCtx().Invoiced.CurTime.Date
		if startDate < d || endDate > d {
			continue
		}

		//continue if doesn't have the sender specified in the from or to flag
		cont := false
		for _, from := range froms {
			if from != ctx.Sender {
				cont = true
				break
			}
		}
		for _, to := range toes {
			if to != ctx.Sender {
				cont = true
				break
			}
		}
		if cont {
			continue
		}

		//check the type filter flags
		if (contractFilt && !isContract) ||
			(expenseFilt && !isExpense) ||
			(paidFilt && !ctx.paid) ||
			(unpaidFilt && ctx.paid) {

			continue
		}

		if isExpense {
			err = downloadExp(expense)
			if err != nil {
				return errors.Errorf("problem writing reciept file %v", err)
			}
		}

		//all tests have passed so add to the invoices list
		invoices = append(invoices, invoice)

		//Limit the number of invoices retrieved
		maxInv := viper.GetInt(FlagNum)
		if len(invoices) > maxInv && maxInv > 0 {
			break
		}
	}

	//compute the sum if flag is set
	if viper.GetString(FlagSums) {
		var sum *AmtCurTime
		for _, invoice := range invoices {
			sum = sum.Add(invoice.GetCtx().Unpaid())
		}
		out := struct {
			finalInvoice types.Invoice
			sumDue       *AmtCurTime
		}{
			invoices[len(invoices)-1],
			sum,
		}

		switch viper.GetString("output") {
		case "text":
			fmt.Println(string(wire.JSONBytes(out))) //TODO Actually make text
		case "json":
			fmt.Println(string(wire.JSONBytes(out)))
		}
		return nil
	}

	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(invoices))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(invoices)))
	}
	return nil
}

func downloadExp(expense *types.Expense) error {
	savePath := viper.GetString(FlagDownloadExp)
	if len(savePath) > 0 {
		savePath = path.Join(savePath, expense.DocFileName)
		err := ioutil.WriteFile(savePath, expense.Document, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func queryProfileCmd(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errCmdReqArg("name")
	}

	name := args[0]

	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	profile, err := queryProfile(tmAddr, name)
	if err != nil {
		return err
	}
	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(profile))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(profile)))
	}
	return nil
}

func queryProfilesCmd(cmd *cobra.Command, args []string) error {

	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()

	listProfiles, err := queryListString(tmAddr)
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

func queryPaymentCmd(cmd *cobra.Command, args []string) error {

	if len(args) != 1 {
		return errCmdReqArg("id")
	}
	if !isHex(args[0]) {
		return errBadHexID
	}
	id, err := hex.DecodeString(StripHex(args[0]))
	if err != nil {
		return err
	}

	//get the invoicer object and print it
	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()
	invoice, err := queryPayment(tmAddr, id)
	if err != nil {
		return err
	}

	switch viper.GetString("output") {
	case "text":
		fmt.Println(string(wire.JSONBytes(invoice))) //TODO Actually make text
	case "json":
		fmt.Println(string(wire.JSONBytes(invoice)))
	}
	return nil
}

func queryPaymentsCmd(cmd *cobra.Command, args []string) error {

	//TODO Upgrade to viper once basecoin viper upgrade complete
	tmAddr := cmd.Parent().Flag("node").Value.String()

	listPayments, err := queryListBytes(tmAddr)
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

func queryProfile(tmAddr, name string) (profile types.Profile, err error) {

	if len(name) == 0 {
		return invoice, errors.New("invalid query id")
	}
	key := invoicer.ProfileKey(name)

	res, err := query(tmAddr, key)
	if err != nil {
		return profile, err
	}

	return invoicer.GetProfileFromWire(res)
}

func queryInvoice(tmAddr string, id []byte) (invoice types.Invoice, err error) {

	if len(id) == 0 {
		return invoice, errors.New("invalid query id")
	}

	key := invoicer.InvoiceKey(id)
	res, err := query(tmAddr, key)
	if err != nil {
		return invoice, err
	}

	return invoicer.GetInvoiceFromWire(res)
}

func queryPayment(tmAddr string, id []byte) (payment types.Payment, err error) {

	if len(id) == 0 {
		return invoice, errors.New("invalid query id")
	}

	key := invoicer.PaymentKey(id)
	res, err := query(tmAddr, key)
	if err != nil {
		return payment, err
	}

	return invoicer.GetInvoiceFromWire(res)
}

func queryListString(tmAddr string) (profile []string, err error) {
	key := invoicer.ListProfileKey()
	res, err := query(tmAddr, key)
	if err != nil {
		return profile, err
	}
	return invoicer.GetListStringFromWire(res)
}

func queryListBytes(tmAddr string) (invoice [][]byte, err error) {
	key := invoicer.ListInvoiceKey()
	res, err := query(tmAddr, key)
	if err != nil {
		return invoice, err
	}
	return invoicer.GetListBytesFromWire(res)
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
