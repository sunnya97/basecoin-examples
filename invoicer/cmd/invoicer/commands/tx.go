package commands

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/tendermint/basecoin-examples/invoicer"
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
	CreateProfileCmd = &cobra.Command{
		Use:   "invoicer",
		Short: "commands relating to invoicer system",
	}

	CreateProfileCmd = &cobra.Command{
		Use:   "create-profile",
		Short: "open a profile for sending/receiving invoices and expense claims",
		RunE:  createProfileCmd,
	}

	OpenInvoiceCmd = &cobra.Command{
		Use:   "invoice",
		Short: "send an invoice",
		RunE:  openInvoiceCmd,
	}

	OpenInvoiceCmd = &cobra.Command{
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

func createIssueCmd(cmd *cobra.Command, args []string) error {

	voteFee, err := bcmd.ParseCoins(voteFeeFlag)
	if err != nil {
		return err
	}

	createIssueFee := types.Coins{{"issueToken", 1}} //manually set the cost to create a new issue here

	txBytes := invoicer.NewCreateIssueTxBytes(issueFlag, voteFee, createIssueFee)

	fmt.Println("Issue creation transaction sent")
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

func queryIssueCmd(cmd *cobra.Command, args []string) error {

	//get the parent context
	parentContext := cmd.Parent()

	//get the issue, generate issue key
	if len(args) != 1 {
		return fmt.Errorf("query command requires an argument ([issue])") //never stack trace
	}
	issue := args[0]
	issueKey := invoicer.IssueKey(issue)

	//perform the query, get response
	resp, err := bcmd.Query(parentContext.Flag("node").Value.String(), issueKey)
	if err != nil {
		return err
	}
	if !resp.Code.IsOK() {
		return errors.Errorf("Query for issueKey (%v) returned non-zero code (%v): %v",
			string(issueKey), resp.Code, resp.Log)
	}

	//get the invoicer issue object and print it
	p2vIssue, err := invoicer.GetIssueFromWire(resp.Value)
	if err != nil {
		return err
	}
	fmt.Println(string(wire.JSONBytes(p2vIssue)))
	return nil
}
