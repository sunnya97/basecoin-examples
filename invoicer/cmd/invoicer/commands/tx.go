package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/tendermint/basecoin-examples/invoicer/plugins/invoicer"
	bcmd "github.com/tendermint/basecoin/cmd/commands"
	"github.com/tendermint/basecoin/types"
	"github.com/tendermint/go-wire"
)

const InvoicerName = "invoicer"

var (
	//flags
	issueFlag   string
	voteFeeFlag string
	voteForFlag bool

	//commands
	InvoicerCmd = &cobra.Command{
		Use:   "invoicer",
		Short: "commands relating to invoicer system",
	}

	NewProfileCmd = &cobra.Command{
		Use:   "create-profile",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  newProfileCmd,
	}

	OpenInvoiceCmd = &cobra.Command{
		Use:   "invoice",
		Short: "send an invoice",
		RunE:  openInvoiceCmd,
	}

	OpenExpenseCmd = &cobra.Command{
		Use:   "expense",
		Short: "send an expense",
		RunE:  openExpenseCmd,
	}
	CloseCmd = &cobra.Command{
		Use:   "close",
		Short: "close an invoice or expense",
		RunE:  openExpenseCmd,
	}
)

func init() {

	//register flags

	issueFlag2Reg := bcmd.Flag2Register{&issueFlag, "issue", "default issue", "name of the issue to generate or vote for"}

	createIssueFlags := []bcmd.Flag2Register{
		issueFlag2Reg,
		{&voteFeeFlag, "voteFee", "1voteToken",
			"the fees required to  vote on this new issue, uses the format <amt><coin>,<amt2><coin2>,... (eg: 1gold,2silver,5btc)"},
	}

	voteFlags := []bcmd.Flag2Register{
		issueFlag2Reg,
		{&voteForFlag, "voteFor", false, "if present vote will be a vote-for, if absent a vote-against"},
	}

	bcmd.RegisterFlags(P2VCreateIssueCmd, createIssueFlags)
	bcmd.RegisterFlags(P2VVoteCmd, voteFlags)

	//register commands
	bcmd.RegisterTxSubcommand(P2VCreateIssueCmd)
}

//type Invoice struct {
//ID             []byte
//AccSender      []byte
//AccReceiver    []byte
//DepositInfo    string
//Amount         AmtCurTime
//AcceptedCur    []Currency
//TransactionID  string     //empty when unpaid
//PaymentCurTime AmtCurTime //currency used to pay invoice, empty when unpaid
//}

//type Expense struct {
//Invoice
//pdfReceipt []byte
//notes      string
//taxesPaid  AmtCurTime
//}

//type Profile struct {
//Receiver           []byte //address
//Nickname           string //nickname for querying TODO check to make sure only one word
//LegalName          string
//acceptedCur        []currency //currencies you will accept payment in
//DefaultDepositInfo string     //default deposit information (mostly for fiat)
//dueDurationDays    int        //default duration until a sent invoice due date
//}

func createProfileCmd(cmd *cobra.Command, args []string) error {

	createIssueFee := types.Coins{{"profileToken", 1}} //manually set the cost to create a new issue here

	txBytes := invoicer.NewCreateIssueTxBytes(issueFlag, voteFee, createIssueFee)

	return bcmd.AppTx(InvoicerName, txBytes)
}

func voteCmd(cmd *cobra.Command, args []string) error {

	var voteTB byte = invoicer.TypeByteVoteFor
	if !voteForFlag {
		voteTB = invoicer.TypeByteVoteAgainst
	}

	txBytes := invoicer.NewVoteTxBytes(issueFlag, voteTB)

	fmt.Println("Vote transaction sent")
	return bcmd.AppTx(InvoicerName, txBytes)
}
