#! /bin/bash

# start tendermint
invoicer init
invoicer unsafe_reset_all
invoicer start  > /dev/null &

sleep 5

pid_basecoin=$!

echo "running basecoin"

echo "opening profiles"
#invoicer tx invoicer profile-open Jae         --cur BTC --from key.json --amount 1mycoin
invoicer tx invoicer profile-open Frey        --cur EUR --from key.json --amount 1mycoin
invoicer tx invoicer profile-open Rige        --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Speckle     --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Bucky       --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open CoinCulture --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open AllInBits   --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Interchain  --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Adrian      --cur EUR --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Matt        --cur BTC --from key.json --amount 1mycoin
#invoicer tx invoicer profile-open Peng        --cur BTC --from key.json --amount 1mycoin

#echo "deleting a profile"
#invoicer tx invoicer profile-close Bucky --from key.json --amount 1mycoin

echo "sending a wage invoice"
invoicer tx invoicer wage-open Rige 20.1BTC --to Frey --notes wudduxxp --from key.json --amount 1mycoin

#ID=$(invoicer query invoices | jq .[0][1].ID | tr -d '"')
#echo "editing the already open invoice"
#nvoicer tx invoicer wage-Edit Rige 10.001ETH --id 0x$ID--to AllInBits --notes wudduxxp --from key.json --amount 1mycoin

echo "query all invoices"
invoicer query invoices | jq

ID2=$(invoicer query invoices | jq .[0][1].ID | tr -d '"')

echo "closing the opened invoice with some cash!"
invoicer tx invoicer close-invoice 0x$ID2 --cur 10BTC --id "Tranzact10" --from key.json --amount 1mycoin

echo "open a receipt"
invoicer tx invoicer expense-open Rige 20.1BTC --receipt ~/Desktop/testReceipt.png --taxes 1btc --to Frey --notes wuddup --from key.json --amount 1mycoin

echo "Download the receipt"
ID3=$(invoicer query invoices | jq .[1][1].ID | tr -d '"')
invoicer query invoice 0x$ID3 --download-expense ~/Desktop/rec

kill -9 $pid_basecoin
