package commands

//TODO
// edit open profile
// edit an unpaid invoice,
// bulk import from csv,
// JSON imports
// interoperability with ebuchman rates tool

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	"github.com/tendermint/basecoin-examples/invoicer/types"
	bcmd "github.com/tendermint/basecoin/cmd/commands"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/tmlibs/merkle"
)

const invoicerName = "invoicer"

var (
	//profile flags
	flagCur                string
	flagDefaultDepositInfo string
	flagDueDurationDays    int
	flagTimezone           string

	//invoice flags
	flagReceiver    string //hex
	flagDepositInfo string
	flagNotes       string
	flagAmount      string //AmtCurDate
	flagDate        string
	flagCur         string
	flagDue         string

	//expense flags
	flagReceiptFile string //hex
	flagNotes       string
	flagTaxesPaid   string //AmtCurDate

	//close flags
	flagTransactionID  string //empty when unpaid
	flagPaymentCurTime string //AmtCurDate

	//commands
	InvoicerCmd = &cobra.Command{
		Use:   "invoicer",
		Short: "commands relating to invoicer system",
	}

	NewProfileCmd = &cobra.Command{
		Use:   "new-profile [name]",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  newProfileCmd,
	}

	OpenInvoiceCmd = &cobra.Command{
		Use:   "invoice [sender][amount]",
		Short: "send an invoice",
		RunE:  openInvoiceCmd,
	}

	OpenExpenseCmd = &cobra.Command{
		Use:   "expense [sender][amount]",
		Short: "send an expense",
		RunE:  openExpenseCmd,
	}

	CloseCmd = &cobra.Command{
		Use:   "close [ID]",
		Short: "close an invoice or expense",
		RunE:  openExpenseCmd,
	}
)

func init() {

	//register flags
	//issueFlag2Reg := bcmd.Flag2Register{&issueFlag, "issue", "default issue", "name of the issue to generate or vote for"}

	profileFlags := []bcmd.Flag2Register{
		{&flagAcceptedCur, "cur", "btc", "currencies accepted for invoice payments"},
		{&flagDefaultDepositInfo, "deposit-info", "", "default deposit information to be provided"},
		{&flagDueDurationDays, "due-days", 14, "default number of days until invoice is due from invoice submission"},
		{&flagTimezone, "timezone", "UTC", "timezone for invoice calculations"},
	}

	invoiceFlags := []bcmd.Flag2Register{
		{&lagReceiver, "receiver", "allinbits", "name of the invoice/expense receiver"},
		{&flagDepositInfo, "deposit", "", "deposit information for invoice payment (default: profile)"},
		{&flagNotes, "notes", "", "notes regarding the expense"},
		{&flagAmount, "amount", "", "invoice/expense amount in the format <decimal><currency> eg. 100.23usd"},
		{&flagInvoiceDate, "date", "", "invoice/expense date in the format YYYY-MM-DD eg. 2016-12-31 (default: today)"},
		{&flagTimezone, "timezone", "", "invoice/expense timezone (default: profile)"},
		{&flagCur, "cur", "btc", "currency which invoice/expense should be paid in"},
		{&flagInvoiceDate, "due", "", "invoice/expense due date in the format YYYY-MM-DD eg. 2016-12-31 (default: profile)"},
	}

	expenseFlags := []bcmd.Flag2Register{
		{&flagPdfReceipt, "pdf", "", "directory to pdf document of receipt"},
		{&flagTaxesPaid, "taxes", "", "taxes amount in the format <decimal><currency> eg. 10.23usd"},
	}

	closeFlags := []bcmd.Flag2Register{
		{&flagTransactionID, "transaction", "", "completed transaction ID"},
		{&flagPaymentCurTime, "cur", "", "payment amount in the format <decimal><currency> eg. 10.23usd"},
		{&flagPaymentDate, "date", "", "date payment in the format YYYY-MM-DD eg. 2016-12-31 (default: today)"},
	}

	bcmd.RegisterFlags(NewProfileCmd, profileFlags)
	bcmd.RegisterFlags(OpenInvoiceCmd, invoiceFlags)
	bcmd.RegisterFlags(OpenExpenseCmd, invoiceFlags)
	bcmd.RegisterFlags(OpenExpenseCmd, expenseFlags)
	bcmd.RegisterFlags(CloseCmd, closeFlags)

	//register commands
	InvoicerCmd.AddCommand(
		NewProfileCmd,
		OpenInvoiceCmd,
		OpenExpenseCmd,
		CloseCmd,
	)
	bcmd.RegisterTxSubcommand(InvoicerCmd)
}

