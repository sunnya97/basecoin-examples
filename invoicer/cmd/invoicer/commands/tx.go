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
	FlagCur                string = "Cur"
	FlagDefaultDepositInfo string = "info"
	FlagDueDurationDays    string = "due-days"
	FlagTimezone           string = "timezone"

	//invoice flags
	FlagReceiver    string = "to"
	FlagDepositInfo string = "info"
	FlagNotes       string = "notes"
	FlagDate        string = "date"
	FlagCur         string = "cur"
	FlagDue         string = "dur"

	//expense flags
	FlagReceipt   string = "receipt"
	FlagNotes     string = "notes"
	FlagTaxesPaid string = "taxes"

	//close flags
	FlagTransactionID  string = "id"
	FlagPaymentCurTime string = "cur"

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

	fsProfile := flag.NewFlagSet("", flag.ContinueOnError)
	fsProfile.String(FlagTo, "", "Destination address for the bits")
	fsProfile.String(FlagAcceptedCur, "btc", "currencies accepted for invoice payments")
	fsProfile.String(FlagDefaultDepositInfo, "", "default deposit information to be provided")
	fsProfile.String(FlagDueDurationDays, 14, "default number of days until invoice is due from invoice submission")
	fsProfile.String(FlagTimezone, "UTC", "timezone for invoice calculations")

	fsInvoice := flag.NewFlagSet("", flag.ContinueOnError)
	fsInvoice.String(FlagReceiver, "allinbits", "name of the invoice/expense receiver")
	fsInvoice.String(FlagDepositInfo, "", "deposit information for invoice payment (default: profile)")
	fsInvoice.String(FlagNotes, "", "notes regarding the expense")
	fsInvoice.String(FlagAmount, "", "invoice/expense amount in the format <decimal><currency> eg. 100.23usd")
	fsInvoice.String(FlagInvoiceDate, "", "invoice/expense date in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")
	fsInvoice.String(FlagTimezone, "", "invoice/expense timezone (default: profile)")
	fsInvoice.String(FlagCur, "btc", "currency which invoice/expense should be paid in")
	fsInvoice.String(FlagInvoiceDate, "", "invoice/expense due date in the format YYYY-MM-DD eg. 2016-12-31 (default: profile)")

	fsExpense := flag.NewFlagSet("", flag.ContinueOnError)
	fsExpense.String(FlagReceipt, "", "directory to receipt document file")
	fsExpense.String(FlagTaxesPaid, "", "taxes amount in the format <decimal><currency> eg. 10.23usd")

	fsClose := flag.NewFlagSet("", flag.ContinueOnError)
	fsClose.String(FlagTransactionID, "", "completed transaction ID")
	fsClose.String(FlagPaymentCurTime, "", "payment amount in the format <decimal><currency> eg. 10.23usd")
	fsClose.String(FlagPaymentDate, "", "date payment in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")

	NewProfileCmd.AddFlagSet(profileFlags)
	OpenInvoiceCmd.AddFlagSet(invoiceFlags)
	OpenExpenseCmd.AddFlagSet(invoiceFlags) //add invoice and expense flags here, intentional
	OpenExpenseCmd.AddFlagSet(expenseFlags)
	CloseCmd.AddFlagSet(closeFlags)

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

	timezone, err := time.LoadLocation(viper.GetString(flagTimezone))
	if err != nil {
		return fmt.Errorf("error loading timezone, error: ", err) //never stack trace
	}

	txBytes := types.NewTxBytesNewProfile(
		name,
		viper.GetString(flagAcceptedCur).(types.Currency),
		viper.GetString(flagDefaultDepositInfo),
		viper.GetString(flagDueDurationDays),
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

	date, err := types.ParseDate(viper.GetString(flagDate), viper.GetString(flagTimezone))
	if err != nil {
		return err
	}
	amt, err := types.ParseAmtCurDate(amountStr, date)
	if err != nil {
		return err
	}

	var dueDate time.Time
	if len(viper.GetString(flagDue)) > 0 {
		dueDate, err = types.ParseDate(viper.GetString(flagDue), viper.GetString(flagTimezone))
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
			viper.GetString(flagCur).(types.Currency),
		)
		return bcmd.AppTx(invoicerName, txBytes)
	}

	taxes, err := types.ParseAmtCurDate(viper.GetString(flagTaxesPaid), date)
	if err != nil {
		return err
	}
	docBytes, err := ioutil.ReadFile(viper.GetString(flagDocumentPath))
	if err != nil {
		return err
	}
	_, filename := path.Split(viper.GetString(flagDocumentPath))
	txBytes = NewTxBytesOpenExpense(
		sender,
		viper.GetString(flagReceiver),
		FlagDepositInfo,
		amt,
		viper.GetString(flagCur).(types.Currency),
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

	date, err := types.ParseDate(viper.GetString(flagDate), viper.GetString(flagTimezone))
	if err != nil {
		return err
	}
	cur := viper.GetString(flagCurrency).(types.Currency)

	txBytes := NewTxBytesClose(
		id,
		viper.GetString(flagTransactionID),
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
