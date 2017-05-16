package commands

var (
	//common
	FlagTo          string = "to"
	FlagCur         string = "cur"
	FlagDate        string = "date"
	FlagDateRange   string = "date-range"
	FlagDepositInfo string = "info"
	FlagNotes       string = "notes"
	FlagID          string = "id"

	//query
	FlagNum         string = "num"
	FlagType        string = "type"
	FlagFrom        string = "from"
	FlagDownloadExp string = "download-expense"

	//transaction
	//profile flags
	FlagDueDurationDays string = "due-days"
	FlagTimezone        string = "timezone"

	//invoice flags
	FlagDueDate string = "due-date"

	//expense flags
	FlagReceipt   string = "receipt"
	FlagTaxesPaid string = "taxes"

	//close/edit flags
	FlagTransactionID string = "trans"
	FlagIDs           string = "ids"
)