func newProfileCmd(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("new-profile command requires an argument ([name])") //never stack trace
	}
	name := args[0]

	timezone, err := time.LoadLocation(flagTimezone)
	if err != nil {
		return fmt.Errorf("error loading timezone, error: ", err) //never stack trace
	}

	txBytes := types.NewTxBytesNewProfile(
		name,
		flagAcceptedCur.(types.Currency),
		flagDefaultDepositInfo,
		flagDueDurationDays,
		timezone,
	)
	return bcmd.AppTx(InvoicerName, txBytes)
}

func getProfile(cmd *cobra.Command, name string) (profile Profile, err error) {

	key := invoicer.ProfileKey(name)

	//perform the query, get response
	resp, err := bcmd.Query(cmd.Parent().Flag("node").Value.String(), key)
	if err != nil {
		return
	}
	if !resp.Code.IsOK() {
		err = errors.Errorf("Query for invoice key (%v) returned non-zero code (%v): %v",
			string(key), resp.Code, resp.Log)
		return
	}

	return invoicer.GetProfileFromWire(resp.Value)
}

func openInvoiceCmd(cmd *cobra.Command, args []string) error {
	return openInvoiceOrExpense(cmd, args, false)
}

func openExpenseCmd(cmd *cobra.Command, args []string) error {
	return openInvoiceOrExpense(cmd, args, true)
}

func openInvoiceOrExpense(cmd *cobra.Command, args []string, isExpense bool) error {
	if len(args) != 2 {
		return fmt.Errorf("Command requires two arguments ([sender][amount])") //never stack trace
	}
	sender := args[0]
	amountStr := args[1]

	profile, err := getProfile(cmd, sender)
	if err != nil {
		return err
	}

	date, err := types.ParseDate(flagDate, flagTimezone)
	if err != nil {
		return err
	}
	amt, err := types.ParseAmtCurDate(amountStr, date)
	if err != nil {
		return err
	}

	var dueDate time.Time
	if len(flagDue) > 0 {
		dueDate, err = types.ParseDate(flagDue, flagTimezone)
		if err != nil {
			return err
		}
	} else {
		dueDate := time.Now().AddDate(0, 0, profile.DueDurationDays)
	}

	var depositInfo string
	if len(FlagDepositInfo) > 0 {
		depositInfo := FlagDepositInfo
	} else {
		depositInfo := profile.DepositInfo
	}

	var accCur types.Currency
	if len(FlagDepositInfo) > 0 {
		depositInfo := FlagDepositInfo
	} else {
		depositInfo := profile.DepositInfo
	}

	//if not an expense then we're almost done!
	if !isExpense {
		txBytes := NewTxBytesOpenInvoice(
			sender,
			FlagReceiver,
			depositInfo,
			FlagNotes,
			amt,
			flagCur.(types.Currency),
		)
		return bcmd.AppTx(invoicerName, txBytes)
	}

	taxes, err := types.ParseAmtCurDate(flagTaxesPaid, date)
	if err != nil {
		return err
	}
	docBytes, err := ioutil.ReadFile(flagDocumentPath)
	if err != nil {
		return err
	}
	_, filename := path.Split(flagDocumentPath)
	txBytes = NewTxBytesOpenExpense(
		sender,
		flagReceiver,
		FlagDepositInfo,
		amt,
		flagCur.(types.Currency),
		docbytes,
		filename,
		FlagDepositNotes,
		taxes,
	)

	return bcmd.AppTx(invoicerName, txBytes)
}

func closeCmd(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("close command requires an argument ([HexID])") //never stack trace
	}
	if !isHex(args[0]) {
		return fmt.Errorf("HexID is not formatted correctly") //never stack trace
	}
	id := StripHex(args[0])

	date, err := types.ParseDate(flagDate, flagTimezone)
	if err != nil {
		return err
	}
	cur := flagCurrency.(types.Currency)

	txBytes := NewTxBytesClose(
		id,
		flagTransactionID,
		types.CurDate{cur, date},
	)
	return bcmd.AppTx(invoicerName, txBytes)
}

//TODO Move to tmlibs/common
// Returns true for non-empty hex-string prefixed with "0x"
func isHex(s string) bool {
	if len(s) > 2 && s[:2] == "0x" {
		_, err := hex.DecodeString(s[2:])
		if err != nil {
			return false
		}
		return true
	}
	return false
}

func StripHex(s string) string {
	if isHex(s) {
		return s[2:]
	}
	return s
}
