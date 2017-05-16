package commands

//TODO build pubkey section to the profiles

import (
	"encoding/hex"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
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
		Short: "Commands relating to invoicer system",
	}

	ProfileOpenCmd = &cobra.Command{
		Use:   "profile-open [name]",
		Short: "Open a profile for sending/receiving invoices",
		RunE:  profileOpenCmd,
	}

	ProfileEditCmd = &cobra.Command{
		Use:   "profile-edit",
		Short: "Edit an existing profile",
		RunE:  profileEditCmd,
	}

	ProfileDeactivateCmd = &cobra.Command{
		Use:   "profile-deactivate",
		Short: "Deactivate and existing profile",
		RunE:  profileDeactivateCmd,
	}

	WageOpenCmd = &cobra.Command{
		Use:   "wage-open [amount]",
		Short: "Send a wage invoice of amount <value><currency>",
		RunE:  wageOpenCmd,
	}

	WageEditCmd = &cobra.Command{
		Use:   "wage-edit [amount]",
		Short: "Edit an open wage invoice to amount <value><currency>",
		RunE:  wageEditCmd,
	}

	ExpenseOpenCmd = &cobra.Command{
		Use:   "expense-open [amount]",
		Short: "Send an expense invoice of amount <value><currency>",
		RunE:  expenseOpenCmd,
	}

	ExpenseEditCmd = &cobra.Command{
		Use:   "expense-edit [amount]",
		Short: "Edit an open expense invoice to amount <value><currency>",
		RunE:  expenseEditCmd,
	}

	CloseInvoicesCmd = &cobra.Command{
		Use:   "close-invoices",
		Short: "Close invoices and expenses with transaction infomation",
		RunE:  closeInvoicesCmd,
	}
)

func init() {

	//register flags
	fsProfile := flag.NewFlagSet("", flag.ContinueOnError)
	fsProfile.String(FlagTo, "", "Who you're invoicing")
	fsProfile.String(FlagCur, "BTC", "Payment curreny accepted")
	fsProfile.String(FlagDepositInfo, "", "Default deposit information to be provided")
	fsProfile.Int(FlagDueDurationDays, 14, "Default number of days until invoice is due from invoice submission")
	fsProfile.String(FlagTimezone, "UTC", "Timezone for invoice calculations")

	fsInvoice := flag.NewFlagSet("", flag.ContinueOnError)
	fsInvoice.String(FlagTo, "allinbits", "Name of the invoice/expense receiver")
	fsInvoice.String(FlagDepositInfo, "", "Deposit information for invoice payment (default: profile)")
	fsInvoice.String(FlagNotes, "", "Notes regarding the expense")
	fsInvoice.String(FlagTimezone, "", "Invoice timezone (default: profile)")
	fsInvoice.String(FlagCur, "btc", "Currency which invoice/expense should be paid in")
	fsInvoice.String(FlagDueDate, "", "Invoice due date in the format YYYY-MM-DD eg. 2016-12-31 (default: profile)")

	fsExpense := flag.NewFlagSet("", flag.ContinueOnError)
	fsExpense.String(FlagReceipt, "", "Directory to receipt document file")
	fsExpense.String(FlagTaxesPaid, "", "Taxes amount in the format <decimal><currency> eg. 10.23usd")

	fsClose := flag.NewFlagSet("", flag.ContinueOnError)
	fsClose.String(FlagIDs, "", "IDs to close during this transaction <id1>,<id2>,<id3>... ")
	fsClose.String(FlagTransactionID, "", "Completed transaction ID")
	fsClose.String(FlagCur, "", "Payment amount in the format <decimal><currency> eg. 10.23usd")
	fsClose.String(FlagDate, "", "Date payment in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")
	fsClose.String(FlagDateRange, "",
		"Query within the date range start:end, where start/end are in the format YYYY-MM-DD, or empty. ex. --date 1991-10-21:")

	fsEdit := flag.NewFlagSet("", flag.ContinueOnError)
	fsEdit.String(FlagID, "", "ID (hex) of the invoice to modify")

	ProfileOpenCmd.Flags().AddFlagSet(fsProfile)
	ProfileEditCmd.Flags().AddFlagSet(fsProfile)

	WageOpenCmd.Flags().AddFlagSet(fsInvoice)
	WageEditCmd.Flags().AddFlagSet(fsInvoice)
	WageEditCmd.Flags().AddFlagSet(fsEdit)

	ExpenseOpenCmd.Flags().AddFlagSet(fsInvoice)
	ExpenseOpenCmd.Flags().AddFlagSet(fsExpense)
	ExpenseEditCmd.Flags().AddFlagSet(fsInvoice)
	ExpenseEditCmd.Flags().AddFlagSet(fsExpense)
	ExpenseEditCmd.Flags().AddFlagSet(fsEdit)

	CloseInvoiceCmd.Flags().AddFlagSet(fsClose)

	//register commands
	InvoicerCmd.AddCommand(
		ProfileOpenCmd,
		ProfileEditCmd,
		ProfileCloseCmd,
		WageOpenCmd,
		WageEditCmd,
		ExpenseOpenCmd,
		ExpenseEditCmd,
		CloseInvoicesCmd,
	)
	bcmd.RegisterTxSubcommand(InvoicerCmd)
}

