package slim

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/QOSGroup/qbase/txs"
	"github.com/QOSGroup/qstars/slim/funcInlocal/respwrap"
	"github.com/QOSGroup/qstars/x/bank/tx"
	"github.com/pkg/errors"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/bech32"
	"io/ioutil"
	"net/http"

	qbasetypes "github.com/QOSGroup/qbase/types"
	qosaccount "github.com/QOSGroup/qos/account"
	"github.com/QOSGroup/qstars/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

// IP initialization
var (
	HostIP     string
	Accounturl string
	KVurl      string
)

func GetIPfrom(host string) {
	HostIP = host
	Accounturl = "http://" + HostIP + "/accounts/"
	KVurl = "http://" + HostIP + "/kv"
}

func init() {
	var h string
	GetIPfrom(h)
}

func QSCQueryAccountGet(addr string) string {
	aurl := Accounturl + addr
	resp, _ := http.Get(aurl)
	var body []byte
	var err error
	if resp.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}
	}

	defer resp.Body.Close()
	output := string(body)
	return output
}

//genStdSendTx for the Tx send operation
func genStdSendTx(cdc *amino.Codec, sendTx txs.ITx, priKey ed25519.PrivKeyEd25519, chainid string, nonce int64) *txs.TxStd {
	gas := qbasetypes.NewInt(int64(0))
	stx := txs.NewTxStd(sendTx, chainid, gas)
	signature, _ := stx.SignTx(priKey, nonce)
	stx.Signature = []txs.Signature{txs.Signature{
		Pubkey:    priKey.PubKey(),
		Signature: signature,
		Nonce:     nonce,
	}}

	return stx
}

func getAddrFromBech32(bech32Addr string) (address []byte, err error) {
	prefix, bz, err := bech32.DecodeAndConvert(bech32Addr)
	address = bz
	if prefix != PREF_ADD {
		return nil, errors.Wrap(err, "Valid Address string should begin with")
	}
	return
}

//only need the following arguments, it`s enough!
func QSCtransferSendStr(addrto, coinstr, privkey, chainid string) string {
	//generate the receiver address, i.e. "addrto" with the following format
	to, err := getAddrFromBech32(addrto)
	if err != nil {
		fmt.Println(err)
	}
	//generate the sender address, i.e. the "from" part as the input with privkey in hex string format
	//_, addrben32, priv := utility.PubAddrRetrievalFromAmino(privkey, cmCdc)

	bz, _ := base64.StdEncoding.DecodeString(privkey)
	var key ed25519.PrivKeyEd25519
	Cdc.MustUnmarshalBinaryBare(bz, &key)
	priv := key
	addrben32, _ := bech32.ConvertAndEncode(PREF_ADD, key.PubKey().Address().Bytes())

	from, err := getAddrFromBech32(addrben32)

	//coins generate from input
	var ccs []qbasetypes.BaseCoin
	coins, err := types.ParseCoins(coinstr)
	if err != nil {
		fmt.Println(err)
	}
	for _, coin := range coins {
		ccs = append(ccs, qbasetypes.BaseCoin{
			Name:   coin.Denom,
			Amount: qbasetypes.NewInt(coin.Amount.Int64()),
		})
	}

	//Get "nonce" from the func QSCQueryAccountGet
	AccountStr := QSCQueryAccountGet(addrben32)
	accb := []byte(AccountStr)
	data := respwrap.RPCResponse{}
	err = Cdc.UnmarshalJSON(accb, &data)
	rawresp := data.Result
	acc := qosaccount.QOSAccount{}
	Cdc.UnmarshalJSON(rawresp, &acc)

	//coins check to further improvement
	/*	var qcoins types.Coins
		for _, qsc := range acc.QSCs {
			amount := qsc.Amount
			qcoins = append(qcoins, types.NewCoin(qsc.Name, types.NewInt(amount.Int64())))
		}
		qcoins = append(qcoins, types.NewCoin("qos", types.NewInt(acc.QOS.Int64())))

		if !qcoins.IsGTE(coins) {
			fmt.Println("Address %s doesn't have enough coins to pay for this transaction.", from)
		}
	*/
	var nn int64
	nn = int64(acc.Nonce)
	nn++

	//New transfer for QOS transaction
	t := tx.NewTransfer(from, to, ccs)
	msg := genStdSendTx(Cdc, t, priv, chainid, nn)

	jasonpayload, err := Cdc.MarshalJSON(msg)
	if err != nil {
		fmt.Println(err)
	}
	datas := bytes.NewBuffer(jasonpayload)
	aurl := Accounturl + "txSend"
	req, _ := http.NewRequest("POST", aurl, datas)
	req.Header.Set("Content-Type", "application/json")
	clt := http.Client{}
	resp, _ := clt.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	output := string(body)
	return output
}