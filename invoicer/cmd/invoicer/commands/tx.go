package commands

import (
	"bytes"
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

	ContractOpenCmd = &cobra.Command{
		Use:   "contract-open [amount]",
		Short: "Send a contract invoice of amount <value><currency>",
		RunE:  contractOpenCmd,
	}

	ContractEditCmd = &cobra.Command{
		Use:   "contract-edit [amount]",
		Short: "Edit an open contract invoice to amount <value><currency>",
		RunE:  contractEditCmd,
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

	PaymentCmd = &cobra.Command{
		Use:   "payment [receiver]",
		Short: "pay invoices and expenses with transaction infomation",
		RunE:  paymentCmd,
	}
)

func init() {

	//register flags
	fsProfile := flag.NewFlagSet("", flag.ContinueOnError)
	fsProfile.String(FlagTo, "", "Who you're invoicing")
	fsProfile.String(FlagCur, "BTC", "Payment curreny accepted")
	fsProfile.String(FlagDepositInfo, "", "Default deposit information to be provided")
	fsProfile.Int(FlagDueDurationDays, 14, "Default number of days until invoice is due from invoice submission")

	fsInvoice := flag.NewFlagSet("", flag.ContinueOnError)
	fsInvoice.String(FlagTo, "allinbits", "Name of the invoice receiver")
	fsInvoice.String(FlagDepositInfo, "", "Deposit information for invoice payment (default: profile)")
	fsInvoice.String(FlagNotes, "", "Notes regarding the expense")
	fsInvoice.String(FlagCur, "btc", "Currency which invoice should be paid in")
	fsInvoice.String(FlagDate, "", "Invoice demon date in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")
	fsInvoice.String(FlagDueDate, "", "Invoice due date in the format YYYY-MM-DD eg. 2016-12-31 (default: profile)")

	fsExpense := flag.NewFlagSet("", flag.ContinueOnError)
	fsExpense.String(FlagReceipt, "", "Directory to receipt document file")
	fsExpense.String(FlagTaxesPaid, "", "Taxes amount in the format <decimal><currency> eg. 10.23usd")

	fsPayment := flag.NewFlagSet("", flag.ContinueOnError)
	fsPayment.String(FlagIDs, "", "IDs to close during this transaction <id1>,<id2>,<id3>... ")
	fsPayment.String(FlagTransactionID, "", "Completed transaction ID")
	fsPayment.String(FlagAmount, "", "Payment amount in the format <decimal><currency> eg. 10.23usd")
	fsPayment.String(FlagDate, "", "Date payment in the format YYYY-MM-DD eg. 2016-12-31 (default: today)")
	fsPayment.String(FlagDateRange, "",
		"Autoselect IDs within the date range start:end, where start/end are in the format YYYY-MM-DD, or empty. ex. --date 1991-10-21:")

	fsEdit := flag.NewFlagSet("", flag.ContinueOnError)
	fsEdit.String(FlagID, "", "ID (hex) of the invoice to modify")

	ProfileOpenCmd.Flags().AddFlagSet(fsProfile)
	ProfileEditCmd.Flags().AddFlagSet(fsProfile)

	ContractOpenCmd.Flags().AddFlagSet(fsInvoice)
	ContractEditCmd.Flags().AddFlagSet(fsInvoice)
	ContractEditCmd.Flags().AddFlagSet(fsEdit)

	ExpenseOpenCmd.Flags().AddFlagSet(fsInvoice)
	ExpenseOpenCmd.Flags().AddFlagSet(fsExpense)
	ExpenseEditCmd.Flags().AddFlagSet(fsInvoice)
	ExpenseEditCmd.Flags().AddFlagSet(fsExpense)
	ExpenseEditCmd.Flags().AddFlagSet(fsEdit)

	PaymentCmd.Flags().AddFlagSet(fsPayment)

	//register commands
	InvoicerCmd.AddCommand(
		ProfileOpenCmd,
		ProfileEditCmd,
		ProfileCloseCmd,
		ContractOpenCmd,
		ContractEditCmd,
		ExpenseOpenCmd,
		ExpenseEditCmd,
		PaymentCmd,
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

	address, err := getAddress()
	if err != nil {
		return errors.Wrap(err, "Error loading address")
	}

	profile := types.NewProfile(
		address,
		name,
		viper.GetString(FlagCur),
		viper.GetString(FlagDepositInfo),
		viper.GetInt(FlagDueDurationDays),
	)

	txBytes := types.TxBytes(*profile, TxTB)
	return bcmd.AppTx(invoicer.Name, txBytes)
}

func getAddress() (bcmd.Address, error) {
	keyPath := viper.GetString("from") //TODO update to proper basecoin key once integrated
	return bcmd.LoadKey(keyPath).Address
}

func contractOpenCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxContractOpen)
}

func contractEditCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxContractEdit)
}

func expenseOpenCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxExpenseOpen)
}

func expenseEditCmd(cmd *cobra.Command, args []string) error {
	return invoiceCmd(cmd, args, types.TBTxExpenseEdit)
}

func invoiceCmd(cmd *cobra.Command, args []string, txTB byte) error {
	if len(args) != 1 {
		return errCmdReqArg("amount<amt><cur>")
	}
	sender := args[0]
	amountStr := args[1]

	var id []byte = nil

	//if editing
	if txTB == types.TBTxContractEdit || //require this flag if
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
	}

	//get the sender's address
	address, err := getAddress()
	if err != nil {
		return errors.Wrap(err, "Error loading address")
	}

	//determine the senders profile TODO optimize, move to the ABCI app
	tmAddr := cmd.Parent().Flag("node").Value.String()
	profiles, err := queryListProfile(tmAddr)
	if err != nil {
		return err
	}
	var profile *types.Profile
	for _, name := range profiles {
		p, err := queryProfile(tmAddr, name)
		if err != nil {
			return err
		}
		if bytes.Compare(p.Address, address) == 0 {
			profile = p
			break
		}

	}

	dateStr := viper.GetString(FlagDate)
	date := time.Now()
	if len(dateStr) > 0 {
		date, err := types.ParseDate(dateStr)
		if err != nil {
			return err
		}
	}
	amt, err := types.ParseAmtCurTime(amountStr, date)
	if err != nil {
		return err
	}

	//retrieve flags, or if they aren't used, use the senders profile's default

	var dueDate time.Time
	if len(viper.GetString(FlagDueDate)) > 0 {
		dueDate, err = types.ParseDate(viper.GetString(FlagDueDate))
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
	case types.TBTxContractOpen, types.TBTxContractEdit:
		invoice = types.NewContract(
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

func paymentCmd(cmd *cobra.Command, args []string) error {

	var receiver string
	if len(args) != 1 {
		return errCmdReqArg("receiver")
	}
	receiver = args[0]

	flagsIDs := viper.GetString(FlagIDs)
	flagsDateRange := viper.GetString(FlagDateRange)

	if len(flagIDs) > 0 && len(flagDateRange) > 0 {
		return errors.New("Cannot use both the IDs flag and date-range flag")
	}
	if len(flagIDs) == 0 && len(flagDateRange) == 0 {
		return errors.New("Must include an IDs flag or date-range flag")
	}

	//Get the date range or list of IDs
	var ids [][]byte
	var startDate, endDate *time.Time = nil, nil
	if len(flagDateRange) > 0 {
		dates := strings.Split(flagDateRange)
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
	} else {
		idsStr := strings.Split(len(flagIDs), ",")
		for i, arg := range args {
			if !isHex(arg) {
				return errBadHexID
			}
			id, err := hex.DecodeString(StripHex(arg))
			if err != nil {
				return err
			}
		}
	}

	date, err := types.ParseDate(viper.GetString(FlagDate))
	if err != nil {
		return err
	}
	amt, err := types.ParseAmtCurTime(viper.GetString(FlagAmount), date)
	if err != nil {
		return err
	}

	payment := types.NewPayment(
		ids,
		receiver,
		viper.GetString(FlagTransactionID),
		amt,
		startDate,
		endDate,
	)
	//txBytes := closeInvoice.TxBytes()
	txBytes := types.TxBytes(*payment, types.TBTxCloseInvoices)
	return bcmd.AppTx(invoicer.Name, txBytes)
}