func profileOpenCmd(cmd *cobra.Command, args []string) error {
	return profileCmd(args, types.TBTxProfileOpen)
}

func profileEditCmd(cmd *cobra.Command, args []string) error {
	return profileCmd(args, types.TBTxProfileEdit)
}

func profileDeactivateCmd(cmd *cobra.Command, args []string) error {
	return profileCmd(args, types.TBTxProfileClose)
}

func profileCmd(args []string, TxTB byte) error {

	var name string
	if TxTB == types.TBTxProfileOpen {
		if len(args) != 1 {
			return errCmdReqArg("name")
		}
		name = args[0]
	}

	keyPath := viper.GetString("from") //TODO update to proper basecoin key once integrated
	address, err := bcmd.LoadKey(keyPath).Address
	if err != nil {
		return errors.Wrap(err, "Error loading address")
	}

	timezone, err := time.LoadLocation(viper.GetString(FlagTimezone))
	if err != nil {
		return errors.Wrap(err, "Error loading timezone")
	}

	profile := types.NewProfile(
		address,
		name,
		viper.GetString(FlagCur),
		viper.GetString(FlagDepositInfo),
		viper.GetInt(FlagDueDurationDays),
		*timezone,
	)

	txBytes := types.TxBytes(*profile, TxTB)
	return bcmd.AppTx(invoicer.Name, txBytes)
}

func wageOpenCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxWageOpen)
}

func wageEditCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxWageEdit)
}

func expenseOpenCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxExpenseOpen)
}

func expenseEditCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxExpenseEdit)
}

func invoiceCmd(cmd *cobra.Command, args []string, txTB byte) error {
	if len(args) != 2 {
		return errCmdReqArg("amount<amt><cur>")
	}
	sender := args[0]
	amountStr := args[1]

	var id []byte = nil

	//if editing
	if txTB == types.TBTxWageEdit || //require this flag if
		txTB == types.TBTxExpenseEdit { //require this flag if

		//get the old id to remove if editing
		idRaw := viper.GetString(FlagTransactionID)
		if len(idRaw) == 0 {
			errors.New("Need the id to edit, please specify through the flag --id")
		}
		if !isHex(idRaw) {
			return errBadHexID
		}
		id, err = hex.DecodeString(StripHex(idRaw))
		if err != nil {
			return err
		}

		//only allow editing if the invoice is open
	}

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

	//retrieve flags, or if they aren't used, use the senders profile's default

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
	if len(viper.GetString(FlagCur)) > 0 {
		accCur = viper.GetString(FlagCur)
	} else {
		accCur = profile.AcceptedCur
	}

	var invoice types.Invoice

	switch txTB {
	//if not an expense then we're almost done!
	case types.TBTxWageOpen, types.TBTxWageEdit:
		invoice = types.NewWage(
			id,
			sender,
			viper.GetString(FlagTo),
			depositInfo,
			viper.GetString(FlagNotes),
			amt,
			accCur,
			dueDate,
		).Wrap()
	case types.TBTxExpenseOpen, types.TBTxExpenseEdit:
		if len(viper.GetString(FlagTaxesPaid)) == 0 {
			return errors.New("Need --taxes flag")
		}

		taxes, err := types.ParseAmtCurTime(viper.GetString(FlagTaxesPaid), date)
		if err != nil {
			return err
		}
		docBytes, err := ioutil.ReadFile(viper.GetString(FlagReceipt))
		if err != nil {
			return errors.Wrap(err, "Problem reading receipt file")
		}
		_, filename := path.Split(viper.GetString(FlagReceipt))

		invoice = types.NewExpense(
			id,
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
		).Wrap()
	default:
		return errors.New("Unrecognized type-bytes")
	}

	//txBytes := invoice.TxBytesOpen()
	txBytes := types.TxBytes(invoice, txTB)
	return bcmd.AppTx(invoicer.Name, txBytes)
}

func closeInvoicesCmd(cmd *cobra.Command, args []string) error {

	//TODO add conflict checking for ids flag or date range
	//TODO add conflict checking for ids flag or types
	//TODO build get ids from date or types section

	var ids [][]byte
	idsStr := strings.Split(len(viper.GetString(FlagIDs)), ",")

	for i, arg := range args {
		if !isHex(arg) {
			return errBadHexID
		}
		id, err := hex.DecodeString(StripHex(arg))
		if err != nil {
			return err
		}
	}

	date, err := types.ParseDate(viper.GetString(FlagDate), viper.GetString(FlagTimezone))
	if err != nil {
		return err
	}
	act, err := types.ParseAmtCurTime(viper.GetString(FlagCur), date)
	if err != nil {
		return err
	}

	closeInvoices := types.NewCloseInvoices(
		id,
		viper.GetString(FlagTransactionID),
		act,
	)
	//txBytes := closeInvoice.TxBytes()
	txBytes := types.TxBytes(*closeInvoices, types.TBTxCloseInvoices)
	return bcmd.AppTx(invoicer.Name, txBytes)
}
