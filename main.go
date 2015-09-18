package main

import (
	"UpdateStellarAddrDB/strAddrManage"
	"fmt"
)

var stellarAddr *strAddrManage.StellarAddressManager

const (
	READ_FROM_FILE   bool = true
	READ_FROM_CREATE bool = false
)

func main() {
	fmt.Println(stellarAddr.StrAddrs)
	defer stellarAddr.Destroy()

	num, err := stellarAddr.UpdateDB()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("\r\nInset %d records in database!\r\n", num)
	}
}

func init() {
	stellarAddr = &strAddrManage.StellarAddressManager{}
	stellarAddr.Init(READ_FROM_CREATE, 10)
}
