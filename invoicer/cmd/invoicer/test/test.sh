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
invoicer tx invoicer profile-deactivate --from ${TESTKEY[3]} --amount 1mycoin > /dev/null

#test if profile is active
ACTIVE=$(invoicer query profile ${NAMES[3]} | jq .Active)
if [ "$ACTIVE" != "false" ]; then 
    echo "Error profile should be inactive: ${NAMES[3]}"
    echo $ACTIVE
    exit 1
fi

#verify it doesn't exist in the active list
PROFILES=$(invoicer query profiles)
if [[ "${PROFILES}" == *"${NAMES[3]}"* ]]; then
    echo "Error profile should be removed: ${NAMES[3]}"
    echo $PROFILES
    exit 1
fi

#verify it does exist in the inactive list
PROFILES=$(invoicer query profiles --inactive)
if [[ "${PROFILES}" != *"${NAMES[3]}"* ]]; then
    echo "Error profile should be removed: ${NAMES[3]}"
    echo $PROFILES
    exit 1
fi

#make sure that we're prevented from editing an inactive profile
invoicer tx invoicer profile-edit --from ${TESTKEY[3]} --cur USD --amount 1mycoin &> /dev/null
CUR=$(invoicer query profile ${NAMES[3]} | jq .AcceptedCur | tr -d '"')
if [ "$CUR" == "USD" ]; then 
    echo "Error inactive profile should not be editable"
    exit 1
fi

echo "editing an existing active profile"
invoicer tx invoicer profile-edit --from ${TESTKEY[0]} --cur USD --amount 1mycoin > /dev/null
CUR=$(invoicer query profile ${NAMES[0]} | jq .AcceptedCur | tr -d '"')
if [ "$CUR" != "USD" ]; then 
    printf 'Error active profile should be editable: want USD, have %s\n' $CUR
    exit 1
fi


echo "sending a contract invoice"
invoicer tx invoicer contract-open 1000.99USD --date 2017-01-01 --to AllInBits --notes thanks! --from ${TESTKEY[1]} --amount 1mycoin > /dev/null

echo "Here is the open invoice:"
ID=$(invoicer query invoices | jq .[0][1].ID | tr -d '"')
invoicer query invoice 0x$ID

echo "editing the already open invoice"
invoicer tx invoicer contract-edit 1000.99CAD --id 0x$ID --date 2017-01-01 --to AllInBits --notes thanks! --from ${TESTKEY[1]} --amount 1mycoin --debug
echo "Here is the edited invoice:"
invoicer query invoice 0x$ID


echo "query all invoices"
invoicer query invoices | jq

echo "pay the opened invoice with some cash!"
invoicer tx invoicer payment Bucky --ids 0x$ID --paid 0.5BTC --date 2017-01-01 --tx-id "FOOBTC-TX-01" --from ${TESTKEY[0]} --amount 1mycoin 
invoicer query invoice 0x$ID | jq
invoicer tx invoicer payment Bucky --ids 0x$ID --paid 0.2454003323983133BTC --date 2017-01-01 --tx-id "FOOBTC-TX-02" --from ${TESTKEY[0]} --amount 1mycoin 
invoicer query invoice 0x$ID | jq #TODO test if open or not right here
invoicer query payments | jq #TODO test if open or not right here

echo "cool"
exit 1

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

