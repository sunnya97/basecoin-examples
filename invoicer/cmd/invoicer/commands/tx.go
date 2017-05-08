package commands

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	"github.com/tendermint/basecoin-examples/invoicer/types"
	bcmd "github.com/tendermint/basecoin/cmd/commands"
)

const InvoicerName = "invoicer"

var (
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
	fsProfile := flag.NewFlagSet("", flag.ContinueOnError)
	fsProfile.String(FlagTo, "", "Destination address for the bits")
	fsProfile.String(FlagCur, "btc", "currencies accepted for invoice payments")
	fsProfile.String(FlagDepositInfo, "", "default deposit information to be provided")
	fsProfile.Int(FlagDueDurationDays, 14, "default number of days until invoice is due from invoice submission")
	fsProfile.String(FlagTimezone, "UTC", "timezone for invoice calculations")

	fsInvoice := flag.NewFlagSet("", flag.ContinueOnError)
	fsInvoice.String(FlagTo, "allinbits", "name of the invoice/expense receiver")
	fsInvoice.String(FlagDepositInfo, "", "deposit information for invoice payment (default: profile)")
	fsInvoice.String(FlagNotes, "", "notes regarding the expense")
	fsInvoice.String(FlagTimezone, "", "invoice/expense timezone (default: profile)")
	fsInvoice.String(FlagCur, "btc", "currency which invoice/expense should be paid in")
	fsInvoice.String(FlagDueDate, "", "invoice/expense due date in the format YYYY-MM-DD eg. 2016-12-31 (default: profile)")

	fsExpense := flag.NewFlagSet("", flag.ContinueOnError)
	fsExpense.String(FlagReceipt, "", "directory to receipt document file")
	fsExpense.String(FlagTaxesPaid, "", "taxes amount in the format <decimal><currency> eg. 10.23usd")

	fsClose := flag.NewFlagSet("", flag.ContinueOnError)
	fsClose.String(FlagTransactionID, "", "completed transaction ID")
	fsClose.String(FlagCur, "", "payment amount in the format <decimal><currency> eg. 10.23usd")
	fsClose.String(FlagDate, "", "date payment in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")

	NewProfileCmd.Flags().AddFlagSet(fsProfile)
	OpenInvoiceCmd.Flags().AddFlagSet(fsInvoice)
	OpenExpenseCmd.Flags().AddFlagSet(fsInvoice) //intentional
	OpenExpenseCmd.Flags().AddFlagSet(fsExpense)
	CloseCmd.Flags().AddFlagSet(fsClose)

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

	timezone, err := time.LoadLocation(viper.GetString(FlagTimezone))
	if err != nil {
		return fmt.Errorf("error loading timezone, error: ", err) //never stack trace
	}

	txBytes := types.NewTxBytesNewProfile(
		name,
		viper.GetString(FlagCur),
		viper.GetString(FlagDepositInfo),
		viper.GetInt(FlagDueDurationDays),
		*timezone,
	)
	return bcmd.AppTx(InvoicerName, txBytes)
}

func getProfile(cmd *cobra.Command, name string) (profile types.Profile, err error) {

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

	date, err := types.ParseDate(viper.GetString(FlagDate), viper.GetString(FlagTimezone))
	if err != nil {
		return err
	}
	amt, err := types.ParseAmtCurTime(amountStr, date)
	if err != nil {
		return err
	}

	var dueDate time.Time
	if len(viper.GetString(FlagDueDate)) > 0 {
		dueDate, err = types.ParseDate(viper.GetString(FlagDueDate), viper.GetString(FlagTimezone))
		if err != nil {
			return err
		}
	} else {
		dueDate = time.Now().AddDate(0, 0, profile.DueDurationDays)
	}

	var depositInfo string
	if len(viper.GetString(FlagDepositInfo)) > 0 {
		depositInfo = viper.GetString(FlagDepositInfo)
	} else {
		depositInfo = profile.DepositInfo
	}

	var accCur string
	if len(viper.GetString(FlagDepositInfo)) > 0 {
		accCur = viper.GetString(FlagCur)
	} else {
		accCur = profile.AcceptedCur
	}

	//if not an expense then we're almost done!
	if !isExpense {
		txBytes := types.NewTxBytesOpenInvoice(
			sender,
			viper.GetString(FlagTo),
			depositInfo,
			viper.GetString(FlagNotes),
			amt,
			accCur,
			dueDate,
		)
		return bcmd.AppTx(InvoicerName, txBytes)
	}

	taxes, err := types.ParseAmtCurTime(viper.GetString(FlagTaxesPaid), date)
	if err != nil {
		return err
	}
	docBytes, err := ioutil.ReadFile(viper.GetString(FlagReceipt))
	if err != nil {
		return err
	}

	//func NewTxBytesOpenExpense(Sender, Receiver, DepositInfo, Notes string,
	//Amount *AmtCurTime, AcceptedCur string, Due time.Time,
	//Document []byte, DocFileName string, TaxesPaid *AmtCurTime) []byte {

	_, filename := path.Split(viper.GetString(FlagReceipt))
	txBytes := types.NewTxBytesOpenExpense(
		sender,
		viper.GetString(FlagTo),
		depositInfo,
		viper.GetString(FlagNotes),
		amt,
		viper.GetString(FlagCur),
		dueDate,
		docBytes,
		filename,
		taxes,
	)

	return bcmd.AppTx(InvoicerName, txBytes)
}

func closeCmd(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("close command requires an argument ([HexID])") //never stack trace
	}
	if !isHex(args[0]) {
		return fmt.Errorf("HexID is not formatted correctly") //never stack trace
	}
	id, err := hex.DecodeString(StripHex(args[0]))
	if err != nil {
		return err
	}

	date, err := types.ParseDate(viper.GetString(FlagDate), viper.GetString(FlagTimezone))
	if err != nil {
		return err
	}
	act, err := types.ParseAmtCurTime(viper.GetString(FlagCur), date)
	if err != nil {
		return err
	}

	txBytes := types.NewTxBytesClose(
		id,
		viper.GetString(FlagTransactionID),
		act,
	)
	return bcmd.AppTx(InvoicerName, txBytes)
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
