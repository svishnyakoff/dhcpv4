package transaction

import (
	"math/rand"
)

type TxId = uint32

func RandomTransactionId() TxId {
	return rand.Uint32()
}
