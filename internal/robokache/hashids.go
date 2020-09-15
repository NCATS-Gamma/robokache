package robokache

import (
	"github.com/speps/go-hashids"
	"log"
	"fmt"
)

var hid *hashids.HashID

func init() {
	hd := hashids.NewData()
	hd.Salt = "This salt is unguessable. Don't even try"
	hd.MinLength = 8

	var err error
	hid, err = hashids.NewWithData(hd)
	if err != nil {
		log.Fatal(err)
	}
}

// Convert an API hash to an integer ID (database primary key)
func hashToID(hash string) (int, error) {
	ids, err := hid.DecodeWithError(hash)
	if err != nil || len(ids) != 1 {
		return -1, fmt.Errorf("Bad Request: Invalid document ID")
	}
	return ids[0], nil
}
// Convert an API hash to an integer ID (database primary key)
func idToHash(id int) (string, error) {
	hash, err := hid.Encode([]int{id})
	if err != nil {
		return "", err
	}
	return hash, nil
}
