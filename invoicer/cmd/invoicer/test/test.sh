#! /bin/bash

echo "starting tendermint network"

# start tendermint
invoicer init
invoicer unsafe_reset_all
invoicer start  > /dev/null &

sleep 5

pid_basecoin=$!


function cleanup {
    echo "cleaning up"
    rm -rf /tmp/invoicer/
    kill -9 $pid_basecoin
}
trap cleanup EXIT


echo "opening profiles"

WORKDIR=(~/.invoicer)
NAMES=(AllInBits Bucky Satoshi Dummy)

for i in "${!NAMES[@]}"; do 
    #make some keys and send them some mycoin 
    TESTKEY[$i]=testkey$i.json
    invoicer key new > $WORKDIR/${TESTKEY[$i]}
    ADDR[$i]=$(cat $WORKDIR/${TESTKEY[$i]} | jq .address | tr -d '"')
    invoicer tx send --from key.json --to ${ADDR[$i]} --amount 1000mycoin > /dev/null 

    #open the profile
    invoicer tx invoicer profile-open ${NAMES[$i]} --cur BTC --from ${TESTKEY[$i]} --amount 1mycoin > /dev/null
done

#check if the profiles have been opened
PROFILES=$(invoicer query profiles)
for i in "${!NAMES[@]}"; do 
   if [[ $PROFILES != *"${NAMES[$i]}"* ]]; then
         echo "Error Missing Profile ${NAMES[$i]}"
         echo $PROFILES
         exit 1
     fi
done


echo "deleting a profile"
echo "invoicer tx invoicer profile-deactivate --from ${TESTKEY[3]} --amount 1mycoin"
invoicer query profile ${NAMES[3]}
invoicer tx invoicer profile-deactivate --from ${TESTKEY[3]} --amount 1mycoin
invoicer query profile ${NAMES[3]}

#test if profile is active
ACTIVE=$(invoicer query profile ${NAMES[3]} | jq .Active)
if [ "$ACTIVE" != "false" ]; then 
    echo "Error profile should be inactive: ${NAMES[3]}"
    echo $ACTIVE
    exit 1
fi

#verify it doesn't exist in the list
PROFILES=$(invoicer query profiles --active)
if [[ "${PROFILES}" == *"${NAMES[3]}"* ]]; then
    echo "Error profile should be removed: ${NAMES[3]}"
    echo $PROFILES
    exit 1
fi

echo "cool"
exit 1

echo "sending a wage invoice"
invoicer tx invoicer wage-open 99.99BTC --to AllInBits --notes thanks! --from $TESTKEY[1] --amount 1mycoin

#ID=$(invoicer query invoices | jq .[0][1].ID | tr -d '"')
echo "editing the already open invoice"
invoicer tx invoicer wage-Edit Rige 10.001ETH --id 0x$ID--to AllInBits --notes wudduxxp --from key.json --amount 1mycoin

echo "query all invoices"
invoicer query invoices | jq

ID2=$(invoicer query invoices | jq .[0][1].ID | tr -d '"')

echo "closing the opened invoice with some cash!"
invoicer tx invoicer close-invoice 0x$ID2 --cur 10BTC --id "Tranzact10" --from key.json --amount 1mycoin


echo "open a receipt"
DIR1=$(/tmp/invoicer)
DIR2=$($DIR1/retrieved)
mkdir $DIR1 ; mkdir $DIR2
wget $DIR1/invoicerDoc.png https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_120x44dp.png
invoicer tx invoicer expense-open Rige 20.1BTC --receipt $DIR1/invoicerDoc.png --taxes 1btc --to Frey --notes wuddup --from key.json --amount 1mycoin

echo "Download the receipt"
ID3=$(invoicer query invoices | jq .[1][1].ID | tr -d '"')
invoicer query invoice 0x$ID3 --download-expense $DIR2

if [ ! -f $DIR2/invoicerDoc.png ]; then
    echo "ERROR: receipt didn't download from query"
fi

