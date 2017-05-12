package commands

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	"github.com/tendermint/basecoin-examples/invoicer/types"
	bcmd "github.com/tendermint/basecoin/cmd/commands"
)

var (
	//commands
	InvoicerCmd = &cobra.Command{
		Use:   invoicer.Name,
		Short: "commands relating to invoicer system",
	}

	ProfileOpenCmd = &cobra.Command{
		Use:   "new-profile [name]",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  profileOpenCmd,
	}

	ProfileEditCmd = &cobra.Command{
		Use:   "new-profile [name]",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  profileEditCmd,
	}

	ProfileCloseCmd = &cobra.Command{
		Use:   "new-profile [name]",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  profileCloseCmd,
	}

	WageOpenCmd = &cobra.Command{
		Use:   "invoice [sender][amount]",
		Short: "send an invoice",
		RunE:  wageOpenCmd,
	}

	WageEditCmd = &cobra.Command{
		Use:   "invoice [sender][amount]",
		Short: "send an invoice",
		RunE:  wageEditCmd,
	}

	ExpenseOpenCmd = &cobra.Command{
		Use:   "expense [sender][amount]",
		Short: "send an expense",
		RunE:  expenseOpenCmd,
	}

	ExpenseEditCmd = &cobra.Command{
		Use:   "expense [sender][amount]",
		Short: "send an expense",
		RunE:  expenseEditCmd,
	}

	CloseInvoiceCmd = &cobra.Command{
		Use:   "close [ID]",
		Short: "close an invoice or expense",
		RunE:  closeInvoiceCmd,
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

	ProfileOpenCmd.Flags().AddFlagSet(fsProfile)
	WageOpenCmd.Flags().AddFlagSet(fsInvoice)
	ExpenseOpenCmd.Flags().AddFlagSet(fsInvoice) //intentional
	ExpenseOpenCmd.Flags().AddFlagSet(fsExpense)
	CloseInvoiceCmd.Flags().AddFlagSet(fsClose)

	//register commands
	InvoicerCmd.AddCommand(
		ProfileOpenCmd,
		WageOpenCmd,
		ExpenseOpenCmd,
		CloseInvoiceCmd,
	)
	bcmd.RegisterTxSubcommand(InvoicerCmd)
}

func profileOpenCmd(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("new-profile command requires an argument ([name])") //never stack trace
	}
	name := args[0]

	timezone, err := time.LoadLocation(viper.GetString(FlagTimezone))
	if err != nil {
		return fmt.Errorf("error loading timezone, error: ", err) //never stack trace
	}

	profile := types.NewProfile(
		name,
		viper.GetString(FlagCur),
		viper.GetString(FlagDepositInfo),
		viper.GetInt(FlagDueDurationDays),
		*timezone,
	)

	txBytes := profile.TxBytesOpen()

	return bcmd.AppTx(invoicer.Name, txBytes)
}

func profileEditCmd(cmd *cobra.Command, args []string) error {
	return nil //TODO implement
}

func profileCloseCmd(cmd *cobra.Command, args []string) error {
	return nil //TODO implement
}

func wageOpenCmd(cmd *cobra.Command, args []string) error {
	return openWageOrExpense(cmd, args, false)
}

func wageEditCmd(cmd *cobra.Command, args []string) error {
	return nil //TODO implement
}

func expenseOpenCmd(cmd *cobra.Command, args []string) error {
	return openWageOrExpense(cmd, args, true)
}

func expenseEditCmd(cmd *cobra.Command, args []string) error {
	return nil //TODO implement
}

func openWageOrExpense(cmd *cobra.Command, args []string, isExpense bool) error {
	if len(args) != 2 {
		return fmt.Errorf("Command requires two arguments ([sender][amount])") //never stack trace
	}
	sender := args[0]
	amountStr := args[1]

	profile, err := queryProfile(cmd.Parent().Flag("node").Value.String(), sender)
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

	var invoice types.Invoice

	switch isExpense {
	case false:
		invoice = types.NewWage(
			sender,
			viper.GetString(FlagTo),
			depositInfo,
			viper.GetString(FlagNotes),
			amt,
			accCur,
			dueDate,
		)
	case true:
		taxes, err := types.ParseAmtCurTime(viper.GetString(FlagTaxesPaid), date)
		if err != nil {
			return err
		}
		docBytes, err := ioutil.ReadFile(viper.GetString(FlagReceipt))
		if err != nil {
			return err
		}

		_, filename := path.Split(viper.GetString(FlagReceipt))
		invoice = types.NewExpense(
			sender,
			viper.GetString(FlagTo),
			depositInfo,
			viper.GetString(FlagNotes),
			amt,
			accCur,
			dueDate,
			docBytes,
			filename,
			taxes,
		)
	}

	txBytes := invoice.TxBytesOpen()
	return bcmd.AppTx(invoicer.Name, txBytes)
}

func closeInvoiceCmd(cmd *cobra.Command, args []string) error {
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

	closeInvoice := types.NewCloseInvoice(
		id,
		viper.GetString(FlagTransactionID),
		act,
	)
	txBytes := closeInvoice.TxBytes()
	return bcmd.AppTx(invoicer.Name, txBytes)
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
